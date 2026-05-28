package services

import (
	"bbs-go/internal/models/models"
	"errors"

	"github.com/mlogclub/simple/common/dates"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var PredictUserStatService = newPredictUserStatService()

func newPredictUserStatService() *predictUserStatService {
	return &predictUserStatService{}
}

type predictUserStatService struct{}

type PredictUserMarketStatInput struct {
	UserId          int64
	MarketId        int64
	Result          string
	BetAmount       int64
	Payout          int64
	SettledBetCount int64
	SettleTime      int64
}

// RecordMarketResult 幂等记录用户在单个预测市场的战绩，并增量更新用户总战绩。
func (s *predictUserStatService) RecordMarketResult(tx *gorm.DB, input PredictUserMarketStatInput) error {
	if tx == nil {
		return errors.New("tx is required")
	}
	if input.UserId <= 0 {
		return errors.New("userId is required")
	}
	if input.MarketId <= 0 {
		return errors.New("marketId is required")
	}
	if input.Result != "WIN" && input.Result != "LOSE" && input.Result != "VOID" {
		return errors.New("result must be WIN/LOSE/VOID")
	}
	if input.SettleTime <= 0 {
		input.SettleTime = dates.NowTimestamp()
	}

	existing := &models.PredictUserMarketStat{}
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND market_id = ?", input.UserId, input.MarketId).
		Take(existing).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	now := dates.NowTimestamp()
	marketStat := &models.PredictUserMarketStat{
		UserId:          input.UserId,
		MarketId:        input.MarketId,
		Result:          input.Result,
		BetAmount:       input.BetAmount,
		Payout:          input.Payout,
		SettledBetCount: input.SettledBetCount,
		SettleTime:      input.SettleTime,
		CreateTime:      now,
		UpdateTime:      now,
	}
	createResult := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "market_id"}},
		DoNothing: true,
	}).Create(marketStat)
	if createResult.Error != nil {
		return createResult.Error
	}
	if createResult.RowsAffected == 0 {
		return nil
	}
	if err := createResult.Error; err != nil {
		return err
	}
	if input.Result == "VOID" {
		return nil
	}

	seed := &models.PredictUserStat{
		UserId:     input.UserId,
		CreateTime: now,
		UpdateTime: now,
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoNothing: true,
	}).Create(seed).Error; err != nil {
		return err
	}

	stat := &models.PredictUserStat{}
	err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", input.UserId).
		Take(stat).Error
	if err != nil {
		return err
	}

	stat.SettledMarketCount++
	stat.LastSettledMarketId = input.MarketId
	stat.LastSettledAt = input.SettleTime
	stat.UpdateTime = now

	if input.Result == "WIN" {
		stat.WinMarketCount++
		stat.CurrentWinStreak++
		if stat.CurrentWinStreak > stat.BestWinStreak {
			stat.BestWinStreak = stat.CurrentWinStreak
		}
	} else {
		stat.LoseMarketCount++
		stat.CurrentWinStreak = 0
	}

	if stat.SettledMarketCount > 0 {
		stat.WinRate = float64(stat.WinMarketCount) / float64(stat.SettledMarketCount)
	} else {
		stat.WinRate = 0
	}

	return tx.Save(stat).Error
}
