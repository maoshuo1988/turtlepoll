package services

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/models"
	"bbs-go/internal/repositories"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

// PetEggService 用户开蛋服务：概率依据开蛋池配置（PetGachaService.GetConfig）。
var PetEggService = newPetEggService()

func newPetEggService() *petEggService { return &petEggService{} }

type petEggService struct{}

type EggHatchResult struct {
	PetId         int64  `json:"petId"`
	PetKey        string `json:"petKey"`
	Rarity        string `json:"rarity"`
	Cost          int64  `json:"cost"`
	Refund        int64  `json:"refund"`
	IsDuplicate   bool   `json:"isDuplicate"`
	BalanceBefore int64  `json:"balanceBefore"`
	BalanceAfter  int64  `json:"balanceAfter"`
}

// HatchEgg 开蛋：
// 1) 读取开蛋池配置（若无配置则写默认）。
// 2) 校验 enabled=true。
// 3) 按 rarity_weights 先抽稀有度。
// 4) 在可开蛋宠物定义中筛选该稀有度候选，均匀抽取一个。
// 5) 事务内扣费 + 发放宠物（若已拥有则不重复发放，仍返回抽中结果）。
func (s *petEggService) HatchEgg(userId int64) (*EggHatchResult, error) {
	if userId <= 0 {
		return nil, errors.New("userId is required")
	}

	cfg, err := PetGachaService.GetConfig()
	if err != nil {
		slog.Error("pet egg HatchEgg: get config failed", "err", err)
		return nil, err
	}
	if !cfg.Enabled {
		return nil, errors.New("GACHA_DISABLED")
	}
	if cfg.BaseCost < 0 {
		return nil, errors.New("invalid gacha base_cost")
	}

	weights, err := parseRarityWeightsJSON(cfg.RarityWeightsJSON)
	if err != nil {
		slog.Error("pet egg HatchEgg: parse rarity weights failed", "err", err)
		return nil, err
	}

	rarKey, err := pickRarityByWeights(weights)
	if err != nil {
		slog.Error("pet egg HatchEgg: pick rarity failed", "err", err)
		return nil, err
	}
	rarVal, err := rarityKeyToValue(rarKey)
	if err != nil {
		return nil, err
	}

	candidates, err := s.listEggCandidatesByRarity(sqls.DB(), rarVal)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no obtainable_by_egg pets for rarity=%s", rarKey)
	}
	picked := candidates[rand.Intn(len(candidates))]

	now := dates.NowTimestamp()
	result := &EggHatchResult{PetId: picked.Id, PetKey: picked.PetKey, Rarity: rarKey, Cost: cfg.BaseCost}

	// 原子事务：扣费 + 发放（去重）
	err = sqls.DB().Transaction(func(tx *gorm.DB) error {
		// 记录扣费前余额（用于接口返回；并发情况下这是本次事务看到的余额快照）
		uc, err := repositories.UserCoinRepository.GetOrCreate(tx, userId)
		if err != nil {
			return err
		}
		result.BalanceBefore = uc.Balance

		// 扣费：目前项目里没有“开蛋扣费”专用 bizType，这里新增一个类型字符串。
		if cfg.BaseCost > 0 {
			// bizId 暂用 petDefinitionId（审计足够），后续如有 egg_hatch_log 表可替换。
			if err := UserCoinService.SpendToPool(tx, userId, "PET_EGG_HATCH", picked.Id, cfg.BaseCost, fmt.Sprintf("pet egg hatch rarity=%s petId=%d", rarKey, picked.Id)); err != nil {
				return err
			}
		}

		// 发放：如果已拥有则不再 create，但要按文档规则返还（abilities.egg_duplicate_refund）
		if repositories.UserPetRepository.Get(tx, userId, picked.Id) != nil {
			result.IsDuplicate = true
			// 返还也需要记账：这里使用 PayFromPoolToUser，会对“用户”和“资金池”各写一条 coin log。
			// refund = floor(cost * 0.3)
			refund := int64(float64(result.Cost) * 0.3)
			if refund > result.Cost {
				refund = result.Cost
			}
			if refund > 0 {
				// 返还从资金池出池给用户，bizType 单独区分，方便审计。
				if err := UserCoinService.PayFromPoolToUser(tx, userId, "PET_EGG_DUPLICATE_REFUND", picked.Id, refund, fmt.Sprintf("pet egg duplicate refund petId=%d", picked.Id)); err != nil {
					return err
				}
				result.Refund = refund
			}
			// 更新事务内余额快照
			uc2, err := repositories.UserCoinRepository.GetOrCreate(tx, userId)
			if err != nil {
				return err
			}
			result.BalanceAfter = uc2.Balance
			return nil
		}
		if err := repositories.UserPetRepository.Create(tx, &models.UserPet{
			UserId:     userId,
			PetId:      picked.Id,
			Level:      1,
			XP:         0,
			ObtainedAt: now,
			CreateTime: now,
			UpdateTime: now,
		}); err != nil {
			return err
		}
		uc3, err := repositories.UserCoinRepository.GetOrCreate(tx, userId)
		if err != nil {
			return err
		}
		result.BalanceAfter = uc3.Balance
		return nil
	})
	if err != nil {
		slog.Error("pet egg HatchEgg: tx failed", "err", err)
		return nil, err
	}

	return result, nil
}

func (s *petEggService) listEggCandidatesByRarity(db *gorm.DB, rarity int) ([]models.PetDefinition, error) {
	cnd := sqls.NewCnd().
		Eq("status", constants.StatusOk).
		Eq("obtainable_by_egg", true).
		Eq("rarity", rarity).
		Desc("id")
	list := repositories.PetDefinitionRepository.Find(db, cnd)
	return list, nil
}

func parseRarityWeightsJSON(s string) (map[string]float64, error) {
	if s == "" {
		return nil, errors.New("rarity weights json is empty")
	}
	var m map[string]float64
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}
	if len(m) == 0 {
		return nil, errors.New("rarity weights is empty")
	}
	return m, nil
}

// rarityKeyToValue 将配置用的 key 转为 PetDefinition.Rarity(int) 值。
// 约定：当前仓库里 PetDefinition.Rarity 就直接存 C/B/A/S/SS/SSS 的序号（1..6），与后台入参一致。
// 若后续改成常量枚举，这里集中改映射即可。
func rarityKeyToValue(k string) (int, error) {
	switch k {
	case "C":
		return 1, nil
	case "B":
		return 2, nil
	case "A":
		return 3, nil
	case "S":
		return 4, nil
	case "SS":
		return 5, nil
	case "SSS":
		return 6, nil
	default:
		return 0, fmt.Errorf("invalid rarity key: %s", k)
	}
}

// pickRarityByWeights 按权重抽稀有度。
// 约束：weights 的 key 必须在 {C,B,A,S,SS,SSS}，value 范围 [0,1]，sum==1（由配置保存时保证）。
func pickRarityByWeights(weights map[string]float64) (string, error) {
	if len(weights) == 0 {
		return "", errors.New("rarity weights is empty")
	}
	// 为保证可预测遍历顺序（避免 map 迭代随机导致线上抽取漂移），固定顺序。
	order := []string{"C", "B", "A", "S", "SS", "SSS"}
	v := rand.Float64()
	acc := 0.0
	for _, k := range order {
		w, ok := weights[k]
		if !ok {
			continue
		}
		if w < 0 {
			return "", fmt.Errorf("invalid weight for %s", k)
		}
		acc += w
		if v <= acc {
			return k, nil
		}
	}
	// 浮点误差兜底：返回最后一个非零项
	for i := len(order) - 1; i >= 0; i-- {
		k := order[i]
		if weights[k] > 0 {
			return k, nil
		}
	}
	return "", errors.New("no rarity picked")
}
