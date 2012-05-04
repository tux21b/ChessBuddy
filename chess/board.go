// Copyright (c) 2012 by Christoph Hack <christoph@tux21b.org>
// All rights reserved. Distributed under the Simplified BSD License.

// Package chess implements a basic chess board which represents the state of
// a chess game. It can be used for validating and applying moves, formatting
// them using SAN (standard algebraic notation) and to generate a list of
// possible moves.
//
// The package doesn't provide a way to rank and choose moves, but further
// packages might be built on top of this one to add this functionality.
//
package chess

import (
    "bytes"
    "fmt"
    "regexp"
    "strings"
)

// The chess pieces are identified by a single letter from the standard
// English names (i.e. pawn, knight, bishop, rook, queen, king). White pieces
// have the 3rd bit set, black pieces the 4th. The bitmasks PieceMask and
// ColorMask can be used to extract the piece or color information.
const (
    P uint8 = 0x1
    N uint8 = 0x2
    B uint8 = 0x3
    R uint8 = 0x4
    Q uint8 = 0x5
    K uint8 = 0x6

    White uint8 = 0x08
    Black uint8 = 0x10

    PieceMask uint8 = 0x07
    ColorMask uint8 = 0x18
)

// A square represents a position on the chess board.
type Square int

// Sq parses a position on the chess board and returns that square. It will
// panic if the input doesn't match the expression "[a-h][1-9]".
func Sq(v string) Square {
    if len(v) != 2 || v[0] < 'a' || v[0] > 'h' || v[1] < '1' || v[1] > '9' {
        panic("invalid square")
    }
    return Square((v[1]-'1')*8 + v[0] - 'A')
}

// File returns the column number (ranging from 0 to 7) of the square.
func (s Square) File() int {
    return int(s & 7)
}

// Rank returns the row number (ranging from 0 to 7) of the square.
func (s Square) Rank() int {
    return int(s >> 3)
}

// String formats the square using the standard algebraic notation.
func (s Square) String() string {
    return fmt.Sprintf("%c%c", 'a'+s&7, '1'+s>>3)
}

// A Bitboard is a good alternative to square centric representations of the
// chessboard. The 8x8 bits are used to represent the existance of a certain
// piece at that position. They are also usefull for lookup-tables storing
// possible target locations because of they are quite compact.
type Bitboard uint64

// String will format the bitboard by using ASCII art to draw the chessboard.
// Only useful for debugging purposes.
func (b Bitboard) String() string {
    buf := &bytes.Buffer{}
    buf.WriteString("  A B C D E F G H\n")
    for rank := 7; rank >= 0; rank-- {
        buf.WriteByte('1' + byte(rank))
        for file := 0; file <= 7; file++ {
            buf.WriteByte(' ')
            if b&(1<<Bitboard(rank<<3+file)) != 0 {
                buf.WriteByte('x')
            } else {
                buf.WriteByte('.')
            }
        }
        buf.WriteByte('\n')
    }
    return buf.String()
}

// Board stores and maintains a full chess position. In addition to the
// placement of all pieces, some additional information is required, including
// the side to move, castling rights and a possible en passant target.
type Board struct {

    // board is a square centric representation of all pieces.
    board [64]uint8

    // occupied is a piece centric representation of all occupied squares.
    occupied Bitboard

    // moved tracks pieces which have been moved to determine castling
    // rights
    moved Bitboard

    // color of the current side to move
    color uint8

    // possible square for en-passant captures
    eps Square

    // is the current player in check or stalemate?
    check, stalemate bool

    // hist is a slice containing proper notations of applied half-moves.
    hist []string
}

// NewBoard generates a new chess board with all pieces placed on their
// initial starting position.
func NewBoard() *Board {
    return &Board{
        board: [64]uint8{
            R | White, N | White, B | White, Q | White,
            K | White, B | White, N | White, R | White,
            P | White, P | White, P | White, P | White,
            P | White, P | White, P | White, P | White,
            0, 0, 0, 0, 0, 0, 0, 0,
            0, 0, 0, 0, 0, 0, 0, 0,
            0, 0, 0, 0, 0, 0, 0, 0,
            0, 0, 0, 0, 0, 0, 0, 0,
            P | Black, P | Black, P | Black, P | Black,
            P | Black, P | Black, P | Black, P | Black,
            R | Black, N | Black, B | Black, Q | Black,
            K | Black, B | Black, N | Black, R | Black,
        },
        occupied: 0xffff00000000ffff,
        color:    White,
        eps:      -1,
    }
}

// String returns a compact textual representation of the boards
// position using FEN (Forsythe-Edwards Notation).
func (b *Board) String() string {
    buf := &bytes.Buffer{}
    for rank := 7; rank >= 0; rank-- {
        empty := 0
        for file := 0; file <= 7; file++ {
            if piece := b.board[file+rank<<3]; piece != 0 {
                if empty > 0 {
                    buf.WriteByte(byte('0' + empty))
                    empty = 0
                }
                switch piece & ColorMask {
                case White:
                    buf.WriteByte(" PNBRQK"[piece&PieceMask])
                case Black:
                    buf.WriteByte(" pnbrqk"[piece&PieceMask])
                }
            } else {
                empty++
            }
        }
        if empty > 0 {
            buf.WriteByte(byte('0' + empty))
        }
        if rank != 0 {
            buf.WriteByte('/')
        }
    }
    switch b.color {
    case White:
        buf.WriteString(" w ")
    case Black:
        buf.WriteString(" b ")
    }
    switch {
    case b.moved&0x90 == 0:
        buf.WriteByte('K')
    case b.moved&0x11 == 0:
        buf.WriteByte('Q')
    case b.moved&(0x90<<14) == 0:
        buf.WriteByte('k')
    case b.moved&(0x11<<14) == 0:
        buf.WriteByte('q')
    default:
        buf.WriteByte('-')
    }
    fmt.Fprintf(buf, " %d %d", len(b.hist), b.Turn())
    return buf.String()
}

var reSAN = regexp.MustCompile(`^([PNBRQK]?)([a-h])?([1-8])?([\-x]?)([a-h])([1-8])$`)

// MoveSAN applies a move given in the SAN (standard algebraic notation) format.
func (b *Board) MoveSAN(text string) error {
    san := strings.Replace(strings.TrimRight(text, "?!+#"), "O", "0", -1)
    if san == "0-0" || san == "0-0-0" {
        switch {
        case san == "0-0" && b.color == White && b.doCastle(4, 7):
        case san == "0-0" && b.color == Black && b.doCastle(60, 63):
        case san == "0-0-0" && b.color == White && b.doCastle(4, 0):
        case san == "0-0-0" && b.color == Black && b.doCastle(60, 56):
        default:
            return fmt.Errorf("can not castle")
        }
        return nil
    }

    m := reSAN.FindStringSubmatch(san)
    if m == nil {
        return fmt.Errorf("invalid move text %q. Please use SAN.", text)
    }

    dst := Square(m[5][0] - 'a' + (m[6][0]-'1')<<3)
    if m[4] == "x" && b.board[dst]&ColorMask != b.color^ColorMask {
        return fmt.Errorf("can not capture the square %s", dst)
    }

    piece := P | b.color
    switch m[1] {
    case "N":
        piece = N | b.color
    case "B":
        piece = B | b.color
    case "R":
        piece = R | b.color
    case "Q":
        piece = Q | b.color
    case "K":
        piece = K | b.color
    }

    src := Square(-1)
    if m[2] != "" && m[3] != "" {
        src = Square(m[2][0] - 'a' + (m[3][0]-'1')<<3)
    } else {
        for p := Square(0); p < 64; p++ {
            if b.board[p] == piece && (m[2] == "" || m[2][0]-'a' == uint8(p&7)) &&
                (m[3] == "" || m[3][0]-'1' == uint8(p>>3)) && b.mayMove(p, dst) {
                if src < 0 {
                    src = p
                } else {
                    return fmt.Errorf("The move %q is ambigous.", text)
                }
            }
        }
    }
    if src < 0 || !b.Move(src, dst) {
        return fmt.Errorf("The move %q is invalid.", text)
    }
    return nil
}

// Move moves a piece from square src to the square dst. The return value
// indicates whetever the move was sucessful or not.
func (b *Board) Move(src, dst Square) bool {
    if src < 0 || dst >= 64 || src < 0 || dst >= 64 {
        return false
    }

    if src == 4 && b.board[src] == K|White {
        switch dst {
        case 6:
            return b.doCastle(4, 7)
        case 2:
            return b.doCastle(4, 0)
        }
    } else if src == 60 && b.board[src] == K|Black {
        switch dst {
        case 62:
            return b.doCastle(60, 63)
        case 58:
            return b.doCastle(60, 56)
        }
    }

    if !b.canMove(src, dst) {
        return false
    }

    log := b.formatMove(src, dst)
    b.board[dst], b.board[src] = b.board[src], 0
    b.occupied &^= Bitboard(1) << uint(src)
    b.occupied |= Bitboard(1) << uint(dst)

    // additional rules for en-passant captures
    if b.board[dst] == P|White && dst == b.eps {
        b.board[dst-8] = 0
        b.occupied &^= Bitboard(1) << uint(dst-8)
    } else if b.board[dst] == P|Black && dst == b.eps {
        b.board[dst+8] = 0
        b.occupied &^= Bitboard(1) << uint(dst+8)
    }
    b.eps = -1
    if b.board[dst] == P|White && dst-src == 16 {
        b.eps = dst - 8
    } else if b.board[dst] == P|Black && dst-src == -16 {
        b.eps = dst + 8
    }

    // promotion
    if b.board[dst]&PieceMask == P && (dst>>3 == 0 || dst>>3 == 7) {
        b.board[dst] = Q | (b.board[dst] & ColorMask)
    }

    b.moved |= Bitboard(1) << uint(src)
    b.color ^= ColorMask
    b.check, b.stalemate = b.isCheck(), b.isStalemate()
    b.hist = append(b.hist, log+b.formatStatus())

    return true
}

// Moves generates a list of all possible target squares for a specific piece
// located at the square src.
func (b *Board) Moves(src Square) (moves []Square) {
    for dst := Square(0); dst < 64; dst++ {
        if b.canMove(src, dst) {
            moves = append(moves, dst)
        }
    }
    if b.board[src] == K|White {
        if b.canCastle(4, 7) {
            moves = append(moves, 6)
        }
        if b.canCastle(4, 0) {
            moves = append(moves, 2)
        }
    } else if b.board[src] == K|Black {
        if b.canCastle(60, 63) {
            moves = append(moves, 62)
        }
        if b.canCastle(60, 56) {
            moves = append(moves, 58)
        }
    }
    return
}

// mayMove checks whetever it might be possible to move from src to dst. This
// method ignores castling rules and might report pseud-legal moves.
func (b *Board) mayMove(src, dst Square) bool {
    piece, victim := b.board[src], b.board[dst]

    // must not capture own pieces
    if piece&ColorMask == victim&ColorMask {
        return false
    }

    // check basic movement patterns
    x88diff := int(dst - src + (dst | 7) - (src | 7) + 120)
    occ := b.occupied>>Bitboard(src) | b.occupied<<Bitboard(64-src)
    if blockers[piece&PieceMask][x88diff]&occ != 0 {
        return false
    }

    // additional rules for pawn movements and captures
    if piece&PieceMask == P &&
        ((b.board[dst] == 0 && src&7 != dst&7 && dst != b.eps) ||
            (piece == P|White && (src > dst || (x88diff == 152 && src>>3 != 1))) ||
            (piece == P|Black && (src < dst || (x88diff == 88 && src>>3 != 6)))) {
        return false
    }

    return true
}

// canMove checks if its possible to move from src to dst. This method ignores
// castling rules.
func (b *Board) canMove(src, dst Square) (valid bool) {
    if !b.mayMove(src, dst) {
        return false
    }

    piece, victim := b.board[src], b.board[dst]
    b.board[dst], b.board[src] = piece, 0
    b.occupied &^= Bitboard(1) << uint(src)
    b.occupied |= Bitboard(1) << uint(dst)

    valid = !b.isCheck()

    b.board[src], b.board[dst] = piece, victim
    b.occupied |= Bitboard(1) << uint(src)
    if victim == 0 {
        b.occupied &^= Bitboard(1) << uint(dst)
    }

    return
}

// canCastle checks if its possible to castle with the given king and
// rook position.
func (b *Board) canCastle(king, rook Square) (valid bool) {
    if b.moved&((Bitboard(1)<<uint(king))|(Bitboard(1)<<uint(rook))) != 0 {
        return false
    }
    nking, nrook, step := king+2, rook-2, Square(1)
    if rook < king {
        nking, nrook, step = king-2, rook+3, Square(-1)
    }

    // one cannot castle out of, through, or into check
    if b.check || !b.mayMove(rook, nrook) {
        return false
    }
    for i := king; i != nking; i += step {
        if !b.canMove(i, i+step) {
            b.board[king], b.board[i] = b.board[i], 0
            return false
        }
        b.board[i+step], b.board[i] = b.board[i], 0
    }

    b.board[nking], b.board[king] = 0, b.board[nking]
    return true
}

// doCastle applies a castling move if possible.
func (b *Board) doCastle(king, rook Square) (valid bool) {
    if !b.canCastle(king, rook) {
        return false
    }

    nking, nrook, log := king+2, rook-2, "0-0"
    if rook < king {
        nking, nrook, log = king-2, rook+3, "0-0-0"
    }

    b.board[nking], b.board[king] = b.board[king], 0
    b.board[nrook], b.board[rook] = b.board[rook], 0
    b.occupied &^= (Bitboard(1) << uint(king)) | (Bitboard(1) << uint(rook))
    b.occupied |= (Bitboard(1) << uint(nking)) | (Bitboard(1) << uint(nrook))
    b.moved |= (Bitboard(1) << uint(king)) | (Bitboard(1) << uint(rook))
    b.color ^= ColorMask
    b.hist = append(b.hist, log+b.formatStatus())

    return true
}

// isCheck returns true if the current player is in check.
func (b *Board) isCheck() bool {
    dst, piece := Square(0), K|b.color
    for p := Square(0); p < 64; p++ {
        if b.board[p] == piece {
            dst = p
            break
        }
    }
    opponent := b.color ^ ColorMask
    for src := Square(0); src < 64; src++ {
        if b.board[src]&ColorMask == opponent && b.mayMove(src, dst) {
            return true
        }
    }
    return false
}

// isStalemate returns true if the current player can not make any moves
// anymore.
func (b *Board) isStalemate() bool {
    for src := Square(0); src < 64; src++ {
        if b.board[src]&ColorMask != b.color {
            continue
        }
        for dst := Square(0); dst < 64; dst++ {
            if b.canMove(src, dst) {
                return false
            }
        }
    }
    return true
}

// formatMove formats a move from src to dst according to SAN. This method
// doesn't support formatting of castling moves and it must be called before
// the move was applied to dissolve ambiguity and to format captures properly.
func (b *Board) formatMove(src, dst Square) string {
    buf := &bytes.Buffer{}
    if x := b.board[src] & PieceMask; x != P {
        buf.WriteByte(" PNBRQK"[x])
    }

    // check if the rank or file is ambigous
    file, rank := false, false
    for p := Square(0); p < 64; p++ {
        if b.board[p] == b.board[src] && p != src && b.mayMove(p, dst) {
            if p&7 != src&7 {
                file = true
            } else {
                rank = true
            }
        }
    }
    // pawn captures always include the file, even if not ambigous
    capture := b.board[dst] != 0 || (b.board[src]&PieceMask == P && b.eps == dst)
    if file || (b.board[src]&PieceMask == P && capture) {
        buf.WriteByte('a' + byte(src&7))
    }
    if rank {
        buf.WriteByte('1' + byte(src>>3))
    }

    if capture {
        buf.WriteByte('x')
    }

    buf.Write([]byte{byte('a' + dst&7), byte('1' + dst>>3)})

    return buf.String()
}

// formatStatus returns the proper SAN annotations for moves which result
// in a check or checkmate.
func (b *Board) formatStatus() string {
    if b.check {
        if b.stalemate {
            return "#"
        } else {
            return "+"
        }
    }
    return ""
}

// Checkmate returns true if the current player is checkmate.
func (b *Board) Checkmate() bool {
    return b.check && b.stalemate
}

// Stalemate returns true if the current player is stalemate.
func (b *Board) Stalemate() bool {
    return !b.check && b.stalemate
}

// Check returns true if the current player is in check only. This method
// returns false if the player is checkmate.
func (b *Board) Check() bool {
    return b.check && !b.stalemate
}

// Color returns the color of the current side to play.
func (b *Board) Color() uint8 {
    return b.color
}

// Turn returns the current halfturn number starting by one.
func (b *Board) Turn() int {
    return len(b.hist) + 1
}

// LastMove returns the last half move formatted using the extended algebraic
// notation.
func (b *Board) LastMove() string {
    if len(b.hist) == 0 {
        return ""
    }
    return b.hist[len(b.hist)-1]
}

// blockers is a relatively small lookup table (just 14 KB) which stores for
// each piece and 0x88 difference a set of possible blockers, i.e. squares
// which can not be passed if they are non-empty. Impossible moves are blocked
// by all other squares and non sliding moves are blocked by nothing.
var blockers [7][240]Bitboard

// init initializes the blockers lookup table.
func init() {
    for i := 0; i < 240; i++ {
        blockers[0][i] = ^Bitboard(0)
        blockers[P][i] = ^Bitboard(0)
        blockers[N][i] = ^Bitboard(0)
        blockers[B][i] = ^Bitboard(0)
        blockers[R][i] = ^Bitboard(0)
        blockers[Q][i] = ^Bitboard(0)
        blockers[K][i] = ^Bitboard(0)
    }

    // pawns
    blockers[P][136] = 1 << 8
    blockers[P][152] = 1<<8 | 1<<16
    blockers[P][135] = 0
    blockers[P][137] = 0
    blockers[P][104] = 1 << 56
    blockers[P][88] = 1<<56 | 1<<48
    blockers[P][103] = 0
    blockers[P][105] = 0

    // knights
    blockers[N][153] = 0
    blockers[N][151] = 0
    blockers[N][138] = 0
    blockers[N][134] = 0
    blockers[N][106] = 0
    blockers[N][102] = 0
    blockers[N][89] = 0
    blockers[N][87] = 0

    // bishops
    blockers[B][137] = 0
    blockers[B][135] = 0
    blockers[B][105] = 0
    blockers[B][103] = 0

    // rooks
    blockers[R][121] = 0
    blockers[R][136] = 0
    blockers[R][119] = 0
    blockers[R][104] = 0

    // queens
    blockers[Q][121] = 0
    blockers[Q][136] = 0
    blockers[Q][119] = 0
    blockers[Q][104] = 0
    blockers[Q][137] = 0
    blockers[Q][135] = 0
    blockers[Q][105] = 0
    blockers[Q][103] = 0

    // kings
    blockers[K][137] = 0
    blockers[K][136] = 0
    blockers[K][135] = 0
    blockers[K][121] = 0
    blockers[K][119] = 0
    blockers[K][105] = 0
    blockers[K][104] = 0
    blockers[K][103] = 0

    // complete movement patterns of sliding pieces (bishops, rooks, queens)
    for _, p := range []uint8{B, R, Q} {
        for i := 1; i < 7; i++ {
            blockers[p][120+(i+1)*1] = blockers[p][120+i*1] | 1<<uint(i*1)
            blockers[p][120-(i+1)*1] = blockers[p][120-i*1] | 1<<uint(64-i*1)
            blockers[p][120+(i+1)*15] = blockers[p][120+i*15] | 1<<uint(i*7)
            blockers[p][120-(i+1)*15] = blockers[p][120-i*15] | 1<<uint(64-i*7)
            blockers[p][120+(i+1)*16] = blockers[p][120+i*16] | 1<<uint(i*8)
            blockers[p][120-(i+1)*16] = blockers[p][120-i*16] | 1<<uint(64-i*8)
            blockers[p][120+(i+1)*17] = blockers[p][120+i*17] | 1<<uint(i*9)
            blockers[p][120-(i+1)*17] = blockers[p][120-i*17] | 1<<uint(64-i*9)
        }
    }
}
