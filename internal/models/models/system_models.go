package models

// SysConfig 系统配置
type SysConfig struct {
	Model
	Key         string `gorm:"not null;size:128;unique" json:"key" form:"key"`  // 配置key
	Value       string `gorm:"type:text" json:"value" form:"value"`             // 配置值
	Name        string `gorm:"not null;size:128" json:"name" form:"name"`       // 配置名称
	Description string `gorm:"size:1024" json:"description" form:"description"` // 配置描述
	CreateTime  int64  `gorm:"not null" json:"createTime" form:"createTime"`    // 创建时间
	UpdateTime  int64  `gorm:"not null" json:"updateTime" form:"updateTime"`    // 更新时间
}

// Link 友链
type Link struct {
	Model
	Url        string `gorm:"not null;type:text" json:"url" form:"url"`      // 链接
	Title      string `gorm:"not null;size:128" json:"title" form:"title"`   // 标题
	Summary    string `gorm:"size:1024" json:"summary" form:"summary"`       // 站点描述
	Status     int    `gorm:"type:int;not null" json:"status" form:"status"` // 状态
	CreateTime int64  `gorm:"not null" json:"createTime" form:"createTime"`  // 创建时间
}
