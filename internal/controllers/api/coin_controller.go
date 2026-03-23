package api

import (
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/pkg/errs"
	"bbs-go/internal/services"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
)

// CoinController 用户金币接口
// 路由：/api/coin（需要登录）
type CoinController struct {
	Ctx iris.Context
}

// 获取当前用户金币账户：GET /api/coin/me
func (c *CoinController) GetMe() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	uc, err := services.UserCoinService.GetOrCreate(user.Id)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(uc)
}

// 下注：POST /api/coin/bet
// 表单参数：marketId, option(A/B), amount
func (c *CoinController) PostBet() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	marketId, _ := params.GetInt64(c.Ctx, "marketId")
	option := params.FormValue(c.Ctx, "option")
	amount, _ := params.GetInt64(c.Ctx, "amount")

	res, err := services.PredictBetService.PlaceBet(user.Id, marketId, option, amount)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(res)
}

// 结算：POST /api/coin/settle
// 表单参数：marketId
// 说明：用户对自己下注过的预测市场进行结算，获得金币。
func (c *CoinController) PostSettle() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	marketId, _ := params.GetInt64(c.Ctx, "marketId")
	res, err := services.PredictSettleService.SettleMyBet(user.Id, marketId)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(map[string]any{
		"list":  res,
		"count": len(res),
	})
}
