package api

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/models"
	"bbs-go/internal/models/req"
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/pkg/errs"
	"bbs-go/internal/spam"
	"strconv"
	"strings"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/sqls"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"

	"bbs-go/internal/controllers/render"
	"bbs-go/internal/services"
)

type CommentController struct {
	Ctx iris.Context
}

func (c *CommentController) GetComments() *web.JsonResult {
	var (
		cursor, _     = params.GetInt64(c.Ctx, "cursor")
		entityType, _ = params.Get(c.Ctx, "entityType")
		entityId      = common.GetID(c.Ctx, "entityId")
		currentUser   = common.GetCurrentUser(c.Ctx)
	)
	comments, cursor, hasMore := services.CommentService.GetComments(entityType, entityId, cursor)
	return web.JsonCursorData(render.BuildComments(comments, currentUser, true, false), strconv.FormatInt(cursor, 10), hasMore)
}

func (c *CommentController) GetReplies() *web.JsonResult {
	var (
		cursor, _    = params.GetInt64(c.Ctx, "cursor")
		commentId, _ = params.GetInt64(c.Ctx, "commentId")
	)
	currentUser := common.GetCurrentUser(c.Ctx)
	comments, cursor, hasMore := services.CommentService.GetReplies(commentId, cursor, 10)
	return web.JsonCursorData(render.BuildComments(comments, currentUser, false, true), strconv.FormatInt(cursor, 10), hasMore)
}

func (c *CommentController) PostCreate() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	if err := services.UserService.CheckPostStatus(user); err != nil {
		return web.JsonError(err)
	}
	form := req.GetCreateCommentForm(c.Ctx)
	if err := c.validatePredictCommentForm(form); err != nil {
		return c.mapPredictError(err)
	}
	if err := spam.CheckComment(user, form); err != nil {
		return web.JsonError(err)
	}

	comment, err := services.CommentService.Publish(user.Id, form)
	if err != nil {
		return c.mapPredictError(err)
	}

	return web.JsonData(render.BuildComment(comment))
}

func (c *CommentController) PostReply() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	if err := services.UserService.CheckPostStatus(user); err != nil {
		return web.JsonError(err)
	}
	commentId, _ := params.GetInt64(c.Ctx, "commentId")
	form := req.GetCreateCommentForm(c.Ctx)
	form.EntityType = constants.EntityComment
	form.EntityId = commentId
	if err := c.validatePredictCommentForm(form); err != nil {
		return c.mapPredictError(err)
	}
	if err := spam.CheckComment(user, form); err != nil {
		return web.JsonError(err)
	}
	comment, err := services.CommentService.Publish(user.Id, form)
	if err != nil {
		return c.mapPredictError(err)
	}

	return web.JsonData(render.BuildComment(comment))
}

func (c *CommentController) validatePredictCommentForm(form req.CreateCommentForm) error {
	if form.EntityType == constants.EntityPredictMarket {
		if form.EntityId <= 0 {
			return web.NewError(400400, "PREDICT_400_MARKET_ID_REQUIRED")
		}
		market := &models.PredictMarket{}
		if err := sqls.DB().Take(market, "id = ?", form.EntityId).Error; err != nil {
			return web.NewError(404404, "PREDICT_404_MARKET_NOT_FOUND")
		}
		option := strings.ToUpper(strings.TrimSpace(form.Option))
		if option == "" {
			return web.NewError(422400, "TEAR_CAMP_OPTION_REQUIRED")
		}
		if !services.IsValidPredictOption(market.MarketType, option) {
			return web.NewError(422401, "PREDICT_422_OPTION_INVALID")
		}
	}

	if form.EntityType == constants.EntityComment {
		parent := services.CommentService.Get(form.EntityId)
		if parent == nil {
			return web.NewError(404405, "COMMENT_404_PARENT_NOT_FOUND")
		}
		if parent.EntityType == constants.EntityPredictMarket {
			market := &models.PredictMarket{}
			if err := sqls.DB().Take(market, "id = ?", parent.EntityId).Error; err != nil {
				return web.NewError(404404, "PREDICT_404_MARKET_NOT_FOUND")
			}
			option := strings.ToUpper(strings.TrimSpace(form.Option))
			if option != "" && !services.IsValidPredictOption(market.MarketType, option) {
				return web.NewError(422401, "PREDICT_422_OPTION_INVALID")
			}
		}
	}
	return nil
}

func (c *CommentController) mapPredictError(err error) *web.JsonResult {
	if err == nil {
		return web.JsonSuccess()
	}
	msg := strings.TrimSpace(err.Error())
	switch msg {
	case "TEAR_CAMP_LOCKED_BY_BET":
		return web.JsonError(web.NewError(423001, msg))
	case "TEAR_CAMP_CONFLICT":
		return web.JsonError(web.NewError(409001, msg))
	case "TEAR_CAMP_OPTION_REQUIRED":
		return web.JsonError(web.NewError(422400, msg))
	case "unsupported event type":
		return web.JsonError(web.NewError(422510, "HEAT_422_EVENT_TYPE_UNSUPPORTED"))
	case "db is required", "tx is required":
		return web.JsonError(web.NewError(500510, "HEAT_500_DB_REQUIRED"))
	case "eventId/topicId/roundId are required", "eventId and roundId are required":
		return web.JsonError(web.NewError(400510, "HEAT_400_REQUIRED_FIELDS_MISSING"))
	default:
		return web.JsonError(err)
	}
}
