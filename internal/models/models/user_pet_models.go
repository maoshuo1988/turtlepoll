package models

// TurtlePoll: user-side pet models

// UserPetState 用户宠物状态（当前装备 + 当日切换限制等）。
// 说明：
// - DayName 口径：北京时间（UTC+8），YYYYMMDD。
// - 先用 PetId（引用 PetDefinition.Id）；后续如要稳定对外暴露 petKey，可在接口层转换。
type UserPetState struct {
	Model
	UserId int64 `gorm:"not null;uniqueIndex" json:"userId" form:"userId"`

	EquippedPetId int64 `gorm:"not null;default:0" json:"equippedPetId" form:"equippedPetId"`
	// EquipDayName 记录当天是否已切换（P0：每日仅可切换一次）
	EquipDayName int `gorm:"type:int;not null;default:0;index" json:"equipDayName" form:"equipDayName"`

	CreateTime int64 `gorm:"not null;default:0" json:"createTime" form:"createTime"`
	UpdateTime int64 `gorm:"not null;default:0" json:"updateTime" form:"updateTime"`
}

// UserPet 用户拥有的龟种资产。
type UserPet struct {
	Model
	UserId int64 `gorm:"not null;index;uniqueIndex:idx_user_pet" json:"userId" form:"userId"`
	PetId  int64 `gorm:"not null;index;uniqueIndex:idx_user_pet" json:"petId" form:"petId"`

	Level int `gorm:"not null;default:1" json:"level" form:"level"`
	XP    int `gorm:"not null;default:0" json:"xp" form:"xp"`

	ObtainedAt int64 `gorm:"not null;default:0" json:"obtainedAt" form:"obtainedAt"`

	CreateTime int64 `gorm:"not null;default:0" json:"createTime" form:"createTime"`
	UpdateTime int64 `gorm:"not null;default:0" json:"updateTime" form:"updateTime"`
}

// PetDailySettleLog 每日登录结算幂等记录。
//
// 幂等键：user_id + day_name（唯一）。
// DetailJSON 记录当日结算 summary（给重复登录返回 alreadySettled + items）。
type PetDailySettleLog struct {
	Model
	UserId  int64 `gorm:"not null;index;uniqueIndex:idx_pet_daily_settle" json:"userId" form:"userId"`
	DayName int   `gorm:"type:int;not null;index;uniqueIndex:idx_pet_daily_settle" json:"dayName" form:"dayName"`

	DetailJSON string `gorm:"type:text" json:"detail" form:"detail"`

	CreateTime int64 `gorm:"not null;default:0" json:"createTime" form:"createTime"`
}
