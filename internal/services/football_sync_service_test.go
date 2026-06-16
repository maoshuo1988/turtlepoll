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
	if got, ok := translateKnownFootballTeamName(0, "Bosnia-Herzegovina"); !ok || got != "波黑" {
		t.Fatalf("expected Bosnia-Herzegovina translated to 波黑, got=%q ok=%v", got, ok)
	}
	if got, ok := translateKnownFootballTeamName(0, "Jordan"); !ok || got != "约旦" {
		t.Fatalf("expected Jordan translated to 约旦, got=%q ok=%v", got, ok)
	}
	if got, ok := translateKnownFootballTeamName(0, "Algeria"); !ok || got != "阿尔及利亚" {
		t.Fatalf("expected Algeria translated to 阿尔及利亚, got=%q ok=%v", got, ok)
	}
	if got, ok := translateKnownFootballTeamName(0, "Ivory Coast"); !ok || got != "科特迪瓦" {
		t.Fatalf("expected Ivory Coast translated to 科特迪瓦, got=%q ok=%v", got, ok)
	}
	if got, ok := translateKnownFootballTeamName(0, "Cape Verde Islands"); !ok || got != "佛得角" {
		t.Fatalf("expected Cape Verde Islands translated to 佛得角, got=%q ok=%v", got, ok)
	}
	if got, ok := translateKnownFootballTeamName(0, "Congo DR"); !ok || got != "刚果（金）" {
		t.Fatalf("expected Congo DR translated to 刚果（金）, got=%q ok=%v", got, ok)
	}
	if got, ok := translateKnownFootballTeamName(0, "Uzbekistan"); !ok || got != "乌兹别克斯坦" {
		t.Fatalf("expected Uzbekistan translated to 乌兹别克斯坦, got=%q ok=%v", got, ok)
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
