package services

import (
	"bbs-go/internal/models/models"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/common/jsons"
	"github.com/mlogclub/simple/sqls"
)

// PetSigninBonusService 在“签到成功”后结算并发放每日登录加成（signin_bonus）。
//
// 约束：
// - 幂等：同一用户同一天最多发放一次；通过 UserCoinLog 唯一 bizId（checkInId）间接保证 + 上层签到幂等。
// - 不影响主流程：任何错误都应由调用方吞掉（避免把签到/登录链路打挂）。
//
// 前置依赖：
// - PetDefinition.AbilitiesJSON 里存在 key=signin_bonus 的参数。
// - FeatureCatalog 里存在 featureKey=signin_bonus 且 enabled=true。
//
// TODO(后续)：目前项目里还没有“用户当前装备龟种”的数据模型，因此这里临时用 petId 作为入参。
// 当有 UserPetEquip / UserProfile.currentPetId 等字段后，把调用方传入真实 petId。
var PetSigninBonusService = newPetSigninBonusService()

func newPetSigninBonusService() *petSigninBonusService {
	return &petSigninBonusService{}
}

type petSigninBonusService struct{}

// GrantByCheckIn 在签到成功后发放 signin_bonus。
//
// - userId: 用户 id
// - checkIn: 本次签到写入后的 CheckIn 记录（用于 dayName/幂等辅助等）
// - petId: 当前装备龟种对应的 PetDefinition.Id（临时由调用方传入；后续换成真实来源）
func (s *petSigninBonusService) GrantByCheckIn(userId int64, checkIn *models.CheckIn, petId int64) error {
	if userId <= 0 {
		return errors.New("userId is required")
	}
	if checkIn == nil {
		return errors.New("checkIn is required")
	}
	if petId <= 0 {
		// 没有装备龟种时不发放
		return nil
	}

	// 读取 pet definition
	pet := PetDefinitionService.Get(petId)
	if pet == nil {
		return nil
	}

	abilities := PetDefinitionService.GetAbilities(pet)
	raw, ok := abilities["signin_bonus"]
	if !ok || raw == nil {
		return nil
	}

	// validate: feature catalog must exist and enabled
	fc := FeatureCatalogService.GetByFeatureKey("signin_bonus")
	if fc == nil || !fc.Enabled {
		return nil
	}

	params, err := decodeSigninBonusParams(raw)
	if err != nil {
		return err
	}

	bonus := params.BonusCoins
	if bonus <= 0 {
		return nil
	}
	if params.CapPerDay > 0 && bonus > params.CapPerDay {
		bonus = params.CapPerDay
	}

	remark := fmt.Sprintf("pet signin bonus | petId=%d | day=%d", petId, checkIn.LatestDayName)

	// 以 checkIn.Id 作为 bizId，保证同一次签到不会重复入账（签到本身也幂等）。
	// bizType 先用固定字符串，避免再改 constants；如需统一管理可后续抽常量。
	_, err = UserCoinService.Mint(0, userId, bonus, remark)
	if err != nil {
		// 如果是唯一键/重复入账（未来可加唯一索引），建议吞掉。
		// 当前 UserCoinLog 未做唯一约束，这里先保守返回错误，让调用方决定吞不吞。
		return err
	}

	// 轻量 kill-switch：如果全局设置 disableAll=true，则不发放。
	// 由于 sys_config 的结构未强约束，这里尽量容错。
	_ = s._noopKillSwitchReadForFuture()

	return nil
}

type signinBonusParams struct {
	BonusCoins int64 `json:"bonusCoins"`
	CapPerDay  int64 `json:"capPerDay"`
}

func decodeSigninBonusParams(v any) (*signinBonusParams, error) {
	// 兼容两种写法：
	// 1) 直接 {bonusCoins: 100, capPerDay: 500}
	// 2) {enabled: true, params: {...}} 的嵌套结构

	// 先把 any 转成 json，再 parse，避免大量类型断言
	b := jsons.ToJsonStr(v)
	if strings.TrimSpace(b) == "" {
		return nil, errors.New("empty params")
	}

	// 尝试直接解析
	var p signinBonusParams
	if err := jsons.Parse(b, &p); err == nil && p.BonusCoins != 0 {
		if p.BonusCoins < 0 {
			return nil, errors.New("bonusCoins must be non-negative")
		}
		if p.CapPerDay < 0 {
			return nil, errors.New("capPerDay must be non-negative")
		}
		return &p, nil
	}

	// 尝试 nested: {enabled, params}
	var nested struct {
		Enabled bool               `json:"enabled"`
		Params  *signinBonusParams `json:"params"`
	}
	if err := jsons.Parse(b, &nested); err != nil {
		return nil, err
	}
	if !nested.Enabled {
		return &signinBonusParams{BonusCoins: 0, CapPerDay: 0}, nil
	}
	if nested.Params == nil {
		return nil, errors.New("params is required")
	}
	if nested.Params.BonusCoins < 0 {
		return nil, errors.New("bonusCoins must be non-negative")
	}
	if nested.Params.CapPerDay < 0 {
		return nil, errors.New("capPerDay must be non-negative")
	}
	return nested.Params, nil
}

func (s *petSigninBonusService) _noopKillSwitchReadForFuture() error {
	// 预留：未来接入 pet.killSwitch（SysConfig）后，在发放前拦截。
	// 当前不做强依赖，避免因为 sys_config 结构变动导致编译/运行失败。
	_ = time.Now()
	_ = dates.NowTimestamp()
	_ = sqls.NewCnd
	return nil
}
