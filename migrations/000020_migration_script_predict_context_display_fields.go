package migrations

import (
	"bbs-go/internal/models/models"
	"log/slog"

	"github.com/mlogclub/simple/sqls"
)

func migrate_predict_context_display_fields() error {
	db := sqls.DB()
	if err := db.AutoMigrate(&models.PredictContext{}); err != nil {
		slog.Error("migrate predict context display fields failed", "error", err)
		return err
	}

	stmts := []string{
		"ALTER TABLE t_predict_context ADD COLUMN IF NOT EXISTS list_image VARCHAR(512) NOT NULL DEFAULT ''",
		"ALTER TABLE t_predict_context ADD COLUMN IF NOT EXISTS side_a_bg_image VARCHAR(512) NOT NULL DEFAULT ''",
		"ALTER TABLE t_predict_context ADD COLUMN IF NOT EXISTS side_b_bg_image VARCHAR(512) NOT NULL DEFAULT ''",
		"ALTER TABLE t_predict_context ADD COLUMN IF NOT EXISTS side_a_bg_color VARCHAR(64) NOT NULL DEFAULT '#E23D3D'",
		"ALTER TABLE t_predict_context ADD COLUMN IF NOT EXISTS side_b_bg_color VARCHAR(64) NOT NULL DEFAULT '#276EF1'",
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			slog.Error("migrate predict context display field statement failed", "stmt", stmt, "error", err)
			return err
		}
	}

	return nil
}
