package admin

import (
	"bbs-go/internal/services"
	"log/slog"
	"net/http"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
)

// PetGachaController 开蛋池配置（Admin）
// 路由：/api/admin/pet/gacha/*
// - GET  /config
// - POST /config
// - POST /config/reset

type PetGachaController struct {
	Ctx iris.Context
}

func (c *PetGachaController) GetConfig() *web.JsonResult {
	slog.Info("admin pet gacha GetConfig: enter")
	cfg, err := services.PetGachaService.GetConfig()
	if err != nil {
		slog.Error("admin pet gacha GetConfig: service error", "error", err)
		return web.JsonErrorMsg(err.Error())
	}
	slog.Info("admin pet gacha GetConfig: ok", "id", cfg.Id, "enabled", cfg.Enabled, "base_cost", cfg.BaseCost)
	return web.JsonData(cfg)
}

func (c *PetGachaController) PostConfig() *web.JsonResult {
	slog.Info("admin pet gacha PostConfig: enter")
	var req services.UpdateGachaPoolConfigRequest
	if err := params.ReadJSON(c.Ctx, &req); err != nil {
		slog.Warn("admin pet gacha PostConfig: read json failed", "error", err)
		c.Ctx.StatusCode(http.StatusBadRequest)
		return web.JsonErrorMsg(err.Error())
	}
	slog.Info("admin pet gacha PostConfig: parsed", "has_enabled", req.Enabled != nil, "has_base_cost", req.BaseCost != nil, "has_rarity_weights", req.RarityWeight != nil)
	cfg, err := services.PetGachaService.UpdateConfig(req)
	if err != nil {
		slog.Warn("admin pet gacha PostConfig: service error", "error", err)
		c.Ctx.StatusCode(http.StatusBadRequest)
		return web.JsonErrorMsg(err.Error())
	}
	slog.Info("admin pet gacha PostConfig: ok", "id", cfg.Id, "enabled", cfg.Enabled, "base_cost", cfg.BaseCost)
	return web.JsonData(cfg)
}

func (c *PetGachaController) PostConfigReset() *web.JsonResult {
	slog.Info("admin pet gacha PostConfigReset: enter")
	cfg, err := services.PetGachaService.ResetConfig()
	if err != nil {
		slog.Error("admin pet gacha PostConfigReset: service error", "error", err)
		return web.JsonErrorMsg(err.Error())
	}
	slog.Info("admin pet gacha PostConfigReset: ok", "id", cfg.Id, "enabled", cfg.Enabled, "base_cost", cfg.BaseCost)
	return web.JsonData(cfg)
}
