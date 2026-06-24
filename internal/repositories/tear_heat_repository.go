package repositories

import (
	"bbs-go/internal/models/models"

	"gorm.io/gorm"
)

var TearHeatRepository = newTearHeatRepository()

func newTearHeatRepository() *tearHeatRepository {
	return &tearHeatRepository{}
}

type tearHeatRepository struct{}

func (r *tearHeatRepository) CreateSnapshot(db *gorm.DB, t *models.TearHeatSnapshot) error {
	return db.Create(t).Error
}
