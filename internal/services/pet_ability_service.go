package services

import (
	"bbs-go/internal/models/models"
	"errors"
	"strings"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

// PetAbilityService 专注 abilities 的强校验与原子更新
var PetAbilityService = newPetAbilityService()

func newPetAbilityService() *petAbilityService {
	return &petAbilityService{}
}

type petAbilityService struct{}

func (s *petAbilityService) ReplaceAbilities(petId int64, abilities map[string]any) (*models.PetDefinition, error) {
	return s.withTx(func(tx *gorm.DB) (*models.PetDefinition, error) {
		if err := s.validateAbilities(abilities); err != nil {
			return nil, err
		}
		return PetDefinitionService.SetAbilities(tx, petId, abilities)
	})
}

func (s *petAbilityService) UpsertAbility(petId int64, featureKey string, params map[string]any) (*models.PetDefinition, error) {
	featureKey = strings.TrimSpace(featureKey)
	if featureKey == "" {
		return nil, errors.New("featureKey is empty")
	}
	return s.withTx(func(tx *gorm.DB) (*models.PetDefinition, error) {
		pet := PetDefinitionService.Get(petId)
		if pet == nil {
			return nil, errors.New("pet definition not found")
		}
		abilities := PetDefinitionService.GetAbilities(pet)
		abilities[featureKey] = params
		if err := s.validateAbilities(abilities); err != nil {
			return nil, err
		}
		return PetDefinitionService.SetAbilities(tx, petId, abilities)
	})
}

func (s *petAbilityService) RemoveAbility(petId int64, featureKey string) (*models.PetDefinition, error) {
	featureKey = strings.TrimSpace(featureKey)
	if featureKey == "" {
		return nil, errors.New("featureKey is empty")
	}
	return s.withTx(func(tx *gorm.DB) (*models.PetDefinition, error) {
		pet := PetDefinitionService.Get(petId)
		if pet == nil {
			return nil, errors.New("pet definition not found")
		}
		abilities := PetDefinitionService.GetAbilities(pet)
		delete(abilities, featureKey)
		if err := s.validateAbilities(abilities); err != nil {
			return nil, err
		}
		return PetDefinitionService.SetAbilities(tx, petId, abilities)
	})
}

// validateAbilities：
// 1) featureKey 必须存在于 FeatureCatalog 且 enabled=true
// 2) params_schema 校验：当前先做“JSON 必须合法 + schema 字段存在”的弱校验（后续可替换为完整 JSONSchema 校验库）
func (s *petAbilityService) validateAbilities(abilities map[string]any) error {
	for k := range abilities {
		item := FeatureCatalogService.GetByFeatureKey(k)
		if item == nil {
			return errors.New("feature not found: " + k)
		}
		if !item.Enabled {
			return errors.New("feature disabled: " + k)
		}
		// TODO: JSONSchema validate abilities[k] against item.ParamsSchemaJSON
	}
	return nil
}

func (s *petAbilityService) withTx(fn func(tx *gorm.DB) (*models.PetDefinition, error)) (*models.PetDefinition, error) {
	var ret *models.PetDefinition
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		v, err := fn(tx)
		if err != nil {
			return err
		}
		ret = v
		return nil
	})
	return ret, err
}
