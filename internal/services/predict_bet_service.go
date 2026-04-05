package services

import (
	"bbs-go/internal/models/models"
	"bbs-go/internal/repositories"
	"errors"
	"log/slog"
	"strings"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var PredictBetService = newPredictBetService()

func newPredictBetService() *predictBetService {
	return &predictBetService{}
}

type predictBetService struct{}

type PlaceBetResult struct {
	Bet        *models.PredictBet    `json:"bet"`
	Market     *models.PredictMarket `json:"market"`
	UserCoin   *models.UserCoin      `json:"userCoin"`
	LockedOdds float64               `json:"lockedOdds"`
}

// PlaceBet 用户对预测市场下注。
// - 校验市场状态/截止时间
// - 基于当前池计算赔率并锁定到订单
// - 扣减用户金币
// - 增加市场池（PoolA/PoolB）
func (s *predictBetService) PlaceBet(userId, marketId int64, option string, amount int64) (*PlaceBetResult, error) {
	if userId <= 0 {
		return nil, errors.New("userId is required")
	}
	if marketId <= 0 {
		return nil, errors.New("marketId is required")
	}
	option = strings.ToUpper(strings.TrimSpace(option))
	if option != PredictOptionA && option != PredictOptionB {
		return nil, errors.New("option must be A or B")
	}
	if amount <= 0 {
		return nil, errors.New("amount must be positive")
	}

	nowMs := dates.NowTimestamp()
	// 统一时间戳单位为“秒”，避免 ms/sec 混用导致 closeTime 判断错误。
	nowSec := nowMs
	if nowSec > 1_000_000_000_000 {
		nowSec = nowSec / 1000
	}
	ret := &PlaceBetResult{}

	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		market := &models.PredictMarket{}
		// 加行锁，避免并发下注导致池更新丢失
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Take(market, "id = ?", marketId).Error; err != nil {
			return err
		}
		if market.Status != "OPEN" {
			slog.Warn("predict bet rejected: market not open",
				"marketId", market.Id,
				"status", market.Status,
				"closeTime", market.CloseTime,
				"nowMs", nowMs,
				"nowSec", nowSec,
			)
			return errors.New("market is not open")
		}
		if market.CloseTime > 0 && nowSec >= market.CloseTime {
			slog.Warn("predict bet rejected: market closed by closeTime",
				"marketId", market.Id,
				"status", market.Status,
				"closeTime", market.CloseTime,
				"nowMs", nowMs,
				"nowSec", nowSec,
				"nowGteCloseTime", nowSec >= market.CloseTime,
			)
			return errors.New("market is closed")
		}

		oddsA, oddsB, effA, effB, _ := CalcClampedOdds(market.BaseA, market.BaseB, market.PoolA, market.PoolB)
		lockedOdds := oddsA
		if option == PredictOptionB {
			lockedOdds = oddsB
		}

		bet := &models.PredictBet{
			UserId:     userId,
			MarketId:   marketId,
			Option:     option,
			Amount:     amount,
			Odds:       lockedOdds,
			EffA:       effA,
			EffB:       effB,
			Status:     "OPEN",
			CreateTime: nowSec,
		}
		if err := repositories.PredictBetRepository.Create(tx, bet); err != nil {
			return err
		}

		// 扣款 + 记录流水
		uc, err := UserCoinService.SpendBet(tx, userId, bet.Id, amount, "predict bet")
		if err != nil {
			return err
		}

		// 写入池
		if option == PredictOptionA {
			market.PoolA += amount
		} else {
			market.PoolB += amount
		}
		market.UpdateTime = nowSec
		if err := tx.Save(market).Error; err != nil {
			return err
		}

		ret.Bet = bet
		ret.Market = market
		ret.UserCoin = uc
		ret.LockedOdds = lockedOdds
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}
