package repositories

import (
	"bbs-go/internal/models/models"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

type featureCatalogRepository struct{}

var FeatureCatalogRepository = new(featureCatalogRepository)

func (r *featureCatalogRepository) Get(db *gorm.DB, id int64) *models.FeatureCatalogItem {
	ret := &models.FeatureCatalogItem{}
	if err := db.First(ret, id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *featureCatalogRepository) GetByFeatureKey(db *gorm.DB, featureKey string) *models.FeatureCatalogItem {
	ret := &models.FeatureCatalogItem{}
	if err := db.Where("feature_key = ?", featureKey).First(ret).Error; err != nil {
		return nil
	}
	return ret
}

func (r *featureCatalogRepository) Take(db *gorm.DB, where ...interface{}) *models.FeatureCatalogItem {
	ret := &models.FeatureCatalogItem{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *featureCatalogRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.FeatureCatalogItem) {
	cnd.Find(db, &list)
	return
}

func (r *featureCatalogRepository) FindOne(db *gorm.DB, cnd *sqls.Cnd) *models.FeatureCatalogItem {
	ret := &models.FeatureCatalogItem{}
	if err := cnd.FindOne(db, &ret); err != nil {
		return nil
	}
	return ret
}

func (r *featureCatalogRepository) FindPageByCnd(db *gorm.DB, cnd *sqls.Cnd) (list []models.FeatureCatalogItem, paging *sqls.Paging) {
	cnd.Find(db, &list)
	count := cnd.Count(db, &models.FeatureCatalogItem{})

	paging = &sqls.Paging{
		Page:  cnd.Paging.Page,
		Limit: cnd.Paging.Limit,
		Total: count,
	}
	return
}

func (r *featureCatalogRepository) Create(db *gorm.DB, t *models.FeatureCatalogItem) error {
	return db.Create(t).Error
}

func (r *featureCatalogRepository) Update(db *gorm.DB, t *models.FeatureCatalogItem) error {
	return db.Save(t).Error
}

func (r *featureCatalogRepository) DeleteByFeatureKey(db *gorm.DB, featureKey string) error {
	return db.Where("feature_key = ?", featureKey).Delete(&models.FeatureCatalogItem{}).Error
}
