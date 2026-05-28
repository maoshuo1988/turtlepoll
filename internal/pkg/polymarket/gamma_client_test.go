package polymarket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGammaClient_ListMarketsKeyset(t *testing.T) {
	var seenPath, seenLimit, seenCursor, seenTag string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		seenLimit = r.URL.Query().Get("limit")
		seenCursor = r.URL.Query().Get("after_cursor")
		seenTag = r.URL.Query().Get("tag_id")

		_ = json.NewEncoder(w).Encode(MarketsKeysetPage{
			Markets: []Market{
				{ID: float64(123), Slug: "sample-market", Question: "Will this test pass?"},
			},
			NextCursor: "cursor-2",
		})
	}))
	defer server.Close()

	client := NewGammaClient(server.URL)
	markets, nextCursor, err := client.ListMarketsKeyset(context.Background(), 5, "cursor-1", map[string]string{"tag_id": "99"})
	if err != nil {
		t.Fatalf("list markets keyset: %v", err)
	}
	if seenPath != "/markets/keyset" {
		t.Fatalf("expected /markets/keyset, got %q", seenPath)
	}
	if seenLimit != "5" || seenCursor != "cursor-1" || seenTag != "99" {
		t.Fatalf("unexpected query: limit=%q cursor=%q tag=%q", seenLimit, seenCursor, seenTag)
	}
	if nextCursor != "cursor-2" {
		t.Fatalf("expected next cursor cursor-2, got %q", nextCursor)
	}
	if len(markets) != 1 || markets[0].Slug != "sample-market" {
		t.Fatalf("unexpected markets: %+v", markets)
	}
}
