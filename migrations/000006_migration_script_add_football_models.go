package migrations

import (
	"bbs-go/internal/models"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
)

// 说明：表结构由 gorm AutoMigrate 创建，这里只做必要的 schema 变更占位。
// 当前阶段：创建/更新表结构（赛程、预测市场、用户金币），不初始化任何业务数据。
func migrate_add_football_models() error {
	// 确保表已存在（AutoMigrate 已做），这里用一个轻量写入作为“迁移占位”。
	_ = sqls.DB().AutoMigrate(&models.MatchSchedule{}, &models.PredictMarket{}, &models.PredictContext{}, &models.UserCoin{})
	// no-op
	_ = dates.NowTimestamp()
	return nil
}
