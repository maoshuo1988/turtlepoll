package services

import (
	"bbs-go/internal/models"
	"bbs-go/internal/repositories"
	"errors"
	"strings"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
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
