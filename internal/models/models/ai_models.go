package models

// AIMessage AI 聊天消息与主动推送记录。
type AIMessage struct {
	Model
	UserId           int64  `gorm:"not null;index:idx_ai_message_user_time;index:idx_ai_message_user_scene" json:"userId" form:"userId"`
	Role             string `gorm:"size:16;not null;index" json:"role" form:"role"` // user/assistant/system
	Scene            string `gorm:"size:32;not null;index:idx_ai_message_user_scene" json:"scene" form:"scene"`
	Content          string `gorm:"type:text;not null" json:"content" form:"content"`
	ModelName        string `gorm:"size:64" json:"model" form:"model"`
	PromptTokens     int    `gorm:"not null;default:0" json:"promptTokens" form:"promptTokens"`
	CompletionTokens int    `gorm:"not null;default:0" json:"completionTokens" form:"completionTokens"`
	TotalTokens      int    `gorm:"not null;default:0" json:"totalTokens" form:"totalTokens"`
	StaminaCost      int    `gorm:"not null;default:0" json:"staminaCost" form:"staminaCost"`
	ContextType      string `gorm:"size:64;index:idx_ai_message_context" json:"contextType" form:"contextType"`
	ContextId        int64  `gorm:"not null;default:0;index:idx_ai_message_context" json:"contextId" form:"contextId"`
	TemplateId       int64  `gorm:"not null;default:0;index" json:"templateId" form:"templateId"`
	RequestId        string `gorm:"size:128;index" json:"requestId" form:"requestId"`
	Status           int    `gorm:"type:int;not null;default:0;index" json:"status" form:"status"` // 0 unread, 1 read
	CreateTime       int64  `gorm:"not null;default:0;index:idx_ai_message_user_time" json:"createTime" form:"createTime"`
}

// UserAIMemory 用户 AI 记忆 Flag。
type UserAIMemory struct {
	Model
	UserId     int64  `gorm:"not null;uniqueIndex:uk_user_ai_memory_key" json:"userId" form:"userId"`
	FlagKey    string `gorm:"size:64;not null;uniqueIndex:uk_user_ai_memory_key" json:"flagKey" form:"flagKey"`
	FlagValue  string `gorm:"type:text" json:"flagValue" form:"flagValue"`
	FlagMeta   string `gorm:"type:text" json:"flagMeta" form:"flagMeta"`
	CreateTime int64  `gorm:"not null;default:0" json:"createTime" form:"createTime"`
	UpdateTime int64  `gorm:"not null;default:0" json:"updateTime" form:"updateTime"`
}

// DialogueTemplate 主动推送模板。
type DialogueTemplate struct {
	Model
	Scene      string `gorm:"size:64;not null;index:idx_dialogue_template_scene" json:"scene" form:"scene"`
	Content    string `gorm:"type:text;not null" json:"content" form:"content"`
	Enabled    bool   `gorm:"not null;default:true;index:idx_dialogue_template_scene" json:"enabled" form:"enabled"`
	Weight     int    `gorm:"not null;default:1" json:"weight" form:"weight"`
	CreateTime int64  `gorm:"not null;default:0" json:"createTime" form:"createTime"`
	UpdateTime int64  `gorm:"not null;default:0" json:"updateTime" form:"updateTime"`
}

// TemplateUserView 用户模板曝光次数。
type TemplateUserView struct {
	Model
	UserId       int64 `gorm:"not null;uniqueIndex:uk_template_user_view;index:idx_template_user_view_user" json:"userId" form:"userId"`
	TemplateId   int64 `gorm:"not null;uniqueIndex:uk_template_user_view" json:"templateId" form:"templateId"`
	ViewCount    int   `gorm:"not null;default:0;index:idx_template_user_view_user" json:"viewCount" form:"viewCount"`
	LastViewedAt int64 `gorm:"not null;default:0" json:"lastViewedAt" form:"lastViewedAt"`
}

// UserAIPresence 用户 AI 在线/闲置状态。
type UserAIPresence struct {
	UserId            int64  `gorm:"primaryKey;not null" json:"userId" form:"userId"`
	Page              string `gorm:"size:64" json:"page" form:"page"`
	Active            bool   `gorm:"not null;default:false" json:"active" form:"active"`
	FirstSeenAt       int64  `gorm:"not null;default:0" json:"firstSeenAt" form:"firstSeenAt"`
	LastSeenAt        int64  `gorm:"not null;default:0;index" json:"lastSeenAt" form:"lastSeenAt"`
	LastAIInteractAt  int64  `gorm:"not null;default:0" json:"lastAIInteractAt" form:"lastAIInteractAt"`
	LastIdlePushAt    int64  `gorm:"not null;default:0" json:"lastIdlePushAt" form:"lastIdlePushAt"`
	IdlePushCountDate int64  `gorm:"not null;default:0" json:"idlePushCountDate" form:"idlePushCountDate"`
	IdlePushCount     int    `gorm:"not null;default:0" json:"idlePushCount" form:"idlePushCount"`
	UpdateTime        int64  `gorm:"not null;default:0" json:"updateTime" form:"updateTime"`
}

// UserAIStamina 用户 AI 体力状态。
type UserAIStamina struct {
	UserId         int64 `gorm:"primaryKey;not null" json:"userId" form:"userId"`
	Stamina        int   `gorm:"not null;default:0" json:"stamina" form:"stamina"`
	MaxStamina     int   `gorm:"not null;default:5" json:"maxStamina" form:"maxStamina"`
	LastRecoverAt  int64 `gorm:"not null;default:0" json:"lastRecoverAt" form:"lastRecoverAt"`
	DailyUsedDate  int   `gorm:"not null;default:0" json:"dailyUsedDate" form:"dailyUsedDate"`
	DailyUsedCount int   `gorm:"not null;default:0" json:"dailyUsedCount" form:"dailyUsedCount"`
	CreateTime     int64 `gorm:"not null;default:0" json:"createTime" form:"createTime"`
	UpdateTime     int64 `gorm:"not null;default:0" json:"updateTime" form:"updateTime"`
}

// UserAIStaminaLog 用户 AI 体力流水。
type UserAIStaminaLog struct {
	Model
	UserId       int64  `gorm:"not null;index:idx_user_ai_stamina_log_user_time" json:"userId" form:"userId"`
	BizType      string `gorm:"size:32;not null;index:idx_user_ai_stamina_log_biz" json:"bizType" form:"bizType"`
	BizId        string `gorm:"size:128;index:idx_user_ai_stamina_log_biz" json:"bizId" form:"bizId"`
	Amount       int    `gorm:"not null" json:"amount" form:"amount"`
	StaminaAfter int    `gorm:"not null;default:0" json:"staminaAfter" form:"staminaAfter"`
	Remark       string `gorm:"size:256" json:"remark" form:"remark"`
	CreateTime   int64  `gorm:"not null;default:0;index:idx_user_ai_stamina_log_user_time" json:"createTime" form:"createTime"`
}
