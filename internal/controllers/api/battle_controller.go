package api

import (
	"bbs-go/internal/models"
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/pkg/errs"
	"bbs-go/internal/repositories"
	"bbs-go/internal/services"
	"strings"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
)

// BattleController 开战广场接口
// 路由：/api/battle（需要登录）
type BattleController struct {
	Ctx iris.Context
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
// - mine=1：只看我参与的（庄家或挑战者）
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

	db := repositories.BattleRepository.DB()

	var list []map[string]any
	var bs []*models.Battle
	queryDB := db.Model(&models.Battle{})
	if status != "" {
		queryDB = queryDB.Where("status = ?", status)
	}
	if mine == "1" {
		// 我是庄家或我有下注
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

	// 附带我对每个 battle 的动作（仅 challengers）
	for _, b := range bs {
		act := repositories.BattleChallengeActionRepository.TakeByUser(db, b.Id, user.Id)
		myAction := ""
		if act != nil {
			myAction = act.Action
		}
		list = append(list, map[string]any{"battle": b, "myAction": myAction})
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
