package api

import (
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/pkg/errs"
	"bbs-go/internal/services"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/web"
)

// AIController 用户 AI 聊天接口。
// 路由：/api/ai（需要登录）
type AIController struct {
	Ctx iris.Context
}

// POST /api/ai/chat
func (c *AIController) PostChat() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}

	var form services.AIChatForm
	if err := c.Ctx.ReadJSON(&form); err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	ret, err := services.AIChatService.Chat(c.Ctx.Request().Context(), user.Id, form)
	if err != nil {
		if ret != nil {
			return web.JsonData(ret)
		}
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(ret)
}
