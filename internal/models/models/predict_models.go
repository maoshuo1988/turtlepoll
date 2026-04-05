package models

// TurtlePoll: football schedule & prediction market

// MatchSchedule 世界杯/赛事赛程（数据源：football-data.org）
// 这里先做最小可用字段，后续可按需要扩展（比分、比赛状态、阶段、裁判等）。
type MatchSchedule struct {
	Model

	// 数据源标识
	Source       string `gorm:"size:32;not null;index:idx_match_schedule_source_match" json:"source" form:"source"`
	Competition  string `gorm:"size:64;not null" json:"competition" form:"competition"` // e.g. WC
	Season       int    `gorm:"not null;default:0" json:"season" form:"season"`
	ExternalId   int64  `gorm:"not null;uniqueIndex:idx_match_schedule_source_external" json:"externalId" form:"externalId"`
	Matchday     int    `gorm:"not null;default:0" json:"matchday" form:"matchday"`
	Stage        string `gorm:"size:64" json:"stage" form:"stage"`
	GroupName    string `gorm:"size:64" json:"groupName" form:"groupName"`
	Status       string `gorm:"size:32" json:"status" form:"status"` // SCHEDULED/LIVE/FINISHED...
	UtcDate      int64  `gorm:"not null;index:idx_match_schedule_utc_date" json:"utcDate" form:"utcDate"`
	HomeTeam     string `gorm:"size:128" json:"homeTeam" form:"homeTeam"`
	AwayTeam     string `gorm:"size:128" json:"awayTeam" form:"awayTeam"`
	HomeTeamId   int64  `gorm:"not null;default:0" json:"homeTeamId" form:"homeTeamId"`
	AwayTeamId   int64  `gorm:"not null;default:0" json:"awayTeamId" form:"awayTeamId"`
	LastSyncedAt int64  `gorm:"not null;default:0" json:"lastSyncedAt" form:"lastSyncedAt"`

	CreateTime int64 `gorm:"not null;default:0" json:"createTime" form:"createTime"`
	UpdateTime int64 `gorm:"not null;default:0" json:"updateTime" form:"updateTime"`
}

// PredictMarket 预测市场（每个赛程一条市场记录）
type PredictMarket struct {
	Model

	// 来源模型（例如：MatchSchedule）
	SourceModel string `gorm:"size:64;not null;uniqueIndex:idx_predict_market_source" json:"sourceModel" form:"sourceModel"`
	// 来源模型 ID（例如：MatchSchedule.Id）
	SourceModelId int64 `gorm:"not null;uniqueIndex:idx_predict_market_source" json:"sourceModelId" form:"sourceModelId"`

	// 市场基础信息
	Title       string `gorm:"size:256;not null" json:"title" form:"title"`
	MarketType  string `gorm:"size:32;not null" json:"marketType" form:"marketType"` // e.g. 1x2
	Status      string `gorm:"size:32;not null" json:"status" form:"status"`         // OPEN/CLOSED/SETTLED
	CloseTime   int64  `gorm:"not null;default:0" json:"closeTime" form:"closeTime"` // 截止下注时间（先预留）
	Result      string `gorm:"size:32" json:"result" form:"result"`                  // HOME/DRAW/AWAY（先预留）
	ExternalKey string `gorm:"size:128" json:"externalKey" form:"externalKey"`       // 预留：外部业务 key

	// ============ TurtlePoll：外部市场结算（例如 Polymarket Gamma） ============
	// resolved=false 表示未结算或尚未从外部同步到最终结果。
	Resolved bool `gorm:"not null;default:false;index:idx_predict_market_resolved" json:"resolved" form:"resolved"`
	// 外部 winner outcome 的唯一 ID（按原样保存）。
	ResolvedOutcomeId string `gorm:"size:128" json:"resolvedOutcomeId" form:"resolvedOutcomeId"`
	// 外部 winner outcome 的展示名（YES/NO、队名、候选人等）。
	ResolvedOutcomeName string `gorm:"size:255" json:"resolvedOutcomeName" form:"resolvedOutcomeName"`
	// 外部结算时间（若外部不提供，则写首次检测到 resolved 的时间戳）。
	ResolvedAt int64 `gorm:"not null;default:0" json:"resolvedAt" form:"resolvedAt"`

	// ============ TurtlePoll：二元预测市场（A/B）下注池与赔率 ============
	// baseA/baseB：系统默认投放的虚拟底池（用于早期赔率稳定）
	// poolA/poolB：所有用户对 A/B 的下注累计（真实用户下注池）
	// 注意：赔率在下注时锁定到订单里，不在结算时按最新赔率重算。
	BaseA int64 `gorm:"not null;default:500" json:"baseA" form:"baseA"`
	BaseB int64 `gorm:"not null;default:500" json:"baseB" form:"baseB"`
	PoolA int64 `gorm:"not null;default:0" json:"poolA" form:"poolA"`
	PoolB int64 `gorm:"not null;default:0" json:"poolB" form:"poolB"`

	CreateTime int64 `gorm:"not null;default:0" json:"createTime" form:"createTime"`
	UpdateTime int64 `gorm:"not null;default:0" json:"updateTime" form:"updateTime"`
}

// PredictContext 预测事件上下文（与 PredictMarket 一对一）
// 说明：这里放 UI 展示相关的文案、图片、标签等，可随时扩展。
type PredictContext struct {
	Model

	MarketId int64 `gorm:"not null;uniqueIndex" json:"marketId" form:"marketId"`

	// 预测事件名称（展示用）
	EventName string `gorm:"size:256;not null" json:"eventName" form:"eventName"`
	// 图片地址
	ImageUrl string `gorm:"type:text" json:"imageUrl" form:"imageUrl"`
	// 参与人数
	ParticipantCount int64 `gorm:"not null;default:0" json:"participantCount" form:"participantCount"`
	// 正方文案
	ProText string `gorm:"size:256" json:"proText" form:"proText"`
	// 正方投票数
	ProVoteCount int64 `gorm:"not null;default:0" json:"proVoteCount" form:"proVoteCount"`
	// 反方文案
	ConText string `gorm:"size:256" json:"conText" form:"conText"`
	// 反方投票数
	ConVoteCount int64 `gorm:"not null;default:0" json:"conVoteCount" form:"conVoteCount"`
	// 热度（用于排序/榜单，可由定时任务或人工维护）
	Heat int64 `gorm:"not null;default:0" json:"heat" form:"heat"`
	// 预测事件详情
	Detail string `gorm:"type:text" json:"detail" form:"detail"`
	// 标签（先用逗号分隔，后续如需结构化可拆表）
	Tags string `gorm:"size:1024" json:"tags" form:"tags"`

	CreateTime int64 `gorm:"not null;default:0" json:"createTime" form:"createTime"`
	UpdateTime int64 `gorm:"not null;default:0" json:"updateTime" form:"updateTime"`
}

// PredictTag 预测市场标签（由 PredictContext.Tags 物化聚合得到）。
// 说明：项目仅使用 PostgreSQL，不依赖 Redis；标签表用于避免线上请求对 t_predict_context 做全表拆分聚合。
type PredictTag struct {
	Model

	Slug string `gorm:"size:128;not null;uniqueIndex" json:"slug" form:"slug"`
	// Name 可选：若没有独立中文名体系，可先等于 Slug。
	Name string `gorm:"size:256;not null" json:"name" form:"name"`
	// LastSeenAt 最近一次在任意 PredictContext.Tags 中出现的时间（秒级时间戳）。
	LastSeenAt int64 `gorm:"not null;default:0;index" json:"lastSeenAt" form:"lastSeenAt"`

	CreateTime int64 `gorm:"not null;default:0" json:"createTime" form:"createTime"`
	UpdateTime int64 `gorm:"not null;default:0;index" json:"updateTime" form:"updateTime"`
}

// PredictTagStat 标签统计（可选，但用于排序 marketCount 时避免在线聚合）。
type PredictTagStat struct {
	Model

	TagId int64 `gorm:"not null;uniqueIndex" json:"tagId" form:"tagId"`
	// MarketCount 出现该 tag 的 marketId 去重计数。
	MarketCount int64 `gorm:"not null;default:0;index" json:"marketCount" form:"marketCount"`
	// RefreshedAt 最近一次刷新时间（秒级时间戳）。
	RefreshedAt int64 `gorm:"not null;default:0;index" json:"refreshedAt" form:"refreshedAt"`

	CreateTime int64 `gorm:"not null;default:0" json:"createTime" form:"createTime"`
	UpdateTime int64 `gorm:"not null;default:0" json:"updateTime" form:"updateTime"`
}

// UserCoin 用户金币账户（未来下注使用；此阶段只建表不接业务）
type UserCoin struct {
	Model
	UserId  int64 `gorm:"not null;uniqueIndex" json:"userId" form:"userId"`
	Balance int64 `gorm:"not null;default:0" json:"balance" form:"balance"`
	// 预留：冻结金额等
	Frozen int64 `gorm:"not null;default:0" json:"frozen" form:"frozen"`

	CreateTime int64 `gorm:"not null;default:0" json:"createTime" form:"createTime"`
	UpdateTime int64 `gorm:"not null;default:0" json:"updateTime" form:"updateTime"`
}

// UserCoinLog 金币流水
// 说明：
// - 铸币（管理员）：bizType=MINT，amount>0
// - 下注扣减：bizType=BET，amount<0
type UserCoinLog struct {
	Model
	UserId int64 `gorm:"not null;index" json:"userId" form:"userId"`
	// 业务类型：MINT/BET/SETTLE/REFUND...
	BizType string `gorm:"size:32;not null;index" json:"bizType" form:"bizType"`
	// 业务关联（如下注单 id）
	BizId int64 `gorm:"not null;default:0;index" json:"bizId" form:"bizId"`
	// 变更金额：正数入账、负数出账
	Amount int64 `gorm:"not null" json:"amount" form:"amount"`
	// 变更后余额（便于审计）
	BalanceAfter int64  `gorm:"not null" json:"balanceAfter" form:"balanceAfter"`
	Remark       string `gorm:"size:256" json:"remark" form:"remark"`

	CreateTime int64 `gorm:"not null;default:0" json:"createTime" form:"createTime"`
}

// PredictBet 预测下注单
type PredictBet struct {
	Model
	UserId   int64 `gorm:"not null;index" json:"userId" form:"userId"`
	MarketId int64 `gorm:"not null;index" json:"marketId" form:"marketId"`
	// 下注选项：A/B
	Option string `gorm:"size:8;not null;index" json:"option" form:"option"`
	// 下注金额（金币）
	Amount int64 `gorm:"not null" json:"amount" form:"amount"`
	// 下单时锁定赔率（范围 1.2 ~ 5.0）
	Odds float64 `gorm:"not null" json:"odds" form:"odds"`
	// 下单时的有效池（用于审计/展示）
	EffA int64 `gorm:"not null" json:"effA" form:"effA"`
	EffB int64 `gorm:"not null" json:"effB" form:"effB"`

	// 结算状态：
	// - OPEN：未结算
	// - SETTLED：已结算（无论输赢，都已完成本单的最终入账）
	// - CANCELED：已取消（预留）
	Status string `gorm:"size:16;not null;default:'OPEN'" json:"status" form:"status"` // OPEN/SETTLED/CANCELED
	// 结算结果：WIN/LOSE/DRAW（DRAW 预留）
	SettleResult string `gorm:"size:16;not null;default:''" json:"settleResult" form:"settleResult"`
	// 派奖金额（金币）。输单为 0；赢单一般为 floor(amount * odds)
	Payout     int64 `gorm:"not null;default:0" json:"payout" form:"payout"`
	SettleTime int64 `gorm:"not null;default:0" json:"settleTime" form:"settleTime"`

	CreateTime int64 `gorm:"not null;default:0" json:"createTime" form:"createTime"`
}
