package admin

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/models"
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/pkg/errs"
	"bbs-go/internal/services"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
)

// PolymarketController Polymarket 运营侧接口。
// 路由：/api/admin/polymarket
type PolymarketController struct {
	Ctx iris.Context
}

func (c *PolymarketController) PostDiscovery_sync() *web.JsonResult {
	adminUser := common.GetCurrentUser(c.Ctx)
	if adminUser == nil {
		return web.JsonError(errs.NotLogin())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	if err := services.PolymarketSyncService.DiscoveryPoller(ctx); err != nil {
		return web.JsonError(err)
	}
	services.OperateLogService.AddOperateLog(adminUser.Id, constants.OpTypeUpdate, "polymarketDiscovery", 0, "admin trigger polymarket discovery", c.Ctx.Request())
	return web.JsonSuccess()
}

func (c *PolymarketController) PostTracking_sync() *web.JsonResult {
	adminUser := common.GetCurrentUser(c.Ctx)
	if adminUser == nil {
		return web.JsonError(errs.NotLogin())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	if err := services.PolymarketSyncService.TrackingPoller(ctx); err != nil {
		return web.JsonError(err)
	}
	services.OperateLogService.AddOperateLog(adminUser.Id, constants.OpTypeUpdate, "polymarketTracking", 0, "admin trigger polymarket tracking", c.Ctx.Request())
	return web.JsonSuccess()
}

func (c *PolymarketController) GetIssues() *web.JsonResult {
	p := params.NewPagedSqlCnd(c.Ctx)
	status := strings.ToUpper(strings.TrimSpace(c.Ctx.URLParamDefault("status", "")))
	if status == "" {
		status = "OPEN"
	}
	if status != "ALL" {
		p.Where("status = ?", status)
	}
	marketId, _ := params.GetInt64(c.Ctx, "marketId")
	if marketId > 0 {
		p.Where("market_id = ?", marketId)
	}
	reason := strings.ToUpper(strings.TrimSpace(c.Ctx.URLParamDefault("reason", "")))
	if reason != "" {
		p.Where("reason = ?", reason)
	}
	var list []models.PredictMarketSettleIssue
	p.Desc("id").Find(sqls.DB(), &list)
	paging := p.Paging
	paging.Total = p.Count(sqls.DB(), &models.PredictMarketSettleIssue{})
	return web.JsonData(&web.PageResult{Results: list, Page: paging})
}

type polymarketOutcomeUpdateForm struct {
	MarketId          int64  `json:"marketId"`
	ExternalOutcomeId string `json:"externalOutcomeId"`
	Option            string `json:"option"`
	DisplayName       string `json:"displayName"`
	Locked            bool   `json:"locked"`
}

func (c *PolymarketController) PostOutcome_update() *web.JsonResult {
	adminUser := common.GetCurrentUser(c.Ctx)
	if adminUser == nil {
		return web.JsonError(errs.NotLogin())
	}
	var form polymarketOutcomeUpdateForm
	if err := c.Ctx.ReadJSON(&form); err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	if form.MarketId <= 0 {
		return web.JsonErrorMsg("marketId is required")
	}
	form.ExternalOutcomeId = strings.TrimSpace(form.ExternalOutcomeId)
	if form.ExternalOutcomeId == "" {
		return web.JsonErrorMsg("externalOutcomeId is required")
	}
	option := strings.ToUpper(strings.TrimSpace(form.Option))
	if option != services.PredictOptionA && option != services.PredictOptionB {
		return web.JsonErrorMsg("option must be A or B")
	}
	now := dates.NowTimestamp()
	var outcome models.PredictMarketOutcome
	if err := sqls.DB().Where("market_id = ? AND external_outcome_id = ?", form.MarketId, form.ExternalOutcomeId).First(&outcome).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	outcome.Option = option
	if strings.TrimSpace(form.DisplayName) != "" {
		outcome.DisplayName = strings.TrimSpace(form.DisplayName)
	}
	outcome.Locked = form.Locked
	outcome.UpdateTime = now
	if err := sqls.DB().Save(&outcome).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	desc := fmt.Sprintf("admin update polymarket outcome: marketId=%d externalOutcomeId=%s option=%s", form.MarketId, form.ExternalOutcomeId, option)
	services.OperateLogService.AddOperateLog(adminUser.Id, constants.OpTypeUpdate, "polymarketOutcome", outcome.Id, desc, c.Ctx.Request())
	return web.JsonData(outcome)
}

func (c *PolymarketController) PostTracking_retry() *web.JsonResult {
	adminUser := common.GetCurrentUser(c.Ctx)
	if adminUser == nil {
		return web.JsonError(errs.NotLogin())
	}
	marketId, _ := params.GetInt64(c.Ctx, "marketId")
	externalMarketId := strings.TrimSpace(c.Ctx.FormValue("externalMarketId"))
	if marketId <= 0 && externalMarketId == "" {
		return web.JsonErrorMsg("marketId or externalMarketId is required")
	}
	q := sqls.DB().Model(&models.PredictMarketTracking{})
	if marketId > 0 {
		q = q.Where("market_id = ?", marketId)
	}
	if externalMarketId != "" {
		q = q.Where("external_market_id = ?", externalMarketId)
	}
	now := dates.NowTimestamp()
	if err := q.Updates(map[string]any{
		"tracking_status": "TRACKING",
		"fail_count":      0,
		"last_error":      "",
		"next_retry_at":   0,
		"update_time":     now,
	}).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	services.OperateLogService.AddOperateLog(adminUser.Id, constants.OpTypeUpdate, "polymarketTracking", marketId, "admin retry polymarket tracking", c.Ctx.Request())
	return web.JsonSuccess()
}

func (c *PolymarketController) PostIssue_ignore() *web.JsonResult {
	adminUser := common.GetCurrentUser(c.Ctx)
	if adminUser == nil {
		return web.JsonError(errs.NotLogin())
	}
	issueId, _ := params.GetInt64(c.Ctx, "issueId")
	if issueId <= 0 {
		return web.JsonErrorMsg("issueId is required")
	}
	now := dates.NowTimestamp()
	if err := sqls.DB().Model(&models.PredictMarketSettleIssue{}).Where("id = ?", issueId).
		Updates(map[string]any{"status": "IGNORED", "update_time": now}).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	services.OperateLogService.AddOperateLog(adminUser.Id, constants.OpTypeUpdate, "polymarketIssue", issueId, "admin ignore polymarket issue", c.Ctx.Request())
	return web.JsonSuccess()
}
