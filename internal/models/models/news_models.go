package models

// NewsArticle 站内资讯主表
type NewsArticle struct {
	Model
	Source       string `gorm:"size:16;not null;default:'hupu';uniqueIndex:idx_news_source_source_id" json:"source" form:"source"`
	SourceId     string `gorm:"size:64;not null;uniqueIndex:idx_news_source_source_id" json:"sourceId" form:"sourceId"`
	SourceUrl    string `gorm:"size:512;not null;index:idx_news_source_url" json:"sourceUrl" form:"sourceUrl"`
	Slug         string `gorm:"size:128;uniqueIndex:idx_news_slug" json:"slug" form:"slug"`
	Title        string `gorm:"size:255;not null" json:"title" form:"title"`
	Summary      string `gorm:"type:text" json:"summary" form:"summary"`
	Content      string `gorm:"type:text" json:"content" form:"content"`
	CoverUrl     string `gorm:"size:512" json:"coverUrl" form:"coverUrl"`
	Channel      string `gorm:"size:64" json:"channel" form:"channel"`
	Category     string `gorm:"size:64;not null;default:'general';index:idx_news_category_published,priority:1" json:"category" form:"category"`
	Tags         string `gorm:"type:jsonb;not null;default:'[]'" json:"tags" form:"tags"`
	PublishedAt  int64  `gorm:"not null;default:0;index:idx_news_category_published,priority:2;index:idx_news_status_published,priority:2" json:"publishedAt" form:"publishedAt"`
	FetchedAt    int64  `gorm:"not null;default:0" json:"fetchedAt" form:"fetchedAt"`
	HotScore     int    `gorm:"not null;default:0" json:"hotScore" form:"hotScore"`
	DetailStatus string `gorm:"size:16;not null;default:'pending'" json:"detailStatus" form:"detailStatus"`
	Status       string `gorm:"size:16;not null;default:'normal';index:idx_news_status_published,priority:1" json:"status" form:"status"`
	CreateTime   int64  `gorm:"not null;default:0" json:"createTime" form:"createTime"`
	UpdateTime   int64  `gorm:"not null;default:0" json:"updateTime" form:"updateTime"`
}

// NewsSource 数据来源维表
type NewsSource struct {
	Model
	SourceKey  string `gorm:"size:16;not null;uniqueIndex:idx_news_source_key" json:"sourceKey" form:"sourceKey"`
	Name       string `gorm:"size:32;not null" json:"name" form:"name"`
	BaseURL    string `gorm:"size:255;not null" json:"baseUrl" form:"baseUrl"`
	Enabled    bool   `gorm:"not null;default:true" json:"enabled" form:"enabled"`
	CreateTime int64  `gorm:"not null;default:0" json:"createTime" form:"createTime"`
	UpdateTime int64  `gorm:"not null;default:0" json:"updateTime" form:"updateTime"`
}

// NewsCategory 分类维表
type NewsCategory struct {
	Model
	CategoryKey string `gorm:"size:64;not null;uniqueIndex:idx_news_category_key" json:"categoryKey" form:"categoryKey"`
	Name        string `gorm:"size:64;not null" json:"name" form:"name"`
	ParentId    int64  `gorm:"not null;default:0" json:"parentId" form:"parentId"`
	SortNo      int    `gorm:"type:int;not null;default:0" json:"sortNo" form:"sortNo"`
	Enabled     bool   `gorm:"not null;default:true" json:"enabled" form:"enabled"`
	CreateTime  int64  `gorm:"not null;default:0" json:"createTime" form:"createTime"`
	UpdateTime  int64  `gorm:"not null;default:0" json:"updateTime" form:"updateTime"`
}

// NewsTag 标签维表
type NewsTag struct {
	Model
	TagKey     string `gorm:"size:64;not null;uniqueIndex:idx_news_tag_key" json:"tagKey" form:"tagKey"`
	Name       string `gorm:"size:64;not null" json:"name" form:"name"`
	Enabled    bool   `gorm:"not null;default:true" json:"enabled" form:"enabled"`
	CreateTime int64  `gorm:"not null;default:0" json:"createTime" form:"createTime"`
	UpdateTime int64  `gorm:"not null;default:0" json:"updateTime" form:"updateTime"`
}

// NewsArticleCategory 资讯-分类映射
type NewsArticleCategory struct {
	Model
	ArticleId  int64 `gorm:"not null;index:idx_news_article_category_article;uniqueIndex:idx_news_article_category_unique" json:"articleId" form:"articleId"`
	CategoryId int64 `gorm:"not null;uniqueIndex:idx_news_article_category_unique" json:"categoryId" form:"categoryId"`
	CreateTime int64 `gorm:"not null;default:0" json:"createTime" form:"createTime"`
}

// NewsArticleTag 资讯-标签映射
type NewsArticleTag struct {
	Model
	ArticleId  int64 `gorm:"not null;index:idx_news_article_tag_article;uniqueIndex:idx_news_article_tag_unique" json:"articleId" form:"articleId"`
	TagId      int64 `gorm:"not null;uniqueIndex:idx_news_article_tag_unique" json:"tagId" form:"tagId"`
	CreateTime int64 `gorm:"not null;default:0" json:"createTime" form:"createTime"`
}

// NewsCrawlTask 采集任务表
type NewsCrawlTask struct {
	Model
	Source     string `gorm:"size:16;not null;default:'hupu';index:idx_news_task_source_type_created,priority:1" json:"source" form:"source"`
	TaskType   string `gorm:"size:16;not null;default:'list';index:idx_news_task_source_type_created,priority:2" json:"taskType" form:"taskType"`
	Cursor     string `gorm:"size:255" json:"cursor" form:"cursor"`
	Status     string `gorm:"size:16;not null;default:'pending';index:idx_news_task_status_created,priority:1" json:"status" form:"status"`
	FailCount  int    `gorm:"not null;default:0" json:"failCount" form:"failCount"`
	RetryAfter int64  `gorm:"not null;default:0;index:idx_news_task_retry_after" json:"retryAfter" form:"retryAfter"`
	LastError  string `gorm:"type:text" json:"lastError" form:"lastError"`
	StartedAt  int64  `gorm:"not null;default:0" json:"startedAt" form:"startedAt"`
	FinishedAt int64  `gorm:"not null;default:0" json:"finishedAt" form:"finishedAt"`
	CreateTime int64  `gorm:"not null;default:0;index:idx_news_task_status_created,priority:2;index:idx_news_task_source_type_created,priority:3" json:"createTime" form:"createTime"`
	UpdateTime int64  `gorm:"not null;default:0" json:"updateTime" form:"updateTime"`
}
