# 开战广场（Battle Square）

> 本文档基于真实后端代码（Iris MVC 路由）整理。
>
> - 用户接口：`/api/battle/**`（需要登录：`AuthMiddleware`）
> - 管理员接口：`/api/admin/battle/**`（需要管理员：`AdminMiddleware`）
>
> 代码位置：
>
> - 控制器：`internal/controllers/api/battle_controller.go`
> - 服务：`internal/services/battle_service.go`
> - 模型：`internal/models/models.go`（`Battle/BattleBet/BattleChallengeAction/BattleSettlement/*`）

## 数据模型速览

## 统一响应与错误约定

项目统一使用 JSON 包装返回（见 `github.com/mlogclub/simple/web`）：

- 成功：`{"success":true,"data":...}`
- 失败：`{"success":false,"message":"..."}`

> 备注：错误场景目前主要依赖 `message` 文案区分（例如 `battle not found`、`battleId is required`、`未登录` 等）。
> 如需前端更稳定的错误码（`code`），建议后续在 `web.JsonError*` 输出中加上统一 `code`。

### message 来源层级（便于定位与提示策略）

当前 battle 相关 `message` 主要来自以下几层（通常会直接透传到前端）：

1) Controller 参数校验（例如 `battleId is required`、`battle not found`）
2) BattleService 业务校验（例如 `battle is not open`、`permission denied`、`invalid result`）
3) Coin/转账层（`UserCoinService`），主要是扣款/转账失败（例如 `insufficient balance`）
4) DB/ORM 层错误（`gorm` 返回的错误字符串，一般不稳定，不建议直接面向用户）

建议前端策略：

- 对“高频可预期”的 message 做**白名单映射成中文提示**（详见下方建议表）。
- 对 DB/未知错误：统一提示「系统繁忙，请稍后重试」，同时上报 message 便于排查。

### 前端 message 映射建议（高频）

| message（后端原文） | 建议中文提示 |
| --- | --- |
| `battle not found` | 赌局不存在或已删除 |
| `battle is not open` | 赌局当前不可加入 |
| `battle is full` | 赌局已满 |
| `invalid inviteCode` | 邀请码错误 |
| `permission denied` | 无权限操作 |
| `insufficient balance` | 余额不足 |
| `battle is not settled` | 尚未结算，无法提取 |
| `no payout for this user` | 你在本局没有可提取金额 |
| `banker has not declared result` | 庄家尚未宣判 |
| `battle is not pending` | 当前阶段不允许该操作 |
| `battle is not disputed` | 当前不在争议仲裁状态 |

### 示例：错误响应

```json
{
  "success": false,
  "message": "battle not found"
}
```

## 常见错误场景清单（message 对照）

以下内容来自 `internal/services/battle_service.go` 与控制器参数校验（message 以当前实现为准）。

### 通用

- 未登录：`web.JsonError(errs.NotLogin())`（message 通常为「未登录」或类似文案，取决于 `errs.NotLogin()` 实现）
- 参数缺失/非法（控制器侧）：
  - `battleId` 缺失：`battleId is required`
  - 资源不存在：`battle not found`

### 创建赌局：`POST /api/battle/create`

- `title` 为空：`title is required`
- `bankerSide/challengerSide` 为空：`sides are required`
- `stakeAmount < 100`：`stakeAmount must be >= 100`
- `settleTime <= 0`：`settleTime is required`
- 私密场未传 `inviteCode`：`inviteCode is required for private battle`

> 余额不足：由 `UserCoinService.SpendToPool` 返回的 message 决定（通常是余额不足/扣款失败），文案需以实际实现为准。

补充（来自 `internal/services/user_coin_service.go`）：

- 余额不足：`insufficient balance`
- 金额非法（<=0）：`amount must be positive`
- userId 非法：`userId is required`

### 加入/追加下注：`POST /api/battle/join`

- `amount <= 0`：`amount must be positive`
- `requestId` 为空：`requestId is required`
- 庄家尝试作为挑战者加入：`banker cannot join as challenger`
- battle 不在 open：`battle is not open`
- 私密场邀请码错误：`invalid inviteCode`
- 容量已满：`battle is full`
- 下注超过剩余容量：`amount exceeds remaining capacity: {remaining}`

> 幂等说明：同一个用户对同一 battle 传相同 `requestId` 会直接返回已有下注明细，不会重复扣款。

补充（coin/转账层可能返回）：

- 余额不足：`insufficient balance`
- 金额非法（<=0）：`amount must be positive`

### 庄家加注：`POST /api/battle/banker_add_stake`

- `amount <= 0`：`amount must be positive`
- `requestId` 为空：`requestId is required`
- 非庄家操作：`permission denied`
- battle 已进入 `pending/disputed/settled`：`battle is not allowed to add stake`
- 已到结算时间：`battle already reached settle time`

补充（coin/转账层可能返回）：

- 余额不足：`insufficient balance`
- 金额非法（<=0）：`amount must be positive`

### 庄家宣布结果：`POST /api/battle/declare`

- `result` 非法：`invalid result`
- 非庄家操作：`permission denied`
- battle 不在 pending：`battle is not pending`

### 挑战者确认：`POST /api/battle/challenger_confirm`

- battle 不在 pending：`battle is not pending`
- 庄家未宣判：`banker has not declared result`
- 非挑战者尝试确认：`only challenger can confirm`

> 幂等/去重：
> - 同一 `requestId` 重试：直接返回当前 battle（不重复写动作）。
> - 同一个用户已做过 confirm/dispute：当前实现也会直接返回 battle（不会报错）。

### 挑战者异议：`POST /api/battle/challenger_dispute`

- battle 不在 pending：`battle is not pending`
- 庄家未宣判：`banker has not declared result`
- 非挑战者尝试异议：`only challenger can dispute`

### 提取：`POST /api/battle/withdraw`

- battle 未结算：`battle is not settled`
- 当前用户无 payout：`no payout for this user`

补充（coin/转账层可能返回）：

- 资金池余额不足（理论上不应发生；若发生通常是账不平）：`insufficient balance`

> 幂等说明：若 `withdrawn=true`，重复调用会直接返回 item，不会重复出池。

### 管理员裁决：`POST /api/admin/battle/resolve`

- `result` 非法：`invalid result`
- battle 不在 disputed：`battle is not disputed`

> 幂等说明：同一管理员 + 同一 `requestId` 重试会直接返回当前 battle，不会重复执行处罚 burn / 重复生成结算单。

### Battle（赌局）

字段以 `internal/models/models.go` 为准，常用字段：

- `id`: int64
- `title`: string
- `bankerUserId`: int64
- `bankerSide` / `challengerSide`: string
- `isPublic`: bool（公开场收取入场费；私密场需邀请码且不收入场费）
- `inviteCode`: string
- `status`: string（`open/sealed/pending/disputed/settled`）
- `settleTime`: int64（到点后自动进入待宣判/待确认流程）
- `pendingDeadline`: int64（庄家宣判截止；超时默认庄家输）
- `confirmDeadline`: int64（挑战者确认截止；超时默认确认）
- `disputeDeadline`: int64（管理员仲裁截止；超时默认 void）
- `disputedByUserId`: int64（发起争议的挑战者）
- `result`: string（`banker_wins/banker_loses/void`，空字符串表示未出最终结果）
- `resultBy`: string（`banker/timeout/confirm_timeout/admin/admin_timeout/...`）
- `resultTime`: int64

资金汇总字段：

- `bankerStakeTotal`: int64
- `challengerStakeTotal`: int64
- `poolPrincipalTotal`: int64（本金托管资金池：userId=-1）
- `entryFeeTotal`: int64（入场费累计，实时转给庄家）
- `burnTotal`: int64（燃烧累计；包括 void 规则 burn + 管理员裁决处罚 burn）

### BattleChallengeAction（挑战者动作）

- `battleId`: int64
- `userId`: int64
- `action`: string（`confirm/dispute`）
- `requestId`: string（幂等）

### BattleSettlement / BattleSettlementItem（结算单与明细）

- 进入 `settled` 后生成结算单（battleId 唯一幂等），每个用户一条 item。
- 提取接口按 item 一次性全提，重复请求幂等。

## 用户接口（/api/battle）

### 1) 赌局列表

- **接口**：`GET /api/battle/list`
- **认证**：需要登录

#### 请求参数（query）
- `page`: int，默认 1
- `pageSize`: int，默认 20，最大 100
- `status`: string，可选（`open/sealed/pending/disputed/settled`）
- `mine`: string，可选；传 `1` 时仅返回我参与的（我是庄家或我下过注）【历史兼容】
- `role`: string，可选；更精确的参与角色筛选（优先级高于 `mine`）
  - `role=banker`：只看我做庄的赌局（`battle.bankerUserId = me`）
  - `role=challenger`：只看我挑战的赌局（我作为 challenger 下过注；不包含我做庄的）

#### 返回值（data）
- `list`: 数组，每项结构：
  - `battle`: Battle
  - `myAction`: string（当前用户对该 battle 的挑战者动作：`confirm/dispute/""`）
  - `bankerNickname`: string（庄家昵称；可能为空字符串）
  - `commentCount`: int64（该 battle 的评论数；基于 `Comment.entity_type="battle"` 聚合）
  - `likeCount`: int64（该 battle 的点赞数；基于 `UserLike.entity_type="battle"` 聚合）
- `count`: int64（总数）
- `page`: int
- `pageSize`: int

#### 示例

请求：

```http
GET /api/battle/list?page=1&pageSize=20&status=open&mine=1
```

只看我做庄：

```http
GET /api/battle/list?page=1&pageSize=20&role=banker
```

只看我挑战：

```http
GET /api/battle/list?page=1&pageSize=20&role=challenger
```

响应（示意，仅展示关键字段）：

```json
{
  "success": true,
  "data": {
    "list": [
      {
        "battle": {
          "id": 1,
          "title": "xxx",
          "status": "open",
          "bankerUserId": 10001,
          "bankerStakeTotal": 1000,
          "challengerStakeTotal": 300,
          "poolPrincipalTotal": 1300,
          "entryFeeTotal": 0,
          "burnTotal": 0
        },
        "myAction": ""
  },
  "myAction": "",
  "bankerNickname": "Alice",
  "commentCount": 12,
  "likeCount": 8
    ],
    "count": 1,
    "page": 1,
    "pageSize": 20
  }
}
```

---

### 1.1) 赌局统计（全局）

- **接口**：`GET /api/battle/stats`
- **认证**：需要登录
- **说明**：返回未结算的赌局数量、等待结算（pending）的赌局数量、未结算赌局的资金池本金总额，以及庄家去重数量。

#### 返回值（data）

- `unsettledCount`: int64（未结算赌局数：`status != settled`）
- `pendingCount`: int64（等待结算赌局数：`status = pending`）
- `poolTotal`: int64（未结算赌局 `poolPrincipalTotal` 总和）
- `bankerCount`: int64（未结算赌局庄家去重数：distinct `bankerUserId`）

#### 示例

请求：

```http
GET /api/battle/stats
```

响应：

```json
{
  "success": true,
  "data": {
    "unsettledCount": 12,
    "pendingCount": 3,
    "poolTotal": 456789,
    "bankerCount": 5
  }
}
```

## 管理员接口（/api/admin/battle）

### 手动触发轮巡

- **接口**：`POST /api/admin/battle/cron_tick`
- **认证**：需要管理员（`AdminMiddleware`）
- **说明**：人工触发一次 `BattleService.CronTick()`，用于立即执行 open->sealed、sealed->pending、pending 超时等状态迁移。

#### 请求参数

无。

#### 返回

成功：

```json
{ "success": true }
```

失败示例：

```json
{ "success": false, "message": "..." }
```

---

### 2) 赌局详情

- **接口**：`GET /api/battle/by?battleId=1`
- **认证**：需要登录

#### 请求参数（query）
- `battleId`: int64，必填

#### 返回值（data）
- `battle`: Battle
- `myAction`: string（`confirm/dispute/""`）
- `settlement`:
  - `settlement`: BattleSettlement（若未生成则为 null）
  - `myItem`: BattleSettlementItem（若我无 payout 或未生成则为 null）

#### 示例

请求：

```http
GET /api/battle/by?battleId=1
```

响应（未结算时 `settlement.settlement=null`）：

```json
{
  "success": true,
  "data": {
    "battle": {
      "id": 1,
      "status": "pending",
      "result": "",
      "pendingDeadline": 1760000000,
      "confirmDeadline": 1760003600
    },
    "myAction": "",
    "settlement": {
      "settlement": null,
      "myItem": null
    }
  }
}
```

响应（已结算，含我的提取明细）：

```json
{
  "success": true,
  "data": {
    "battle": {
      "id": 1,
      "status": "settled",
      "result": "banker_wins",
      "resultBy": "admin",
      "resultTime": 1760000500
    },
    "myAction": "confirm",
    "settlement": {
      "settlement": {
        "id": 10,
        "battleId": 1,
        "result": "banker_wins",
        "createdAt": 1760000500
      },
      "myItem": {
        "id": 100,
        "battleId": 1,
        "userId": 20001,
        "payoutAmount": 1234,
        "withdrawn": false,
        "withdrawTime": 0
      }
    }
  }
}
```

---

### 3) 创建赌局

- **接口**：`POST /api/battle/create`
- **认证**：需要登录
- **请求格式**：JSON

#### 请求体（JSON）
字段来源：`services.CreateBattleForm`

- `title`: string，必填
- `bankerSide`: string，必填
- `challengerSide`: string，必填
- `stakeAmount`: int64，必填（最小 100）
- `isPublic`: bool，必填
- `inviteCode`: string，私密场必填
- `settleTime`: int64，必填（秒级时间戳）
- `requestId`: string，可选

#### 返回值（data）
- Battle

#### 示例

请求：

```http
POST /api/battle/create
Content-Type: application/json

{
  "title": "巴萨 vs 皇马，谁赢？",
  "bankerSide": "巴萨",
  "challengerSide": "皇马",
  "stakeAmount": 1000,
  "isPublic": true,
  "inviteCode": "",
  "settleTime": 1760000000,
  "requestId": "create-001"
}
```

响应（示意）：

```json
{
  "success": true,
  "data": {
    "id": 1,
    "title": "巴萨 vs 皇马，谁赢？",
    "status": "open",
    "bankerUserId": 10001,
    "bankerStakeTotal": 1000,
    "challengerStakeTotal": 0
  }
}
```

---

### 4) 加入/追加下注

- **接口**：`POST /api/battle/join`
- **认证**：需要登录
- **请求格式**：JSON

#### 请求体（JSON）
字段来源：`services.JoinBattleForm`

- `battleId`: int64，必填
- `amount`: int64，必填（>0）
- `requestId`: string，必填（幂等）
- `inviteCode`: string，私密场必填

#### 说明
- 公开场会对 `amount` 收取 5% 入场费（向下取整），实时转给庄家。
- 本金入资金池（userId=-1）。
- 挑战者总额不得超过庄家押注额，满额会自动封盘。

#### 返回值（data）
- `battle`: Battle
- `bet`: BattleBet

#### 示例

请求：

```http
POST /api/battle/join
Content-Type: application/json

{
  "battleId": 1,
  "amount": 300,
  "requestId": "join-20001-0001",
  "inviteCode": ""
}
```

响应（示意）：

```json
{
  "success": true,
  "data": {
    "battle": {
      "id": 1,
      "status": "open",
      "bankerStakeTotal": 1000,
      "challengerStakeTotal": 300,
      "poolPrincipalTotal": 1295,
      "entryFeeTotal": 5
    },
    "bet": {
      "id": 11,
      "battleId": 1,
      "userId": 20001,
      "amount": 300
    }
  }
}
```

---

### 5) 庄家加注

- **接口**：`POST /api/battle/banker_add_stake`
- **认证**：需要登录
- **请求格式**：JSON

#### 请求体（JSON）
字段来源：`services.BankerAddStakeForm`

- `battleId`: int64，必填
- `amount`: int64，必填
- `requestId`: string，必填（幂等）

#### 返回值（data）
- Battle

#### 示例

```http
POST /api/battle/banker_add_stake
Content-Type: application/json

{
  "battleId": 1,
  "amount": 500,
  "requestId": "banker-add-0001"
}
```

---

### 6) 庄家宣布结果

- **接口**：`POST /api/battle/declare`
- **认证**：需要登录
- **请求格式**：表单（通过 `params` 读取）

#### 请求参数（form）
- `battleId`: int64，必填
- `result`: string，必填（`banker_wins/banker_loses`）

#### 说明
- 宣布后进入挑战者确认窗口（`confirmDeadline = now + 24h`）。
- 任一挑战者异议会进入 `disputed` 等待管理员裁决。

#### 返回值（data）
- Battle

#### 示例

```http
POST /api/battle/declare
Content-Type: application/x-www-form-urlencoded

battleId=1&result=banker_wins
```

---

### 7) 挑战者确认

- **接口**：`POST /api/battle/challenger_confirm`
- **认证**：需要登录
- **请求格式**：JSON

#### 请求体（JSON）
字段来源：`services.ChallengeActionForm`

- `battleId`: int64，必填
- `requestId`: string，必填（幂等）
- `remark`: string，可选

#### 返回值（data）
- Battle（若全部挑战者确认达成，会变更为 `settled` 并生成结算单）

#### 示例

```http
POST /api/battle/challenger_confirm
Content-Type: application/json

{
  "battleId": 1,
  "requestId": "confirm-20001-0001",
  "remark": "同意结果"
}
```

---

### 8) 挑战者异议

- **接口**：`POST /api/battle/challenger_dispute`
- **认证**：需要登录
- **请求格式**：JSON

#### 请求体（JSON）
字段来源：`services.ChallengeActionForm`

- `battleId`: int64，必填
- `requestId`: string，必填（幂等）
- `remark`: string，可选

#### 返回值（data）
- Battle（进入 `disputed`，并设置 `disputeDeadline = now + 24h`）

#### 示例

```http
POST /api/battle/challenger_dispute
Content-Type: application/json

{
  "battleId": 1,
  "requestId": "dispute-20001-0001",
  "remark": "我认为庄家宣判不正确"
}
```

---

### 9) 提取（一次性全提）

- **接口**：`POST /api/battle/withdraw`
- **认证**：需要登录
- **请求格式**：JSON

#### 请求体（JSON）
字段来源：`services.WithdrawForm`

- `battleId`: int64，必填
- `requestId`: string，必填（幂等）

#### 返回值（data）
- BattleSettlementItem（我的结算明细，带 `withdrawn/withdrawTime`）

#### 示例

```http
POST /api/battle/withdraw
Content-Type: application/json

{
  "battleId": 1,
  "requestId": "withdraw-20001-0001"
}
```

响应（示意）：

```json
{
  "success": true,
  "data": {
    "battleId": 1,
    "userId": 20001,
    "payoutAmount": 1234,
    "withdrawn": true,
    "withdrawTime": 1760000600
  }
}
```

---

## 管理员接口（/api/admin/battle）

### 1) 管理员裁决

- **接口**：`POST /api/admin/battle/resolve`
- **认证**：需要管理员
- **请求格式**：JSON

#### 请求体（JSON）
字段来源：`services.AdminResolveForm`

- `battleId`: int64，必填
- `requestId`: string，必填（幂等）
- `result`: string，必填（`banker_wins/banker_loses/void`）
- `remark`: string，可选

#### 说明
- 裁决后 battle 进入 `settled` 并生成结算单。
- 当前实现包含一条“处罚 burn”资金流：管理员裁决时按 `poolPrincipalTotal * 10%` 从资金池 burn（幂等）。

#### 返回值（data）
- Battle

#### 示例

```http
POST /api/admin/battle/resolve
Content-Type: application/json

{
  "battleId": 1,
  "requestId": "admin-resolve-0001",
  "result": "void",
  "remark": "证据不足，作废"
}
```

## 前端对接建议：按钮态（可选，建议前端自行计算）

后端目前返回 `battle.status/result/*deadline` 与 `myAction`（详情、列表都带）。
前端通常会用这些字段计算按钮可用性，建议统一成以下布尔值（不需要后端改代码即可实现）：

- `canJoin`：`battle.status in [open]` 且（私密场邀请码已校验）
- `canBankerAddStake`：我是庄家 且 `battle.status == open`
- `canDeclare`：我是庄家 且 `battle.status == pending` 且 `now <= pendingDeadline`
- `canConfirm`：我是挑战者 且 `battle.status == pending` 且 `myAction == ""` 且 `now <= confirmDeadline`
- `canDispute`：我是挑战者 且 `battle.status == pending` 且 `myAction == ""` 且 `now <= confirmDeadline`
- `canWithdraw`：`battle.status == settled` 且 `settlement.myItem != null` 且 `settlement.myItem.withdrawn == false`

> 注意：列表接口未返回 `settlement`，如需在列表直接显示 `canWithdraw`，需要额外请求详情或扩展列表返回。

