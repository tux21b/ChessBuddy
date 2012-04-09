package main

import (
    "testing"
)

func mv(t *testing.T, b *Board, ax, ay, bx, by int, log string) {
    if !b.Move(ax, ay, bx, by) && log != "" {
        t.Errorf("the move from (%d,%d) to (%d,%d) failed.", ax, ay, bx, by)
        return
    }
    if b.LastMove() != log {
        t.Errorf("invalid log entry. expected %q, got %q", log, b.LastMove())
        return
    }
}

func TestFoolsMate(t *testing.T) {
    b := NewBoard()
    mv(t, b, 5, 1, 5, 2, "f2-f3")
    mv(t, b, 4, 6, 4, 5, "e7-e6")
    mv(t, b, 6, 1, 6, 3, "g2-g4")
    mv(t, b, 3, 7, 7, 3, "Qd8-h4#")
}

// a=0  b=1  c=2  d=3  e=4  f=5  g=6  h=7
func TestImmortalLosingGame(t *testing.T) {
    b := NewBoard()
    mv(t, b, 3, 1, 3, 3, "d2-d4")
    mv(t, b, 5, 6, 5, 4, "f7-f5")
    mv(t, b, 6, 1, 6, 2, "g2-g3")
    mv(t, b, 6, 6, 6, 5, "g7-g6")
    mv(t, b, 5, 0, 6, 1, "Bf1-g2")
    mv(t, b, 5, 7, 6, 6, "Bf8-g7")
    mv(t, b, 1, 0, 2, 2, "Nb1-c3")
    mv(t, b, 6, 7, 5, 5, "Ng8-f6")
    mv(t, b, 2, 0, 6, 4, "Bc1-g5")
    mv(t, b, 1, 7, 2, 5, "Nb8-c6")
    mv(t, b, 3, 0, 3, 1, "Qd1-d2")
    mv(t, b, 3, 6, 3, 5, "d7-d6")
    mv(t, b, 7, 1, 7, 3, "h2-h4")
    mv(t, b, 4, 6, 4, 5, "e7-e6")
    // 8. 0-0-0 h6
    // 9. Bf4 Bd7
    // 10. e4 fxe4
    // 11. Nxe4 Nd5
    // 12. Ne2 Qe7
    // 13. c4 Nb6?
    // 14. c5! dxc5
    // 15. Bxc7! 0-0
    // 16. Bd6 Qf7
    // 17. Bxf8 Rxf8
    // 18. dxc5 Nd5
    // 19. f4 Rd8
    // 20. N2c3 Ndb4
    // 21. Nd6 Qf8
    // 22. Nxb7 Nd4!
    // 23. Nxd8 Bb5!
    // 24. Nxe6! Bd3!
    // 25. Bd5! Qf5!
    // 26. Nxd4+ Qxd5!
    // 27. Nc2! Bxc3
    // 28. bxc3! Qxa2 29. cxb4!
    // 29. Nxb4 Qb1#
}
