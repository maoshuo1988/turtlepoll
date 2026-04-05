package api

import (
	"bbs-go/internal/models"
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/services"
	"strings"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
)

type UserReportController struct {
	Ctx iris.Context
}

func (c *UserReportController) PostSubmit() *web.JsonResult {
	var (
		dataId, _ = params.FormValueInt64(c.Ctx, "dataId")
		dataType  = params.FormValue(c.Ctx, "dataType")
		reason    = params.FormValue(c.Ctx, "reason")
	)
	if dataId <= 0 {
		return web.JsonErrorMsg("dataId is required")
	}
	dataType = strings.TrimSpace(dataType)
	if dataType == "" {
		return web.JsonErrorMsg("dataType is required")
	}
	if len(reason) > 1024 {
		reason = reason[:1024]
	}
	report := &models.UserReport{
		DataId:     dataId,
		DataType:   dataType,
		Reason:     reason,
		CreateTime: dates.NowTimestamp(),
	}

	if user := common.GetCurrentUser(c.Ctx); user != nil {
		report.UserId = user.Id
	}
	services.UserReportService.Create(report)
	return web.JsonSuccess()
}
