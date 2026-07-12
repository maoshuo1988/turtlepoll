package migrations

import "github.com/mlogclub/simple/sqls"

// migrate_normalize_predict_market_close_time_unit
// 将历史毫秒 close_time 统一归一为秒，避免倒计时和封盘判断出现量级错误。
func migrate_normalize_predict_market_close_time_unit() error {
	return sqls.DB().Exec("UPDATE t_predict_market SET close_time = close_time / 1000 WHERE close_time > 1000000000000").Error
}
