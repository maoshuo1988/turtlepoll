package biztime

import "time"

// 业务统一口径：北京时间（UTC+8）。
//
// 注意：
// - 不依赖服务器 time.Local，避免部署环境时区不一致导致“日切”错乱。
// - DayName 使用 YYYYMMDD 的 int（例如 20260408），便于作为唯一键与索引。

var cst = time.FixedZone("UTC+8", 8*60*60)

// NowInCST 返回当前北京时间。
func NowInCST() time.Time {
	return time.Now().In(cst)
}

// DayNameCST 返回北京时间下的 dayName（YYYYMMDD）。
func DayNameCST(t time.Time) int {
	lt := t.In(cst)
	return lt.Year()*10000 + int(lt.Month())*100 + lt.Day()
}

// DateStringCST 返回北京时间下的日期字符串（YYYY-MM-DD）。
func DateStringCST(t time.Time) string {
	lt := t.In(cst)
	return lt.Format("2006-01-02")
}

// NextMidnightCST 返回“北京时间次日 0 点”的时间点。
//
// 例：
// - t=2026-04-08 11:00 +08:00 => 2026-04-09 00:00 +08:00
// - t=2026-04-08 00:00 +08:00 => 2026-04-09 00:00 +08:00
func NextMidnightCST(t time.Time) time.Time {
	lt := t.In(cst)
	// 当天 0 点
	start := time.Date(lt.Year(), lt.Month(), lt.Day(), 0, 0, 0, 0, cst)
	// 次日 0 点
	return start.Add(24 * time.Hour)
}

// NextMidnightCSTUnix 返回“北京时间次日 0 点”的 unix 秒。
func NextMidnightCSTUnix(t time.Time) int64 {
	return NextMidnightCST(t).Unix()
}
