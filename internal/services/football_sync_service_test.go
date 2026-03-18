package services

import (
	"bbs-go/internal/pkg/footballdata"
	"context"
	"os"
	"testing"
)

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
