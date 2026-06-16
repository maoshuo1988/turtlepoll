package admin

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/models"
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/pkg/config"
	"bbs-go/internal/pkg/errs"
	"bbs-go/internal/services"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PredictController 预测市场运营侧聚合接口
// 路由：/api/admin/predict
// 注意：该 party 已经过 AuthMiddleware + AdminMiddleware。
type PredictController struct {
	Ctx iris.Context
}

func normalizeAdminPredictMarketStatus(v string) string {
	v = strings.ToUpper(strings.TrimSpace(v))
	switch v {
	case "OPEN", "CLOSED", "SETTLED":
		return v
	default:
		return "OPEN"
	}
}

// ensureMarketForContextUpdate 保证 context_update 对应的 market 存在：
// - 传了 marketId：若存在直接返回；不存在则按该 ID 创建。
// - 没传 marketId：创建新 market 并回填 form.MarketId。
func (c *PredictController) ensureMarketForContextUpdate(form *models.PredictContext) (*models.PredictMarket, error) {
	db := sqls.DB()
	if form == nil {
		return nil, errors.New("invalid form")
	}

	title := strings.TrimSpace(c.Ctx.FormValue("title"))
	if title == "" {
		title = strings.TrimSpace(form.EventName)
	}
	if title == "" {
		title = "Untitled Market"
	}

	marketType := services.NormalizePredictMarketType(c.Ctx.FormValue("marketType"))
	status := normalizeAdminPredictMarketStatus(c.Ctx.FormValue("status"))
	closeTime, _ := params.GetInt64(c.Ctx, "closeTime")
	if closeTime < 0 {
		closeTime = 0
	}
	sourceModel := strings.TrimSpace(c.Ctx.FormValue("sourceModel"))
	if sourceModel == "" {
		sourceModel = "AdminManual"
	}
	sourceModelId, _ := params.GetInt64(c.Ctx, "sourceModelId")

	if form.MarketId > 0 {
		market := &models.PredictMarket{}
		err := db.Where("id = ?", form.MarketId).First(market).Error
		if err == nil {
			return market, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		if sourceModelId <= 0 {
			sourceModelId = form.MarketId
		}
		market = &models.PredictMarket{
			Model:         models.Model{Id: form.MarketId},
			SourceModel:   sourceModel,
			SourceModelId: sourceModelId,
			Title:         title,
			MarketType:    marketType,
			Status:        status,
			CloseTime:     closeTime,
			CreateTime:    dates.NowTimestamp(),
			UpdateTime:    dates.NowTimestamp(),
		}
		if e := db.Create(market).Error; e != nil {
			return nil, e
		}
		return market, nil
	}

	if sourceModelId <= 0 {
		sourceModelId = time.Now().UnixNano()
	}
	market := &models.PredictMarket{}
	err := db.Where("source_model = ? AND source_model_id = ?", sourceModel, sourceModelId).First(market).Error
	if err == nil {
		form.MarketId = market.Id
		return market, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	now := dates.NowTimestamp()
	market = &models.PredictMarket{
		SourceModel:   sourceModel,
		SourceModelId: sourceModelId,
		Title:         title,
		MarketType:    marketType,
		Status:        status,
		CloseTime:     closeTime,
		CreateTime:    now,
		UpdateTime:    now,
	}
	if e := db.Create(market).Error; e != nil {
		return nil, e
	}
	form.MarketId = market.Id
	return market, nil
}

func (c *PredictController) BeforeActivation(b mvc.BeforeActivation) {
	b.Handle("POST", "/market/settle", "PostMarket_settle")
	b.Handle("POST", "/comment_reward/run", "PostComment_rewardRun")
	b.Handle("POST", "/comment_reward/retry", "PostComment_rewardRetry")
	b.Handle("GET", "/comment_reward/logs", "GetComment_rewardLogs")
}

// PostSyncWorldcup POST /api/admin/predict/sync_worldcup
// 管理员手动同步世界杯赛程。可选 query: competition, season。
func (c *PredictController) PostSync_worldcup() *web.JsonResult {
	adminUser := common.GetCurrentUser(c.Ctx)
	if adminUser == nil {
		return web.JsonError(errs.NotLogin())
	}

	competition := c.Ctx.URLParamDefault("competition", "")
	season, _ := params.GetInt(c.Ctx, "season")

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

	services.OperateLogService.AddOperateLog(adminUser.Id, constants.OpTypeUpdate, "predictWorldCupSync", 0,
		fmt.Sprintf("admin sync worldcup schedules: competition=%s season=%d", competition, season), c.Ctx.Request())
	return web.JsonSuccess()
}

// PostContextUpdate POST /api/admin/predict/context/update
// 管理员修改/创建预测市场上下文展示字段。
func (c *PredictController) PostContext_update() *web.JsonResult {
	adminUser := common.GetCurrentUser(c.Ctx)
	if adminUser == nil {
		return web.JsonError(errs.NotLogin())
	}

	form := &models.PredictContext{}
	if err := params.ReadForm(c.Ctx, form); err != nil {
		return web.JsonError(err)
	}
	if form.MarketId <= 0 {
		marketId, _ := params.GetInt64(c.Ctx, "marketId")
		form.MarketId = marketId
	}
	market, err := c.ensureMarketForContextUpdate(form)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	if strings.TrimSpace(form.EventName) == "" {
		form.EventName = market.Title
	}

	ctxModel, err := services.PredictContextService.UpsertByMarketId(form)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	services.OperateLogService.AddOperateLog(adminUser.Id, constants.OpTypeUpdate, "predictContext", ctxModel.Id,
		fmt.Sprintf("admin update predict context: marketId=%d", form.MarketId), c.Ctx.Request())
	return web.JsonData(ctxModel)
}

// GetStats GET /api/admin/predict/stats
// 返回：进行中/已结算市场数 + 今日新增市场数
func (c *PredictController) GetStats() *web.JsonResult {
	var openCount int64
	if err := sqls.DB().Model(&models.PredictMarket{}).Where("status = ?", "OPEN").Count(&openCount).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	var closedCount int64
	if err := sqls.DB().Model(&models.PredictMarket{}).Where("status = ?", "CLOSED").Count(&closedCount).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	var settledCount int64
	if err := sqls.DB().Model(&models.PredictMarket{}).Where("status = ?", "SETTLED").Count(&settledCount).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	nowSec := dates.NowTimestamp()
	startOfDaySec := nowSec - (nowSec % 86400)
	var todayNewMarkets int64
	if err := sqls.DB().Model(&models.PredictMarket{}).Where("create_time >= ?", startOfDaySec).Count(&todayNewMarkets).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	// 今日下注额：按 PredictBet.create_time 聚合 amount
	type sumRow struct {
		Sum int64 `gorm:"column:sum"`
	}
	var todayBetAmountRow sumRow
	if err := sqls.DB().Model(&models.PredictBet{}).
		Select("COALESCE(SUM(amount), 0) as sum").
		Where("create_time >= ?", startOfDaySec).
		Scan(&todayBetAmountRow).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	// 今日入场费/燃烧：当前预测市场下注实现只会写 UserCoinLog(BET/SETTLE...)，没有 fee/burn 的独立流水，先返回 0。
	// 若后续引入 fee/burn（例如在 SpendBet 中拆单或新增 bizType），可在此处按 UserCoinLog.bizType 聚合。
	todayFee := int64(0)
	todayBurn := int64(0)

	return web.JsonData(map[string]any{
		"openCount":       openCount,
		"closedCount":     closedCount,
		"settledCount":    settledCount,
		"todayNewMarkets": todayNewMarkets,
		"todayBetAmount":  todayBetAmountRow.Sum,
		"todayFee":        todayFee,
		"todayBurn":       todayBurn,
	})
}

type predictTrendItem struct {
	Day   string `json:"day"`
	Count int64  `json:"count"`
}

// GetTrends GET /api/admin/predict/trends?range=7d
// 说明：当前仅提供“每日新增市场数”趋势（market create_time）。
func (c *PredictController) GetTrends() *web.JsonResult {
	rangeStr := c.Ctx.URLParamDefault("range", "7d")
	days := 7
	if rangeStr == "14d" {
		days = 14
	}
	if rangeStr == "30d" {
		days = 30
	}
	if days <= 0 {
		days = 7
	}
	if days > 90 {
		days = 90
	}

	nowSec := dates.NowTimestamp()
	startOfToday := nowSec - (nowSec % 86400)
	startSec := startOfToday - int64((days-1)*86400)

	// Postgres: to_timestamp(create_time) 把秒转 timestamp；date_trunc('day', ...) 聚合。
	rows := make([]struct {
		Day   string
		Count int64
	}, 0)

	if err := sqls.DB().
		Model(&models.PredictMarket{}).
		Select("to_char(date_trunc('day', to_timestamp(create_time)), 'YYYY-MM-DD') as day, count(1) as count").
		Where("create_time >= ?", startSec).
		Group("day").
		Order("day asc").
		Scan(&rows).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	// 补齐空日期，保证前端柱状图连续
	m := map[string]int64{}
	for _, r := range rows {
		m[r.Day] = r.Count
	}
	list := make([]predictTrendItem, 0, days)
	for i := 0; i < days; i++ {
		daySec := startSec + int64(i*86400)
		dayStr := time.Unix(daySec, 0).UTC().Format("2006-01-02")
		list = append(list, predictTrendItem{Day: dayStr, Count: m[dayStr]})
	}

	return web.JsonData(map[string]any{
		"range": rangeStr,
		"days":  days,
		"list":  list,
	})
}

type activeUserItem struct {
	Day             string `json:"day"`
	ActiveUserCount int64  `json:"activeUserCount"`
}

// GetActive_users GET /api/admin/predict/active_users?range=7d
// 说明：按 PredictBet 的 create_time（秒级）统计每日下注过的去重用户数。
func (c *PredictController) GetActive_users() *web.JsonResult {
	rangeStr := c.Ctx.URLParamDefault("range", "7d")
	days := 7
	if rangeStr == "14d" {
		days = 14
	}
	if rangeStr == "30d" {
		days = 30
	}
	if days <= 0 {
		days = 7
	}
	if days > 90 {
		days = 90
	}

	nowSec := dates.NowTimestamp()
	startOfToday := nowSec - (nowSec % 86400)
	startSec := startOfToday - int64((days-1)*86400)

	rows := make([]struct {
		Day   string
		Count int64
	}, 0)
	if err := sqls.DB().
		Model(&models.PredictBet{}).
		Select("to_char(date_trunc('day', to_timestamp(create_time)), 'YYYY-MM-DD') as day, count(distinct user_id) as count").
		Where("create_time >= ?", startSec).
		Group("day").
		Order("day asc").
		Scan(&rows).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	m := map[string]int64{}
	for _, r := range rows {
		m[r.Day] = r.Count
	}
	list := make([]activeUserItem, 0, days)
	for i := 0; i < days; i++ {
		daySec := startSec + int64(i*86400)
		dayStr := time.Unix(daySec, 0).UTC().Format("2006-01-02")
		list = append(list, activeUserItem{Day: dayStr, ActiveUserCount: m[dayStr]})
	}

	return web.JsonData(map[string]any{
		"range": rangeStr,
		"days":  days,
		"list":  list,
	})
}

type predictMarketStatsResp struct {
	MarketId      int64  `json:"marketId"`
	Status        string `json:"status"`
	Result        string `json:"result"`
	CloseTime     int64  `json:"closeTime"`
	Resolved      bool   `json:"resolved"`
	ResolvedAt    int64  `json:"resolvedAt"`
	ProUserCount  int64  `json:"proUserCount"`
	ConUserCount  int64  `json:"conUserCount"`
	DrawUserCount int64  `json:"drawUserCount"`
	ProAmount     int64  `json:"proAmount"`
	ConAmount     int64  `json:"conAmount"`
	DrawAmount    int64  `json:"drawAmount"`
	TotalAmount   int64  `json:"totalAmount"`
	TotalBetCount int64  `json:"totalBetCount"`
}

// GetMarketStats GET /api/admin/predict/market/stats?marketId=...
// 返回：结算前盘口统计（A/B 人数与金额）。
func (c *PredictController) GetMarket_stats() *web.JsonResult {
	marketId, _ := params.GetInt64(c.Ctx, "marketId")
	if marketId <= 0 {
		return web.JsonErrorMsg("marketId is required")
	}

	market := &models.PredictMarket{}
	if err := sqls.DB().Take(market, "id = ?", marketId).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	// 盘口统计：按 option=A/B 聚合下注金额与去重用户数
	rows := make([]struct {
		Option    string
		AmountSum int64
		UserCnt   int64
		BetCnt    int64
	}, 0)
	if err := sqls.DB().
		Model(&models.PredictBet{}).
		Select("option, coalesce(sum(amount), 0) as amount_sum, count(distinct user_id) as user_cnt, count(1) as bet_cnt").
		Where("market_id = ?", marketId).
		Group("option").
		Scan(&rows).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	get := func(opt string) (amount, userCnt, betCnt int64) {
		for _, r := range rows {
			if strings.ToUpper(strings.TrimSpace(r.Option)) == opt {
				return r.AmountSum, r.UserCnt, r.BetCnt
			}
		}
		return 0, 0, 0
	}
	proAmount, proUserCnt, proBetCnt := get("A")
	conAmount, conUserCnt, conBetCnt := get("B")
	drawAmount, drawUserCnt, drawBetCnt := get("DRAW")

	resp := &predictMarketStatsResp{
		MarketId:      market.Id,
		Status:        market.Status,
		Result:        market.Result,
		CloseTime:     market.CloseTime,
		Resolved:      market.Resolved,
		ResolvedAt:    market.ResolvedAt,
		ProUserCount:  proUserCnt,
		ConUserCount:  conUserCnt,
		DrawUserCount: drawUserCnt,
		ProAmount:     proAmount,
		ConAmount:     conAmount,
		DrawAmount:    drawAmount,
		TotalAmount:   proAmount + conAmount + drawAmount,
		TotalBetCount: proBetCnt + conBetCnt + drawBetCnt,
	}
	return web.JsonData(resp)
}

type adminSettlePredictMarketForm struct {
	MarketId   int64  `json:"marketId"`
	Result     string `json:"result"`     // A/B/DRAW（DRAW only for 1x2）
	RequestId  string `json:"requestId"`  // for audit
	Remark     string `json:"remark"`     // optional
	AllowReset bool   `json:"allowReset"` // if true: allow SETTLED -> SETTLED (admin fix), default false
}

// PostMarketSettle POST /api/admin/predict/market/settle
// 管理员结算：将 market 从 CLOSED 结算为 SETTLED，并写入最终结果（A/B/DRAW）。
func (c *PredictController) PostMarket_settle() *web.JsonResult {
	adminUser := common.GetCurrentUser(c.Ctx)
	if adminUser == nil {
		return web.JsonError(errs.NotLogin())
	}

	var form adminSettlePredictMarketForm
	if err := c.Ctx.ReadJSON(&form); err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	if form.MarketId <= 0 {
		return web.JsonErrorMsg("marketId is required")
	}
	result := strings.ToUpper(strings.TrimSpace(form.Result))
	if strings.TrimSpace(form.RequestId) == "" {
		return web.JsonErrorMsg("requestId is required")
	}
	remark := strings.TrimSpace(form.Remark)
	if len(remark) > 1024 {
		remark = remark[:1024]
	}

	now := dates.NowTimestamp()
	var updated *models.PredictMarket
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		m := &models.PredictMarket{}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Take(m, "id = ?", form.MarketId).Error; err != nil {
			return err
		}
		if !services.IsValidPredictOption(m.MarketType, result) {
			return errors.New(services.PredictOptionErrMsg(m.MarketType))
		}

		// 默认仅允许 CLOSED -> SETTLED
		if m.Status == "SETTLED" {
			if !form.AllowReset {
				return errors.New("market already settled")
			}
		}
		if m.Status != "CLOSED" && m.Status != "CLOSE" && !(form.AllowReset && m.Status == "SETTLED") {
			return fmt.Errorf("market status must be CLOSED (or SETTLED with allowReset), got=%s", m.Status)
		}

		m.Status = "SETTLED"
		m.Result = result
		// 兼容 TurtlePoll：同时标记 resolved
		m.Resolved = true
		if m.ResolvedAt == 0 {
			m.ResolvedAt = now
		}
		m.UpdateTime = now
		if err := tx.Save(m).Error; err != nil {
			return err
		}
		updated = m
		return nil
	})
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	// 操作日志（不影响主流程）
	desc := fmt.Sprintf("admin settle predict market: marketId=%d, result=%s, requestId=%s, remark=%s", form.MarketId, result, form.RequestId, remark)
	services.OperateLogService.AddOperateLog(adminUser.Id, constants.OpTypeUpdate, "predictMarket", form.MarketId, desc, c.Ctx.Request())

	return web.JsonData(updated)
}

func (c *PredictController) PostComment_rewardRun() *web.JsonResult {
	marketId, _ := params.GetInt64(c.Ctx, "marketId")
	log, err := services.PredictCommentRewardService.RunForMarket(marketId, false)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(log)
}

func (c *PredictController) PostComment_rewardRetry() *web.JsonResult {
	rewardLogId, _ := params.GetInt64(c.Ctx, "rewardLogId")
	log, err := services.PredictCommentRewardService.Retry(rewardLogId)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(log)
}

func (c *PredictController) GetComment_rewardLogs() *web.JsonResult {
	query := params.NewQueryParams(c.Ctx).
		EqByReq("market_id").
		EqByReq("status").
		PageByReq().
		Desc("id")
	var list []models.PredictCommentRewardLog
	query.Cnd.Find(sqls.DB(), &list)
	count := query.Cnd.Count(sqls.DB(), &models.PredictCommentRewardLog{})
	logIds := make([]int64, 0, len(list))
	for _, row := range list {
		logIds = append(logIds, row.Id)
	}
	itemsByLogId := map[int64][]models.PredictCommentRewardItem{}
	if len(logIds) > 0 {
		var items []models.PredictCommentRewardItem
		_ = sqls.DB().Where("reward_log_id IN ?", logIds).Order("id asc").Find(&items).Error
		for _, item := range items {
			itemsByLogId[item.RewardLogId] = append(itemsByLogId[item.RewardLogId], item)
		}
	}
	results := make([]map[string]any, 0, len(list))
	for _, row := range list {
		results = append(results, map[string]any{
			"log":   row,
			"items": itemsByLogId[row.Id],
		})
	}
	return web.JsonData(&web.PageResult{
		Results: results,
		Page: &sqls.Paging{
			Page:  query.Cnd.Paging.Page,
			Limit: query.Cnd.Paging.Limit,
			Total: count,
		},
	})
}

// 避免 iris 未使用（预留后续可能返回 map）
var _ = iris.Map{}
