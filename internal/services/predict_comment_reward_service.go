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
	CommentCount   int64
	FirstCommentId int64
}

func (s *predictCommentRewardService) RunDue() error {
	now := dates.NowTimestamp()
	var markets []models.PredictMarket
	if err := sqls.DB().
		Where("status = ? AND resolved_at > 0 AND resolved_at <= ?", "SETTLED", now).
		Where("resolved_at >= ?", now-3600).
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
	now := dates.NowTimestamp()
	return sqls.DB().Model(&models.PredictCommentRewardLog{}).
		Where("status = ? AND deadline_at > 0 AND deadline_at < ?", predictCommentRewardStatusPending, now).
		Updates(map[string]interface{}{
			"status":      predictCommentRewardStatusExpired,
			"reason":      "deadline expired",
			"update_time": now,
		}).Error
}

func (s *predictCommentRewardService) runForMarketTx(tx *gorm.DB, marketId int64, forceRetry bool) (*models.PredictCommentRewardLog, error) {
	now := dates.NowTimestamp()
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
	settledAt := market.ResolvedAt
	if settledAt <= 0 {
		settledAt = market.UpdateTime
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

	users, err := s.findWinnerCommentUsers(tx, marketId, winner)
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

	perUserReward := rewardPool / int64(len(users))
	remainder := rewardPool - perUserReward*int64(len(users))
	log.PerUserReward = perUserReward
	log.Remainder = remainder
	if perUserReward <= 0 {
		log.Status = predictCommentRewardStatusPaid
		log.Reason = "per user reward is zero"
		log.PaidAt = now
		log.UpdateTime = now
		return log, tx.Save(log).Error
	}
	if err := tx.Save(log).Error; err != nil {
		return nil, err
	}

	for _, u := range users {
		item := &models.PredictCommentRewardItem{}
		err := tx.Where("reward_log_id = ? AND user_id = ?", log.Id, u.UserId).Take(item).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, s.failLog(tx, log, err.Error())
		}
		remark := fmt.Sprintf("predict comment reward: marketId=%d option=%s", marketId, winner)
		_, coinLog, err := UserCoinService.AddReward(tx, u.UserId, PredictCommentRewardBizType, log.Id, perUserReward, remark)
		if err != nil {
			return nil, s.failLog(tx, log, err.Error())
		}
		item = &models.PredictCommentRewardItem{
			RewardLogId:    log.Id,
			MarketId:       marketId,
			UserId:         u.UserId,
			Amount:         perUserReward,
			CommentCount:   u.CommentCount,
			FirstCommentId: u.FirstCommentId,
			CoinLogId:      coinLog.Id,
			CreateTime:     now,
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
	log.UpdateTime = dates.NowTimestamp()
	return tx.Save(log).Error
}

func (s *predictCommentRewardService) findWinnerCommentUsers(tx *gorm.DB, marketId int64, winner string) ([]PredictCommentRewardUser, error) {
	var users []PredictCommentRewardUser
	err := tx.Table("t_predict_comment_meta AS m").
		Select("m.user_id as user_id, COUNT(1) as comment_count, MIN(m.comment_id) as first_comment_id").
		Joins("JOIN t_comment c ON c.id = m.comment_id").
		Where("m.market_id = ? AND m.option = ?", marketId, winner).
		Where("c.status = ?", constants.StatusOk).
		Group("m.user_id").
		Order("m.user_id ASC").
		Scan(&users).Error
	return users, err
}
