package admin

import (
	"bbs-go/internal/services"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
)

// MessageNotifyController 主站消息通知后台接口。
type MessageNotifyController struct {
	Ctx iris.Context
}

func (c *MessageNotifyController) BeforeActivation(b mvc.BeforeActivation) {
	b.Handle("GET", "/templates", "GetTemplates")
}

func (c *MessageNotifyController) GetTemplates() *web.JsonResult {
	businessCode := params.FormValue(c.Ctx, "businessCode")
	page := params.FormValueIntDefault(c.Ctx, "page", 1)
	limit := params.FormValueIntDefault(c.Ctx, "limit", 20)
	var status *int
	statusRaw := params.FormValue(c.Ctx, "status")
	if statusRaw != "" {
		v := params.FormValueIntDefault(c.Ctx, "status", 0)
		status = &v
	}
	list, paging := services.MessageNotifyService.FindTemplates(businessCode, status, page, limit)
	return web.JsonPageData(list, paging)
}
