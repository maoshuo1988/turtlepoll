package models

// PKTopic 对立PK永恒话题。
type PKTopic struct {
	Model
	Slug            string `gorm:"size:64;not null;uniqueIndex" json:"slug"`
	Title           string `gorm:"size:256;not null" json:"title"`
	SideAName       string `gorm:"size:128;not null" json:"sideAName"`
	SideBName       string `gorm:"size:128;not null" json:"sideBName"`
	Status          string `gorm:"size:16;not null;default:'enabled';index" json:"status"`
	Sort            int    `gorm:"not null;default:0;index" json:"sort"`
	Cover           string `gorm:"size:512" json:"cover"`
	CurrentRoundId  int64  `gorm:"not null;default:0;index" json:"currentRoundId"`
	CurrentSeasonId int64  `gorm:"not null;default:0;index" json:"currentSeasonId"`

	TotalRounds       int64  `gorm:"not null;default:0" json:"totalRounds"`
	WinsA             int64  `gorm:"not null;default:0" json:"winsA"`
	WinsB             int64  `gorm:"not null;default:0" json:"winsB"`
	CurrentStreakSide string `gorm:"size:8;not null;default:''" json:"currentStreakSide"`
	CurrentStreak     int64  `gorm:"not null;default:0" json:"currentStreak"`
	MaxStreakA        int64  `gorm:"not null;default:0" json:"maxStreakA"`
	MaxStreakB        int64  `gorm:"not null;default:0" json:"maxStreakB"`
	LastWinner        string `gorm:"size:8;not null;default:''" json:"lastWinner"`

	CreateTime int64 `gorm:"not null;default:0" json:"createTime"`
	UpdateTime int64 `gorm:"not null;default:0" json:"updateTime"`
}

// PKSeason 对立PK赛季，默认30天。
type PKSeason struct {
	Model
	TopicId     int64  `gorm:"not null;index;uniqueIndex:idx_pk_season_topic_no" json:"topicId"`
	SeasonNo    int    `gorm:"not null;uniqueIndex:idx_pk_season_topic_no" json:"seasonNo"`
	StartTime   int64  `gorm:"not null;index" json:"startTime"`
	EndTime     int64  `gorm:"not null;index" json:"endTime"`
	TotalRounds int64  `gorm:"not null;default:0" json:"totalRounds"`
	WinsA       int64  `gorm:"not null;default:0" json:"winsA"`
	WinsB       int64  `gorm:"not null;default:0" json:"winsB"`
	Champion    string `gorm:"size:8;not null;default:''" json:"champion"`
	Status      string `gorm:"size:16;not null;default:'active';index" json:"status"`
	CreateTime  int64  `gorm:"not null;default:0" json:"createTime"`
	UpdateTime  int64  `gorm:"not null;default:0" json:"updateTime"`
}

// PKRound 对立PK单局。
type PKRound struct {
	Model
	TopicId       int64   `gorm:"not null;index;uniqueIndex:idx_pk_round_topic_no" json:"topicId"`
	SeasonId      int64   `gorm:"not null;index" json:"seasonId"`
	RoundNo       int     `gorm:"not null;uniqueIndex:idx_pk_round_topic_no" json:"roundNo"`
	Phase         string  `gorm:"size:16;not null;default:'betting';index" json:"phase"`
	StartTime     int64   `gorm:"not null;index" json:"startTime"`
	LockTime      int64   `gorm:"not null;index" json:"lockTime"`
	EndTime       int64   `gorm:"not null;index" json:"endTime"`
	NextRoundTime int64   `gorm:"not null;index" json:"nextRoundTime"`
	HeatA         float64 `gorm:"not null;default:0" json:"heatA"`
	HeatB         float64 `gorm:"not null;default:0" json:"heatB"`
	PoolA         int64   `gorm:"not null;default:0" json:"poolA"`
	PoolB         int64   `gorm:"not null;default:0" json:"poolB"`
	BetCountA     int64   `gorm:"not null;default:0" json:"betCountA"`
	BetCountB     int64   `gorm:"not null;default:0" json:"betCountB"`
	CommentCount  int64   `gorm:"not null;default:0" json:"commentCount"`
	LikeCount     int64   `gorm:"not null;default:0" json:"likeCount"`
	DownvoteCount int64   `gorm:"not null;default:0" json:"downvoteCount"`
	Winner        string  `gorm:"size:8;not null;default:'';index" json:"winner"`
	SettledAt     int64   `gorm:"not null;default:0" json:"settledAt"`
	CreateTime    int64   `gorm:"not null;default:0" json:"createTime"`
	UpdateTime    int64   `gorm:"not null;default:0" json:"updateTime"`
}

// PKBet 用户单局下注。
type PKBet struct {
	Model
	TopicId      int64  `gorm:"not null;index" json:"topicId"`
	RoundId      int64  `gorm:"not null;index;uniqueIndex:idx_pk_bet_round_user;uniqueIndex:idx_pk_bet_idem" json:"roundId"`
	UserId       int64  `gorm:"not null;index;uniqueIndex:idx_pk_bet_round_user;uniqueIndex:idx_pk_bet_idem" json:"userId"`
	Side         string `gorm:"size:8;not null;index" json:"side"`
	Amount       int64  `gorm:"not null" json:"amount"`
	RequestId    string `gorm:"size:64;not null;uniqueIndex:idx_pk_bet_idem" json:"requestId"`
	SettleResult string `gorm:"size:16;not null;default:''" json:"settleResult"`
	Payout       int64  `gorm:"not null;default:0" json:"payout"`
	SettledAt    int64  `gorm:"not null;default:0" json:"settledAt"`
	CreateTime   int64  `gorm:"not null;default:0" json:"createTime"`
	UpdateTime   int64  `gorm:"not null;default:0" json:"updateTime"`
}

// PKSettlementItem 对立PK结算明细。
type PKSettlementItem struct {
	Model
	TopicId      int64  `gorm:"not null;index" json:"topicId"`
	RoundId      int64  `gorm:"not null;index;uniqueIndex:idx_pk_settle_round_user" json:"roundId"`
	BetId        int64  `gorm:"not null;index;uniqueIndex" json:"betId"`
	UserId       int64  `gorm:"not null;index;uniqueIndex:idx_pk_settle_round_user" json:"userId"`
	Side         string `gorm:"size:8;not null" json:"side"`
	Result       string `gorm:"size:16;not null" json:"result"`
	StakeAmount  int64  `gorm:"not null;default:0" json:"stakeAmount"`
	PayoutAmount int64  `gorm:"not null;default:0" json:"payoutAmount"`
	Paid         bool   `gorm:"not null;default:false" json:"paid"`
	CreateTime   int64  `gorm:"not null;default:0" json:"createTime"`
	UpdateTime   int64  `gorm:"not null;default:0" json:"updateTime"`
}

// PKCommentMeta 复用通用评论时的PK归属信息。
type PKCommentMeta struct {
	Model
	CommentId     int64   `gorm:"not null;uniqueIndex" json:"commentId"`
	TopicId       int64   `gorm:"not null;index" json:"topicId"`
	RoundId       int64   `gorm:"not null;index" json:"roundId"`
	Side          string  `gorm:"size:8;not null;index" json:"side"`
	QualityScore  float64 `gorm:"not null;default:1" json:"qualityScore"`
	DownvoteCount int64   `gorm:"not null;default:0" json:"downvoteCount"`
	HeatScore     float64 `gorm:"not null;default:0" json:"heatScore"`
	CreateTime    int64   `gorm:"not null;default:0" json:"createTime"`
	UpdateTime    int64   `gorm:"not null;default:0" json:"updateTime"`
}

// PKAction 对立PK热度来源与幂等动作。
type PKAction struct {
	Model
	TopicId    int64   `gorm:"not null;index" json:"topicId"`
	RoundId    int64   `gorm:"not null;index;uniqueIndex:idx_pk_action_idem" json:"roundId"`
	UserId     int64   `gorm:"not null;index;uniqueIndex:idx_pk_action_idem" json:"userId"`
	Side       string  `gorm:"size:8;not null;index" json:"side"`
	ActionType string  `gorm:"size:32;not null;index;uniqueIndex:idx_pk_action_idem" json:"actionType"`
	EntityType string  `gorm:"size:32;not null;uniqueIndex:idx_pk_action_idem" json:"entityType"`
	EntityId   int64   `gorm:"not null;uniqueIndex:idx_pk_action_idem" json:"entityId"`
	Amount     int64   `gorm:"not null;default:0" json:"amount"`
	Heat       float64 `gorm:"not null;default:0" json:"heat"`
	AntiSpam   float64 `gorm:"not null;default:1" json:"antiSpam"`
	RequestId  string  `gorm:"size:64;not null;default:'';index" json:"requestId"`
	CreateTime int64   `gorm:"not null;default:0" json:"createTime"`
}
