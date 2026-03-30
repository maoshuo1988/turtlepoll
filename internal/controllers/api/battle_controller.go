package api

import (
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/pkg/errs"
	"bbs-go/internal/services"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
)

// BattleController 开战广场接口
// 路由：/api/battle（需要登录）
type BattleController struct {
	Ctx iris.Context
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
