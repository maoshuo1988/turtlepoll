package api

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/pkg/errs"
	"bbs-go/internal/services"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
)

type DislikeController struct {
	Ctx iris.Context
}

// PostCreate maps to POST /api/dislike/create
func (c *DislikeController) PostCreate() *web.JsonResult {
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
		err = services.UserDislikeService.TopicDislike(user.Id, entityId)
	default:
		return web.JsonErrorMsg("unsupported entityType")
	}

	if err != nil {
		return web.JsonError(err)
	}
	return web.JsonSuccess()
}

// PostCancle maps to POST /api/dislike/cancle
func (c *DislikeController) PostCancle() *web.JsonResult {
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
		err = services.UserDislikeService.TopicCancelDislike(user.Id, entityId)
	default:
		return web.JsonErrorMsg("unsupported entityType")
	}
	if err != nil {
		return web.JsonError(err)
	}
	return web.JsonSuccess()
}
