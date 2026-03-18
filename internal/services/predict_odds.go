package services

import "math"

const (
	PredictOptionA = "A"
	PredictOptionB = "B"
)

// clampOdds 将赔率限制到 [min, max]，用于避免前端显示极端值。
func clampOdds(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// calcOddsRaw 使用“有效总池 / 选项有效池”的最简模型：odds = T_eff / V_eff。
// baseA/baseB 为系统虚拟底池；poolA/poolB 为真实下注池。
func calcOddsRaw(baseA, baseB, poolA, poolB int64) (oddsA, oddsB float64, effA, effB, total int64) {
	effA = baseA + poolA
	effB = baseB + poolB
	totalEff := effA + effB
	if effA <= 0 {
		effA = 1
	}
	if effB <= 0 {
		effB = 1
	}
	oddsA = float64(totalEff) / float64(effA)
	oddsB = float64(totalEff) / float64(effB)
	return oddsA, oddsB, effA, effB, totalEff
}

// CalcClampedOdds 计算并对赔率做上下限处理（默认 1.2 ~ 5.0）。
func CalcClampedOdds(baseA, baseB, poolA, poolB int64) (oddsA, oddsB float64, effA, effB, total int64) {
	oddsA, oddsB, effA, effB, total = calcOddsRaw(baseA, baseB, poolA, poolB)
	oddsA = clampOdds(oddsA, 1.2, 5.0)
	oddsB = clampOdds(oddsB, 1.2, 5.0)
	// 统一保留两位小数，方便展示与锁赔率
	oddsA = math.Round(oddsA*100) / 100
	oddsB = math.Round(oddsB*100) / 100
	return
}
