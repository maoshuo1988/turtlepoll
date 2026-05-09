package services

import "testing"

func TestComputeAIStaminaNaturalRecoverNoFullPeriod(t *testing.T) {
	recovered, last := computeAIStaminaNaturalRecover(2, 5, 1000, 1000+59*60, 60)
	if recovered != 0 {
		t.Fatalf("recovered = %d, want 0", recovered)
	}
	if last != 1000 {
		t.Fatalf("lastRecoverAt = %d, want 1000", last)
	}
}

func TestComputeAIStaminaNaturalRecoverMultiplePeriods(t *testing.T) {
	recovered, last := computeAIStaminaNaturalRecover(1, 5, 1000, 1000+2*60*60+30, 60)
	if recovered != 2 {
		t.Fatalf("recovered = %d, want 2", recovered)
	}
	wantLast := int64(1000 + 2*60*60)
	if last != wantLast {
		t.Fatalf("lastRecoverAt = %d, want %d", last, wantLast)
	}
}

func TestComputeAIStaminaNaturalRecoverCapsAtMax(t *testing.T) {
	now := int64(1000 + 10*60*60)
	recovered, last := computeAIStaminaNaturalRecover(4, 5, 1000, now, 60)
	if recovered != 1 {
		t.Fatalf("recovered = %d, want 1", recovered)
	}
	if last != now {
		t.Fatalf("lastRecoverAt = %d, want %d", last, now)
	}
}
