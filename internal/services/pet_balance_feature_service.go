package services

import (
	"bbs-go/internal/models/models"
	"bbs-go/internal/repositories"
	"errors"
	"math"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/common/jsons"
	"gorm.io/gorm"
)

var PetBalanceFeatureService = newPetBalanceFeatureService()

func newPetBalanceFeatureService() *petBalanceFeatureService {
	return &petBalanceFeatureService{}
}

type petBalanceFeatureService struct{}

type DebtParams struct {
	Enabled             bool   `json:"enabled"`
	DebtFloor           int64  `json:"debtFloor"`
	ForbidEquipWhenDebt bool   `json:"forbidEquipWhenDebt"`
	ErrorCode           string `json:"errorCode"`
	MinBalance          int64  `json:"min_balance"`
	LockSwitchWhenDebt  bool   `json:"lock_switch_when_debt"`
}

type DebtSubsidyParams struct {
	Enabled        bool    `json:"enabled"`
	SubsidyRate    float64 `json:"subsidyRate"`
	CapPerDay      int64   `json:"capPerDay"`
	RateBase       float64 `json:"rate_base"`
	DailyCapAmount int64   `json:"daily_cap_amount"`
}

type DepositInterestParams struct {
	Enabled        bool    `json:"enabled"`
	InterestRate   float64 `json:"interestRate"`
	CapPerDay      int64   `json:"capPerDay"`
	RateBase       float64 `json:"rate_base"`
	DailyCapAmount int64   `json:"daily_cap_amount"`
}

func (s *petBalanceFeatureService) DecodeDebtParams(v any) (*DebtParams, error) {
	if v == nil {
		return nil, errors.New("empty params")
	}
	var p DebtParams
	if err := jsons.Parse(jsons.ToJsonStr(v), &p); err != nil {
		return nil, err
	}
	if p.DebtFloor == 0 && p.MinBalance < 0 {
		p.DebtFloor = p.MinBalance
	}
	if !p.ForbidEquipWhenDebt && p.LockSwitchWhenDebt {
		p.ForbidEquipWhenDebt = true
	}
	if p.ErrorCode == "" {
		p.ErrorCode = "DEBT_UNPAID"
	}
	if p.DebtFloor > 0 {
		return nil, errors.New("debtFloor must be non-positive")
	}
	return &p, nil
}

func (s *petBalanceFeatureService) DecodeDebtSubsidyParams(v any) (*DebtSubsidyParams, error) {
	if v == nil {
		return nil, errors.New("empty params")
	}
	var p DebtSubsidyParams
	if err := jsons.Parse(jsons.ToJsonStr(v), &p); err != nil {
		return nil, err
	}
	if p.SubsidyRate == 0 && p.RateBase > 0 {
		p.SubsidyRate = p.RateBase
	}
	if p.CapPerDay == 0 && p.DailyCapAmount > 0 {
		p.CapPerDay = p.DailyCapAmount
	}
	if p.SubsidyRate < 0 || p.SubsidyRate > 1 {
		return nil, errors.New("subsidyRate must be between 0 and 1")
	}
	if p.CapPerDay < 0 {
		return nil, errors.New("capPerDay must be non-negative")
	}
	return &p, nil
}

func (s *petBalanceFeatureService) DecodeDepositInterestParams(v any) (*DepositInterestParams, error) {
	if v == nil {
		return nil, errors.New("empty params")
	}
	var p DepositInterestParams
	if err := jsons.Parse(jsons.ToJsonStr(v), &p); err != nil {
		return nil, err
	}
	if p.InterestRate == 0 && p.RateBase > 0 {
		p.InterestRate = p.RateBase
	}
	if p.CapPerDay == 0 && p.DailyCapAmount > 0 {
		p.CapPerDay = p.DailyCapAmount
	}
	if p.InterestRate < 0 || p.InterestRate > 1 {
		return nil, errors.New("interestRate must be between 0 and 1")
	}
	if p.CapPerDay < 0 {
		return nil, errors.New("capPerDay must be non-negative")
	}
	return &p, nil
}

func (s *petBalanceFeatureService) ComputeDebtSubsidy(balance int64, params DebtSubsidyParams) int64 {
	if balance >= 0 {
		return 0
	}
	if !params.Enabled && params.SubsidyRate == 0 {
		return 0
	}
	rate := params.SubsidyRate
	if rate <= 0 {
		return 0
	}
	amount := int64(math.Floor(float64(-balance) * rate))
	if amount < 0 {
		amount = 0
	}
	if params.CapPerDay > 0 && amount > params.CapPerDay {
		amount = params.CapPerDay
	}
	return amount
}

func (s *petBalanceFeatureService) ComputeDepositInterest(balance int64, params DepositInterestParams) int64 {
	if balance <= 0 {
		return 0
	}
	if !params.Enabled && params.InterestRate == 0 {
		return 0
	}
	rate := params.InterestRate
	if rate <= 0 {
		return 0
	}
	amount := int64(math.Floor(float64(balance) * rate))
	if amount < 0 {
		amount = 0
	}
	if params.CapPerDay > 0 && amount > params.CapPerDay {
		amount = params.CapPerDay
	}
	return amount
}

func (s *petBalanceFeatureService) ResolveDebtFloor(userId int64) int64 {
	return s.ResolveDebtFloorForTx(nil, userId)
}

func (s *petBalanceFeatureService) ResolveDebtFloorForTx(tx *gorm.DB, userId int64) int64 {
	if userId <= 0 {
		return 0
	}
	var (
		state *models.UserPetState
		err   error
	)
	if tx != nil {
		state = repositories.UserPetStateRepository.GetByUserId(tx, userId)
		if state == nil {
			now := dates.NowTimestamp()
			state = &models.UserPetState{
				UserId:        userId,
				EquippedPetId: 0,
				EquipDayName:  0,
				CreateTime:    now,
				UpdateTime:    now,
			}
			if err = repositories.UserPetStateRepository.Create(tx, state); err != nil {
				return 0
			}
		}
	} else {
		state, err = UserPetService.GetOrCreateState(userId)
	}
	if err != nil || state == nil || state.EquippedPetId <= 0 {
		return 0
	}
	pet := PetDefinitionService.Get(state.EquippedPetId)
	if pet == nil {
		return 0
	}
	abilities := PetDefinitionService.GetAbilities(pet)
	raw, ok := abilities["debt"]
	if !ok || raw == nil {
		return 0
	}
	fc := FeatureCatalogService.GetByFeatureKey("debt")
	if fc == nil || !fc.Enabled {
		return 0
	}
	params, err := s.DecodeDebtParams(raw)
	if err != nil {
		return 0
	}
	if !params.Enabled && params.DebtFloor == 0 {
		return 0
	}
	return params.DebtFloor
}

func (s *petBalanceFeatureService) CanSpend(balance, amount, debtFloor int64) bool {
	if amount <= 0 {
		return true
	}
	return balance-amount >= debtFloor
}
