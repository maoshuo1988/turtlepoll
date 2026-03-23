package api

import (
	"bbs-go/internal/models"
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/pkg/config"
	"bbs-go/internal/pkg/errs"
	"bbs-go/internal/services"
	"context"
	"log/slog"
	"strings"
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

// 热度榜：GET /api/football/predict_context/hot?limit=10
// 返回 heat 倒序的 PredictContext 列表。
// 说明：当前挂在 /api 下，默认需要登录（AuthMiddleware）。
func (c *FootballController) GetPredict_contextHot() *web.JsonResult {
	limit, _ := params.GetInt(c.Ctx, "limit")
	if limit <= 0 {
		limit = 10
	}
	// 给个上限，避免被当作全表导出
	if limit > 100 {
		limit = 100
	}

	var list []models.PredictContext
	sqls.DB().Order("heat desc, id desc").Limit(limit).Find(&list)
	return web.JsonData(map[string]any{
		"list":  list,
		"limit": limit,
	})
}

// 热门标签 TOP10：GET /api/football/predict_tags/hot?limit=10
// 说明：统计 PredictContext.tags 中出现的标签，按“标签热度”倒序返回。
// 当前热度口径：对每条 PredictContext，把它的 heat 计入其所有 tags 的累计热度。
// 注意：tags 存储为逗号分隔字符串，属于轻量实现。
func (c *FootballController) GetPredict_tagsHot() *web.JsonResult {
	limit, _ := params.GetInt(c.Ctx, "limit")
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	// 只取必要字段，避免扫描时传输大字段
	type row struct {
		Tags string
		Heat int64
	}
	var rows []row
	// 过滤空 tags，减少无效行
	if err := sqls.DB().Model(&models.PredictContext{}).
		Select("tags, heat").
		Where("tags <> ''").
		Find(&rows).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	tagHeat := make(map[string]int64)
	for _, r := range rows {
		if strings.TrimSpace(r.Tags) == "" {
			continue
		}
		parts := strings.Split(r.Tags, ",")
		for _, p := range parts {
			tag := strings.ToLower(strings.TrimSpace(p))
			if tag == "" {
				continue
			}
			tagHeat[tag] += r.Heat
		}
	}

	top := services.PredictTagService.TopN(tagHeat, limit)
	return web.JsonData(map[string]any{
		"list":  top,
		"limit": limit,
	})
}

// 按标签查询预测市场：GET /api/football/markets/by_tag?tag=xxx&page=1&limit=20
// 返回结构与 /api/football/markets 类似：market + context。
func (c *FootballController) GetMarketsBy_tag() *web.JsonResult {
	tag := strings.TrimSpace(c.Ctx.URLParamDefault("tag", ""))
	if tag == "" {
		return web.JsonErrorMsg("tag is required")
	}
	// 统一成小写匹配（存储侧不强制，小写匹配能覆盖更多情况）
	tag = strings.ToLower(tag)

	// 先查 context（模糊匹配，确保命中逗号分隔的 tag 边界）
	// 包含四种情况：
	// - 单标签：tags = 'xxx'
	// - 开头：tags like 'xxx,%'
	// - 结尾：tags like '%,xxx'
	// - 中间：tags like '%,xxx,%'
	var ctxList []models.PredictContext
	q := sqls.DB().Model(&models.PredictContext{})
	q = q.Where(
		"lower(tags) = ? OR lower(tags) LIKE ? OR lower(tags) LIKE ? OR lower(tags) LIKE ?",
		tag,
		tag+",%",
		"%,"+tag,
		"%,"+tag+",%",
	)
	// 分页参数
	page, _ := params.GetInt(c.Ctx, "page")
	if page <= 0 {
		page = 1
	}
	limit, _ := params.GetInt(c.Ctx, "limit")
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	if err := q.Order("heat desc, id desc").Offset(offset).Limit(limit).Find(&ctxList).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	if len(ctxList) == 0 {
		return web.JsonData(map[string]any{
			"list":  []any{},
			"total": total,
			"page":  page,
			"limit": limit,
			"tag":   tag,
		})
	}

	marketIds := make([]int64, 0, len(ctxList))
	ctxMap := make(map[int64]models.PredictContext, len(ctxList))
	for _, mc := range ctxList {
		marketIds = append(marketIds, mc.MarketId)
		ctxMap[mc.MarketId] = mc
	}

	var markets []models.PredictMarket
	if err := sqls.DB().Where("id in (?)", marketIds).Find(&markets).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	marketMap := make(map[int64]models.PredictMarket, len(markets))
	for _, m := range markets {
		marketMap[m.Id] = m
	}

	respList := make([]map[string]any, 0, len(ctxList))
	for _, mc := range ctxList {
		m, ok := marketMap[mc.MarketId]
		if !ok {
			continue
		}
		respList = append(respList, map[string]any{
			"market":  m,
			"context": mc,
		})
	}

	return web.JsonData(map[string]any{
		"list":  respList,
		"total": total,
		"page":  page,
		"limit": limit,
		"tag":   tag,
	})
}

// 修改/创建预测市场上下文：POST /api/football/predict_context/update
// 说明：当前挂在 /api 下，默认需要登录（AuthMiddleware）。
func (c *FootballController) PostPredict_contextUpdate() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}

	form := &models.PredictContext{}
	if err := params.ReadForm(c.Ctx, form); err != nil {
		return web.JsonError(err)
	}

	// 允许用路由参数 marketId 覆盖（方便前端传参）
	if form.MarketId <= 0 {
		marketId, _ := params.GetInt64(c.Ctx, "marketId")
		form.MarketId = marketId
	}

	ctxModel, err := services.PredictContextService.UpsertByMarketId(form)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(ctxModel)
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
