package repositories

import (
	"bbs-go/internal/models/models"

	"gorm.io/gorm"
)

var BattleChallengeActionRepository = newBattleChallengeActionRepository()

func newBattleChallengeActionRepository() *battleChallengeActionRepository {
	return &battleChallengeActionRepository{}
}

type battleChallengeActionRepository struct{}

func (r *battleChallengeActionRepository) Take(db *gorm.DB, where ...interface{}) *models.BattleChallengeAction {
	ret := &models.BattleChallengeAction{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *battleChallengeActionRepository) TakeByRequest(db *gorm.DB, battleId, userId int64, requestId string) *models.BattleChallengeAction {
	return r.Take(db, "battle_id = ? AND user_id = ? AND request_id = ?", battleId, userId, requestId)
}

func (r *battleChallengeActionRepository) TakeByUser(db *gorm.DB, battleId, userId int64) *models.BattleChallengeAction {
	return r.Take(db, "battle_id = ? AND user_id = ?", battleId, userId)
}

func (r *battleChallengeActionRepository) Create(db *gorm.DB, t *models.BattleChallengeAction) error {
	return db.Create(t).Error
}
