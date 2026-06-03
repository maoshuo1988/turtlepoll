package repositories

import (
	"bbs-go/internal/models/models"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var NewsRepository = newNewsRepository()

func newNewsRepository() *newsRepository {
	return &newsRepository{}
}

type newsRepository struct{}

func (r *newsRepository) GetArticle(db *gorm.DB, id int64) *models.NewsArticle {
	ret := &models.NewsArticle{}
	if err := db.First(ret, "id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *newsRepository) TakeArticle(db *gorm.DB, where ...interface{}) *models.NewsArticle {
	ret := &models.NewsArticle{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *newsRepository) FindArticles(db *gorm.DB, cnd *sqls.Cnd) (list []models.NewsArticle) {
	cnd.Find(db, &list)
	return
}

func (r *newsRepository) CountArticles(db *gorm.DB, cnd *sqls.Cnd) int64 {
	return cnd.Count(db, &models.NewsArticle{})
}

func (r *newsRepository) UpsertArticle(db *gorm.DB, article *models.NewsArticle) error {
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "source"}, {Name: "source_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"source_url",
			"slug",
			"title",
			"summary",
			"content",
			"content_images",
			"cover_url",
			"channel",
			"category",
			"tags",
			"published_at",
			"fetched_at",
			"hot_score",
			"detail_status",
			"status",
			"update_time",
		}),
	}).Create(article).Error
}

func (r *newsRepository) CreateTask(db *gorm.DB, task *models.NewsCrawlTask) error {
	return db.Create(task).Error
}

func (r *newsRepository) UpdateTask(db *gorm.DB, task *models.NewsCrawlTask) error {
	return db.Save(task).Error
}

func (r *newsRepository) GetTask(db *gorm.DB, id int64) *models.NewsCrawlTask {
	ret := &models.NewsCrawlTask{}
	if err := db.First(ret, "id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}
