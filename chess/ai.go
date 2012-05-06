// Copyright (c) 2012 by Christoph Hack <christoph@tux21b.org>
// All rights reserved. Distributed under the Simplified BSD License.

package chess

import (
    "math"
    "math/rand"
)

func (b *Board) MoveAI() (src, dst Square) {
    src, dst, _ = b.negaMax(4)
    return
}

func (b *Board) negaMax(depth int) (bsrc, bdst Square, max float64) {
    if depth <= 0 {
        max = b.evaluate()
        return
    }

    max = math.Inf(-1)
    src := Square(rand.Intn(64))
    for i := 0; i < 64; i++ {
        src = (src + 1) % 64
        if b.board[src]&ColorMask != b.color {
            continue
        }
        dst := Square(rand.Intn(64))
        for j := 0; j < 64; j++ {
            dst = (dst + 1) % 64
            if b.mayMove(src, dst) {

                piece, victim := b.board[src], b.board[dst]
                b.board[dst], b.board[src] = piece, 0
                b.occupied &^= Bitboard(1) << uint(src)
                b.occupied |= Bitboard(1) << uint(dst)

                if !b.isCheck() {
                    b.color ^= ColorMask
                    _, _, score := b.negaMax(depth - 1)
                    score = -score
                    b.color ^= ColorMask

                    if score > max {
                        bsrc, bdst, max = src, dst, score
                    }
                }

                b.board[src], b.board[dst] = piece, victim
                b.occupied |= Bitboard(1) << uint(src)
                if victim == 0 {
                    b.occupied &^= Bitboard(1) << uint(dst)
                }
            }
        }
    }
    return
}

func (b *Board) evaluate() float64 {
    values := []float64{0, 1, 3, 3, 5, 9, 200}
    score := 0.0
    for p := Square(0); p < 64; p++ {
        s := values[b.board[p]&PieceMask]
        if (p>>3 == 0 || p>>3 == 7) && b.board[p]|PieceMask == P {
            s = 9
        }
        if b.board[p]&ColorMask != b.color {
            s = -s
        }
        score += s
    }
    return score
}
