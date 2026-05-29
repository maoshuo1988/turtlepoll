package api

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/models"
	"bbs-go/internal/pkg/biztime"
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/services"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/common/jsons"
	"github.com/mlogclub/simple/sqls"
	"github.com/mlogclub/simple/web"
)

// PetController 用户侧宠物接口：/api/pet/*
type PetController struct {
	Ctx iris.Context
}

// GetDefs GET /api/pet/defs
func (c *PetController) GetDefs() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonErrorMsg("unauthorized")
	}
	page, size := parsePetDefsPageSize(c.Ctx)
	cnd := sqls.NewCnd().Desc("id").Page(page, size)
	cnd.Eq("status", constants.StatusOk)
	if ok, has := parsePetDefsBoolQuery(c.Ctx.URLParam("enabled")); has {
		cnd.Eq("obtainable_by_egg", ok)
	}
	if rarity := strings.TrimSpace(c.Ctx.URLParam("rarity")); rarity != "" {
		if r := parsePetDefsRarity(rarity); r > 0 {
			cnd.Eq("rarity", r)
		}
	}
	list, paging := services.PetDefinitionService.FindPageByCnd(cnd)
	items := make([]map[string]any, 0, len(list))
	for i := range list {
		items = append(items, buildUserPetInfo(&list[i]))
	}
	return web.JsonData(map[string]any{
		"items": items,
		"total": paging.Total,
	})
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
		"petId":               state.EquippedPetId,
		"petKey":              "",
		"petName":             "",
		"rarity":              0,
		"rarityKey":           "",
		"abilities":           map[string]any{},
		"abilityDescriptions": []map[string]any{},
		"image":               "",
		"icon":                "",
		"level":               1,
		"equippedAt":          state.UpdateTime,
		"equipDayName":        state.EquipDayName,
		"pet":                 petInfo,
	}
	if pet != nil {
		resp["petKey"] = pet.PetKey
		resp["petName"] = petInfo["name"]
		resp["rarity"] = pet.Rarity
		resp["rarityKey"] = petInfo["rarityKey"]
		resp["abilities"] = petInfo["abilities"]
		resp["abilityDescriptions"] = petInfo["abilityDescriptions"]
		resp["image"] = petInfo["image"]
		resp["icon"] = petInfo["icon"]
	}
	return web.JsonData(resp)
}

func parsePetDefsBoolQuery(v string) (bool, bool) {
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

func parsePetDefsPageSize(ctx iris.Context) (int, int) {
	page := atoiDefault(ctx.URLParamDefault("page", "1"), 1)
	size := atoiDefault(ctx.URLParamDefault("size", "20"), 20)
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

func atoiDefault(v string, def int) int {
	n := 0
	for _, ch := range strings.TrimSpace(v) {
		if ch < '0' || ch > '9' {
			return def
		}
		n = n*10 + int(ch-'0')
	}
	if n == 0 {
		return def
	}
	return n
}

func parsePetDefsRarity(r string) int {
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
		"ok":                  true,
		"petId":               state.EquippedPetId,
		"petKey":              petInfo["petKey"],
		"petName":             petInfo["name"],
		"rarity":              petInfo["rarity"],
		"rarityKey":           petInfo["rarityKey"],
		"abilities":           petInfo["abilities"],
		"abilityDescriptions": petInfo["abilityDescriptions"],
		"image":               petInfo["image"],
		"icon":                petInfo["icon"],
		"pet":                 petInfo,
		"equipDayName":        state.EquipDayName,
		"nextEffectiveAt":     next,
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
			"petId":               it.PetId,
			"petKey":              "",
			"petName":             "",
			"rarity":              0,
			"rarityKey":           "",
			"abilities":           map[string]any{},
			"abilityDescriptions": []map[string]any{},
			"image":               "",
			"icon":                "",
			"level":               it.Level,
			"xp":                  it.XP,
			"isEquipped":          state != nil && state.EquippedPetId == it.PetId,
			"obtainedAt":          it.ObtainedAt,
			"pet":                 petInfo,
		}
		if pet != nil {
			m["petKey"] = pet.PetKey
			m["petName"] = petInfo["name"]
			m["rarity"] = pet.Rarity
			m["rarityKey"] = petInfo["rarityKey"]
			m["abilities"] = petInfo["abilities"]
			m["abilityDescriptions"] = petInfo["abilityDescriptions"]
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

// GetGachaConfig GET /api/pet/gacha/config
func (c *PetController) GetGachaConfig() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonErrorMsg("unauthorized")
	}
	cfg, err := services.PetGachaService.GetConfig()
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonData(cfg)
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
		"id":                  int64(0),
		"petId":               int64(0),
		"petKey":              "",
		"petCode":             "",
		"name":                "",
		"rarity":              0,
		"rarityKey":           "",
		"image":               "",
		"icon":                "",
		"abilities":           map[string]any{},
		"abilityDescriptions": []map[string]any{},
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
	ret["abilityDescriptions"] = buildAbilityDescriptions(abilities)
	return ret
}

func buildAbilityDescriptions(abilities map[string]any) []map[string]any {
	ret := make([]map[string]any, 0, len(abilities))
	keys := make([]string, 0, len(abilities))
	for featureKey := range abilities {
		featureKey = strings.TrimSpace(featureKey)
		if featureKey == "" {
			continue
		}
		keys = append(keys, featureKey)
	}
	sort.Strings(keys)
	for _, featureKey := range keys {
		params := anyMap(abilities[featureKey])
		name := featureKey
		if fc := services.FeatureCatalogService.GetByFeatureKey(featureKey); fc != nil {
			if v := pickI18nText(fc.NameJSON); v != "" {
				name = v
			} else if v := strings.TrimSpace(fc.Name); v != "" {
				name = v
			}
		}
		ret = append(ret, map[string]any{
			"featureKey":  featureKey,
			"name":        name,
			"description": describeAbility(featureKey, params),
			"enabled":     boolParam(params, "enabled", true),
		})
	}
	return ret
}

func pickI18nText(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	var m map[string]string
	if err := jsons.Parse(raw, &m); err != nil {
		return ""
	}
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
	return ""
}

func anyMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	ret := map[string]any{}
	if bs, err := json.Marshal(v); err == nil {
		_ = json.Unmarshal(bs, &ret)
	}
	if ret == nil {
		return map[string]any{}
	}
	return ret
}

func boolParam(params map[string]any, key string, fallback bool) bool {
	v, ok := params[key]
	if !ok {
		return fallback
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "true", "1", "yes", "y":
			return true
		case "false", "0", "no", "n":
			return false
		}
	}
	return fallback
}

func int64Param(params map[string]any, key string, fallback int64) int64 {
	v, ok := params[key]
	if !ok {
		return fallback
	}
	switch t := v.(type) {
	case int:
		return int64(t)
	case int64:
		return t
	case int32:
		return int64(t)
	case float64:
		return int64(t)
	case float32:
		return int64(t)
	case string:
		var out int64
		if _, err := fmt.Sscanf(strings.TrimSpace(t), "%d", &out); err == nil {
			return out
		}
	}
	return fallback
}

func floatParam(params map[string]any, key string, fallback float64) float64 {
	v, ok := params[key]
	if !ok {
		return fallback
	}
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case int32:
		return float64(t)
	case string:
		var out float64
		if _, err := fmt.Sscanf(strings.TrimSpace(t), "%f", &out); err == nil {
			return out
		}
	}
	return fallback
}

func formatInt64(v int64) string {
	return fmt.Sprintf("%d", v)
}

func formatFloat(v float64) string {
	if math.Abs(v-math.Round(v)) < 0.0000001 {
		return fmt.Sprintf("%.0f", v)
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", v), "0"), ".")
}

func formatPercent(v float64) string {
	return formatFloat(v*100) + "%"
}

func describeAbility(featureKey string, params map[string]any) string {
	switch featureKey {
	case "signin_bonus":
		amount := int64Param(params, "bonusCoins", int64Param(params, "base_amount", 0))
		capPerDay := int64Param(params, "capPerDay", int64Param(params, "daily_cap", 0))
		if capPerDay > 0 {
			return "每日登录额外获得" + formatInt64(amount) + "龟币，每日上限" + formatInt64(capPerDay) + "龟币。"
		}
		return "每日登录额外获得" + formatInt64(amount) + "龟币。"
	case "spark_multiplier":
		base := floatParam(params, "base", 0)
		perLevel := floatParam(params, "per_level", 0)
		capPerDay := int64Param(params, "cap", 0)
		desc := "每日火花奖励按" + formatFloat(base) + "倍基础倍率加成"
		if perLevel > 0 {
			desc += "，每级额外增加" + formatPercent(perLevel)
		}
		if capPerDay > 0 {
			desc += "，每日额外奖励上限" + formatInt64(capPerDay) + "龟币"
		}
		return desc + "。"
	case "debt":
		floor := int64Param(params, "debtFloor", 0)
		desc := "允许龟币余额最低欠至" + formatInt64(floor) + "。"
		if boolParam(params, "forbidEquipWhenDebt", false) {
			desc += "欠款未还清时禁止切换龟种。"
		}
		return desc
	case "debt_subsidy":
		rate := floatParam(params, "subsidyRate", 0)
		capPerDay := int64Param(params, "capPerDay", 0)
		desc := "每日登录结算时按欠款金额的" + formatPercent(rate) + "补贴。"
		if capPerDay > 0 {
			desc += "每日上限" + formatInt64(capPerDay) + "龟币。"
		}
		return desc
	case "deposit_interest":
		rate := floatParam(params, "interestRate", 0)
		capPerDay := int64Param(params, "capPerDay", 0)
		desc := "每日登录结算时按正余额的" + formatPercent(rate) + "发放利息。"
		if capPerDay > 0 {
			desc += "每日上限" + formatInt64(capPerDay) + "龟币。"
		}
		return desc
	case "first_bet_bonus":
		amount := int64Param(params, "amount", 0)
		return "每日首次下注额外获得" + formatInt64(amount) + "龟币。"
	default:
		return "已启用该能力。"
	}
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
