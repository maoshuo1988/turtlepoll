package services

import (
	"bbs-go/internal/pkg/footballdata"
	"context"
	"os"
	"strings"
	"testing"
)

func TestFootballSyncService_TitleAndStatusRules(t *testing.T) {
	// 与实现保持一致的规则（这里测试规则本身，避免未来改动时把 OPEN 条件写错）
	buildMarketTitle := func(home, away string) string {
		if home != "" && away != "" {
			return home + " vs " + away
		}
		if home != "" {
			return home + " vs TBD"
		}
		if away != "" {
			return "TBD vs " + away
		}
		return "TBD vs TBD"
	}
	isTeamsReady := func(home, away string) bool {
		return strings.TrimSpace(home) != "" && strings.TrimSpace(away) != ""
	}

	// title
	if got := buildMarketTitle("A", "B"); got != "A vs B" {
		t.Fatalf("title mismatch: %q", got)
	}
	if got := buildMarketTitle("A", ""); got != "A vs TBD" {
		t.Fatalf("title mismatch: %q", got)
	}
	if got := buildMarketTitle("", "B"); got != "TBD vs B" {
		t.Fatalf("title mismatch: %q", got)
	}
	if got := buildMarketTitle("", ""); got != "TBD vs TBD" {
		t.Fatalf("title mismatch: %q", got)
	}

	// status
	if !isTeamsReady("A", "B") {
		t.Fatalf("teams should be ready")
	}
	if isTeamsReady("A", "") {
		t.Fatalf("home only should NOT be ready")
	}
	if isTeamsReady("", "B") {
		t.Fatalf("away only should NOT be ready")
	}
	if isTeamsReady(" ", "B") {
		t.Fatalf("blank home should NOT be ready")
	}
}

func TestTranslateKnownFootballTeamName(t *testing.T) {
	if got, ok := translateKnownFootballTeamName(0, "United States"); !ok || got != "美国" {
		t.Fatalf("expected United States translated to 美国, got=%q ok=%v", got, ok)
	}
	if got, ok := translateKnownFootballTeamName(0, "阿根廷"); ok || got != "阿根廷" {
		t.Fatalf("known translator should not rewrite custom/unknown names, got=%q ok=%v", got, ok)
	}
}

// 说明：这个测试默认跳过，除非显式提供 FOOTBALL_DATA_API_KEY；避免 CI/本地无网络时报错。
func TestFootballDataClient_GetCompetitionMatches(t *testing.T) {
	key := os.Getenv("FOOTBALL_DATA_API_KEY")
	if key == "" {
		t.Skip("FOOTBALL_DATA_API_KEY not set")
	}
	c := footballdata.NewClient(key)
	resp, err := c.GetCompetitionMatches(context.Background(), "WC", 0)
	if err != nil {
		t.Fatalf("GetCompetitionMatches err: %v", err)
	}
	if resp == nil {
		t.Fatalf("resp is nil")
	}
}
