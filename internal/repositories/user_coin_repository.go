package repositories

import (
	"bbs-go/internal/models/models"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

var UserCoinRepository = newUserCoinRepository()

func newUserCoinRepository() *userCoinRepository {
	return &userCoinRepository{}
}

type userCoinRepository struct{}

func (r *userCoinRepository) Take(db *gorm.DB, where ...interface{}) *models.UserCoin {
	ret := &models.UserCoin{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

// GetOrCreate 确保用户金币账户存在。
func (r *userCoinRepository) GetOrCreate(db *gorm.DB, userId int64) (*models.UserCoin, error) {
	existing := r.Take(db, "user_id = ?", userId)
	if existing != nil {
		return existing, nil
	}
	uc := &models.UserCoin{UserId: userId}
	if err := db.Create(uc).Error; err != nil {
		// 并发下可能被其他请求创建了，重试读取
		existing = r.Take(db, "user_id = ?", userId)
		if existing != nil {
			return existing, nil
		}
		return nil, err
	}
	return uc, nil
}

func (r *userCoinRepository) Update(db *gorm.DB, t *models.UserCoin) error {
	return db.Save(t).Error
}

// DB helper for convenience
func (r *userCoinRepository) DB() *gorm.DB {
	return sqls.DB()
}
