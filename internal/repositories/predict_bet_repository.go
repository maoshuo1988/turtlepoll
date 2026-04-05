package repositories

import (
	"bbs-go/internal/models/models"

	"gorm.io/gorm"
)

var PredictBetRepository = newPredictBetRepository()

func newPredictBetRepository() *predictBetRepository {
	return &predictBetRepository{}
}

type predictBetRepository struct{}

func (r *predictBetRepository) Create(db *gorm.DB, t *models.PredictBet) error {
	return db.Create(t).Error
}
