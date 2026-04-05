package models

import (
	"bbs-go/internal/models/constants"
	"database/sql"
	"time"
)

// User 用户
type User struct {
	Model
	Type             int              `gorm:"not null;default:0" json:"type" form:"type"`                          // 用户类型（0：用户、1：员工）
	Phone            sql.NullString   `gorm:"size:16;unique;" json:"phone" form:"phone"`                           // 电话
	Username         sql.NullString   `gorm:"size:32;unique;" json:"username" form:"username"`                     // 用户名
	Email            sql.NullString   `gorm:"size:128;unique;" json:"email" form:"email"`                          // 邮箱
	EmailVerified    bool             `gorm:"not null;default:false" json:"emailVerified" form:"emailVerified"`    // 邮箱是否验证
	Nickname         string           `gorm:"size:16;" json:"nickname" form:"nickname"`                            // 昵称
	Avatar           string           `gorm:"type:text" json:"avatar" form:"avatar"`                               // 头像
	Gender           constants.Gender `gorm:"size:16;default:''" json:"gender" form:"gender"`                      // 性别
	Birthday         *time.Time       `json:"birthday" form:"birthday"`                                            // 生日
	BackgroundImage  string           `gorm:"type:text" json:"backgroundImage" form:"backgroundImage"`             // 个人中心背景图片
	Password         string           `gorm:"size:512" json:"password" form:"password"`                            // 密码
	HomePage         string           `gorm:"size:1024" json:"homePage" form:"homePage"`                           // 个人主页
	Description      string           `gorm:"type:text" json:"description" form:"description"`                     // 个人描述
	Score            int              `gorm:"type:int;not null;index:idx_user_score" json:"score" form:"score"`    // 积分
	Exp              int              `gorm:"type:int;not null;default:0" json:"exp" form:"exp"`                   // 经验
	Level            int              `gorm:"type:int;not null;default:1" json:"level" form:"level"`               // 等级（从 1 开始）
	Status           int              `gorm:"type:int;index:idx_user_status;not null" json:"status" form:"status"` // 状态
	TopicCount       int              `gorm:"type:int;not null" json:"topicCount" form:"topicCount"`               // 帖子数量
	CommentCount     int              `gorm:"type:int;not null" json:"commentCount" form:"commentCount"`           // 跟帖数量
	FollowCount      int              `gorm:"type:int;not null" json:"followCount" form:"followCount"`             // 关注数量
	FansCount        int              `gorm:"type:int;not null" json:"fansCount" form:"fansCount"`                 // 粉丝数量
	Roles            string           `gorm:"type:text" json:"roles" form:"roles"`                                 // 角色
	ForbiddenEndTime int64            `gorm:"not null;default:0" json:"forbiddenEndTime" form:"forbiddenEndTime"`  // 禁言结束时间
	CreateTime       int64            `json:"createTime" form:"createTime"`                                        // 创建时间
	UpdateTime       int64            `json:"updateTime" form:"updateTime"`                                        // 更新时间
}
