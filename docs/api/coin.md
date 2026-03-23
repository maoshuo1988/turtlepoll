# 金币与预测下注（Coin / PredictBet）

## 功能

本模块提供“金币账户（余额）+ 金币流水 + 预测市场下注（锁赔率）”相关能力：

- 用户接口：`/api/coin/**`（需要登录，`AuthMiddleware`）
- 管理员接口：`/api/admin/coin/**`（需要管理员权限，`AdminMiddleware`）

代码位置：

- 用户控制器：`internal/controllers/api/coin_controller.go`
- 管理员控制器：`internal/controllers/admin/coin_controller.go`
- 下注服务：`internal/services/predict_bet_service.go`
- 金币服务：`internal/services/user_coin_service.go`

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
为了支持下注赔率，本项目在 `PredictMarket` 上新增了 A/B 虚拟底池与真实下注池累计：

- `baseA` / `baseB`: int64（虚拟底池，默认 `500/500`）
- `poolA` / `poolB`: int64（用户下注累计池）

赔率使用“有效池 = base + pool”计算。

### PredictBet（预测下注订单）
代码定义：`internal/models/models.go` -> `type PredictBet`

常用字段：
- `id`: int64
- `userId`: int64
- `marketId`: int64
- `option`: string（`A` 或 `B`）
- `amount`: int64（下注金币）
- `odds`: float64（下单时锁定赔率；结算时不重算）
- `effA` / `effB`: int64（下单时看到的有效池快照）
- `status`: string（当前实现使用 `OPEN`）
- `createTime`: int64

## 赔率说明

下注时锁赔率（非常重要）：

- 下单时基于当前池计算 odds，并写入 `PredictBet.odds`
- 后续池变化不会影响该订单已经锁定的赔率

项目内实现：`internal/services/predict_odds.go` -> `CalcClampedOdds(...)`

- 赔率范围 clamp：`[1.2, 5.0]`
- 展示保留两位小数（代码中做了四舍五入）

## 接口列表

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
- `PredictMarket.result` 必须为 `A` 或 `B`（忽略大小写）
- 只结算该用户在该市场中 `PredictBet.status = OPEN` 的订单（幂等：重复调用不会重复派奖）
- 中奖派发：`payout = floor(amount * odds)`（odds 为下注时锁定赔率）

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
  - `market result must be A or B`

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
- `option`: string，必填，只能是 `A` 或 `B`（不区分大小写）
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
- `market`: PredictMarket（已更新 poolA/poolB）
- `userCoin`: UserCoin（已扣款后的余额）
- `lockedOdds`: float64（等同于 bet.odds）

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
