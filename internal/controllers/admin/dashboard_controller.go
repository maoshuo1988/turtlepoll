package admin

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/models"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/sqls"
	"github.com/mlogclub/simple/web"
)

// DashboardController 运营总览看板聚合接口
// 路由：/api/admin/dashboard
// 注意：该 party 已经过 AuthMiddleware + AdminMiddleware。
type DashboardController struct {
	Ctx iris.Context
}

// GetStats GET /api/admin/dashboard/stats
// 返回：总用户/总评论/总帖子数
func (c *DashboardController) GetStats() *web.JsonResult {
	// 总用户
	var totalUsers int64
	if err := sqls.DB().Model(&models.User{}).Count(&totalUsers).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	// 总评论（仅统计 status=ok）
	var totalComments int64
	if err := sqls.DB().Model(&models.Comment{}).
		Where("status = ?", constants.StatusOk).
		Count(&totalComments).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	// 总帖子数（Topic，status=ok）
	var totalTopics int64
	if err := sqls.DB().Model(&models.Topic{}).
		Where("status = ?", constants.StatusOk).
		Count(&totalTopics).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	return web.JsonData(map[string]any{
		"totalUsers":    totalUsers,
		"totalComments": totalComments,
		"totalTopics":   totalTopics,
	})
}
