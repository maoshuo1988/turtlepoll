package admin

import (
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/services"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
)

// CoinController 管理员金币接口
// 路由：/api/admin/coin（需要管理员权限）
type CoinController struct {
	Ctx iris.Context
}

// 铸币：POST /api/admin/coin/mint
// 表单参数：userId, amount, remark
func (c *CoinController) PostMint() *web.JsonResult {
	adminUser := common.GetCurrentUser(c.Ctx)
	userId, _ := params.GetInt64(c.Ctx, "userId")
	amount, _ := params.GetInt64(c.Ctx, "amount")
	remark := params.FormValue(c.Ctx, "remark")

	uc, err := services.UserCoinService.Mint(adminUser.Id, userId, amount, remark)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(uc)
}
