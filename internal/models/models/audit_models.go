package models

// OperateLog 操作日志
type OperateLog struct {
	Model
	UserId      int64  `gorm:"not null;index:idx_operate_log_user_id" json:"userId" form:"userId"`          // 用户编号
	OpType      string `gorm:"not null;index:idx_op_type;size:32" json:"opType" form:"opType"`              // 操作类型
	DataType    string `gorm:"not null;index:idx_operate_log_data;size:32" json:"dataType" form:"dataType"` // 数据类型
	DataId      int64  `gorm:"not null;index:idx_operate_log_data" json:"dataId" form:"dataId" `            // 数据编号
	Description string `gorm:"not null;size:1024" json:"description" form:"description"`                    // 描述
	Ip          string `gorm:"size:128" json:"ip" form:"ip"`                                                // ip地址
	UserAgent   string `gorm:"type:text" json:"userAgent" form:"userAgent"`                                 // UserAgent
	Referer     string `gorm:"type:text" json:"referer" form:"referer"`                                     // Referer
	CreateTime  int64  `json:"createTime" form:"createTime"`                                                // 创建时间
}

// EmailCode 邮箱验证码
type EmailCode struct {
	Model
	UserId     int64  `gorm:"not null;index:idx_email_code_user_id" json:"userId" form:"userId"` // 用户编号
	BizType    string `gorm:"not null;default:'';size:32;index:idx_email_code_biz_type" json:"bizType" form:"bizType"`
	Email      string `gorm:"not null;size:128" json:"email" form:"email"`       // 邮箱
	Code       string `gorm:"not null;size:8" json:"code" form:"code"`           // 验证码
	Token      string `gorm:"not null;size:32;unique" json:"token" form:"token"` // 验证码token
	Title      string `gorm:"size:1024" json:"title" form:"title"`               // 标题
	Content    string `gorm:"type:text" json:"content" form:"content"`           // 内容
	Used       bool   `gorm:"not null" json:"used" form:"used"`                  // 是否使用
	CreateTime int64  `json:"createTime" form:"createTime"`                      // 创建时间
}

// EmailLog 邮件发送记录
type EmailLog struct {
	Model
	ToEmail    string `gorm:"not null;size:128;index:idx_email_log_to_email" json:"toEmail" form:"toEmail"`
	Subject    string `gorm:"size:1024" json:"subject" form:"subject"`
	Content    string `gorm:"type:text" json:"content" form:"content"`
	BizType    string `gorm:"not null;default:'';size:32;index:idx_email_log_biz_type" json:"bizType" form:"bizType"`
	Status     int    `gorm:"not null;index:idx_email_log_status" json:"status" form:"status"`
	ErrorMsg   string `gorm:"type:text" json:"errorMsg" form:"errorMsg"`
	CreateTime int64  `gorm:"index:idx_email_log_create_time" json:"createTime" form:"createTime"`
}

// SmsCode 短信验证码
type SmsCode struct {
	Model
	SmsId      string `gorm:"size:32;unique" json:"smsId" form:"smsId"`
	Phone      string `gorm:"size:32" json:"phone" form:"phone"`
	Code       string `gorm:"size:16" json:"code" form:"code"`
	ExpireAt   int64  `json:"expireAt" form:"expireAt"`
	Status     int    `json:"status" form:"status"`
	CreateTime int64  `json:"createTime" form:"createTime"`
}

// CheckIn 签到
type CheckIn struct {
	Model
	UserId          int64 `gorm:"not null;uniqueIndex:idx_check_in_user_id" json:"userId" form:"userId"` // 用户编号
	LatestDayName   int   `gorm:"type:int;not null;index:idx_latest" json:"dayName" form:"dayName"`      // 最后一次签到
	ConsecutiveDays int   `gorm:"type:int;not null;" json:"consecutiveDays" form:"consecutiveDays"`      // 连续签到天数
	CreateTime      int64 `json:"createTime" form:"createTime"`                                          // 创建时间
	UpdateTime      int64 `gorm:"index:idx_latest" json:"updateTime" form:"updateTime"`                  // 更新时间
}

// UserFollow 粉丝关注
type UserFollow struct {
	Model
	UserId     int64 `gorm:"not null;uniqueIndex:idx_user_follow" json:"userId"`       // 用户编号
	OtherId    int64 `gorm:"not null;uniqueIndex:idx_user_follow" json:"otherId"`      // 对方的ID（被关注用户编号）
	Status     int   `gorm:"type:int;not null" json:"status"`                          // 关注状态
	CreateTime int64 `gorm:"type:bigint;not null" json:"createTime" form:"createTime"` // 创建时间
}

// UserFeed 用户信息流
type UserFeed struct {
	Model
	UserId     int64  `gorm:"not null;uniqueIndex:idx_data;index:idx_user_feed_user_id;index:idx_search" json:"userId"`                 // 用户编号
	DataId     int64  `gorm:"not null;uniqueIndex:idx_data;index:idx_data_id" json:"dataId" form:"dataId"`                              // 数据ID
	DataType   string `gorm:"not null;uniqueIndex:idx_data;index:idx_data_id;index:idx_search;size:32" json:"dataType" form:"dataType"` // 数据类型
	AuthorId   int64  `gorm:"not null;index:idx_user_feed_user_id" json:"authorId" form:"authorId"`                                     // 作者编号
	CreateTime int64  `gorm:"type:bigint;not null;index:idx_search" json:"createTime" form:"createTime"`                                // 数据的创建时间
}

// UserReport 用户举报
type UserReport struct {
	Model
	DataId      int64  `json:"dataId" form:"dataId"`           // 举报数据ID
	DataType    string `json:"dataType" form:"dataType"`       // 举报数据类型
	UserId      int64  `json:"userId" form:"userId"`           // 举报人ID
	Reason      string `json:"reason" form:"reason"`           // 举报原因
	AuditStatus int64  `json:"auditStatus" form:"auditStatus"` // 审核状态
	AuditTime   int64  `json:"auditTime" form:"auditTime"`     // 审核时间
	AuditUserId int64  `json:"auditUserId" form:"auditUserId"` // 审核人ID
	CreateTime  int64  `json:"createTime" form:"createTime"`   // 举报时间
}

// ForbiddenWord 违禁词
type ForbiddenWord struct {
	Model
	Type       string `gorm:"size:16" json:"type" form:"type"`       // 类型：word/regex
	Word       string `gorm:"size:128" json:"word" form:"word"`      // 违禁词
	Remark     string `gorm:"size:1024" json:"remark" form:"remark"` // 备注
	CreateTime int64  `json:"createTime" form:"createTime"`          // 举报时间
}
