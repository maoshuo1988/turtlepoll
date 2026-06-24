package migrations

import (
	"bbs-go/internal/models/models"

	"github.com/mlogclub/simple/sqls"
)

func migrate_tear_heat_support_tables() error {
	db := sqls.DB()
	if err := db.AutoMigrate(
		&models.TearInteractLog{},
		&models.TearUserEventStat{},
		&models.TearCampMember{},
		&models.TearHeatSnapshot{},
	); err != nil {
		return err
	}
	return nil
}
