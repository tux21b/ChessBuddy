// ChessBuddy - Play chess with Go, HTML5, WebSockets and random strangers!
//
// Copyright (c) 2012 by Christoph Hack <christoph@tux21b.org>
// All rights reserved. Distributed under the Simplified BSD License.

package main

import (
    "code.google.com/p/go.net/websocket"
    "flag"
    "fmt"
    "go/build"
    "html/template"
    "log"
    "math/rand"
    "net"
    "net/http"
    "path/filepath"
    "runtime"
    "sync/atomic"
    "time"
)

// General message struct which is used for parsing client requests and sending
// back responses.
type Message struct {
    Cmd                    string
    Turn                   int
    Ax, Ay                 int
    Bx, By                 int
    White                  bool
    NumPlayers             int32
    History                string
    RemainingA, RemainingB time.Duration
    Text                   string
    Moves                  []pos
}

type Player struct {
    Conn      *websocket.Conn
    White     bool
    Remaining time.Duration
    Out       chan<- Message
}

// Check wethever the player is still connected by sending a ping command.
func (p *Player) Alive() bool {
    if err := websocket.JSON.Send(p.Conn, Message{Cmd: "ping"}); err != nil {
        return false
    }
    var msg Message
    if err := websocket.JSON.Receive(p.Conn, &msg); err != nil {
        return false
    }
    return msg.Cmd == "pong"
}

func (p *Player) String() string {
    if p.White {
        return "White"
    }
    return "Black"
}

// Available Players which are currently looking for a taff opponent.
var available = make(chan *Player, 100)

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
            close(a.Out)
            a = b
        }
    }
}

func play(a, b *Player) {
    defer func() {
        close(a.Out)
        close(b.Out)
    }()

    log.Println("Starting new game")

    board := NewBoard()
    if rand.Float32() > 0.5 {
        a, b = b, a
    }
    a.White = true
    a.Remaining = *timeLimit
    b.Remaining = *timeLimit

    a.Out <- Message{Cmd: "start", White: a.White, Turn: board.Turn(),
        RemainingA: a.Remaining, RemainingB: b.Remaining}
    b.Out <- Message{Cmd: "start", White: b.White, Turn: board.Turn(),
        RemainingA: a.Remaining, RemainingB: b.Remaining}

    start := time.Now()
    for {
        var msg Message
        a.Conn.SetReadDeadline(start.Add(a.Remaining))
        if err := websocket.JSON.Receive(a.Conn, &msg); err != nil {
            if err, ok := err.(net.Error); ok && err.Timeout() {
                a.Remaining = 0
                msg = Message{
                    Cmd:  "msg",
                    Text: fmt.Sprintf("Out of time: %v wins!", b),
                }
                b.Out <- msg
                a.Out <- msg
            } else {
                msg = Message{
                    Cmd:  "msg",
                    Text: "Opponent quit... Reload?",
                }
                b.Out <- msg
                a.Out <- msg
            }
            break
        }
        if msg.Cmd == "move" && msg.Turn == board.Turn() &&
            msg.White == board.White() &&
            board.Move(msg.Ax, msg.Ay, msg.Bx, msg.By) {

            msg.History = board.LastMove()
            now := time.Now()
            a.Remaining -= now.Sub(start)
            if a.Remaining <= 10*time.Millisecond {
                a.Remaining = 10 * time.Millisecond
            }
            start = now
            msg.RemainingA, msg.RemainingB = a.Remaining, b.Remaining
            if !a.White {
                msg.RemainingA, msg.RemainingB = b.Remaining, a.Remaining
            }
            a, b = b, a
            a.Out <- msg
            b.Out <- msg
        } else if msg.Cmd == "select" && msg.Turn == board.Turn() &&
            msg.White == board.White() {
            msg.Moves = board.Moves(msg.Ax, msg.Ay)
            a.Out <- msg
        }
    }
}

// Serve the index page.
func handleIndex(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path != "/" {
        http.Error(w, "Not Found", http.StatusNotFound)
        return
    }
    if err := tmpl.Execute(w, r.Host); err != nil {
        log.Printf("tmpl.Execute: %v", err)
    }
}

// Serve a static file (e.g. style sheets, scripts or images).
func handleFile(path string) http.HandlerFunc {
    path = filepath.Join(root, path)
    return func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, path)
    }
}

func handleWS(ws *websocket.Conn) {
    log.Println("Connected:", ws.Request().RemoteAddr)
    atomic.AddInt32(&numPlayers, 1)
    exitStat := make(chan bool, 1)

    defer func() {
        exitStat <- true
        atomic.AddInt32(&numPlayers, -1)
        log.Println("Disconnected", ws.Request().RemoteAddr)
        ws.Close()
    }()

    // Send statistics (i.e. player count) regularly. This will help to to
    // detect disconnected players earlier and will prevent stupid proxies
    // from closing inactive connections.
    go func() {
        ticker := time.NewTicker(20 * time.Second)
        defer ticker.Stop()
        msg := Message{Cmd: "stat"}
        for {
            msg.NumPlayers = atomic.LoadInt32(&numPlayers)
            if err := websocket.JSON.Send(ws, msg); err != nil {
                if nerr, ok := err.(net.Error); ok && !nerr.Temporary() {
                    log.Printf("Network Error: %v", nerr)
                    ws.Close()
                    return
                }
            }
            select {
            case <-ticker.C:
                // continue
            case <-exitStat:
                return
            }
        }
    }()

    // Add the player to the pool of available players so that he can get
    // hooked up
    out := make(chan Message, 1)
    available <- &Player{Conn: ws, Out: out}

    // Send the move commands from the game asynchronously, so that a slow
    // internet connection can not be simulated to use up the opponents
    // time limit.
    for msg := range out {
        if err := websocket.JSON.Send(ws, msg); err != nil {
            log.Printf("websocket.Send: %v", err)
            return
        }
    }
}

const basePkg = "github.com/tux21b/ChessBuddy"

var tmpl *template.Template
var root string = "."

var timeLimit *time.Duration = flag.Duration("time", 5*time.Minute,
    "time limit per side (sudden death, no add)")
var listenAddr *string = flag.String("http", ":8000",
    "listen on this http address")

func main() {
    flag.Parse()
    runtime.GOMAXPROCS(runtime.NumCPU())

    p, err := build.Default.Import(basePkg, "", build.FindOnly)
    if err != nil {
        log.Fatalf("Couldn't find ChessBuddy files: %v", err)
    }
    root = p.Dir

    tmpl, err = template.ParseFiles(filepath.Join(root, "chess.html"))
    if err != nil {
        log.Fatalf("Couldn't parse chess.html: %v", err)
    }

    go hookUp()

    http.HandleFunc("/", handleIndex)
    http.HandleFunc("/chess.js", handleFile("chess.js"))
    http.HandleFunc("/chess.css", handleFile("chess.css"))
    http.HandleFunc("/bg.png", handleFile("bg.png"))
    http.HandleFunc("/favicon.ico", handleFile("favicon.ico"))
    http.Handle("/ws", websocket.Handler(handleWS))

    if err := http.ListenAndServe(*listenAddr, nil); err != nil {
        log.Fatalf("http.ListenAndServe: %v", err)
    }
}
