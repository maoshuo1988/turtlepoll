package migrations

import (
	"bbs-go/internal/models/models"

	"github.com/mlogclub/simple/sqls"
)

func migrate_polymarket_tracking_tables() error {
	db := sqls.DB()
	if err := db.AutoMigrate(
		&models.PredictMarketTracking{},
		&models.PredictMarketOutcome{},
		&models.PredictMarketSettleIssue{},
	); err != nil {
		return err
	}
	return db.Model(&models.PredictMarket{}).Where("status = ?", "CLOSE").Update("status", "CLOSED").Error
}
