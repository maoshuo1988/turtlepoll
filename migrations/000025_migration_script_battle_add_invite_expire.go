package migrations

import "github.com/mlogclub/simple/sqls"

func migrate_battle_add_invite_expire() error {
	db := sqls.DB()
	if err := db.Exec(`ALTER TABLE t_battle ADD COLUMN IF NOT EXISTS invite_code_expire_at bigint NOT NULL DEFAULT 0`).Error; err != nil {
		return err
	}
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_t_battle_invite_code_expire_at ON t_battle(invite_code_expire_at)`).Error; err != nil {
		return err
	}
	return nil
}
