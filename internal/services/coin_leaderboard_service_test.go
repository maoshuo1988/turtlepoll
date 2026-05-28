package services

import (
	"bbs-go/internal/models/models"
	"bbs-go/internal/pkg/idcodec"
	"testing"

	"github.com/mlogclub/simple/sqls"
	"github.com/stretchr/testify/require"
)

func TestCoinLeaderboardService_TopByBalance(t *testing.T) {
	db := sqls.DB()
	if db == nil {
		t.Skip("sqls.DB() is nil; skipping integration-style leaderboard test")
	}

	users := []models.User{
		{Nickname: "u1", Avatar: "a1"},
		{Nickname: "u2", Avatar: "a2"},
		{Nickname: "u3", Avatar: "a3"},
	}
	for i := range users {
		require.NoError(t, db.Create(&users[i]).Error)
	}
	t.Cleanup(func() {
		ids := []int64{users[0].Id, users[1].Id, users[2].Id}
		db.Unscoped().Where("user_id IN ?", ids).Delete(&models.PredictUserStat{})
		db.Unscoped().Where("user_id IN ?", ids).Delete(&models.UserCoinLog{})
		db.Unscoped().Where("user_id IN ?", ids).Delete(&models.UserCoin{})
		db.Unscoped().Where("id IN ?", ids).Delete(&models.User{})
	})

	coins := []models.UserCoin{
		{UserId: users[0].Id, Balance: 100},
		{UserId: users[1].Id, Balance: 300},
		{UserId: users[2].Id, Balance: 200},
	}
	for i := range coins {
		require.NoError(t, db.Create(&coins[i]).Error)
	}
	require.NoError(t, db.Create(&models.PredictUserStat{
		UserId:             users[1].Id,
		SettledMarketCount: 2,
		WinMarketCount:     1,
		LoseMarketCount:    1,
		WinRate:            0.5,
		CurrentWinStreak:   1,
	}).Error)

	res, err := CoinLeaderboardService.TopByBalance(users[2].Id, 2)
	require.NoError(t, err)
	require.Len(t, res.Items, 2)
	require.Equal(t, idcodec.Encode(users[1].Id), res.Items[0].UserId)
	require.EqualValues(t, 1, res.Items[0].Rank)
	require.Equal(t, "u2", res.Items[0].Nickname)
	require.Equal(t, "a2", res.Items[0].Avatar)
	require.EqualValues(t, 300, res.Items[0].Balance)
	require.InDelta(t, 0.5, res.Items[0].WinRate, 0.0001)
	require.EqualValues(t, 1, res.Items[0].CurrentWinStreak)
	require.Equal(t, idcodec.Encode(users[2].Id), res.Items[1].UserId)

	if res.MyRank == nil {
		t.Fatal("myRank should not be nil")
	}
	require.EqualValues(t, 2, *res.MyRank)
	require.EqualValues(t, 200, res.MyBalance)
	require.EqualValues(t, 0, res.MyWinRate)
	require.EqualValues(t, 0, res.MyCurrentWinStreak)
}
