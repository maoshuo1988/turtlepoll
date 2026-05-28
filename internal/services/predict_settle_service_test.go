package services

import (
	"bbs-go/internal/models/models"
	"testing"

	"github.com/mlogclub/simple/sqls"
	"github.com/stretchr/testify/require"
)

func TestPredictSettleService_CalcPayoutAndIdempotent(t *testing.T) {
	// 这个测试依赖项目现有的测试环境初始化（sqls.DB() 可用）。
	// 如果你的 CI 没有 DB，可把它降级为“纯函数”测试；不过当前仓库多数 service 测试是轻量的。
	db := sqls.DB()
	if db == nil {
		t.Skip("sqls.DB() is nil; skipping integration-style settle test")
	}

	userId := int64(999001)
	// 准备 market + bet + coin
	market := &models.PredictMarket{
		SourceModel:   "Test",
		SourceModelId: 1,
		Title:         "test market",
		MarketType:    "AB",
		Status:        "SETTLED",
		Result:        "A",
		BaseA:         500,
		BaseB:         500,
		PoolA:         0,
		PoolB:         0,
	}
	require.NoError(t, db.Create(market).Error)
	t.Cleanup(func() {
		db.Unscoped().Where("market_id = ?", market.Id).Delete(&models.PredictBet{})
		db.Unscoped().Where("market_id = ?", market.Id).Delete(&models.PredictUserMarketStat{})
		db.Unscoped().Where("user_id = ?", userId).Delete(&models.PredictUserStat{})
		db.Unscoped().Where("user_id = ?", userId).Delete(&models.UserCoinLog{})
		db.Unscoped().Where("user_id = ?", userId).Delete(&models.UserCoin{})
		db.Unscoped().Delete(&models.PredictMarket{}, market.Id)
	})

	// 给用户铸币，确保够扣
	_, err := UserCoinService.Mint(1, userId, 10000, "test")
	require.NoError(t, err)

	// 下 2 笔：A 赢、B 输。
	betA := &models.PredictBet{UserId: userId, MarketId: market.Id, Option: "A", Amount: 100, Odds: 1.83, EffA: 500, EffB: 500, Status: "OPEN"}
	betB := &models.PredictBet{UserId: userId, MarketId: market.Id, Option: "B", Amount: 200, Odds: 2.20, EffA: 500, EffB: 500, Status: "OPEN"}
	require.NoError(t, db.Create(betA).Error)
	require.NoError(t, db.Create(betB).Error)

	// 扣款模拟（测试只关心结算逻辑，不走 PlaceBet）
	_, err = UserCoinService.SpendBet(db, userId, betA.Id, betA.Amount, "test")
	require.NoError(t, err)
	_, err = UserCoinService.SpendBet(db, userId, betB.Id, betB.Amount, "test")
	require.NoError(t, err)

	res, err := PredictSettleService.SettleMyBet(userId, market.Id)
	require.NoError(t, err)
	require.Len(t, res, 2)
	stat := &models.PredictUserStat{}
	require.NoError(t, db.Take(stat, "user_id = ?", userId).Error)
	require.EqualValues(t, 1, stat.SettledMarketCount)
	require.EqualValues(t, 1, stat.WinMarketCount)
	require.EqualValues(t, 0, stat.LoseMarketCount)
	require.EqualValues(t, 1, stat.CurrentWinStreak)
	require.InDelta(t, 1.0, stat.WinRate, 0.0001)

	marketStat := &models.PredictUserMarketStat{}
	require.NoError(t, db.Take(marketStat, "user_id = ? AND market_id = ?", userId, market.Id).Error)
	require.Equal(t, "WIN", marketStat.Result)
	require.EqualValues(t, 300, marketStat.BetAmount)
	require.EqualValues(t, 183, marketStat.Payout)
	require.EqualValues(t, 2, marketStat.SettledBetCount)

	// 再结算一次，应当幂等：没有 OPEN 的单了
	res2, err := PredictSettleService.SettleMyBet(userId, market.Id)
	require.NoError(t, err)
	require.Len(t, res2, 0)
	stat2 := &models.PredictUserStat{}
	require.NoError(t, db.Take(stat2, "user_id = ?", userId).Error)
	require.EqualValues(t, 1, stat2.SettledMarketCount)
	require.EqualValues(t, 1, stat2.CurrentWinStreak)
}
