package main

import (
    "strings"
    "testing"
)

func testGame(t *testing.T, text string) {
    b := NewBoard()
    for i, mv := range strings.Fields(text) {
        if i%3 == 0 {
            continue // skip turn numbers
        }
        prev := *b
        if !b.MoveSAN(mv) {
            t.Fatalf("the move %q failed. board=%q", mv, b)
        }
        mv = strings.Trim(mv, "!?")
        if log := b.LastMove(); log != mv {
            t.Errorf("unexpected log entry. want=%q, got=%q, board=%q",
                mv, log, &prev)
        }
    }
}

func TestFoolsMate(t *testing.T) {
    testGame(t, "1. e4 g5 2. d4 f6 3. Qh5#")
}

func TestImmortalLosingGame(t *testing.T) {
    testGame(t, `1. d4 f5 2. g3 g6 3. Bg2 Bg7 4. Nc3 Nf6 5. Bg5 Nc6 6. Qd2 d6
        7. h4 e6 8. 0-0-0 h6 9. Bf4 Bd7 10. e4 fxe4 11. Nxe4 Nd5 12. Ne2 Qe7
        13. c4 Nb6? 14. c5! dxc5 15. Bxc7! 0-0 16. Bd6 Qf7 17. Bxf8 Rxf8
        18. dxc5 Nd5 19. f4 Rd8 20. N2c3 Ndb4 21. Nd6 Qf8 22. Nxb7 Nd4!
        23. Nxd8 Bb5! 24. Nxe6! Bd3! 25. Bd5! Qf5! 26. Nxd4+ Qxd5!
        27. Nc2! Bxc3 28. bxc3! Qxa2 29. cxb4!`)
}

func TestKasparovsImmortal(t *testing.T) {
    testGame(t, `1. e4 d6 2. d4 Nf6 3. Nc3 g6 4. Be3 Bg7 5. Qd2 c6 6. f3 b5
        7. Nge2 Nbd7 8. Bh6 Bxh6 9. Qxh6 Bb7 10. a3 e5 11. 0-0-0 Qe7
        12. Kb1 a6 13. Nc1 0-0-0 14. Nb3 exd4 15. Rxd4 c5 16. Rd1 Nb6
        17. g3 Kb8 18. Na5 Ba8 19. Bh3 d5 20. Qf4+ Ka7 21. Rhe1 d4
        22. Nd5 Nbxd5 23. exd5 Qd6 24. Rxd4 cxd4 25. Re7+ Kb6
        26. Qxd4+ Kxa5 27. b4+ Ka4 28. Qc3 Qxd5 29. Ra7 Bb7 30. Rxb7
        Qc4 31. Qxf6 Kxa3 32. Qxa6+ Kxb4 33. c3+ Kxc3 34. Qa1+ Kd2
        35. Qb2+ Kd1 36. Bf1 Rd2 37. Rd7 Rxd7 38. Bxc4 bxc4 39. Qxh8
        Rd3 40. Qa8 c3 41. Qa4+ Ke1 42. f4 f5 43. Kc1 Rd2 44. Qa7`)
}
