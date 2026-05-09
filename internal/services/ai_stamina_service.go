package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"bbs-go/internal/models/models"
	"bbs-go/internal/pkg/biztime"
	"bbs-go/internal/pkg/config"
	"bbs-go/internal/repositories"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	aiStaminaBizChat    = "CHAT"
	aiStaminaBizApple   = "APPLE"
	aiStaminaBizAdmin   = "ADMIN"
	aiStaminaBizRecover = "RECOVER"
)

var AIStaminaService = newAIStaminaService()

func newAIStaminaService() *aiStaminaService {
	return &aiStaminaService{}
}

type aiStaminaService struct{}

type AIStaminaStatus struct {
	UserId         int64 `json:"userId"`
	Stamina        int   `json:"stamina"`
	MaxStamina     int   `json:"maxStamina"`
	NextRecoverAt  int64 `json:"nextRecoverAt"`
	DailyUsedCount int   `json:"dailyUsedCount"`
	DailyLimit     int   `json:"dailyLimit"`
	RecoverMinutes int   `json:"recoverMinutes"`
	AppleCoinCost  int   `json:"appleCoinCost"`
	LastRecoverAt  int64 `json:"lastRecoverAt"`
	DailyRemaining int   `json:"dailyRemaining"`
}

type AIStaminaAppleResult struct {
	AIStaminaStatus
	RequestedCount int   `json:"requestedCount"`
	RecoveredCount int   `json:"recoveredCount"`
	CoinCost       int64 `json:"coinCost"`
	BalanceAfter   int64 `json:"balanceAfter"`
}

type AIStaminaGrantResult struct {
	AIStaminaStatus
	GrantedAmount int `json:"grantedAmount"`
}

func (s *aiStaminaService) GetStatus(userId int64) (*AIStaminaStatus, error) {
	if userId <= 0 {
		return nil, errors.New("userId is required")
	}
	var ret *AIStaminaStatus
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		st, err := s.getOrCreateForUpdate(tx, userId, dates.NowTimestamp())
		if err != nil {
			return err
		}
		if err := s.recoverByTime(tx, st, time.Now()); err != nil {
			return err
		}
		ret = s.buildStatus(st)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *aiStaminaService) HasEnough(userId int64, cost int) (*AIStaminaStatus, bool, error) {
	if userId <= 0 {
		return nil, false, errors.New("userId is required")
	}
	if cost <= 0 {
		cost = 1
	}
	var status *AIStaminaStatus
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		st, err := s.getOrCreateForUpdate(tx, userId, dates.NowTimestamp())
		if err != nil {
			return err
		}
		if err := s.recoverByTime(tx, st, time.Now()); err != nil {
			return err
		}
		status = s.buildStatus(st)
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return status, status.Stamina >= cost, nil
}

func (s *aiStaminaService) ConsumeForChatTx(tx *gorm.DB, userId int64, messageId int64, cost int) (*AIStaminaStatus, error) {
	if tx == nil {
		return nil, errors.New("tx is required")
	}
	if userId <= 0 {
		return nil, errors.New("userId is required")
	}
	if messageId <= 0 {
		return nil, errors.New("messageId is required")
	}
	if cost <= 0 {
		cost = 1
	}
	now := dates.NowTimestamp()
	st, err := s.getOrCreateForUpdate(tx, userId, now)
	if err != nil {
		return nil, err
	}
	if err := s.recoverByTime(tx, st, time.Now()); err != nil {
		return nil, err
	}
	if st.Stamina < cost {
		return nil, errors.New("AI_STAMINA_NOT_ENOUGH")
	}
	if st.Stamina >= st.MaxStamina {
		st.LastRecoverAt = now
	}
	st.Stamina -= cost
	st.DailyUsedDate = biztime.DayNameCST(time.Now())
	st.DailyUsedCount++
	st.UpdateTime = now
	if err := tx.Save(st).Error; err != nil {
		return nil, err
	}
	if err := s.createLog(tx, userId, aiStaminaBizChat, fmt.Sprintf("%d", messageId), -cost, st.Stamina, fmt.Sprintf("ai chat: messageId=%d", messageId), now); err != nil {
		return nil, err
	}
	return s.buildStatus(st), nil
}

func (s *aiStaminaService) RecoverByApple(userId int64, count int) (*AIStaminaAppleResult, error) {
	if userId <= 0 {
		return nil, errors.New("userId is required")
	}
	if count <= 0 {
		return nil, errors.New("count must be positive")
	}
	var ret *AIStaminaAppleResult
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		now := dates.NowTimestamp()
		st, err := s.getOrCreateForUpdate(tx, userId, now)
		if err != nil {
			return err
		}
		if err := s.recoverByTime(tx, st, time.Now()); err != nil {
			return err
		}
		missing := st.MaxStamina - st.Stamina
		if missing <= 0 {
			return errors.New("AI_STAMINA_FULL")
		}
		recoverCount := count
		if recoverCount > missing {
			recoverCount = missing
		}
		coinCost := int64(recoverCount * s.appleCoinCost())
		uc, err := repositories.UserCoinRepository.GetOrCreate(tx.Clauses(clause.Locking{Strength: "UPDATE"}), userId)
		if err != nil {
			return err
		}
		debtFloor := PetBalanceFeatureService.ResolveDebtFloorForTx(tx, userId)
		if !PetBalanceFeatureService.CanSpend(uc.Balance, coinCost, debtFloor) {
			return errors.New("insufficient balance")
		}
		uc.Balance -= coinCost
		uc.UpdateTime = now
		if uc.CreateTime == 0 {
			uc.CreateTime = now
		}
		if err := repositories.UserCoinRepository.Update(tx, uc); err != nil {
			return err
		}
		if err := repositories.UserCoinLogRepository.Create(tx, &models.UserCoinLog{
			UserId:       userId,
			BizType:      "AI_STAMINA_APPLE",
			BizId:        0,
			Amount:       -coinCost,
			BalanceAfter: uc.Balance,
			Remark:       fmt.Sprintf("ai stamina apple: count=%d", recoverCount),
			CreateTime:   now,
		}); err != nil {
			return err
		}

		st.Stamina += recoverCount
		if st.Stamina >= st.MaxStamina {
			st.Stamina = st.MaxStamina
			st.LastRecoverAt = now
		}
		st.UpdateTime = now
		if err := tx.Save(st).Error; err != nil {
			return err
		}
		if err := s.createLog(tx, userId, aiStaminaBizApple, fmt.Sprintf("%d", now), recoverCount, st.Stamina, fmt.Sprintf("apple recover: count=%d coinCost=%d", recoverCount, coinCost), now); err != nil {
			return err
		}
		status := s.buildStatus(st)
		ret = &AIStaminaAppleResult{
			AIStaminaStatus: *status,
			RequestedCount:  count,
			RecoveredCount:  recoverCount,
			CoinCost:        coinCost,
			BalanceAfter:    uc.Balance,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *aiStaminaService) GrantByAdmin(adminUserId, userId int64, amount int, remark string) (*AIStaminaGrantResult, error) {
	if adminUserId <= 0 {
		return nil, errors.New("adminUserId is required")
	}
	if userId <= 0 {
		return nil, errors.New("userId is required")
	}
	if amount <= 0 {
		return nil, errors.New("amount must be positive")
	}
	remark = strings.TrimSpace(remark)
	if remark == "" {
		return nil, errors.New("remark is required")
	}
	var ret *AIStaminaGrantResult
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		now := dates.NowTimestamp()
		st, err := s.getOrCreateForUpdate(tx, userId, now)
		if err != nil {
			return err
		}
		if err := s.recoverByTime(tx, st, time.Now()); err != nil {
			return err
		}
		missing := st.MaxStamina - st.Stamina
		if missing <= 0 {
			return errors.New("AI_STAMINA_FULL")
		}
		grantAmount := amount
		if grantAmount > missing {
			grantAmount = missing
		}
		st.Stamina += grantAmount
		if st.Stamina >= st.MaxStamina {
			st.Stamina = st.MaxStamina
			st.LastRecoverAt = now
		}
		st.UpdateTime = now
		if err := tx.Save(st).Error; err != nil {
			return err
		}
		if err := s.createLog(tx, userId, aiStaminaBizAdmin, fmt.Sprintf("%d", adminUserId), grantAmount, st.Stamina, remark, now); err != nil {
			return err
		}
		status := s.buildStatus(st)
		ret = &AIStaminaGrantResult{
			AIStaminaStatus: *status,
			GrantedAmount:   grantAmount,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *aiStaminaService) getOrCreateForUpdate(tx *gorm.DB, userId int64, now int64) (*models.UserAIStamina, error) {
	st := &models.UserAIStamina{}
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userId).Take(st).Error
	if err == nil {
		s.normalize(st, now)
		return st, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	st = &models.UserAIStamina{
		UserId:         userId,
		Stamina:        s.defaultMaxStamina(),
		MaxStamina:     s.defaultMaxStamina(),
		LastRecoverAt:  now,
		DailyUsedDate:  biztime.DayNameCST(time.Now()),
		DailyUsedCount: 0,
		CreateTime:     now,
		UpdateTime:     now,
	}
	if err := tx.Create(st).Error; err != nil {
		err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userId).Take(st).Error
		if err != nil {
			return nil, err
		}
	}
	s.normalize(st, now)
	return st, nil
}

func (s *aiStaminaService) normalize(st *models.UserAIStamina, now int64) {
	if st.MaxStamina <= 0 {
		st.MaxStamina = s.defaultMaxStamina()
	}
	if st.Stamina > st.MaxStamina {
		st.Stamina = st.MaxStamina
	}
	if st.Stamina < 0 {
		st.Stamina = 0
	}
	if st.LastRecoverAt <= 0 {
		st.LastRecoverAt = now
	}
	today := biztime.DayNameCST(time.Now())
	if st.DailyUsedDate != today {
		st.DailyUsedDate = today
		st.DailyUsedCount = 0
	}
}

func (s *aiStaminaService) recoverByTime(tx *gorm.DB, st *models.UserAIStamina, nowTime time.Time) error {
	if st == nil {
		return errors.New("stamina is required")
	}
	recovered, nextLastRecoverAt := computeAIStaminaNaturalRecover(st.Stamina, st.MaxStamina, st.LastRecoverAt, nowTime.Unix(), s.recoverMinutes())
	if recovered <= 0 {
		return nil
	}
	st.Stamina += recovered
	st.LastRecoverAt = nextLastRecoverAt
	st.UpdateTime = dates.NowTimestamp()
	if err := tx.Save(st).Error; err != nil {
		return err
	}
	return s.createLog(tx, st.UserId, aiStaminaBizRecover, fmt.Sprintf("%d", st.LastRecoverAt), recovered, st.Stamina, "natural recover", st.UpdateTime)
}

func computeAIStaminaNaturalRecover(stamina int, maxStamina int, lastRecoverAt int64, now int64, recoverMinutes int) (int, int64) {
	if maxStamina <= 0 || stamina >= maxStamina || lastRecoverAt <= 0 || now <= lastRecoverAt || recoverMinutes <= 0 {
		return 0, lastRecoverAt
	}
	recoverSeconds := int64(recoverMinutes * 60)
	periods := int((now - lastRecoverAt) / recoverSeconds)
	if periods <= 0 {
		return 0, lastRecoverAt
	}
	recovered := periods
	missing := maxStamina - stamina
	if recovered > missing {
		recovered = missing
	}
	nextLastRecoverAt := lastRecoverAt + int64(periods)*recoverSeconds
	if stamina+recovered >= maxStamina {
		nextLastRecoverAt = now
	}
	return recovered, nextLastRecoverAt
}

func (s *aiStaminaService) buildStatus(st *models.UserAIStamina) *AIStaminaStatus {
	nextRecoverAt := int64(0)
	if st.Stamina < st.MaxStamina {
		nextRecoverAt = st.LastRecoverAt + int64(s.recoverMinutes()*60)
	}
	limit := s.dailyLimit()
	remaining := -1
	if limit > 0 {
		remaining = limit - st.DailyUsedCount
		if remaining < 0 {
			remaining = 0
		}
	}
	return &AIStaminaStatus{
		UserId:         st.UserId,
		Stamina:        st.Stamina,
		MaxStamina:     st.MaxStamina,
		NextRecoverAt:  nextRecoverAt,
		DailyUsedCount: st.DailyUsedCount,
		DailyLimit:     limit,
		RecoverMinutes: s.recoverMinutes(),
		AppleCoinCost:  s.appleCoinCost(),
		LastRecoverAt:  st.LastRecoverAt,
		DailyRemaining: remaining,
	}
}

func (s *aiStaminaService) createLog(tx *gorm.DB, userId int64, bizType string, bizId string, amount int, staminaAfter int, remark string, now int64) error {
	return tx.Create(&models.UserAIStaminaLog{
		UserId:       userId,
		BizType:      bizType,
		BizId:        bizId,
		Amount:       amount,
		StaminaAfter: staminaAfter,
		Remark:       remark,
		CreateTime:   now,
	}).Error
}

func (s *aiStaminaService) defaultMaxStamina() int {
	if config.Instance != nil && config.Instance.AIChat.DefaultMaxStamina > 0 {
		return config.Instance.AIChat.DefaultMaxStamina
	}
	return 5
}

func (s *aiStaminaService) recoverMinutes() int {
	if config.Instance != nil && config.Instance.AIChat.StaminaRecoverMinutes > 0 {
		return config.Instance.AIChat.StaminaRecoverMinutes
	}
	return 60
}

func (s *aiStaminaService) appleCoinCost() int {
	if config.Instance != nil && config.Instance.AIChat.AppleCoinCost > 0 {
		return config.Instance.AIChat.AppleCoinCost
	}
	return 5
}

func (s *aiStaminaService) dailyLimit() int {
	if config.Instance != nil {
		return config.Instance.AIChat.DailyUserMessageLimit
	}
	return 50
}
