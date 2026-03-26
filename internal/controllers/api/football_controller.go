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
	// 热度榜按“状态优先级”排序：OPEN -> CLOSE -> (已结算/其他)
	// 同状态下：heat desc, id desc
	// 关联关系：PredictContext.market_id = PredictMarket.id
	if err := sqls.DB().
		Model(&models.PredictContext{}).
		Joins("JOIN t_predict_market pm ON pm.id = t_predict_context.market_id").
		Order("CASE pm.status WHEN 'OPEN' THEN 0 WHEN 'CLOSE' THEN 1 ELSE 2 END, t_predict_context.heat desc, t_predict_context.id desc").
		Limit(limit).
		Find(&list).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}
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
	// 优先级：
	// 1) tags 精确等于 tag（单标签）
	// 2) tags 以 tag 开头
	// 3) tags 以 tag 结尾
	// 4) tags 中间包含 ,tag,
	// 然后再按 heat desc, id desc
	orderBy := "CASE " +
		"WHEN lower(tags) = '" + tag + "' THEN 0 " +
		"WHEN lower(tags) LIKE '" + tag + ",%' THEN 1 " +
		"WHEN lower(tags) LIKE '%," + tag + "' THEN 2 " +
		"ELSE 3 END, heat desc, id desc"
	if err := q.
		Order(orderBy).
		Offset(offset).
		Limit(limit).
		Find(&ctxList).Error; err != nil {
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

	// markets 排序优先级：
	// 1) 非 TBD（主客队已确定）优先
	// 2) 状态优先级：OPEN -> CLOSE -> (已结算/其他)
	// 3) 同状态下：closeTime asc（更接近封盘/比分揭晓的优先），再按 id desc 保证稳定
	var markets []models.PredictMarket
	if err := sqls.DB().
		Where("id in (?)", marketIds).
		Order("CASE WHEN title = 'TBD vs TBD' THEN 1 ELSE 0 END, CASE status WHEN 'OPEN' THEN 0 WHEN 'CLOSE' THEN 1 ELSE 2 END, close_time asc, id desc").
		Find(&markets).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	respList := make([]map[string]any, 0, len(markets))
	for _, m := range markets {
		mc, ok := ctxMap[m.Id]
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

// 查询用户在某个预测市场的下注结算结果：GET /api/football/bet_settle_result?userId=1&marketId=2
// 返回 betSettleResult（聚合 PredictBet.SettleResult）：
// - 无下注：""
// - 多笔：WIN 优先，其次 LOSE，其次最新非空值
func (c *FootballController) GetBet_settle_result() *web.JsonResult {
	// 基础鉴权：接口挂在 /api 下通常已走 AuthMiddleware，但这里显式校验权限
	currentUser := common.GetCurrentUser(c.Ctx)
	if currentUser == nil {
		return web.JsonError(errs.NotLogin())
	}

	userId, _ := params.GetInt64(c.Ctx, "userId")
	marketId, _ := params.GetInt64(c.Ctx, "marketId")
	if userId <= 0 {
		return web.JsonErrorMsg("userId is required")
	}
	if marketId <= 0 {
		return web.JsonErrorMsg("marketId is required")
	}

	// 仅允许查询自己，避免越权
	if userId != currentUser.Id {
		return web.JsonError(errs.NoPermission())
	}

	type betRow struct {
		SettleResult string
		SettleTime   int64
		CreateTime   int64
	}
	var betRows []betRow
	sqls.DB().Model(&models.PredictBet{}).
		Select("settle_result, settle_time, create_time").
		Where("user_id = ? AND market_id = ?", userId, marketId).
		Find(&betRows)

	result := ""
	hasWin := false
	hasLose := false
	latestScore := int64(0)
	latestVal := ""
	for _, br := range betRows {
		v := strings.ToUpper(strings.TrimSpace(br.SettleResult))
		if v == "WIN" {
			hasWin = true
			break
		}
		if v == "LOSE" {
			hasLose = true
		}
		if v == "" {
			continue
		}
		score := br.SettleTime
		if score <= 0 {
			score = br.CreateTime
		}
		if score > latestScore {
			latestScore = score
			latestVal = v
		}
	}
	if hasWin {
		result = "WIN"
	} else if hasLose {
		result = "LOSE"
	} else {
		result = latestVal
	}

	return web.JsonData(map[string]any{
		"userId":          userId,
		"marketId":        marketId,
		"betSettleResult": result,
	})
}

// 查询预测市场：GET /api/football/markets?page=1&limit=20
func (c *FootballController) GetMarkets() *web.JsonResult {
	p := params.NewQueryParams(c.Ctx)
	currentUser := common.GetCurrentUser(c.Ctx)
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

	// 当前用户在各市场的下注结算结果（SettleResult）。
	// - 若该用户在该 market 没有任何下注单，则返回空字符串
	// - 若有多笔下注单：优先返回 WIN，其次 LOSE，其次其他值（尽量给出“总体是否赢”）
	betSettleResultMap := make(map[int64]string, len(marketIds))
	// 当前用户是否在该市场下过注（用于前端快速判断“是否下注过”）
	hasBetMap := make(map[int64]bool, len(marketIds))
	if currentUser != nil && len(marketIds) > 0 {
		type betRow struct {
			MarketId      int64
			SettleResult  string
			SettleTime    int64
			CreateTime    int64
			Status        string
			EffectiveSort int64
		}
		var betRows []betRow
		// 只取必要字段，避免全表扫描/大字段
		sqls.DB().Model(&models.PredictBet{}).
			Select("market_id, settle_result, settle_time, create_time, status").
			Where("user_id = ? AND market_id in (?)", currentUser.Id, marketIds).
			Find(&betRows)

		// 聚合策略：
		// - 只要有任意 WIN => WIN
		// - 否则只要有任意 LOSE => LOSE
		// - 否则返回最新一条（按 settle_time/create_time）非空 settle_result
		hasWin := make(map[int64]bool, len(marketIds))
		hasLose := make(map[int64]bool, len(marketIds))
		latestScore := make(map[int64]int64, len(marketIds))
		latestVal := make(map[int64]string, len(marketIds))
		for _, br := range betRows {
			hasBetMap[br.MarketId] = true
			v := strings.ToUpper(strings.TrimSpace(br.SettleResult))
			if v == "WIN" {
				hasWin[br.MarketId] = true
				continue
			}
			if v == "LOSE" {
				hasLose[br.MarketId] = true
			}
			if v == "" {
				continue
			}
			score := br.SettleTime
			if score <= 0 {
				score = br.CreateTime
			}
			if score > latestScore[br.MarketId] {
				latestScore[br.MarketId] = score
				latestVal[br.MarketId] = v
			}
		}
		for _, m := range list {
			mid := m.Id
			if hasWin[mid] {
				betSettleResultMap[mid] = "WIN"
				continue
			}
			if hasLose[mid] {
				betSettleResultMap[mid] = "LOSE"
				continue
			}
			betSettleResultMap[mid] = latestVal[mid]
		}
	}

	respList := make([]map[string]any, 0, len(list))
	for _, m := range list {
		item := map[string]any{
			"market":          m,
			"context":         ctxMap[m.Id],
			"betSettleResult": betSettleResultMap[m.Id],
			"hasBet":          hasBetMap[m.Id],
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
