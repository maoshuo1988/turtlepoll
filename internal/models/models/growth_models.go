package models

import "bbs-go/internal/models/constants"

// TaskConfig 任务配置
type TaskConfig struct {
	Model
	GroupName   constants.TaskGroup `gorm:"size:32;not null;default:'newbie';index:idx_task_config_group_name" json:"groupName" form:"groupName"` // 任务分组
	EventType   string              `gorm:"size:64;not null;index:idx_task_config_event_type" json:"eventType" form:"eventType"`                  // 事件类型
	Title       string              `gorm:"size:64;not null" json:"title" form:"title"`                                                           // 标题（单语言）
	Description string              `gorm:"size:512;not null" json:"description" form:"description"`                                              // 描述（单语言）

	Score   int   `gorm:"type:int;not null;default:0" json:"score" form:"score"`        // 完成一次获得积分
	Exp     int   `gorm:"type:int;not null;default:0" json:"exp" form:"exp"`            // 完成一次获得经验
	BadgeId int64 `gorm:"type:bigint;not null;default:0" json:"badgeId" form:"badgeId"` // 完成一次授予勋章（0 表示无）

	Period         int `gorm:"type:int;not null;default:0;index:idx_task_config_period" json:"period" form:"period"` // 0一次性/1每日...
	MaxFinishCount int `gorm:"type:int;not null;default:1" json:"maxFinishCount" form:"maxFinishCount"`              // 周期内最多完成次数
	EventCount     int `gorm:"type:int;not null;default:1" json:"eventCount" form:"eventCount"`                      // 多少次事件算完成一次

	BtnName   string `gorm:"size:32" json:"btnName" form:"btnName"`                              // 按钮文案（单语言）
	ActionUrl string `gorm:"size:1024" json:"actionUrl" form:"actionUrl"`                        // 按钮跳转
	SortNo    int    `gorm:"type:int;index:idx_task_config_sort_no" json:"sortNo" form:"sortNo"` // 排序

	StartTime int64 `gorm:"type:bigint;not null;default:0;index:idx_task_config_time" json:"startTime" form:"startTime"` // 生效时间（0 表示立即）
	EndTime   int64 `gorm:"type:bigint;not null;default:0;index:idx_task_config_time" json:"endTime" form:"endTime"`     // 结束时间（0 表示不结束）

	Status     int   `gorm:"type:int; not null;default:0;index:idx_task_config_status" json:"status" form:"status"` // 状态
	CreateTime int64 `gorm:"type:bigint;not null" json:"createTime" form:"createTime"`                              // 创建时间
	UpdateTime int64 `gorm:"type:bigint;not null" json:"updateTime" form:"updateTime"`                              // 更新时间
}

// UserTaskEvent 用户任务事件累计（UserId + PeriodKey + TaskId 唯一）
type UserTaskEvent struct {
	Model
	UserId          int64 `gorm:"type:bigint;not null;uniqueIndex:uk_user_task_event_upt;index:idx_user_task_event_user_id" json:"userId" form:"userId"`
	PeriodKey       int   `gorm:"type:int;not null;uniqueIndex:uk_user_task_event_upt;index:idx_user_task_event_period_key" json:"periodKey" form:"periodKey"` // 一次性=0；每日=yyyyMMdd
	TaskId          int64 `gorm:"type:bigint;not null;uniqueIndex:uk_user_task_event_upt;index:idx_user_task_event_task_id" json:"taskId" form:"taskId"`
	EventCount      int   `gorm:"type:int;not null;default:0" json:"eventCount" form:"eventCount"`            // 事件次数（余量）
	TaskFinishCount int   `gorm:"type:int; not null;default:0" json:"taskFinishCount" form:"taskFinishCount"` // 已完成次数（周期内）
	CreateTime      int64 `gorm:"type:bigint;not null" json:"createTime" form:"createTime"`                   // 创建时间
	UpdateTime      int64 `gorm:"type:bigint;not null" json:"updateTime" form:"updateTime"`                   // 更新时间
}

// UserTaskLog 用户任务完成/发奖记录（每完成一次生成一条记录）
type UserTaskLog struct {
	Model
	UserId    int64 `gorm:"type:bigint;not null;uniqueIndex:uk_user_task_log_uptf;index:idx_user_task_log_user_id" json:"userId" form:"userId"`       // 用户编号
	PeriodKey int   `gorm:"type:int;not null;uniqueIndex:uk_user_task_log_uptf;index:idx_user_task_log_period_key" json:"periodKey" form:"periodKey"` // 一次性=0；每日=yyyyMMdd
	TaskId    int64 `gorm:"type:bigint;not null;uniqueIndex:uk_user_task_log_uptf;index:idx_user_task_log_task_id" json:"taskId" form:"taskId"`       // 任务ID
	FinishNo  int   `gorm:"type:int;not null;default:1;uniqueIndex:uk_user_task_log_uptf" json:"finishNo" form:"finishNo"`                            // 周期内第几次完成（1..MaxFinishCount）

	Score   int   `gorm:"type:int;not null;default:0" json:"score" form:"score"`        // 发放积分
	Exp     int   `gorm:"type:int;not null;default:0" json:"exp" form:"exp"`            // 发放经验
	BadgeId int64 `gorm:"type:bigint;not null;default:0" json:"badgeId" form:"badgeId"` // 发放勋章（0 表示无）

	CreateTime int64 `gorm:"type:bigint;not null" json:"createTime" form:"createTime"` // 创建时间
	UpdateTime int64 `gorm:"type:bigint;not null" json:"updateTime" form:"updateTime"` // 更新时间
}

// Badge 勋章配置
type Badge struct {
	Model
	Name        string `gorm:"size:64;not null;uniqueIndex:uk_badge_name" json:"name" form:"name"` // 稳定标识
	Title       string `gorm:"size:64;not null" json:"title" form:"title"`                         // 标题（单语言）
	Description string `gorm:"size:512" json:"description" form:"description"`                     // 描述（单语言）
	Icon        string `gorm:"size:1024" json:"icon" form:"icon"`                                  // 图标
	SortNo      int    `gorm:"type:int;index:idx_badge_sort_no" json:"sortNo" form:"sortNo"`       // 排序
	Status      int    `gorm:"type:int;not null;default:0;index:idx_badge_status" json:"status" form:"status"`
	CreateTime  int64  `gorm:"type:bigint;not null" json:"createTime" form:"createTime"`
	UpdateTime  int64  `gorm:"type:bigint;not null" json:"updateTime" form:"updateTime"`
}

// UserBadge 用户勋章（避免重复授予：UserId + BadgeId 唯一）
type UserBadge struct {
	Model
	UserId     int64  `gorm:"type:bigint;not null;uniqueIndex:uk_user_badge_ub;index:idx_user_badge_user_id" json:"userId" form:"userId"`
	BadgeId    int64  `gorm:"type:bigint;not null;uniqueIndex:uk_user_badge_ub;index:idx_user_badge_badge_id" json:"badgeId" form:"badgeId"`
	SourceType string `gorm:"size:32;not null;index:idx_user_badge_source" json:"sourceType" form:"sourceType"` // 例如：task
	SourceId   string `gorm:"size:64;not null;index:idx_user_badge_source" json:"sourceId" form:"sourceId"`     // 例如：UserTaskLog.Id
	IsWorn     bool   `gorm:"not null;default:false" json:"isWorn" form:"isWorn"`                               // 是否佩戴（可选能力）
	SortNo     int    `gorm:"type:int" json:"sortNo" form:"sortNo"`                                             // 佩戴/展示排序（可选）
	CreateTime int64  `gorm:"type:bigint;not null" json:"createTime" form:"createTime"`
	UpdateTime int64  `gorm:"type:bigint;not null" json:"updateTime" form:"updateTime"`
}

// LevelConfig 等级配置（Level -> NeedExp）
type LevelConfig struct {
	Model
	Level      int    `gorm:"type:int;not null;uniqueIndex:uk_level_config_level" json:"level" form:"level"` // 等级（必须从 1 开始且连续）
	NeedExp    int    `gorm:"type:int;not null" json:"needExp" form:"needExp"`                               // 达到该等级所需累计经验（必须严格递增）
	Title      string `gorm:"size:64" json:"title" form:"title"`                                             // 等级称号（可选）
	Status     int    `gorm:"type:int;not null;default:0;index:idx_level_config_status" json:"status" form:"status"`
	CreateTime int64  `gorm:"type:bigint;not null" json:"createTime" form:"createTime"`
	UpdateTime int64  `gorm:"type:bigint;not null" json:"updateTime" form:"updateTime"`
}

// UserScoreLog 用户积分流水
type UserScoreLog struct {
	Model
	UserId      int64  `gorm:"index:idx_user_score_log_user_id" json:"userId" form:"userId"`           // 用户编号
	SourceType  string `gorm:"size:32;index:idx_user_score_score" json:"sourceType" form:"sourceType"` // 积分来源类型
	SourceId    string `gorm:"size:32;index:idx_user_score_score" json:"sourceId" form:"sourceId"`     // 积分来源编号
	Description string `json:"description" form:"description"`                                         // 描述
	Type        int    `gorm:"type:int" json:"type" form:"type"`                                       // 类型(增加、减少)
	Score       int    `gorm:"type:int" json:"score" form:"score"`                                     // 积分
	CreateTime  int64  `json:"createTime" form:"createTime"`                                           // 创建时间
}

// UserExpLog 用户经验流水
type UserExpLog struct {
	Model
	UserId      int64  `gorm:"index:idx_user_exp_log_user_id" json:"userId" form:"userId"`                // 用户编号
	SourceType  string `gorm:"size:32;index:idx_user_exp_log_source" json:"sourceType" form:"sourceType"` // 经验来源类型
	SourceId    string `gorm:"size:64;index:idx_user_exp_log_source" json:"sourceId" form:"sourceId"`     // 经验来源编号
	Description string `json:"description" form:"description"`                                            // 描述
	Type        int    `gorm:"type:int" json:"type" form:"type"`                                          // 类型(增加、减少)
	Exp         int    `gorm:"type:int" json:"exp" form:"exp"`                                            // 经验
	CreateTime  int64  `json:"createTime" form:"createTime"`                                              // 创建时间
}
