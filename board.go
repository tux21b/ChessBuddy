// ChessBuddy - Play chess with Go, HTML5, WebSockets and random strangers!
//
// Copyright (c) 2012 by Christoph Hack <christoph@tux21b.org>
// All rights reserved. Distributed under the Simplified BSD License.
//
package main

import "bytes"

type piece int8

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
    CheckFlag     = 0x1
    StalemateFlag = 0x2
    CheckmateFlag = 0x3
    BlackFlag     = 0x8
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
    }
}

// Move a piece from (ax, ay) to (bx, by). The coordinates of the A1 field
// are (0, 0) and the H2 field has (7, 0). The return value indicates if the
// move was valid.
func (b *Board) Move(ax, ay, bx, by int) bool {
    if ax < 0 || ax > 7 || ay < 0 || ay > 7 ||
        bx < 0 || bx > 7 || by < 0 || by > 7 {
        return false
    }
    return b.move(ax+16*ay, bx+16*by, true, true)
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

func (b *Board) move(from, to int, exec, check bool) bool {
    // only move existing pieces and do not capture own pieces
    piece, victim := b.board[from], b.board[to]
    if piece == Empty || (b.status&BlackFlag != int(piece&BlackFlag)) ||
        (victim != Empty && piece&BlackFlag == victim&BlackFlag) {
        return false
    }

    // check basic movement patterns (incl. sliding)
    d := to - from
    d2 := d * d
    switch {
    case piece == WhitePawn && (d == 16 ||
        (from>>4 == 1 && d == 32 && victim == Empty) ||
        (victim != Empty && (d == 15 || d == 17))):
    case piece == BlackPawn && (d == -16 ||
        (from>>4 == 6 && d == -32 && victim == Empty) ||
        (victim != Empty && (d == -15 || d == -17))):
    case piece&0x7 == WhiteKnight && (d2 == 18*18 || d2 == 14*14 ||
        d2 == 31*31 || d2 == 33*33):
    case piece&0x7 == WhiteKing && d2 == 1:
    case (piece&0x6 == 0x6 && (from>>4 == to>>4 || from&7 == to&7) &&
        (b.slide(from, to, 1) || b.slide(from, to, -1) ||
            b.slide(from, to, 16) || b.slide(from, to, -16))) ||
        (piece&0x5 == 0x5 &&
            (from>>4-to>>4)*(from>>4-to>>4) == (from&7-to&7)*(from&7-to&7) &&
            (b.slide(from, to, 15) || b.slide(from, to, 17) ||
                b.slide(from, to, -15) || b.slide(from, to, -17))):
    default:
        return false
    }

    // try to apply the move
    if exec || check {
        backup := b.board
        b.board[to], b.board[from] = b.board[from], Empty

        if check && b.check() {
            b.board = backup
            return false
        }

        if exec {
            b.status ^= BlackFlag
            b.status &^= CheckFlag | StalemateFlag

            if b.check() {
                b.status |= CheckFlag
            }
            if b.stalemate() {
                b.status |= StalemateFlag
            }

            b.hist = append(b.hist,
                b.formatMove(piece, victim, from, to, b.status))
        } else {
            b.board = backup
        }
    }
    return true
}

func (b *Board) slide(from, to, pattern int) bool {
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
    end := 0
    for p := 0; p < 128; p++ {
        if b.board[p] == WhiteKing|piece(b.status&BlackFlag) {
            end = p
            break
        }
    }
    b.status ^= BlackFlag
    for p := 0; p < 128; p++ {
        if p&0x88 == 0 && b.move(p, end, false, false) {
            b.status ^= BlackFlag
            return true
        }
    }
    b.status ^= BlackFlag
    return false
}

func (b *Board) stalemate() bool {
    for start := 0; start < 128; start++ {
        if b.board[start]&BlackFlag != piece(b.status&BlackFlag) {
            continue
        }
        for end := 0; end < 128; end++ {
            if b.move(start, end, false, true) {
                return false
            }
        }
    }
    return true
}

func (b *Board) formatMove(piece, victim piece, from, to, status int) string {
    buf := &bytes.Buffer{}
    switch piece & 0x7 {
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
    buf.Write([]byte{byte('a' + from&7), byte('1' + from>>4)})
    if victim != Empty {
        buf.WriteByte('x')
    } else {
        buf.WriteByte('-')
    }
    buf.Write([]byte{byte('a' + to&7), byte('1' + to>>4)})
    if status&CheckmateFlag == CheckmateFlag {
        buf.WriteByte('#')
    } else if status&CheckFlag != 0 {
        buf.WriteByte('+')
    }
    return buf.String()
}
