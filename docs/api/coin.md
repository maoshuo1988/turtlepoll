# 金币与预测下注（Coin / PredictBet）

## 功能

本模块提供“金币账户（余额）+ 金币流水 + 预测市场下注（锁赔率）”相关能力：

- 用户接口：`/api/coin/**`（需要登录，`AuthMiddleware`）
- 管理员接口：`/api/admin/coin/**`（需要管理员权限，`AdminMiddleware`）
- 账户余额榜：`GET /api/coin/leaderboard`，返回余额 Top N 用户和当前用户排名摘要

代码位置：

- 用户控制器：`internal/controllers/api/coin_controller.go`
- 管理员控制器：`internal/controllers/admin/coin_controller.go`
- 下注服务：`internal/services/predict_bet_service.go`
- 金币服务：`internal/services/user_coin_service.go`
- 排行榜服务：`internal/services/coin_leaderboard_service.go`
- 用户预测战绩服务：`internal/services/predict_user_stat_service.go`
- 预测评论奖励服务：`internal/services/predict_comment_reward_service.go`

## 数据模型

### UserCoin（用户金币账户）
代码定义：`internal/models/models.go` -> `type UserCoin`

常用字段：
- `id`: int64
- `userId`: int64
- `balance`: int64（当前金币余额）
- `createTime` / `updateTime`: int64

### UserCoinLog（金币流水）
代码定义：`internal/models/models.go` -> `type UserCoinLog`

常用字段：
- `id`: int64
- `userId`: int64
- `bizType`: string（例如：`MINT`、`BET`）
- `bizId`: int64（业务主键，`MINT` 时记录操作者 adminUserId；`BET` 时记录 betId）
- `amount`: int64（变动金额；下注为负数）
- `balanceAfter`: int64（变动后的余额）
- `remark`: string
- `createTime`: int64

### PredictMarket（预测市场池字段）
为了支持下注赔率，本项目在 `PredictMarket` 上维护虚拟底池与真实下注池累计：

- `baseA` / `baseB`: int64（二元与三元市场的 A/B 虚拟底池，默认 `500/500`）
- `baseDraw`: int64（三元市场 DRAW 虚拟底池，默认 `500`）
- `poolA` / `poolB`: int64（二元与三元市场的 A/B 用户下注累计池）
- `poolDraw`: int64（三元市场 DRAW 用户下注累计池）

赔率使用“有效池 = base + pool”计算。淘汰赛 `marketType=binary` 使用 A/B 二元池；小组赛 `marketType=1x2` 使用 A/B/DRAW 三元池。

### PredictBet（预测下注订单）
代码定义：`internal/models/models.go` -> `type PredictBet`

常用字段：
- `id`: int64
- `userId`: int64
- `marketId`: int64
- `option`: string（淘汰赛 `A/B`；小组赛 `A/B/DRAW`）
- `amount`: int64（下注金币）
- `odds`: float64（下单时锁定赔率；结算时不重算）
- `effA` / `effB` / `effDraw`: int64（下单时看到的有效池快照）
- `status`: string（当前实现使用 `OPEN`）
- `createTime`: int64

### PredictUserStat（用户预测战绩）
代码定义：`internal/models/models.go` -> `type PredictUserStat`

常用字段：
- `userId`: int64
- `settledMarketCount`: int64（计入胜率的已结算市场数）
- `winMarketCount`: int64（命中市场数）
- `loseMarketCount`: int64（未命中市场数）
- `winRate`: float64（胜率）
- `currentWinStreak`: int64（最新连胜场数）
- `bestWinStreak`: int64（历史最高连胜，预留展示）

### PredictUserMarketStat（用户单市场战绩）
代码定义：`internal/models/models.go` -> `type PredictUserMarketStat`

常用字段：
- `userId`: int64
- `marketId`: int64
- `result`: string（`WIN/LOSE/VOID`）
- `betAmount`: int64（该用户在该市场总下注额）
- `payout`: int64（该用户在该市场总派奖额）
- `settledBetCount`: int64（归并下注单数量）
- `settleTime`: int64

约束：
- 同一 `userId + marketId` 只记录一次，避免重复结算重复增加胜率或连胜。

### PredictCommentMeta（预测市场评论选项）
代码定义：`internal/models/models.go` -> `type PredictCommentMeta`

常用字段：
- `commentId`: int64
- `marketId`: int64
- `userId`: int64
- `option`: string（`A/B/DRAW`）

### PredictCommentRewardLog / PredictCommentRewardItem（评论奖励审计）
代码定义：`internal/models/models.go`

奖励规则：
- 市场结算后 1 小时内发放。
- 奖励池 = `floor((poolA + poolB + poolDraw) * 10%)`。
- 只奖励获胜方评论用户。
- 同一用户同一市场多条获胜方评论只发一份。
- 金币流水 `bizType = COMMENT_REWARD`。

## 赔率说明

下注时锁赔率（非常重要）：

- 下单时基于当前池计算 odds，并写入 `PredictBet.odds`
- 后续池变化不会影响该订单已经锁定的赔率

项目内实现：`internal/services/predict_odds.go`

- 赔率范围 clamp：`[1.2, 5.0]`
- 展示保留两位小数（代码中做了四舍五入）

二元市场算法：

```text
effA = baseA + poolA
effB = baseB + poolB
total = effA + effB

oddsA = clamp(total / effA, 1.2, 5.0)
oddsB = clamp(total / effB, 1.2, 5.0)
```

三元市场算法：

```text
effA = baseA + poolA
effB = baseB + poolB
effDraw = baseDraw + poolDraw
total = effA + effB + effDraw

oddsA = clamp(total / effA, 1.2, 5.0)
oddsB = clamp(total / effB, 1.2, 5.0)
oddsDraw = clamp(total / effDraw, 1.2, 5.0)
```

## 接口列表

### 0) 管理员查询金币流水（分页）

- **接口**：`GET /api/admin/coin/log/list`
- **认证**：需要管理员权限
- **参数（query）**：
  - `userId`: int64（可选，按用户筛选）
  - `bizType`: string（可选，按流水类型筛选，如 `MINT/BET/SETTLE/REFUND`）
  - `startDate`: string（可选，格式 `YYYY-MM-DD`，按创建时间起始日筛选，包含当天 00:00:00）
  - `endDate`: string（可选，格式 `YYYY-MM-DD`，按创建时间结束日筛选，包含当天整日）
  - `page`: int（可选，默认 1）
  - `limit`: int（可选，默认 20）

> 说明：
>
> - 该接口由 `internal/controllers/admin/coin_controller.go` 中 `AnyLogList()` 实现。
> - 日期筛选基于 `UserCoinLog.createTime`（秒级时间戳）。
> - `endDate` 实现为“小于次日 00:00:00”，因此传 `2026-04-27` 会包含 `2026-04-27` 全天数据。

#### 返回值（data）

分页结果 `web.PageResult`

- `results`: `UserCoinLog[]`
- `page`: 分页信息

示例：

```json
{
  "results": [
    {
      "id": 12,
      "userId": 100,
      "bizType": "BET",
      "bizId": 23,
      "amount": -100,
      "balanceAfter": 9900,
      "remark": "predict bet marketId=8 option=A",
      "createTime": 1777219200
    },
    {
      "id": 13,
      "userId": 100,
      "bizType": "SETTLE",
      "bizId": 23,
      "amount": 183,
      "balanceAfter": 10083,
      "remark": "predict settle marketId=8 result=A",
      "createTime": 1777305600
    }
  ],
  "page": {
    "page": 1,
    "limit": 20,
    "total": 2
  }
}
```

#### 可能错误

- `invalid startDate, expected YYYY-MM-DD`
- `invalid endDate, expected YYYY-MM-DD`

### 3) 结算（用户对自己下注过的预测市场进行结算，领取金币）

- **接口**：`POST /api/coin/settle`
- **认证**：需要登录
- **请求格式**：表单（`application/x-www-form-urlencoded` 或 `multipart/form-data`）

#### 请求参数（form）
- `marketId`: int64，必填

（文档用 JSON 展示字段结构；实际是表单）

```json
{
  "marketId": 1
}
```

#### 结算规则（当前实现）
- 仅允许结算 `PredictMarket.status = SETTLED` 的市场
- `PredictMarket.result` 按市场类型校验：
  - `binary`：只允许 `A/B`
  - `1x2`：允许 `A/B/DRAW`
- 只结算该用户在该市场中 `PredictBet.status = OPEN` 的订单（幂等：重复调用不会重复派奖）
- 中奖判断：`PredictBet.option == PredictMarket.result`
- 中奖派发：`payout = floor(amount * odds)`（odds 为下注时锁定赔率）
- 输单：`payout = 0`
- 用户战绩按“用户 + 市场”统计，同一用户同一市场多笔下注只计为一场
- 该市场用户结果为 `WIN` 时，用户胜场 +1、最新连胜 +1
- 该市场用户结果为 `LOSE` 时，用户负场 +1、最新连胜归零

小组赛平局结算：

```text
marketType = 1x2
PredictMarket.result = DRAW

option = DRAW -> WIN, payout = floor(amount * odds)
option = A/B  -> LOSE, payout = 0
```

#### 返回值（data）

返回结构：

- `list`: `SettleMyBetResult[]`
  - `bet`: PredictBet（更新为 `status=SETTLED`，并补充 `settleResult/payout/settleTime`）
  - `payout`: int64（本单派奖金币，输单为 0）
  - `userCoin`: UserCoin（派奖后的余额快照）
- `count`: int（list 数量）

示例：

```json
{
  "list": [
    {
      "bet": {
        "id": 10,
        "userId": 100,
        "marketId": 1,
        "option": "A",
        "amount": 100,
        "odds": 1.83,
        "effA": 500,
        "effB": 500,
        "status": "SETTLED",
        "settleResult": "WIN",
        "payout": 183,
        "settleTime": 1734019999,
        "createTime": 1734012345
      },
      "payout": 183,
      "userCoin": {
        "id": 1,
        "userId": 100,
        "balance": 12428,
        "createTime": 1734010000,
        "updateTime": 1734019999
      }
    },
    {
      "bet": {
        "id": 11,
        "userId": 100,
        "marketId": 1,
        "option": "B",
        "amount": 200,
        "odds": 2.2,
        "effA": 500,
        "effB": 500,
        "status": "SETTLED",
        "settleResult": "LOSE",
        "payout": 0,
        "settleTime": 1734019999,
        "createTime": 1734012500
      },
      "payout": 0,
      "userCoin": {
        "id": 1,
        "userId": 100,
        "balance": 12428,
        "createTime": 1734010000,
        "updateTime": 1734019999
      }
    }
  ],
  "count": 2
}
```

#### 可能错误
- 未登录：`errs.NotLogin()`
- 参数校验：
  - `marketId is required`
- 业务错误：
  - `market is not settled`
  - `market result must match market options`

---

### 3.1) 账户余额排行榜

- **接口**：`GET /api/coin/leaderboard`
- **认证**：需要登录

#### 请求参数（query）
- `limit`: int，可选，默认 20，最大 100

#### 排序规则

```text
balance desc
userId asc
```

#### 返回值（data）

- `items`: 账户余额 Top N 用户列表
- `myRank`: 当前用户排名；当前用户没有金币账户时为空
- `myBalance`: 当前用户余额；当前用户没有金币账户时为 0
- `myWinRate`: 当前用户预测胜率
- `myCurrentWinStreak`: 当前用户最新连胜场数

`items` 字段：

- `rank`: 排名
- `userId`: 用户 ID
- `nickname`: 用户名称
- `avatar`: 用户头像
- `balance`: 当前金币余额
- `winRate`: 用户预测胜率
- `currentWinStreak`: 用户最新连胜场数

示例：

```json
{
  "items": [
    {
      "rank": 1,
      "userId": 101,
      "nickname": "Alice",
      "avatar": "https://example.com/a.png",
      "balance": 9000,
      "winRate": 0.75,
      "currentWinStreak": 3
    }
  ],
  "myRank": 12,
  "myBalance": 1200,
  "myWinRate": 0.5,
  "myCurrentWinStreak": 1
}
```

---

### 3.2) 预测市场评论并绑定选项

- **接口**：`POST /api/comment/create`
- **认证**：需要登录
- **请求格式**：表单

推荐参数：

- `entityType`: `predict_market`
- `entityId`: marketId
- `option`: `A/B/DRAW`
- `content`: 评论内容

兼容参数：

- 如果仍使用 `entityType=topic` 且 `entityId` 恰好为 marketId，传入 `option` 时也会写入 `PredictCommentMeta`。

示例：

```bash
curl -X POST "http://localhost:8082/api/comment/create" \
  --cookie "bbsgo_token=YOUR_TOKEN" \
  -F "entityType=predict_market" \
  -F "entityId=1" \
  -F "option=A" \
  -F "content=支持 A"
```

### 3.3) 管理员手动触发评论奖励

- **接口**：`POST /api/admin/predict/comment_reward/run`
- **认证**：需要管理员权限
- **请求格式**：表单

参数：

- `marketId`: int64

示例：

```bash
curl -X POST "http://localhost:8082/api/admin/predict/comment_reward/run" \
  --cookie "bbsgo_token=ADMIN_TOKEN" \
  -F "marketId=1"
```

### 3.4) 管理员重试失败评论奖励

- **接口**：`POST /api/admin/predict/comment_reward/retry`
- **认证**：需要管理员权限

参数：

- `rewardLogId`: int64

### 3.5) 管理员查询评论奖励审计

- **接口**：`GET /api/admin/predict/comment_reward/logs`
- **认证**：需要管理员权限

参数：

- `marketId`: int64，可选
- `status`: string，可选
- `page`: int，可选
- `limit`: int，可选

---

## 错误码与错误信息

本服务接口统一返回 `web.JsonResult`；错误通常以 `msg` 文本形式返回（以实际实现为准）。本模块涉及的常见错误信息包括：

- 认证错误：`NotLogin`
- 参数校验：
  - `marketId is required`
  - `option must be A or B`
  - `amount must be positive`
  - `userId is required`
- 业务错误：
  - `market is not open`
  - `market is closed`
  - `insufficient balance`

### 1) 查询我的金币账户

- **接口**：`GET /api/coin/me`
- **认证**：需要登录

#### 返回值（data）
`UserCoin`

示例：

```json
{
  "id": 1,
  "userId": 100,
  "balance": 12345,
  "createTime": 1734010000,
  "updateTime": 1734012345
}
```

---

### 2) 预测下注（会扣金币 + 锁赔率 + 更新池）

- **接口**：`POST /api/coin/bet`
- **认证**：需要登录
- **请求格式**：表单（`application/x-www-form-urlencoded` 或 `multipart/form-data`）

#### 请求参数（form）
- `marketId`: int64，必填
- `option`: string，必填；淘汰赛 `binary` 只能是 `A/B`，小组赛 `1x2` 可以是 `A/B/DRAW`（不区分大小写）
- `amount`: int64，必填，必须 > 0

（文档用 JSON 展示字段结构；实际是表单）

```json
{
  "marketId": 1,
  "option": "A",
  "amount": 100
}
```

#### 返回值（data）
`PlaceBetResult`

- `bet`: PredictBet
- `market`: PredictMarket（已更新 `poolA/poolB/poolDraw`）
- `userCoin`: UserCoin（已扣款后的余额）
- `lockedOdds`: float64（等同于 bet.odds）

设计补充：

- 下注成功后会按下注金额增加对应 `PredictContext.heat`。
- P0 公式：`heatDelta = amount`，不设置上限。
- 该热度规则覆盖世界杯市场与 Polymarket 市场。
- 预测市场一级评论也会增加同一个 `PredictContext.heat`，评论热度增量为 1。

示例（字段会随模型演进，这里仅展示结构）：

```json
{
  "bet": {
    "id": 10,
    "userId": 100,
    "marketId": 1,
    "option": "A",
    "amount": 100,
    "odds": 1.83,
    "effA": 500,
    "effB": 500,
    "status": "OPEN",
    "createTime": 1734012345
  },
  "market": {
    "id": 1,
    "status": "OPEN",
    "baseA": 500,
    "baseB": 500,
    "poolA": 100,
    "poolB": 0
  },
  "userCoin": {
    "userId": 100,
    "balance": 12245
  },
  "lockedOdds": 1.83
}
```

#### 可能错误
- 未登录：`errs.NotLogin()`
- 参数校验：
  - `marketId is required`
  - `option must be A or B`
  - `option must be A, B or DRAW`
  - `amount must be positive`
- 业务错误：
  - `market is not open`
  - `market is closed`
  - `insufficient balance`

---

### 4) 管理员铸币（给用户加金币）

- **接口**：`POST /api/admin/coin/mint`
- **认证**：需要管理员权限
- **请求格式**：表单

#### 请求参数（form）
- `userId`: int64，必填
- `amount`: int64，必填，必须 > 0
- `remark`: string，可选

```json
{
  "userId": 100,
  "amount": 1000,
  "remark": "活动派奖"
}
```

#### 返回值（data）
`UserCoin`（加币后的余额）

示例：

```json
{
  "userId": 100,
  "balance": 13245
}
```
