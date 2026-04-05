package api

import (
	"bbs-go/internal/models"
	"bbs-go/internal/models/constants"
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/pkg/errs"
	"bbs-go/internal/repositories"
	"bbs-go/internal/services"
	"log/slog"
	"strings"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
)

// BattleController 开战广场接口
// 路由：/api/battle（需要登录）
type BattleController struct {
	Ctx iris.Context
}

// 赌局统计：GET /api/battle/stats
// 说明：返回未结算赌局数量、待结算（pending）赌局数量、这些未结算赌局的资金池本金总额、以及庄家去重数量。
func (c *BattleController) GetStats() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}

	db := repositories.BattleRepository.DB()

	// 未结算：status != settled
	var unsettledCount int64
	if err := db.Model(&models.Battle{}).Where("status <> ?", services.BattleStatusSettled).Count(&unsettledCount).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	// 等待结算：pending
	var pendingCount int64
	if err := db.Model(&models.Battle{}).Where("status = ?", services.BattleStatusPending).Count(&pendingCount).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	// 未结算赌局的资金池本金总额（poolPrincipalTotal）
	type sumRow struct {
		Sum int64 `gorm:"column:sum"`
	}
	var sr sumRow
	if err := db.Model(&models.Battle{}).
		Select("COALESCE(SUM(pool_principal_total), 0) as sum").
		Where("status <> ?", services.BattleStatusSettled).
		Scan(&sr).Error; err != nil {
		slog.Error("load battle pool sum failed", slog.Any("err", err))
		return web.JsonErrorMsg(err.Error())
	}

	// 未结算赌局的庄家去重数
	var bankerCount int64
	if err := db.Model(&models.Battle{}).
		Where("status <> ?", services.BattleStatusSettled).
		Distinct("banker_user_id").
		Count(&bankerCount).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	return web.JsonData(map[string]any{
		"unsettledCount": unsettledCount,
		"pendingCount":   pendingCount,
		"poolTotal":      sr.Sum,
		"bankerCount":    bankerCount,
	})
}

// 赌局详情：GET /api/battle/by?battleId=1
// 说明：返回 battle + 当前用户 challenger 动作（confirm/dispute）+ 结算单/我的结算明细（如已生成）。
func (c *BattleController) GetBy() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	battleId, _ := params.GetInt64(c.Ctx, "battleId")
	if battleId <= 0 {
		return web.JsonErrorMsg("battleId is required")
	}

	db := repositories.BattleRepository.DB()
	b := repositories.BattleRepository.Take(db, "id = ?", battleId)
	if b == nil {
		return web.JsonErrorMsg("battle not found")
	}

	// 当前用户动作（仅挑战者会有）
	act := repositories.BattleChallengeActionRepository.TakeByUser(db, b.Id, user.Id)
	myAction := ""
	if act != nil {
		myAction = act.Action
	}

	// 结算单（如已生成）
	st := repositories.BattleSettlementRepository.TakeByBattleId(db, b.Id)
	var myItem any
	if st != nil {
		it := repositories.BattleSettlementRepository.TakeItemByBattleUser(db, b.Id, user.Id)
		if it != nil {
			myItem = it
		}
	}

	return web.JsonData(map[string]any{
		"battle":   b,
		"myAction": myAction,
		"settlement": map[string]any{
			"settlement": st,
			"myItem":     myItem,
		},
	})
}

// 赌局列表：GET /api/battle/list?page=1&pageSize=20&status=open
// 说明：
// - status 可选：open/sealed/pending/disputed/settled
// - mine=1：只看我参与的（庄家或挑战者）【兼容旧参数】
// - role=banker：只看我做庄的
// - role=challenger：只看我挑战的（我作为 challenger 下过注）
func (c *BattleController) GetList() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	page, _ := params.GetInt(c.Ctx, "page")
	pageSize, _ := params.GetInt(c.Ctx, "pageSize")
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	status := strings.TrimSpace(params.FormValue(c.Ctx, "status"))
	mine := strings.TrimSpace(params.FormValue(c.Ctx, "mine"))
	role := strings.TrimSpace(strings.ToLower(params.FormValue(c.Ctx, "role")))

	db := repositories.BattleRepository.DB()

	var list []map[string]any
	var bs []*models.Battle
	queryDB := db.Model(&models.Battle{})
	if status != "" {
		queryDB = queryDB.Where("status = ?", status)
	}
	// 参与维度筛选：
	// - role 优先级高于 mine（mine 为历史兼容参数）
	// - role=challenger：只看我作为 challenger 下注过的 battle（不包含我做庄的）
	if role == "banker" {
		queryDB = queryDB.Where("banker_user_id = ?", user.Id)
	} else if role == "challenger" {
		sub := db.Model(&models.BattleBet{}).Select("battle_id").
			Where("user_id = ? AND role = ?", user.Id, "challenger")
		queryDB = queryDB.Where("id IN (?)", sub)
	} else if mine == "1" {
		// 我是庄家或我有下注（包含 banker/challenger 两种 bet）
		sub := db.Model(&models.BattleBet{}).Select("battle_id").Where("user_id = ?", user.Id)
		queryDB = queryDB.Where("banker_user_id = ? OR id IN (?)", user.Id, sub)
	}

	var count int64
	if err := queryDB.Count(&count).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	if err := queryDB.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&bs).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	// 批量附带：庄家昵称、评论数、点赞数（避免 N+1）
	bankerIds := make([]int64, 0, len(bs))
	battleIds := make([]int64, 0, len(bs))
	bankerIdSet := make(map[int64]struct{}, len(bs))
	for _, b := range bs {
		battleIds = append(battleIds, b.Id)
		if _, ok := bankerIdSet[b.BankerUserId]; !ok {
			bankerIdSet[b.BankerUserId] = struct{}{}
			bankerIds = append(bankerIds, b.BankerUserId)
		}
	}

	bankerNicknames := map[int64]string{}
	if len(bankerIds) > 0 {
		type row struct {
			Id       int64  `gorm:"column:id"`
			Nickname string `gorm:"column:nickname"`
		}
		var rows []row
		if err := db.Model(&models.User{}).Select("id, nickname").Where("id IN ?", bankerIds).Find(&rows).Error; err != nil {
			slog.Error("load banker nicknames failed", slog.Any("err", err))
		} else {
			for _, r := range rows {
				bankerNicknames[r.Id] = r.Nickname
			}
		}
	}

	commentCounts := map[int64]int64{}
	likeCounts := map[int64]int64{}
	if len(battleIds) > 0 {
		// 评论数
		type ccRow struct {
			EntityId int64 `gorm:"column:entity_id"`
			Cnt      int64 `gorm:"column:cnt"`
		}
		var ccRows []ccRow
		if err := db.Model(&models.Comment{}).
			Select("entity_id, count(1) as cnt").
			Where("entity_type = ? AND entity_id IN ?", constants.EntityBattle, battleIds).
			Group("entity_id").
			Find(&ccRows).Error; err != nil {
			slog.Error("load battle comment counts failed", slog.Any("err", err))
		} else {
			for _, r := range ccRows {
				commentCounts[r.EntityId] = r.Cnt
			}
		}

		// 点赞数
		type lcRow struct {
			EntityId int64 `gorm:"column:entity_id"`
			Cnt      int64 `gorm:"column:cnt"`
		}
		var lcRows []lcRow
		if err := db.Model(&models.UserLike{}).
			Select("entity_id, count(1) as cnt").
			Where("entity_type = ? AND entity_id IN ?", constants.EntityBattle, battleIds).
			Group("entity_id").
			Find(&lcRows).Error; err != nil {
			slog.Error("load battle like counts failed", slog.Any("err", err))
		} else {
			for _, r := range lcRows {
				likeCounts[r.EntityId] = r.Cnt
			}
		}
	}

	// 附带我对每个 battle 的动作（仅 challengers）
	for _, b := range bs {
		act := repositories.BattleChallengeActionRepository.TakeByUser(db, b.Id, user.Id)
		myAction := ""
		if act != nil {
			myAction = act.Action
		}
		list = append(list, map[string]any{
			"battle":         b,
			"myAction":       myAction,
			"bankerNickname": bankerNicknames[b.BankerUserId],
			"commentCount":   commentCounts[b.Id],
			"likeCount":      likeCounts[b.Id],
		})
	}

	return web.JsonData(map[string]any{
		"list":     list,
		"count":    count,
		"page":     page,
		"pageSize": pageSize,
	})
}

// 创建赌局：POST /api/battle/create
func (c *BattleController) PostCreate() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	var form services.CreateBattleForm
	if err := c.Ctx.ReadJSON(&form); err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	b, err := services.BattleService.CreateBattle(user.Id, form)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(b)
}

// 加入/追加下注：POST /api/battle/join
func (c *BattleController) PostJoin() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	var form services.JoinBattleForm
	if err := c.Ctx.ReadJSON(&form); err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	b, bet, err := services.BattleService.JoinOrAddStake(user.Id, form)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(map[string]any{"battle": b, "bet": bet})
}

// 庄家加注：POST /api/battle/banker_add_stake
func (c *BattleController) PostBanker_add_stake() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	var form services.BankerAddStakeForm
	if err := c.Ctx.ReadJSON(&form); err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	b, err := services.BattleService.BankerAddStake(user.Id, form)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(b)
}

// 庄家宣布结果（一期直接 settled 并生成结算单）：POST /api/battle/declare
// 参数：battleId, result(banker_wins/banker_loses)
func (c *BattleController) PostDeclare() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	battleId, _ := params.GetInt64(c.Ctx, "battleId")
	result := params.FormValue(c.Ctx, "result")
	b, err := services.BattleService.DeclareResultByBanker(user.Id, battleId, result)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(b)
}

// 提取（一次性全提）：POST /api/battle/withdraw
func (c *BattleController) PostWithdraw() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	var form services.WithdrawForm
	if err := c.Ctx.ReadJSON(&form); err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	item, err := services.BattleService.Withdraw(user.Id, form)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(item)
}

// 挑战者确认：POST /api/battle/challenger_confirm
func (c *BattleController) PostChallenger_confirm() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	var form services.ChallengeActionForm
	if err := c.Ctx.ReadJSON(&form); err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	b, err := services.BattleService.ChallengeConfirm(user.Id, form)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(b)
}

// 挑战者异议：POST /api/battle/challenger_dispute
func (c *BattleController) PostChallenger_dispute() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	var form services.ChallengeActionForm
	if err := c.Ctx.ReadJSON(&form); err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	b, err := services.BattleService.ChallengeDispute(user.Id, form)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(b)
}

// AdminBattleController 管理员仲裁接口
// 路由：/api/admin/battle
type AdminBattleController struct {
	Ctx iris.Context
}

type battleTrendItem struct {
	Day   string `json:"day"`
	Count int64  `json:"count"`
}

// 管理员看板：赌局趋势（每日新增赌局数）：GET /api/admin/battle/trends?range=7d
func (c *AdminBattleController) GetTrends() *web.JsonResult {
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
		Model(&models.Battle{}).
		Select("to_char(date_trunc('day', to_timestamp(create_time)), 'YYYY-MM-DD') as day, count(1) as count").
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
	list := make([]battleTrendItem, 0, days)
	for i := 0; i < days; i++ {
		daySec := startSec + int64(i*86400)
		dayStr := time.Unix(daySec, 0).UTC().Format("2006-01-02")
		list = append(list, battleTrendItem{Day: dayStr, Count: m[dayStr]})
	}

	return web.JsonData(map[string]any{
		"range": rangeStr,
		"days":  days,
		"list":  list,
	})
}

type battleActiveUserItem struct {
	Day             string `json:"day"`
	ActiveUserCount int64  `json:"activeUserCount"`
}

// 管理员看板：活跃下注用户（按天去重）：GET /api/admin/battle/active_users?range=7d
// 说明：按 BattleBet.create_time（秒级）统计每日下注过的去重用户数。
func (c *AdminBattleController) GetActive_users() *web.JsonResult {
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
		Model(&models.BattleBet{}).
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
	list := make([]battleActiveUserItem, 0, days)
	for i := 0; i < days; i++ {
		daySec := startSec + int64(i*86400)
		dayStr := time.Unix(daySec, 0).UTC().Format("2006-01-02")
		list = append(list, battleActiveUserItem{Day: dayStr, ActiveUserCount: m[dayStr]})
	}

	return web.JsonData(map[string]any{
		"range": rangeStr,
		"days":  days,
		"list":  list,
	})
}

// 管理员裁决：POST /api/admin/battle/resolve
func (c *AdminBattleController) PostResolve() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	var form services.AdminResolveForm
	if err := c.Ctx.ReadJSON(&form); err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	b, err := services.BattleService.AdminResolve(user.Id, form)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(b)
}

// 手动触发赌局轮巡：POST /api/admin/battle/cron_tick
// 说明：仅管理员可调用（由 AdminMiddleware 控制），用于人工触发 open->sealed、sealed->pending 等状态迁移。
func (c *AdminBattleController) PostCron_tick() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	if err := services.BattleService.CronTick(); err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonSuccess()
}
