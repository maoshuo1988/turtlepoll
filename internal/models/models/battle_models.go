package models

// Battle Square

// Battle 赌局（开战广场）
// 说明：
// - 资金托管：下注本金进资金池（userId=-1）；入场费实时转给庄家；罚没转销毁账户（userId=-2）。
// - 结算：settled 时生成结算单；提取时按结算单出池并幂等。
type Battle struct {
	Model

	Title          string `gorm:"size:256;not null" json:"title"`
	BankerUserId   int64  `gorm:"not null;index" json:"bankerUserId"`
	BankerSide     string `gorm:"size:1024;not null" json:"bankerSide"`
	ChallengerSide string `gorm:"size:1024;not null" json:"challengerSide"`

	// 是否公开（公开：收取入场费；私人：邀请码加入且不收取入场费）
	IsPublic   bool   `gorm:"not null;default:true;index" json:"isPublic"`
	InviteCode string `gorm:"size:64;index" json:"inviteCode"`

	// 状态：open/sealed/pending/disputed/settled
	Status string `gorm:"size:16;not null;index" json:"status"`

	// 结算时间（到达后自动 sealed -> pending）
	SettleTime int64 `gorm:"not null;index" json:"settleTime"`
	// pending 状态下庄家宣布截止时间（now+24h）
	PendingDeadline int64 `gorm:"not null;default:0;index" json:"pendingDeadline"`
	// pending 状态下挑战者确认截止时间（庄家宣布后 now+24h；超时默认确认）
	ConfirmDeadline int64 `gorm:"not null;default:0;index" json:"confirmDeadline"`
	// 挑战者已确认人数（仅用于快速查询；以 action 表为准）
	ConfirmedCount int64 `gorm:"not null;default:0" json:"confirmedCount"`
	// disputed 状态下管理员裁决截止时间（now+24h）
	DisputeDeadline int64 `gorm:"not null;default:0;index" json:"disputeDeadline"`
	// 发起争议的挑战者（disputed 时记录；可用于后台展示/定位）
	DisputedByUserId int64 `gorm:"not null;default:0;index" json:"disputedByUserId"`

	// 结果：banker_wins/banker_loses/void（空表示未产生最终裁定）
	Result string `gorm:"size:16;not null;default:'';index" json:"result"`
	// 结果来源：banker/timeout/challenger/admin/system
	ResultBy   string `gorm:"size:16;not null;default:''" json:"resultBy"`
	ResultTime int64  `gorm:"not null;default:0" json:"resultTime"`

	// 冗余汇总字段（便于查询）
	BankerStakeTotal     int64 `gorm:"not null;default:0" json:"bankerStakeTotal"`
	ChallengerStakeTotal int64 `gorm:"not null;default:0" json:"challengerStakeTotal"`
	PoolPrincipalTotal   int64 `gorm:"not null;default:0" json:"poolPrincipalTotal"`
	EntryFeeTotal        int64 `gorm:"not null;default:0" json:"entryFeeTotal"`
	BurnTotal            int64 `gorm:"not null;default:0" json:"burnTotal"`

	Heat int64 `gorm:"not null;default:0;index" json:"heat"`

	CreateTime int64 `gorm:"not null;default:0" json:"createTime"`
	UpdateTime int64 `gorm:"not null;default:0" json:"updateTime"`
}

// BattleChallengeAction 挑战者确认/异议动作（幂等）
// 说明：
// - 每个挑战者对同一 battle 最多产生一次 confirm 或 dispute（confirm/dispute 二选一）。
// - requestId 做幂等：重复提交不会重复变更。
type BattleChallengeAction struct {
	Model
	BattleId   int64  `gorm:"not null;index;uniqueIndex:idx_battle_challenge_user" json:"battleId"`
	UserId     int64  `gorm:"not null;index;uniqueIndex:idx_battle_challenge_user" json:"userId"`
	Action     string `gorm:"size:16;not null;index" json:"action"` // confirm/dispute
	RequestId  string `gorm:"size:64;not null;uniqueIndex:idx_battle_challenge_user" json:"requestId"`
	Remark     string `gorm:"size:256" json:"remark"`
	CreateTime int64  `gorm:"not null;default:0" json:"createTime"`
}

// BattleBet 下注明细（庄家加注/挑战者加入/挑战者追加下注都会产生一条记录）
// 注意：结算按用户维度聚合（同一 userId 多条 bet 需要 sum）。
type BattleBet struct {
	Model
	BattleId  int64  `gorm:"not null;index;uniqueIndex:idx_battle_bet_user_req" json:"battleId"`
	UserId    int64  `gorm:"not null;index;uniqueIndex:idx_battle_bet_user_req" json:"userId"`
	Role      string `gorm:"size:16;not null;index" json:"role"`                                    // banker/challenger
	Amount    int64  `gorm:"not null" json:"amount"`                                                // 下注本金（进资金池）
	EntryFee  int64  `gorm:"not null;default:0" json:"entryFee"`                                    // 入场费（不进资金池）
	RequestId string `gorm:"size:64;not null;uniqueIndex:idx_battle_bet_user_req" json:"requestId"` // 幂等键（由客户端传入）

	CreateTime int64 `gorm:"not null;default:0" json:"createTime"`
	UpdateTime int64 `gorm:"not null;default:0" json:"updateTime"`
}

// BattleLedger 业务流水（与 UserCoinLog 配合用于对账与审计）
// 说明：action + requestId + (battleId,userId) 应保证幂等。
type BattleLedger struct {
	Model
	BattleId   int64  `gorm:"not null;index;uniqueIndex:idx_battle_ledger_idem" json:"battleId"`
	UserId     int64  `gorm:"not null;index;uniqueIndex:idx_battle_ledger_idem" json:"userId"`
	Action     string `gorm:"size:32;not null;index;uniqueIndex:idx_battle_ledger_idem" json:"action"` // stake_in/entry_fee/burn/payout...
	RequestId  string `gorm:"size:64;not null;index;uniqueIndex:idx_battle_ledger_idem" json:"requestId"`
	Amount     int64  `gorm:"not null" json:"amount"` // 正数表示用户入账，负数表示用户出账
	Remark     string `gorm:"size:256" json:"remark"`
	CreateTime int64  `gorm:"not null;default:0" json:"createTime"`
}

// BattleSettlement 结算单（battle 进入 settled 时生成；提取时使用）
type BattleSettlement struct {
	Model
	BattleId  int64  `gorm:"not null;uniqueIndex" json:"battleId"`
	Result    string `gorm:"size:16;not null" json:"result"` // banker_wins/banker_loses/void
	CreatedAt int64  `gorm:"not null;default:0" json:"createdAt"`
}

// BattleSettlementItem 结算单明细（每个用户一条）
type BattleSettlementItem struct {
	Model
	SettlementId int64 `gorm:"not null;index" json:"settlementId"`
	BattleId     int64 `gorm:"not null;index;uniqueIndex:idx_battle_settle_user" json:"battleId"`
	UserId       int64 `gorm:"not null;index;uniqueIndex:idx_battle_settle_user" json:"userId"`
	// 应得金额（从资金池出池）；注意：入场费不在此体现（入场费已实时给庄家）。
	PayoutAmount int64 `gorm:"not null;default:0" json:"payoutAmount"`
	// 是否已提取（一次性全提）
	Withdrawn         bool   `gorm:"not null;default:false;index" json:"withdrawn"`
	WithdrawRequestId string `gorm:"size:64;not null;default:''" json:"withdrawRequestId"`
	WithdrawTime      int64  `gorm:"not null;default:0" json:"withdrawTime"`

	CreateTime int64 `gorm:"not null;default:0" json:"createTime"`
	UpdateTime int64 `gorm:"not null;default:0" json:"updateTime"`
}
