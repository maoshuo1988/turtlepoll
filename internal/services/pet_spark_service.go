package services

import (
	"errors"
	"math"

	"github.com/mlogclub/simple/common/jsons"
)

var PetSparkService = newPetSparkService()

func newPetSparkService() *petSparkService {
	return &petSparkService{}
}

type petSparkService struct{}

type SparkMultiplierParams struct {
	Enabled            bool    `json:"enabled"`
	Base               float64 `json:"base"`
	PerLevel           float64 `json:"per_level"`
	Cap                int64   `json:"cap"`
	BaseMultiplier     float64 `json:"baseMultiplier"`
	LevelScale         float64 `json:"levelScale"`
	ExtraCapPerDay     int64   `json:"extraCapPerDay"`
	MultiplierBase     float64 `json:"multiplier_base"`
	MultiplierPerLevel float64 `json:"multiplier_per_level"`
	ExtraCapAmount     int64   `json:"extra_cap_amount"`
}

func (s *petSparkService) ComputeSparkRaw(loginStreak int) int64 {
	if loginStreak <= 0 {
		return 0
	}
	raw := 100 * (1 - math.Exp(-0.1*float64(loginStreak)))
	if raw <= 0 {
		return 0
	}
	return int64(math.Floor(raw))
}

func (s *petSparkService) ApplySparkMultiplier(raw int64, level int, params SparkMultiplierParams) (extra int64, final int64) {
	if raw <= 0 {
		return 0, 0
	}
	if level <= 0 {
		level = 1
	}

	base := firstPositiveFloat(
		params.BaseMultiplier,
		params.MultiplierBase,
		params.Base,
	)
	perLevel := firstPositiveFloat(
		params.LevelScale,
		params.MultiplierPerLevel,
		params.PerLevel,
	)
	capAmount := firstPositiveInt64(
		params.ExtraCapPerDay,
		params.ExtraCapAmount,
		params.Cap,
	)

	if base <= 0 {
		base = 1
	}
	multiplier := base + perLevel*float64(level)
	if multiplier < 1 {
		multiplier = 1
	}

	extra = int64(math.Floor(float64(raw) * (multiplier - 1)))
	if extra < 0 {
		extra = 0
	}
	if capAmount > 0 && extra > capAmount {
		extra = capAmount
	}
	return extra, raw + extra
}

func (s *petSparkService) DecodeSparkMultiplierParams(v any) (*SparkMultiplierParams, error) {
	if v == nil {
		return nil, errors.New("empty params")
	}

	var p SparkMultiplierParams
	if err := jsons.Parse(jsons.ToJsonStr(v), &p); err != nil {
		return nil, err
	}

	if p.Base < 0 || p.PerLevel < 0 || p.Cap < 0 ||
		p.BaseMultiplier < 0 || p.LevelScale < 0 || p.ExtraCapPerDay < 0 ||
		p.MultiplierBase < 0 || p.MultiplierPerLevel < 0 || p.ExtraCapAmount < 0 {
		return nil, errors.New("spark multiplier params must be non-negative")
	}

	return &p, nil
}

func firstPositiveFloat(values ...float64) float64 {
	for _, v := range values {
		if v > 0 {
			return v
		}
	}
	return 0
}

func firstPositiveInt64(values ...int64) int64 {
	for _, v := range values {
		if v > 0 {
			return v
		}
	}
	return 0
}
