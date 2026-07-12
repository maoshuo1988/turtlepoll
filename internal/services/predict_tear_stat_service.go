package services

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/models"
	"errors"
	"strings"

	"github.com/mlogclub/simple/common/dates"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var PredictTearStatService = newPredictTearStatService()

func newPredictTearStatService() *predictTearStatService {
	return &predictTearStatService{}
}

type predictTearStatService struct{}

func (s *predictTearStatService) RecordInteraction(tx *gorm.DB, marketId, userId int64, option, actionType, entityType string, entityId int64, heat float64, requestId string) error {
	if tx == nil {
		return errors.New("tx is required")
	}
	if marketId <= 0 || userId <= 0 || entityId <= 0 {
		return errors.New("marketId/userId/entityId are required")
	}
	actionType = strings.ToLower(strings.TrimSpace(actionType))
	if actionType == "" {
		return errors.New("actionType is required")
	}
	option = strings.ToUpper(strings.TrimSpace(option))
	if option == "" {
		return errors.New("option is required")
	}
	now := dates.NowTimestamp()
	log := &models.TearInteractLog{
		EventType:        constants.EntityPredictMarket,
		EventId:          marketId,
		TopicId:          marketId,
		RoundId:          0,
		UserId:           userId,
		OptionAtAction:   option,
		ActionType:       actionType,
		EntityType:       entityType,
		EntityId:         entityId,
		HeatContribution: heat,
		RequestId:        requestId,
		CreateTime:       now,
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(log).Error
}
