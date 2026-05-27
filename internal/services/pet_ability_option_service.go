package services

import "strings"

var PetAbilityOptionService = newPetAbilityOptionService()

func newPetAbilityOptionService() *petAbilityOptionService {
	return &petAbilityOptionService{}
}

type petAbilityOptionService struct{}

type PetAbilityOptionSourcePet struct {
	PetKey string `json:"petKey"`
	Name   string `json:"name"`
	Rarity string `json:"rarity"`
}

type PetAbilityOption struct {
	OptionKey       string                    `json:"optionKey"`
	Name            string                    `json:"name"`
	Description     string                    `json:"description"`
	SourcePet       PetAbilityOptionSourcePet `json:"sourcePet"`
	FeatureKeys     []string                  `json:"featureKeys"`
	EffectiveEvents []string                  `json:"effectiveEvents"`
	Abilities       map[string]any            `json:"abilities"`
	Selectable      bool                      `json:"selectable"`
	DisabledReason  string                    `json:"disabledReason"`
}

type PetAbilityOptionFilter struct {
	FeatureKey     string
	Rarity         string
	SelectableOnly bool
}

func (s *petAbilityOptionService) List(filter PetAbilityOptionFilter) []PetAbilityOption {
	options := s.defaultOptions()
	s.applyFeatureCatalog(options)
	return filterAbilityOptions(options, filter)
}

func (s *petAbilityOptionService) defaultOptions() []PetAbilityOption {
	return []PetAbilityOption{
		signinBonusOption("signin_bonus_15", "每日登录额外奖励15", "ninja", "忍者龟", "B", 15),
		signinBonusOption("signin_bonus_25", "每日登录额外奖励25", "stone", "石头龟 / 竹叶龟", "C", 25),
		signinBonusOption("signin_bonus_40", "每日登录额外奖励40", "angel", "天使龟 / 寒冰龟", "B", 40),
		signinBonusOption("signin_bonus_50", "每日登录额外奖励50", "headless", "无头龟", "SS", 50),
		signinBonusOption("signin_bonus_60", "每日登录额外奖励60", "rainbow", "彩虹龟", "A", 60),
		signinBonusOption("signin_bonus_100", "每日登录额外奖励100", "phoenix", "凤凰龟", "S", 100),
		{
			OptionKey:   "spark_multiplier_lava",
			Name:        "火花倍率1.3加等级成长",
			Description: "来自熔岩龟的火花倍率能力",
			SourcePet:   PetAbilityOptionSourcePet{PetKey: "lava", Name: "熔岩龟", Rarity: "S"},
			FeatureKeys: []string{"spark_multiplier"},
			Abilities: map[string]any{
				"spark_multiplier": map[string]any{
					"enabled":   true,
					"base":      1.3,
					"per_level": 0.03,
					"cap":       400,
				},
			},
		},
		{
			OptionKey:   "debt_300",
			Name:        "欠账上限300",
			Description: "来自闪电龟的欠账能力",
			SourcePet:   PetAbilityOptionSourcePet{PetKey: "lightning", Name: "闪电龟", Rarity: "A"},
			FeatureKeys: []string{"debt"},
			Abilities: map[string]any{
				"debt": map[string]any{
					"enabled":             true,
					"debtFloor":           -300,
					"forbidEquipWhenDebt": true,
					"errorCode":           "DEBT_UNPAID",
				},
			},
		},
		{
			OptionKey:   "debt_1000",
			Name:        "欠账上限1000",
			Description: "来自龟壳的欠账能力",
			SourcePet:   PetAbilityOptionSourcePet{PetKey: "shell", Name: "龟壳", Rarity: "SSS"},
			FeatureKeys: []string{"debt"},
			Abilities: map[string]any{
				"debt": map[string]any{
					"enabled":             true,
					"debtFloor":           -1000,
					"forbidEquipWhenDebt": true,
					"errorCode":           "DEBT_UNPAID",
				},
			},
		},
		{
			OptionKey:   "debt_subsidy_300",
			Name:        "欠账300加欠款补贴",
			Description: "来自闪电龟的欠账与欠款补贴组合能力",
			SourcePet:   PetAbilityOptionSourcePet{PetKey: "lightning", Name: "闪电龟", Rarity: "A"},
			FeatureKeys: []string{"debt", "debt_subsidy"},
			Abilities: map[string]any{
				"debt": map[string]any{
					"enabled":             true,
					"debtFloor":           -300,
					"forbidEquipWhenDebt": true,
					"errorCode":           "DEBT_UNPAID",
				},
				"debt_subsidy": map[string]any{
					"enabled":     true,
					"subsidyRate": 0.25,
				},
			},
		},
		{
			OptionKey:   "debt_subsidy_1000",
			Name:        "欠账1000加欠款补贴",
			Description: "来自龟壳的欠账与欠款补贴组合能力",
			SourcePet:   PetAbilityOptionSourcePet{PetKey: "shell", Name: "龟壳", Rarity: "SSS"},
			FeatureKeys: []string{"debt", "debt_subsidy"},
			Abilities: map[string]any{
				"debt": map[string]any{
					"enabled":             true,
					"debtFloor":           -1000,
					"forbidEquipWhenDebt": true,
					"errorCode":           "DEBT_UNPAID",
				},
				"debt_subsidy": map[string]any{
					"enabled":     true,
					"subsidyRate": 0.22,
				},
			},
		},
		{
			OptionKey:   "deposit_interest_space",
			Name:        "存款生息3%",
			Description: "来自星际龟的存款生息能力",
			SourcePet:   PetAbilityOptionSourcePet{PetKey: "space", Name: "星际龟", Rarity: "S"},
			FeatureKeys: []string{"deposit_interest"},
			Abilities: map[string]any{
				"deposit_interest": map[string]any{
					"enabled":      true,
					"interestRate": 0.03,
					"capPerDay":    1000,
				},
			},
		},
	}
}

func (s *petAbilityOptionService) applyFeatureCatalog(options []PetAbilityOption) {
	for i := range options {
		option := &options[i]
		option.Selectable = true
		option.DisabledReason = ""
		option.EffectiveEvents = option.EffectiveEvents[:0]
		seenEvents := map[string]bool{}
		for _, featureKey := range option.FeatureKeys {
			item := FeatureCatalogService.GetByFeatureKey(featureKey)
			if item == nil {
				option.Selectable = false
				option.DisabledReason = "feature not found: " + featureKey
				continue
			}
			if event := strings.TrimSpace(item.EffectiveEvent); event != "" && !seenEvents[event] {
				option.EffectiveEvents = append(option.EffectiveEvents, event)
				seenEvents[event] = true
			}
			if !item.Enabled {
				option.Selectable = false
				option.DisabledReason = "feature disabled: " + featureKey
			}
		}
	}
}

func signinBonusOption(optionKey, name, petKey, petName, rarity string, bonusCoins int64) PetAbilityOption {
	return PetAbilityOption{
		OptionKey:   optionKey,
		Name:        name,
		Description: "来自" + petName + "的每日登录额外奖励能力",
		SourcePet:   PetAbilityOptionSourcePet{PetKey: petKey, Name: petName, Rarity: rarity},
		FeatureKeys: []string{"signin_bonus"},
		Abilities: map[string]any{
			"signin_bonus": map[string]any{
				"enabled":    true,
				"bonusCoins": bonusCoins,
				"capPerDay":  500,
			},
		},
	}
}

func filterAbilityOptions(options []PetAbilityOption, filter PetAbilityOptionFilter) []PetAbilityOption {
	featureKey := strings.TrimSpace(filter.FeatureKey)
	rarity := strings.ToUpper(strings.TrimSpace(filter.Rarity))
	ret := make([]PetAbilityOption, 0, len(options))
	for _, option := range options {
		if filter.SelectableOnly && !option.Selectable {
			continue
		}
		if featureKey != "" && !containsString(option.FeatureKeys, featureKey) {
			continue
		}
		if rarity != "" && strings.ToUpper(option.SourcePet.Rarity) != rarity {
			continue
		}
		ret = append(ret, option)
	}
	return ret
}

func containsString(values []string, target string) bool {
	for _, v := range values {
		if strings.EqualFold(strings.TrimSpace(v), target) {
			return true
		}
	}
	return false
}
