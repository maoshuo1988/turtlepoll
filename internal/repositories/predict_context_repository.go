package repositories

import (
	"bbs-go/internal/models"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

var PredictContextRepository = newPredictContextRepository()

func newPredictContextRepository() *predictContextRepository {
	return &predictContextRepository{}
}

type predictContextRepository struct {
}

func (r *predictContextRepository) FindOne(db *gorm.DB, cnd *sqls.Cnd) *models.PredictContext {
	ret := &models.PredictContext{}
	if err := cnd.FindOne(db, ret); err != nil {
		return nil
	}
	return ret
}

func (r *predictContextRepository) Take(db *gorm.DB, where ...interface{}) *models.PredictContext {
	ret := &models.PredictContext{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *predictContextRepository) Create(db *gorm.DB, t *models.PredictContext) (err error) {
	err = db.Create(t).Error
	return
}

func (r *predictContextRepository) Update(db *gorm.DB, t *models.PredictContext) (err error) {
	err = db.Save(t).Error
	return
}
