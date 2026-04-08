package services

import (
	"bbs-go/internal/models/models"
	"bbs-go/internal/repositories"
	"errors"
	"strings"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
)

var FeatureCatalogService = newFeatureCatalogService()

func newFeatureCatalogService() *featureCatalogService {
	return &featureCatalogService{}
}

type featureCatalogService struct{}

func (s *featureCatalogService) GetByFeatureKey(featureKey string) *models.FeatureCatalogItem {
	featureKey = strings.TrimSpace(featureKey)
	if featureKey == "" {
		return nil
	}
	return repositories.FeatureCatalogRepository.GetByFeatureKey(sqls.DB(), featureKey)
}

func (s *featureCatalogService) FindPageByCnd(cnd *sqls.Cnd) (list []models.FeatureCatalogItem, paging *sqls.Paging) {
	return repositories.FeatureCatalogRepository.FindPageByCnd(sqls.DB(), cnd)
}

func (s *featureCatalogService) Create(t *models.FeatureCatalogItem) error {
	if strings.TrimSpace(t.FeatureKey) == "" {
		return errors.New("featureKey is empty")
	}
	now := dates.NowTimestamp()
	if t.CreateTime <= 0 {
		t.CreateTime = now
	}
	t.UpdateTime = now
	return repositories.FeatureCatalogRepository.Create(sqls.DB(), t)
}

func (s *featureCatalogService) Update(t *models.FeatureCatalogItem) error {
	t.UpdateTime = dates.NowTimestamp()
	return repositories.FeatureCatalogRepository.Update(sqls.DB(), t)
}

func (s *featureCatalogService) DeleteByFeatureKey(featureKey string) error {
	featureKey = strings.TrimSpace(featureKey)
	if featureKey == "" {
		return errors.New("featureKey is empty")
	}
	return repositories.FeatureCatalogRepository.DeleteByFeatureKey(sqls.DB(), featureKey)
}
