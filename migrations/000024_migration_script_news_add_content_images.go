package migrations

import "github.com/mlogclub/simple/sqls"

func migrate_news_add_content_images() error {
	db := sqls.DB()
	if err := db.Exec(`ALTER TABLE t_news_article ADD COLUMN IF NOT EXISTS content_images jsonb NOT NULL DEFAULT '[]'::jsonb`).Error; err != nil {
		return err
	}
	if err := db.Exec(`UPDATE t_news_article SET content_images = '[]'::jsonb WHERE content_images IS NULL`).Error; err != nil {
		return err
	}
	return nil
}
