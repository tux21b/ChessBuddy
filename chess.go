// ChessBuddy - Play chess with Go, HTML5, WebSockets and random strangers!
//
// Copyright (c) 2012 by Christoph Hack <christoph@tux21b.org>
// All rights reserved. Distributed under the Simplified BSD License.
//
package main

import (
    "code.google.com/p/go.net/websocket"
    "fmt"
    "html/template"
    "log"
    "math/rand"
    "net/http"
    "sync/atomic"
    "time"
)

type figur int

const (
    EMPTY figur = iota
    KING
    QUEEN
    ROOK
    BISHOP
    KNIGHT
    PAWN
)
const (
    WHITE = 1
    BLACK = -1
)

const SIZE = 8

type Board [SIZE * SIZE]figur

// Generate a new chess board with all pieces placed at their initial position.
func NewBoard() *Board {
    return &Board{
        ROOK * WHITE, KNIGHT * WHITE, BISHOP * WHITE, QUEEN * WHITE,
        KING * WHITE, BISHOP * WHITE, KNIGHT * WHITE, ROOK * WHITE,
        PAWN * WHITE, PAWN * WHITE, PAWN * WHITE, PAWN * WHITE,
        PAWN * WHITE, PAWN * WHITE, PAWN * WHITE, PAWN * WHITE,
        EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY,
        EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY,
        EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY,
        EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY,
        PAWN * BLACK, PAWN * BLACK, PAWN * BLACK, PAWN * BLACK,
        PAWN * BLACK, PAWN * BLACK, PAWN * BLACK, PAWN * BLACK,
        ROOK * BLACK, KNIGHT * BLACK, BISHOP * BLACK, QUEEN * BLACK,
        KING * BLACK, BISHOP * BLACK, KNIGHT * BLACK, ROOK * BLACK,
    }
}

// Check wethever the it's valid to move the current piece located at
// (ax, ay) to a new position at (bx, by).
func (b *Board) ValidMove(ax, ay, bx, by int) bool {
    if ax < 0 || ax > 7 || ay < 0 || ay > 7 ||
        bx < 0 || bx > 7 || by < 0 || by > 7 ||
        b[ay*SIZE+ax]*b[by*SIZE+bx] > 0 || (ax == bx && ay == by) {
        return false
    }
    switch b[ay*SIZE+ax] {
    case KNIGHT * WHITE, KNIGHT * BLACK:
        return (ax-bx)*(ax-bx)+(ay-by)*(ay-by) == 5
    case PAWN * WHITE:
        return (ax == bx && (ay+1 == by || (ay == 1 && by == 3)) &&
            b[by*SIZE+bx] == EMPTY) || ((ax-bx)*(ax-bx) == 1 &&
            ay+1 == by && b[by*SIZE+bx] != EMPTY)
    case PAWN * BLACK:
        return (ax == bx && (ay-1 == by || (ay == 6 && by == 4)) &&
            b[by*SIZE+bx] == EMPTY) || ((ax-bx)*(ax-bx) == 1 &&
            ay-1 == by && b[by*SIZE+bx] != EMPTY)
    case BISHOP * WHITE, BISHOP * BLACK:
        return (ax-bx)*(ax-bx) == (ay-by)*(ay-by) && b.freeWay(ax, ay, bx, by)
    case ROOK * WHITE, ROOK * BLACK:
        return (ax == bx || ay == by) && b.freeWay(ax, ay, bx, by)
    case QUEEN * WHITE, QUEEN * BLACK:
        return (ax == bx || ay == by || (ax-bx)*(ax-bx) == (ay-by)*(ay-by)) &&
            b.freeWay(ax, ay, bx, by)
    case KING * WHITE, KING * BLACK:
        return (ax-bx)*(ax-bx) <= 1 && (ay-by)*(ay-by) <= 1
    }

    return false
}

// Move the piece located at (ax, ay) to (bx, by) if that's a valid move.
func (b *Board) Move(ax, ay, bx, by int) bool {
    if !b.ValidMove(ax, ay, bx, by) {
        return false
    }
    b[by*SIZE+bx] = b[ay*SIZE+ax]
    b[ay*SIZE+ax] = EMPTY
    return true
}

// Check for obstacles at the way from (ax, ay) to (bx, by). The start and
// the end location do not count as obstacles, even if they are occupied.
func (b *Board) freeWay(ax, ay, bx, by int) bool {
    m := (bx - ax)
    if m2 := (ax - bx); m2 > m {
        m = m2
    }
    if m2 := (by - ay); m2 > m {
        m = m2
    }
    if m2 := (ay - by); m2 > m {
        m = m2
    }
    for i := 1; i < m; i++ {
        x := ax + (i*(bx-ax))/m
        y := ay + (i*(by-ay))/m
        if b[y*SIZE+x] != EMPTY {
            return false
        }
    }
    return true
}

// General message struct which is used for parsing client requests and sending
// back responses.
type Message struct {
    Cmd                    string
    Turn                   int
    Ax, Ay                 int
    Bx, By                 int
    Board                  *Board
    Color                  int
    NumPlayers             int32
    History                string
    RemainingA, RemainingB time.Duration
}

type Player struct {
    Conn      *websocket.Conn
    Exit      chan bool
    Color     int
    Remaining time.Duration
}

// Check wethever the player is still connected by sending a ping command.
func (p Player) Alive() bool {
    if err := websocket.JSON.Send(p.Conn, Message{Cmd: "ping",
        NumPlayers: atomic.LoadInt32(&numPlayers)}); err != nil {
        return false
    }
    var msg Message
    if err := websocket.JSON.Receive(p.Conn, &msg); err != nil {
        return false
    }
    return msg.Cmd == "pong"
}

// Available Players which are currently looking for a taff opponent.
var available = make(chan Player, 100)

// Total number of connected players
var numPlayers int32 = 0

// GoRoutine for hooking up pairs of available players.
func hookUp() {
    a := <-available
    for {
        b := <-available
        if a.Alive() {
            go play(a, b)
            a = <-available
        } else {
            a.Exit <- true
            a = b
        }
    }
}

func play(a, b Player) {
    defer func() {
        a.Exit <- true
        b.Exit <- true
    }()

    log.Println("Starting new game")

    board := NewBoard()
    turn := 1
    if rand.Float32() > 0.5 {
        a, b = b, a
    }
    a.Color = WHITE
    b.Color = BLACK
    a.Remaining = 5 * time.Minute
    b.Remaining = 5 * time.Minute

    err := websocket.JSON.Send(a.Conn, Message{Cmd: "start", Board: board,
        Color: a.Color, Turn: turn, NumPlayers: atomic.LoadInt32(&numPlayers),
        RemainingA: a.Remaining, RemainingB: b.Remaining})
    if err != nil {
        return
    }
    err = websocket.JSON.Send(b.Conn, Message{Cmd: "start", Board: board,
        Color: b.Color, Turn: turn, NumPlayers: atomic.LoadInt32(&numPlayers),
        RemainingA: a.Remaining, RemainingB: b.Remaining})
    if err != nil {
        return
    }

    start := time.Now()
    for {
        var msg Message
        if err := websocket.JSON.Receive(a.Conn, &msg); err != nil {
            break
        }
        msg.History = board.History(msg.Ax, msg.Ay, msg.Bx, msg.By)
        if msg.Cmd == "move" && msg.Turn == turn &&
            ((a.Color == WHITE && turn&1 == 1) ||
                (a.Color == BLACK && turn&1 == 0)) &&
            board.Move(msg.Ax, msg.Ay, msg.Bx, msg.By) {
            msg.NumPlayers = atomic.LoadInt32(&numPlayers)
            if turn&1 == 1 {
                msg.History = fmt.Sprintf("%d. %s", (turn+1)/2, msg.History)
            }
            now := time.Now()
            a.Remaining -= now.Sub(start)
            if a.Remaining < 0 {
                a.Remaining = 0
            }
            start = now
            msg.RemainingA, msg.RemainingB = a.Remaining, b.Remaining
            if a.Color == BLACK {
                msg.RemainingA, msg.RemainingB = b.Remaining, a.Remaining
            }
            websocket.JSON.Send(a.Conn, msg)
            websocket.JSON.Send(b.Conn, msg)
            a, b = b, a
            turn++
        }
    }
}

// Generates a single log entry for the history.
func (b *Board) History(ax, ay, bx, by int) string {
    if !b.ValidMove(ax, ay, bx, by) {
        return ""
    }
    cols := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
    symb := []string{"", "N", "B", "R", "Q", "K", "",
        "K", "Q", "R", "B", "N", ""}
    if b[by*SIZE+bx] == EMPTY {
        return fmt.Sprintf("%s%s%d-%s%d", symb[b[ay*SIZE+ax]+6], cols[ax],
            ay+1, cols[bx], by+1)
    }
    return fmt.Sprintf("%s%s%dx%s%d", symb[b[ay*SIZE+ax]+6], cols[ax],
        ay+1, cols[bx], by+1)
}

var tmpl = template.Must(template.ParseFiles("chess.html"))

// Serve the index page.
func handleIndex(w http.ResponseWriter, r *http.Request) {
    if err := tmpl.Execute(w, r.Host); err != nil {
        log.Printf("tmpl.Execute: %v", err)
    }
}

// Serve a static file (e.g. style sheets, scripts or images).
func handleFile(path string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, path)
    }
}

func handleChess(ws *websocket.Conn) {
    atomic.AddInt32(&numPlayers, 1)
    log.Printf("Connected: %v", ws.RemoteAddr())

    player := Player{Conn: ws, Exit: make(chan bool)}
    available <- player

    <-player.Exit
    ws.Close()
    log.Printf("Disconnected: %v", ws.RemoteAddr())
    atomic.AddInt32(&numPlayers, -1)
}

func main() {
    http.HandleFunc("/", handleIndex)
    http.HandleFunc("/chess.js", handleFile("chess.js"))
    http.HandleFunc("/chess.css", handleFile("chess.css"))
    http.Handle("/ws", websocket.Handler(handleChess))

    go hookUp()

    http.ListenAndServe(":8000", nil)
}
