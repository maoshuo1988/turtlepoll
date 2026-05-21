package repositories

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/resp"

	"github.com/mlogclub/simple/sqls"
	"github.com/mlogclub/simple/web/params"
	"gorm.io/gorm"

	"bbs-go/internal/models/models"
)

var UserDislikeRepository = newUserDislikeRepository()

func newUserDislikeRepository() *userDislikeRepository {
	return &userDislikeRepository{}
}

type userDislikeRepository struct {
}

func (r *userDislikeRepository) Get(db *gorm.DB, id int64) *models.UserDislike {
	ret := &models.UserDislike{}
	if err := db.First(ret, "id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *userDislikeRepository) Take(db *gorm.DB, where ...interface{}) *models.UserDislike {
	ret := &models.UserDislike{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *userDislikeRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.UserDislike) {
	cnd.Find(db, &list)
	return
}

func (r *userDislikeRepository) FindOne(db *gorm.DB, cnd *sqls.Cnd) *models.UserDislike {
	ret := &models.UserDislike{}
	if err := cnd.FindOne(db, &ret); err != nil {
		return nil
	}
	return ret
}

func (r *userDislikeRepository) FindPageByParams(db *gorm.DB, params *params.QueryParams) (list []models.UserDislike, paging *sqls.Paging) {
	return r.FindPageByCnd(db, &params.Cnd)
}

func (r *userDislikeRepository) FindPageByCnd(db *gorm.DB, cnd *sqls.Cnd) (list []models.UserDislike, paging *sqls.Paging) {
	cnd.Find(db, &list)
	count := cnd.Count(db, &models.UserDislike{})

	paging = &sqls.Paging{
		Page:  cnd.Paging.Page,
		Limit: cnd.Paging.Limit,
		Total: count,
	}
	return
}

func (r *userDislikeRepository) FindUserCenterDislikePage(db *gorm.DB, userId int64, page, limit int) (list []resp.UserCenterDislikeResponse, paging *sqls.Paging, err error) {
	baseQuery := db.Table("t_user_dislike AS d").
		Joins("INNER JOIN t_topic t ON t.id = d.entity_id").
		Where("d.user_id = ?", userId).
		Where("d.entity_type = ?", constants.EntityTopic).
		Where("d.status = ?", 1).
		Where("d.user_id <> t.user_id").
		Where("t.status = ?", constants.StatusOk)

	var total int64
	if err = baseQuery.Count(&total).Error; err != nil {
		return
	}

	err = baseQuery.
		Select("d.id, d.user_id, d.entity_id, d.entity_type, t.title, t.content, t.user_id AS topic_user_id, d.create_time").
		Order("d.create_time DESC, d.id DESC").
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

func (r *userDislikeRepository) Create(db *gorm.DB, t *models.UserDislike) (err error) {
	err = db.Create(t).Error
	return
}

func (r *userDislikeRepository) Update(db *gorm.DB, t *models.UserDislike) (err error) {
	err = db.Save(t).Error
	return
}

func (r *userDislikeRepository) Updates(db *gorm.DB, id int64, columns map[string]interface{}) (err error) {
	err = db.Model(&models.UserDislike{}).Where("id = ?", id).Updates(columns).Error
	return
}

func (r *userDislikeRepository) UpdateColumn(db *gorm.DB, id int64, name string, value interface{}) (err error) {
	err = db.Model(&models.UserDislike{}).Where("id = ?", id).UpdateColumn(name, value).Error
	return
}

func (r *userDislikeRepository) Delete(db *gorm.DB, id int64) {
	db.Delete(&models.UserDislike{}, "id = ?", id)
}

func (r *userDislikeRepository) Exists(db *gorm.DB, userId int64, entityType string, entityId int64) bool {
	return r.FindOne(db, sqls.NewCnd().Eq("user_id", userId).Eq("entity_id", entityId).Eq("entity_type", entityType)) != nil
}
