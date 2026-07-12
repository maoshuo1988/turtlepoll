package models

// UserLike 用户点赞
//
// EntityId + EntityType 表示被点赞对象（帖子/评论/文章等）。
type UserLike struct {
	Model
	UserId     int64  `gorm:"not null;uniqueIndex:idx_user_like_unique;" json:"userId" form:"userId"`                                            // 用户
	EntityId   int64  `gorm:"not null;uniqueIndex:idx_user_like_unique;index:idx_user_like_entity;" json:"topicId" form:"topicId"`               // 实体编号
	EntityType string `gorm:"not null;size:32;uniqueIndex:idx_user_like_unique;index:idx_user_like_entity;" json:"entityType" form:"entityType"` // 实体类型
	CreateTime int64  `json:"createTime" form:"createTime"`                                                                                      // 创建时间
}

// UserDislike 用户点踩
//
// EntityId + EntityType 表示被点踩对象（当前版本仅用于 topic 点踩）。
type UserDislike struct {
	Model
	UserId     int64  `gorm:"not null;uniqueIndex:idx_user_dislike_unique;" json:"userId" form:"userId"`                                               // 用户
	EntityId   int64  `gorm:"not null;uniqueIndex:idx_user_dislike_unique;index:idx_user_dislike_entity;" json:"entityId" form:"entityId"`             // 实体编号
	EntityType string `gorm:"not null;size:32;uniqueIndex:idx_user_dislike_unique;index:idx_user_dislike_entity;" json:"entityType" form:"entityType"` // 实体类型
	Status     int    `gorm:"type:int;not null;default:1;index:idx_user_dislike_status;" json:"status" form:"status"`                                  // 状态：0-取消点踩，1-点踩
	CreateTime int64  `json:"createTime" form:"createTime"`                                                                                            // 创建时间
}

// Message 站内消息
type Message struct {
	Model
	FromId       int64  `gorm:"not null" json:"fromId" form:"fromId"`                            // 消息发送人
	UserId       int64  `gorm:"not null;index:idx_message_user_id;" json:"userId" form:"userId"` // 用户编号(消息接收人)
	Title        string `gorm:"size:1024" json:"title" form:"title"`                             // 消息标题
	Content      string `gorm:"type:text;not null" json:"content" form:"content"`                // 消息内容
	QuoteContent string `gorm:"type:text" json:"quoteContent" form:"quoteContent"`               // 引用内容
	Type         int    `gorm:"type:int;not null" json:"type" form:"type"`                       // 消息类型：评论/点赞/收藏/推荐/删除/文章评论/等级提升/获得勋章
	ExtraData    string `gorm:"type:text" json:"extraData" form:"extraData"`                     // 扩展数据
	Status       int    `gorm:"type:int;not null" json:"status" form:"status"`                   // 状态：0：未读、1：已读
	CreateTime   int64  `json:"createTime" form:"createTime"`                                    // 创建时间
}

// MessageNotifyTemplate 主站消息通知模板。
type MessageNotifyTemplate struct {
	Model
	BusinessCode      string `gorm:"size:64;not null;uniqueIndex:uk_message_notify_template_biz_tpl;index:idx_message_notify_template_status" json:"businessCode" form:"businessCode"`
	TemplateCode      string `gorm:"size:128;not null;uniqueIndex:uk_message_notify_template_biz_tpl" json:"templateCode" form:"templateCode"`
	SubjectTemplate   string `gorm:"size:255;not null" json:"subjectTemplate" form:"subjectTemplate"`
	BodyTemplate      string `gorm:"type:text;not null" json:"bodyTemplate" form:"bodyTemplate"`
	DetailUrlTemplate string `gorm:"size:512" json:"detailUrlTemplate" form:"detailUrlTemplate"`
	TemplateParams    string `gorm:"type:text;not null" json:"templateParams" form:"templateParams"`
	Status            int    `gorm:"type:int;not null;default:1;index:idx_message_notify_template_status" json:"status" form:"status"`
	Remark            string `gorm:"size:512" json:"remark" form:"remark"`
	CreateTime        int64  `gorm:"not null" json:"createTime" form:"createTime"`
	UpdateTime        int64  `gorm:"not null" json:"updateTime" form:"updateTime"`
}

// UserMessageNotifyRecord 用户主站消息通知记录。
type UserMessageNotifyRecord struct {
	Model
	BusinessCode   string `gorm:"size:64;not null;index:idx_user_message_notify_biz;index:idx_user_message_notify_user_business,priority:2" json:"businessCode" form:"businessCode"`
	TemplateCode   string `gorm:"size:128;not null;index:idx_user_message_notify_biz" json:"templateCode" form:"templateCode"`
	TemplateId     int64  `gorm:"not null" json:"templateId" form:"templateId"`
	UserId         int64  `gorm:"not null;index:idx_user_message_notify_user_time,priority:1;index:idx_user_message_notify_user_status,priority:1;index:idx_user_message_notify_user_business,priority:1;index:idx_user_message_notify_idem,priority:1" json:"userId" form:"userId"`
	Subject        string `gorm:"size:255;not null" json:"subject" form:"subject"`
	Body           string `gorm:"type:text;not null" json:"body" form:"body"`
	DetailUrl      string `gorm:"size:512" json:"detailUrl" form:"detailUrl"`
	Status         int    `gorm:"type:int;not null;default:0;index:idx_user_message_notify_user_status,priority:2" json:"status" form:"status"`
	TemplateParams string `gorm:"type:text" json:"templateParams" form:"templateParams"`
	ExtraData      string `gorm:"type:text" json:"extraData" form:"extraData"`
	BizId          string `gorm:"size:128;index:idx_user_message_notify_biz" json:"bizId" form:"bizId"`
	IdempotencyKey string `gorm:"size:191;index:idx_user_message_notify_idem,priority:2" json:"idempotencyKey" form:"idempotencyKey"`
	CreateTime     int64  `gorm:"not null;index:idx_user_message_notify_user_time,priority:2;index:idx_user_message_notify_user_business,priority:3" json:"createTime" form:"createTime"`
	UpdateTime     int64  `gorm:"not null" json:"updateTime" form:"updateTime"`
}
