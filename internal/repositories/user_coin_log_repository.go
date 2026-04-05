package repositories

import (
	"bbs-go/internal/models/models"

	"gorm.io/gorm"
)

var UserCoinLogRepository = newUserCoinLogRepository()

func newUserCoinLogRepository() *userCoinLogRepository {
	return &userCoinLogRepository{}
}

type userCoinLogRepository struct{}

func (r *userCoinLogRepository) Create(db *gorm.DB, t *models.UserCoinLog) error {
	return db.Create(t).Error
}
