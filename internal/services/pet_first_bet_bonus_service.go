package services

import (
	"bbs-go/internal/models/models"
	"bbs-go/internal/pkg/biztime"
	"bbs-go/internal/repositories"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/common/jsons"
	"gorm.io/gorm"
)

var PetFirstBetBonusService = newPetFirstBetBonusService()

func newPetFirstBetBonusService() *petFirstBetBonusService {
	return &petFirstBetBonusService{}
}

type petFirstBetBonusService struct{}

type FirstBetBonusResult struct {
	Granted bool   `json:"granted"`
	Amount  int64  `json:"amount"`
	Reason  string `json:"reason,omitempty"`
}

type firstBetBonusParams struct {
	Enabled      bool  `json:"enabled"`
	Amount       int64 `json:"amount"`
	BonusCoins   int64 `json:"bonusCoins"`
	MinBetAmount int64 `json:"minBetAmount"`
}

func (s *petFirstBetBonusService) GrantOnBetPlaced(tx *gorm.DB, userId, marketId, betId, betAmount int64) (*FirstBetBonusResult, error) {
	ret := &FirstBetBonusResult{Granted: false, Amount: 0}
	if tx == nil {
		return ret, errors.New("tx is required")
	}
	if userId <= 0 || betId <= 0 {
		ret.Reason = "INVALID_INPUT"
		return ret, nil
	}

	state := repositories.UserPetStateRepository.GetByUserId(tx, userId)
	if state == nil || state.EquippedPetId <= 0 {
		ret.Reason = "NOT_EQUIPPED"
		return ret, nil
	}

	pet := repositories.PetDefinitionRepository.Get(tx, state.EquippedPetId)
	if pet == nil {
		ret.Reason = "PET_NOT_FOUND"
		return ret, nil
	}

	fc := FeatureCatalogService.GetByFeatureKey("first_bet_bonus")
	if fc == nil || !fc.Enabled {
		ret.Reason = "FEATURE_DISABLED"
		return ret, nil
	}

	abilities := PetDefinitionService.GetAbilities(pet)
	raw, ok := abilities["first_bet_bonus"]
	if !ok || raw == nil {
		ret.Reason = "ABILITY_MISSING"
		return ret, nil
	}

	params, err := decodeFirstBetBonusParams(raw)
	if err != nil {
		ret.Reason = "INVALID_PARAMS"
		return ret, nil
	}
	if !params.Enabled {
		ret.Reason = "ABILITY_DISABLED"
		return ret, nil
	}
	if params.MinBetAmount > 0 && betAmount < params.MinBetAmount {
		ret.Reason = "BELOW_MIN_BET"
		return ret, nil
	}
	if params.Amount <= 0 {
		ret.Reason = "ZERO_AMOUNT"
		return ret, nil
	}

	dayName := biztime.DayNameCST(time.Now())
	now := dates.NowTimestamp()
	log := &models.PetBetRewardLog{
		UserId:     userId,
		DayName:    dayName,
		RewardType: "first_bet_bonus",
		MarketId:   marketId,
		BetId:      betId,
		PetId:      pet.Id,
		Amount:     params.Amount,
		Status:     "PENDING",
		DetailJSON: jsons.ToJsonStr(map[string]any{"petKey": pet.PetKey, "featureKey": "first_bet_bonus"}),
		CreateTime: now,
		UpdateTime: now,
	}
	created, err := repositories.PetBetRewardLogRepository.CreateIfAbsent(tx, log)
	if err != nil {
		return ret, err
	}
	if !created {
		ret.Reason = "ALREADY_GRANTED"
		return ret, nil
	}

	remark := fmt.Sprintf("pet first bet bonus | day=%d | petId=%d", dayName, pet.Id)
	if _, _, err := UserCoinService.AddReward(tx, userId, "PET_FIRST_BET_BONUS", betId, params.Amount, remark); err != nil {
		_ = repositories.PetBetRewardLogRepository.Delete(tx, log.Id)
		return ret, err
	}

	log.Status = "SUCCESS"
	log.UpdateTime = dates.NowTimestamp()
	if err := repositories.PetBetRewardLogRepository.Update(tx, log); err != nil {
		return ret, err
	}

	ret.Granted = true
	ret.Amount = params.Amount
	ret.Reason = "GRANTED"
	return ret, nil
}

func (s *petFirstBetBonusService) PushMessage(userId int64, bonus *FirstBetBonusResult, templateCode, sceneName, targetTitle, detailUrl, bizId string, extraData map[string]any) {
	if bonus == nil || !bonus.Granted || bonus.Amount <= 0 {
		return
	}
	if strings.TrimSpace(sceneName) == "" {
		sceneName = "下注"
	}
	if strings.TrimSpace(targetTitle) == "" {
		targetTitle = sceneName
	}
	params := map[string]string{
		"amount":      strconv.FormatInt(bonus.Amount, 10),
		"sceneName":   sceneName,
		"targetTitle": targetTitle,
		"detailUrl":   detailUrl,
	}
	if extraData == nil {
		extraData = map[string]any{}
	}
	extraData["amount"] = bonus.Amount
	extraData["sceneName"] = sceneName
	extraData["targetTitle"] = targetTitle
	_, _ = MessageNotifyService.PushByTemplate(MessageNotifyPushInput{
		BusinessCode:   MessageNotifyBusinessReward,
		TemplateCode:   templateCode,
		UserId:         userId,
		Params:         params,
		ExtraData:      extraData,
		BizId:          bizId,
		IdempotencyKey: fmt.Sprintf("%s:%d:%s", templateCode, userId, bizId),
	})
}

func decodeFirstBetBonusParams(v any) (*firstBetBonusParams, error) {
	if v == nil {
		return nil, errors.New("empty first_bet_bonus params")
	}
	var p firstBetBonusParams
	if err := jsons.Parse(jsons.ToJsonStr(v), &p); err != nil {
		return nil, err
	}
	if p.Amount == 0 && p.BonusCoins > 0 {
		p.Amount = p.BonusCoins
	}
	if p.MinBetAmount < 0 {
		return nil, errors.New("minBetAmount must be non-negative")
	}
	if p.Amount < 0 {
		return nil, errors.New("amount must be non-negative")
	}
	if !p.Enabled {
		if b, ok := v.(map[string]any); ok {
			if enabledRaw, ok := b["enabled"]; !ok || strings.TrimSpace(jsons.ToJsonStr(enabledRaw)) == "" {
				p.Enabled = true
			}
		}
	}
	return &p, nil
}
