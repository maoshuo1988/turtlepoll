package services

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/models"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	PredictCommentRewardBizType = "COMMENT_REWARD"

	predictCommentRewardStatusPending    = "PENDING"
	predictCommentRewardStatusProcessing = "PROCESSING"
	predictCommentRewardStatusPaid       = "PAID"
	predictCommentRewardStatusExpired    = "EXPIRED"
	predictCommentRewardStatusFailed     = "FAILED"
)

var PredictCommentRewardService = newPredictCommentRewardService()

func newPredictCommentRewardService() *predictCommentRewardService {
	return &predictCommentRewardService{}
}

type predictCommentRewardService struct{}

type PredictCommentRewardUser struct {
	UserId         int64
	Option         string
	Heat           float64
	CommentCount   int64
	FirstCommentId int64
}

func toUnixSeconds(ts int64) int64 {
	if ts <= 0 {
		return ts
	}
	if ts > 1_000_000_000_000 {
		return ts / 1000
	}
	return ts
}

func (s *predictCommentRewardService) RunDue() error {
	nowMs := dates.NowTimestamp()
	var markets []models.PredictMarket
	if err := sqls.DB().
		Where("status = ? AND resolved_at > 0 AND resolved_at <= ?", "SETTLED", nowMs).
		Where("resolved_at >= ?", nowMs-3600*1000).
		Order("resolved_at asc").
		Limit(100).
		Find(&markets).Error; err != nil {
		return err
	}
	for i := range markets {
		if _, err := s.RunForMarket(markets[i].Id, false); err != nil {
			// 单个市场失败不阻断后续市场。
			continue
		}
	}
	return s.MarkExpired()
}

func (s *predictCommentRewardService) RunForMarket(marketId int64, forceRetry bool) (*models.PredictCommentRewardLog, error) {
	if marketId <= 0 {
		return nil, errors.New("marketId is required")
	}
	var ret *models.PredictCommentRewardLog
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		log, err := s.runForMarketTx(tx, marketId, forceRetry)
		if err != nil {
			return err
		}
		ret = log
		return nil
	})
	return ret, err
}

func (s *predictCommentRewardService) Retry(rewardLogId int64) (*models.PredictCommentRewardLog, error) {
	if rewardLogId <= 0 {
		return nil, errors.New("rewardLogId is required")
	}
	log := &models.PredictCommentRewardLog{}
	if err := sqls.DB().Take(log, "id = ?", rewardLogId).Error; err != nil {
		return nil, err
	}
	if log.Status != predictCommentRewardStatusFailed {
		return nil, errors.New("only FAILED reward log can be retried")
	}
	return s.RunForMarket(log.MarketId, true)
}

func (s *predictCommentRewardService) MarkExpired() error {
	now := toUnixSeconds(dates.NowTimestamp())
	return sqls.DB().Model(&models.PredictCommentRewardLog{}).
		Where("status = ? AND deadline_at > 0 AND deadline_at < ?", predictCommentRewardStatusPending, now).
		Updates(map[string]interface{}{
			"status":      predictCommentRewardStatusExpired,
			"reason":      "deadline expired",
			"update_time": now,
		}).Error
}

func (s *predictCommentRewardService) runForMarketTx(tx *gorm.DB, marketId int64, forceRetry bool) (*models.PredictCommentRewardLog, error) {
	now := toUnixSeconds(dates.NowTimestamp())
	market := &models.PredictMarket{}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Take(market, "id = ?", marketId).Error; err != nil {
		return nil, err
	}
	if market.Status != "SETTLED" {
		return nil, errors.New("market is not settled")
	}
	winner := strings.ToUpper(strings.TrimSpace(market.Result))
	if !IsValidPredictOption(market.MarketType, winner) {
		return nil, errors.New("market result must match market options")
	}
	settledAt := toUnixSeconds(market.ResolvedAt)
	if settledAt <= 0 {
		settledAt = toUnixSeconds(market.UpdateTime)
	}
	if settledAt <= 0 {
		settledAt = now
	}
	deadlineAt := settledAt + 3600

	log := &models.PredictCommentRewardLog{}
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Take(log, "market_id = ?", marketId).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		log = &models.PredictCommentRewardLog{
			MarketId:     marketId,
			WinnerOption: winner,
			Status:       predictCommentRewardStatusPending,
			SettledAt:    settledAt,
			DeadlineAt:   deadlineAt,
			CreateTime:   now,
			UpdateTime:   now,
		}
		if err := tx.Create(log).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	if log.Status == predictCommentRewardStatusPaid || log.Status == predictCommentRewardStatusProcessing {
		return log, nil
	}
	if log.Status == predictCommentRewardStatusExpired && !forceRetry {
		return log, nil
	}
	if log.Status == predictCommentRewardStatusFailed && !forceRetry {
		return log, nil
	}
	if now > deadlineAt && !forceRetry {
		log.Status = predictCommentRewardStatusExpired
		log.Reason = "deadline expired"
		log.UpdateTime = now
		if err := tx.Save(log).Error; err != nil {
			return nil, err
		}
		return log, nil
	}

	log.Status = predictCommentRewardStatusProcessing
	log.WinnerOption = winner
	log.SettledAt = settledAt
	log.DeadlineAt = deadlineAt
	log.UpdateTime = now
	if err := tx.Save(log).Error; err != nil {
		return nil, err
	}

	users, winnerHeat, err := s.findWinnerCommentUsers(tx, marketId, winner)
	if err != nil {
		return nil, s.failLog(tx, log, err.Error())
	}
	marketBetTotal := market.PoolA + market.PoolB + market.PoolDraw
	rewardPool := int64(math.Floor(float64(marketBetTotal) * 0.1))
	log.MarketBetTotal = marketBetTotal
	log.RewardPool = rewardPool
	log.WinnerCommentUserCount = int64(len(users))

	if rewardPool <= 0 {
		log.Status = predictCommentRewardStatusPaid
		log.Reason = "empty reward pool"
		log.PaidAt = now
		log.UpdateTime = now
		return log, tx.Save(log).Error
	}
	if len(users) == 0 {
		log.Status = predictCommentRewardStatusPaid
		log.Reason = "no winner comment users"
		log.PaidAt = now
		log.UpdateTime = now
		return log, tx.Save(log).Error
	}
	if winnerHeat <= 0 {
		log.Status = predictCommentRewardStatusPaid
		log.Reason = "empty winner comment heat"
		log.PaidAt = now
		log.UpdateTime = now
		return log, tx.Save(log).Error
	}

	// 兼容旧字段：保留 perUserReward 为均值观测值，实际按热度占比分配。
	log.PerUserReward = rewardPool / int64(len(users))
	if log.PerUserReward < 0 {
		log.PerUserReward = 0
	}
	if log.PerUserReward <= 0 {
		log.Status = predictCommentRewardStatusPaid
		log.Reason = "per user reward is zero"
		log.PaidAt = now
		log.UpdateTime = now
		return log, tx.Save(log).Error
	}
	log.WinnerTotalCommentHeat = winnerHeat
	if err := tx.Save(log).Error; err != nil {
		return nil, err
	}

	allocated := int64(0)
	amountByUser := make(map[int64]int64, len(users))
	for _, u := range users {
		if u.Heat <= 0 {
			amountByUser[u.UserId] = 0
			continue
		}
		amount := int64(math.Floor((u.Heat / winnerHeat) * float64(rewardPool)))
		if amount < 0 {
			amount = 0
		}
		amountByUser[u.UserId] = amount
		allocated += amount
	}
	if allocated < rewardPool {
		remainderUsers := make([]PredictCommentRewardUser, 0, len(users))
		for _, u := range users {
			if u.Heat > 0 {
				remainderUsers = append(remainderUsers, u)
			}
		}
		if len(remainderUsers) > 0 {
			left := rewardPool - allocated
			for i := int64(0); i < left; i++ {
				idx := int(i % int64(len(remainderUsers)))
				target := remainderUsers[idx]
				amountByUser[target.UserId] = amountByUser[target.UserId] + 1
			}
			allocated = rewardPool
		}
	}
	log.Remainder = rewardPool - allocated

	for _, u := range users {
		amount := amountByUser[u.UserId]
		if amount <= 0 {
			continue
		}
		item := &models.PredictCommentRewardItem{}
		err := tx.Where("reward_log_id = ? AND user_id = ?", log.Id, u.UserId).Take(item).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, s.failLog(tx, log, err.Error())
		}
		remark := fmt.Sprintf("predict comment reward: marketId=%d option=%s", marketId, winner)
		_, coinLog, err := UserCoinService.AddReward(tx, u.UserId, PredictCommentRewardBizType, log.Id, amount, remark)
		if err != nil {
			return nil, s.failLog(tx, log, err.Error())
		}
		item = &models.PredictCommentRewardItem{
			RewardLogId:     log.Id,
			MarketId:        marketId,
			UserId:          u.UserId,
			Amount:          amount,
			UserCommentHeat: u.Heat,
			CommentCount:    u.CommentCount,
			FirstCommentId:  u.FirstCommentId,
			CoinLogId:       coinLog.Id,
			CreateTime:      now,
		}
		if err := tx.Create(item).Error; err != nil {
			return nil, s.failLog(tx, log, err.Error())
		}
	}

	log.Status = predictCommentRewardStatusPaid
	log.Reason = ""
	log.PaidAt = now
	log.UpdateTime = now
	if err := tx.Save(log).Error; err != nil {
		return nil, err
	}
	return log, nil
}

func (s *predictCommentRewardService) failLog(tx *gorm.DB, log *models.PredictCommentRewardLog, reason string) error {
	log.Status = predictCommentRewardStatusFailed
	log.Reason = reason
	log.UpdateTime = toUnixSeconds(dates.NowTimestamp())
	return tx.Save(log).Error
}

func (s *predictCommentRewardService) findWinnerCommentUsers(tx *gorm.DB, marketId int64, winner string) ([]PredictCommentRewardUser, float64, error) {
	var users []PredictCommentRewardUser
	options, _, err := TearHeatService.ComputeHeatSnapshot(tx, TearHeatSnapshotComputeRequest{
		EventType:    constants.EntityPredictMarket,
		EventId:      marketId,
		TopicId:      marketId,
		RoundId:      0,
		SnapshotType: TearSnapshotSettle,
		FreezeSource: "ON_DEMAND",
	})
	if err != nil {
		return nil, 0, err
	}
	heatByOption := map[string]float64{}
	for _, item := range options {
		opt, _ := item["option"].(string)
		hTotal, _ := item["hTotal"].(float64)
		heatByOption[strings.ToUpper(strings.TrimSpace(opt))] = hTotal
	}
	winnerHeat := heatByOption[strings.ToUpper(strings.TrimSpace(winner))]
	if winnerHeat <= 0 {
		return nil, 0, nil
	}
	userHeatItems, err := TearHeatService.GetUserHeatItems(tx, constants.EntityPredictMarket, marketId, 0)
	if err != nil {
		return nil, 0, err
	}
	userHeatMap := map[int64]float64{}
	for _, item := range userHeatItems {
		if strings.EqualFold(strings.TrimSpace(item.Option), winner) && item.CommentHeat > 0 {
			userHeatMap[item.UserId] += item.CommentHeat
		}
	}

	err = tx.Table("t_predict_comment_meta AS m").
		Select("m.user_id as user_id, m.option as option, COUNT(1) as comment_count, MIN(m.comment_id) as first_comment_id").
		Joins("JOIN t_comment c ON c.id = m.comment_id").
		Where("m.market_id = ? AND m.option = ?", marketId, winner).
		Where("c.status = ?", constants.StatusOk).
		Group("m.user_id, m.option").
		Order("m.user_id ASC").
		Scan(&users).Error
	if err != nil {
		return nil, 0, err
	}
	if len(users) == 0 {
		return users, winnerHeat, nil
	}
	validUsers := make([]PredictCommentRewardUser, 0, len(users))
	actualWinnerHeat := float64(0)
	for i := range users {
		users[i].Heat = userHeatMap[users[i].UserId]
		if users[i].Heat <= 0 {
			continue
		}
		actualWinnerHeat += users[i].Heat
		validUsers = append(validUsers, users[i])
	}
	if actualWinnerHeat <= 0 {
		return nil, 0, nil
	}
	return validUsers, actualWinnerHeat, nil
}
