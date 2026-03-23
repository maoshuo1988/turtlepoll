package services

import (
	"bbs-go/internal/models"
	"bbs-go/internal/repositories"
	"errors"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

var UserCoinService = newUserCoinService()

func newUserCoinService() *userCoinService {
	return &userCoinService{}
}

type userCoinService struct{}

func (s *userCoinService) GetOrCreate(userId int64) (*models.UserCoin, error) {
	if userId <= 0 {
		return nil, errors.New("userId is required")
	}
	return repositories.UserCoinRepository.GetOrCreate(sqls.DB(), userId)
}

// SettleBet 结算入账（派奖）。
// 说明：
// - 必须在外层事务中调用（结算服务需要同时更新 bet 状态），因此这里接收 tx。
// - payout 允许为 0（输单也会标记为已结算，但不会入账），此时仍会写一条流水便于审计。
func (s *userCoinService) SettleBet(tx *gorm.DB, userId, betId, payout int64, remark string) (*models.UserCoin, error) {
	if userId <= 0 {
		return nil, errors.New("userId is required")
	}
	if betId <= 0 {
		return nil, errors.New("betId is required")
	}
	if payout < 0 {
		return nil, errors.New("payout must be non-negative")
	}

	now := dates.NowTimestamp()
	uc, err := repositories.UserCoinRepository.GetOrCreate(tx, userId)
	if err != nil {
		return nil, err
	}
	uc.Balance += payout
	uc.UpdateTime = now
	if uc.CreateTime == 0 {
		uc.CreateTime = now
	}
	if err := repositories.UserCoinRepository.Update(tx, uc); err != nil {
		return nil, err
	}

	log := &models.UserCoinLog{
		UserId:       userId,
		BizType:      "SETTLE",
		BizId:        betId,
		Amount:       payout,
		BalanceAfter: uc.Balance,
		Remark:       remark,
		CreateTime:   now,
	}
	if err := repositories.UserCoinLogRepository.Create(tx, log); err != nil {
		return nil, err
	}
	return uc, nil
}

// Mint 管理员铸币：给用户加金币，并记录流水。
func (s *userCoinService) Mint(adminUserId, userId, amount int64, remark string) (*models.UserCoin, error) {
	if userId <= 0 {
		return nil, errors.New("userId is required")
	}
	if amount <= 0 {
		return nil, errors.New("amount must be positive")
	}

	now := dates.NowTimestamp()
	return s.withTx(func(tx *gorm.DB) (*models.UserCoin, error) {
		uc, err := repositories.UserCoinRepository.GetOrCreate(tx, userId)
		if err != nil {
			return nil, err
		}
		newBalance := uc.Balance + amount
		uc.Balance = newBalance
		uc.UpdateTime = now
		if uc.CreateTime == 0 {
			uc.CreateTime = now
		}
		if err := repositories.UserCoinRepository.Update(tx, uc); err != nil {
			return nil, err
		}

		log := &models.UserCoinLog{
			UserId:       userId,
			BizType:      "MINT",
			BizId:        adminUserId, // 记录操作者
			Amount:       amount,
			BalanceAfter: newBalance,
			Remark:       remark,
			CreateTime:   now,
		}
		if err := repositories.UserCoinLogRepository.Create(tx, log); err != nil {
			return nil, err
		}

		return uc, nil
	})
}

// SpendBet 下单扣款（下注）。
func (s *userCoinService) SpendBet(tx *gorm.DB, userId, betId, amount int64, remark string) (*models.UserCoin, error) {
	if userId <= 0 {
		return nil, errors.New("userId is required")
	}
	if amount <= 0 {
		return nil, errors.New("amount must be positive")
	}

	now := dates.NowTimestamp()
	uc, err := repositories.UserCoinRepository.GetOrCreate(tx, userId)
	if err != nil {
		return nil, err
	}
	if uc.Balance < amount {
		return nil, errors.New("insufficient balance")
	}
	uc.Balance -= amount
	uc.UpdateTime = now
	if uc.CreateTime == 0 {
		uc.CreateTime = now
	}
	if err := repositories.UserCoinRepository.Update(tx, uc); err != nil {
		return nil, err
	}
	log := &models.UserCoinLog{
		UserId:       userId,
		BizType:      "BET",
		BizId:        betId,
		Amount:       -amount,
		BalanceAfter: uc.Balance,
		Remark:       remark,
		CreateTime:   now,
	}
	if err := repositories.UserCoinLogRepository.Create(tx, log); err != nil {
		return nil, err
	}
	return uc, nil
}

func (s *userCoinService) withTx(fn func(tx *gorm.DB) (*models.UserCoin, error)) (*models.UserCoin, error) {
	returnValue := &models.UserCoin{}
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		uc, err := fn(tx)
		if err != nil {
			return err
		}
		*returnValue = *uc
		return nil
	})
	if err != nil {
		return nil, err
	}
	return returnValue, nil
}
