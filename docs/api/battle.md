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
- `mine`: string，可选；传 `1` 时仅返回我参与的（我是庄家或我下过注）

#### 返回值（data）
- `list`: 数组，每项结构：
  - `battle`: Battle
  - `myAction`: string（当前用户对该 battle 的挑战者动作：`confirm/dispute/""`）
- `count`: int64（总数）
- `page`: int
- `pageSize`: int

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

