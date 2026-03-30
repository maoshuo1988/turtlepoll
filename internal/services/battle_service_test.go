package services

import (
	"bbs-go/internal/models"
	"testing"
)

func TestBattle_SystemUserIdsAreNegative(t *testing.T) {
	if models.BattlePoolUserId >= 0 {
		t.Fatalf("BattlePoolUserId should be negative, got %d", models.BattlePoolUserId)
	}
	if models.BattleBurnUserId >= 0 {
		t.Fatalf("BattleBurnUserId should be negative, got %d", models.BattleBurnUserId)
	}
	if models.BattlePoolUserId == models.BattleBurnUserId {
		t.Fatalf("pool and burn userId should be different")
	}
}
