package services

import "testing"

func TestValidateRarityWeights_OK(t *testing.T) {
	w := map[string]float64{"C": 0.4, "B": 0.3, "A": 0.15, "S": 0.1, "SS": 0.04, "SSS": 0.01}
	if err := validateRarityWeights(w); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateRarityWeights_SumNotOne(t *testing.T) {
	w := map[string]float64{"C": 0.4, "B": 0.3, "A": 0.15, "S": 0.1, "SS": 0.04, "SSS": 0.02}
	if err := validateRarityWeights(w); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestValidateRarityWeights_InvalidKey(t *testing.T) {
	w := map[string]float64{"X": 1.0}
	if err := validateRarityWeights(w); err == nil {
		t.Fatalf("expected error, got nil")
	}
}
