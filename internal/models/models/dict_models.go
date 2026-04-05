package models

// DictType 字典分类。
type DictType struct {
	Model
	Name       string `gorm:"size:32" json:"name" form:"name"`
	Code       string `gorm:"size:64;unique" json:"code" form:"code"`
	Status     int    `gorm:"not null;default:0" json:"status" form:"status"`
	Remark     string `gorm:"size:512" json:"remark" form:"remark"`
	CreateTime int64  `gorm:"not null;default:0" json:"createTime" form:"createTime"` // 创建时间
	UpdateTime int64  `gorm:"not null;default:0" json:"updateTime" form:"updateTime"` // 更新时间
}

// Dict 字典项。
type Dict struct {
	Model
	TypeId     int64  `gorm:"uniqueIndex:idx_dict_name" json:"typeId" form:"typeId"`     // 分类
	ParentId   int64  `gorm:"default:0" json:"parentId" form:"parentId"`                 // 上级
	Name       string `gorm:"size:64;uniqueIndex:idx_dict_name" json:"name" form:"name"` // 名称
	Label      string `gorm:"size:64" json:"label" form:"label"`                         // Label
	Value      string `gorm:"type:text" json:"value" form:"value"`                       // Value
	SortNo     int    `gorm:"not null;default:0" json:"sortNo" form:"sortNo"`            // 排序
	Status     int    `gorm:"not null;default:0" json:"status" form:"status"`            // 状态
	CreateTime int64  `gorm:"not null;default:0" json:"createTime" form:"createTime"`    // 创建时间
	UpdateTime int64  `gorm:"not null;default:0" json:"updateTime" form:"updateTime"`    // 更新时间
}
