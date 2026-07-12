package services

import (
	"bbs-go/internal/models/models"
	"errors"
	"fmt"
	"math"
	"strconv"
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
// - market.Result 作为胜方选项（A/B/DRAW），忽略大小写
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
	pushEvents := make([]AISettlementEvent, 0)
	marketTitle := ""

	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		market := &models.PredictMarket{}
		// 加锁：避免并发结算导致同一笔订单重复派奖
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Take(market, "id = ?", marketId).Error; err != nil {
			return err
		}
		if market.Status != "SETTLED" {
			return errors.New("market is not settled")
		}
		marketTitle = market.Title
		winner := strings.ToUpper(strings.TrimSpace(market.Result))
		if !IsValidPredictOption(market.MarketType, winner) {
			return errors.New("market result must match market options")
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

		totalBetAmount := int64(0)
		totalPayout := int64(0)
		settledBetCount := int64(0)
		hasWin := false
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
				hasWin = hasWin || payout > 0
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
			pushEvents = append(pushEvents, AISettlementEvent{
				UserId:       bet.UserId,
				BizType:      "predict",
				BizId:        fmt.Sprintf("%d", bet.Id),
				ContextType:  "predict_market",
				ContextId:    marketId,
				ProfitAmount: payout - bet.Amount,
				SettleResult: strings.ToLower(settleResult),
				EventTitle:   market.Title,
				SettledAt:    now,
			})
			totalBetAmount += bet.Amount
			totalPayout += payout
			settledBetCount++
		}

		marketResult := "LOSE"
		if hasWin && totalPayout > 0 {
			marketResult = "WIN"
		}
		if err := PredictUserStatService.RecordMarketResult(tx, PredictUserMarketStatInput{
			UserId:          userId,
			MarketId:        marketId,
			Result:          marketResult,
			BetAmount:       totalBetAmount,
			Payout:          totalPayout,
			SettledBetCount: settledBetCount,
			SettleTime:      now,
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, event := range pushEvents {
		_, _ = AIPushService.OnSettlement(event)
	}
	for _, result := range results {
		s.pushPredictSettleMessage(userId, marketId, marketTitle, result)
	}
	return results, nil
}

func (s *predictSettleService) pushPredictSettleMessage(userId, marketId int64, marketTitle string, result *SettleMyBetResult) {
	if result == nil || result.Bet == nil {
		return
	}
	templateCode := "predict_settle_lose"
	settleResult := strings.ToLower(strings.TrimSpace(result.Bet.SettleResult))
	if settleResult == "win" {
		templateCode = "predict_settle_win"
	}
	if settleResult == "refund" || settleResult == "draw" {
		templateCode = "predict_settle_refund"
	}
	if strings.TrimSpace(marketTitle) == "" {
		marketTitle = fmt.Sprintf("预测市场 %d", marketId)
	}
	_, _ = MessageNotifyService.PushByTemplate(MessageNotifyPushInput{
		BusinessCode: MessageNotifyBusinessDarkMarket,
		TemplateCode: templateCode,
		UserId:       userId,
		Params: map[string]string{
			"marketTitle": marketTitle,
			"payout":      strconv.FormatInt(result.Payout, 10),
			"amount":      strconv.FormatInt(result.Bet.Amount, 10),
			"marketId":    strconv.FormatInt(marketId, 10),
		},
		ExtraData: map[string]any{
			"betId":        result.Bet.Id,
			"marketId":     marketId,
			"option":       result.Bet.Option,
			"settleResult": result.Bet.SettleResult,
			"payout":       result.Payout,
		},
		BizId:          strconv.FormatInt(result.Bet.Id, 10),
		IdempotencyKey: fmt.Sprintf("predict_settle:%d:%s", result.Bet.Id, settleResult),
	})
}
