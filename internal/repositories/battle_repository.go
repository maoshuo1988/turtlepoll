package repositories

import (
	"bbs-go/internal/models/models"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var BattleRepository = newBattleRepository()

func newBattleRepository() *battleRepository {
	return &battleRepository{}
}

type battleRepository struct{}

func (r *battleRepository) Take(db *gorm.DB, where ...interface{}) *models.Battle {
	ret := &models.Battle{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *battleRepository) TakeForUpdate(db *gorm.DB, battleId int64) (*models.Battle, error) {
	ret := &models.Battle{}
	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Take(ret, "id = ?", battleId).Error; err != nil {
		return nil, err
	}
	return ret, nil
}

func (r *battleRepository) Create(db *gorm.DB, t *models.Battle) error {
	return db.Create(t).Error
}

func (r *battleRepository) Update(db *gorm.DB, t *models.Battle) error {
	return db.Save(t).Error
}

func (r *battleRepository) DB() *gorm.DB {
	return sqls.DB()
}
