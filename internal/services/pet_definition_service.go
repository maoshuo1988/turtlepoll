package services

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/models"
	"bbs-go/internal/repositories"
	"errors"
	"strings"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/common/jsons"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

var PetDefinitionService = newPetDefinitionService()

func newPetDefinitionService() *petDefinitionService {
	return &petDefinitionService{}
}

type petDefinitionService struct{}

func (s *petDefinitionService) Get(id int64) *models.PetDefinition {
	return repositories.PetDefinitionRepository.Get(sqls.DB(), id)
}

func (s *petDefinitionService) GetByPetKey(petKey string) *models.PetDefinition {
	petKey = strings.TrimSpace(petKey)
	if petKey == "" {
		return nil
	}
	return repositories.PetDefinitionRepository.GetByPetKey(sqls.DB(), petKey)
}

func (s *petDefinitionService) FindPageByCnd(cnd *sqls.Cnd) (list []models.PetDefinition, paging *sqls.Paging) {
	return repositories.PetDefinitionRepository.FindPageByCnd(sqls.DB(), cnd)
}

func (s *petDefinitionService) Create(t *models.PetDefinition) error {
	now := dates.NowTimestamp()
	if t.CreateTime <= 0 {
		t.CreateTime = now
	}
	t.UpdateTime = now
	if t.Status == 0 {
		t.Status = constants.StatusOk
	}
	return repositories.PetDefinitionRepository.Create(sqls.DB(), t)
}

func (s *petDefinitionService) Update(t *models.PetDefinition) error {
	t.UpdateTime = dates.NowTimestamp()
	return repositories.PetDefinitionRepository.Update(sqls.DB(), t)
}

func (s *petDefinitionService) Delete(id int64) error {
	return repositories.PetDefinitionRepository.Updates(sqls.DB(), id, map[string]interface{}{
		"status":      constants.StatusDeleted,
		"update_time": dates.NowTimestamp(),
	})
}

// abilities helpers

func (s *petDefinitionService) GetAbilities(t *models.PetDefinition) map[string]any {
	if t == nil || len(t.AbilitiesJSON) == 0 {
		return map[string]any{}
	}
	var m map[string]any
	_ = jsons.Parse(t.AbilitiesJSON, &m)
	if m == nil {
		return map[string]any{}
	}
	return m
}

func (s *petDefinitionService) SetAbilities(tx *gorm.DB, petId int64, abilities map[string]any) (*models.PetDefinition, error) {
	pet := repositories.PetDefinitionRepository.Get(tx, petId)
	if pet == nil {
		return nil, errors.New("pet definition not found")
	}

	pet.AbilitiesJSON = jsons.ToJsonStr(abilities)
	pet.UpdateTime = dates.NowTimestamp()
	if err := repositories.PetDefinitionRepository.Update(tx, pet); err != nil {
		return nil, err
	}
	return pet, nil
}
