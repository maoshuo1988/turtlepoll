package repositories

import (
	"bbs-go/internal/models/models"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

var MessageNotifyTemplateRepository = newMessageNotifyTemplateRepository()
var UserMessageNotifyRecordRepository = newUserMessageNotifyRecordRepository()

func newMessageNotifyTemplateRepository() *messageNotifyTemplateRepository {
	return &messageNotifyTemplateRepository{}
}

type messageNotifyTemplateRepository struct{}

func (r *messageNotifyTemplateRepository) Get(db *gorm.DB, id int64) *models.MessageNotifyTemplate {
	ret := &models.MessageNotifyTemplate{}
	if err := db.First(ret, "id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *messageNotifyTemplateRepository) GetByCode(db *gorm.DB, businessCode, templateCode string) *models.MessageNotifyTemplate {
	ret := &models.MessageNotifyTemplate{}
	if err := db.Take(ret, "business_code = ? and template_code = ?", businessCode, templateCode).Error; err != nil {
		return nil
	}
	return ret
}

func (r *messageNotifyTemplateRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.MessageNotifyTemplate) {
	cnd.Find(db, &list)
	return
}

func (r *messageNotifyTemplateRepository) Create(db *gorm.DB, t *models.MessageNotifyTemplate) error {
	return db.Create(t).Error
}

func (r *messageNotifyTemplateRepository) Update(db *gorm.DB, t *models.MessageNotifyTemplate) error {
	return db.Save(t).Error
}

func newUserMessageNotifyRecordRepository() *userMessageNotifyRecordRepository {
	return &userMessageNotifyRecordRepository{}
}

type userMessageNotifyRecordRepository struct{}

func (r *userMessageNotifyRecordRepository) Get(db *gorm.DB, id int64) *models.UserMessageNotifyRecord {
	ret := &models.UserMessageNotifyRecord{}
	if err := db.First(ret, "id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *userMessageNotifyRecordRepository) FindOne(db *gorm.DB, cnd *sqls.Cnd) *models.UserMessageNotifyRecord {
	ret := &models.UserMessageNotifyRecord{}
	if err := cnd.FindOne(db, ret); err != nil {
		return nil
	}
	return ret
}

func (r *userMessageNotifyRecordRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.UserMessageNotifyRecord) {
	cnd.Find(db, &list)
	return
}

func (r *userMessageNotifyRecordRepository) Count(db *gorm.DB, cnd *sqls.Cnd) int64 {
	return cnd.Count(db, &models.UserMessageNotifyRecord{})
}

func (r *userMessageNotifyRecordRepository) Create(db *gorm.DB, t *models.UserMessageNotifyRecord) error {
	return db.Create(t).Error
}

func (r *userMessageNotifyRecordRepository) Update(db *gorm.DB, t *models.UserMessageNotifyRecord) error {
	return db.Save(t).Error
}

func (r *userMessageNotifyRecordRepository) Updates(db *gorm.DB, id int64, columns map[string]any) error {
	return db.Model(&models.UserMessageNotifyRecord{}).Where("id = ?", id).Updates(columns).Error
}
