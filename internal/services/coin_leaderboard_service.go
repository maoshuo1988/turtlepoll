package services

import (
	"bbs-go/internal/models/models"
	"errors"
	"sort"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

var CoinLeaderboardService = newCoinLeaderboardService()

func newCoinLeaderboardService() *coinLeaderboardService {
	return &coinLeaderboardService{}
}

type coinLeaderboardService struct{}

type CoinLeaderboardItem struct {
	Rank             int64   `json:"rank"`
	UserId           int64   `json:"userId"`
	Nickname         string  `json:"nickname"`
	Avatar           string  `json:"avatar"`
	Balance          int64   `json:"balance"`
	WinRate          float64 `json:"winRate"`
	CurrentWinStreak int64   `json:"currentWinStreak"`
}

type CoinLeaderboardResult struct {
	Items              []CoinLeaderboardItem `json:"items"`
	MyRank             *int64                `json:"myRank"`
	MyBalance          int64                 `json:"myBalance"`
	MyWinRate          float64               `json:"myWinRate"`
	MyCurrentWinStreak int64                 `json:"myCurrentWinStreak"`
}

func (s *coinLeaderboardService) TopByBalance(userId int64, limit int) (*CoinLeaderboardResult, error) {
	if userId <= 0 {
		return nil, errors.New("userId is required")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	db := sqls.DB()
	var coins []models.UserCoin
	if err := db.Where("user_id > 0").
		Order("balance DESC").
		Order("user_id ASC").
		Limit(limit).
		Find(&coins).Error; err != nil {
		return nil, err
	}

	userIds := make([]int64, 0, len(coins)+1)
	seen := map[int64]struct{}{}
	for _, coin := range coins {
		userIds = append(userIds, coin.UserId)
		seen[coin.UserId] = struct{}{}
	}
	if _, ok := seen[userId]; !ok {
		userIds = append(userIds, userId)
	}

	usersById, err := s.loadUsers(userIds)
	if err != nil {
		return nil, err
	}
	statsByUserId, err := s.loadStats(userIds)
	if err != nil {
		return nil, err
	}

	items := make([]CoinLeaderboardItem, 0, len(coins))
	for i, coin := range coins {
		item := CoinLeaderboardItem{
			Rank:    int64(i + 1),
			UserId:  coin.UserId,
			Balance: coin.Balance,
		}
		if user := usersById[coin.UserId]; user != nil {
			item.Nickname = user.Nickname
			item.Avatar = user.Avatar
		}
		if stat := statsByUserId[coin.UserId]; stat != nil {
			item.WinRate = stat.WinRate
			item.CurrentWinStreak = stat.CurrentWinStreak
		}
		items = append(items, item)
	}

	myCoin, err := s.getUserCoin(userId)
	if err != nil {
		return nil, err
	}
	myRank, err := s.calcRank(userId, myCoin)
	if err != nil {
		return nil, err
	}

	result := &CoinLeaderboardResult{
		Items:     items,
		MyRank:    myRank,
		MyBalance: 0,
	}
	if myCoin != nil {
		result.MyBalance = myCoin.Balance
	}
	if stat := statsByUserId[userId]; stat != nil {
		result.MyWinRate = stat.WinRate
		result.MyCurrentWinStreak = stat.CurrentWinStreak
	}
	return result, nil
}

func (s *coinLeaderboardService) loadUsers(userIds []int64) (map[int64]*models.User, error) {
	ret := map[int64]*models.User{}
	if len(userIds) == 0 {
		return ret, nil
	}
	userIds = uniqueInt64(userIds)
	var users []models.User
	if err := sqls.DB().Where("id IN ?", userIds).Find(&users).Error; err != nil {
		return nil, err
	}
	for i := range users {
		u := users[i]
		ret[u.Id] = &u
	}
	return ret, nil
}

func (s *coinLeaderboardService) loadStats(userIds []int64) (map[int64]*models.PredictUserStat, error) {
	ret := map[int64]*models.PredictUserStat{}
	if len(userIds) == 0 {
		return ret, nil
	}
	userIds = uniqueInt64(userIds)
	var stats []models.PredictUserStat
	if err := sqls.DB().Where("user_id IN ?", userIds).Find(&stats).Error; err != nil {
		return nil, err
	}
	for i := range stats {
		stat := stats[i]
		ret[stat.UserId] = &stat
	}
	return ret, nil
}

func (s *coinLeaderboardService) getUserCoin(userId int64) (*models.UserCoin, error) {
	coin := &models.UserCoin{}
	err := sqls.DB().Where("user_id = ?", userId).Take(coin).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return coin, nil
}

func (s *coinLeaderboardService) calcRank(userId int64, coin *models.UserCoin) (*int64, error) {
	if coin == nil {
		return nil, nil
	}
	var count int64
	if err := sqls.DB().Model(&models.UserCoin{}).
		Where("user_id > 0").
		Where("balance > ? OR (balance = ? AND user_id < ?)", coin.Balance, coin.Balance, userId).
		Count(&count).Error; err != nil {
		return nil, err
	}
	rank := count + 1
	return &rank, nil
}

func uniqueInt64(input []int64) []int64 {
	seen := map[int64]struct{}{}
	ret := make([]int64, 0, len(input))
	for _, v := range input {
		if v == 0 {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		ret = append(ret, v)
	}
	sort.Slice(ret, func(i, j int) bool { return ret[i] < ret[j] })
	return ret
}
