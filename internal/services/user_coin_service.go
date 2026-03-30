package services

import (
	"bbs-go/internal/models"
	"bbs-go/internal/repositories"
	"errors"
	"fmt"

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

// Transfer 金币转账（同一事务内）：fromUserId 扣款，toUserId 入账。
// 说明：
// - 支持负数 userId（系统账户：资金池/销毁账户）。
// - amount 必须为正数。
// - 该方法只保证账户层面的原子性与审计（UserCoin + UserCoinLog）。
func (s *userCoinService) Transfer(tx *gorm.DB, fromUserId, toUserId int64, bizType string, bizId int64, amount int64, remark string) error {
	if fromUserId == 0 || toUserId == 0 {
		return errors.New("fromUserId/toUserId is required")
	}
	if amount <= 0 {
		return errors.New("amount must be positive")
	}
	if bizType == "" {
		return errors.New("bizType is required")
	}
	if tx == nil {
		return errors.New("tx is required")
	}

	now := dates.NowTimestamp()

	// from
	from, err := repositories.UserCoinRepository.GetOrCreate(tx, fromUserId)
	if err != nil {
		return err
	}
	if from.Balance < amount {
		return errors.New("insufficient balance")
	}
	from.Balance -= amount
	from.UpdateTime = now
	if from.CreateTime == 0 {
		from.CreateTime = now
	}
	if err := repositories.UserCoinRepository.Update(tx, from); err != nil {
		return err
	}
	if err := repositories.UserCoinLogRepository.Create(tx, &models.UserCoinLog{
		UserId:       fromUserId,
		BizType:      bizType,
		BizId:        bizId,
		Amount:       -amount,
		BalanceAfter: from.Balance,
		Remark:       fmt.Sprintf("%s | transfer_out to=%d", remark, toUserId),
		CreateTime:   now,
	}); err != nil {
		return err
	}

	// to
	to, err := repositories.UserCoinRepository.GetOrCreate(tx, toUserId)
	if err != nil {
		return err
	}
	to.Balance += amount
	to.UpdateTime = now
	if to.CreateTime == 0 {
		to.CreateTime = now
	}
	if err := repositories.UserCoinRepository.Update(tx, to); err != nil {
		return err
	}
	if err := repositories.UserCoinLogRepository.Create(tx, &models.UserCoinLog{
		UserId:       toUserId,
		BizType:      bizType,
		BizId:        bizId,
		Amount:       amount,
		BalanceAfter: to.Balance,
		Remark:       fmt.Sprintf("%s | transfer_in from=%d", remark, fromUserId),
		CreateTime:   now,
	}); err != nil {
		return err
	}

	return nil
}

// SpendToPool 扣用户下注本金并入资金池。
func (s *userCoinService) SpendToPool(tx *gorm.DB, userId int64, bizType string, bizId int64, amount int64, remark string) error {
	return s.Transfer(tx, userId, models.BattlePoolUserId, bizType, bizId, amount, remark)
}

// PayEntryFeeToBanker 扣挑战者入场费并实时转给庄家（不进资金池）。
func (s *userCoinService) PayEntryFeeToBanker(tx *gorm.DB, challengerUserId, bankerUserId int64, bizType string, bizId int64, amount int64, remark string) error {
	return s.Transfer(tx, challengerUserId, bankerUserId, bizType, bizId, amount, remark)
}

// PayFromPoolToUser 从资金池给用户出池（提取/退款/结算等）。
func (s *userCoinService) PayFromPoolToUser(tx *gorm.DB, userId int64, bizType string, bizId int64, amount int64, remark string) error {
	return s.Transfer(tx, models.BattlePoolUserId, userId, bizType, bizId, amount, remark)
}

// BurnFromPool 从资金池划转到销毁账户（罚没/处罚）。
func (s *userCoinService) BurnFromPool(tx *gorm.DB, bizType string, bizId int64, amount int64, remark string) error {
	return s.Transfer(tx, models.BattlePoolUserId, models.BattleBurnUserId, bizType, bizId, amount, remark)
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
