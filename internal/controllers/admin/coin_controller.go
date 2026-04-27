package admin

import (
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/services"
	"strings"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
)

// CoinController 管理员金币接口
// 路由：/api/admin/coin（需要管理员权限）
type CoinController struct {
	Ctx iris.Context
}

// 流水列表：GET /api/admin/coin/log/list
// 查询参数：userId, bizType, startDate(YYYY-MM-DD), endDate(YYYY-MM-DD), page, limit
func (c *CoinController) AnyLogList() *web.JsonResult {
	query := params.NewQueryParams(c.Ctx).
		EqByReq("user_id").
		EqByReq("biz_type").
		PageByReq().
		Desc("id")

	startDate := strings.TrimSpace(c.Ctx.URLParam("startDate"))
	if startDate != "" {
		startTs, err := parseDateStart(startDate)
		if err != nil {
			return web.JsonErrorMsg("invalid startDate, expected YYYY-MM-DD")
		}
		query.Gte("create_time", startTs)
	}

	endDate := strings.TrimSpace(c.Ctx.URLParam("endDate"))
	if endDate != "" {
		endTs, err := parseDateEndExclusive(endDate)
		if err != nil {
			return web.JsonErrorMsg("invalid endDate, expected YYYY-MM-DD")
		}
		query.Lt("create_time", endTs)
	}

	list, paging := services.UserCoinService.FindLogPageByParams(query)
	return web.JsonData(&web.PageResult{Results: list, Page: paging})
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

func parseDateStart(dateStr string) (int64, error) {
	t, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
	if err != nil {
		return 0, err
	}
	return t.Unix(), nil
}

func parseDateEndExclusive(dateStr string) (int64, error) {
	t, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
	if err != nil {
		return 0, err
	}
	return t.AddDate(0, 0, 1).Unix(), nil
}
