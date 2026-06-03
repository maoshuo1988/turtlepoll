package migrations

import (
	"bbs-go/internal/models/models"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm/clause"
)

func migrate_news_init_tables() error {
	db := sqls.DB()
	now := dates.NowTimestamp()

	if err := db.AutoMigrate(
		&models.NewsArticle{},
		&models.NewsSource{},
		&models.NewsCategory{},
		&models.NewsTag{},
		&models.NewsArticleCategory{},
		&models.NewsArticleTag{},
		&models.NewsCrawlTask{},
	); err != nil {
		return err
	}

	source := models.NewsSource{
		SourceKey:  "hupu",
		Name:       "虎扑",
		BaseURL:    "https://www.hupu.com",
		Enabled:    true,
		CreateTime: now,
		UpdateTime: now,
	}
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "source_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "base_url", "enabled", "update_time"}),
	}).Create(&source).Error; err != nil {
		return err
	}

	categories := []models.NewsCategory{
		{CategoryKey: "general", Name: "综合", SortNo: 10, Enabled: true, CreateTime: now, UpdateTime: now},
		{CategoryKey: "nba", Name: "NBA", SortNo: 20, Enabled: true, CreateTime: now, UpdateTime: now},
		{CategoryKey: "cba", Name: "CBA", SortNo: 30, Enabled: true, CreateTime: now, UpdateTime: now},
		{CategoryKey: "soccer", Name: "足球", SortNo: 40, Enabled: true, CreateTime: now, UpdateTime: now},
	}
	for _, c := range categories {
		payload := c
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "category_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "sort_no", "enabled", "update_time"}),
		}).Create(&payload).Error; err != nil {
			return err
		}
	}

	return nil
}
