// ChessBuddy - Play chess with Go, HTML5, WebSockets and random strangers!
//
// Copyright (c) 2012 by Christoph Hack <christoph@tux21b.org>
// All rights reserved. Distributed under the Simplified BSD License.
//
package main

import (
    "code.google.com/p/go.net/websocket"
    "flag"
    "fmt"
    "html/template"
    "log"
    "math/rand"
    "net"
    "net/http"
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
    Color                  int
    NumPlayers             int32
    History                string
    RemainingA, RemainingB time.Duration
    Text                   string
}

type Player struct {
    Conn      *websocket.Conn
    Color     int
    Remaining time.Duration
    Out       chan<- Message
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

func (p Player) String() string {
    if p.Color == WHITE {
        return "White"
    } else if p.Color == BLACK {
        return "Black"
    }
    return "Unknown"
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
            close(a.Out)
            a = b
        }
    }
}

func play(a, b Player) {
    defer func() {
        close(a.Out)
        close(b.Out)
    }()

    log.Println("Starting new game")

    game := NewGame()
    if rand.Float32() > 0.5 {
        a, b = b, a
    }
    a.Color = WHITE
    b.Color = BLACK
    a.Remaining = *timeLimit
    b.Remaining = *timeLimit

    a.Out <- Message{Cmd: "start", Color: a.Color, Turn: game.Turn(),
        NumPlayers: atomic.LoadInt32(&numPlayers),
        RemainingA: a.Remaining, RemainingB: b.Remaining}
    b.Out <- Message{Cmd: "start", Color: b.Color, Turn: game.Turn(),
        NumPlayers: atomic.LoadInt32(&numPlayers),
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
        if msg.Cmd == "move" && msg.Turn == game.Turn()+1 &&
            game.Move(msg.Ax, msg.Ay, msg.Bx, msg.By) {

            msg.History = game.History()[len(game.History())-1]
            msg.NumPlayers = atomic.LoadInt32(&numPlayers)
            if game.Turn()&1 == 1 {
                msg.History = fmt.Sprintf("%d. %s", (game.Turn()+1)/2, msg.History)
            }
            now := time.Now()
            a.Remaining -= now.Sub(start)
            if a.Remaining <= 10*time.Millisecond {
                a.Remaining = 10 * time.Millisecond
            }
            start = now
            msg.RemainingA, msg.RemainingB = a.Remaining, b.Remaining
            if a.Color == BLACK {
                msg.RemainingA, msg.RemainingB = b.Remaining, a.Remaining
            }
            a, b = b, a
            a.Out <- msg
            b.Out <- msg
        }
    }
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

func handleWS(ws *websocket.Conn) {
    count := atomic.AddInt32(&numPlayers, 1)
    defer atomic.AddInt32(&numPlayers, -1)
    defer ws.Close()

    msg := Message{
        Cmd:        "msg",
        Text:       "Waiting for another player...",
        NumPlayers: count,
    }
    if err := websocket.JSON.Send(ws, msg); err != nil {
        return
    }

    out := make(chan Message, 1)
    available <- Player{Conn: ws, Out: out}

    for msg := range out {
        if err := websocket.JSON.Send(ws, msg); err != nil {
            log.Printf("websocket.Send: %v", err)
            return
        }
    }
}

var timeLimit *time.Duration = flag.Duration("time", 5*time.Minute,
    "time limit per side (sudden death, no add)")
var listenAddr *string = flag.String("http", ":8000",
    "listen on this http address")

func main() {
    flag.Parse()

    http.HandleFunc("/", handleIndex)
    http.HandleFunc("/chess.js", handleFile("chess.js"))
    http.HandleFunc("/chess.css", handleFile("chess.css"))
    http.HandleFunc("/bg.png", handleFile("bg.png"))
    http.Handle("/ws", websocket.Handler(handleWS))

    go hookUp()

    if err := http.ListenAndServe(*listenAddr, nil); err != nil {
        log.Fatalf("http.ListenAndServe: %v", err)
    }
}
