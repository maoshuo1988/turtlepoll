package services

import "testing"

func TestDefaultAbilityOptionsIncludePhoenixSigninBonus(t *testing.T) {
	options := newPetAbilityOptionService().defaultOptions()
	var found *PetAbilityOption
	for i := range options {
		if options[i].OptionKey == "signin_bonus_100" {
			found = &options[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected signin_bonus_100 option")
	}
	if found.SourcePet.PetKey != "phoenix" {
		t.Fatalf("expected phoenix source pet, got %s", found.SourcePet.PetKey)
	}
	raw, ok := found.Abilities["signin_bonus"].(map[string]any)
	if !ok {
		t.Fatalf("expected signin_bonus ability map")
	}
	if got := toInt64ForTest(raw["bonusCoins"]); got != 100 {
		t.Fatalf("expected bonusCoins 100, got %#v", got)
	}
	if got := toInt64ForTest(raw["capPerDay"]); got != 500 {
		t.Fatalf("expected capPerDay 500, got %#v", got)
	}
}

func TestFilterAbilityOptions(t *testing.T) {
	options := []PetAbilityOption{
		{
			OptionKey:   "signin_bonus_100",
			SourcePet:   PetAbilityOptionSourcePet{Rarity: "S"},
			FeatureKeys: []string{"signin_bonus"},
			Selectable:  true,
		},
		{
			OptionKey:   "debt_300",
			SourcePet:   PetAbilityOptionSourcePet{Rarity: "A"},
			FeatureKeys: []string{"debt"},
			Selectable:  false,
		},
	}
	ret := filterAbilityOptions(options, PetAbilityOptionFilter{
		FeatureKey:     "signin_bonus",
		Rarity:         "s",
		SelectableOnly: true,
	})
	if len(ret) != 1 || ret[0].OptionKey != "signin_bonus_100" {
		t.Fatalf("unexpected filtered options: %#v", ret)
	}

	ret = filterAbilityOptions(options, PetAbilityOptionFilter{
		SelectableOnly: false,
	})
	if len(ret) != 2 {
		t.Fatalf("expected disabled option when selectableOnly=false, got %d", len(ret))
	}
}

func toInt64ForTest(v any) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int64:
		return n
	default:
		return 0
	}
}
