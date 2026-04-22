package services

import (
	"bbs-go/internal/models/models"
	"bbs-go/internal/pkg/biztime"
	"bbs-go/internal/repositories"
	"errors"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

// UserPetService 用户侧宠物读写（装备/资产）。
var UserPetService = newUserPetService()

func newUserPetService() *userPetService {
	return &userPetService{}
}

type userPetService struct{}

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
