package services

import (
	"bbs-go/internal/models/models"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
)

// CloseExpiredOpenMarkets 将已过封盘时间但仍标记为 OPEN 的市场修正为 CLOSED。
// 返回本次更新的行数。
func CloseExpiredOpenMarkets(nowSec int64) (int64, error) {
	if nowSec <= 0 {
		nowSec = dates.NowTimestamp()
		if nowSec > 1_000_000_000_000 {
			nowSec = nowSec / 1000
		}
	}

	res := sqls.DB().Model(&models.PredictMarket{}).
		Where("status = ? AND close_time > 0 AND close_time <= ?", "OPEN", nowSec).
		Updates(map[string]any{
			"status":      "CLOSED",
			"update_time": nowSec,
		})
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}
