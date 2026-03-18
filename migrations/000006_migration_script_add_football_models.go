package migrations

import (
	"bbs-go/internal/models"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
)

// 说明：表结构由 gorm AutoMigrate 创建，这里只做必要的数据初始化（例如默认市场状态枚举等）。
// 当前阶段：仅创建空表，不初始化任何业务数据。
func migrate_add_football_models() error {
	// 确保表已存在（AutoMigrate 已做），这里用一个轻量写入作为“迁移占位”。
	_ = sqls.DB().AutoMigrate(&models.MatchSchedule{}, &models.PredictMarket{}, &models.UserCoin{})
	// no-op
	_ = dates.NowTimestamp()
	return nil
}
