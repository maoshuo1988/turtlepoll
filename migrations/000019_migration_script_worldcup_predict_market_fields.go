package migrations

import (
	"bbs-go/internal/models/models"
	"log/slog"

	"github.com/mlogclub/simple/sqls"
)

func migrate_worldcup_predict_market_fields() error {
	db := sqls.DB()
	if err := db.AutoMigrate(
		&models.MatchSchedule{},
		&models.PredictMarket{},
		&models.PredictBet{},
	); err != nil {
		slog.Error("migrate worldcup predict market fields failed", "error", err)
		return err
	}

	stmts := []string{
		"ALTER TABLE t_match_schedule ADD COLUMN IF NOT EXISTS match_phase VARCHAR(32) NOT NULL DEFAULT ''",
		"ALTER TABLE t_match_schedule ADD COLUMN IF NOT EXISTS home_score BIGINT NOT NULL DEFAULT -1",
		"ALTER TABLE t_match_schedule ADD COLUMN IF NOT EXISTS away_score BIGINT NOT NULL DEFAULT -1",
		"ALTER TABLE t_match_schedule ADD COLUMN IF NOT EXISTS winner VARCHAR(32) NOT NULL DEFAULT ''",

		"ALTER TABLE t_predict_market ADD COLUMN IF NOT EXISTS base_draw BIGINT NOT NULL DEFAULT 500",
		"ALTER TABLE t_predict_market ADD COLUMN IF NOT EXISTS pool_draw BIGINT NOT NULL DEFAULT 0",

		"ALTER TABLE t_predict_bet ADD COLUMN IF NOT EXISTS eff_draw BIGINT NOT NULL DEFAULT 0",
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			slog.Error("migrate worldcup predict market field statement failed", "stmt", stmt, "error", err)
			return err
		}
	}

	return nil
}
