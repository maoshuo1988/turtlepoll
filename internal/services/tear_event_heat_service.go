package services

import (
	"bbs-go/internal/models/models"
	"bbs-go/internal/repositories"
	"strings"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

var TearEventHeatService = newTearEventHeatService()

func newTearEventHeatService() *tearEventHeatService {
	return &tearEventHeatService{}
}

type tearEventHeatService struct{}

func (s *tearEventHeatService) AddHeat(tx *gorm.DB, eventId, userId int64, option string, businessType int, hLike, hComment, hCoin float64) error {
	if tx == nil {
		tx = sqls.DB()
	}
	if eventId <= 0 || userId <= 0 {
		return nil
	}
	option = strings.ToUpper(strings.TrimSpace(option))
	if option == "" {
		return nil
	}
	if businessType <= 0 {
		businessType = TearBusinessTypeDark
	}
	if hLike == 0 && hComment == 0 && hCoin == 0 {
		return nil
	}

	now := dates.NowTimestamp()
	eventHeat := repositories.TearEventHeatRepository.Take(tx, eventId, userId, option, businessType)
	if eventHeat == nil {
		eventHeat = &models.TearEventHeat{
			EventId:      eventId,
			UserId:       userId,
			Option:       option,
			BusinessType: businessType,
			SnapshotType: tearHeatLive,
			CreateTime:   now,
		}
	}
	eventHeat.HLike += hLike
	eventHeat.HComment += hComment
	eventHeat.HCoin += hCoin
	eventHeat.HTotal = eventHeat.HLike + eventHeat.HComment + eventHeat.HCoin
	eventHeat.SnapshotTime = now
	eventHeat.SnapshotType = tearHeatLive
	eventHeat.UpdateTime = now
	if eventHeat.Id > 0 {
		return repositories.TearEventHeatRepository.Update(tx, eventHeat)
	}
	return repositories.TearEventHeatRepository.Create(tx, eventHeat)
}

func (s *tearEventHeatService) SetHeat(tx *gorm.DB, eventId, userId int64, option string, businessType int, hLike, hComment, hCoin float64) error {
	if tx == nil {
		tx = sqls.DB()
	}
	if eventId <= 0 || userId <= 0 {
		return nil
	}
	option = strings.ToUpper(strings.TrimSpace(option))
	if option == "" {
		return nil
	}
	if businessType <= 0 {
		businessType = TearBusinessTypeDark
	}

	now := dates.NowTimestamp()
	eventHeat := repositories.TearEventHeatRepository.Take(tx, eventId, userId, option, businessType)
	if eventHeat == nil {
		eventHeat = &models.TearEventHeat{
			EventId:      eventId,
			UserId:       userId,
			Option:       option,
			BusinessType: businessType,
			SnapshotType: tearHeatLive,
			CreateTime:   now,
		}
	}
	eventHeat.HLike = hLike
	eventHeat.HComment = hComment
	eventHeat.HCoin = hCoin
	eventHeat.HTotal = hLike + hComment + hCoin
	eventHeat.SnapshotTime = now
	eventHeat.SnapshotType = tearHeatLive
	eventHeat.UpdateTime = now
	if eventHeat.Id > 0 {
		return repositories.TearEventHeatRepository.Update(tx, eventHeat)
	}
	return repositories.TearEventHeatRepository.Create(tx, eventHeat)
}
