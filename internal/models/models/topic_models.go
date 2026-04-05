package models

import "bbs-go/internal/models/constants"

// TopicNode 话题节点
type TopicNode struct {
	Model
	Name        string `gorm:"size:32;unique" json:"name" form:"name"`                 // 名称
	Description string `gorm:"size:1024" json:"description" form:"description"`        // 描述
	Logo        string `gorm:"size:1024" json:"logo" form:"logo"`                      // 图标
	SortNo      int    `gorm:"type:int;index:idx_sort_no" json:"sortNo" form:"sortNo"` // 排序编号
	Status      int    `gorm:"type:int;not null" json:"status" form:"status"`          // 状态
	CreateTime  int64  `json:"createTime" form:"createTime"`                           // 创建时间
}

// Topic 话题
//
// 注意：表名与字段名保持历史兼容（同 package 多文件拆分）。
type Topic struct {
	Model
	Type              constants.TopicType   `gorm:"type:int;not null;default:0" json:"type" form:"type"`                             // 类型
	NodeId            int64                 `gorm:"not null;index:idx_node_id;" json:"nodeId" form:"nodeId"`                         // 节点编号
	UserId            int64                 `gorm:"not null;index:idx_topic_user_id;" json:"userId" form:"userId"`                   // 用户
	Title             string                `gorm:"size:128" json:"title" form:"title"`                                              // 标题
	ContentType       constants.ContentType `gorm:"size:32;default:markdown" json:"contentType" form:"contentType"`                  // 内容类型（html/markdown）
	Content           string                `gorm:"type:text" json:"content" form:"content"`                                         // 内容
	ImageList         string                `gorm:"type:text" json:"imageList" form:"imageList"`                                     // 图片
	HideContent       string                `gorm:"type:text" json:"hideContent" form:"hideContent"`                                 // 回复可见内容
	VoteId            int64                 `gorm:"not null;default:0" json:"voteId" form:"voteId"`                                  // 投票ID
	Recommend         bool                  `gorm:"not null;index:idx_recommend" json:"recommend" form:"recommend"`                  // 是否推荐
	RecommendTime     int64                 `gorm:"not null" json:"recommendTime" form:"recommendTime"`                              // 推荐时间
	Sticky            bool                  `gorm:"not null;index:idx_sticky_sticky_time" json:"sticky" form:"sticky"`               // 置顶
	StickyTime        int64                 `gorm:"not null;index:idx_sticky_sticky_time" json:"stickyTime" form:"stickyTime"`       // 置顶时间
	ViewCount         int64                 `gorm:"not null" json:"viewCount" form:"viewCount"`                                      // 查看数量
	CommentCount      int64                 `gorm:"not null" json:"commentCount" form:"commentCount"`                                // 跟帖数量
	LikeCount         int64                 `gorm:"not null" json:"likeCount" form:"likeCount"`                                      // 点赞数量
	Status            int                   `gorm:"type:int;index:idx_topic_status;" json:"status" form:"status"`                    // 状态：0：正常、1：删除
	LastCommentTime   int64                 `gorm:"index:idx_topic_last_comment_time" json:"lastCommentTime" form:"lastCommentTime"` // 最后回复时间
	LastCommentUserId int64                 `json:"lastCommentUserId" form:"lastCommentUserId"`                                      // 最后回复用户
	UserAgent         string                `gorm:"size:1024" json:"userAgent" form:"userAgent"`                                     // UserAgent
	Ip                string                `gorm:"size:128" json:"ip" form:"ip"`                                                    // IP
	IpLocation        string                `gorm:"size:64" json:"ipLocation" form:"ipLocation"`                                     // IP属地
	CreateTime        int64                 `gorm:"index:idx_topic_create_time" json:"createTime" form:"createTime"`                 // 创建时间
	ExtraData         string                `gorm:"type:text" json:"extraData" form:"extraData"`                                     // 扩展数据
}

// TopicTag 主题标签
// （topicId + tagId 组合关系）
type TopicTag struct {
	Model
	TopicId           int64 `gorm:"not null;index:idx_topic_tag_topic_id;" json:"topicId" form:"topicId"`                // 主题编号
	TagId             int64 `gorm:"not null;index:idx_topic_tag_tag_id;" json:"tagId" form:"tagId"`                      // 标签编号
	Status            int64 `gorm:"not null;index:idx_topic_tag_status" json:"status" form:"status"`                     // 状态：正常、删除
	LastCommentTime   int64 `gorm:"index:idx_topic_tag_last_comment_time" json:"lastCommentTime" form:"lastCommentTime"` // 最后回复时间
	LastCommentUserId int64 `json:"lastCommentUserId" form:"lastCommentUserId"`                                          // 最后回复用户
	CreateTime        int64 `json:"createTime" form:"createTime"`                                                        // 创建时间
}

// Vote 投票
type Vote struct {
	Model
	Type        constants.VoteType `json:"type" form:"type" redis:"type"`                                          // 投票类型(1:单选 / 2:多选)
	Title       string             `gorm:"size:128" json:"title" form:"title" redis:"title"`                       // 标题
	ExpiredAt   int64              `gorm:"not null" json:"expiredAt" form:"expiredAt" redis:"expiredAt"`           // 截止日期
	TopicId     int64              `gorm:"not null" json:"topicId" form:"topicId" redis:"topicId"`                 // 帖子ID
	UserId      int64              `gorm:"not null" json:"userId" form:"userId" redis:"userId"`                    // 用户ID
	VoteNum     int                `gorm:"not null" json:"voteNum" form:"voteNum" redis:"voteNum"`                 // 可投票数量
	OptionCount int                `gorm:"not null" json:"optionCount" form:"optionCount" redis:"optionCount"`     // 选项数量
	VoteCount   int                `gorm:"not null;default:0" json:"voteCount" form:"voteCount" redis:"voteCount"` // 投票数量
	CreateTime  int64              `gorm:"not null" json:"createTime" form:"createTime" redis:"createTime"`        // 创建时间
}

// VoteOption 投票选项
type VoteOption struct {
	Model
	VoteId     int64  `gorm:"not null;index:idx_vote_id" json:"voteId" form:"voteId" redis:"voteId"`  // 投票ID
	Content    string `gorm:"size:256" json:"content" form:"content" redis:"content"`                 // 选项内容
	SortNo     int    `gorm:"not null" json:"sortNo" form:"sortNo" redis:"sortNo"`                    // 排序
	VoteCount  int    `gorm:"not null;default:0" json:"voteCount" form:"voteCount" redis:"voteCount"` // 票数
	CreateTime int64  `gorm:"not null" json:"createTime" form:"createTime" redis:"createTime"`        // 创建时间
}

// VoteRecord 投票记录
type VoteRecord struct {
	Model
	UserId     int64  `gorm:"uniqueIndex:idx_user_vote" json:"userId" form:"userId"` // 用户ID
	VoteId     int64  `gorm:"uniqueIndex:idx_user_vote" json:"voteId" form:"voteId"` // 投票ID
	OptionIds  string `gorm:"type:text" json:"optionIds" form:"optionIds"`           // 选项ID列表，逗号分隔
	CreateTime int64  `json:"createTime" form:"createTime"`                          // 投票时间
}
