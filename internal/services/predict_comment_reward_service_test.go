package services

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/models"
	"testing"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"github.com/stretchr/testify/require"
)

func TestPredictCommentRewardService_RunForMarket(t *testing.T) {
	db := sqls.DB()
	if db == nil {
		t.Skip("sqls.DB() is nil; skipping integration-style comment reward test")
	}

	now := dates.NowTimestamp()
	market := &models.PredictMarket{
		SourceModel:   "RewardTest",
		SourceModelId: now,
		Title:         "reward market",
		MarketType:    "binary",
		Status:        "SETTLED",
		Result:        "A",
		Resolved:      true,
		ResolvedAt:    now,
		PoolA:         700,
		PoolB:         300,
	}
	require.NoError(t, db.Create(market).Error)

	users := []int64{990101, 990102, 990103}
	t.Cleanup(func() {
		db.Unscoped().Where("market_id = ?", market.Id).Delete(&models.PredictCommentRewardItem{})
		db.Unscoped().Where("market_id = ?", market.Id).Delete(&models.PredictCommentRewardLog{})
		db.Unscoped().Where("market_id = ?", market.Id).Delete(&models.PredictCommentMeta{})
		db.Unscoped().Where("entity_type = ? AND entity_id = ?", constants.EntityPredictMarket, market.Id).Delete(&models.Comment{})
		db.Unscoped().Where("user_id IN ?", users).Delete(&models.UserCoinLog{})
		db.Unscoped().Where("user_id IN ?", users).Delete(&models.UserCoin{})
		db.Unscoped().Delete(&models.PredictMarket{}, market.Id)
	})

	commentRows := []struct {
		userId int64
		option string
	}{
		{users[0], "A"},
		{users[0], "A"},
		{users[1], "A"},
		{users[2], "B"},
	}
	for _, row := range commentRows {
		comment := &models.Comment{
			UserId:      row.userId,
			EntityType:  constants.EntityPredictMarket,
			EntityId:    market.Id,
			Content:     "comment",
			ContentType: constants.ContentTypeText,
			Status:      constants.StatusOk,
			CreateTime:  now,
		}
		require.NoError(t, db.Create(comment).Error)
		require.NoError(t, PredictCommentMetaService.CreateForComment(db, market, comment, row.option))
	}

	log, err := PredictCommentRewardService.RunForMarket(market.Id, false)
	require.NoError(t, err)
	require.Equal(t, "PAID", log.Status)
	require.EqualValues(t, 1000, log.MarketBetTotal)
	require.EqualValues(t, 100, log.RewardPool)
	require.EqualValues(t, 2, log.WinnerCommentUserCount)
	require.EqualValues(t, 50, log.PerUserReward)
	require.EqualValues(t, 0, log.Remainder)

	var items []models.PredictCommentRewardItem
	require.NoError(t, db.Where("reward_log_id = ?", log.Id).Order("user_id asc").Find(&items).Error)
	require.Len(t, items, 2)
	require.Equal(t, users[0], items[0].UserId)
	require.EqualValues(t, 2, items[0].CommentCount)
	require.EqualValues(t, 50, items[0].Amount)
	require.Equal(t, users[1], items[1].UserId)
	require.EqualValues(t, 1, items[1].CommentCount)

	var coinLogs []models.UserCoinLog
	require.NoError(t, db.Where("biz_type = ? AND biz_id = ?", PredictCommentRewardBizType, log.Id).Find(&coinLogs).Error)
	require.Len(t, coinLogs, 2)

	log2, err := PredictCommentRewardService.RunForMarket(market.Id, false)
	require.NoError(t, err)
	require.Equal(t, log.Id, log2.Id)
	var itemCount int64
	require.NoError(t, db.Model(&models.PredictCommentRewardItem{}).Where("reward_log_id = ?", log.Id).Count(&itemCount).Error)
	require.EqualValues(t, 2, itemCount)
}
