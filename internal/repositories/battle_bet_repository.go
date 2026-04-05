package repositories

import (
	"bbs-go/internal/models/models"

	"gorm.io/gorm"
)

var BattleBetRepository = newBattleBetRepository()

func newBattleBetRepository() *battleBetRepository {
	return &battleBetRepository{}
}

type battleBetRepository struct{}

func (r *battleBetRepository) Create(db *gorm.DB, t *models.BattleBet) error {
	return db.Create(t).Error
}

func (r *battleBetRepository) TakeByUserRequest(db *gorm.DB, battleId, userId int64, requestId string) *models.BattleBet {
	ret := &models.BattleBet{}
	if err := db.Take(ret, "battle_id = ? AND user_id = ? AND request_id = ?", battleId, userId, requestId).Error; err != nil {
		return nil
	}
	return ret
}

func (r *battleBetRepository) SumUserStake(db *gorm.DB, battleId, userId int64) (int64, error) {
	var sum int64
	err := db.Model(&models.BattleBet{}).
		Select("COALESCE(SUM(amount),0)").
		Where("battle_id = ? AND user_id = ?", battleId, userId).
		Scan(&sum).Error
	return sum, err
}

func (r *battleBetRepository) SumChallengerStake(db *gorm.DB, battleId int64) (int64, error) {
	var sum int64
	err := db.Model(&models.BattleBet{}).
		Select("COALESCE(SUM(amount),0)").
		Where("battle_id = ? AND role = ?", battleId, "challenger").
		Scan(&sum).Error
	return sum, err
}
