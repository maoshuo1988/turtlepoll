package models

import "bbs-go/internal/models/constants"

// Battle 资金相关的系统账户约定（userId 为负数，保证与正常用户 userId 不冲突）。
const (
	// BattlePoolUserId 资金池（托管账户）：下注本金扣款后转入该账户；结算提取时从该账户出池。
	BattlePoolUserId int64 = -1
	// BattleBurnUserId 销毁账户：罚没/处罚等销毁资金转入该账户用于审计；不参与任何出池。
	BattleBurnUserId int64 = -2
)

var Models = []interface{}{
	&Migration{},
	&UserRole{}, &Role{}, &Menu{}, &RoleMenu{}, &Api{}, &MenuApi{}, &DictType{}, &Dict{},

	&User{}, &UserToken{}, &ThirdUser{}, &Tag{}, &Article{}, &ArticleTag{}, &Comment{}, &Favorite{}, &Topic{}, &TopicNode{},
	&TopicTag{}, &UserLike{}, &Message{}, &SysConfig{}, &Link{},
	&TaskConfig{}, &UserTaskEvent{}, &UserTaskLog{},
	&Badge{}, &UserBadge{},
	&LevelConfig{},
	&Vote{}, &VoteOption{}, &VoteRecord{},
	&UserScoreLog{}, &UserExpLog{},
	&OperateLog{}, &EmailLog{}, &EmailCode{}, &SmsCode{}, &CheckIn{}, &UserFollow{}, &UserFeed{}, &UserReport{},
	&ForbiddenWord{},

	// TurtlePoll: football schedule & prediction market
	&MatchSchedule{},
	&PredictMarket{},
	&PredictContext{},
	&PredictTag{},
	&PredictTagStat{},
	&UserCoin{},
	&UserCoinLog{},
	&PredictBet{},

	// Battle Square
	&Battle{},
	&BattleBet{},
	&BattleChallengeAction{},
	&BattleLedger{},
	&BattleSettlement{},
	&BattleSettlementItem{},

	// TurtlePoll: Pet
	&PetDefinition{},
	&FeatureCatalogItem{},
	&GachaPoolConfig{},
	&UserPetState{},
	&UserPet{},
	&PetDailySettleLog{},
}

type Model struct {
	Id int64 `gorm:"primaryKey;autoIncrement" json:"id" form:"id"`
}

type UserToken struct {
	Model
	Token      string `gorm:"size:32;unique;not null" json:"token" form:"token"`
	UserId     int64  `gorm:"not null;index:idx_user_token_user_id;" json:"userId" form:"userId"`
	Username   string `gorm:"size:64;index:idx_user_token_username" json:"username" form:"username"`
	ExpiredAt  int64  `gorm:"not null" json:"expiredAt" form:"expiredAt"`
	Status     int    `gorm:"type:int;not null;index:idx_user_token_status" json:"status" form:"status"`
	CreateTime int64  `gorm:"not null" json:"createTime" form:"createTime"`
}

type ThirdUser struct {
	Model
	UserId     int64               `gorm:"not null;uniqueIndex:idx_third_user_user_id" json:"userId" form:"userId"`
	OpenId     string              `gorm:"size:64;not null;uniqueIndex:idx_open_id" json:"openId" form:"openId"`
	ThirdType  constants.ThirdType `gorm:"size:32;not null;uniqueIndex:idx_open_id;uniqueIndex:idx_third_user_user_id" json:"thirdType" form:"thirdType"`
	Nickname   string              `gorm:"size:32" json:"nickname" form:"nickname"`
	Avatar     string              `gorm:"size:1024" json:"avatar" form:"avatar"`
	ExtraData  string              `gorm:"type:text" json:"extraData" form:"extraData"`
	CreateTime int64               `json:"createTime" form:"createTime"`
	UpdateTime int64               `json:"updateTime" form:"updateTime"`
}
