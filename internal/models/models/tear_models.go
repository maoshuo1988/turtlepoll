package models

// TearInteractLog 撕裂带互动日志，记录 option 选择与互动动作。
type TearInteractLog struct {
	Model
	EventType        string  `gorm:"size:32;not null;index;uniqueIndex:idx_tear_interact_idem" json:"eventType"`
	EventId          int64   `gorm:"not null;index" json:"eventId"`
	TopicId          int64   `gorm:"not null;index" json:"topicId"`
	RoundId          int64   `gorm:"not null;index;uniqueIndex:idx_tear_interact_idem" json:"roundId"`
	UserId           int64   `gorm:"not null;index;uniqueIndex:idx_tear_interact_idem" json:"userId"`
	OptionAtAction   string  `gorm:"size:16;not null;index" json:"optionAtAction"`
	ActionType       string  `gorm:"size:32;not null;index;uniqueIndex:idx_tear_interact_idem" json:"actionType"`
	EntityType       string  `gorm:"size:32;not null;uniqueIndex:idx_tear_interact_idem" json:"entityType"`
	EntityId         int64   `gorm:"not null;uniqueIndex:idx_tear_interact_idem" json:"entityId"`
	HeatContribution float64 `gorm:"not null;default:0" json:"heatContribution"`
	RequestId        string  `gorm:"size:64;not null;default:'';index" json:"requestId"`
	CreateTime       int64   `gorm:"not null;default:0" json:"createTime"`
}

// TearUserEventStat 撕裂带用户事件汇总。
type TearUserEventStat struct {
	Model
	EventType        string  `gorm:"size:32;not null;index;uniqueIndex:idx_tear_user_event" json:"eventType"`
	EventId          int64   `gorm:"not null;index" json:"eventId"`
	TopicId          int64   `gorm:"not null;index" json:"topicId"`
	RoundId          int64   `gorm:"not null;index;uniqueIndex:idx_tear_user_event" json:"roundId"`
	UserId           int64   `gorm:"not null;index;uniqueIndex:idx_tear_user_event" json:"userId"`
	BetOption        string  `gorm:"size:16;not null;default:''" json:"betOption"`
	BetAmount        int64   `gorm:"not null;default:0" json:"betAmount"`
	ActionCount      int64   `gorm:"not null;default:0" json:"actionCount"`
	CommentCount     int64   `gorm:"not null;default:0" json:"commentCount"`
	ReplyCount       int64   `gorm:"not null;default:0" json:"replyCount"`
	LikeCount        int64   `gorm:"not null;default:0" json:"likeCount"`
	HeatContribution float64 `gorm:"not null;default:0" json:"heatContribution"`
	CreateTime       int64   `gorm:"not null;default:0" json:"createTime"`
	UpdateTime       int64   `gorm:"not null;default:0" json:"updateTime"`
}

// TearCampMember 撕裂带阵营锁定关系。
type TearCampMember struct {
	Model
	EventType       string `gorm:"size:32;not null;index;uniqueIndex:idx_tear_camp_event_user" json:"eventType"`
	EventId         int64  `gorm:"not null;index" json:"eventId"`
	TopicId         int64  `gorm:"not null;index;uniqueIndex:idx_tear_camp_event_user" json:"topicId"`
	RoundId         int64  `gorm:"not null;index;uniqueIndex:idx_tear_camp_event_user" json:"roundId"`
	UserId          int64  `gorm:"not null;index;uniqueIndex:idx_tear_camp_event_user" json:"userId"`
	Option          string `gorm:"size:16;not null;index" json:"option"`
	LockType        string `gorm:"size:16;not null;default:'INTERACT'" json:"lockType"`
	FirstActionTime int64  `gorm:"not null;default:0" json:"firstActionTime"`
	CreateTime      int64  `gorm:"not null;default:0" json:"createTime"`
	UpdateTime      int64  `gorm:"not null;default:0" json:"updateTime"`
}

// TearHeatSnapshot 撕裂带热度快照。
type TearHeatSnapshot struct {
	Model
	EventType    string  `gorm:"size:32;not null;index;uniqueIndex:idx_tear_snapshot_idem" json:"eventType"`
	EventId      int64   `gorm:"not null;index;uniqueIndex:idx_tear_snapshot_idem" json:"eventId"`
	TopicId      int64   `gorm:"not null;index" json:"topicId"`
	RoundId      int64   `gorm:"not null;index;uniqueIndex:idx_tear_snapshot_idem" json:"roundId"`
	Option       string  `gorm:"size:16;not null;index;uniqueIndex:idx_tear_snapshot_idem" json:"option"`
	HLike        float64 `gorm:"not null;default:0" json:"hLike"`
	HComment     float64 `gorm:"not null;default:0" json:"hComment"`
	HCoin        float64 `gorm:"not null;default:0" json:"hCoin"`
	HTotal       float64 `gorm:"not null;default:0" json:"hTotal"`
	SnapshotType string  `gorm:"size:16;not null;default:'CHECKPOINT';index;uniqueIndex:idx_tear_snapshot_idem" json:"snapshotType"`
	FreezeSource string  `gorm:"size:16;not null;default:'ON_DEMAND'" json:"freezeSource"`
	SnapshotTime int64   `gorm:"not null;default:0;index" json:"snapshotTime"`
	CreateTime   int64   `gorm:"not null;default:0" json:"createTime"`
	UpdateTime   int64   `gorm:"not null;default:0" json:"updateTime"`
}
