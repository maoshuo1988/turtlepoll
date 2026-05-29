package services

import "testing"

func TestDefaultPetDefinitionSeeds(t *testing.T) {
	seeds := DefaultPetDefinitionSeeds()
	if len(seeds) < 5 {
		t.Fatalf("expected at least 5 default pet seeds, got %d", len(seeds))
	}

	found := map[string]bool{}
	foundPetKeys := map[string]bool{}
	for _, seed := range seeds {
		if seed.PetId == "" {
			t.Fatalf("pet_id should not be empty")
		}
		if seed.PetKey == "" {
			t.Fatalf("pet_key should not be empty for %s", seed.PetId)
		}
		if seed.Name == "" {
			t.Fatalf("name should not be empty for %s", seed.PetId)
		}
		foundPetKeys[seed.PetKey] = true
		abilities := PetDefinitionService.GetAbilities(&seed)
		for key := range abilities {
			found[key] = true
		}
	}

	for _, petKey := range []string{"basic", "stone", "bamboo", "angel", "ice", "fortune", "rainbow", "lightning", "phoenix", "lava", "shell"} {
		if !foundPetKeys[petKey] {
			t.Fatalf("expected pet %s to exist in default pet seeds", petKey)
		}
	}

	for _, key := range []string{"signup_bonus", "signin_bonus", "debt", "debt_subsidy", "deposit_interest"} {
		if !found[key] {
			t.Fatalf("expected feature %s to exist in default pet seeds", key)
		}
	}
}

func TestDefaultPetDefinitionSeedsCount(t *testing.T) {
	seeds := DefaultPetDefinitionSeeds()
	if len(seeds) != 11 {
		t.Fatalf("expected 11 default pets, got %d", len(seeds))
	}
}
