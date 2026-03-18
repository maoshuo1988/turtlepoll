package services

import "testing"

func TestCalcClampedOdds_BaseOnly(t *testing.T) {
	oddsA, oddsB, effA, effB, total := CalcClampedOdds(500, 500, 0, 0)
	if effA != 500 || effB != 500 || total != 1000 {
		t.Fatalf("unexpected eff pools: effA=%d effB=%d total=%d", effA, effB, total)
	}
	if oddsA != 2.0 || oddsB != 2.0 {
		t.Fatalf("unexpected odds: oddsA=%v oddsB=%v", oddsA, oddsB)
	}
}

func TestCalcClampedOdds_WithBets(t *testing.T) {
	oddsA, oddsB, _, _, _ := CalcClampedOdds(500, 500, 100, 0)
	// 例子里 A 100：oddsA≈1.83 oddsB≈2.2
	if oddsA < 1.7 || oddsA > 1.9 {
		t.Fatalf("unexpected oddsA: %v", oddsA)
	}
	if oddsB < 2.1 || oddsB > 2.3 {
		t.Fatalf("unexpected oddsB: %v", oddsB)
	}
}

func TestCalcClampedOdds_Clamp(t *testing.T) {
	// 让某一侧极热，raw oddsA 接近 1.0，但应被 clamp 到 1.2
	oddsA, _, _, _, _ := CalcClampedOdds(500, 500, 1000000, 0)
	if oddsA != 1.2 {
		t.Fatalf("expected oddsA clamped to 1.2, got %v", oddsA)
	}
	// 让某一侧极冷：A 极热、B 极冷，raw oddsB 很大，但应被 clamp 到 5.0
	_, oddsB, _, _, _ := CalcClampedOdds(500, 500, 1000000, 0)
	if oddsB != 5.0 {
		t.Fatalf("expected oddsB clamped to 5.0, got %v", oddsB)
	}
}
