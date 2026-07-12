package api

import (
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/pkg/errs"
	"bbs-go/internal/services"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
)

// MessageNotifyController 主站消息通知接口。
type MessageNotifyController struct {
	Ctx iris.Context
}

func (c *MessageNotifyController) BeforeActivation(b mvc.BeforeActivation) {
	b.Handle("GET", "/list", "GetList")
	b.Handle("GET", "/unread-count", "GetUnread_count")
	b.Handle("GET", "/by/{id}", "GetBy")
	b.Handle("POST", "/read", "PostRead")
}

func (c *MessageNotifyController) GetList() *web.JsonResult {
	user, err := common.CheckLogin(c.Ctx)
	if err != nil {
		return web.JsonError(errs.NotLogin())
	}
	businessCode := params.FormValue(c.Ctx, "businessCode")
	cursor, _ := params.GetInt64(c.Ctx, "cursor")
	limit, _ := params.GetInt(c.Ctx, "limit")
	var status *int
	statusRaw := params.FormValue(c.Ctx, "status")
	if statusRaw != "" {
		v, _ := params.GetInt(c.Ctx, "status")
		status = &v
	}
	ret, err := services.MessageNotifyService.List(user.Id, businessCode, status, cursor, limit)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(ret)
}

func (c *MessageNotifyController) GetUnread_count() *web.JsonResult {
	user, err := common.CheckLogin(c.Ctx)
	if err != nil {
		return web.JsonError(errs.NotLogin())
	}
	ret, err := services.MessageNotifyService.UnreadCount(user.Id)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(ret)
}

func (c *MessageNotifyController) GetBy(id int64) *web.JsonResult {
	user, err := common.CheckLogin(c.Ctx)
	if err != nil {
		return web.JsonError(errs.NotLogin())
	}
	ret, err := services.MessageNotifyService.Get(user.Id, id)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(ret)
}

func (c *MessageNotifyController) PostRead() *web.JsonResult {
	user, err := common.CheckLogin(c.Ctx)
	if err != nil {
		return web.JsonError(errs.NotLogin())
	}
	var form struct {
		Id int64 `json:"id"`
	}
	_ = c.Ctx.ReadJSON(&form)
	if form.Id <= 0 {
		form.Id, _ = params.GetInt64(c.Ctx, "id")
	}
	ret, err := services.MessageNotifyService.MarkRead(user.Id, form.Id)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(ret)
}
