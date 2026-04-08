package services

import (
	"bbs-go/internal/models/models"
	"bbs-go/internal/pkg/biztime"
	"bbs-go/internal/repositories"
	"errors"

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

type DailySettleResult struct {
	Date           string             `json:"date"`
	AlreadySettled bool               `json:"alreadySettled"`
	BalanceBefore  int64              `json:"balanceBefore"`
	BalanceAfter   int64              `json:"balanceAfter"`
	Items          []DailySettleItem  `json:"items"`
	Pet            DailySettlePetInfo `json:"pet"`
	ErrorCode      string             `json:"errorCode,omitempty"`
	ErrorMsg       string             `json:"errorMsg,omitempty"`
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

	// P0：最小结算 = base_checkin(100) + pet_signin_bonus(可选)
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

		items := make([]DailySettleItem, 0, 2)
		balanceBefore := uc.Balance
		balanceAfter := balanceBefore

		// base_checkin
		base := int64(100)
		items = append(items, DailySettleItem{Type: "base_checkin", Amount: base, Desc: "每日登录基础奖励"})
		balanceAfter += base

		// 写入金币账户 + 流水（复用 Mint 语义：这里用 remark 表示来源；BizType 仍是 MINT）
		// TODO: 后续如需审计分类，可扩展 UserCoinLog 增加 BizType=DAILY_SETTLE。
		_, err := UserCoinService.Mint(0, userId, base, "daily settle | base_checkin")
		if err != nil {
			return err
		}

		// pet_signin_bonus（可选）
		state, _ := UserPetService.GetOrCreateState(userId)
		if state != nil && state.EquippedPetId > 0 {
			checkIn := CheckInService.GetByUserId(userId)
			if checkIn != nil {
				_ = PetSigninBonusService.GrantByCheckIn(userId, checkIn, state.EquippedPetId)
				// 这里暂时无法拿到具体 bonus 数额（GrantByCheckIn 内部直接入账），P0 先不回 items。
				// 后续应把 GrantByCheckIn 改为“计算 + 返回 bonus，再由 dailySettle 统一入账与记录明细”。
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
		ret.Pet = DailySettlePetInfo{PetId: state.EquippedPetId, Level: 1}

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
