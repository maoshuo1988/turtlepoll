package repositories

import (
	"bbs-go/internal/models"

	"gorm.io/gorm"
)

var BattleSettlementRepository = newBattleSettlementRepository()

func newBattleSettlementRepository() *battleSettlementRepository {
	return &battleSettlementRepository{}
}

type battleSettlementRepository struct{}

func (r *battleSettlementRepository) TakeByBattleId(db *gorm.DB, battleId int64) *models.BattleSettlement {
	ret := &models.BattleSettlement{}
	if err := db.Take(ret, "battle_id = ?", battleId).Error; err != nil {
		return nil
	}
	return ret
}

func (r *battleSettlementRepository) Create(db *gorm.DB, t *models.BattleSettlement) error {
	return db.Create(t).Error
}

func (r *battleSettlementRepository) CreateItem(db *gorm.DB, t *models.BattleSettlementItem) error {
	return db.Create(t).Error
}

func (r *battleSettlementRepository) TakeItemByBattleUser(db *gorm.DB, battleId, userId int64) *models.BattleSettlementItem {
	ret := &models.BattleSettlementItem{}
	if err := db.Take(ret, "battle_id = ? AND user_id = ?", battleId, userId).Error; err != nil {
		return nil
	}
	return ret
}

func (r *battleSettlementRepository) ListItems(db *gorm.DB, settlementId int64) ([]*models.BattleSettlementItem, error) {
	var list []*models.BattleSettlementItem
	if err := db.Where("settlement_id = ?", settlementId).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
