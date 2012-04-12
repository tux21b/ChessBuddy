// ChessBuddy - Play chess with Go, HTML5, WebSockets and random strangers!
//
// Copyright (c) 2012 by Christoph Hack <christoph@tux21b.org>
// All rights reserved. Distributed under the Simplified BSD License.
//
package main

import (
    "bytes"
    "fmt"
    "strings"
)

type piece int8

type pos int

func Pos(file, rank int) pos {
    return pos(file + rank<<4)
}

func (p pos) String() string {
    if p&0x88 != 0 {
        return "[invalid position]"
    }
    return fmt.Sprintf("%c%c", 'a'+p&7, '1'+p>>4)
}

// black pieces have the 4th bit set (mask 0x8)
// sliding pieces have the 3rd bit set (mask 0x4)
// orthogonal movement
const (
    Empty       piece = 0x0
    WhitePawn   piece = 0x1
    WhiteKnight piece = 0x2
    WhiteKing   piece = 0x3
    WhiteBishop piece = 0x5
    WhiteRook   piece = 0x6
    WhiteQueen  piece = 0x7
    BlackPawn   piece = 0x9
    BlackKnight piece = 0xA
    BlackKing   piece = 0xB
    BlackBishop piece = 0xD
    BlackRook   piece = 0xE
    BlackQueen  piece = 0xF
)

const (
    CheckFlag     = 0x01
    StalemateFlag = 0x02
    CheckmateFlag = 0x03
    BlackFlag     = 0x08
    castleKW      = 0x10
    castleQW      = 0x20
    castleKB      = 0x40
    castleQB      = 0x80
)

// Board stores and maintains a full chess position. In addition to the
// placement of all pieces, some additional information is required, including
// the side to move, castling rights and a possible en passant target.
type Board struct {

    // 0x88 board representation. One half of this array isn't used, but the
    // the size is neglibible and the bit-gaps drastically simplify off-board
    // checks and the validation of movement patterns.
    board [128]piece

    // status is a set of flags containing the BlackFlag, CheckFlag and
    // Stalemate Flag. Checkmate is a combination of the later two flags.
    status int

    // hist is a slice containing proper notations of applied half-moves.
    hist []string
}

// NewBoard generate a new chess board with all pieces placed on their initial
// starting position.
func NewBoard() *Board {
    return &Board{
        board: [128]piece{
            WhiteRook, WhiteKnight, WhiteBishop, WhiteQueen,
            WhiteKing, WhiteBishop, WhiteKnight, WhiteRook,
            Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty,
            WhitePawn, WhitePawn, WhitePawn, WhitePawn,
            WhitePawn, WhitePawn, WhitePawn, WhitePawn,
            Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty,
            Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty,
            Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty,
            Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty,
            Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty,
            Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty,
            Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty,
            Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty,
            Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty,
            BlackPawn, BlackPawn, BlackPawn, BlackPawn,
            BlackPawn, BlackPawn, BlackPawn, BlackPawn,
            Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty,
            BlackRook, BlackKnight, BlackBishop, BlackQueen,
            BlackKing, BlackBishop, BlackKnight, BlackRook,
            Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty,
        },
        status: castleKB | castleKW | castleQB | castleQW,
    }
}

// String returns a compact textual representation of the boards
// position using FEN (Forsythe-Edwards Notation).
func (t *Board) String() string {
    buf := &bytes.Buffer{}
    for r := pos(0x70); r >= 0; r -= 16 {
        empty := 0
        for p := r; p&0x88 == 0; p++ {
            if t.board[p] != 0 && empty > 0 {
                buf.WriteByte(byte('0' + empty))
                empty = 0
            }
            if t.board[p] != 0 {
                buf.WriteByte(" PNK?BRQ?pnk?brq"[t.board[p]])
            } else {
                empty++
            }
        }
        if empty > 0 {
            buf.WriteByte(byte('0' + empty))
        }
        if r != 0 {
            buf.WriteByte('/')
        }
    }

    if t.status&BlackFlag == 0 {
        buf.WriteString(" w ")
    } else {
        buf.WriteString(" b ")
    }

    switch {
    case t.status&castleKW != 0:
        buf.WriteByte('K')
    case t.status&castleQW != 0:
        buf.WriteByte('Q')
    case t.status&castleKB != 0:
        buf.WriteByte('k')
    case t.status&castleQB != 0:
        buf.WriteByte('q')
    default:
        buf.WriteByte('-')
    }

    fmt.Fprintf(buf, " %d %d", len(t.hist), t.Turn())

    return buf.String()
}

// Move a piece from (ax, ay) to (bx, by). The coordinates of the A1 field
// are (0, 0) and the H2 field has (7, 0). The return value indicates if the
// move was valid.
func (b *Board) Move(ax, ay, bx, by int) bool {
    if ax < 0 || ax > 7 || ay < 0 || ay > 7 ||
        bx < 0 || bx > 7 || by < 0 || by > 7 {
        return false
    }
    return b.move(Pos(ax, ay), Pos(bx, by), true, true)
}

// White returns true if the current side to move is the white one.
func (b *Board) White() bool {
    return b.status&BlackFlag == 0
}

// Turn returns the current turn number.
func (b *Board) Turn() int {
    return len(b.hist)/2 + 1
}

// Last move returns the last half move formatted using the extended algebraic
// notation.
func (b *Board) LastMove() string {
    if len(b.hist) == 0 {
        return ""
    }
    return b.hist[len(b.hist)-1]
}

func (t *Board) move(a, b pos, exec, check bool) (valid bool) {
    // only move existing pieces and do not capture own pieces
    piece, victim := t.board[a], t.board[b]
    if piece == Empty || (t.status&BlackFlag != int(piece&BlackFlag)) ||
        (victim != Empty && piece&BlackFlag == victim&BlackFlag) {
        return false
    }

    log := ""
    d, d2 := int(b-a), int((b-a)*(b-a))
    switch {
    // white pawns
    case piece == WhitePawn && ((d == 16 && victim == 0) ||
        (a>>4 == 1 && d == 32 && victim == 0) ||
        (victim != 0 && (d == 15 || d == 17))):

    // black pawns
    case piece == BlackPawn && ((d == -16 && victim == 0) ||
        (a>>4 == 6 && d == -32 && victim == 0) ||
        (victim != 0 && (d == -15 || d == -17))):

    // kings
    case piece&0x7 == WhiteKing && (d2 == 1 || (d2 >= 15*15 && d2 <= 17*17)):

    // knights
    case piece&0x7 == WhiteKnight && (d2 == 18*18 || d2 == 14*14 ||
        d2 == 31*31 || d2 == 33*33):

    // orthogonal sliding pieces (rooks and queens)
    case piece&0x6 == 0x6 && (a>>4 == b>>4 || a&7 == b&7) &&
        (t.slide(a, b, 1) || t.slide(a, b, -1) || t.slide(a, b, 16) ||
            t.slide(a, b, -16)):

    // diagonal sliding pieces (bishops and queens)
    case piece&0x5 == 0x5 && (a>>4-b>>4)*(a>>4-b>>4) == (a&7-b&7)*(a&7-b&7) &&
        (t.slide(a, b, 15) || t.slide(a, b, 17) || t.slide(a, b, -15) ||
            t.slide(a, b, -17)):

    // castling rules
    case piece == WhiteKing && a == 4 && b == 2 &&
        t.status&castleQW > 0 && t.status&CheckFlag == 0 && t.slide(4, 0, -1):
        if exec {
            log = "0-0-0"
            t.board[3], t.board[0] = WhiteRook, 0
        }
    case piece == WhiteKing && a == 4 && b == 6 &&
        t.status&castleKW > 0 && t.status&CheckFlag == 0 && t.slide(4, 7, 1):
        if exec {
            log = "0-0"
            t.board[5], t.board[7] = WhiteRook, 0
        }
    case piece == BlackKing && a == 116 && b == 114 &&
        t.status&castleQB > 0 && t.status&CheckFlag == 0 &&
        t.slide(116, 112, -1):
        if exec {
            log = "0-0-0"
            t.board[115], t.board[112] = BlackRook, 0
        }
    case piece == BlackKing && a == 116 && b == 118 &&
        t.status&castleKB > 0 && t.status&CheckFlag == 0 &&
        t.slide(116, 119, 1):
        if exec {
            log = "0-0"
            t.board[117], t.board[119] = BlackRook, 0
        }

    default:
        return false
    }

    if !(exec || check) {
        return true
    }

    if exec && log == "" {
        log = t.formatMove(a, b)
    }

    // copy the current board state and revert any changes if the move
    // turned out to be invalid or shouldn't have been executed
    backup := *t
    defer func() {
        if !valid || !exec {
            *t = backup
        }
    }()

    // apply the move
    t.board[b], t.board[a] = t.board[a], 0

    if check && t.check() {
        return false
    }

    if exec {
        t.status ^= BlackFlag
        t.status &^= CheckFlag | StalemateFlag

        switch a {
        case 0:
            t.status &^= castleQW
        case 4:
            t.status &^= castleQW | castleKW
        case 7:
            t.status &^= castleKW
        case 112:
            t.status &^= castleQB
        case 116:
            t.status &^= castleQB | castleKB
        case 119:
            t.status &^= castleKB
        }

        if t.check() {
            t.status |= CheckFlag
        }
        if t.stalemate() {
            t.status |= StalemateFlag
        }
        log = log + t.formatStatus()

        t.hist = append(t.hist, log)
    }

    return true
}

func (b *Board) slide(from, to, pattern pos) bool {
    for p := from + pattern; p&0x88 == 0; p += pattern {
        if p == to {
            return true
        } else if b.board[p] != Empty {
            break
        }
    }
    return false
}

func (b *Board) check() bool {
    end := pos(0)
    for p := pos(0); p < 128; p++ {
        if b.board[p] == WhiteKing|piece(b.status&BlackFlag) {
            end = p
            break
        }
    }
    b.status ^= BlackFlag
    for p := pos(0); p < 128; p++ {
        if p&0x88 == 0 && b.move(p, end, false, false) {
            b.status ^= BlackFlag
            return true
        }
    }
    b.status ^= BlackFlag
    return false
}

func (b *Board) stalemate() bool {
    for start := pos(0); start < 128; start++ {
        if b.board[start]&BlackFlag != piece(b.status&BlackFlag) {
            continue
        }
        for end := pos(0); end < 128; end++ {
            if b.move(start, end, false, true) {
                return false
            }
        }
    }
    return true
}

func (t *Board) formatMove(a, b pos) string {
    buf := &bytes.Buffer{}
    switch t.board[a] & 0x7 {
    case WhiteRook:
        buf.WriteByte('R')
    case WhiteKnight:
        buf.WriteByte('N')
    case WhiteBishop:
        buf.WriteByte('B')
    case WhiteQueen:
        buf.WriteByte('Q')
    case WhiteKing:
        buf.WriteByte('K')
    }

    // check if the rank or file is ambigous
    file, rank := false, false
    for p := pos(0); p < 128; p++ {
        if t.board[p] == t.board[a] && p != a && t.move(p, b, false, false) {
            if p&7 != a&7 {
                file = true
            } else {
                rank = true
            }
        }
    }
    // pawn captures always include the file, even if not ambigous
    if file || (t.board[a]&0x7 == WhitePawn && t.board[b] != 0) {
        buf.WriteByte('a' + byte(a&7))
    }
    if rank {
        buf.WriteByte('1' + byte(a>>4))
    }

    if t.board[b] != 0 {
        buf.WriteByte('x')
    }

    buf.Write([]byte{byte('a' + b&7), byte('1' + b>>4)})

    return buf.String()
}

func (t *Board) formatStatus() string {
    if t.status&CheckmateFlag == CheckmateFlag {
        return "#"
    } else if t.status&CheckFlag != 0 {
        return "+"
    }
    return ""
}

func (t *Board) MoveSAN(san string) bool {
    // ignore annotations
    san = strings.TrimRight(san, "?!+#")

    // handle special moves (castling)
    switch {
    case san == "0-0-0" && t.White():
        return t.move(4, 2, true, true)
    case san == "0-0-0" && !t.White():
        return t.move(116, 114, true, true)
    case san == "0-0" && t.White():
        return t.move(4, 6, true, true)
    case san == "0-0" && !t.White():
        return t.move(116, 118, true, true)
    }

    ax, ay := -1, -1
    piece := WhitePawn

    if len(san) > 0 && san[0] >= 'A' && san[0] <= 'Z' {
        switch san[0] {
        case 'K':
            piece = WhiteKing
        case 'Q':
            piece = WhiteQueen
        case 'B':
            piece = WhiteBishop
        case 'N':
            piece = WhiteKnight
        case 'R':
            piece = WhiteRook
        default:
            return false
        }
        san = san[1:]
    }

    if t.status&BlackFlag != 0 {
        piece |= BlackFlag
    }

    b := pos(0)
    if l := len(san); l < 2 || san[l-2] < 'a' || san[l-2] > 'h' ||
        san[l-1] < '1' || san[l-1] > '8' {
        return false
    } else {
        b = Pos(int(san[l-2]-'a'), int(san[l-1]-'1'))
    }

    san = strings.TrimRight(san[:len(san)-2], "-x")

    if len(san) > 0 && san[0] >= 'a' && san[0] <= 'h' {
        ax = int(san[0] - 'a')
        san = san[1:]
    }
    if len(san) > 0 && san[0] >= '1' && san[0] <= '9' {
        ay = int(san[0] - '1')
        san = san[1:]
    }

    if len(san) > 0 {
        return false
    }

    a := Pos(ax, ay)
    if ax < 0 || ay < 0 {
        for p := pos(0); p < 128; p++ {
            if t.board[p] == piece && (ax < 0 || int(p&7) == ax) &&
                (ay < 0 || int(p>>4) == ay) && t.move(p, b, false, false) {
                a = p
            }
        }
    }

    if a < 0 {
        return false
    }

    return t.move(a, b, true, true)
}
