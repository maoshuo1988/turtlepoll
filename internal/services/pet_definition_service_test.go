package services

import "testing"

func TestDefaultPetDefinitionSeeds(t *testing.T) {
	seeds := DefaultPetDefinitionSeeds()
	if len(seeds) < 5 {
		t.Fatalf("expected at least 5 default pet seeds, got %d", len(seeds))
	}

	found := map[string]bool{}
	for _, seed := range seeds {
		if seed.PetId == "" {
			t.Fatalf("pet_id should not be empty")
		}
		if seed.Name == "" {
			t.Fatalf("name should not be empty for %s", seed.PetId)
		}
		abilities := PetDefinitionService.GetAbilities(&seed)
		for key := range abilities {
			found[key] = true
		}
	}

	for _, key := range []string{"signin_bonus", "spark_multiplier", "debt", "debt_subsidy", "deposit_interest"} {
		if !found[key] {
			t.Fatalf("expected feature %s to exist in default pet seeds", key)
		}
	}
}

func TestDefaultPetDefinitionSeedsCount(t *testing.T) {
	seeds := DefaultPetDefinitionSeeds()
	if len(seeds) != 7 {
		t.Fatalf("expected 7 default pets, got %d", len(seeds))
	}
}
