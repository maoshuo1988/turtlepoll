# 对立PK（PK）

路由前缀：/api/pk
认证：按需登录。中间件会尝试识别登录态，但不强制；部分接口要求已登录。

## 接口总览

| 方法 | 路径 | 登录要求 | 说明 |
|------|------|----------|------|
| GET | /api/pk/topics | 否 | 话题列表（登录后含个人态） |
| GET | /api/pk/topic | 否 | 话题详情 |
| POST | /api/pk/bet | 是 | 下注 |
| GET | /api/pk/heat | 否 | 当前热度 |
| GET | /api/pk/comments | 否 | 评论列表 |
| GET | /api/pk/comment/replies | 否 | 评论回复列表 |
| POST | /api/pk/comment/create | 是 | 发评论 |
| POST | /api/pk/comment/reply | 是 | 回复评论 |
| POST | /api/pk/like | 是 | 点赞评论 |
| POST | /api/pk/downvote | 是 | 拉踩评论 |
| POST | /api/pk/settle | 是 | 触发结算 |
| GET | /api/pk/heat/rank | 否 | 热度榜 |
| GET | /api/pk/heat/me | 是 | 我的热度数据 |
| GET | /api/pk/odds/current | 否 | 当前赔率 |
| POST | /api/pk/recordOption | 是 | 记录阵营行为 |
| GET | /api/pk/history | 否 | 历史回合 |
| GET | /api/pk/seasons | 否 | 赛季列表 |
| GET | /api/pk/my/bets | 是 | 我参与（下注）对局列表（支持按状态查询） |

## 关键接口

### 1) GET /api/pk/topics
请求参数：page, pageSize

请求参数 JSON 示例（Query）：
```json
{
	"page": 1,
	"pageSize": 20
}
```

返回 data 结构：
- list: 每项包含 topic, round, season, oddsA, oddsB, leader, streakStatus, countdownSeconds
- 登录用户额外返回 mySide, myBet, hasBet, canSettle, settleDisabledReason
- count, page, pageSize

返回 JSON 示例（data）：
```json
{
	"list": [
		{
			"topic": {
				"id": 1,
				"slug": "pk-hero",
				"title": "足球GOAT之争",
				"sideAName": "梅西",
				"sideBName": "C罗",
				"status": "enabled",
				"currentRoundId": 101,
				"currentSeasonId": 11
			},
			"round": {
				"id": 101,
				"topicId": 1,
				"phase": "betting",
				"heatA": 123.5,
				"heatB": 118.2,
				"poolA": 2300,
				"poolB": 2100,
				"endTime": 1782500000,
				"settledAt": 0
			},
			"season": {
				"id": 11,
				"topicId": 1,
				"seasonNo": 3,
				"status": "active"
			},
			"oddsA": 1.91,
			"oddsB": 2.04,
			"leader": "A",
			"streakStatus": "defending",
			"countdownSeconds": 3600,
			"mySide": "A",
			"myBet": {
				"id": 9001,
				"roundId": 101,
				"userId": 10001,
				"side": "A",
				"amount": 100,
				"requestId": "bet-uuid-1"
			},
			"hasBet": true,
			"canSettle": false,
			"settleDisabledReason": "ROUND_NOT_ENDED"
		}
	],
	"count": 1,
	"page": 1,
	"pageSize": 20
}
```

**请求字段说明：**
| 字段 | 说明 |
|------|------|
| page | 页码，从 1 开始，默认 1 |
| pageSize | 每页条数，默认 20 |

**返回字段说明（list 每项）：**
| 字段 | 说明 |
|------|------|
| topic.slug | 话题唯一标识，URL 友好 |
| topic.sideAName / sideBName | A/B 阵营名称 |
| topic.status | 话题状态：enabled=启用，disabled=停用 |
| topic.currentRoundId | 当前进行中的回合 ID |
| round.phase | 回合阶段：betting=下注期，locked=锁局期，cooldown=冷却期 |
| round.heatA / heatB | A/B 阵营热度，来自撕裂带热度快照 |
| round.poolA / poolB | A/B 阵营奖池，单位龟币 |
| round.endTime | 回合结束时间戳（秒级 Unix） |
| round.settledAt | 结算完成时间戳，0 表示未结算 |
| oddsA / oddsB | A/B 阵营动态赔率，基于奖池实时计算 |
| leader | 热度领先方：A / B / draw |
| streakStatus | 守擂翻盘状态：defending=守擂，challenging=翻盘中 |
| countdownSeconds | 距回合结束剩余秒数，已过期为 0 |
| mySide | 登录用户本局下注阵营，未下注为空字符串 |
| myBet | 登录用户本局下注记录，未下注为 null |
| hasBet | 当前用户是否已在本回合完成下注 |
| canSettle | 是否可触发结算：已登录 + now≥endTime + 未结算 |
| settleDisabledReason | 不可结算原因：NOT_LOGIN / ROUND_NOT_ENDED / ROUND_ALREADY_SETTLED / ROUND_NOT_READY |

### 2) GET /api/pk/topic
请求参数：topicId 或 slug（二选一）

请求参数 JSON 示例（Query）：
```json
{
	"topicId": 1,
	"slug": ""
}
```

返回 data 结构：
- 复用 topics 单项结构
- recentRounds: 最近已结算回合
- stats: totalRounds, winsA, winsB, currentStreakSide, currentStreak, maxStreakA, maxStreakB

返回 JSON 示例（data）：
```json
{
	"topic": {
		"id": 1,
		"slug": "pk-hero",
		"title": "足球GOAT之争",
		"sideAName": "梅西",
		"sideBName": "C罗"
	},
	"round": {
		"id": 101,
		"phase": "betting",
		"heatA": 123.5,
		"heatB": 118.2,
		"poolA": 2300,
		"poolB": 2100,
		"endTime": 1782500000,
		"settledAt": 0
	},
	"season": {
		"id": 11,
		"seasonNo": 3,
		"status": "active"
	},
	"oddsA": 1.91,
	"oddsB": 2.04,
	"leader": "A",
	"streakStatus": "defending",
	"countdownSeconds": 3600,
	"recentRounds": [
		{
			"id": 100,
			"roundNo": 10,
			"winner": "B",
			"settledAt": 1782000000
		}
	],
	"stats": {
		"totalRounds": 10,
		"winsA": 6,
		"winsB": 4,
		"currentStreakSide": "A",
		"currentStreak": 2,
		"maxStreakA": 3,
		"maxStreakB": 2
	}
}
```

**请求字段说明：**
| 字段 | 说明 |
|------|------|
| topicId | 话题 ID，与 slug 二选一传入 |
| slug | 话题唯一标识，与 topicId 二选一传入 |

**返回字段说明（额外字段）：**
| 字段 | 说明 |
|------|------|
| recentRounds | 最近已结算回合列表，含 winner / settledAt |
| stats.totalRounds | 该话题历史总回合数 |
| stats.winsA / winsB | A/B 阵营历史胜场数 |
| stats.currentStreakSide | 当前连胜方，A 或 B |
| stats.currentStreak | 当前连胜场数，0 表示暂无连胜 |
| stats.maxStreakA / maxStreakB | A/B 阵营历史最高连胜场数 |

### 3) POST /api/pk/bet
请求体：topicId, side(A/B), requestId, amount(可选)

请求 JSON 示例：
```json
{
	"topicId": 1,
	"side": "A",
	"requestId": "bet-uuid-1",
	"amount": 300
}
```

返回 data 结构：
- bet
- round
- userCoin
- oddsA, oddsB

返回 JSON 示例（data）：
```json
{
	"bet": {
		"id": 9001,
		"topicId": 1,
		"roundId": 101,
		"userId": 10001,
		"side": "A",
		"amount": 100,
		"requestId": "bet-uuid-1"
	},
	"round": {
		"id": 101,
		"poolA": 2400,
		"poolB": 2100,
		"betCountA": 25,
		"betCountB": 21
	},
	"userCoin": {
		"userId": 10001,
		"balance": 8800
	},
	"oddsA": 1.88,
	"oddsB": 2.08
}
```

**请求字段说明：**
| 字段 | 说明 |
|------|------|
| topicId | 话题 ID |
| side | 下注阵营：A 或 B |
| requestId | 客户端幂等键，相同 requestId 重复提交返回已有记录且不重复扣款 |
| amount | 下注金额（龟币，可选）；未传时默认 100，必须为正整数 |

**返回字段说明：**
| 字段 | 说明 |
|------|------|
| bet.amount | 实际下注金额（龟币）；未传 amount 时为默认值 100 |
| bet.settleResult | 结算结果，未结算时为空，结算后为 win / lose / draw |
| bet.payout | 派奖金额，未结算时为 0 |
| round.poolA / poolB | 下注后最新奖池 |
| round.betCountA / betCountB | 下注后各阵营最新下注人数 |
| userCoin.balance | 扣款后用户龟币余额 |
| oddsA / oddsB | 下注后最新动态赔率 |

常见错误：
- topicId is required
- invalid side
- requestId is required
- invalid amount
- pk topic not found
- pk round not found
- pk round is not betting
- already bet in this round

### 4) GET /api/pk/heat
请求参数：topicId

请求参数 JSON 示例（Query）：
```json
{
	"topicId": 1
}
```

返回 data 结构：
- roundId, phase, heatA, heatB
- options（来自热度快照）
- leader, streakStatus, countdownSeconds

返回 JSON 示例（data）：
```json
{
	"roundId": 101,
	"phase": "betting",
	"heatA": 140.5,
	"heatB": 132.2,
	"options": [
		{
			"option": "A",
			"hLike": 28,
			"hComment": 96.5,
			"hCoin": 16,
			"hTotal": 140.5,
			"snapshotType": "CHECKPOINT",
			"snapshotTime": 1782400000
		},
		{
			"option": "B",
			"hLike": 25,
			"hComment": 91.2,
			"hCoin": 16,
			"hTotal": 132.2,
			"snapshotType": "CHECKPOINT",
			"snapshotTime": 1782400000
		}
	],
	"leader": "A",
	"streakStatus": "defending",
	"countdownSeconds": 3200
}
```

**请求字段说明：**
| 字段 | 说明 |
|------|------|
| topicId | 话题 ID |

**返回字段说明：**
| 字段 | 说明 |
|------|------|
| phase | 当前阶段：betting=下注期，locked=锁局期，cooldown=冷却期 |
| heatA / heatB | A/B 阵营热度总值（来自快照 hTotal） |
| options[].option | 阵营标识：A 或 B |
| options[].hLike | 点赞贡献热度 |
| options[].hComment | 评论贡献热度 |
| options[].hCoin | 下注投币贡献热度（betAmount × 0.02） |
| options[].hTotal | 总热度 = hLike + hComment + hCoin |
| options[].snapshotType | 快照类型：CHECKPOINT=实时检查点，SETTLE=结算冻结快照 |
| options[].snapshotTime | 快照生成时间戳 |
| leader | 当前热度领先方：A / B / draw |
| streakStatus | 守擂翻盘状态：defending=守擂，challenging=翻盘中 |
| countdownSeconds | 距回合结束剩余秒数 |

### 5) POST /api/pk/settle
请求体：topicId 或 roundId，支持 requestId, snapshotType, freezeSource

请求 JSON 示例：
```json
{
	"topicId": 1,
	"roundId": 101,
	"requestId": "settle-uuid-1",
	"snapshotType": "SETTLE",
	"freezeSource": "ON_DEMAND"
}
```

返回 data 结构：
- topicId, roundId
- snapshotType, freezeSource
- winner, settledAt
- options
- settlement（当前用户结算项，未命中可为 null）

返回 JSON 示例（data）：
```json
{
	"topicId": 1,
	"roundId": 101,
	"snapshotType": "SETTLE",
	"freezeSource": "ON_DEMAND",
	"winner": "A",
	"settledAt": 1782500000,
	"options": [
		{
			"option": "A",
			"hLike": 30,
			"hComment": 98.5,
			"hCoin": 16,
			"hTotal": 144.5,
			"snapshotType": "SETTLE",
			"snapshotTime": 1782500000
		},
		{
			"option": "B",
			"hLike": 27,
			"hComment": 92.1,
			"hCoin": 16,
			"hTotal": 135.1,
			"snapshotType": "SETTLE",
			"snapshotTime": 1782500000
		}
	],
	"settlement": {
		"id": 7001,
		"roundId": 101,
		"userId": 10001,
		"result": "win",
		"stakeAmount": 100,
		"payoutAmount": 187,
		"paid": true
	}
}
```

**请求字段说明：**
| 字段 | 说明 |
|------|------|
| topicId | 话题 ID，与 roundId 二选一传入 |
| roundId | 回合 ID，与 topicId 二选一传入 |
| requestId | 幂等键（可选） |
| snapshotType | 快照类型，默认 SETTLE；传 CHECKPOINT 则只刷新检查点不冻结 |
| freezeSource | 冻结来源标识，默认 ON_DEMAND |

**返回字段说明：**
| 字段 | 说明 |
|------|------|
| winner | 胜方：A / B / draw（draw 表示热度完全相同的完全中立平局） |
| settledAt | 结算完成时间戳，触发前为 0 |
| options | 本次结算使用的热度冻结快照，格式同 /heat 接口 |
| settlement | 当前用户的结算明细，未参与本回合下注为 null |
| settlement.result | 结算结果：win=胜出，lose=落败，draw=平局 |
| settlement.stakeAmount | 本局下注本金（与用户下注金额一致） |
| settlement.payoutAmount | 实际获得派奖：win=本金+奖励，draw=原路退回，lose=0 |
| settlement.paid | 龟币是否已完成入账 |

注意：当前实现仅校验 now >= endTime 和 round.SettledAt，不含 settle_ready 字段判断。

### 6) GET /api/pk/heat/rank
请求参数：topicId/roundId, scope, page, pageSize

请求参数 JSON 示例（Query）：
```json
{
	"topicId": 1,
	"roundId": 101,
	"scope": "ALL",
	"page": 1,
	"pageSize": 20
}
```

返回 data 结构：
- topicId, roundId, phase, options, leaderSide
- list: rank, userId, username, nickname, avatar, option, totalHeat, heat, actionCount, firstActionTime
- count, page, pageSize

返回 JSON 示例（data）：
```json
{
	"topicId": 1,
	"roundId": 101,
	"phase": "betting",
	"options": [
		{"option": "A", "hTotal": 140.5},
		{"option": "B", "hTotal": 132.2}
	],
	"leaderSide": "A",
	"list": [
		{
			"rank": 1,
			"userId": 10001,
			"username": "turtle001",
			"nickname": "龟友A",
			"avatar": "https://example.com/a.png",
			"option": "A",
			"totalHeat": 38.5,
			"heat": 38.5,
			"actionCount": 12,
			"firstActionTime": 1782390000
		}
	],
	"count": 120,
	"page": 1,
	"pageSize": 20
}
```

**请求字段说明：**
| 字段 | 说明 |
|------|------|
| topicId / roundId | 话题或回合 ID，至少传一个 |
| scope | 榜单范围：ALL=全榜，MY_SIDE=仅当前用户所在阵营 |
| page / pageSize | 分页参数，page 默认 1，pageSize 默认 20，最大 100 |

**返回字段说明：**
| 字段 | 说明 |
|------|------|
| topicId / roundId | 目标话题与回合 ID |
| phase | 当前回合阶段 |
| options | 当前热度快照，格式同 /heat 接口 |
| leaderSide | 当前热度领先方：A / B |
| list[].userId | 用户 ID |
| list[].username / nickname | 用户名与昵称；当前实现两者保持一致 |
| list[].avatar | 用户头像 |
| list[].heat | 用户本回合热度贡献总值 |
| list[].rank | 排名，从 1 起，按总热度倒序排列 |
| list[].option | 该用户所在阵营 |
| list[].totalHeat | 用户本回合热度贡献总值（与 heat 相同） |
| list[].actionCount | 用户本回合互动总次数（评论+回复+点赞+下注） |
| list[].firstActionTime | 用户首次互动时间戳 |
| count | 总条数 |
| page / pageSize | 当前页与分页大小 |

### 7) GET /api/pk/heat/me
请求参数：topicId/roundId

请求参数 JSON 示例（Query）：
```json
{
	"topicId": 1,
	"roundId": 101
}
```

返回 data 结构：
- topicId, roundId, phase, options
- myOption, receivedLikeCount, myCommentCount, myBetAmount, estimatedPayout
- myHeat, myActionCount, myRank, mySideHeat
- myBet

返回 JSON 示例（data）：
```json
{
	"topicId": 1,
	"roundId": 101,
	"phase": "betting",
	"options": [
		{"option": "A", "hTotal": 140.5},
		{"option": "B", "hTotal": 132.2}
	],
	"myOption": "A",
	"receivedLikeCount": 18,
	"myCommentCount": 6,
	"myBetAmount": 100,
	"estimatedPayout": 187,
	"myHeat": 38.5,
	"myActionCount": 12,
	"myRank": 1,
	"mySideHeat": {
		"A": {
			"heat": 38.5,
			"count": 12
		}
	},
	"myBet": {
		"id": 9001,
		"roundId": 101,
		"side": "A",
		"amount": 100
	}
}
```

**请求字段说明：**
| 字段 | 说明 |
|------|------|
| topicId / roundId | 话题或回合 ID，至少传一个 |

**返回字段说明：**
| 字段 | 说明 |
|------|------|
| myOption | 我当前所在阵营（A/B），未参与为空字符串 |
| receivedLikeCount | 我在本回合评论/回复收到的点赞总数 |
| myCommentCount | 我在本回合的评论与回复总条数 |
| myBetAmount | 我的下注金额，未下注为 0 |
| estimatedPayout | 基于当前热度与奖池估算的预计派奖，结算前为动态估算值 |
| myHeat | 我的热度贡献总值 |
| myActionCount | 我的互动总次数 |
| myRank | 我在全榜中的名次，热度为 0 时返回 0 |
| mySideHeat | 我所在阵营的热度分项（heat=热度，count=互动次数） |
| myBet | 我本局下注记录，未下注为 null |

### 8) GET /api/pk/odds/current
请求参数：topicId/roundId

请求参数 JSON 示例（Query）：
```json
{
	"topicId": 1,
	"roundId": 101
}
```

返回 data 结构：
- topicId, roundId, phase
- options
- oddsA, oddsB

返回 JSON 示例（data）：
```json
{
	"topicId": 1,
	"roundId": 101,
	"phase": "betting",
	"options": [
		{"option": "A", "hTotal": 140.5},
		{"option": "B", "hTotal": 132.2}
	],
	"oddsA": 1.88,
	"oddsB": 2.08
}
```

**请求字段说明：**
| 字段 | 说明 |
|------|------|
| topicId / roundId | 话题或回合 ID，至少传一个 |

**返回字段说明：**
| 字段 | 说明 |
|------|------|
| options | 各阵营当前热度快照（简化版，含 hTotal） |
| oddsA / oddsB | A/B 阵营动态赔率，每次下注后实时更新 |

### 9) POST /api/pk/recordOption
请求体：topicId/roundId, option(A/B), actionType, requestId, entityType, entityId

请求 JSON 示例：
```json
{
	"topicId": 1,
	"roundId": 101,
	"option": "A",
	"actionType": "view",
	"requestId": "record-uuid-1",
	"entityType": "pk_option",
	"entityId": 101
}
```

返回 data 结构：
- recorded, optionAtAction, actionType, requestId
- action

返回 JSON 示例（data）：
```json
{
	"recorded": true,
	"optionAtAction": "A",
	"actionType": "view",
	"requestId": "record-uuid-1",
	"action": {
		"id": 8801,
		"topicId": 1,
		"roundId": 101,
		"userId": 10001,
		"side": "A",
		"actionType": "view",
		"entityType": "pk_option",
		"entityId": 101,
		"amount": 1,
		"heat": 1
	}
}
```

**请求字段说明：**
| 字段 | 说明 |
|------|------|
| topicId / roundId | 话题或回合 ID，至少传一个 |
| option | 用户主动选择的阵营：A 或 B |
| actionType | 行为类型，由调用方定义（如 view / share 等） |
| requestId | 幂等键，相同 requestId 重复提交幂等返回 |
| entityType | 关联实体类型，默认 pk_option |
| entityId | 关联实体 ID，默认为 roundId |

**返回字段说明：**
| 字段 | 说明 |
|------|------|
| recorded | 是否成功写入：true=新写入，幂等命中也返回 true |
| optionAtAction | 本次记录的阵营归属 |
| action | 写入的 PKAction 记录（含 heat=贡献热度审计值） |

### 9.1) POST /api/pk/comment/reply
请求体：commentId, content

请求 JSON 示例：
```json
{
	"commentId": 6001,
	"content": "不同意，我看好B方后程发力"
}
```

返回 data 结构：
- comment
- heat
- optionAtAction

返回 JSON 示例（data）：
```json
{
	"comment": {
		"id": 6010,
		"userId": 10002,
		"entityType": "comment",
		"entityId": 6001,
		"content": "不同意，我看好B方后程发力",
		"optionAtAction": "B",
		"createTime": 1782390300
	},
	"heat": {
		"heatA": 141.2,
		"heatB": 136.1
	},
	"optionAtAction": "B"
}
```

**请求字段说明：**
| 字段 | 说明 |
|------|------|
| commentId | 父评论 ID |
| content | 回复内容 |

**返回字段说明：**
| 字段 | 说明 |
|------|------|
| optionAtAction | 回复写入时用户所属阵营快照（A/B） |
| comment.optionAtAction | 同上，落在通用评论对象中的写入时阵营字段 |

说明：回复写入时会固化阵营快照，后续在 `GET /api/comment/replies` 等评论查询里可读取该快照。

### 10) GET /api/pk/comments
请求参数：topicId, side(A/B), cursor, sort(time/heat)

请求参数 JSON 示例（Query）：
```json
{
	"topicId": 1,
	"side": "A",
	"cursor": 0,
	"sort": "time"
}
```

返回：游标分页结构（web.JsonCursorData）
- data 列表项：comment, option, side, heatScore, downvoteCount, liked
- cursor, hasMore

返回 JSON 示例（data）：
```json
{
	"data": [
		{
			"comment": {
				"id": 6001,
				"userId": 10001,
				"content": "我看好A方",
				"likeCount": 12,
				"createTime": 1782390000
			},
			"option": "A",
			"side": "A",
			"heatScore": 8.76,
			"downvoteCount": 1,
			"liked": true
		}
	],
	"cursor": 6001,
	"hasMore": true
}
```

**请求字段说明：**
| 字段 | 说明 |
|------|------|
| topicId | 话题 ID |
| side | 阵营过滤（必填）：A 或 B |
| cursor | 游标值：首次请求传 0，续页传上次返回的 cursor |
| sort | 排序：time=按发布时间倒序，heat=按热度分倒序 |

**返回字段说明：**
| 字段 | 说明 |
|------|------|
| comment | 通用评论对象（含 id / userId / content / likeCount 等） |
| comment.optionAtAction | 评论/回复写入时的用户阵营快照（A/B）；在通用评论查询中可见 |
| option / side | 评论归属阵营，写入时确定，与查询者身份无关 |
| heatScore | 评论热度分，基于收到点赞数 log 公式计算 |
| downvoteCount | 评论被拉踩次数 |
| liked | 当前登录用户是否已点赞该评论，未登录时固定 false |
| cursor | 下页游标，续页时原样传回 |
| hasMore | 是否还有更多数据 |

补充说明（回复阵营记录）：
- `POST /api/pk/comment/reply` 的响应会返回 `optionAtAction`，表示本次回复写入时用户所属阵营。
- 回复会把该阵营快照落库，后续在 `GET /api/comment/replies` 与通用评论回包中可通过 `comment.optionAtAction` 获取。

### 10.1) GET /api/pk/comment/replies
请求参数：commentId, cursor, pageSize

请求参数 JSON 示例（Query）：
```json
{
	"commentId": 6001,
	"cursor": 0,
	"pageSize": 20
}
```

返回：游标分页结构（web.JsonCursorData）
- data 列表项：comment, option, side, heatScore, downvoteCount, liked
- cursor, hasMore

返回 JSON 示例（data）：
```json
{
	"data": [
		{
			"comment": {
				"id": 6010,
				"userId": 10002,
				"entityType": "comment",
				"entityId": 6001,
				"content": "不同意，我看好B方后程发力",
				"optionAtAction": "B",
				"createTime": 1782390300
			},
			"option": "B",
			"side": "B",
			"heatScore": 4.22,
			"downvoteCount": 0,
			"liked": false
		}
	],
	"cursor": 6010,
	"hasMore": true
}
```

**请求字段说明：**
| 字段 | 说明 |
|------|------|
| commentId | 父评论 ID |
| cursor | 游标值：首次请求传 0，续页传上次返回的 cursor |
| pageSize | 每页条数，默认 20，最大 100 |

**返回字段说明：**
| 字段 | 说明 |
|------|------|
| comment | 回复评论对象 |
| comment.optionAtAction | 回复写入时的用户阵营快照 |
| option / side | 回复归属阵营，等价于写入时阵营快照 |
| heatScore | 回复热度分 |
| downvoteCount | 回复被拉踩次数 |
| liked | 当前登录用户是否已点赞该回复，未登录时固定 false |
| cursor | 下页游标 |
| hasMore | 是否还有更多数据 |

### 11) POST /api/pk/like
请求参数：commentId, requestId

请求 JSON 示例：
```json
{
	"commentId": 6001,
	"requestId": "like-uuid-1"
}
```

返回 data 结构：
- liked
- optionAtAction
- heat

返回 JSON 示例（data）：
```json
{
	"liked": true,
	"optionAtAction": "A",
	"heat": {
		"round": {
			"id": 101,
			"heatA": 141.5,
			"heatB": 132.2
		},
		"heatA": 141.5,
		"heatB": 132.2
	}
}
```

**请求字段说明：**
| 字段 | 说明 |
|------|------|
| commentId | 要点赞的评论 ID |
| requestId | 幂等键 |

**返回字段说明：**
| 字段 | 说明 |
|------|------|
| liked | 固定 true（点赞成功） |
| optionAtAction | 被点赞评论所属阵营，本次点赞热度贡献归入该阵营 |
| heat.heatA / heatB | 点赞后回合 A/B 阵营最新热度 |

### 12) POST /api/pk/downvote
请求体：commentId, requestId

请求 JSON 示例：
```json
{
	"commentId": 6002,
	"requestId": "downvote-uuid-1"
}
```

返回 data 结构：
- RecalcRoundHeat 的结果（round, heatA, heatB）

返回 JSON 示例（data）：
```json
{
	"round": {
		"id": 101,
		"heatA": 140.5,
		"heatB": 131.7
	},
	"heatA": 140.5,
	"heatB": 131.7
}
```

**请求字段说明：**
| 字段 | 说明 |
|------|------|
| commentId | 要拉踩的评论 ID |
| requestId | 幂等键 |

**返回字段说明：**
| 字段 | 说明 |
|------|------|
| round | 拉踩后重算热度的回合信息 |
| heatA / heatB | 拉踩后 A/B 阵营最新热度 |

### 13) GET /api/pk/history
请求参数：topicId, page, pageSize

请求参数 JSON 示例（Query）：
```json
{
	"topicId": 1,
	"page": 1,
	"pageSize": 20
}
```

返回 data 结构：
- list（已结算 round）
- count, page, pageSize

返回 JSON 示例（data）：
```json
{
	"list": [
		{
			"id": 100,
			"topicId": 1,
			"roundNo": 10,
			"winner": "B",
			"heatA": 122.1,
			"heatB": 130.4,
			"settledAt": 1782000000
		}
	],
	"count": 10,
	"page": 1,
	"pageSize": 20
}
```

**请求字段说明：**
| 字段 | 说明 |
|------|------|
| topicId | 话题 ID |
| page / pageSize | 分页参数 |

**返回字段说明：**
| 字段 | 说明 |
|------|------|
| list[].roundNo | 回合序号，从 1 开始递增 |
| list[].winner | 胜方：A / B / draw |
| list[].heatA / heatB | 该回合结算时的热度值（来自冻结快照） |
| list[].settledAt | 结算时间戳 |

### 14) GET /api/pk/seasons
请求参数：topicId, page, pageSize

请求参数 JSON 示例（Query）：
```json
{
	"topicId": 1,
	"page": 1,
	"pageSize": 20
}
```

返回 data 结构：
- list（season）
- count, page, pageSize

返回 JSON 示例（data）：
```json
{
	"list": [
		{
			"id": 11,
			"topicId": 1,
			"seasonNo": 3,
			"status": "active",
			"totalRounds": 10,
			"winsA": 6,
			"winsB": 4
		}
	],
	"count": 3,
	"page": 1,
	"pageSize": 20
}
```
**请求字段说明：**
| 字段 | 说明 |
|------|------|
| topicId | 话题 ID |
| page / pageSize | 分页参数 |

**返回字段说明：**
| 字段 | 说明 |
|------|------|
| list[].seasonNo | 赛季序号，从 1 开始递增 |
| list[].status | 赛季状态：active=进行中，finished=已结束 |
| list[].totalRounds | 本赛季总回合数 |
| list[].winsA / winsB | 本赛季 A/B 阵营胜场数 |
### 15) GET /api/pk/my/bets
请求参数：page, pageSize, status

请求参数 JSON 示例（Query）：
```json
{
	"page": 1,
	"pageSize": 20,
	"status": "pending"
}
```

返回 data 结构：
- list: 每项顶层结构对齐 `/api/battle/by`：`battle + myAction + myRole + settlement`
- 兼容字段：每项继续保留 `bet/topic/round`
- count, page, pageSize, status

返回 JSON 示例（data）：
```json
{
	"list": [
		{
			"battle": {
				"id": 101,
				"topicId": 1,
				"status": "pending",
				"phase": "cooldown",
				"result": "",
				"resultBy": "system",
				"resultTime": 0,
				"bet": {
					"id": 9001,
					"topicId": 1,
					"roundId": 101,
					"side": "A",
					"amount": 100,
					"settleResult": "",
					"payout": 0
				},
				"topic": {
					"id": 1,
					"slug": "pk-hero",
					"title": "足球GOAT之争",
					"sideAName": "梅西",
					"sideBName": "C罗"
				},
				"round": {
					"id": 101,
					"roundNo": 11,
					"phase": "cooldown"
				}
			},
			"myAction": "bet",
			"myRole": "challenger",
			"settlement": {
				"settlement": null,
				"myItem": null
			},
			"bet": {"id": 9001, "topicId": 1, "roundId": 101, "side": "A", "amount": 100},
			"topic": {"id": 1, "slug": "pk-hero", "title": "足球GOAT之争"},
			"round": {"id": 101, "roundNo": 11, "phase": "cooldown"}
		}
	],
	"count": 12,
	"page": 1,
	"pageSize": 20,
	"status": "pending"
}
```

**请求字段说明：**
| 字段 | 说明 |
|------|------|
| page / pageSize | 分页参数 |
| status | 可选：`in_progress/pending/settled`；也兼容 `betting/locked/cooldown/settled` |

**返回字段说明：**
| 字段 | 说明 |
|------|------|
| battle.status | 对局状态：`in_progress/pending/settled` |
| battle.phase | 回合阶段：`betting/locked/cooldown/settled` |
| battle.result | 结算结果（A/B/draw），未结算为空 |
| myAction | 当前用户动作（固定 `bet`） |
| myRole | 当前用户角色（固定 `challenger`） |
| settlement.settlement | 已结算时返回结算摘要，未结算为 null |
| settlement.myItem | 当前用户结算明细，存在则返回 |
| bet/topic/round | 兼容旧调用方的历史字段 |
