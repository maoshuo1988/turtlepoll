package repositories

import (
	"bbs-go/internal/models/models"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

type petDefinitionRepository struct{}

var PetDefinitionRepository = new(petDefinitionRepository)

func (r *petDefinitionRepository) Get(db *gorm.DB, id int64) *models.PetDefinition {
	ret := &models.PetDefinition{}
	if err := db.First(ret, id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *petDefinitionRepository) GetByPetKey(db *gorm.DB, petKey string) *models.PetDefinition {
	ret := &models.PetDefinition{}
	if err := db.Where("pet_key = ?", petKey).First(ret).Error; err != nil {
		return nil
	}
	return ret
}

func (r *petDefinitionRepository) Take(db *gorm.DB, where ...interface{}) *models.PetDefinition {
	ret := &models.PetDefinition{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *petDefinitionRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.PetDefinition) {
	cnd.Find(db, &list)
	return
}

func (r *petDefinitionRepository) FindOne(db *gorm.DB, cnd *sqls.Cnd) *models.PetDefinition {
	ret := &models.PetDefinition{}
	if err := cnd.FindOne(db, &ret); err != nil {
		return nil
	}
	return ret
}

func (r *petDefinitionRepository) FindPageByCnd(db *gorm.DB, cnd *sqls.Cnd) (list []models.PetDefinition, paging *sqls.Paging) {
	cnd.Find(db, &list)
	count := cnd.Count(db, &models.PetDefinition{})

	paging = &sqls.Paging{
		Page:  cnd.Paging.Page,
		Limit: cnd.Paging.Limit,
		Total: count,
	}
	return
}

func (r *petDefinitionRepository) Create(db *gorm.DB, t *models.PetDefinition) error {
	return db.Create(t).Error
}

func (r *petDefinitionRepository) Update(db *gorm.DB, t *models.PetDefinition) error {
	return db.Save(t).Error
}

func (r *petDefinitionRepository) Updates(db *gorm.DB, id int64, columns map[string]interface{}) error {
	return db.Model(&models.PetDefinition{}).Where("id = ?", id).Updates(columns).Error
}
