package services

import (
	"bbs-go/internal/models/models"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"

	"github.com/mlogclub/simple/sqls"
)

// PetGachaService 开蛋池配置服务（运营侧维护，用户侧抽取概率依据 rarity_weights）。
var PetGachaService = newPetGachaService()

type petGachaService struct{}

type UpdateGachaPoolConfigRequest struct {
	Enabled      *bool              `json:"enabled"`
	BaseCost     *int64             `json:"base_cost"`
	RarityWeight map[string]float64 `json:"rarity_weights"`
}

func newPetGachaService() *petGachaService { return &petGachaService{} }

func (s *petGachaService) defaultConfig() models.GachaPoolConfig {
	slog.Info("pet gacha defaultConfig: build")
	b, _ := json.Marshal(models.DefaultGachaRarityWeights)
	return models.GachaPoolConfig{
		Enabled:           true,
		BaseCost:          models.DefaultGachaBaseCost,
		RarityWeightsJSON: string(b),
	}
}

// GetConfig 获取开蛋池配置；若表中没有任何配置记录，则写入默认配置后返回。
func (s *petGachaService) GetConfig() (*models.GachaPoolConfig, error) {
	slog.Info("pet gacha GetConfig: enter")
	db := sqls.DB()
	var cfg models.GachaPoolConfig
	err := db.Order("id asc").First(&cfg).Error
	if err == nil {
		slog.Info("pet gacha GetConfig: found", "id", cfg.Id)
		return &cfg, nil
	}
	slog.Warn("pet gacha GetConfig: first failed, will check count", "error", err)
	// gorm.ErrRecordNotFound 的判断不要引入 gorm 依赖：这里用 count 再决定是否初始化。
	var cnt int64
	if e := db.Model(&models.GachaPoolConfig{}).Count(&cnt).Error; e != nil {
		slog.Error("pet gacha GetConfig: count failed", "error", e)
		return nil, e
	}
	slog.Info("pet gacha GetConfig: count", "count", cnt)
	if cnt > 0 {
		// 有记录但 First 出错，直接返回原始错误
		slog.Error("pet gacha GetConfig: unexpected first error when count>0", "error", err)
		return nil, err
	}

	def := s.defaultConfig()
	slog.Info("pet gacha GetConfig: table empty, will create default")
	if e := db.Create(&def).Error; e != nil {
		slog.Error("pet gacha GetConfig: create default failed", "error", e)
		return nil, e
	}
	slog.Info("pet gacha GetConfig: created default", "id", def.Id)
	return &def, nil
}

func (s *petGachaService) ResetConfig() (*models.GachaPoolConfig, error) {
	slog.Info("pet gacha ResetConfig: enter")
	cfg, err := s.GetConfig()
	if err != nil {
		slog.Error("pet gacha ResetConfig: GetConfig failed", "error", err)
		return nil, err
	}
	def := s.defaultConfig()
	updates := map[string]any{
		"enabled":             def.Enabled,
		"base_cost":           def.BaseCost,
		"rarity_weights_json": def.RarityWeightsJSON,
	}
	if e := sqls.DB().Model(&models.GachaPoolConfig{}).Where("id = ?", cfg.Id).Updates(updates).Error; e != nil {
		slog.Error("pet gacha ResetConfig: update failed", "id", cfg.Id, "error", e)
		return nil, e
	}
	slog.Info("pet gacha ResetConfig: ok", "id", cfg.Id)
	return s.GetConfig()
}

func (s *petGachaService) UpdateConfig(req UpdateGachaPoolConfigRequest) (*models.GachaPoolConfig, error) {
	slog.Info("pet gacha UpdateConfig: enter", "has_enabled", req.Enabled != nil, "has_base_cost", req.BaseCost != nil, "has_rarity_weights", req.RarityWeight != nil)
	cfg, err := s.GetConfig()
	if err != nil {
		slog.Error("pet gacha UpdateConfig: GetConfig failed", "error", err)
		return nil, err
	}

	updates := map[string]any{}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.BaseCost != nil {
		if *req.BaseCost < 0 {
			slog.Warn("pet gacha UpdateConfig: invalid base_cost", "base_cost", *req.BaseCost)
			return nil, errors.New("base_cost must be >= 0")
		}
		updates["base_cost"] = *req.BaseCost
	}
	if req.RarityWeight != nil {
		slog.Info("pet gacha UpdateConfig: validate rarity_weights", "keys", len(req.RarityWeight))
		if err := validateRarityWeights(req.RarityWeight); err != nil {
			slog.Warn("pet gacha UpdateConfig: validate rarity_weights failed", "error", err)
			return nil, err
		}
		b, _ := json.Marshal(req.RarityWeight)
		updates["rarity_weights_json"] = string(b)
	}

	if len(updates) == 0 {
		slog.Info("pet gacha UpdateConfig: no updates, return current")
		return cfg, nil
	}

	if e := sqls.DB().Model(&models.GachaPoolConfig{}).Where("id = ?", cfg.Id).Updates(updates).Error; e != nil {
		slog.Error("pet gacha UpdateConfig: db update failed", "id", cfg.Id, "error", e)
		return nil, e
	}
	slog.Info("pet gacha UpdateConfig: ok", "id", cfg.Id)
	return s.GetConfig()
}

func validateRarityWeights(w map[string]float64) error {
	allowed := map[string]struct{}{"C": {}, "B": {}, "A": {}, "S": {}, "SS": {}, "SSS": {}}
	sum := 0.0
	for k, v := range w {
		if _, ok := allowed[k]; !ok {
			return fmt.Errorf("invalid rarity key: %s", k)
		}
		if v < 0 || v > 1 {
			return fmt.Errorf("rarity_weights[%s] must be within [0,1]", k)
		}
		sum += v
	}
	slog.Info("pet gacha validateRarityWeights: computed sum", "sum", sum)
	if math.Abs(sum-1.0) > 1e-9 {
		return fmt.Errorf("rarity_weights sum must equal 1, current sum=%v", sum)
	}
	return nil
}
