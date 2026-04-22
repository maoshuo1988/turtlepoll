package services

import (
	"bbs-go/internal/models/models"
	"bbs-go/internal/repositories"
	"errors"
	"log/slog"
	"strings"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/common/jsons"
	"github.com/mlogclub/simple/sqls"
)

var FeatureCatalogService = newFeatureCatalogService()

func newFeatureCatalogService() *featureCatalogService {
	return &featureCatalogService{}
}

type featureCatalogService struct{}

// EnsureDefaultSeeds 会在服务启动时调用，用于把内置特性模板（FeatureCatalog）幂等写入数据库。
//
// 设计目标：
// - 避免新环境启动后 FeatureCatalog 为空导致业务（例如每日登录加成）无法执行。
// - 幂等：重复启动不会重复插入。
// - 兼容运营改动：默认策略只在“记录不存在”时创建；若你希望强制覆盖某些字段，可按需启用 Update。
func (s *featureCatalogService) EnsureDefaultSeeds() {
	seeds := DefaultFeatureCatalogSeeds()
	db := sqls.DB()
	for _, seed := range seeds {
		seed := seed
		if strings.TrimSpace(seed.FeatureKey) == "" {
			continue
		}
		exists := repositories.FeatureCatalogRepository.GetByFeatureKey(db, seed.FeatureKey)
		if exists == nil {
			if err := s.Create(&seed); err != nil {
				slog.Error("seed feature_catalog create failed", slog.Any("err", err), slog.String("featureKey", seed.FeatureKey))
			}
			continue
		}
		// 默认不覆盖运营已维护的数据。
		// 如需“随代码升级自动更新 Schema/名称/开关”，可以在这里做字段对比后 Save。
	}
}

// DefaultFeatureCatalogSeeds 内置特性模板种子数据。
//
// 这里的 NameJSON/ParamsSchemaJSON/MetadataJSON 使用 JSON 文本，便于直接存储。
func DefaultFeatureCatalogSeeds() []models.FeatureCatalogItem {
	return []models.FeatureCatalogItem{
		{
			FeatureKey:       "signin_bonus",
			NameJSON:         `{"zh-CN":"每日登录加成","en-US":"Signin Bonus"}`,
			Scope:            "PET_DEF",
			EffectiveEvent:   "DAILY_SIGNIN",
			ParamsSchemaJSON: `{"type":"object","required":["enabled","base_amount","level_step","daily_cap"],"properties":{"enabled":{"type":"boolean"},"base_amount":{"type":"integer","minimum":0},"level_step":{"type":"integer","minimum":0},"daily_cap":{"type":"integer","minimum":0}}}`,
			Enabled:          true,
			MetadataJSON:     `{}`,
		},
		{
			FeatureKey:       "spark_multiplier",
			NameJSON:         `{"zh-CN":"火花倍率","en-US":"Spark Multiplier"}`,
			Scope:            "PET_DEF",
			EffectiveEvent:   "DAILY_SIGNIN",
			ParamsSchemaJSON: `{"type":"object","required":["enabled","base","per_level","cap"],"properties":{"enabled":{"type":"boolean"},"base":{"type":"number","minimum":0},"per_level":{"type":"number","minimum":0},"cap":{"type":"integer","minimum":0}}}`,
			Enabled:          true,
			MetadataJSON:     `{}`,
		},
		{
			FeatureKey:       "debt",
			NameJSON:         `{"zh-CN":"欠账能力","en-US":"Debt Ability"}`,
			Scope:            "PET_DEF",
			EffectiveEvent:   "BALANCE_CHANGE",
			ParamsSchemaJSON: `{"type":"object","required":["enabled","debtFloor","forbidEquipWhenDebt"],"properties":{"enabled":{"type":"boolean"},"debtFloor":{"type":"integer","maximum":0},"forbidEquipWhenDebt":{"type":"boolean"},"errorCode":{"type":"string"}}}`,
			Enabled:          true,
			MetadataJSON:     `{}`,
		},
		{
			FeatureKey:       "debt_subsidy",
			NameJSON:         `{"zh-CN":"欠款补贴","en-US":"Debt Subsidy"}`,
			Scope:            "PET_DEF",
			EffectiveEvent:   "DAILY_SIGNIN",
			ParamsSchemaJSON: `{"type":"object","required":["enabled","subsidyRate"],"properties":{"enabled":{"type":"boolean"},"subsidyRate":{"type":"number","minimum":0,"maximum":1},"capPerDay":{"type":"integer","minimum":0}}}`,
			Enabled:          true,
			MetadataJSON:     `{}`,
		},
		{
			FeatureKey:       "deposit_interest",
			NameJSON:         `{"zh-CN":"存款生息","en-US":"Deposit Interest"}`,
			Scope:            "PET_DEF",
			EffectiveEvent:   "DAILY_SIGNIN",
			ParamsSchemaJSON: `{"type":"object","required":["enabled","interestRate"],"properties":{"enabled":{"type":"boolean"},"interestRate":{"type":"number","minimum":0,"maximum":1},"capPerDay":{"type":"integer","minimum":0}}}`,
			Enabled:          true,
			MetadataJSON:     `{}`,
		},
		{
			FeatureKey:       "first_bet_bonus",
			NameJSON:         `{"zh-CN":"首次下注奖励","en-US":"First Bet Bonus"}`,
			Scope:            "PET_DEF",
			EffectiveEvent:   "BET_PLACED",
			ParamsSchemaJSON: `{"type":"object","required":["enabled","amount"],"properties":{"enabled":{"type":"boolean"},"amount":{"type":"integer","minimum":0}}}`,
			Enabled:          true,
			MetadataJSON:     `{}`,
		},
	}
}

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
	s.fillLegacyName(t)
	now := dates.NowTimestamp()
	if t.CreateTime <= 0 {
		t.CreateTime = now
	}
	t.UpdateTime = now
	return repositories.FeatureCatalogRepository.Create(sqls.DB(), t)
}

func (s *featureCatalogService) Update(t *models.FeatureCatalogItem) error {
	s.fillLegacyName(t)
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

func (s *featureCatalogService) fillLegacyName(t *models.FeatureCatalogItem) {
	if t == nil || strings.TrimSpace(t.Name) != "" {
		return
	}
	if strings.TrimSpace(t.NameJSON) == "" {
		t.Name = t.FeatureKey
		return
	}
	var m map[string]string
	if err := jsons.Parse(t.NameJSON, &m); err != nil || len(m) == 0 {
		t.Name = t.FeatureKey
		return
	}
	if v := strings.TrimSpace(m["zh-CN"]); v != "" {
		t.Name = v
		return
	}
	if v := strings.TrimSpace(m["en-US"]); v != "" {
		t.Name = v
		return
	}
	for _, v := range m {
		v = strings.TrimSpace(v)
		if v != "" {
			t.Name = v
			return
		}
	}
	t.Name = t.FeatureKey
}
