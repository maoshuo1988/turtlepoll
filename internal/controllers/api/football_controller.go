package api

import (
	"bbs-go/internal/models"
	"bbs-go/internal/pkg/config"
	"bbs-go/internal/services"
	"context"
	"log/slog"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/sqls"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
)

// FootballController 提供“收集/同步赛事赛程”的手动接口。
// 注意：当前路由挂在 /api 下，会经过 AuthMiddleware（必须登录）。
// 后续如果需要管理员权限，可以把它挂到 /api/admin 下或增加权限校验。

type FootballController struct {
	Ctx iris.Context
}

// 手动触发同步：POST /api/football/sync_worldcup
func (c *FootballController) PostSync_worldcup() *web.JsonResult {
	// 可选：允许临时覆盖 competition/season（不传则走配置默认）
	competition := c.Ctx.URLParamDefault("competition", "")
	season, _ := params.GetInt(c.Ctx, "season")

	// 临时覆写（仅对本次请求生效）
	bak := config.Instance.FootballData
	if competition != "" {
		config.Instance.FootballData.CompetitionCode = competition
	}
	if season > 0 {
		config.Instance.FootballData.Season = season
	}
	defer func() { config.Instance.FootballData = bak }()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := services.FootballSyncService.SyncWorldCupSchedules(ctx); err != nil {
		return web.JsonError(err)
	}
	return web.JsonSuccess()
}

// 查询赛程列表：GET /api/football/schedules?page=1&limit=20
func (c *FootballController) GetSchedules() *web.JsonResult {
	p := params.NewQueryParams(c.Ctx)
	// 允许简单筛选
	competition := c.Ctx.URLParamDefault("competition", "")
	status := c.Ctx.URLParamDefault("status", "")
	if competition != "" {
		p.Cnd.Where("competition = ?", competition)
	}
	if status != "" {
		p.Cnd.Where("status = ?", status)
	}
	p.Cnd.Desc("utc_date")

	var list []models.MatchSchedule
	p.Cnd.Find(sqls.DB(), &list)
	count := p.Cnd.Count(sqls.DB(), &models.MatchSchedule{})

	return web.JsonData(map[string]any{
		"list":  list,
		"total": count,
	})
}

// 查询预测市场：GET /api/football/markets?page=1&limit=20
func (c *FootballController) GetMarkets() *web.JsonResult {
	p := params.NewQueryParams(c.Ctx)
	sourceModel := c.Ctx.URLParamDefault("sourceModel", "")
	sourceModelId, _ := params.GetInt64(c.Ctx, "sourceModelId")
	if sourceModel != "" {
		p.Cnd.Where("source_model = ?", sourceModel)
	}
	if sourceModelId > 0 {
		p.Cnd.Where("source_model_id = ?", sourceModelId)
	}
	p.Cnd.Desc("id")
	var list []models.PredictMarket
	p.Cnd.Find(sqls.DB(), &list)
	count := p.Cnd.Count(sqls.DB(), &models.PredictMarket{})

	// 查询上下文并拼装
	marketIds := make([]int64, 0, len(list))
	for _, m := range list {
		if m.Id > 0 {
			marketIds = append(marketIds, m.Id)
		}
	}
	ctxMap := make(map[int64]models.PredictContext, len(marketIds))
	if len(marketIds) > 0 {
		var ctxList []models.PredictContext
		sqls.DB().Where("market_id in (?)", marketIds).Find(&ctxList)
		for _, mc := range ctxList {
			ctxMap[mc.MarketId] = mc
		}
	}

	respList := make([]map[string]any, 0, len(list))
	for _, m := range list {
		item := map[string]any{
			"market":  m,
			"context": ctxMap[m.Id],
		}
		respList = append(respList, item)
	}
	return web.JsonData(map[string]any{
		"list":  respList,
		"total": count,
	})
}

func init() {
	slog.Info("football api controller loaded")
}
