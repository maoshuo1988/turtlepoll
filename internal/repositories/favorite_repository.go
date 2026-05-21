package repositories

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/resp"

	"github.com/mlogclub/simple/sqls"
	"github.com/mlogclub/simple/web/params"
	"gorm.io/gorm"

	"bbs-go/internal/models/models"
)

var FavoriteRepository = newFavoriteRepository()

func newFavoriteRepository() *favoriteRepository {
	return &favoriteRepository{}
}

type favoriteRepository struct {
}

func (r *favoriteRepository) Get(db *gorm.DB, id int64) *models.Favorite {
	ret := &models.Favorite{}
	if err := db.First(ret, "id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *favoriteRepository) Take(db *gorm.DB, where ...interface{}) *models.Favorite {
	ret := &models.Favorite{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *favoriteRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.Favorite) {
	cnd.Find(db, &list)
	return
}

func (r *favoriteRepository) FindOne(db *gorm.DB, cnd *sqls.Cnd) *models.Favorite {
	ret := &models.Favorite{}
	if err := cnd.FindOne(db, &ret); err != nil {
		return nil
	}
	return ret
}

func (r *favoriteRepository) FindPageByParams(db *gorm.DB, params *params.QueryParams) (list []models.Favorite, paging *sqls.Paging) {
	return r.FindPageByCnd(db, &params.Cnd)
}

func (r *favoriteRepository) FindPageByCnd(db *gorm.DB, cnd *sqls.Cnd) (list []models.Favorite, paging *sqls.Paging) {
	cnd.Find(db, &list)
	count := cnd.Count(db, &models.Favorite{})

	paging = &sqls.Paging{
		Page:  cnd.Paging.Page,
		Limit: cnd.Paging.Limit,
		Total: count,
	}
	return
}

func (r *favoriteRepository) FindUserCenterFavoritePage(db *gorm.DB, userId int64, page, limit int) (list []resp.UserCenterFavoriteResponse, paging *sqls.Paging, err error) {
	baseQuery := db.Table("t_favorite AS f").
		Joins("INNER JOIN t_topic t ON t.id = f.entity_id").
		Where("f.user_id = ?", userId).
		Where("f.entity_type = ?", constants.EntityTopic).
		Where("f.user_id <> t.user_id").
		Where("t.status = ?", constants.StatusOk)

	var total int64
	if err = baseQuery.Count(&total).Error; err != nil {
		return
	}

	err = baseQuery.
		Select("f.id, f.user_id, f.entity_id, t.title, t.content, f.create_time").
		Order("f.create_time DESC, f.id DESC").
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

func (r *favoriteRepository) Create(db *gorm.DB, t *models.Favorite) (err error) {
	err = db.Create(t).Error
	return
}

func (r *favoriteRepository) Update(db *gorm.DB, t *models.Favorite) (err error) {
	err = db.Save(t).Error
	return
}

func (r *favoriteRepository) Updates(db *gorm.DB, id int64, columns map[string]interface{}) (err error) {
	err = db.Model(&models.Favorite{}).Where("id = ?", id).Updates(columns).Error
	return
}

func (r *favoriteRepository) UpdateColumn(db *gorm.DB, id int64, name string, value interface{}) (err error) {
	err = db.Model(&models.Favorite{}).Where("id = ?", id).UpdateColumn(name, value).Error
	return
}

func (r *favoriteRepository) Delete(db *gorm.DB, id int64) {
	db.Delete(&models.Favorite{}, "id = ?", id)
}
