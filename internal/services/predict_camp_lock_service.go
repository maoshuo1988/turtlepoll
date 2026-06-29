package services

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/models"
	"errors"
	"strings"

	"gorm.io/gorm"
)

var PredictCampLockService = newPredictCampLockService()

func newPredictCampLockService() *predictCampLockService {
	return &predictCampLockService{}
}

type predictCampLockService struct{}

const (
	predictCampRoundID      int64 = 0
	predictCampLockInteract       = "INTERACT"
	predictCampLockBet            = "BET"
)

func (s *predictCampLockService) GetOption(db *gorm.DB, marketId, userId int64) string {
	if db == nil || marketId <= 0 || userId <= 0 {
		return ""
	}
	member := &models.TearCampMember{}
	err := db.Where("event_type = ? AND topic_id = ? AND round_id = ? AND user_id = ?",
		constants.EntityPredictMarket, marketId, predictCampRoundID, userId).Take(member).Error
	if err != nil {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(member.Option))
}

func (s *predictCampLockService) EnsureBetLock(tx *gorm.DB, marketId, userId int64, option string, now int64) error {
	return s.ensure(tx, marketId, userId, option, predictCampLockBet, now)
}

func (s *predictCampLockService) EnsureInteractLock(tx *gorm.DB, marketId, userId int64, option string, now int64) error {
	return s.ensure(tx, marketId, userId, option, predictCampLockInteract, now)
}

func (s *predictCampLockService) ensure(tx *gorm.DB, marketId, userId int64, option, lockType string, now int64) error {
	if tx == nil {
		return errors.New("db is required")
	}
	if marketId <= 0 || userId <= 0 {
		return errors.New("marketId and userId are required")
	}
	option = strings.ToUpper(strings.TrimSpace(option))
	if option == "" {
		return errors.New("option is required")
	}

	member := &models.TearCampMember{}
	err := tx.Where("event_type = ? AND topic_id = ? AND round_id = ? AND user_id = ?",
		constants.EntityPredictMarket, marketId, predictCampRoundID, userId).Take(member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(&models.TearCampMember{
				EventType:       constants.EntityPredictMarket,
				EventId:         marketId,
				TopicId:         marketId,
				RoundId:         predictCampRoundID,
				UserId:          userId,
				Option:          option,
				LockType:        lockType,
				FirstActionTime: now,
				CreateTime:      now,
				UpdateTime:      now,
			}).Error
		}
		return err
	}

	memberOpt := strings.ToUpper(strings.TrimSpace(member.Option))
	if memberOpt != option {
		if strings.EqualFold(member.LockType, predictCampLockBet) {
			return errors.New("TEAR_CAMP_LOCKED_BY_BET")
		}
		return errors.New("TEAR_CAMP_CONFLICT")
	}

	if lockType == predictCampLockBet {
		member.LockType = predictCampLockBet
	}
	if member.FirstActionTime == 0 {
		member.FirstActionTime = now
	}
	member.UpdateTime = now
	return tx.Save(member).Error
}
