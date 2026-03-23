package services

import (
	"bbs-go/internal/models"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PredictSettleService 用户结算服务
var PredictSettleService = newPredictSettleService()

func newPredictSettleService() *predictSettleService {
	return &predictSettleService{}
}

type predictSettleService struct{}

type SettleMyBetResult struct {
	Bet      *models.PredictBet `json:"bet"`
	Payout   int64              `json:"payout"`
	UserCoin *models.UserCoin   `json:"userCoin"`
}

// SettleMyBet 用户结算自己在某个 market 的所有未结算下注单。
// 结算规则（当前版本约定）：
// - market.Status 必须为 SETTLED
// - market.Result 作为胜方选项（A/B），忽略大小写
// - bet.Odds 为下注时锁定赔率
// - payout = floor(bet.Amount * bet.Odds)（输单 payout=0）
// - 幂等：bet.Status=SETTLED 的单不会重复结算
func (s *predictSettleService) SettleMyBet(userId, marketId int64) ([]*SettleMyBetResult, error) {
	if userId <= 0 {
		return nil, errors.New("userId is required")
	}
	if marketId <= 0 {
		return nil, errors.New("marketId is required")
	}

	now := dates.NowTimestamp()
	results := make([]*SettleMyBetResult, 0)

	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		market := &models.PredictMarket{}
		// 加锁：避免并发结算导致同一笔订单重复派奖
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Take(market, "id = ?", marketId).Error; err != nil {
			return err
		}
		if market.Status != "SETTLED" {
			return errors.New("market is not settled")
		}
		winner := strings.ToUpper(strings.TrimSpace(market.Result))
		if winner != PredictOptionA && winner != PredictOptionB {
			return errors.New("market result must be A or B")
		}

		// 锁住该用户在该市场的未结算订单
		var bets []*models.PredictBet
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND market_id = ? AND status = ?", userId, marketId, "OPEN").
			Find(&bets).Error; err != nil {
			return err
		}
		if len(bets) == 0 {
			return nil
		}

		for _, bet := range bets {
			betOption := strings.ToUpper(strings.TrimSpace(bet.Option))
			isWin := betOption == winner
			payout := int64(0)
			settleResult := "LOSE"
			if isWin {
				settleResult = "WIN"
				payout = int64(math.Floor(float64(bet.Amount) * bet.Odds))
				if payout < 0 {
					payout = 0
				}
			}

			bet.Status = "SETTLED"
			bet.SettleResult = settleResult
			bet.Payout = payout
			bet.SettleTime = now
			if err := tx.Save(bet).Error; err != nil {
				return err
			}

			remark := fmt.Sprintf("predict settle: marketId=%d, result=%s, odds=%.2f", marketId, winner, bet.Odds)
			uc, err := UserCoinService.SettleBet(tx, bet.UserId, bet.Id, payout, remark)
			if err != nil {
				return err
			}

			results = append(results, &SettleMyBetResult{
				Bet:      bet,
				Payout:   payout,
				UserCoin: uc,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}
