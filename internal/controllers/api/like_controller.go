package api

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/models"
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/pkg/errs"
	"bbs-go/internal/services"
	"strings"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/common/strs"
	"github.com/mlogclub/simple/sqls"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
)

type LikeController struct {
	Ctx iris.Context
}

func (c *LikeController) PostLike() *web.JsonResult {
	var (
		entityType, _ = params.Get(c.Ctx, "entityType")
		entityId      = common.GetID(c.Ctx, "entityId")
		user          = common.GetCurrentUser(c.Ctx)
		err           error
	)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	if entityType == constants.EntityComment {
		if err := c.validatePredictCommentLike(entityId); err != nil {
			return c.mapPredictLikeError(err)
		}
	}
	switch entityType {
	case constants.EntityTopic:
		err = services.UserLikeService.TopicLike(user.Id, entityId)
	case constants.EntityArticle:
		err = services.UserLikeService.ArticleLike(user.Id, entityId)
	case constants.EntityComment:
		err = services.UserLikeService.CommentLike(user.Id, entityId)
	}
	if err != nil {
		return c.mapPredictLikeError(err)
	}
	return web.JsonSuccess()
}

func (c *LikeController) PostUnlike() *web.JsonResult {
	var (
		entityType, _ = params.Get(c.Ctx, "entityType")
		entityId      = common.GetID(c.Ctx, "entityId")
		user          = common.GetCurrentUser(c.Ctx)
		err           error
	)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	switch entityType {
	case constants.EntityTopic:
		err = services.UserLikeService.TopicUnLike(user.Id, entityId)
	case constants.EntityArticle:
		err = services.UserLikeService.ArticleUnLike(user.Id, entityId)
	case constants.EntityComment:
		err = services.UserLikeService.CommentUnLike(user.Id, entityId)
	}
	if err != nil {
		return web.JsonError(err)
	}
	return web.JsonSuccess()
}

func (c *LikeController) validatePredictCommentLike(commentId int64) error {
	if commentId <= 0 {
		return web.NewError(400420, "LIKE_400_COMMENT_ID_REQUIRED")
	}
	comment := services.CommentService.Get(commentId)
	if comment == nil || comment.Status != constants.StatusOk {
		return web.NewError(404420, "LIKE_404_COMMENT_NOT_FOUND")
	}
	if comment.EntityType == constants.EntityPredictMarket {
		market := &models.PredictMarket{}
		if err := sqls.DB().Take(market, "id = ?", comment.EntityId).Error; err != nil {
			return web.NewError(404404, "PREDICT_404_MARKET_NOT_FOUND")
		}
	}
	if comment.EntityType == constants.EntityComment {
		parent := services.CommentService.Get(comment.EntityId)
		if parent == nil {
			return web.NewError(404405, "COMMENT_404_PARENT_NOT_FOUND")
		}
		if parent.EntityType == constants.EntityPredictMarket {
			market := &models.PredictMarket{}
			if err := sqls.DB().Take(market, "id = ?", parent.EntityId).Error; err != nil {
				return web.NewError(404404, "PREDICT_404_MARKET_NOT_FOUND")
			}
		}
	}
	return nil
}

func (c *LikeController) mapPredictLikeError(err error) *web.JsonResult {
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

func (c *LikeController) GetLiked_ids() *web.JsonResult {
	var (
		user           = common.GetCurrentUser(c.Ctx)
		entityType     = params.FormValue(c.Ctx, "entityType")
		entityIds      = params.FormValueInt64Array(c.Ctx, "entityIds")
		likedEntityIds []int64
	)
	if user != nil {
		likedEntityIds = services.UserLikeService.IsLiked(user.Id, entityType, entityIds)
	}
	return web.JsonData(likedEntityIds)
}

func (c *LikeController) GetLiked() *web.JsonResult {
	var (
		user          = common.GetCurrentUser(c.Ctx)
		entityType, _ = params.Get(c.Ctx, "entityType")
		entityId      = common.GetID(c.Ctx, "entityId")
	)
	if user == nil || strs.IsBlank(entityType) || entityId <= 0 {
		return web.JsonData(false)
	} else {
		liked := services.UserLikeService.Exists(user.Id, entityType, entityId)
		return web.JsonData(liked)
	}
}
