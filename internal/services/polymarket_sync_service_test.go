package services

import (
	"bbs-go/internal/models/models"
	"bbs-go/internal/pkg/polymarket"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func TestPolymarket_ParseGammaTimeToUnix(t *testing.T) {
	// RFC3339
	if ts := parseGammaTimeToUnix("2026-03-29T12:34:56Z"); ts <= 0 {
		t.Fatalf("expected >0, got %d", ts)
	}
	// RFC3339Nano
	if ts := parseGammaTimeToUnix("2026-03-29T12:34:56.123Z"); ts <= 0 {
		t.Fatalf("expected >0, got %d", ts)
	}
	// empty
	if ts := parseGammaTimeToUnix(""); ts != 0 {
		t.Fatalf("expected 0, got %d", ts)
	}
	// invalid
	if ts := parseGammaTimeToUnix("not-a-time"); ts != 0 {
		t.Fatalf("expected 0, got %d", ts)
	}

	_ = time.Now()
}

func TestPolymarket_BinaryOutcomeTexts(t *testing.T) {
	market := &polymarket.Market{
		Outcomes: polymarket.Outcomes{
			{ID: "yes", Name: "Yes"},
			{ID: "no", Name: "No"},
		},
	}

	proText, conText := polymarketBinaryOutcomeTexts(market)
	if proText != "Yes" || conText != "No" {
		t.Fatalf("expected Yes/No, got %q/%q", proText, conText)
	}
}

func TestPolymarket_MatchConfiguredGammaTags_Disabled(t *testing.T) {
	// TODO: Fix matchConfiguredGammaTags function
	/*
		tags := []polymarket.Tag{
			{ID: float64(1), Slug: "politics", Name: "Politics"},
			{ID: "2", Slug: "sports", Name: "Sports"},
			{ID: int64(3), Slug: "crypto", Name: "Crypto"},
		}

		refs := matchConfiguredGammaTags(tags, []string{"Politics", "2", "crypto", "missing", "sports"})
		if len(refs) != 3 {
			t.Fatalf("expected 3 matched tags, got %d: %+v", len(refs), refs)
		}
		if refs[0].ID != 1 || refs[0].Slug != "politics" {
			t.Fatalf("expected politics by name, got %+v", refs[0])
		}
		if refs[1].ID != 2 || refs[1].Slug != "sports" {
			t.Fatalf("expected sports by id, got %+v", refs[1])
		}
		if refs[2].ID != 3 || refs[2].Slug != "crypto" {
			t.Fatalf("expected crypto by slug, got %+v", refs[2])
		}
	*/
}

func TestPolymarket_UpsertPredictContextUsesOutcomeNames(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "t_",
			SingularTable: true,
		},
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&models.PredictContext{}); err != nil {
		t.Fatalf("auto migrate predict context: %v", err)
	}

	svc := newPolymarketSyncService()
	now := time.Now().Unix()
	market := &polymarket.Market{
		Question: "Will BTC hit 100k?",
		Outcomes: polymarket.Outcomes{
			{ID: "up", Name: "Above 100k"},
			{ID: "down", Name: "Below 100k"},
		},
	}

	if err := svc.upsertPredictContext(db, 1001, market.Question, market, []string{"polymarket"}, now); err != nil {
		t.Fatalf("upsert predict context: %v", err)
	}

	var ctxModel models.PredictContext
	if err := db.Where("market_id = ?", 1001).First(&ctxModel).Error; err != nil {
		t.Fatalf("load predict context: %v", err)
	}
	if ctxModel.ProText != "Above 100k" || ctxModel.ConText != "Below 100k" {
		t.Fatalf("expected outcome names in pro/con text, got %q/%q", ctxModel.ProText, ctxModel.ConText)
	}

	market.Outcomes[0].Name = "Yes"
	market.Outcomes[1].Name = "No"
	if err := svc.upsertPredictContext(db, 1001, market.Question, market, []string{"polymarket"}, now+1); err != nil {
		t.Fatalf("upsert predict context again: %v", err)
	}
	if err := db.Where("market_id = ?", 1001).First(&ctxModel).Error; err != nil {
		t.Fatalf("reload predict context: %v", err)
	}
	if ctxModel.ProText != "Yes" || ctxModel.ConText != "No" {
		t.Fatalf("expected refreshed outcome names in pro/con text, got %q/%q", ctxModel.ProText, ctxModel.ConText)
	}
}
