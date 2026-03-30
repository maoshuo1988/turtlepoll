package migrations

import (
	"bbs-go/internal/models"

	"github.com/mlogclub/simple/sqls"
)

func migrate_battle_square_init_tables() error {
	db := sqls.DB()
	return db.AutoMigrate(
		&models.Battle{},
		&models.BattleBet{},
		&models.BattleChallengeAction{},
		&models.BattleLedger{},
		&models.BattleSettlement{},
		&models.BattleSettlementItem{},
	)
}
