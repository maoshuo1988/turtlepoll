package services

import (
	"bbs-go/internal/models/models"
	"bbs-go/internal/pkg/config"
	"bbs-go/internal/repositories"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

var NewsCrawlerService = newNewsCrawlerService()

func newNewsCrawlerService() *newsCrawlerService {
	return &newsCrawlerService{}
}

type newsCrawlerService struct{}

type newsCrawlItem struct {
	Title   string
	URL     string
	Channel string
	Summary string
	Cover   string
}

var (
	reHupuAnchor   = regexp.MustCompile(`<a[^>]+href="(https://bbs\.hupu\.com/\d+\.html)"[^>]*>(.*?)</a>`)
	reScriptStyle  = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>|<style[^>]*>.*?</style>`)
	reStripHTML    = regexp.MustCompile(`<[^>]+>`)
	reMultiSpace   = regexp.MustCompile(`\s+`)
	reSourceID     = regexp.MustCompile(`https://bbs\.hupu\.com/(\d+)\.html`)
	reTitleTag     = regexp.MustCompile(`(?is)<title>(.*?)</title>`)
	reDescription  = regexp.MustCompile(`(?is)<meta\s+name="description"\s+content="(.*?)"`)
	reOgImage      = regexp.MustCompile(`(?is)<meta\s+property="og:image"\s+content="(.*?)"`)
	reImgSrc       = regexp.MustCompile(`(?is)<img[^>]+src=["']([^"']+)["'][^>]*>`)
	reChannelLabel = regexp.MustCompile(`(?is)<a[^>]+href="https://bbs\.hupu\.com/(\d+|[a-zA-Z0-9\-]+)"[^>]*>([^<]{2,20})</a>`)
	reBodyTag      = regexp.MustCompile(`(?is)<body[^>]*>(.*?)</body>`)
	reArticleTag   = regexp.MustCompile(`(?is)<article[^>]*>(.*?)</article>`)
	reContentBlock = regexp.MustCompile(`(?is)<div[^>]+class="[^"]*(?:post-content|detail-content|article-content|thread-content|content-main|bbs-detail-wrap|bbs-post-content)[^"]*"[^>]*>(.*?)</div>`)
)

func (s *newsCrawlerService) buildHTTPClient() *http.Client {
	timeout := 5
	if config.Instance.News.RequestTimeoutSeconds > 0 {
		timeout = config.Instance.News.RequestTimeoutSeconds
	}
	return &http.Client{Timeout: time.Duration(timeout) * time.Second}
}

func (s *newsCrawlerService) fetchURL(url string) (string, error) {
	client := s.buildHTTPClient()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "turtlepoll-news-bot/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", errors.New("external status: " + strconv.Itoa(resp.StatusCode))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (s *newsCrawlerService) fetchWithRetry(url string) (string, int, error) {
	maxRetry := config.Instance.News.MaxRetry
	if maxRetry <= 0 {
		maxRetry = 3
	}
	var lastErr error
	for i := 0; i < maxRetry; i++ {
		body, err := s.fetchURL(url)
		if err == nil {
			return body, i, nil
		}
		lastErr = err
		if i == maxRetry-1 {
			break
		}
		waitSeconds := NewsService.RetryDelaySeconds(i + 1)
		NewsService.SleepRetryDelay(waitSeconds)
	}
	return "", maxRetry, lastErr
}

func cleanHTMLText(in string) string {
	in = reScriptStyle.ReplaceAllString(in, " ")
	in = reStripHTML.ReplaceAllString(in, " ")
	in = strings.ReplaceAll(in, "&nbsp;", " ")
	in = strings.ReplaceAll(in, "&amp;", "&")
	in = strings.ReplaceAll(in, "&quot;", `"`)
	in = strings.ReplaceAll(in, "&#39;", "'")
	in = strings.ReplaceAll(in, "&lt;", "<")
	in = strings.ReplaceAll(in, "&gt;", ">")
	in = reMultiSpace.ReplaceAllString(in, " ")
	return strings.TrimSpace(in)
}

func clipRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	rs := []rune(strings.TrimSpace(s))
	if len(rs) <= max {
		return string(rs)
	}
	return string(rs[:max])
}

func normalizeImageURL(url string) string {
	url = strings.TrimSpace(url)
	if url == "" {
		return ""
	}
	if strings.HasPrefix(url, "//") {
		return "https:" + url
	}
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url
	}
	return ""
}

func extractImageURLs(rawHTML string) []string {
	matches := reImgSrc.FindAllStringSubmatch(rawHTML, -1)
	if len(matches) == 0 {
		return []string{}
	}
	ret := make([]string, 0, len(matches))
	seen := make(map[string]struct{})
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		u := normalizeImageURL(m[1])
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		ret = append(ret, u)
	}
	return ret
}

func sourceIDFromURL(url string) string {
	m := reSourceID.FindStringSubmatch(url)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

func (s *newsCrawlerService) parseList(html string) []newsCrawlItem {
	matches := reHupuAnchor.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return nil
	}
	items := make([]newsCrawlItem, 0, len(matches))
	seen := make(map[string]struct{})
	channel := "综合"
	if cm := reChannelLabel.FindStringSubmatch(html); len(cm) >= 3 {
		channel = cleanHTMLText(cm[2])
	}
	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		url := strings.TrimSpace(m[1])
		title := cleanHTMLText(m[2])
		if url == "" || title == "" {
			continue
		}
		if _, ok := seen[url]; ok {
			continue
		}
		seen[url] = struct{}{}
		items = append(items, newsCrawlItem{Title: title, URL: url, Channel: channel, Summary: title})
	}
	return items
}

func (s *newsCrawlerService) parseDetail(html string) (title, summary, content, cover string, contentImages []string) {
	if m := reTitleTag.FindStringSubmatch(html); len(m) >= 2 {
		title = cleanHTMLText(m[1])
	}
	if m := reDescription.FindStringSubmatch(html); len(m) >= 2 {
		summary = cleanHTMLText(m[1])
	}
	rawContent := ""
	if m := reContentBlock.FindStringSubmatch(html); len(m) >= 2 {
		rawContent = m[1]
	} else if m := reArticleTag.FindStringSubmatch(html); len(m) >= 2 {
		rawContent = m[1]
	} else if m := reBodyTag.FindStringSubmatch(html); len(m) >= 2 {
		rawContent = m[1]
	}
	contentImages = extractImageURLs(rawContent)
	content = cleanHTMLText(rawContent)
	if summary == "" {
		summary = clipRunes(content, 200)
	}
	if m := reOgImage.FindStringSubmatch(html); len(m) >= 2 {
		cover = strings.TrimSpace(m[1])
	}
	if cover == "" && len(contentImages) > 0 {
		cover = contentImages[0]
	}
	return
}

func inferCategoryByChannel(channel string) string {
	lc := strings.ToLower(channel)
	switch {
	case strings.Contains(lc, "nba"):
		return "nba"
	case strings.Contains(lc, "cba"):
		return "cba"
	case strings.Contains(lc, "足球") || strings.Contains(lc, "soccer"):
		return "soccer"
	default:
		return "general"
	}
}

func (s *newsCrawlerService) upsertItem(item newsCrawlItem, withDetail bool) error {
	now := dates.NowTimestamp()
	sourceId := sourceIDFromURL(item.URL)
	if sourceId == "" {
		return errors.New("invalid source id")
	}

	article := &models.NewsArticle{
		Source:        "hupu",
		SourceId:      sourceId,
		SourceUrl:     item.URL,
		Slug:          "",
		Title:         item.Title,
		Summary:       item.Summary,
		Content:       "",
		ContentImages: EncodeNewsImages([]string{}),
		CoverUrl:      item.Cover,
		Channel:       item.Channel,
		Category:      inferCategoryByChannel(item.Channel),
		Tags:          EncodeNewsTags([]string{}),
		PublishedAt:   now,
		FetchedAt:     now,
		HotScore:      50,
		DetailStatus:  "pending",
		Status:        "normal",
		CreateTime:    now,
		UpdateTime:    now,
	}

	if withDetail {
		html, _, err := s.fetchWithRetry(item.URL)
		if err == nil {
			detailTitle, detailSummary, detailContent, detailCover, detailImages := s.parseDetail(html)
			if detailTitle != "" {
				article.Title = detailTitle
			}
			if detailSummary != "" {
				article.Summary = detailSummary
			}
			if detailContent != "" {
				article.Content = detailContent
			}
			article.ContentImages = EncodeNewsImages(detailImages)
			if detailCover != "" {
				article.CoverUrl = detailCover
			}
			article.DetailStatus = "success"
		} else {
			article.DetailStatus = "failed"
		}
	}

	return repositories.NewsRepository.UpsertArticle(sqls.DB(), article)
}

func (s *newsCrawlerService) RunTask(taskId int64, limit int, withDetail bool) error {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if err := NewsService.UpdateTaskStatus(taskId, "running", 0, ""); err != nil {
		return err
	}
	isOpen, until, err := NewsService.IsCircuitOpen("hupu")
	if err != nil {
		_ = NewsService.UpdateTaskStatus(taskId, "failed", 1, err.Error())
		return err
	}
	if isOpen {
		_ = NewsService.MarkTaskRetry(taskId, 1, "circuit_open_until_"+strconv.FormatInt(until, 10))
		return errors.New("circuit_open")
	}

	html, retries, err := s.fetchWithRetry("https://www.hupu.com/")
	if err != nil {
		_ = NewsService.MarkTaskRetry(taskId, retries, err.Error())
		return err
	}
	items := s.parseList(html)
	if len(items) == 0 {
		err = errors.New("parse list failed")
		_ = NewsService.MarkTaskRetry(taskId, 1, err.Error())
		return err
	}
	if len(items) > limit {
		items = items[:limit]
	}

	failCount := 0
	for _, item := range items {
		if e := s.upsertItem(item, withDetail); e != nil {
			failCount++
		}
	}
	if failCount > 0 {
		_ = NewsService.MarkTaskRetry(taskId, failCount, "partial failed")
		return errors.New("partial failed")
	}
	_ = NewsService.UpdateTaskStatus(taskId, "success", 0, "")
	return nil
}

func (s *newsCrawlerService) RunScheduledListCrawl() {
	isOpen, _, err := NewsService.IsCircuitOpen("hupu")
	if err != nil {
		return
	}
	if isOpen {
		return
	}
	task, err := NewsService.CreateCrawlTask("list", "hupu")
	if err != nil {
		return
	}
	_ = s.RunTask(task.Id, config.Instance.News.BatchSize, config.Instance.News.AllowDetailRefresh)
}

func (s *newsCrawlerService) RefreshDetails(articleIds []int64) error {
	if len(articleIds) == 0 {
		return nil
	}
	var list []models.NewsArticle
	if err := sqls.DB().Where("id in (?)", articleIds).Find(&list).Error; err != nil {
		return err
	}
	for _, a := range list {
		html, _, err := s.fetchWithRetry(a.SourceUrl)
		if err != nil {
			a.DetailStatus = "failed"
			a.UpdateTime = dates.NowTimestamp()
			_ = sqls.DB().Save(&a).Error
			continue
		}
		title, summary, content, cover, images := s.parseDetail(html)
		if title != "" {
			a.Title = title
		}
		if summary != "" {
			a.Summary = summary
		}
		if content != "" {
			a.Content = content
		}
		a.ContentImages = EncodeNewsImages(images)
		if cover != "" {
			a.CoverUrl = cover
		}
		a.DetailStatus = "success"
		a.FetchedAt = dates.NowTimestamp()
		a.UpdateTime = dates.NowTimestamp()
		_ = sqls.DB().Save(&a).Error
	}
	return nil
}

func (s *newsCrawlerService) GetTask(taskId int64) *models.NewsCrawlTask {
	return repositories.NewsRepository.GetTask(sqls.DB(), taskId)
}

func (s *newsCrawlerService) EnsureTaskFinished(taskId int64, err error) {
	if err == nil {
		return
	}
	task := repositories.NewsRepository.GetTask(sqls.DB(), taskId)
	if task == nil {
		return
	}
	if task.Status == "success" || task.Status == "failed" {
		return
	}
	task.Status = "failed"
	task.FailCount++
	task.LastError = err.Error()
	task.FinishedAt = dates.NowTimestamp()
	task.UpdateTime = task.FinishedAt
	_ = sqls.DB().Save(task).Error
}

func IsNewsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
