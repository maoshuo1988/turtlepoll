package admin

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/models"
	"bbs-go/internal/services"
	"encoding/json"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/sqls"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
)

type PetController struct {
	Ctx iris.Context
}

// GetDefs GET /api/admin/pet/defs
func (c *PetController) GetDefs() *web.JsonResult {
	cnd := params.NewQueryParams(c.Ctx).PageByReq().Desc("id")
	if status := params.FormValueIntDefault(c.Ctx, "status", -1); status >= 0 {
		cnd.Eq("status", status)
	} else {
		cnd.Eq("status", constants.StatusOk)
	}
	list, paging := services.PetDefinitionService.FindPageByCnd(&cnd.Cnd)
	return web.JsonData(&web.PageResult{Results: list, Page: paging})
}

// GetDefsBy GET /api/admin/pet/defs/{id}
func (c *PetController) GetDefsBy(id int64) *web.JsonResult {
	t := services.PetDefinitionService.Get(id)
	if t == nil {
		return web.JsonErrorMsg("not found")
	}
	return web.JsonData(t)
}

// PostDefs POST /api/admin/pet/defs
func (c *PetController) PostDefs() *web.JsonResult {
	var body models.PetDefinition
	if err := c.Ctx.ReadJSON(&body); err != nil {
		// fallback: form
		params.ReadForm(c.Ctx, &body)
	}
	// upsert (preferred):
	// 1) if id present -> update by id
	// 2) else -> try update by petKey, fallback to create
	if body.Id > 0 {
		exists := services.PetDefinitionService.Get(body.Id)
		if exists == nil {
			return web.JsonErrorMsg("not found")
		}
		body.CreateTime = exists.CreateTime
		if err := services.PetDefinitionService.Update(&body); err != nil {
			return web.JsonErrorMsg(err.Error())
		}
		return web.JsonData(&body)
	}

	// If client doesn't know id, use PetKey as natural key.
	if byKey := services.PetDefinitionService.GetByPetKey(body.PetKey); byKey != nil {
		body.Id = byKey.Id
		body.CreateTime = byKey.CreateTime
		// keep existing status if caller doesn't provide it
		if body.Status == 0 {
			body.Status = byKey.Status
		}
		if err := services.PetDefinitionService.Update(&body); err != nil {
			return web.JsonErrorMsg(err.Error())
		}
		return web.JsonData(&body)
	}

	if err := services.PetDefinitionService.Create(&body); err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(&body)
}

// DeleteDefsBy DELETE /api/admin/pet/defs/{id}
func (c *PetController) DeleteDefsBy(id int64) *web.JsonResult {
	if err := services.PetDefinitionService.Delete(id); err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonSuccess()
}

// PostKill_switch POST /api/admin/pet/kill-switch
// 当前实现先挂到 SysConfig（不增加额外表），key: pet.killSwitch
func (c *PetController) PostKill_switch() *web.JsonResult {
	body, err := c.Ctx.GetBody()
	if err != nil {
		return web.JsonError(err)
	}
	if len(body) > 0 {
		var tmp any
		if err := json.Unmarshal(body, &tmp); err != nil {
			return web.JsonErrorMsg("invalid json")
		}
	}
	if len(body) == 0 {
		return web.JsonErrorMsg("invalid json")
	}
	if err := services.SysConfigService.Set("pet.killSwitch", string(body)); err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonSuccess()
}

// --- FeatureCatalog ---

// GetFeatures GET /api/admin/pet/features
func (c *PetController) GetFeatures() *web.JsonResult {
	cnd := params.NewQueryParams(c.Ctx).PageByReq().Desc("id")
	list, paging := services.FeatureCatalogService.FindPageByCnd(&cnd.Cnd)
	return web.JsonData(&web.PageResult{Results: list, Page: paging})
}

// GetFeaturesBy GET /api/admin/pet/features/{featureKey}
func (c *PetController) GetFeaturesBy(featureKey string) *web.JsonResult {
	t := services.FeatureCatalogService.GetByFeatureKey(featureKey)
	if t == nil {
		return web.JsonErrorMsg("not found")
	}
	return web.JsonData(t)
}

// PostFeatures POST /api/admin/pet/features
func (c *PetController) PostFeatures() *web.JsonResult {
	var body models.FeatureCatalogItem
	if err := c.Ctx.ReadJSON(&body); err != nil {
		params.ReadForm(c.Ctx, &body)
	}
	// upsert by featureKey
	exists := services.FeatureCatalogService.GetByFeatureKey(body.FeatureKey)
	if exists != nil {
		body.Id = exists.Id
		body.CreateTime = exists.CreateTime
		if err := services.FeatureCatalogService.Update(&body); err != nil {
			return web.JsonErrorMsg(err.Error())
		}
		return web.JsonData(&body)
	}
	if err := services.FeatureCatalogService.Create(&body); err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(&body)
}

// DeleteFeaturesBy DELETE /api/admin/pet/features/{featureKey}
func (c *PetController) DeleteFeaturesBy(featureKey string) *web.JsonResult {
	if err := services.FeatureCatalogService.DeleteByFeatureKey(featureKey); err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonSuccess()
}

// --- abilities attach ---

// PutDefsByAbilities PUT /api/admin/pet/defs/{id}/abilities
func (c *PetController) PutDefsByAbilities(id int64) *web.JsonResult {
	var body struct {
		Abilities map[string]any `json:"abilities"`
	}
	if err := c.Ctx.ReadJSON(&body); err != nil {
		return web.JsonError(err)
	}
	pet, err := services.PetAbilityService.ReplaceAbilities(id, body.Abilities)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(pet)
}

// PatchDefsByAbilitiesBy PATCH /api/admin/pet/defs/{id}/abilities/{featureKey}
func (c *PetController) PatchDefsByAbilitiesBy(id int64, featureKey string) *web.JsonResult {
	var body struct {
		Params map[string]any `json:"params"`
	}
	if err := c.Ctx.ReadJSON(&body); err != nil {
		return web.JsonError(err)
	}
	pet, err := services.PetAbilityService.UpsertAbility(id, featureKey, body.Params)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(pet)
}

// DeleteDefsByAbilitiesBy DELETE /api/admin/pet/defs/{id}/abilities/{featureKey}
func (c *PetController) DeleteDefsByAbilitiesBy(id int64, featureKey string) *web.JsonResult {
	pet, err := services.PetAbilityService.RemoveAbility(id, featureKey)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(pet)
}

// for linter: ensure sqls imported (used by params.NewQueryParams internally in some versions)
var _ = sqls.NewCnd
