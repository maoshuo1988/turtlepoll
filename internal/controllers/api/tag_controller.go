package api

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/models"
	"bbs-go/internal/pkg/common"
	"strings"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"

	"bbs-go/internal/cache"
	"bbs-go/internal/controllers/render"
	"bbs-go/internal/services"
)

type TagController struct {
	Ctx iris.Context
}

func (c *TagController) PostCreate() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if err := services.UserService.CheckPostStatus(user); err != nil {
		return web.JsonError(err)
	}

	name := strings.TrimSpace(c.Ctx.PostValue("name"))
	description := strings.TrimSpace(c.Ctx.PostValue("description"))
	if name == "" {
		return web.JsonErrorMsg("tag name is required")
	}
	if len(name) > 32 {
		return web.JsonErrorMsg("tag name length must be <= 32")
	}
	if len(description) > 1024 {
		return web.JsonErrorMsg("tag description length must be <= 1024")
	}

	tag := services.TagService.GetByName(name)
	if tag != nil {
		if tag.Status != constants.StatusOk || tag.Description != description {
			tag.Status = constants.StatusOk
			tag.Description = description
			tag.UpdateTime = dates.NowTimestamp()
			if err := services.TagService.Update(tag); err != nil {
				return web.JsonError(err)
			}
		}
		return web.JsonData(render.BuildTag(tag))
	}

	tag = &models.Tag{
		Name:        name,
		Description: description,
		Status:      constants.StatusOk,
		CreateTime:  dates.NowTimestamp(),
		UpdateTime:  dates.NowTimestamp(),
	}
	if err := services.TagService.Create(tag); err != nil {
		return web.JsonError(err)
	}
	return web.JsonData(render.BuildTag(tag))
}

// 标签详情
func (c *TagController) GetBy(tagId int64) *web.JsonResult {
	tag := cache.TagCache.Get(tagId)
	if tag == nil {
		return web.JsonErrorMsg("tag not found")
	}
	return web.JsonData(render.BuildTag(tag))
}

// 标签列表
func (c *TagController) GetTags() *web.JsonResult {
	page := params.FormValueIntDefault(c.Ctx, "page", 1)
	limit := params.FormValueIntDefault(c.Ctx, "limit", 20)
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	keyword := strings.TrimSpace(c.Ctx.FormValue("keyword"))

	cnd := sqls.NewCnd().Eq("status", constants.StatusOk)
	if keyword != "" {
		cnd = cnd.Where("name like ?", "%"+keyword+"%")
	}
	tags, paging := services.TagService.FindPageByCnd(cnd.Page(page, limit).Desc("id"))

	return web.JsonPageData(render.BuildTags(tags), paging)
}

// 标签自动完成
func (c *TagController) PostAutocomplete() *web.JsonResult {
	input := c.Ctx.FormValue("input")
	tags := services.TagService.Autocomplete(input)
	return web.JsonData(tags)
}
