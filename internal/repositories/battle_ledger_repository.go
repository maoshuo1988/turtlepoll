package repositories

import (
	"bbs-go/internal/models/models"

	"gorm.io/gorm"
)

var BattleLedgerRepository = newBattleLedgerRepository()

func newBattleLedgerRepository() *battleLedgerRepository {
	return &battleLedgerRepository{}
}

type battleLedgerRepository struct{}

func (r *battleLedgerRepository) Create(db *gorm.DB, t *models.BattleLedger) error {
	return db.Create(t).Error
}

func (r *battleLedgerRepository) TakeIdempotent(db *gorm.DB, battleId, userId int64, action, requestId string) *models.BattleLedger {
	ret := &models.BattleLedger{}
	if err := db.Take(ret, "battle_id = ? AND user_id = ? AND action = ? AND request_id = ?", battleId, userId, action, requestId).Error; err != nil {
		return nil
	}
	return ret
}
