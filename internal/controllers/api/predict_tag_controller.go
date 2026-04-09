package api

import (
	"bbs-go/internal/models/models"
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/pkg/errs"
	"bbs-go/internal/services"
	"strings"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/sqls"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
)

// PredictTagController 预测市场标签查询接口
// 路由：/api/predict-tag
// 说明：当前项目 /api 下默认需要登录（AuthMiddleware）。
// 若希望公开，可迁移到不需要 Auth 的 party（当前不做结构性调整）。

type PredictTagController struct {
	Ctx iris.Context
}

// 标签列表：GET /api/predict-tag/list
// query:
// - page=1
// - pageSize=20 (max 200)
// - q=xxx (name/slug 模糊)
// - slugs=a,b,c
// - sort=updatedAt|marketCount (default updatedAt)
// - includeCounts=0|1
func (c *PredictTagController) GetList() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}

	page, _ := params.GetInt(c.Ctx, "page")
	pageSize, _ := params.GetInt(c.Ctx, "pageSize")
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	q := strings.TrimSpace(params.FormValue(c.Ctx, "q"))
	slugs := strings.TrimSpace(params.FormValue(c.Ctx, "slugs"))
	sort := strings.TrimSpace(params.FormValue(c.Ctx, "sort"))
	includeCounts := strings.TrimSpace(params.FormValue(c.Ctx, "includeCounts"))

	db := sqls.DB()

	query := db.Model(&models.PredictTag{})
	if q != "" {
		like := "%" + q + "%"
		query = query.Where("slug ILIKE ? OR name ILIKE ?", like, like)
	}
	if slugs != "" {
		parts := []string{}
		for _, p := range strings.Split(slugs, ",") {
			s := strings.ToLower(strings.TrimSpace(p))
			if s != "" {
				parts = append(parts, s)
			}
		}
		if len(parts) > 0 {
			query = query.Where("slug IN ?", parts)
		}
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	offset := (page - 1) * pageSize

	// includeCounts=1：join stat
	if includeCounts == "1" {
		query = query.Select("t_predict_tag.*, COALESCE(s.market_count,0) as market_count").
			Joins("LEFT JOIN t_predict_tag_stat s ON s.tag_id = t_predict_tag.id")
	}

	// 排序
	// 注意：market_count 字段只在 includeCounts=1 的 join 查询中才存在。
	if sort == "marketCount" && includeCounts == "1" {
		// 注意：避免某些 SQL 方言/ORM 解析把 market_count 误解析成 t_predict_tag.market_count
		// 我们显式按 join 表字段排序，并用 COALESCE 保证 NULL 时排序稳定。
		query = query.Order("COALESCE(s.market_count,0) desc, t_predict_tag.id desc")
	} else {
		query = query.Order("t_predict_tag.update_time desc, t_predict_tag.id desc")
	}

	// 返回结构：显式字段映射，避免 GORM 因匿名嵌入 models.PredictTag
	// 在某些情况下把统计字段（market_count）误当作 t_predict_tag 的列来选择。
	type item struct {
		Id         int64  `json:"id"`
		Slug       string `json:"slug"`
		Name       string `json:"name"`
		LastSeenAt int64  `json:"lastSeenAt"`
		CreateTime int64  `json:"createTime"`
		UpdateTime int64  `json:"updateTime"`
		// includeCounts=1 时通过 select 别名返回；否则为 0。
		MarketCount int64 `json:"marketCount"`
	}

	// 基础 select：仅来自 t_predict_tag 的真实列
	query = query.Select("t_predict_tag.id, t_predict_tag.slug, t_predict_tag.name, t_predict_tag.last_seen_at, t_predict_tag.create_time, t_predict_tag.update_time")
	if includeCounts == "1" {
		// 叠加 join 的统计字段（别名 market_count -> struct 字段 MarketCount）
		query = query.Select("t_predict_tag.id, t_predict_tag.slug, t_predict_tag.name, t_predict_tag.last_seen_at, t_predict_tag.create_time, t_predict_tag.update_time, COALESCE(s.market_count,0) as market_count")
	}

	var list []item
	if err := query.Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	return web.JsonData(map[string]any{
		"list":     list,
		"count":    count,
		"page":     page,
		"pageSize": pageSize,
	})
}

// 手动刷新标签物化：POST /api/predict-tag/refresh
// 说明：当前挂 /api 下会要求登录；如需管理员控制，可迁移到 /api/admin。
func (c *PredictTagController) PostRefresh() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	_ = user
	if err := services.PredictTagService.RefreshTagsFromContexts(); err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonSuccess()
}
