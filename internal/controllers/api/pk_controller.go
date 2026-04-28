package api

import (
	"bbs-go/internal/controllers/render"
	"bbs-go/internal/models/req"
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/pkg/errs"
	"bbs-go/internal/services"
	"bbs-go/internal/spam"
	"strconv"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
)

// PKController 对立PK用户端接口。
// 路由：/api/pk（需要登录）
type PKController struct {
	Ctx iris.Context
}

func (c *PKController) GetTopics() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	userId := int64(0)
	if user != nil {
		userId = user.Id
	}
	page, _ := params.GetInt(c.Ctx, "page")
	pageSize, _ := params.GetInt(c.Ctx, "pageSize")
	data, err := services.PKService.ListTopics(page, pageSize, userId)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(data)
}

func (c *PKController) GetTopic() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	userId := int64(0)
	if user != nil {
		userId = user.Id
	}
	topicId, _ := params.GetInt64(c.Ctx, "topicId")
	slug := params.FormValue(c.Ctx, "slug")
	data, err := services.PKService.TopicDetail(topicId, slug, userId)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(data)
}

func (c *PKController) PostBet() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	var form services.PKBetForm
	if err := c.Ctx.ReadJSON(&form); err != nil {
		form.TopicId, _ = params.GetInt64(c.Ctx, "topicId")
		form.Side = params.FormValue(c.Ctx, "side")
		form.RequestId = params.FormValue(c.Ctx, "requestId")
	}
	data, err := services.PKService.PlaceBet(user.Id, form)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(data)
}

func (c *PKController) GetHeat() *web.JsonResult {
	topicId, _ := params.GetInt64(c.Ctx, "topicId")
	data, err := services.PKService.Heat(topicId)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(data)
}

func (c *PKController) GetComments() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	userId := int64(0)
	if user != nil {
		userId = user.Id
	}
	topicId, _ := params.GetInt64(c.Ctx, "topicId")
	side := params.FormValue(c.Ctx, "side")
	cursor, _ := params.GetInt64(c.Ctx, "cursor")
	sort := params.FormValue(c.Ctx, "sort")
	list, nextCursor, hasMore, err := services.PKService.Comments(topicId, side, cursor, sort, userId)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonCursorData(list, strconv.FormatInt(nextCursor, 10), hasMore)
}

func (c *PKController) PostCommentCreate() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	if err := services.UserService.CheckPostStatus(user); err != nil {
		return web.JsonError(err)
	}
	topicId, _ := params.GetInt64(c.Ctx, "topicId")
	side := params.FormValue(c.Ctx, "side")
	form := req.GetCreateCommentForm(c.Ctx)
	if err := spam.CheckComment(user, form); err != nil {
		return web.JsonError(err)
	}
	comment, heat, err := services.PKService.CreateComment(user.Id, form, topicId, side)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(map[string]any{"comment": render.BuildComment(comment), "heat": heat})
}

func (c *PKController) PostCommentReply() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	if err := services.UserService.CheckPostStatus(user); err != nil {
		return web.JsonError(err)
	}
	commentId, _ := params.GetInt64(c.Ctx, "commentId")
	form := req.GetCreateCommentForm(c.Ctx)
	form.EntityId = commentId
	if err := spam.CheckComment(user, form); err != nil {
		return web.JsonError(err)
	}
	comment, heat, err := services.PKService.ReplyComment(user.Id, form, commentId)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(map[string]any{"comment": render.BuildComment(comment), "heat": heat})
}

func (c *PKController) PostDownvote() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	var form services.PKDownvoteForm
	if err := c.Ctx.ReadJSON(&form); err != nil {
		form.CommentId, _ = params.GetInt64(c.Ctx, "commentId")
		form.RequestId = params.FormValue(c.Ctx, "requestId")
	}
	data, err := services.PKService.Downvote(user.Id, form)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(data)
}

func (c *PKController) GetHistory() *web.JsonResult {
	topicId, _ := params.GetInt64(c.Ctx, "topicId")
	page, _ := params.GetInt(c.Ctx, "page")
	pageSize, _ := params.GetInt(c.Ctx, "pageSize")
	data, err := services.PKService.History(topicId, page, pageSize)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(data)
}

func (c *PKController) GetSeasons() *web.JsonResult {
	topicId, _ := params.GetInt64(c.Ctx, "topicId")
	page, _ := params.GetInt(c.Ctx, "page")
	pageSize, _ := params.GetInt(c.Ctx, "pageSize")
	data, err := services.PKService.Seasons(topicId, page, pageSize)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(data)
}

func (c *PKController) GetMyBets() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	page, _ := params.GetInt(c.Ctx, "page")
	pageSize, _ := params.GetInt(c.Ctx, "pageSize")
	data, err := services.PKService.MyBets(user.Id, page, pageSize)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(data)
}
