package migrations

import (
	"bbs-go/internal/models/models"
	"log/slog"

	"github.com/mlogclub/simple/sqls"
)

// v14: pet gacha pool config
func migrate_pet_gacha_pool_config_init_tables() error {
	db := sqls.DB()
	if err := db.AutoMigrate(&models.GachaPoolConfig{}); err != nil {
		slog.Error("migrate pet gacha pool config tables failed", "error", err)
		return err
	}
	return nil
}
