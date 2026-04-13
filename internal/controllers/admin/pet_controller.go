package admin

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/models"
	"bbs-go/internal/services"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/sqls"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
)

type petDefinitionUpsertReq struct {
	Id int64 `json:"id"`

	PetId  string `json:"pet_id"`
	PetKey string `json:"petKey"`

	Name        map[string]string `json:"name"`
	Description map[string]string `json:"description"`
	Icon        string            `json:"icon"`
	Display     any               `json:"display"`
	Pricing     any               `json:"pricing"`

	Rarity string `json:"rarity"`

	Enabled         bool `json:"enabled"`
	ObtainableByEgg bool `json:"obtainable_by_egg"`
	Abilities       any  `json:"abilities"`
	Status          int  `json:"status"`
}

// parseBoolQuery 兼容 ?enabled=true/false/1/0
func parseBoolQuery(v string) (bool, bool) {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" {
		return false, false
	}
	if v == "1" || v == "true" {
		return true, true
	}
	if v == "0" || v == "false" {
		return false, true
	}
	return false, false
}

func parsePageSize(ctx iris.Context) (int, int) {
	page, _ := strconv.Atoi(strings.TrimSpace(ctx.URLParamDefault("page", "1")))
	size, _ := strconv.Atoi(strings.TrimSpace(ctx.URLParamDefault("size", "20")))
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 200 {
		size = 200
	}
	return page, size
}

func pickAnyLang(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	if v := strings.TrimSpace(m["zh-CN"]); v != "" {
		return v
	}
	if v := strings.TrimSpace(m["en-US"]); v != "" {
		return v
	}
	for _, v := range m {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func parseRarity(r string) int {
	switch strings.ToUpper(strings.TrimSpace(r)) {
	case "C":
		return 1
	case "B":
		return 2
	case "A":
		return 3
	case "S":
		return 4
	case "SS":
		return 5
	case "SSS":
		return 6
	default:
		return 0
	}
}

type PetController struct {
	Ctx iris.Context
}

// GetDefs GET /api/admin/pet/defs
func (c *PetController) GetDefs() *web.JsonResult {
	// docs/api/pet.md: ?enabled ?rarity ?page ?size
	page, size := parsePageSize(c.Ctx)
	cnd := sqls.NewCnd().Desc("id").Page(page, size)
	cnd.Eq("status", constants.StatusOk)
	if ok, has := parseBoolQuery(c.Ctx.URLParam("enabled")); has {
		if ok {
			cnd.Eq("obtainable_by_egg", true)
		} else {
			cnd.Eq("obtainable_by_egg", false)
		}
	}
	if rarity := strings.TrimSpace(c.Ctx.URLParam("rarity")); rarity != "" {
		if r := parseRarity(rarity); r > 0 {
			cnd.Eq("rarity", r)
		}
	}
	list, paging := services.PetDefinitionService.FindPageByCnd(cnd)
	return web.JsonData(map[string]any{
		"items": list,
		"total": paging.Total,
	})
}

// GetDefsBy GET /api/admin/pet/defs/{id}
func (c *PetController) GetDefsBy(id int64) *web.JsonResult {
	t := services.PetDefinitionService.Get(id)
	if t == nil {
		return web.JsonErrorMsg("not found")
	}
	return web.JsonData(t)
}

// GetDefsByPetId GET /api/admin/pet/defs/{petId:string}
// 兼容按 docs 使用 pet_id 查询。
func (c *PetController) GetDefsByPetId(petId string) *web.JsonResult {
	petId = strings.TrimSpace(petId)
	if petId == "" {
		return web.JsonErrorMsg("not found")
	}
	t := services.PetDefinitionService.GetByPetId(petId)
	if t == nil {
		return web.JsonErrorMsg("not found")
	}
	return web.JsonData(t)
}

// PostDefs POST /api/admin/pet/defs
func (c *PetController) PostDefs() *web.JsonResult {
	var req petDefinitionUpsertReq
	if err := c.Ctx.ReadJSON(&req); err != nil {
		return web.JsonErrorMsg("invalid json")
	}

	body := models.PetDefinition{}
	body.Id = req.Id
	body.PetId = strings.TrimSpace(req.PetId)
	body.PetKey = strings.TrimSpace(req.PetKey)
	body.Icon = strings.TrimSpace(req.Icon)
	body.Rarity = parseRarity(req.Rarity)
	body.ObtainableByEgg = req.ObtainableByEgg
	// status: allow caller override, otherwise keep default via service.Create
	body.Status = req.Status

	if b, err := json.Marshal(req.Name); err == nil {
		body.NameJSON = string(b)
	}
	body.Name = pickAnyLang(req.Name)

	if b, err := json.Marshal(req.Description); err == nil {
		body.DescriptionJSON = string(b)
	}
	body.Description = pickAnyLang(req.Description)

	if req.Display != nil {
		if b, err := json.Marshal(req.Display); err == nil {
			body.DisplayJSON = string(b)
		}
	}
	if req.Pricing != nil {
		if b, err := json.Marshal(req.Pricing); err == nil {
			body.PricingJSON = string(b)
		}
	}

	if req.Abilities != nil {
		if b, err := json.Marshal(req.Abilities); err == nil {
			body.AbilitiesJSON = string(b)
		}
	}
	// upsert (preferred):
	// 1) if id present -> update by id
	// 2) else -> try update by pet_id (natural key), fallback to create
	// 3) compatibility: if pet_id empty, fallback to petKey
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

	// If client doesn't know id, use PetId as natural key.
	if body.PetId != "" {
		if byId := services.PetDefinitionService.GetByPetId(body.PetId); byId != nil {
			body.Id = byId.Id
			body.CreateTime = byId.CreateTime
			// keep existing status if caller doesn't provide it
			if body.Status == 0 {
				body.Status = byId.Status
			}
			// keep existing petKey if caller doesn't provide it
			if body.PetKey == "" {
				body.PetKey = byId.PetKey
			}
			if err := services.PetDefinitionService.Update(&body); err != nil {
				return web.JsonErrorMsg(err.Error())
			}
			return web.JsonData(&body)
		}
	}

	// Compatibility: old clients use petKey.
	if body.PetId == "" {
		if byKey := services.PetDefinitionService.GetByPetKey(body.PetKey); byKey != nil {
			body.Id = byKey.Id
			body.CreateTime = byKey.CreateTime
			if body.Status == 0 {
				body.Status = byKey.Status
			}
			// if pet_id missing, set it from existing record
			if body.PetId == "" {
				body.PetId = byKey.PetId
			}
			if body.PetKey == "" {
				body.PetKey = byKey.PetKey
			}
			if err := services.PetDefinitionService.Update(&body); err != nil {
				return web.JsonErrorMsg(err.Error())
			}
			return web.JsonData(&body)
		}
	}

	if body.PetId == "" {
		return web.JsonErrorMsg("pet_id is required")
	}

	if err := services.PetDefinitionService.Create(&body); err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(&body)
}

// PostDefsByPetId POST /api/admin/pet/defs/{petId}
// 允许将路由 petId 作为幂等键（若 body.pet_id 为空则自动补齐）。
func (c *PetController) PostDefsByPetId(petId string) *web.JsonResult {
	petId = strings.TrimSpace(petId)
	if petId == "" {
		return web.JsonErrorMsg("pet_id is required")
	}
	var req petDefinitionUpsertReq
	if err := c.Ctx.ReadJSON(&req); err != nil {
		return web.JsonErrorMsg("invalid json")
	}
	if strings.TrimSpace(req.PetId) == "" {
		req.PetId = petId
	}
	return c.PostDefs()
}

// DeleteDefsBy DELETE /api/admin/pet/defs/{id}
func (c *PetController) DeleteDefsBy(id int64) *web.JsonResult {
	if err := services.PetDefinitionService.Delete(id); err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonSuccess()
}

// DeleteDefsByPetId DELETE /api/admin/pet/defs/{petId}
func (c *PetController) DeleteDefsByPetId(petId string) *web.JsonResult {
	petId = strings.TrimSpace(petId)
	if petId == "" {
		return web.JsonErrorMsg("not found")
	}
	pet := services.PetDefinitionService.GetByPetId(petId)
	if pet == nil {
		return web.JsonErrorMsg("not found")
	}
	if err := services.PetDefinitionService.Delete(pet.Id); err != nil {
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
func (c *PetController) PutDefsByAbilities(petId string) *web.JsonResult {
	var body struct {
		Abilities map[string]any `json:"abilities"`
	}
	if err := c.Ctx.ReadJSON(&body); err != nil {
		return web.JsonError(err)
	}
	id := int64(0)
	if petId != "" {
		if byNatural := services.PetDefinitionService.GetByPetId(petId); byNatural != nil {
			id = byNatural.Id
		}
	}
	if id <= 0 {
		return web.JsonErrorMsg("not found")
	}
	pet, err := services.PetAbilityService.ReplaceAbilities(id, body.Abilities)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(pet)
}

// PatchDefsByAbilitiesBy PATCH /api/admin/pet/defs/{id}/abilities/{featureKey}
func (c *PetController) PatchDefsByAbilitiesBy(petId string, featureKey string) *web.JsonResult {
	var body struct {
		Params map[string]any `json:"params"`
	}
	if err := c.Ctx.ReadJSON(&body); err != nil {
		return web.JsonError(err)
	}
	id := int64(0)
	if petId != "" {
		if byNatural := services.PetDefinitionService.GetByPetId(petId); byNatural != nil {
			id = byNatural.Id
		}
	}
	if id <= 0 {
		return web.JsonErrorMsg("not found")
	}
	pet, err := services.PetAbilityService.UpsertAbility(id, featureKey, body.Params)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(pet)
}

// DeleteDefsByAbilitiesBy DELETE /api/admin/pet/defs/{id}/abilities/{featureKey}
func (c *PetController) DeleteDefsByAbilitiesBy(petId string, featureKey string) *web.JsonResult {
	id := int64(0)
	if petId != "" {
		if byNatural := services.PetDefinitionService.GetByPetId(petId); byNatural != nil {
			id = byNatural.Id
		}
	}
	if id <= 0 {
		return web.JsonErrorMsg("not found")
	}
	pet, err := services.PetAbilityService.RemoveAbility(id, featureKey)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(pet)
}

// for linter: ensure sqls imported (used by params.NewQueryParams internally in some versions)
var _ = sqls.NewCnd
