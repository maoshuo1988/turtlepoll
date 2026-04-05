package models

import "bbs-go/internal/models/constants"

// 内容域：Tag/Article/Comment/Favorite

// 标签
type Tag struct {
	Model
	Name        string `gorm:"size:32;unique;not null" json:"name" form:"name"`
	Description string `gorm:"size:1024" json:"description" form:"description"`
	Status      int    `gorm:"type:int;index:idx_tag_status;not null" json:"status" form:"status"`
	CreateTime  int64  `json:"createTime" form:"createTime"`
	UpdateTime  int64  `json:"updateTime" form:"updateTime"`
}

// 文章
type Article struct {
	Model
	UserId       int64                 `gorm:"index:idx_article_user_id" json:"userId" form:"userId"`           // 所属用户编号
	Title        string                `gorm:"size:128;not null;" json:"title" form:"title"`                    // 标题
	Summary      string                `gorm:"type:text" json:"summary" form:"summary"`                         // 摘要
	Content      string                `gorm:"type:text;not null;" json:"content" form:"content"`               // 内容
	ContentType  constants.ContentType `gorm:"type:varchar(32);not null" json:"contentType" form:"contentType"` // 内容类型：markdown、html
	Cover        string                `gorm:"type:text;" json:"cover" form:"cover"`                            // 封面图
	Status       int                   `gorm:"type:int;index:idx_article_status" json:"status" form:"status"`   // 状态
	SourceUrl    string                `gorm:"type:text" json:"sourceUrl" form:"sourceUrl"`                     // 原文链接
	ViewCount    int64                 `gorm:"not null;" json:"viewCount" form:"viewCount"`                     // 查看数量
	CommentCount int64                 `gorm:"default:0" json:"commentCount" form:"commentCount"`               // 评论数量
	LikeCount    int64                 `gorm:"default:0" json:"likeCount" form:"likeCount"`                     // 点赞数量
	CreateTime   int64                 `json:"createTime" form:"createTime"`                                    // 创建时间
	UpdateTime   int64                 `json:"updateTime" form:"updateTime"`                                    // 更新时间
}

// 文章标签
type ArticleTag struct {
	Model
	ArticleId  int64 `gorm:"not null;index:idx_article_id;" json:"articleId" form:"articleId"`  // 文章编号
	TagId      int64 `gorm:"not null;index:idx_article_tag_tag_id;" json:"tagId" form:"tagId"`  // 标签编号
	Status     int64 `gorm:"not null;index:idx_article_tag_status" json:"status" form:"status"` // 状态：正常、删除
	CreateTime int64 `json:"createTime" form:"createTime"`                                      // 创建时间
}

// 评论
type Comment struct {
	Model
	UserId       int64                 `gorm:"index:idx_comment_user_id;not null" json:"userId" form:"userId"`                     // 用户编号
	EntityType   string                `gorm:"size:64;index:idx_comment_entity_type;not null" json:"entityType" form:"entityType"` // 被评论实体类型
	EntityId     int64                 `gorm:"index:idx_comment_entity_id;not null" json:"entityId" form:"entityId"`               // 被评论实体编号
	Content      string                `gorm:"type:text;not null" json:"content" form:"content"`                                   // 内容
	ImageList    string                `gorm:"type:text" json:"imageList" form:"imageList"`                                        // 图片
	ContentType  constants.ContentType `gorm:"type:varchar(32);not null" json:"contentType" form:"contentType"`                    // 内容类型：markdown、html
	QuoteId      int64                 `gorm:"not null"  json:"quoteId" form:"quoteId"`                                            // 引用的评论编号
	LikeCount    int64                 `gorm:"not null;default:0" json:"likeCount" form:"likeCount"`                               // 点赞数量
	CommentCount int64                 `gorm:"not null;default:0" json:"commentCount" form:"commentCount"`                         // 评论数量
	UserAgent    string                `gorm:"size:1024" json:"userAgent" form:"userAgent"`                                        // UserAgent
	Ip           string                `gorm:"size:128" json:"ip" form:"ip"`                                                       // IP
	IpLocation   string                `gorm:"size:64" json:"ipLocation" form:"ipLocation"`                                        // IP属地
	Status       int                   `gorm:"type:int;index:idx_comment_status" json:"status" form:"status"`                      // 状态：0：待审核、1：审核通过、2：审核失败、3：已发布
	CreateTime   int64                 `json:"createTime" form:"createTime"`                                                       // 创建时间
}

// 收藏
type Favorite struct {
	Model
	UserId     int64  `gorm:"index:idx_favorite_user_id;not null" json:"userId" form:"userId"`                     // 用户编号
	EntityType string `gorm:"size:32;index:idx_favorite_entity_type;not null" json:"entityType" form:"entityType"` // 收藏实体类型
	EntityId   int64  `gorm:"index:idx_favorite_entity_id;not null" json:"entityId" form:"entityId"`               // 收藏实体编号
	CreateTime int64  `json:"createTime" form:"createTime"`                                                        // 创建时间
}
