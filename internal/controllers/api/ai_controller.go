package api

import (
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/pkg/errs"
	"bbs-go/internal/services"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
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

// GET /api/ai/stamina
func (c *AIController) GetStamina() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	ret, err := services.AIStaminaService.GetStatus(user.Id)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(ret)
}

// POST /api/ai/stamina/apple
func (c *AIController) PostStaminaApple() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	count := params.FormValueIntDefault(c.Ctx, "count", 0)
	if count <= 0 {
		var form struct {
			Count int `json:"count"`
		}
		_ = c.Ctx.ReadJSON(&form)
		count = form.Count
	}
	ret, err := services.AIStaminaService.RecoverByApple(user.Id, count)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(ret)
}

// GET /api/ai/pushes/unread
func (c *AIController) GetPushes_unread() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	limit := params.FormValueIntDefault(c.Ctx, "limit", 20)
	ret, err := services.AIPushService.ListUnread(user.Id, limit)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(ret)
}

// POST /api/ai/pushes/read
func (c *AIController) PostPushes_read() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	var form struct {
		Ids []int64 `json:"ids"`
	}
	_ = c.Ctx.ReadJSON(&form)
	ret, err := services.AIPushService.MarkRead(user.Id, form.Ids)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(ret)
}

// POST /api/ai/presence
func (c *AIController) PostPresence() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	var form services.AIPresenceForm
	if err := c.Ctx.ReadJSON(&form); err != nil {
		form.Page = params.FormValue(c.Ctx, "page")
		form.Active = params.FormValueBoolDefault(c.Ctx, "active", true)
	}
	ret, err := services.AIPushService.UpdatePresence(user.Id, form)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(ret)
}

// GET /api/ai/pushes/stream
func (c *AIController) GetPushes_stream() {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		c.Ctx.StatusCode(iris.StatusUnauthorized)
		return
	}
	lastEventId, _ := strconv.ParseInt(c.Ctx.GetHeader("Last-Event-ID"), 10, 64)
	ch, unregister := services.AIPushService.Stream(user.Id, lastEventId)
	defer unregister()

	c.Ctx.ContentType("text/event-stream")
	c.Ctx.Header("Cache-Control", "no-cache")
	c.Ctx.Header("Connection", "keep-alive")
	c.Ctx.Header("X-Accel-Buffering", "no")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	writer := c.Ctx.ResponseWriter()
	_, _ = fmt.Fprint(writer, ": connected\n\n")
	writer.Flush()

	for {
		select {
		case <-c.Ctx.Request().Context().Done():
			return
		case <-ticker.C:
			_, _ = fmt.Fprint(writer, ": ping\n\n")
			writer.Flush()
		case msg, ok := <-ch:
			if !ok {
				return
			}
			payload, _ := json.Marshal(msg)
			_, _ = fmt.Fprintf(writer, "id: %d\nevent: ai_push\ndata: %s\n\n", msg.Id, payload)
			writer.Flush()
		}
	}
}
