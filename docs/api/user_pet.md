# 用户侧宠物（user_pet）API

> 目标：承载“用户侧宠物系统”的接口契约，避免在规则文档里出现接口设计内讧。
>
> 本文档只描述：接口、请求响应、错误码、幂等。
> 规则与玩法口径请以 `prompt/project/宠物-用户侧宠物使用.md` 为单一事实来源。

---

## 通用约定

- Base URL：`/api/pet`
- 认证：token（与现有用户体系保持一致）
- 金额/数量：统一整数（龟币/体力/XP）。
- 日切：北京时间（UTC+8 0 点）。

### 错误码（建议）

- `ALREADY_SETTLED`：今日已结算
- `EQUIP_DAILY_LIMIT`：今日切龟次数已用尽
- `DEBT_UNPAID`：欠款未还清禁止切龟
- `INSUFFICIENT_COINS`：余额不足
- `STAMINA_NOT_ENOUGH`：体力不足
- `PARAM_INVALID`：参数错误

---

## 1) 登录接口即每日结算入口（主入口）

> 说明：每日登录结算在“登录成功”时触发；同一天重复登录不重复发放。
> 原则：登录不能被结算失败阻断。
> P0 口径：**登录概念等于签到**，因此不再单独返回 checkInStreak。

- `POST /api/user/login`（或项目现有登录 API）

### 响应：新增 dailySettle

> 在原登录响应上新增字段 `dailySettle`。

- `dailySettle.date`：`YYYY-MM-DD`（北京）
- `dailySettle.alreadySettled`：bool
- `dailySettle.balanceBefore` / `dailySettle.balanceAfter`
- `dailySettle.items[]`：结算明细
  - `type`：`base_checkin` | `spark_reward` | `spark_bonus` | `debt_subsidy` | `deposit_interest` | `pet_signin_bonus` | `dice_bonus` ...
  - `amount`：int64
  - `desc`：string
  - `meta`：object（可选）
    - `spark_reward.meta.loginStreak`：本次计算使用的连续登录天数
    - `spark_bonus.meta.raw`：基础火花奖励
    - `spark_bonus.meta.final`：应用倍率后的最终火花奖励
    - `spark_bonus.meta.petId/level/loginStreak`：触发倍率的宠物与等级信息
- `dailySettle.streak.loginStreak`：int
- `dailySettle.pet.petId` / `petKey` / `level`
- `dailySettle.errorCode` / `errorMsg`：（可选）结算失败时填充，但登录仍成功

幂等：同一用户同一天只能结算一次；重复登录返回 `alreadySettled=true` 并带回当日 summary。

---


## 2) 当前装备龟种（读）

- `GET /api/pet/equip`

响应：

- `petId` / `petKey`
- `petName`
- `rarity`
- `level`
- `equippedAt`：时间戳
- `equipDayName`：int（北京 dayName，用于每日切龟限制展示）

---

## 3) 切换龟种（写）

- `POST /api/pet/equip`

请求：

- `petId`（或 `petKey`，二选一；建议只用一个口径）

响应：

- `ok`: bool
- `pet`：同 `GET /api/pet/equip`
- `nextEffectiveAt`：北京时间次日 0 点

校验：

- 今日是否已切换：`EQUIP_DAILY_LIMIT`
- 欠款未还清（余额 < 0）禁止切换：`DEBT_UNPAID`
- pet 是否为用户已拥有：`PARAM_INVALID`（或 `PET_NOT_OWNED`）

---

## 4) 用户龟种资产（列表）

- `GET /api/pet/owned`

响应：

- `equippedPetId`
- `list[]`
  - `petId` / `petKey` / `petName` / `rarity`
  - `level` / `xp`
  - `isEquipped`：bool
  - `obtainedAt`：时间戳

---

## 5) 体力（查询 + 消耗/恢复）

- `GET /api/pet/stamina`

响应：

- `current`
- `cap`（固定 100）
- `regenPerHour`（固定 5）
- `nextRegenAt`（可选）

（可选）

- `POST /api/pet/stamina/consume`
  - 请求：`amount`
  - 错误：`STAMINA_NOT_ENOUGH`

- `POST /api/pet/stamina/feed`
  - 请求：`count`
  - 行为：扣币（含折扣）+ 回体力 + 加 XP
  - 错误：`INSUFFICIENT_COINS`

---

## 6) 开蛋（抽龟）

- `POST /api/pet/egg/hatch`

行为：

- 读取开蛋池配置（enabled/base_cost/rarity_weights）。
- 先按 `rarity_weights` 抽稀有度，再在该稀有度、`obtainable_by_egg=true` 的龟种定义中**均匀随机**抽取一个龟种。
- 事务内完成：扣费 →（若新龟）入库发放 /（若重复）按规则返还。

重复返还规则：

- 若抽中的龟种用户已拥有（`t_user_pet` 已存在），不重复发放。
- 返还金额固定为：`refund = floor(cost * 0.3)`（即实际扣费的 30%）。
- 记账：返还会额外写一条金币流水（`t_user_coin_log`），`biz_type=PET_EGG_DUPLICATE_REFUND`。

响应：

- `cost`：实际扣费（已折扣）
- `refund`：重复返还金额（无则 0）
- `isDuplicate`：bool
- `pet`：抽中的龟种（`petId/petKey/rarity/name`）
- `balanceBefore/balanceAfter`

一致性：扣费/抽取/入库/返还必须在事务内完成。

---

## 7) 状态页聚合

- `GET /api/pet/status`

响应：

- `moodState`
- `voteStats`
- `spark`
- `daily`：今日是否已结算、上次结算时间
- `ai`：最近 N 条对话（可选）
