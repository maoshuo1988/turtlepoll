package repositories

import (
	"bbs-go/internal/models/models"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var PKRepository = newPKRepository()

func newPKRepository() *pkRepository {
	return &pkRepository{}
}

type pkRepository struct{}

func (r *pkRepository) DB() *gorm.DB {
	return sqls.DB()
}

func (r *pkRepository) TakeTopic(db *gorm.DB, where ...interface{}) *models.PKTopic {
	ret := &models.PKTopic{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *pkRepository) TakeRound(db *gorm.DB, where ...interface{}) *models.PKRound {
	ret := &models.PKRound{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *pkRepository) TakeRoundForUpdate(db *gorm.DB, roundId int64) (*models.PKRound, error) {
	ret := &models.PKRound{}
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Take(ret, "id = ?", roundId).Error
	return ret, err
}

func (r *pkRepository) TakeSeason(db *gorm.DB, where ...interface{}) *models.PKSeason {
	ret := &models.PKSeason{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *pkRepository) TakeBet(db *gorm.DB, where ...interface{}) *models.PKBet {
	ret := &models.PKBet{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *pkRepository) TakeCommentMeta(db *gorm.DB, where ...interface{}) *models.PKCommentMeta {
	ret := &models.PKCommentMeta{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *pkRepository) CreateTopic(db *gorm.DB, t *models.PKTopic) error {
	return db.Create(t).Error
}

func (r *pkRepository) CreateSeason(db *gorm.DB, t *models.PKSeason) error {
	return db.Create(t).Error
}

func (r *pkRepository) CreateRound(db *gorm.DB, t *models.PKRound) error {
	return db.Create(t).Error
}

func (r *pkRepository) CreateBet(db *gorm.DB, t *models.PKBet) error {
	return db.Create(t).Error
}

func (r *pkRepository) CreateCommentMeta(db *gorm.DB, t *models.PKCommentMeta) error {
	return db.Create(t).Error
}

func (r *pkRepository) CreateAction(db *gorm.DB, t *models.PKAction) error {
	return db.Create(t).Error
}

func (r *pkRepository) UpdateTopic(db *gorm.DB, t *models.PKTopic) error {
	return db.Save(t).Error
}

func (r *pkRepository) UpdateSeason(db *gorm.DB, t *models.PKSeason) error {
	return db.Save(t).Error
}

func (r *pkRepository) UpdateRound(db *gorm.DB, t *models.PKRound) error {
	return db.Save(t).Error
}

func (r *pkRepository) UpdateBet(db *gorm.DB, t *models.PKBet) error {
	return db.Save(t).Error
}

func (r *pkRepository) UpdateCommentMeta(db *gorm.DB, t *models.PKCommentMeta) error {
	return db.Save(t).Error
}
