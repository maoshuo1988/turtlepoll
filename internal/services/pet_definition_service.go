package services

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/models"
	"bbs-go/internal/repositories"
	"errors"
	"log/slog"
	"strings"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/common/jsons"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

var PetDefinitionService = newPetDefinitionService()

func newPetDefinitionService() *petDefinitionService {
	return &petDefinitionService{}
}

type petDefinitionService struct{}

// EnsureDefaultSeeds 会在服务启动时调用，确保已实现的 P0 特性都有可直接使用的默认宠物定义。
func (s *petDefinitionService) EnsureDefaultSeeds() {
	for _, seed := range DefaultPetDefinitionSeeds() {
		seed := seed
		if strings.TrimSpace(seed.PetId) == "" {
			continue
		}
		exists := s.GetByPetId(seed.PetId)
		if exists == nil && strings.TrimSpace(seed.PetKey) != "" {
			exists = s.GetByPetKey(seed.PetKey)
		}
		if exists == nil {
			if err := s.Create(&seed); err != nil {
				slog.Error("seed pet_definition create failed", slog.Any("err", err), slog.String("petId", seed.PetId))
			}
			continue
		}

		updated := false
		if strings.TrimSpace(seed.PetId) != "" && exists.PetId != seed.PetId && s.GetByPetId(seed.PetId) == nil {
			exists.PetId = seed.PetId
			updated = true
		}
		if strings.TrimSpace(exists.PetKey) == "" {
			exists.PetKey = seed.PetKey
			updated = true
		}
		if strings.TrimSpace(seed.PetKey) != "" && exists.PetKey != seed.PetKey {
			exists.PetKey = seed.PetKey
			updated = true
		}
		if strings.TrimSpace(exists.NameJSON) == "" {
			exists.NameJSON = seed.NameJSON
			updated = true
		}
		if strings.TrimSpace(exists.Name) == "" {
			exists.Name = seed.Name
			updated = true
		}
		if strings.TrimSpace(exists.DescriptionJSON) == "" {
			exists.DescriptionJSON = seed.DescriptionJSON
			updated = true
		}
		if strings.TrimSpace(exists.Description) == "" {
			exists.Description = seed.Description
			updated = true
		}
		if exists.Rarity <= 0 {
			exists.Rarity = seed.Rarity
			updated = true
		}
		if strings.TrimSpace(exists.AbilitiesJSON) == "" {
			exists.AbilitiesJSON = seed.AbilitiesJSON
			updated = true
		}
		if strings.TrimSpace(exists.DisplayJSON) == "" {
			exists.DisplayJSON = seed.DisplayJSON
			updated = true
		}
		if exists.Status != constants.StatusOk {
			exists.Status = constants.StatusOk
			updated = true
		}
		if updated {
			if err := s.Update(exists); err != nil {
				slog.Error("seed pet_definition update failed", slog.Any("err", err), slog.String("petId", seed.PetId))
			}
		}
	}
}

func (s *petDefinitionService) GrantDefaultPetsToAdmins() {
	admins := UserService.Find(sqls.NewCnd().
		Eq("status", constants.StatusOk).
		Where("(roles LIKE ? OR roles LIKE ?)", "%owner%", "%admin%"))
	if len(admins) == 0 {
		return
	}

	seeds := DefaultPetDefinitionSeeds()
	now := dates.NowTimestamp()
	for _, admin := range admins {
		admin := admin
		for _, seed := range seeds {
			seed := seed
			pet := s.GetByPetId(seed.PetId)
			if pet == nil {
				continue
			}
			if UserPetService.HasPet(admin.Id, pet.Id) {
				continue
			}
			if err := repositories.UserPetRepository.Create(sqls.DB(), &models.UserPet{
				UserId:     admin.Id,
				PetId:      pet.Id,
				Level:      1,
				XP:         0,
				ObtainedAt: now,
				CreateTime: now,
				UpdateTime: now,
			}); err != nil {
				slog.Error("grant default pet to admin failed", slog.Any("err", err), slog.Int64("userId", admin.Id), slog.String("petId", seed.PetId))
			}
		}
	}
}

func DefaultPetDefinitionSeeds() []models.PetDefinition {
	seeds := []struct {
		petID           string
		petKey          string
		nameZh          string
		nameEn          string
		display         map[string]any
		rarity          int
		obtainableByEgg bool
		abilities       map[string]any
	}{
		{
			petID:   "15",
			petKey:  "lightning",
			nameZh:  "闪电龟",
			nameEn:  "LightningTurtle",
			display: petThumbnailDisplay("LightningTurtle.png"),
			rarity:  3,
			abilities: map[string]any{
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
			petID:   "10",
			petKey:  "bamboo",
			nameZh:  "竹叶龟",
			nameEn:  "BambooTurtle",
			display: petThumbnailDisplay("BambooTurtle.png"),
			rarity:  1,
			abilities: map[string]any{
				"signin_bonus": map[string]any{
					"enabled":    true,
					"bonusCoins": 25,
					"capPerDay":  500,
				},
			},
		},
		{
			petID:   "12",
			petKey:  "ice",
			nameZh:  "寒冰龟",
			nameEn:  "IceTurtle",
			display: petThumbnailDisplay("IceTurtle.png"),
			rarity:  2,
			abilities: map[string]any{
				"signin_bonus": map[string]any{
					"enabled":    true,
					"bonusCoins": 40,
					"capPerDay":  500,
				},
			},
		},
		{
			petID:   "13",
			petKey:  "fortune",
			nameZh:  "财神龟",
			nameEn:  "GodofWealthTurtle",
			display: petThumbnailDisplay("GodofWealthTurtle.png"),
			rarity:  2,
			abilities: map[string]any{
				"signin_bonus": map[string]any{
					"enabled":    true,
					"bonusCoins": 40,
					"capPerDay":  500,
				},
			},
		},
		{
			petID:   "17",
			petKey:  "lava",
			nameZh:  "熔岩龟",
			nameEn:  "Lavaturtle",
			display: petThumbnailDisplay("Lavaturtle.png"),
			rarity:  4,
			abilities: map[string]any{
				"deposit_interest": map[string]any{
					"enabled":      true,
					"interestRate": 0.03,
					"capPerDay":    1000,
				},
				"signin_bonus": map[string]any{
					"enabled":    true,
					"bonusCoins": 100,
					"capPerDay":  500,
				},
			},
		},
		{
			petID:   "18",
			petKey:  "shell",
			nameZh:  "龟壳",
			nameEn:  "turtleShell",
			display: petThumbnailDisplay("turtleShell.png"),
			rarity:  6,
			abilities: map[string]any{
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
				"deposit_interest": map[string]any{
					"enabled":      true,
					"interestRate": 0.05,
					"capPerDay":    1000,
				},
			},
		},
		{
			petID:   "9",
			petKey:  "stone",
			nameZh:  "石头龟",
			nameEn:  "StoneTurtle",
			display: petThumbnailDisplay("StoneTurtle.png"),
			rarity:  1,
			abilities: map[string]any{
				"signin_bonus": map[string]any{
					"enabled":    true,
					"bonusCoins": 25,
					"capPerDay":  500,
				},
			},
		},
		{
			petID:   "11",
			petKey:  "angel",
			nameZh:  "天使龟",
			nameEn:  "AngelTurtle",
			display: petThumbnailDisplay("AngelTurtle.png"),
			rarity:  2,
			abilities: map[string]any{
				"signin_bonus": map[string]any{
					"enabled":    true,
					"bonusCoins": 40,
					"capPerDay":  500,
				},
			},
		},
		{
			petID:           "8",
			petKey:          "basic",
			nameZh:          "小龟",
			nameEn:          "turtlet",
			display:         basicPetDisplay(),
			rarity:          1,
			obtainableByEgg: true,
			abilities: map[string]any{
				"signup_bonus": map[string]any{
					"enabled": true,
					"coins":   500,
				},
			},
		},
		{
			petID:   "14",
			petKey:  "rainbow",
			nameZh:  "彩虹龟",
			nameEn:  "RainbowTurtle",
			display: petThumbnailDisplay("RainbowTurtle.png"),
			rarity:  3,
			abilities: map[string]any{
				"signin_bonus": map[string]any{
					"enabled":    true,
					"bonusCoins": 60,
					"capPerDay":  500,
				},
			},
		},
		{
			petID:   "16",
			petKey:  "phoenix",
			nameZh:  "凤凰龟",
			nameEn:  "PhoenixTurtle",
			display: petThumbnailDisplay("PhoenixTurtle.png"),
			rarity:  4,
			abilities: map[string]any{
				"signin_bonus": map[string]any{
					"enabled":    true,
					"bonusCoins": 100,
					"capPerDay":  500,
				},
			},
		},
	}

	ret := make([]models.PetDefinition, 0, len(seeds))
	for _, seed := range seeds {
		ret = append(ret, models.PetDefinition{
			PetId:           seed.petID,
			PetKey:          seed.petKey,
			NameJSON:        jsons.ToJsonStr(map[string]string{"zh-CN": seed.nameZh, "en-US": seed.nameEn}),
			DescriptionJSON: jsons.ToJsonStr(map[string]string{}),
			DisplayJSON:     jsons.ToJsonStr(seed.display),
			Name:            seed.nameZh,
			Description:     "",
			Rarity:          seed.rarity,
			ObtainableByEgg: seed.obtainableByEgg,
			AbilitiesJSON:   jsons.ToJsonStr(seed.abilities),
			Status:          constants.StatusOk,
		})
	}
	return ret
}

func petThumbnailDisplay(filename string) map[string]any {
	return map[string]any{
		"thumbnail": "https://turtlepicture20260525-bucket.oss-cn-beijing.aliyuncs.com/turtleImg/" + filename,
	}
}

func basicPetDisplay() map[string]any {
	return map[string]any{
		"icon":      "https://turtlepicture20260525-bucket.oss-cn-beijing.aliyuncs.com/spine/wugui/wugui.json",
		"thumbnail": "https://turtlepicture20260525-bucket.oss-cn-beijing.aliyuncs.com/turtleImg/turtlet.png",
	}
}

func (s *petDefinitionService) Get(id int64) *models.PetDefinition {
	return repositories.PetDefinitionRepository.Get(sqls.DB(), id)
}

func (s *petDefinitionService) GetByPetKey(petKey string) *models.PetDefinition {
	petKey = strings.TrimSpace(petKey)
	if petKey == "" {
		return nil
	}
	return repositories.PetDefinitionRepository.GetByPetKey(sqls.DB(), petKey)
}

func (s *petDefinitionService) GetByPetId(petId string) *models.PetDefinition {
	petId = strings.TrimSpace(petId)
	if petId == "" {
		return nil
	}
	return repositories.PetDefinitionRepository.GetByPetId(sqls.DB(), petId)
}

func (s *petDefinitionService) FindPageByCnd(cnd *sqls.Cnd) (list []models.PetDefinition, paging *sqls.Paging) {
	return repositories.PetDefinitionRepository.FindPageByCnd(sqls.DB(), cnd)
}

func (s *petDefinitionService) Create(t *models.PetDefinition) error {
	now := dates.NowTimestamp()
	if t.CreateTime <= 0 {
		t.CreateTime = now
	}
	t.UpdateTime = now
	if t.Status == 0 {
		t.Status = constants.StatusOk
	}
	return repositories.PetDefinitionRepository.Create(sqls.DB(), t)
}

func (s *petDefinitionService) Update(t *models.PetDefinition) error {
	t.UpdateTime = dates.NowTimestamp()
	return repositories.PetDefinitionRepository.Update(sqls.DB(), t)
}

func (s *petDefinitionService) Delete(id int64) error {
	return repositories.PetDefinitionRepository.Updates(sqls.DB(), id, map[string]interface{}{
		"status":      constants.StatusDeleted,
		"update_time": dates.NowTimestamp(),
	})
}

// abilities helpers

func (s *petDefinitionService) GetAbilities(t *models.PetDefinition) map[string]any {
	if t == nil || len(t.AbilitiesJSON) == 0 {
		return map[string]any{}
	}
	var m map[string]any
	_ = jsons.Parse(t.AbilitiesJSON, &m)
	if m == nil {
		return map[string]any{}
	}
	return m
}

func (s *petDefinitionService) SetAbilities(tx *gorm.DB, petId int64, abilities map[string]any) (*models.PetDefinition, error) {
	pet := repositories.PetDefinitionRepository.Get(tx, petId)
	if pet == nil {
		return nil, errors.New("pet definition not found")
	}

	pet.AbilitiesJSON = jsons.ToJsonStr(abilities)
	pet.UpdateTime = dates.NowTimestamp()
	if err := repositories.PetDefinitionRepository.Update(tx, pet); err != nil {
		return nil, err
	}
	return pet, nil
}
