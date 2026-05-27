package api

import (
	"bbs-go/internal/models/models"
	"bbs-go/internal/pkg/biztime"
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/services"
	"strings"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/common/jsons"
	"github.com/mlogclub/simple/web"
)

// PetController 用户侧宠物接口：/api/pet/*
type PetController struct {
	Ctx iris.Context
}

// GetEquip GET /api/pet/equip
func (c *PetController) GetEquip() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonErrorMsg("unauthorized")
	}
	state, err := services.UserPetService.GetOrCreateState(user.Id)
	if err != nil {
		return web.JsonError(err)
	}
	var pet *models.PetDefinition
	if state.EquippedPetId > 0 {
		pet = services.PetDefinitionService.Get(state.EquippedPetId)
	}
	petInfo := buildUserPetInfo(pet)
	resp := map[string]any{
		"petId":        state.EquippedPetId,
		"petKey":       "",
		"petName":      "",
		"rarity":       0,
		"rarityKey":    "",
		"abilities":    map[string]any{},
		"image":        "",
		"icon":         "",
		"level":        1,
		"equippedAt":   state.UpdateTime,
		"equipDayName": state.EquipDayName,
		"pet":          petInfo,
	}
	if pet != nil {
		resp["petKey"] = pet.PetKey
		resp["petName"] = petInfo["name"]
		resp["rarity"] = pet.Rarity
		resp["rarityKey"] = petInfo["rarityKey"]
		resp["abilities"] = petInfo["abilities"]
		resp["image"] = petInfo["image"]
		resp["icon"] = petInfo["icon"]
	}
	return web.JsonData(resp)
}

// PostEquip POST /api/pet/equip
func (c *PetController) PostEquip() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonErrorMsg("unauthorized")
	}
	var req struct {
		PetId int64 `json:"petId"`
	}
	_ = c.Ctx.ReadJSON(&req)
	if req.PetId <= 0 {
		return web.JsonErrorMsg("PARAM_INVALID")
	}
	// 必须拥有
	if !services.UserPetService.HasPet(user.Id, req.PetId) {
		return web.JsonErrorMsg("PARAM_INVALID")
	}
	state, err := services.UserPetService.EquipPet(user.Id, req.PetId)
	if err != nil {
		// service 里用错误字符串返回错误码（保持最小改动）
		if err.Error() == "EQUIP_DAILY_LIMIT" {
			return web.JsonErrorMsg("EQUIP_DAILY_LIMIT")
		}
		if err.Error() == "DEBT_UNPAID" {
			return web.JsonErrorMsg("DEBT_UNPAID")
		}
		return web.JsonError(err)
	}
	// nextEffectiveAt：北京时间次日 0 点
	next := biztime.NextMidnightCSTUnix(biztime.NowInCST())
	pet := services.PetDefinitionService.Get(state.EquippedPetId)
	petInfo := buildUserPetInfo(pet)
	return web.JsonData(map[string]any{
		"ok":              true,
		"petId":           state.EquippedPetId,
		"petKey":          petInfo["petKey"],
		"petName":         petInfo["name"],
		"rarity":          petInfo["rarity"],
		"rarityKey":       petInfo["rarityKey"],
		"abilities":       petInfo["abilities"],
		"image":           petInfo["image"],
		"icon":            petInfo["icon"],
		"pet":             petInfo,
		"equipDayName":    state.EquipDayName,
		"nextEffectiveAt": next,
	})
}

// GetOwned GET /api/pet/owned
func (c *PetController) GetOwned() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonErrorMsg("unauthorized")
	}
	state, _ := services.UserPetService.GetOrCreateState(user.Id)
	owned, err := services.UserPetService.ListOwned(user.Id)
	if err != nil {
		return web.JsonError(err)
	}
	respList := make([]map[string]any, 0, len(owned))
	for _, it := range owned {
		pet := services.PetDefinitionService.Get(it.PetId)
		petInfo := buildUserPetInfo(pet)
		m := map[string]any{
			"petId":      it.PetId,
			"petKey":     "",
			"petName":    "",
			"rarity":     0,
			"rarityKey":  "",
			"abilities":  map[string]any{},
			"image":      "",
			"icon":       "",
			"level":      it.Level,
			"xp":         it.XP,
			"isEquipped": state != nil && state.EquippedPetId == it.PetId,
			"obtainedAt": it.ObtainedAt,
			"pet":        petInfo,
		}
		if pet != nil {
			m["petKey"] = pet.PetKey
			m["petName"] = petInfo["name"]
			m["rarity"] = pet.Rarity
			m["rarityKey"] = petInfo["rarityKey"]
			m["abilities"] = petInfo["abilities"]
			m["image"] = petInfo["image"]
			m["icon"] = petInfo["icon"]
		}
		respList = append(respList, m)
	}
	return web.JsonData(map[string]any{
		"equippedPetId": func() int64 {
			if state == nil {
				return 0
			}
			return state.EquippedPetId
		}(),
		"list": respList,
	})
}

// GetStamina GET /api/pet/stamina （占位，P1 再落库）
func (c *PetController) GetStamina() *web.JsonResult {
	return web.JsonData(map[string]any{
		"current":      100,
		"cap":          100,
		"regenPerHour": 5,
	})
}

// PostEggHatch POST /api/pet/egg/hatch
func (c *PetController) PostEggHatch() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonErrorMsg("unauthorized")
	}
	ret, err := services.PetEggService.HatchEgg(user.Id)
	if err != nil {
		// 业务错误码：保持简单字符串。
		s := err.Error()
		switch s {
		case "GACHA_DISABLED":
			return web.JsonErrorMsg("GACHA_DISABLED")
		case "insufficient balance":
			return web.JsonErrorMsg("INSUFFICIENT_BALANCE")
		default:
			return web.JsonError(err)
		}
	}
	return web.JsonData(map[string]any{
		"cost":          ret.Cost,
		"refund":        ret.Refund,
		"isDuplicate":   ret.IsDuplicate,
		"balanceBefore": ret.BalanceBefore,
		"balanceAfter":  ret.BalanceAfter,
		"pet": map[string]any{
			"petId":  ret.PetId,
			"petKey": ret.PetKey,
			"rarity": ret.Rarity,
		},
	})
}

// GetStatus GET /api/pet/status （占位）
func (c *PetController) GetStatus() *web.JsonResult {
	return web.JsonData(map[string]any{
		"daily": map[string]any{},
	})
}

func buildUserPetInfo(pet *models.PetDefinition) map[string]any {
	ret := map[string]any{
		"id":        int64(0),
		"petId":     int64(0),
		"petKey":    "",
		"petCode":   "",
		"name":      "",
		"rarity":    0,
		"rarityKey": "",
		"image":     "",
		"icon":      "",
		"abilities": map[string]any{},
	}
	if pet == nil {
		return ret
	}
	abilities := services.PetDefinitionService.GetAbilities(pet)
	display := parsePetDisplay(pet.DisplayJSON)
	image := pickPetImage(display, pet.Icon)
	petCode := strings.TrimSpace(pet.PetId)
	if petCode == "" {
		petCode = strings.TrimSpace(pet.PetKey)
	}
	ret["id"] = pet.Id
	ret["petId"] = pet.Id
	ret["petKey"] = pet.PetKey
	ret["petCode"] = petCode
	ret["name"] = pickPetName(pet)
	ret["rarity"] = pet.Rarity
	ret["rarityKey"] = rarityValueToKey(pet.Rarity)
	ret["image"] = image
	ret["icon"] = image
	ret["abilities"] = abilities
	return ret
}

func pickPetName(pet *models.PetDefinition) string {
	if pet == nil {
		return ""
	}
	if strings.TrimSpace(pet.NameJSON) != "" {
		var m map[string]string
		if err := jsons.Parse(pet.NameJSON, &m); err == nil {
			if v := strings.TrimSpace(m["zh-CN"]); v != "" {
				return v
			}
			if v := strings.TrimSpace(m["en-US"]); v != "" {
				return v
			}
			for _, v := range m {
				if v = strings.TrimSpace(v); v != "" {
					return v
				}
			}
		}
	}
	return strings.TrimSpace(pet.Name)
}

func parsePetDisplay(raw string) map[string]any {
	ret := map[string]any{}
	if strings.TrimSpace(raw) == "" {
		return ret
	}
	_ = jsons.Parse(raw, &ret)
	if ret == nil {
		return map[string]any{}
	}
	return ret
}

func pickPetImage(display map[string]any, fallback string) string {
	for _, key := range []string{"icon", "thumbnail", "cover"} {
		if v, ok := display[key]; ok {
			if s := strings.TrimSpace(toString(v)); s != "" {
				return s
			}
		}
	}
	return strings.TrimSpace(fallback)
}

func toString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		return ""
	}
}

func rarityValueToKey(v int) string {
	switch v {
	case 1:
		return "C"
	case 2:
		return "B"
	case 3:
		return "A"
	case 4:
		return "S"
	case 5:
		return "SS"
	case 6:
		return "SSS"
	default:
		return ""
	}
}
