package api

import (
	"bbs-go/internal/controllers/render"
	"bbs-go/internal/models/req"
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/pkg/errs"
	"bbs-go/internal/services"
	"bbs-go/internal/spam"
	"strconv"
	"strings"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
)

// PKController 对立PK用户端接口。
// 路由：/api/pk（需要登录）
type PKController struct {
	Ctx iris.Context
}

func (c *PKController) BeforeActivation(b mvc.BeforeActivation) {
	b.Handle("POST", "/settle", "PostSettle")
	b.Handle("GET", "/heat/rank", "GetHeatRank")
	b.Handle("GET", "/heat/me", "GetHeatMe")
	b.Handle("GET", "/odds/current", "GetOddsCurrent")
	b.Handle("GET", "/comment/replies", "GetCommentReplies")
	b.Handle("POST", "/recordOption", "PostRecordOption")
	b.Handle("POST", "/like", "PostLike")
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
		form.Amount, _ = strconv.ParseInt(strings.TrimSpace(params.FormValue(c.Ctx, "amount")), 10, 64)
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
	if strings.TrimSpace(side) == "" {
		side = form.Option
	}
	if err := spam.CheckComment(user, form); err != nil {
		return web.JsonError(err)
	}
	comment, heat, optionAtAction, err := services.PKService.CreateComment(user.Id, form, topicId, side)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(map[string]any{"comment": render.BuildComment(comment), "heat": heat, "optionAtAction": optionAtAction})
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
	comment, heat, optionAtAction, err := services.PKService.ReplyComment(user.Id, form, commentId)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(map[string]any{"comment": render.BuildComment(comment), "heat": heat, "optionAtAction": optionAtAction})
}

func (c *PKController) GetCommentReplies() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	userId := int64(0)
	if user != nil {
		userId = user.Id
	}
	commentId, _ := params.GetInt64(c.Ctx, "commentId")
	cursor, _ := params.GetInt64(c.Ctx, "cursor")
	pageSize, _ := params.GetInt(c.Ctx, "pageSize")
	list, nextCursor, hasMore, err := services.PKService.CommentReplies(commentId, cursor, pageSize, userId)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonCursorData(list, strconv.FormatInt(nextCursor, 10), hasMore)
}

func (c *PKController) PostSettle() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	var form services.PKSettleForm
	if err := c.Ctx.ReadJSON(&form); err != nil {
		form.TopicId, _ = params.GetInt64(c.Ctx, "topicId")
		form.RoundId, _ = params.GetInt64(c.Ctx, "roundId")
		form.RequestId = params.FormValue(c.Ctx, "requestId")
		form.SnapshotType = params.FormValue(c.Ctx, "snapshotType")
		form.FreezeSource = params.FormValue(c.Ctx, "freezeSource")
	}
	data, err := services.PKService.SettleForUser(user.Id, form)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(data)
}

func (c *PKController) GetHeatRank() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	userId := int64(0)
	if user != nil {
		userId = user.Id
	}
	topicId, _ := params.GetInt64(c.Ctx, "topicId")
	roundId, _ := params.GetInt64(c.Ctx, "roundId")
	scope := params.FormValue(c.Ctx, "scope")
	page, _ := params.GetInt(c.Ctx, "page")
	pageSize, _ := params.GetInt(c.Ctx, "pageSize")
	data, err := services.PKService.HeatRank(topicId, roundId, scope, userId, page, pageSize)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(data)
}

func (c *PKController) GetHeatMe() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	topicId, _ := params.GetInt64(c.Ctx, "topicId")
	roundId, _ := params.GetInt64(c.Ctx, "roundId")
	data, err := services.PKService.HeatMe(user.Id, topicId, roundId)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(data)
}

func (c *PKController) GetOddsCurrent() *web.JsonResult {
	topicId, _ := params.GetInt64(c.Ctx, "topicId")
	roundId, _ := params.GetInt64(c.Ctx, "roundId")
	data, err := services.PKService.OddsCurrent(topicId, roundId)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(data)
}

func (c *PKController) PostRecordOption() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	var form services.PKRecordOptionForm
	if err := c.Ctx.ReadJSON(&form); err != nil {
		form.TopicId, _ = params.GetInt64(c.Ctx, "topicId")
		form.RoundId, _ = params.GetInt64(c.Ctx, "roundId")
		form.Option = params.FormValue(c.Ctx, "option")
		form.ActionType = params.FormValue(c.Ctx, "actionType")
		form.RequestId = params.FormValue(c.Ctx, "requestId")
		form.EntityType = params.FormValue(c.Ctx, "entityType")
		form.EntityId, _ = params.GetInt64(c.Ctx, "entityId")
	}
	data, err := services.PKService.RecordOption(user.Id, form)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(data)
}

func (c *PKController) PostLike() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	commentId, _ := params.GetInt64(c.Ctx, "commentId")
	requestId := params.FormValue(c.Ctx, "requestId")
	data, err := services.PKService.LikeComment(user.Id, commentId, requestId)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(data)
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
	status := params.FormValue(c.Ctx, "status")
	data, err := services.PKService.MyBets(user.Id, page, pageSize, status)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(data)
}
