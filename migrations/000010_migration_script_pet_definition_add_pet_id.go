package migrations

import (
	"database/sql"
	"fmt"

	"github.com/mlogclub/simple/sqls"
)

// migrate_pet_definition_add_pet_id
// 1) add column pet_id (if not exists)
// 2) backfill pet_id from pet_key (best-effort)
// 3) create unique index on pet_id
func migrate_pet_definition_add_pet_id() error {
	db := sqls.DB()

	// 1) add column
	if err := db.Exec("ALTER TABLE t_pet_definition ADD COLUMN IF NOT EXISTS pet_id VARCHAR(64) DEFAULT ''").Error; err != nil {
		return err
	}

	// 2) backfill (only when empty)
	if err := db.Exec("UPDATE t_pet_definition SET pet_id = pet_key WHERE (pet_id IS NULL OR pet_id = '') AND (pet_key IS NOT NULL AND pet_key <> '')").Error; err != nil {
		return err
	}

	// If still empty, we must fail before the unique index is created, otherwise index build will fail later.
	// We keep this conservative: if any rows still have empty pet_id, return an error with a clear message.
	var cnt int64
	row := db.Raw("SELECT COUNT(1) FROM t_pet_definition WHERE pet_id IS NULL OR pet_id = ''").Row()
	if err := row.Scan(&cnt); err != nil {
		// gorm Row().Scan returns stdlib sql errors
		if err == sql.ErrNoRows {
			cnt = 0
		} else {
			return err
		}
	}
	if cnt > 0 {
		return fmt.Errorf("t_pet_definition has %d rows with empty pet_id; please backfill them before enabling unique index", cnt)
	}

	// 3) unique index (if not exists)
	// create concurrently isn't supported inside a transaction; migrations framework doesn't enforce tx, but keep it simple.
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_t_pet_definition_pet_id ON t_pet_definition(pet_id)").Error; err != nil {
		return err
	}

	return nil
}
