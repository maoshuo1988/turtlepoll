package services

import (
	"bbs-go/internal/models"
	"bbs-go/internal/repositories"
	"errors"
	"strings"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

var PredictContextService = newPredictContextService()

func newPredictContextService() *predictContextService {
	return &predictContextService{}
}

type predictContextService struct{}

func (s *predictContextService) GetByMarketId(marketId int64) *models.PredictContext {
	if marketId <= 0 {
		return nil
	}
	return repositories.PredictContextRepository.Take(sqls.DB(), "market_id = ?", marketId)
}

// IncrHeatByMarketId 如果 marketId 对应的 PredictContext 存在，则给它的 heat 增加 delta。
// 说明：
// - 热榜接口按 PredictContext.heat 排序，因此评论等行为需要同步该字段。
// - 若 PredictContext 不存在（例如未同步/未创建预测上下文），这里选择“跳过不报错”，避免影响主流程。
func (s *predictContextService) IncrHeatByMarketId(tx *gorm.DB, marketId int64, delta int64) error {
	if marketId <= 0 || delta == 0 {
		return nil
	}
	if tx == nil {
		tx = sqls.DB()
	}

	res := tx.Model(&models.PredictContext{}).
		Where("market_id = ?", marketId).
		UpdateColumn("heat", gorm.Expr("heat + ?", delta))
	if res.Error != nil {
		return res.Error
	}
	return nil
}

// UpsertByMarketId 按 marketId 创建或更新 PredictContext。
// 约束：eventName 必填（与模型 not null 对齐）。
func (s *predictContextService) UpsertByMarketId(ctx *models.PredictContext) (*models.PredictContext, error) {
	if ctx == nil {
		return nil, nil
	}
	if ctx.MarketId <= 0 {
		return nil, errors.New("marketId is required")
	}
	ctx.EventName = strings.TrimSpace(ctx.EventName)
	if ctx.EventName == "" {
		return nil, errors.New("eventName is required")
	}

	now := dates.NowTimestamp()
	db := sqls.DB()

	exists := repositories.PredictContextRepository.Take(db, "market_id = ?", ctx.MarketId)
	if exists == nil {
		ctx.CreateTime = now
		ctx.UpdateTime = now
		if err := repositories.PredictContextRepository.Create(db, ctx); err != nil {
			return nil, err
		}
		return ctx, nil
	}

	// 只允许更新业务展示字段；主键/marketId 不允许改
	exists.EventName = ctx.EventName
	exists.ImageUrl = ctx.ImageUrl
	exists.ParticipantCount = ctx.ParticipantCount
	exists.ProText = ctx.ProText
	exists.ProVoteCount = ctx.ProVoteCount
	exists.ConText = ctx.ConText
	exists.ConVoteCount = ctx.ConVoteCount
	exists.Heat = ctx.Heat
	exists.Detail = ctx.Detail
	exists.Tags = ctx.Tags
	exists.UpdateTime = now

	if err := repositories.PredictContextRepository.Update(db, exists); err != nil {
		return nil, err
	}
	return exists, nil
}
