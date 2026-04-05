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
