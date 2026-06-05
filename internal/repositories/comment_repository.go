package repositories

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/resp"

	"github.com/mlogclub/simple/sqls"
	"github.com/mlogclub/simple/web/params"
	"gorm.io/gorm"

	"bbs-go/internal/models/models"
)

var CommentRepository = newCommentRepository()

func newCommentRepository() *commentRepository {
	return &commentRepository{}
}

type commentRepository struct {
}

func (r *commentRepository) Get(db *gorm.DB, id int64) *models.Comment {
	ret := &models.Comment{}
	if err := db.First(ret, "id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *commentRepository) Take(db *gorm.DB, where ...interface{}) *models.Comment {
	ret := &models.Comment{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *commentRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.Comment) {
	cnd.Find(db, &list)
	return
}

func (r *commentRepository) FindOne(db *gorm.DB, cnd *sqls.Cnd) *models.Comment {
	ret := &models.Comment{}
	if err := cnd.FindOne(db, &ret); err != nil {
		return nil
	}
	return ret
}

func (r *commentRepository) FindPageByParams(db *gorm.DB, params *params.QueryParams) (list []models.Comment, paging *sqls.Paging) {
	return r.FindPageByCnd(db, &params.Cnd)
}

func (r *commentRepository) FindPageByCnd(db *gorm.DB, cnd *sqls.Cnd) (list []models.Comment, paging *sqls.Paging) {
	cnd.Find(db, &list)
	count := cnd.Count(db, &models.Comment{})

	paging = &sqls.Paging{
		Page:  cnd.Paging.Page,
		Limit: cnd.Paging.Limit,
		Total: count,
	}
	return
}

func (r *commentRepository) FindUserCenterCommentPage(db *gorm.DB, userId int64, page, limit int) (list []resp.UserCenterCommentResponse, paging *sqls.Paging, err error) {
	baseQuery := db.Table("t_comment AS c").
		Joins("INNER JOIN t_topic t ON t.id = c.entity_id").
		Where("c.user_id = ?", userId).
		Where("c.entity_type = CAST(t.type AS TEXT)").
		Where("c.status = ?", constants.StatusOk).
		Where("t.status = ?", constants.StatusOk).
		Where("c.user_id <> t.user_id")

	var total int64
	if err = baseQuery.Count(&total).Error; err != nil {
		return
	}

	err = baseQuery.
		Select("c.id, c.user_id, c.content, t.title, c.create_time").
		Order("c.create_time DESC, c.id DESC").
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

func (r *commentRepository) CountUserCenterComment(db *gorm.DB, userId int64) (count int64) {
	_ = db.Table("t_comment AS c").
		Joins("INNER JOIN t_topic t ON t.id = c.entity_id").
		Where("c.user_id = ?", userId).
		Where("c.entity_type = CAST(t.type AS TEXT)").
		Where("c.status = ?", constants.StatusOk).
		Where("t.status = ?", constants.StatusOk).
		Where("c.user_id <> t.user_id").
		Count(&count).Error
	return
}

func (r *commentRepository) Count(db *gorm.DB, cnd *sqls.Cnd) int64 {
	return cnd.Count(db, &models.Comment{})
}

func (r *commentRepository) Create(db *gorm.DB, t *models.Comment) (err error) {
	err = db.Create(t).Error
	return
}

func (r *commentRepository) Update(db *gorm.DB, t *models.Comment) (err error) {
	err = db.Save(t).Error
	return
}

func (r *commentRepository) Updates(db *gorm.DB, id int64, columns map[string]interface{}) (err error) {
	err = db.Model(&models.Comment{}).Where("id = ?", id).Updates(columns).Error
	return
}

func (r *commentRepository) UpdateColumn(db *gorm.DB, id int64, name string, value interface{}) (err error) {
	err = db.Model(&models.Comment{}).Where("id = ?", id).UpdateColumn(name, value).Error
	return
}

func (r *commentRepository) Delete(db *gorm.DB, id int64) {
	db.Delete(&models.Comment{}, "id = ?", id)
}
