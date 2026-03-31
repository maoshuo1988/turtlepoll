package repositories

import (
	"bbs-go/internal/models"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

var PredictTagStatRepository = newPredictTagStatRepository()

func newPredictTagStatRepository() *predictTagStatRepository {
	return &predictTagStatRepository{}
}

type predictTagStatRepository struct{}

func (r *predictTagStatRepository) DB() *gorm.DB {
	return sqls.DB()
}

func (r *predictTagStatRepository) TakeByTagId(db *gorm.DB, tagId int64) *models.PredictTagStat {
	ret := &models.PredictTagStat{}
	if err := db.Take(ret, "tag_id = ?", tagId).Error; err != nil {
		return nil
	}
	return ret
}

func (r *predictTagStatRepository) Create(db *gorm.DB, t *models.PredictTagStat) error {
	return db.Create(t).Error
}

func (r *predictTagStatRepository) Update(db *gorm.DB, t *models.PredictTagStat) error {
	return db.Save(t).Error
}
