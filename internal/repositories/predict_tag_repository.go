package repositories

import (
	"bbs-go/internal/models"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

var PredictTagRepository = newPredictTagRepository()

func newPredictTagRepository() *predictTagRepository {
	return &predictTagRepository{}
}

type predictTagRepository struct{}

func (r *predictTagRepository) DB() *gorm.DB {
	return sqls.DB()
}

func (r *predictTagRepository) Take(db *gorm.DB, where ...interface{}) *models.PredictTag {
	ret := &models.PredictTag{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *predictTagRepository) Create(db *gorm.DB, t *models.PredictTag) error {
	return db.Create(t).Error
}

func (r *predictTagRepository) Update(db *gorm.DB, t *models.PredictTag) error {
	return db.Save(t).Error
}

func (r *predictTagRepository) UpsertBySlug(db *gorm.DB, slug string, updates map[string]any) error {
	return db.Model(&models.PredictTag{}).Where("slug = ?", slug).Updates(updates).Error
}
