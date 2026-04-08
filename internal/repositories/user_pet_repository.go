package repositories

import (
	"bbs-go/internal/models/models"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

type userPetStateRepository struct{}

var UserPetStateRepository = new(userPetStateRepository)

func (r *userPetStateRepository) GetByUserId(db *gorm.DB, userId int64) *models.UserPetState {
	ret := &models.UserPetState{}
	if err := db.Where("user_id = ?", userId).First(ret).Error; err != nil {
		return nil
	}
	return ret
}

func (r *userPetStateRepository) Create(db *gorm.DB, t *models.UserPetState) error {
	return db.Create(t).Error
}

func (r *userPetStateRepository) Update(db *gorm.DB, t *models.UserPetState) error {
	return db.Save(t).Error
}

type userPetRepository struct{}

var UserPetRepository = new(userPetRepository)

func (r *userPetRepository) FindByUserId(db *gorm.DB, userId int64) (list []models.UserPet) {
	db.Where("user_id = ?", userId).Find(&list)
	return
}

func (r *userPetRepository) Get(db *gorm.DB, userId, petId int64) *models.UserPet {
	ret := &models.UserPet{}
	if err := db.Where("user_id = ? and pet_id = ?", userId, petId).First(ret).Error; err != nil {
		return nil
	}
	return ret
}

func (r *userPetRepository) Create(db *gorm.DB, t *models.UserPet) error {
	return db.Create(t).Error
}

func (r *userPetRepository) Update(db *gorm.DB, t *models.UserPet) error {
	return db.Save(t).Error
}

type petDailySettleLogRepository struct{}

var PetDailySettleLogRepository = new(petDailySettleLogRepository)

func (r *petDailySettleLogRepository) GetByUserDay(db *gorm.DB, userId int64, dayName int) *models.PetDailySettleLog {
	ret := &models.PetDailySettleLog{}
	if err := db.Where("user_id = ? and day_name = ?", userId, dayName).First(ret).Error; err != nil {
		return nil
	}
	return ret
}

func (r *petDailySettleLogRepository) Create(db *gorm.DB, t *models.PetDailySettleLog) error {
	return db.Create(t).Error
}

func (r *petDailySettleLogRepository) FindPageByCnd(db *gorm.DB, cnd *sqls.Cnd) (list []models.PetDailySettleLog, paging *sqls.Paging) {
	cnd.Find(db, &list)
	count := cnd.Count(db, &models.PetDailySettleLog{})

	paging = &sqls.Paging{Page: cnd.Paging.Page, Limit: cnd.Paging.Limit, Total: count}
	return
}
