package api

import (
	"bbs-go/internal/models/models"
	"bbs-go/internal/pkg/biztime"
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/services"

	"github.com/kataras/iris/v12"
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
	resp := map[string]any{
		"petId":        state.EquippedPetId,
		"petKey":       "",
		"petName":      "",
		"rarity":       0,
		"level":        1,
		"equippedAt":   state.UpdateTime,
		"equipDayName": state.EquipDayName,
	}
	if pet != nil {
		resp["petKey"] = pet.PetKey
		resp["petName"] = pet.Name
		resp["rarity"] = pet.Rarity
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
		return web.JsonError(err)
	}
	// nextEffectiveAt：北京时间次日 0 点
	next := biztime.NextMidnightCSTUnix(biztime.NowInCST())
	return web.JsonData(map[string]any{
		"ok":              true,
		"petId":           state.EquippedPetId,
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
		m := map[string]any{
			"petId":      it.PetId,
			"petKey":     "",
			"petName":    "",
			"rarity":     0,
			"level":      it.Level,
			"xp":         it.XP,
			"isEquipped": state != nil && state.EquippedPetId == it.PetId,
			"obtainedAt": it.ObtainedAt,
		}
		if pet != nil {
			m["petKey"] = pet.PetKey
			m["petName"] = pet.Name
			m["rarity"] = pet.Rarity
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

// PostEggHatch POST /api/pet/egg/hatch （占位，P1 再做事务扣费/抽取/返还）
func (c *PetController) PostEggHatch() *web.JsonResult {
	return web.JsonErrorMsg("NOT_IMPLEMENTED")
}

// GetStatus GET /api/pet/status （占位）
func (c *PetController) GetStatus() *web.JsonResult {
	return web.JsonData(map[string]any{
		"daily": map[string]any{},
	})
}
