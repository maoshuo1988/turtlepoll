package migrations

import (
	"bbs-go/internal/models/models"
	"fmt"

	"github.com/mlogclub/simple/sqls"
)

func migrate_feature_catalog_add_name_json() error {
	db := sqls.DB()
	m := db.Migrator()
	if !m.HasColumn(&models.FeatureCatalogItem{}, "name_json") {
		if err := m.AddColumn(&models.FeatureCatalogItem{}, "name_json"); err != nil {
			return fmt.Errorf("add column name_json: %w", err)
		}
	}
	// Backfill from legacy `name` column if it exists.
	if m.HasColumn(&models.FeatureCatalogItem{}, "name") {
		_ = db.Exec("UPDATE t_feature_catalog_item SET name_json = COALESCE(NULLIF(name_json, ''), CASE WHEN name IS NULL OR name = '' THEN '{}' ELSE ('{\\\"zh-CN\\\":' || to_json(name)::text || '}') END)").Error
	}
	return nil
}
