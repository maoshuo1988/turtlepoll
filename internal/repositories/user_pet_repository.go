package repositories

import (
	"bbs-go/internal/models/models"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

type petBetRewardLogRepository struct{}

var PetBetRewardLogRepository = new(petBetRewardLogRepository)

func (r *petBetRewardLogRepository) GetByUserDayRewardType(db *gorm.DB, userId int64, dayName int, rewardType string) *models.PetBetRewardLog {
	ret := &models.PetBetRewardLog{}
	if err := db.Where("user_id = ? and day_name = ? and reward_type = ?", userId, dayName, rewardType).First(ret).Error; err != nil {
		return nil
	}
	return ret
}

// CreateIfAbsent 按幂等键插入，已存在时返回 created=false 且不报错。
func (r *petBetRewardLogRepository) CreateIfAbsent(db *gorm.DB, t *models.PetBetRewardLog) (created bool, err error) {
	if t == nil {
		return false, nil
	}
	res := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "day_name"}, {Name: "reward_type"}},
		DoNothing: true,
	}).Create(t)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *petBetRewardLogRepository) Update(db *gorm.DB, t *models.PetBetRewardLog) error {
	return db.Save(t).Error
}

func (r *petBetRewardLogRepository) Delete(db *gorm.DB, id int64) error {
	return db.Delete(&models.PetBetRewardLog{}, "id = ?", id).Error
}
