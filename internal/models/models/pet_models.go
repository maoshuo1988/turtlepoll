package models

// TurtlePoll: PetDefinition & FeatureCatalog

// PetDefinition 龟种定义（运营侧维护资源）
// abilities 存储为 JSON 字符串（featureKey -> params），由上层做 schema 强校验。
type PetDefinition struct {
	Model
	// PetKey 业务唯一键（建议与运营配置中的 key 对齐），便于稳定引用
	PetKey string `gorm:"not null;size:64;uniqueIndex" json:"petKey" form:"petKey"`

	Name        string `gorm:"not null;size:128" json:"name" form:"name"`
	Description string `gorm:"size:1024" json:"description" form:"description"`
	Icon        string `gorm:"size:1024" json:"icon" form:"icon"`

	Rarity int `gorm:"not null;default:0" json:"rarity" form:"rarity"`
	// ObtainableByEgg 是否允许进入开蛋池
	ObtainableByEgg bool `gorm:"not null;default:false" json:"obtainableByEgg" form:"obtainableByEgg"`

	// AbilitiesJSON abilities: featureKey -> params（存 JSON 文本）
	AbilitiesJSON string `gorm:"type:text" json:"abilities" form:"abilities"`

	Status     int   `gorm:"not null;default:0;index" json:"status" form:"status"`
	CreateTime int64 `gorm:"not null;default:0" json:"createTime" form:"createTime"`
	UpdateTime int64 `gorm:"not null;default:0" json:"updateTime" form:"updateTime"`
}

// FeatureCatalogItem 特性模板（白名单 + schema）
type FeatureCatalogItem struct {
	Model
	FeatureKey string `gorm:"not null;size:128;uniqueIndex" json:"featureKey" form:"featureKey"`
	Name       string `gorm:"not null;size:128" json:"name" form:"name"`
	// Scope: PET_DEF / GLOBAL
	Scope string `gorm:"not null;size:32" json:"scope" form:"scope"`
	// EffectiveEvent: SIGNIN / EGG_OPEN / BATTLE_SETTLE ...
	EffectiveEvent string `gorm:"not null;size:64" json:"effectiveEvent" form:"effectiveEvent"`
	// ParamsSchemaJSON JSONSchema 文本（用于后端校验与后台动态表单）
	ParamsSchemaJSON string `gorm:"type:text" json:"paramsSchema" form:"paramsSchema"`
	Enabled          bool   `gorm:"not null;default:true" json:"enabled" form:"enabled"`
	MetadataJSON     string `gorm:"type:text" json:"metadata" form:"metadata"`

	CreateTime int64 `gorm:"not null;default:0" json:"createTime" form:"createTime"`
	UpdateTime int64 `gorm:"not null;default:0" json:"updateTime" form:"updateTime"`
}
