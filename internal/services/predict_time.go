package services

// predictTimestampToSeconds 统一将秒/毫秒时间戳归一为秒。
func predictTimestampToSeconds(ts int64) int64 {
	if ts <= 0 {
		return ts
	}
	if ts > 1_000_000_000_000 {
		return ts / 1000
	}
	return ts
}
