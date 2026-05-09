package migrations

import (
	"bbs-go/internal/models/models"
	"log/slog"

	"github.com/mlogclub/simple/sqls"
)

func migrate_ai_stamina_tables() error {
	if err := sqls.DB().AutoMigrate(
		&models.UserAIStamina{},
		&models.UserAIStaminaLog{},
	); err != nil {
		slog.Error("migrate ai stamina tables failed", "error", err)
		return err
	}
	return nil
}
