package services

import (
	"bbs-go/internal/models/models"
	"bbs-go/internal/pkg/biztime"
	"bbs-go/internal/repositories"
	"errors"
	"fmt"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/common/jsons"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

// PetDailySettleService 登录成功时的“每日结算”。
//
// P0 目标：
// - 同一用户同一天只结算一次（幂等）
// - 结算失败不阻断登录（由调用方吞掉，但返回 errorCode/errorMsg）
// - 重复登录返回 alreadySettled=true + 当日 summary
var PetDailySettleService = newPetDailySettleService()

func newPetDailySettleService() *petDailySettleService {
	return &petDailySettleService{}
}

type petDailySettleService struct{}

type DailySettleItem struct {
	Type   string         `json:"type"`
	Amount int64          `json:"amount"`
	Desc   string         `json:"desc"`
	Meta   map[string]any `json:"meta,omitempty"`
}

type DailySettlePetInfo struct {
	PetId int64 `json:"petId"`
	Level int   `json:"level"`
}

type DailySettleStreakInfo struct {
	LoginStreak int `json:"loginStreak"`
}

type DailySettleResult struct {
	Date           string                `json:"date"`
	AlreadySettled bool                  `json:"alreadySettled"`
	BalanceBefore  int64                 `json:"balanceBefore"`
	BalanceAfter   int64                 `json:"balanceAfter"`
	Items          []DailySettleItem     `json:"items"`
	Streak         DailySettleStreakInfo `json:"streak"`
	Pet            DailySettlePetInfo    `json:"pet"`
	ErrorCode      string                `json:"errorCode,omitempty"`
	ErrorMsg       string                `json:"errorMsg,omitempty"`
}

func (s *petDailySettleService) SettleOnLogin(userId int64) *DailySettleResult {
	nowCST := biztime.NowInCST()
	dayName := biztime.DayNameCST(nowCST)
	dateStr := biztime.DateStringCST(nowCST)

	// 先查幂等记录
	if log := repositories.PetDailySettleLogRepository.GetByUserDay(sqls.DB(), userId, dayName); log != nil {
		var cached DailySettleResult
		_ = jsons.Parse(log.DetailJSON, &cached)
		cached.Date = dateStr
		cached.AlreadySettled = true
		return &cached
	}

	// P0：最小结算 = base_checkin(100) + spark_reward_raw + spark_bonus(可选)
	ret := &DailySettleResult{Date: dateStr, AlreadySettled: false}
	// balance before
	uc, err := UserCoinService.GetOrCreate(userId)
	if err != nil {
		ret.ErrorCode = "COIN_ACCOUNT_ERR"
		ret.ErrorMsg = err.Error()
		return ret
	}
	ret.BalanceBefore = uc.Balance

	err = sqls.DB().Transaction(func(tx *gorm.DB) error {
		// double-check 幂等（事务内）
		if log := repositories.PetDailySettleLogRepository.GetByUserDay(tx, userId, dayName); log != nil {
			return errors.New("ALREADY_SETTLED")
		}

		var checkIn *models.CheckIn
		checkIn, err = CheckInService.EnsureLoginStreak(userId)
		if err != nil {
			return err
		}

		items := make([]DailySettleItem, 0, 5)
		balanceBefore := uc.Balance
		balanceAfter := balanceBefore

		// base_checkin
		base := int64(100)
		items = append(items, DailySettleItem{Type: "base_checkin", Amount: base, Desc: "每日登录基础奖励"})
		balanceAfter += base

		// 写入金币账户 + 流水（复用 Mint 语义：这里用 remark 表示来源；BizType 仍是 MINT）
		// TODO: 后续如需审计分类，可扩展 UserCoinLog 增加 BizType=DAILY_SETTLE。
		_, err = UserCoinService.Mint(0, userId, base, "daily settle | base_checkin")
		if err != nil {
			return err
		}

		loginStreak := 0
		if checkIn != nil {
			loginStreak = checkIn.ConsecutiveDays
		}
		ret.Streak = DailySettleStreakInfo{LoginStreak: loginStreak}

		sparkRaw := PetSparkService.ComputeSparkRaw(loginStreak)
		if sparkRaw > 0 {
			items = append(items, DailySettleItem{
				Type:   "spark_reward",
				Amount: sparkRaw,
				Desc:   "连续登录火花奖励",
				Meta: map[string]any{
					"loginStreak": loginStreak,
				},
			})
			balanceAfter += sparkRaw
			if _, err := UserCoinService.Mint(0, userId, sparkRaw, fmt.Sprintf("daily settle | spark_reward | streak=%d", loginStreak)); err != nil {
				return err
			}
		}

		petLevel := 1
		var state *models.UserPetState
		state, _ = UserPetService.GetOrCreateState(userId)
		if state != nil && state.EquippedPetId > 0 {
			if userPet := repositories.UserPetRepository.Get(tx, userId, state.EquippedPetId); userPet != nil && userPet.Level > 0 {
				petLevel = userPet.Level
			}
			pet := PetDefinitionService.Get(state.EquippedPetId)
			if pet != nil {
				abilities := PetDefinitionService.GetAbilities(pet)
				if balanceBefore < 0 {
					if rawAbility, ok := abilities["debt_subsidy"]; ok && rawAbility != nil {
						fc := FeatureCatalogService.GetByFeatureKey("debt_subsidy")
						if fc != nil && fc.Enabled {
							params, err := PetBalanceFeatureService.DecodeDebtSubsidyParams(rawAbility)
							if err != nil {
								return err
							}
							amount := PetBalanceFeatureService.ComputeDebtSubsidy(balanceBefore, *params)
							if amount > 0 {
								items = append(items, DailySettleItem{
									Type:   "debt_subsidy",
									Amount: amount,
									Desc:   "欠款补贴",
									Meta: map[string]any{
										"balanceBefore": balanceBefore,
										"rate":          params.SubsidyRate,
										"cap":           params.CapPerDay,
										"petId":         state.EquippedPetId,
										"featureKey":    "debt_subsidy",
									},
								})
								balanceAfter += amount
								if _, err := UserCoinService.Mint(0, userId, amount, fmt.Sprintf("daily settle | debt_subsidy | petId=%d", state.EquippedPetId)); err != nil {
									return err
								}
							}
						}
					}
				} else if balanceBefore > 0 {
					if rawAbility, ok := abilities["deposit_interest"]; ok && rawAbility != nil {
						fc := FeatureCatalogService.GetByFeatureKey("deposit_interest")
						if fc != nil && fc.Enabled {
							params, err := PetBalanceFeatureService.DecodeDepositInterestParams(rawAbility)
							if err != nil {
								return err
							}
							amount := PetBalanceFeatureService.ComputeDepositInterest(balanceBefore, *params)
							if amount > 0 {
								items = append(items, DailySettleItem{
									Type:   "deposit_interest",
									Amount: amount,
									Desc:   "存款生息",
									Meta: map[string]any{
										"balanceBefore": balanceBefore,
										"rate":          params.InterestRate,
										"cap":           params.CapPerDay,
										"petId":         state.EquippedPetId,
										"featureKey":    "deposit_interest",
									},
								})
								balanceAfter += amount
								if _, err := UserCoinService.Mint(0, userId, amount, fmt.Sprintf("daily settle | deposit_interest | petId=%d", state.EquippedPetId)); err != nil {
									return err
								}
							}
						}
					}
				}
				if sparkRaw > 0 {
					rawAbility, ok := abilities["spark_multiplier"]
					if ok && rawAbility != nil {
						fc := FeatureCatalogService.GetByFeatureKey("spark_multiplier")
						if fc != nil && fc.Enabled {
							params, err := PetSparkService.DecodeSparkMultiplierParams(rawAbility)
							if err != nil {
								return err
							}
							if params.Enabled || (params.Base > 0 || params.BaseMultiplier > 0 || params.MultiplierBase > 0) {
								extra, final := PetSparkService.ApplySparkMultiplier(sparkRaw, petLevel, *params)
								if extra > 0 {
									items = append(items, DailySettleItem{
										Type:   "spark_bonus",
										Amount: extra,
										Desc:   "火花倍率额外奖励",
										Meta: map[string]any{
											"raw":         sparkRaw,
											"final":       final,
											"petId":       state.EquippedPetId,
											"level":       petLevel,
											"loginStreak": loginStreak,
										},
									})
									balanceAfter += extra
									if _, err := UserCoinService.Mint(0, userId, extra, fmt.Sprintf("daily settle | spark_bonus | petId=%d | level=%d", state.EquippedPetId, petLevel)); err != nil {
										return err
									}
								}
							}
						}
					}
				}
			}
		}

		// 再取一次余额作为 after
		uc2, err := repositories.UserCoinRepository.GetOrCreate(tx, userId)
		if err != nil {
			return err
		}
		ret.BalanceBefore = balanceBefore
		ret.BalanceAfter = uc2.Balance
		ret.Items = items
		if state != nil {
			ret.Pet = DailySettlePetInfo{PetId: state.EquippedPetId, Level: petLevel}
		}

		// 写幂等日志
		log := &models.PetDailySettleLog{
			UserId:     userId,
			DayName:    dayName,
			DetailJSON: jsons.ToJsonStr(ret),
			CreateTime: dates.NowTimestamp(),
		}
		return repositories.PetDailySettleLogRepository.Create(tx, log)
	})
	if err != nil {
		if err.Error() == "ALREADY_SETTLED" {
			// 并发下已结算：读缓存返回
			if log := repositories.PetDailySettleLogRepository.GetByUserDay(sqls.DB(), userId, dayName); log != nil {
				var cached DailySettleResult
				_ = jsons.Parse(log.DetailJSON, &cached)
				cached.Date = dateStr
				cached.AlreadySettled = true
				return &cached
			}
		}
		ret.ErrorCode = "DAILY_SETTLE_ERR"
		ret.ErrorMsg = err.Error()
	}
	return ret
}
