package admin

import (
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/pkg/config"
	"bbs-go/internal/pkg/errs"
	"bbs-go/internal/services"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
)

type NewsController struct {
	Ctx iris.Context
}

func (c *NewsController) BeforeActivation(b mvc.BeforeActivation) {
	b.Handle("POST", "/crawl/run", "PostCrawl_run")
	b.Handle("POST", "/crawl/refresh", "PostCrawl_refresh")
	b.Handle("GET", "/crawl/tasks", "GetCrawl_tasks")
	b.Handle("GET", "/crawl/logs", "GetCrawl_logs")
	b.Handle("GET", "/crawl/health", "GetCrawl_health")
}

// PostCrawl_run POST /api/admin/news/crawl/run
func (c *NewsController) PostCrawl_run() *web.JsonResult {
	if common.GetCurrentUser(c.Ctx) == nil {
		return web.JsonError(errs.NotLogin())
	}
	source := c.Ctx.URLParamDefault("source", "hupu")
	taskType := c.Ctx.URLParamDefault("taskType", "list")
	limit, _ := params.GetInt(c.Ctx, "limit")
	if limit <= 0 {
		limit = config.Instance.News.BatchSize
	}
	task, err := services.NewsService.CreateCrawlTask(taskType, source)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	go func(taskId int64, limit int) {
		err := services.NewsCrawlerService.RunTask(taskId, limit, config.Instance.News.AllowDetailRefresh)
		services.NewsCrawlerService.EnsureTaskFinished(taskId, err)
	}(task.Id, limit)

	return web.JsonData(map[string]any{
		"taskId": task.Id,
		"status": "pending",
	})
}

// PostCrawl_refresh POST /api/admin/news/crawl/refresh
func (c *NewsController) PostCrawl_refresh() *web.JsonResult {
	if common.GetCurrentUser(c.Ctx) == nil {
		return web.JsonError(errs.NotLogin())
	}
	articleIds := params.FormValueInt64Array(c.Ctx, "articleIds")
	if len(articleIds) == 0 {
		return web.JsonErrorCode(400, "PARAM_INVALID")
	}
	task, err := services.NewsService.CreateCrawlTask("refresh", "hupu")
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	go func(taskId int64, ids []int64) {
		_ = services.NewsService.UpdateTaskStatus(taskId, "running", 0, "")
		err := services.NewsCrawlerService.RefreshDetails(ids)
		if err != nil {
			_ = services.NewsService.UpdateTaskStatus(taskId, "failed", 1, err.Error())
			return
		}
		_ = services.NewsService.UpdateTaskStatus(taskId, "success", 0, "")
	}(task.Id, articleIds)

	return web.JsonData(map[string]any{
		"taskId": task.Id,
		"status": "pending",
	})
}

// GetCrawl_tasks GET /api/admin/news/crawl/tasks
func (c *NewsController) GetCrawl_tasks() *web.JsonResult {
	status := c.Ctx.URLParamDefault("status", "")
	page, _ := params.GetInt(c.Ctx, "page")
	pageSize, _ := params.GetInt(c.Ctx, "pageSize")
	list, total, err := services.NewsService.ListTasks(status, page, pageSize)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(map[string]any{
		"list":  list,
		"total": total,
		"page":  page,
		"limit": pageSize,
	})
}

// GetCrawl_logs GET /api/admin/news/crawl/logs
func (c *NewsController) GetCrawl_logs() *web.JsonResult {
	taskId, _ := params.GetInt64(c.Ctx, "taskId")
	page, _ := params.GetInt(c.Ctx, "page")
	pageSize, _ := params.GetInt(c.Ctx, "pageSize")
	list, total, err := services.NewsService.ListFailedLogs(taskId, page, pageSize)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(map[string]any{
		"list":  list,
		"total": total,
		"page":  page,
		"limit": pageSize,
	})
}

// GetCrawl_health GET /api/admin/news/crawl/health
func (c *NewsController) GetCrawl_health() *web.JsonResult {
	windowMinutes, _ := params.GetInt(c.Ctx, "windowMinutes")
	if windowMinutes <= 0 {
		windowMinutes = 10
	}
	report, err := services.NewsService.BuildHealthReport(windowMinutes)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	isOpen, until, err := services.NewsService.IsCircuitOpen("hupu")
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(map[string]any{
		"health":       report,
		"circuitOpen":  isOpen,
		"circuitUntil": until,
	})
}
