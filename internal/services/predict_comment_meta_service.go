package services

import (
	"bbs-go/internal/models/models"
	"errors"
	"strings"

	"github.com/mlogclub/simple/common/dates"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var PredictCommentMetaService = newPredictCommentMetaService()

func newPredictCommentMetaService() *predictCommentMetaService {
	return &predictCommentMetaService{}
}

type predictCommentMetaService struct{}

func (s *predictCommentMetaService) CreateForComment(tx *gorm.DB, market *models.PredictMarket, comment *models.Comment, option string) error {
	if tx == nil {
		return errors.New("tx is required")
	}
	if market == nil || market.Id <= 0 {
		return errors.New("market is required")
	}
	if comment == nil || comment.Id <= 0 {
		return errors.New("comment is required")
	}
	option = strings.ToUpper(strings.TrimSpace(option))
	if !IsValidPredictOption(market.MarketType, option) {
		return errors.New(PredictOptionErrMsg(market.MarketType))
	}

	now := dates.NowTimestamp()
	meta := &models.PredictCommentMeta{
		CommentId:  comment.Id,
		MarketId:   market.Id,
		UserId:     comment.UserId,
		Option:     option,
		CreateTime: now,
		UpdateTime: now,
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "comment_id"}},
		DoNothing: true,
	}).Create(meta).Error
}
