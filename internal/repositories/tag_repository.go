package repositories

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/resp"
	"errors"
	"strings"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"github.com/mlogclub/simple/web/params"
	"gorm.io/gorm"

	"bbs-go/internal/models/models"
)

var TagRepository = newTagRepository()

func newTagRepository() *tagRepository {
	return &tagRepository{}
}

type tagRepository struct {
}

func (r *tagRepository) Get(db *gorm.DB, id int64) *models.Tag {
	ret := &models.Tag{}
	if err := db.First(ret, "id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *tagRepository) Take(db *gorm.DB, where ...interface{}) *models.Tag {
	ret := &models.Tag{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *tagRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.Tag) {
	cnd.Find(db, &list)
	return
}

func (r *tagRepository) FindOne(db *gorm.DB, cnd *sqls.Cnd) *models.Tag {
	ret := &models.Tag{}
	if err := cnd.FindOne(db, &ret); err != nil {
		return nil
	}
	return ret
}

func (r *tagRepository) FindPageByParams(db *gorm.DB, params *params.QueryParams) (list []models.Tag, paging *sqls.Paging) {
	return r.FindPageByCnd(db, &params.Cnd)
}

func (r *tagRepository) FindPageByCnd(db *gorm.DB, cnd *sqls.Cnd) (list []models.Tag, paging *sqls.Paging) {
	cnd.Find(db, &list)
	count := cnd.Count(db, &models.Tag{})

	paging = &sqls.Paging{
		Page:  cnd.Paging.Page,
		Limit: cnd.Paging.Limit,
		Total: count,
	}
	return
}

func (r *tagRepository) FindCommentStatPage(db *gorm.DB, page, limit int, keyword string) (list []resp.TagCommentStatResponse, paging *sqls.Paging, err error) {
	baseQuery := db.Table("t_tag AS tg").Where("tg.status = ?", constants.StatusOk)
	if keyword != "" {
		baseQuery = baseQuery.Where("tg.name LIKE ?", "%"+keyword+"%")
	}

	var total int64
	if err = baseQuery.Count(&total).Error; err != nil {
		return
	}

	err = baseQuery.
		Select("tg.id AS tag_id, tg.name AS tag_name, COUNT(c.id) AS comment_count").
		Joins("LEFT JOIN t_topic_tag tt ON tt.tag_id = tg.id AND tt.status = ?", constants.StatusOk).
		Joins("LEFT JOIN t_topic tp ON tp.id = tt.topic_id AND tp.status = ?", constants.StatusOk).
		Joins("LEFT JOIN t_comment c ON c.entity_id = tp.id AND c.entity_type = ? AND c.status = ?", constants.EntityTopic, constants.StatusOk).
		Group("tg.id, tg.name").
		Order("comment_count DESC, tg.id ASC").
		Offset((page - 1) * limit).
		Limit(limit).
		Scan(&list).Error
	if err != nil {
		return
	}

	paging = &sqls.Paging{
		Page:  page,
		Limit: limit,
		Total: total,
	}
	return
}

func (r *tagRepository) Create(db *gorm.DB, t *models.Tag) (err error) {
	err = db.Create(t).Error
	return
}

func (r *tagRepository) Update(db *gorm.DB, t *models.Tag) (err error) {
	err = db.Save(t).Error
	return
}

func (r *tagRepository) Updates(db *gorm.DB, id int64, columns map[string]interface{}) (err error) {
	err = db.Model(&models.Tag{}).Where("id = ?", id).Updates(columns).Error
	return
}

func (r *tagRepository) UpdateColumn(db *gorm.DB, id int64, name string, value interface{}) (err error) {
	err = db.Model(&models.Tag{}).Where("id = ?", id).UpdateColumn(name, value).Error
	return
}

func (r *tagRepository) Delete(db *gorm.DB, id int64) {
	db.Delete(&models.Tag{}, "id = ?", id)
}

func (r *tagRepository) GetTagInIds(db *gorm.DB, tagIds []int64) []models.Tag {
	if len(tagIds) == 0 {
		return nil
	}
	var tags []models.Tag
	db.Where("id in (?)", tagIds).Find(&tags)
	return tags
}

func (r *tagRepository) GetByName(db *gorm.DB, name string) *models.Tag {
	if len(name) == 0 {
		return nil
	}
	return r.Take(db, "name = ?", name)
}

func (r *tagRepository) GetOrCreate(db *gorm.DB, name string) (*models.Tag, error) {
	if len(name) == 0 {
		return nil, errors.New("标签为空")
	}
	// IMPORTANT: use the provided transaction `db` to avoid opening a second
	// connection when the caller is already inside a transaction (SQLite write
	// locks are connection scoped and can cause a deadlock/hang otherwise).
	tag := r.GetByName(db, name)
	if tag != nil {
		return tag, nil
	}

	tag = &models.Tag{
		Name:       name,
		Status:     constants.StatusOk,
		CreateTime: dates.NowTimestamp(),
		UpdateTime: dates.NowTimestamp(),
	}
	if err := r.Create(db, tag); err != nil {
		return nil, err
	}
	return tag, nil
}

func (r *tagRepository) GetOrCreates(db *gorm.DB, tags []string) (tagIds []int64, err error) {
	for _, tagName := range tags {
		var tag *models.Tag
		tag, err = r.GetOrCreate(db, strings.TrimSpace(tagName))
		if err != nil {
			return
		}
		tagIds = append(tagIds, tag.Id)
	}
	return
}
