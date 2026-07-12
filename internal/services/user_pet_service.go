package services

import (
	"bbs-go/internal/models/models"
	"bbs-go/internal/pkg/biztime"
	"bbs-go/internal/repositories"
	"errors"
	"fmt"
	"strings"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/common/jsons"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

// UserPetService 用户侧宠物读写（装备/资产）。
var UserPetService = newUserPetService()

func newUserPetService() *userPetService {
	return &userPetService{}
}

type userPetService struct{}

type signupBonusParams struct {
	Enabled bool  `json:"enabled"`
	Coins   int64 `json:"coins"`
}

type SignupBonusResult struct {
	Granted bool   `json:"granted"`
	Amount  int64  `json:"amount"`
	PetKey  string `json:"petKey,omitempty"`
	PetName string `json:"petName,omitempty"`
}

func (s *userPetService) GetOrCreateState(userId int64) (*models.UserPetState, error) {
	if userId <= 0 {
		return nil, errors.New("userId is required")
	}
	now := dates.NowTimestamp()
	state := repositories.UserPetStateRepository.GetByUserId(sqls.DB(), userId)
	if state != nil {
		return state, nil
	}
	state = &models.UserPetState{
		UserId:        userId,
		EquippedPetId: 0,
		EquipDayName:  0,
		CreateTime:    now,
		UpdateTime:    now,
	}
	if err := repositories.UserPetStateRepository.Create(sqls.DB(), state); err != nil {
		return nil, err
	}
	return state, nil
}

func (s *userPetService) ListOwned(userId int64) ([]models.UserPet, error) {
	if userId <= 0 {
		return nil, errors.New("userId is required")
	}
	return repositories.UserPetRepository.FindByUserId(sqls.DB(), userId), nil
}

func (s *userPetService) HasPet(userId int64, petId int64) bool {
	if userId <= 0 || petId <= 0 {
		return false
	}
	return repositories.UserPetRepository.Get(sqls.DB(), userId, petId) != nil
}

func (s *userPetService) GrantAndEquipBasicOnSignup(tx *gorm.DB, userId int64, now int64) (*SignupBonusResult, error) {
	if userId <= 0 {
		return nil, errors.New("userId is required")
	}
	if now <= 0 {
		now = dates.NowTimestamp()
	}

	pet := repositories.PetDefinitionRepository.GetByPetKey(tx, "basic")
	if pet == nil {
		seed := findDefaultPetDefinitionSeed("basic")
		if seed == nil {
			return nil, errors.New("basic pet definition seed not found")
		}
		seed.CreateTime = now
		seed.UpdateTime = now
		if err := repositories.PetDefinitionRepository.Create(tx, seed); err != nil {
			return nil, err
		}
		pet = seed
	}

	if repositories.UserPetRepository.Get(tx, userId, pet.Id) == nil {
		if err := repositories.UserPetRepository.Create(tx, &models.UserPet{
			UserId:     userId,
			PetId:      pet.Id,
			Level:      1,
			XP:         0,
			ObtainedAt: now,
			CreateTime: now,
			UpdateTime: now,
		}); err != nil {
			return nil, err
		}
	}

	state := repositories.UserPetStateRepository.GetByUserId(tx, userId)
	if state == nil {
		state = &models.UserPetState{
			UserId:        userId,
			EquippedPetId: pet.Id,
			EquipDayName:  0,
			CreateTime:    now,
			UpdateTime:    now,
		}
		if err := repositories.UserPetStateRepository.Create(tx, state); err != nil {
			return nil, err
		}
	} else if state.EquippedPetId <= 0 {
		state.EquippedPetId = pet.Id
		state.UpdateTime = now
		if err := repositories.UserPetStateRepository.Update(tx, state); err != nil {
			return nil, err
		}
	}

	return s.grantSignupBonusByPet(tx, userId, pet, now)
}

func findDefaultPetDefinitionSeed(petId string) *models.PetDefinition {
	for _, seed := range DefaultPetDefinitionSeeds() {
		if seed.PetId == petId || seed.PetKey == petId {
			seed := seed
			return &seed
		}
	}
	return nil
}

func (s *userPetService) grantSignupBonusByPet(tx *gorm.DB, userId int64, pet *models.PetDefinition, now int64) (*SignupBonusResult, error) {
	ret := &SignupBonusResult{}
	if pet == nil {
		return ret, nil
	}
	fc := FeatureCatalogService.GetByFeatureKey("signup_bonus")
	if fc == nil || !fc.Enabled {
		return ret, nil
	}
	abilities := PetDefinitionService.GetAbilities(pet)
	raw, ok := abilities["signup_bonus"]
	if !ok || raw == nil {
		return ret, nil
	}
	params, err := decodeSignupBonusParams(raw)
	if err != nil {
		return ret, err
	}
	if !params.Enabled || params.Coins <= 0 {
		return ret, nil
	}
	remark := fmt.Sprintf("pet signup bonus | petId=%d | petKey=%s", pet.Id, pet.PetKey)
	granted, err := s.mintIfNoSignupBonusLog(tx, userId, params.Coins, remark, now)
	if err != nil {
		return ret, err
	}
	if granted {
		ret.Granted = true
		ret.Amount = params.Coins
		ret.PetKey = pet.PetKey
		ret.PetName = pet.Name
	}
	return ret, nil
}

func decodeSignupBonusParams(v any) (*signupBonusParams, error) {
	b := jsons.ToJsonStr(v)
	if strings.TrimSpace(b) == "" {
		return nil, errors.New("empty signup_bonus params")
	}
	var p signupBonusParams
	if err := jsons.Parse(b, &p); err != nil {
		return nil, err
	}
	if p.Coins < 0 {
		return nil, errors.New("signup_bonus coins must be non-negative")
	}
	return &p, nil
}

func (s *userPetService) mintIfNoSignupBonusLog(tx *gorm.DB, userId int64, amount int64, remark string, now int64) (bool, error) {
	var count int64
	if err := tx.Model(&models.UserCoinLog{}).
		Where("user_id = ? AND biz_type = ? AND remark LIKE ?", userId, "MINT", "pet signup bonus%").
		Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}
	_, err := UserCoinService.mintWithTx(tx, 0, userId, amount, remark, now)
	return err == nil, err
}

func (s *userPetService) EquipPet(userId int64, petId int64) (*models.UserPetState, error) {
	if userId <= 0 {
		return nil, errors.New("userId is required")
	}
	if petId <= 0 {
		return nil, errors.New("petId is required")
	}
	today := biztime.DayNameCST(biztime.NowInCST())

	var ret *models.UserPetState
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		state := repositories.UserPetStateRepository.GetByUserId(tx, userId)
		now := dates.NowTimestamp()
		uc, err := repositories.UserCoinRepository.GetOrCreate(tx, userId)
		if err != nil {
			return err
		}
		if uc.Balance < 0 {
			return errors.New("DEBT_UNPAID")
		}
		if state == nil {
			state = &models.UserPetState{UserId: userId, CreateTime: now}
		}
		// P0：每天只允许切换一次
		if state.EquipDayName == today {
			return errors.New("EQUIP_DAILY_LIMIT")
		}
		state.EquippedPetId = petId
		state.EquipDayName = today
		state.UpdateTime = now
		if state.Id > 0 {
			if err := repositories.UserPetStateRepository.Update(tx, state); err != nil {
				return err
			}
		} else {
			if err := repositories.UserPetStateRepository.Create(tx, state); err != nil {
				return err
			}
		}
		ret = state
		return nil
	})
	return ret, err
}
