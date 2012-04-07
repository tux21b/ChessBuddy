// ChessBuddy - Play chess with Go, HTML5, WebSockets and random strangers!
//
// Copyright (c) 2012 by Christoph Hack <christoph@tux21b.org>
// All rights reserved. Distributed under the Simplified BSD License.
//
package main

type piece int8

const (
    EMPTY piece = iota
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

type Game struct {
    board [SIZE][SIZE]piece
    hist  []string
}

func NewGame() *Game {
    return &Game{
        board: [SIZE][SIZE]piece{
            {+ROOK, +KNIGHT, +BISHOP, +QUEEN, +KING, +BISHOP, +KNIGHT, +ROOK},
            {+PAWN, +PAWN, +PAWN, +PAWN, +PAWN, +PAWN, +PAWN, +PAWN},
            {EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY},
            {EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY},
            {EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY},
            {EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY, EMPTY},
            {-PAWN, -PAWN, -PAWN, -PAWN, -PAWN, -PAWN, -PAWN, -PAWN},
            {-ROOK, -KNIGHT, -BISHOP, -QUEEN, -KING, -BISHOP, -KNIGHT, -ROOK},
        },
    }
}

// Check wethever it's valid to move the current piece located at
// (ax, ay) to a new position at (bx, by).
func (g *Game) validMove(ax, ay, bx, by int) bool {
    if ax < 0 || ax > 7 || ay < 0 || ay > 7 ||
        bx < 0 || bx > 7 || by < 0 || by > 7 ||
        g.board[ay][ax]*g.board[by][bx] > 0 ||
        (ax == bx && ay == by) ||
        (g.Turn()&1 == 0 && g.board[ay][ax] <= 0) ||
        (g.Turn()&1 == 1 && g.board[ay][ax] >= 0) {
        return false
    }
    switch g.board[ay][ax] {
    case +PAWN:
        return (ax == bx && (ay+1 == by || (ay == 1 && by == 3)) &&
            g.board[by][bx] == EMPTY) || ((ax-bx)*(ax-bx) == 1 &&
            ay+1 == by && g.board[by][bx] != EMPTY)
    case -PAWN:
        return (ax == bx && (ay-1 == by || (ay == 6 && by == 4)) &&
            g.board[by][bx] == EMPTY) || ((ax-bx)*(ax-bx) == 1 &&
            ay-1 == by && g.board[by][bx] != EMPTY)
    case +KNIGHT, -KNIGHT:
        return (ax-bx)*(ax-bx)+(ay-by)*(ay-by) == 5
    case +BISHOP, -BISHOP:
        return (ax-bx)*(ax-bx) == (ay-by)*(ay-by) && g.freeWay(ax, ay, bx, by)
    case +ROOK, -ROOK:
        return (ax == bx || ay == by) && g.freeWay(ax, ay, bx, by)
    case +QUEEN, -QUEEN:
        return (ax == bx || ay == by || (ax-bx)*(ax-bx) == (ay-by)*(ay-by)) &&
            g.freeWay(ax, ay, bx, by)
    case +KING, -KING:
        return (ax-bx)*(ax-bx) <= 1 && (ay-by)*(ay-by) <= 1
    }
    return false
}

// Move the piece located at (ax, ay) to (bx, by) if that's a valid move.
func (g *Game) Move(ax, ay, bx, by int) bool {
    if !g.validMove(ax, ay, bx, by) {
        return false
    }
    g.hist = append(g.hist, g.formatMove(ax, ay, bx, by))
    g.board[by][bx], g.board[ay][ax] = g.board[ay][ax], EMPTY
    return true
}

func (g *Game) formatMove(ax, ay, bx, by int) string {
    buf := []byte{"?NBRQK?KQRBN?"[g.board[ay][ax]+6], byte('a' + ay),
        byte('0' + ax), '-', byte('a' + by), byte('0' + ax)}
    if g.board[by][bx] != EMPTY {
        buf[3] = 'x'
    }
    if g.board[ay][ax] == +PAWN || g.board[ay][ax] == -PAWN {
        buf = buf[1:]
    }
    return string(buf)
}

func (g *Game) Turn() int {
    return len(g.hist)
}

func (g *Game) History() []string {
    return g.hist
}

// Check for obstacles at the way from (ax, ay) to (bx, by). The start and
// the end location do not count as obstacles, even if they are occupied.
func (g *Game) freeWay(ax, ay, bx, by int) bool {
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
        if g.board[y][x] != EMPTY {
            return false
        }
    }
    return true
}
