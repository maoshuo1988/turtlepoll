package models

// Migration 用于记录迁移执行情况。
type Migration struct {
	Model
	Version    int64  `json:"version" form:"version" gorm:"unique"`
	Remark     string `json:"remark" form:"remark" gorm:"type:text"`
	Success    bool   `json:"success" form:"success" gorm:"default:false"`
	ErrorInfo  string `json:"errorInfo" form:"errorInfo" gorm:"type:text"`
	RetryCount int    `json:"retryCount" form:"retryCount"`
	CreateTime int64  `json:"createTime" form:"createTime"`
	UpdateTime int64  `json:"updateTime" form:"updateTime"`
}
