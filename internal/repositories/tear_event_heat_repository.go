package repositories

import (
	"bbs-go/internal/models/models"

	"gorm.io/gorm"
)

var TearEventHeatRepository = newTearEventHeatRepository()

func newTearEventHeatRepository() *tearEventHeatRepository {
	return &tearEventHeatRepository{}
}

type tearEventHeatRepository struct{}

func (r *tearEventHeatRepository) Take(db *gorm.DB, eventId, userId int64, option string, businessType int) *models.TearEventHeat {
	ret := &models.TearEventHeat{}
	err := db.Take(ret,
		"event_id = ? AND user_id = ? AND option = ? AND business_type = ?",
		eventId, userId, option, businessType,
	).Error
	if err != nil {
		return nil
	}
	return ret
}

func (r *tearEventHeatRepository) Create(db *gorm.DB, t *models.TearEventHeat) error {
	return db.Create(t).Error
}

func (r *tearEventHeatRepository) Update(db *gorm.DB, t *models.TearEventHeat) error {
	return db.Save(t).Error
}

func (r *tearEventHeatRepository) FindByEvent(db *gorm.DB, eventId int64, businessType int) ([]models.TearEventHeat, error) {
	var list []models.TearEventHeat
	query := db.Where("event_id = ?", eventId)
	if businessType > 0 {
		query = query.Where("business_type = ?", businessType)
	}
	err := query.Order("h_total DESC, id DESC").Find(&list).Error
	return list, err
}

func (r *tearEventHeatRepository) FindByUser(db *gorm.DB, eventId, userId int64, businessType int) ([]models.TearEventHeat, error) {
	var list []models.TearEventHeat
	query := db.Where("event_id = ? AND user_id = ?", eventId, userId)
	if businessType > 0 {
		query = query.Where("business_type = ?", businessType)
	}
	err := query.Order("h_total DESC, id DESC").Find(&list).Error
	return list, err
}
