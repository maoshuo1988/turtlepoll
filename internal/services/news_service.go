package services

import (
	"bbs-go/internal/models/models"
	"bbs-go/internal/pkg/config"
	"bbs-go/internal/repositories"
	"encoding/json"
	"strings"
	"time"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"

	"gorm.io/gorm"
)

var NewsService = newNewsService()

func newNewsService() *newsService {
	return &newsService{}
}

type newsService struct{}

type NewsHealthReport struct {
	Reachability     float64 `json:"reachability"`
	AvgLatencyMs     int64   `json:"avgLatencyMs"`
	ParseSuccessRate float64 `json:"parseSuccessRate"`
	NewItemCount     int64   `json:"newItemCount"`
	WindowMinutes    int     `json:"windowMinutes"`
}

type NewsListQuery struct {
	Page     int
	PageSize int
	Q        string
	Category string
	Tag      string
	Source   string
	Sort     string
}

type NewsListResult struct {
	List     []models.NewsArticle
	Count    int64
	Page     int
	PageSize int
}

func (s *newsService) normalizeListQuery(in NewsListQuery) NewsListQuery {
	if in.Page < 1 {
		in.Page = 1
	}
	if in.PageSize <= 0 {
		in.PageSize = 20
	}
	if in.PageSize > 200 {
		in.PageSize = 200
	}
	if strings.TrimSpace(in.Source) == "" {
		in.Source = "hupu"
	}
	if in.Sort != "hotScore_desc" {
		in.Sort = "publishedAt_desc"
	}
	return in
}

func (s *newsService) List(in NewsListQuery) (*NewsListResult, error) {
	in = s.normalizeListQuery(in)

	db := sqls.DB().Model(&models.NewsArticle{}).Where("status = ?", "normal")
	db = db.Where("source = ?", in.Source)
	if q := strings.TrimSpace(in.Q); q != "" {
		like := "%" + q + "%"
		db = db.Where("title ILIKE ? OR summary ILIKE ? OR content ILIKE ?", like, like, like)
	}
	if category := strings.TrimSpace(in.Category); category != "" {
		db = db.Where("category = ?", category)
	}
	if tag := strings.TrimSpace(in.Tag); tag != "" {
		tag = strings.ToLower(tag)
		db = db.Where("lower(tags) LIKE ?", "%\""+tag+"\"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (in.Page - 1) * in.PageSize
	query := db.Offset(offset).Limit(in.PageSize)
	if in.Sort == "hotScore_desc" {
		query = query.Order("hot_score desc, id desc")
	} else {
		query = query.Order("published_at desc, id desc")
	}

	var list []models.NewsArticle
	if err := query.Find(&list).Error; err != nil {
		return nil, err
	}

	return &NewsListResult{List: list, Count: total, Page: in.Page, PageSize: in.PageSize}, nil
}

func (s *newsService) GetDetail(id int64, sourceId, slug string) (*models.NewsArticle, error) {
	db := sqls.DB().Model(&models.NewsArticle{})
	ret := &models.NewsArticle{}
	if id > 0 {
		if err := db.Where("id = ?", id).First(ret).Error; err != nil {
			return nil, err
		}
		return ret, nil
	}
	if strings.TrimSpace(sourceId) != "" {
		if err := db.Where("source = ? AND source_id = ?", "hupu", sourceId).First(ret).Error; err != nil {
			return nil, err
		}
		return ret, nil
	}
	if strings.TrimSpace(slug) != "" {
		if err := db.Where("slug = ?", slug).First(ret).Error; err != nil {
			return nil, err
		}
		return ret, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *newsService) ListCategories() ([]models.NewsCategory, error) {
	var list []models.NewsCategory
	err := sqls.DB().Where("enabled = ?", true).Order("sort_no asc, id asc").Find(&list).Error
	return list, err
}

func (s *newsService) ListTags() ([]models.NewsTag, error) {
	var list []models.NewsTag
	err := sqls.DB().Where("enabled = ?", true).Order("id asc").Find(&list).Error
	return list, err
}

func (s *newsService) CreateCrawlTask(taskType, source string) (*models.NewsCrawlTask, error) {
	now := dates.NowTimestamp()
	if source == "" {
		source = "hupu"
	}
	task := &models.NewsCrawlTask{
		Source:     source,
		TaskType:   taskType,
		Status:     "pending",
		FailCount:  0,
		RetryAfter: 0,
		StartedAt:  0,
		FinishedAt: 0,
		CreateTime: now,
		UpdateTime: now,
	}
	if err := repositories.NewsRepository.CreateTask(sqls.DB(), task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *newsService) UpdateTaskStatus(taskId int64, status string, failCount int, lastError string) error {
	task := repositories.NewsRepository.GetTask(sqls.DB(), taskId)
	if task == nil {
		return gorm.ErrRecordNotFound
	}
	now := dates.NowTimestamp()
	if task.StartedAt == 0 {
		task.StartedAt = now
	}
	task.Status = status
	task.FailCount = failCount
	task.LastError = lastError
	task.UpdateTime = now
	if status == "success" || status == "failed" {
		task.FinishedAt = now
	}
	return repositories.NewsRepository.UpdateTask(sqls.DB(), task)
}

func (s *newsService) ListTasks(status string, page, pageSize int) ([]models.NewsCrawlTask, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	db := sqls.DB().Model(&models.NewsCrawlTask{})
	if strings.TrimSpace(status) != "" {
		db = db.Where("status = ?", status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []models.NewsCrawlTask
	err := db.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (s *newsService) ListFailedLogs(taskId int64, page, pageSize int) ([]models.NewsCrawlTask, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	db := sqls.DB().Model(&models.NewsCrawlTask{}).Where("status = ?", "failed")
	if taskId > 0 {
		db = db.Where("id = ?", taskId)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []models.NewsCrawlTask
	err := db.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (s *newsService) RetryDelaySeconds(failCount int) int64 {
	if failCount <= 0 {
		return int64(config.Instance.News.RetryBaseSeconds)
	}
	base := config.Instance.News.RetryBaseSeconds
	if base <= 0 {
		base = 60
	}
	max := config.Instance.News.RetryMaxSeconds
	if max <= 0 {
		max = 120
	}
	delay := base * failCount
	if delay > max {
		delay = max
	}
	// 固定抖动，避免任务在完全同一秒重试。
	jitter := failCount % 7
	return int64(delay + jitter)
}

func (s *newsService) IsCircuitOpen(source string) (bool, int64, error) {
	if strings.TrimSpace(source) == "" {
		source = "hupu"
	}
	threshold := config.Instance.News.MaxRetry
	if threshold <= 0 {
		threshold = 3
	}
	breakMinutes := config.Instance.News.CircuitBreakMinutes
	if breakMinutes <= 0 {
		breakMinutes = 15
	}
	windowStart := dates.NowTimestamp() - int64(breakMinutes*60)

	var tasks []models.NewsCrawlTask
	err := sqls.DB().
		Where("source = ? AND task_type = ? AND create_time >= ?", source, "list", windowStart).
		Order("id desc").
		Limit(threshold).
		Find(&tasks).Error
	if err != nil {
		return false, 0, err
	}
	if len(tasks) < threshold {
		return false, 0, nil
	}
	for _, t := range tasks {
		if t.Status != "failed" {
			return false, 0, nil
		}
	}
	openedAt := tasks[0].UpdateTime
	until := openedAt + int64(breakMinutes*60)
	if until <= dates.NowTimestamp() {
		return false, 0, nil
	}
	return true, until, nil
}

func (s *newsService) BuildHealthReport(windowMinutes int) (*NewsHealthReport, error) {
	if windowMinutes <= 0 {
		windowMinutes = 10
	}
	start := dates.NowTimestamp() - int64(windowMinutes*60)

	var allCount int64
	if err := sqls.DB().
		Model(&models.NewsCrawlTask{}).
		Where("task_type = ? AND create_time >= ?", "list", start).
		Count(&allCount).Error; err != nil {
		return nil, err
	}

	var successCount int64
	if err := sqls.DB().
		Model(&models.NewsCrawlTask{}).
		Where("task_type = ? AND status = ? AND create_time >= ?", "list", "success", start).
		Count(&successCount).Error; err != nil {
		return nil, err
	}

	var failedCount int64
	if err := sqls.DB().
		Model(&models.NewsCrawlTask{}).
		Where("task_type = ? AND status = ? AND create_time >= ?", "list", "failed", start).
		Count(&failedCount).Error; err != nil {
		return nil, err
	}

	reachability := 1.0
	parseSuccessRate := 1.0
	if allCount > 0 {
		reachability = float64(successCount) / float64(allCount)
		parseSuccessRate = float64(successCount) / float64(allCount)
	}

	var latest []models.NewsCrawlTask
	if err := sqls.DB().
		Where("task_type = ? AND create_time >= ?", "list", start).
		Order("id desc").
		Limit(20).
		Find(&latest).Error; err != nil {
		return nil, err
	}
	var latencySum int64
	var latencyCnt int64
	for _, t := range latest {
		if t.StartedAt > 0 && t.FinishedAt > 0 && t.FinishedAt >= t.StartedAt {
			latencySum += (t.FinishedAt - t.StartedAt) * 1000
			latencyCnt++
		}
	}
	var avgLatency int64
	if latencyCnt > 0 {
		avgLatency = latencySum / latencyCnt
	}

	var newItemCount int64
	if err := sqls.DB().
		Model(&models.NewsArticle{}).
		Where("create_time >= ?", start).
		Count(&newItemCount).Error; err != nil {
		return nil, err
	}

	return &NewsHealthReport{
		Reachability:     reachability,
		AvgLatencyMs:     avgLatency,
		ParseSuccessRate: parseSuccessRate,
		NewItemCount:     newItemCount,
		WindowMinutes:    windowMinutes,
	}, nil
}

func (s *newsService) MarkTaskRetry(taskId int64, failCount int, errMsg string) error {
	task := repositories.NewsRepository.GetTask(sqls.DB(), taskId)
	if task == nil {
		return gorm.ErrRecordNotFound
	}
	now := dates.NowTimestamp()
	task.Status = "failed"
	task.FailCount = failCount
	task.LastError = errMsg
	task.RetryAfter = now + s.RetryDelaySeconds(failCount)
	task.FinishedAt = now
	task.UpdateTime = now
	return repositories.NewsRepository.UpdateTask(sqls.DB(), task)
}

func (s *newsService) SleepRetryDelay(seconds int64) {
	if seconds <= 0 {
		return
	}
	time.Sleep(time.Duration(seconds) * time.Second)
}

func ParseNewsTags(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	var tags []string
	if err := json.Unmarshal([]byte(raw), &tags); err != nil {
		return []string{}
	}
	return tags
}

func EncodeNewsTags(tags []string) string {
	if tags == nil {
		tags = []string{}
	}
	b, err := json.Marshal(tags)
	if err != nil {
		return "[]"
	}
	return string(b)
}
