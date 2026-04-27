package repositories

import (
	"bbs-go/internal/models/models"

	"github.com/mlogclub/simple/sqls"
	"github.com/mlogclub/simple/web/params"
	"gorm.io/gorm"
)

var UserCoinLogRepository = newUserCoinLogRepository()

func newUserCoinLogRepository() *userCoinLogRepository {
	return &userCoinLogRepository{}
}

type userCoinLogRepository struct{}

func (r *userCoinLogRepository) Get(db *gorm.DB, id int64) *models.UserCoinLog {
	ret := &models.UserCoinLog{}
	if err := db.First(ret, "id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *userCoinLogRepository) Take(db *gorm.DB, where ...interface{}) *models.UserCoinLog {
	ret := &models.UserCoinLog{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *userCoinLogRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.UserCoinLog) {
	cnd.Find(db, &list)
	return
}

func (r *userCoinLogRepository) FindOne(db *gorm.DB, cnd *sqls.Cnd) *models.UserCoinLog {
	ret := &models.UserCoinLog{}
	if err := cnd.FindOne(db, &ret); err != nil {
		return nil
	}
	return ret
}

func (r *userCoinLogRepository) FindPageByParams(db *gorm.DB, params *params.QueryParams) (list []models.UserCoinLog, paging *sqls.Paging) {
	return r.FindPageByCnd(db, &params.Cnd)
}

func (r *userCoinLogRepository) FindPageByCnd(db *gorm.DB, cnd *sqls.Cnd) (list []models.UserCoinLog, paging *sqls.Paging) {
	cnd.Find(db, &list)
	count := cnd.Count(db, &models.UserCoinLog{})

	paging = &sqls.Paging{
		Page:  cnd.Paging.Page,
		Limit: cnd.Paging.Limit,
		Total: count,
	}
	return
}

func (r *userCoinLogRepository) Count(db *gorm.DB, cnd *sqls.Cnd) int64 {
	return cnd.Count(db, &models.UserCoinLog{})
}

func (r *userCoinLogRepository) Create(db *gorm.DB, t *models.UserCoinLog) error {
	return db.Create(t).Error
}
