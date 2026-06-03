package api

import (
	"bbs-go/internal/pkg/errs"
	"bbs-go/internal/services"
	"strings"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
)

type NewsController struct {
	Ctx iris.Context
}

// GetList GET /api/news/list
func (c *NewsController) GetList() *web.JsonResult {
	page, _ := params.GetInt(c.Ctx, "page")
	pageSize, _ := params.GetInt(c.Ctx, "pageSize")
	q := strings.TrimSpace(c.Ctx.URLParamDefault("q", ""))
	category := strings.TrimSpace(c.Ctx.URLParamDefault("category", ""))
	tag := strings.TrimSpace(c.Ctx.URLParamDefault("tag", ""))
	source := strings.TrimSpace(c.Ctx.URLParamDefault("source", "hupu"))
	sort := strings.TrimSpace(c.Ctx.URLParamDefault("sort", "publishedAt_desc"))

	if source != "hupu" {
		return web.JsonErrorCode(400, "PARAM_INVALID")
	}

	ret, err := services.NewsService.List(services.NewsListQuery{
		Page:     page,
		PageSize: pageSize,
		Q:        q,
		Category: category,
		Tag:      tag,
		Source:   source,
		Sort:     sort,
	})
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	list := make([]map[string]any, 0, len(ret.List))
	for _, item := range ret.List {
		list = append(list, map[string]any{
			"id":          item.Id,
			"title":       item.Title,
			"summary":     item.Summary,
			"coverUrl":    item.CoverUrl,
			"source":      item.Source,
			"sourceName":  "虎扑",
			"sourceUrl":   item.SourceUrl,
			"channel":     item.Channel,
			"category":    item.Category,
			"tags":        services.ParseNewsTags(item.Tags),
			"publishedAt": item.PublishedAt,
			"hotScore":    item.HotScore,
		})
	}

	return web.JsonData(map[string]any{
		"list":     list,
		"count":    ret.Count,
		"page":     ret.Page,
		"pageSize": ret.PageSize,
	})
}

// GetDetail GET /api/news/detail
func (c *NewsController) GetDetail() *web.JsonResult {
	id, _ := params.GetInt64(c.Ctx, "id")
	sourceId := strings.TrimSpace(c.Ctx.URLParamDefault("sourceId", ""))
	slug := strings.TrimSpace(c.Ctx.URLParamDefault("slug", ""))

	if id <= 0 && sourceId == "" && slug == "" {
		return web.JsonErrorCode(400, "PARAM_INVALID")
	}

	news, err := services.NewsService.GetDetail(id, sourceId, slug)
	if err != nil {
		if services.IsNewsNotFound(err) {
			return web.JsonErrorCode(404, "NEWS_NOT_FOUND")
		}
		return web.JsonErrorMsg(err.Error())
	}

	return web.JsonData(map[string]any{
		"news": map[string]any{
			"id":            news.Id,
			"source":        news.Source,
			"sourceId":      news.SourceId,
			"sourceUrl":     news.SourceUrl,
			"title":         news.Title,
			"summary":       news.Summary,
			"content":       news.Content,
			"contentImages": services.ParseNewsImages(news.ContentImages),
			"coverUrl":      news.CoverUrl,
			"sourceName":    "虎扑",
			"channel":       news.Channel,
			"category":      news.Category,
			"tags":          services.ParseNewsTags(news.Tags),
			"publishedAt":   news.PublishedAt,
			"fetchedAt":     news.FetchedAt,
			"hotScore":      news.HotScore,
		},
	})
}

// GetCategories GET /api/news/categories
func (c *NewsController) GetCategories() *web.JsonResult {
	list, err := services.NewsService.ListCategories()
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	ret := make([]map[string]any, 0, len(list))
	for _, item := range list {
		ret = append(ret, map[string]any{
			"key":  item.CategoryKey,
			"name": item.Name,
			"sort": item.SortNo,
		})
	}
	return web.JsonData(map[string]any{"list": ret})
}

// GetTags GET /api/news/tags
func (c *NewsController) GetTags() *web.JsonResult {
	list, err := services.NewsService.ListTags()
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	ret := make([]map[string]any, 0, len(list))
	for _, item := range list {
		ret = append(ret, map[string]any{
			"key":  item.TagKey,
			"name": item.Name,
		})
	}
	return web.JsonData(map[string]any{"list": ret})
}

// GetSearch GET /api/news/search
func (c *NewsController) GetSearch() *web.JsonResult {
	return c.GetList()
}

// PostCrawl_run 给普通API保留错误码占位，实际应走admin接口。
func (c *NewsController) PostCrawl_run() *web.JsonResult {
	return web.JsonError(errs.NoPermission())
}
