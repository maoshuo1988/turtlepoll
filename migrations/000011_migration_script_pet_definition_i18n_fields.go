package migrations

import (
	"github.com/mlogclub/simple/sqls"
)

func migrate_pet_definition_i18n_fields() error {
	db := sqls.DB()

	if err := db.Exec("ALTER TABLE t_pet_definition ADD COLUMN IF NOT EXISTS name_json TEXT DEFAULT ''").Error; err != nil {
		return err
	}
	if err := db.Exec("ALTER TABLE t_pet_definition ADD COLUMN IF NOT EXISTS description_json TEXT DEFAULT ''").Error; err != nil {
		return err
	}

	// backfill: if json empty and plain not empty, wrap as zh-CN
	if err := db.Exec("UPDATE t_pet_definition SET name_json = '{\"zh-CN\":' || to_json(name)::text || '}' WHERE (name_json IS NULL OR name_json = '') AND (name IS NOT NULL AND name <> '')").Error; err != nil {
		return err
	}
	if err := db.Exec("UPDATE t_pet_definition SET description_json = '{\"zh-CN\":' || to_json(description)::text || '}' WHERE (description_json IS NULL OR description_json = '') AND (description IS NOT NULL AND description <> '')").Error; err != nil {
		return err
	}

	return nil
}
