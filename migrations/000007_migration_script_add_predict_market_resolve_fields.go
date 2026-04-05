package migrations

import (
	"bbs-go/internal/models/models"

	"github.com/mlogclub/simple/sqls"
)

// migrate_add_predict_market_resolve_fields
// 为外部市场（例如 Polymarket）同步结算结果预留字段。
// 说明：本项目使用 gorm AutoMigrate 管理 schema，本迁移用于确保线上升级时执行到最新结构。
func migrate_add_predict_market_resolve_fields() error {
	_ = sqls.DB().AutoMigrate(
		&models.PredictMarket{},
	)
	return nil
}
