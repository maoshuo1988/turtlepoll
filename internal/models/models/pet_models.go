package models

import (
	"encoding/json"
)

// NOTE: 本文件包含 turtlepoll pet 域模型。

// TurtlePoll: PetDefinition & FeatureCatalog

// PetDefinition 龟种定义（运营侧维护资源）
// abilities 存储为 JSON 字符串（featureKey -> params），由上层做 schema 强校验。
type PetDefinition struct {
	Model
	// PetId 业务自然键（与 docs/api/pet.md 的 pet_id 对齐），用于幂等 upsert。
	// 约束：创建后不可修改。
	PetId string `gorm:"not null;size:64;uniqueIndex" json:"pet_id" form:"pet_id"`

	// PetKey 业务唯一键（建议与运营配置中的 key 对齐），便于稳定引用
	// NOTE: 历史字段。为兼容旧客户端暂时保留，后续可迁移/废弃。
	PetKey string `gorm:"not null;size:64;index" json:"petKey" form:"petKey"`

	// NameJSON 多语言名称：{"zh-CN":"...","en-US":"..."}
	// 对齐 docs/api/pet.md：name 为 object。
	NameJSON string `gorm:"column:name_json;type:text" json:"name" form:"name"`
	// DescriptionJSON 多语言描述：{"zh-CN":"...","en-US":"..."}
	// 对齐 docs/api/pet.md：description 为 object。
	DescriptionJSON string `gorm:"column:description_json;type:text" json:"description" form:"description"`

	// DisplayJSON 展示资源与渲染信息：{"icon":"...","cover":"...","thumbnail":"..."}
	DisplayJSON string `gorm:"column:display_json;type:text" json:"display" form:"display"`
	// PricingJSON 定价信息：{"egg_price":500,"egg_discount":{...}}
	PricingJSON string `gorm:"column:pricing_json;type:text" json:"pricing" form:"pricing"`

	// Name/Description 兼容字段：用于用户侧/旧逻辑展示（优先取 zh-CN 或任意一项）
	// NOTE: 未来可移除，当前保留避免影响 /api/pet/* 的 petName 输出。
	Name        string `gorm:"not null;size:128;default:''" json:"name_plain" form:"name_plain"`
	Description string `gorm:"size:1024" json:"description_plain" form:"description_plain"`
	Icon        string `gorm:"size:1024" json:"icon" form:"icon"`

	Rarity int `gorm:"not null;default:0" json:"rarity" form:"rarity"`
	// ObtainableByEgg 是否允许进入开蛋池（对齐文档：obtainable_by_egg）
	ObtainableByEgg bool `gorm:"not null;default:false" json:"obtainable_by_egg" form:"obtainable_by_egg"`

	// AbilitiesJSON abilities: featureKey -> params（存 JSON 文本）
	AbilitiesJSON string `gorm:"type:text" json:"abilities" form:"abilities"`

	Status     int   `gorm:"not null;default:0;index" json:"status" form:"status"`
	CreateTime int64 `gorm:"not null;default:0" json:"createTime" form:"createTime"`
	UpdateTime int64 `gorm:"not null;default:0" json:"updateTime" form:"updateTime"`
}

// MarshalJSON 确保对外输出时：
// - name/description/display/pricing/abilities 为 JSON object（而不是数据库里的 JSON 字符串）
// - 同时保留 name_plain/description_plain 兼容用户侧。
func (t PetDefinition) MarshalJSON() ([]byte, error) {
	type Alias PetDefinition

	var (
		nameObj      any
		descObj      any
		displayObj   any
		pricingObj   any
		abilitiesObj any
	)
	if t.NameJSON != "" {
		_ = json.Unmarshal([]byte(t.NameJSON), &nameObj)
	}
	if nameObj == nil {
		nameObj = map[string]string{}
	}
	if t.DescriptionJSON != "" {
		_ = json.Unmarshal([]byte(t.DescriptionJSON), &descObj)
	}
	if descObj == nil {
		descObj = map[string]string{}
	}
	if t.DisplayJSON != "" {
		_ = json.Unmarshal([]byte(t.DisplayJSON), &displayObj)
	}
	if displayObj == nil {
		displayObj = map[string]any{}
	}
	if t.PricingJSON != "" {
		_ = json.Unmarshal([]byte(t.PricingJSON), &pricingObj)
	}
	if pricingObj == nil {
		pricingObj = map[string]any{}
	}
	if t.AbilitiesJSON != "" {
		_ = json.Unmarshal([]byte(t.AbilitiesJSON), &abilitiesObj)
	}
	if abilitiesObj == nil {
		abilitiesObj = map[string]any{}
	}

	// 避免把 *_JSON 字符串也原样输出成 name:"{...}" 这种错误形态。
	return json.Marshal(struct {
		Alias
		Name            any    `json:"name"`
		Description     any    `json:"description"`
		Display         any    `json:"display,omitempty"`
		Pricing         any    `json:"pricing,omitempty"`
		Abilities       any    `json:"abilities,omitempty"`
		NameJSON        string `json:"-"`
		DescriptionJSON string `json:"-"`
		DisplayJSON     string `json:"-"`
		PricingJSON     string `json:"-"`
		AbilitiesJSON   string `json:"-"`
	}{
		Alias:           (Alias)(t),
		Name:            nameObj,
		Description:     descObj,
		Display:         displayObj,
		Pricing:         pricingObj,
		Abilities:       abilitiesObj,
		NameJSON:        t.NameJSON,
		DescriptionJSON: t.DescriptionJSON,
		DisplayJSON:     t.DisplayJSON,
		PricingJSON:     t.PricingJSON,
		AbilitiesJSON:   t.AbilitiesJSON,
	})
}

// FeatureCatalogItem 特性模板（白名单 + schema）
type FeatureCatalogItem struct {
	Model
	FeatureKey string `gorm:"not null;size:128;uniqueIndex" json:"feature_key" form:"feature_key"`
	// NameJSON 多语言名称：{"zh-CN":"...","en-US":"..."}
	NameJSON string `gorm:"column:name_json;type:text" json:"name" form:"name"`
	// Scope: PET_DEF / GLOBAL
	Scope string `gorm:"not null;size:32" json:"scope" form:"scope"`
	// EffectiveEvent: SIGNIN / EGG_OPEN / BATTLE_SETTLE ...
	EffectiveEvent string `gorm:"not null;size:64" json:"effective_event" form:"effective_event"`
	// ParamsSchemaJSON JSONSchema 文本（用于后端校验与后台动态表单）
	ParamsSchemaJSON string `gorm:"type:text" json:"params_schema" form:"params_schema"`
	Enabled          bool   `gorm:"not null;default:true" json:"enabled" form:"enabled"`
	MetadataJSON     string `gorm:"type:text" json:"metadata" form:"metadata"`

	CreateTime int64 `gorm:"not null;default:0" json:"createTime" form:"createTime"`
	UpdateTime int64 `gorm:"not null;default:0" json:"updateTime" form:"updateTime"`
}

// GachaPoolConfig 全局开蛋池配置（运营侧维护，通常只有一条记录）。
// 对齐 docs/api/pet_gacha.md。
type GachaPoolConfig struct {
	Model
	Enabled           bool   `gorm:"not null;default:true" json:"enabled" form:"enabled"`
	BaseCost          int64  `gorm:"not null;default:0" json:"base_cost" form:"base_cost"`
	RarityWeightsJSON string `gorm:"column:rarity_weights_json;type:text" json:"rarity_weights" form:"rarity_weights"`
	CreateTime        int64  `gorm:"not null;default:0" json:"createTime" form:"createTime"`
	UpdateTime        int64  `gorm:"not null;default:0" json:"updateTime" form:"updateTime"`
}

// DefaultGachaRarityWeights 与截图默认一致。
var DefaultGachaRarityWeights = map[string]float64{
	"C":   0.4,
	"B":   0.3,
	"A":   0.15,
	"S":   0.1,
	"SS":  0.04,
	"SSS": 0.01,
}

const DefaultGachaBaseCost int64 = 500

func (t GachaPoolConfig) MarshalJSON() ([]byte, error) {
	type Alias GachaPoolConfig
	var weights any
	if t.RarityWeightsJSON != "" {
		_ = json.Unmarshal([]byte(t.RarityWeightsJSON), &weights)
	}
	if weights == nil {
		weights = map[string]any{}
	}
	return json.Marshal(struct {
		Alias
		RarityWeights     any    `json:"rarity_weights"`
		RarityWeightsJSON string `json:"-"`
	}{
		Alias:             (Alias)(t),
		RarityWeights:     weights,
		RarityWeightsJSON: t.RarityWeightsJSON,
	})
}

// MarshalJSON：对外输出 name 为 object（i18n），而不是 name_json 字符串。
func (t FeatureCatalogItem) MarshalJSON() ([]byte, error) {
	type Alias FeatureCatalogItem
	var nameObj any
	if t.NameJSON != "" {
		_ = json.Unmarshal([]byte(t.NameJSON), &nameObj)
	}
	if nameObj == nil {
		nameObj = map[string]string{}
	}
	return json.Marshal(struct {
		Alias
		Name     any    `json:"name"`
		NameJSON string `json:"-"`
	}{
		Alias:    (Alias)(t),
		Name:     nameObj,
		NameJSON: t.NameJSON,
	})
}
