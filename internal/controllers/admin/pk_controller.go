package admin

import (
	"bbs-go/internal/services"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
)

// PKController 对立PK管理端接口。
// 路由：/api/admin/pk（需要管理员权限）
type PKController struct {
	Ctx iris.Context
}

func (c *PKController) GetTopicList() *web.JsonResult {
	page, _ := params.GetInt(c.Ctx, "page")
	pageSize, _ := params.GetInt(c.Ctx, "pageSize")
	status := params.FormValue(c.Ctx, "status")
	q := params.FormValue(c.Ctx, "q")
	data, err := services.PKService.AdminListTopics(page, pageSize, status, q)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(data)
}

func (c *PKController) PostTopicSave() *web.JsonResult {
	var form services.PKTopicSaveForm
	if err := c.Ctx.ReadJSON(&form); err != nil {
		form.Id, _ = params.GetInt64(c.Ctx, "id")
		form.Slug = params.FormValue(c.Ctx, "slug")
		form.Title = params.FormValue(c.Ctx, "title")
		form.SideAName = params.FormValue(c.Ctx, "sideAName")
		form.SideBName = params.FormValue(c.Ctx, "sideBName")
		form.Status = params.FormValue(c.Ctx, "status")
		form.Sort, _ = params.GetInt(c.Ctx, "sort")
		form.Cover = params.FormValue(c.Ctx, "cover")
	}
	topic, err := services.PKService.SaveTopic(form)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(topic)
}

func (c *PKController) PostTopicStatus() *web.JsonResult {
	topicId, _ := params.GetInt64(c.Ctx, "topicId")
	status := params.FormValue(c.Ctx, "status")
	if topicId == 0 {
		var form struct {
			TopicId int64  `json:"topicId"`
			Status  string `json:"status"`
		}
		if err := c.Ctx.ReadJSON(&form); err == nil {
			topicId = form.TopicId
			status = form.Status
		}
	}
	topic, err := services.PKService.SetTopicStatus(topicId, status)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(topic)
}

func (c *PKController) GetRoundList() *web.JsonResult {
	page, _ := params.GetInt(c.Ctx, "page")
	pageSize, _ := params.GetInt(c.Ctx, "pageSize")
	topicId, _ := params.GetInt64(c.Ctx, "topicId")
	phase := params.FormValue(c.Ctx, "phase")
	winner := params.FormValue(c.Ctx, "winner")
	data, err := services.PKService.AdminRounds(page, pageSize, topicId, phase, winner)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(data)
}

func (c *PKController) GetSeasonList() *web.JsonResult {
	page, _ := params.GetInt(c.Ctx, "page")
	pageSize, _ := params.GetInt(c.Ctx, "pageSize")
	topicId, _ := params.GetInt64(c.Ctx, "topicId")
	status := params.FormValue(c.Ctx, "status")
	data, err := services.PKService.AdminSeasons(page, pageSize, topicId, status)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(data)
}

func (c *PKController) PostRecalcHeat() *web.JsonResult {
	roundId, _ := params.GetInt64(c.Ctx, "roundId")
	if roundId == 0 {
		var form struct {
			RoundId int64 `json:"roundId"`
		}
		if err := c.Ctx.ReadJSON(&form); err == nil {
			roundId = form.RoundId
		}
	}
	data, err := services.PKService.RecalcRoundHeat(roundId)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(data)
}
