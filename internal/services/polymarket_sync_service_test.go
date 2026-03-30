package services

import (
	"testing"
	"time"
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
