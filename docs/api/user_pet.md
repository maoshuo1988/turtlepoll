# 用户侧宠物（user_pet）API

> 目标：承载“用户侧宠物系统”的接口契约，避免在规则文档里出现接口设计内讧。
>
> 本文档只描述：接口、请求响应、错误码、幂等。
> 规则与玩法口径请以 `prompt/project/宠物/` 为单一事实来源。

---

## 通用约定

- Base URL：`/api/pet`
- 认证：token（与现有用户体系保持一致）
- 金额/数量：统一整数（龟币/体力/XP）。
- 日切：北京时间（UTC+8 0 点）。

### 宠物展示信息统一结构

`GET /api/pet/defs`、`GET /api/pet/equip`、`POST /api/pet/equip`、`GET /api/pet/owned` 中涉及宠物详情时，统一返回 `pet` 对象。为兼容现有前端，短期内保留旧的扁平字段，如 `petId/petKey/petName/rarity`。

`pet` 字段：

- `id`：number，`PetDefinition.Id`，用于当前后端装备、拥有关系查询。
- `petId`：number，兼容旧字段，值等同于 `id`。
- `petKey`：string，宠物业务 key。
- `petCode`：string，`PetDefinition.pet_id` 自然键；如果历史数据为空，返回 `petKey`。
- `name`：string，宠物展示名，优先取中文名，其次取兼容字段 `name_plain`。
- `rarity`：number，当前后端稀有度数值。
- `rarityKey`：string，可选，将数值映射为 `C/B/A/S/SS/SSS`，便于前端展示。
- `image`：string，宠物主图。取值优先级：`display.icon` -> `display.thumbnail` -> `display.cover` -> `icon`。
- `icon`：string，兼容字段，值同 `image`。
- `abilities`：object，宠物能力 JSON，来自 `PetDefinition.abilities`。
- `abilityDescriptions`：array，能力中文说明列表，由 `abilities` 参数和 `FeatureCatalog` 中文名生成，供前端直接展示。

`abilityDescriptions[]` 字段：

- `featureKey`：string，能力 key，例如 `signin_bonus`。
- `name`：string，中文能力名，优先取 `FeatureCatalog.name.zh-CN`，没有则回退到 featureKey。
- `description`：string，基于当前能力参数生成的中文说明。
- `enabled`：bool，当前能力参数是否启用；未配置时默认 `true`。

示例：

```json
{
  "id": 12,
  "petId": 12,
  "petKey": "phoenix",
  "petCode": "phoenix",
  "name": "凤凰龟",
  "rarity": 4,
  "rarityKey": "S",
  "image": "/res/pets/phoenix.png",
  "icon": "/res/pets/phoenix.png",
  "abilities": {
    "signin_bonus": {
      "enabled": true,
      "bonusCoins": 100,
      "capPerDay": 500
    }
  },
  "abilityDescriptions": [
    {
      "featureKey": "signin_bonus",
      "name": "每日登录加成",
      "description": "每日登录额外获得100龟币，每日上限500龟币。",
      "enabled": true
    }
  ]
}
```

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


## 2) 龟种定义列表（普通用户）

- `GET /api/pet/defs`

说明：普通用户可访问的龟种定义列表接口，参考管理端 `GET /api/admin/pet/defs` 的查询能力，但不需要管理员权限。

查询参数：

- `enabled`：可选，`true/false/1/0`，按 `obtainable_by_egg` 过滤。
- `rarity`：可选，`C/B/A/S/SS/SSS`。
- `page` / `size`：分页参数，`size` 最大 200。

响应：

- `items[]`：统一 `pet` 展示对象。
- `total`：总数。

---

## 3) 当前装备龟种（读）

- `GET /api/pet/equip`

响应：

- `pet`：宠物展示信息统一结构，包含宠物能力、宠物名字、稀有度、ID、图片。
- `petId` / `petKey`：兼容旧字段。
- `petName`：兼容旧字段，值同 `pet.name`。
- `rarity`：兼容旧字段，值同 `pet.rarity`。
- `abilities`：兼容旧字段，值同 `pet.abilities`。
- `abilityDescriptions`：兼容旧字段，值同 `pet.abilityDescriptions`。
- `image` / `icon`：兼容旧字段，值同 `pet.image`。
- `level`
- `equippedAt`：时间戳
- `equipDayName`：int（北京 dayName，用于每日切龟限制展示）

响应示例：

```json
{
  "petId": 12,
  "petKey": "phoenix",
  "petName": "凤凰龟",
  "rarity": 4,
  "abilities": {
    "signin_bonus": {
      "enabled": true,
      "bonusCoins": 100,
      "capPerDay": 500
    }
  },
  "abilityDescriptions": [
    {
      "featureKey": "signin_bonus",
      "name": "每日登录加成",
      "description": "每日登录额外获得100龟币，每日上限500龟币。",
      "enabled": true
    }
  ],
  "image": "/res/pets/phoenix.png",
  "icon": "/res/pets/phoenix.png",
  "level": 3,
  "equippedAt": 1760000000,
  "equipDayName": 20260527,
  "pet": {
    "id": 12,
    "petId": 12,
    "petKey": "phoenix",
    "petCode": "phoenix",
    "name": "凤凰龟",
    "rarity": 4,
    "rarityKey": "S",
    "image": "/res/pets/phoenix.png",
    "icon": "/res/pets/phoenix.png",
    "abilities": {
      "signin_bonus": {
        "enabled": true,
        "bonusCoins": 100,
        "capPerDay": 500
      }
    },
    "abilityDescriptions": [
      {
        "featureKey": "signin_bonus",
        "name": "每日登录加成",
        "description": "每日登录额外获得100龟币，每日上限500龟币。",
        "enabled": true
      }
    ]
  }
}
```

---

## 4) 切换龟种（写）

- `POST /api/pet/equip`

请求：

- `petId`（或 `petKey`，二选一；建议只用一个口径）

响应：

- `ok`: bool
- `pet`：宠物展示信息统一结构，包含宠物能力、宠物名字、稀有度、ID、图片。
- `petId` / `petKey` / `petName` / `rarity`：兼容旧字段。
- `abilities` / `abilityDescriptions` / `image` / `icon`：兼容旧字段。
- `nextEffectiveAt`：北京时间次日 0 点

校验：

- 今日是否已切换：`EQUIP_DAILY_LIMIT`
- 欠款未还清（余额 < 0）禁止切换：`DEBT_UNPAID`
- pet 是否为用户已拥有：`PARAM_INVALID`（或 `PET_NOT_OWNED`）

---

## 5) 开蛋池配置（普通用户）

- `GET /api/pet/gacha/config`

说明：普通用户可访问的开蛋池配置只读接口，参考管理端 `GET /api/admin/pet/gacha/config`，但不需要管理员权限。

响应：

- `enabled`：开蛋池是否启用。
- `base_cost`：开蛋基础费用。
- `rarity_weights`：稀有度概率配置。

---

## 6) 用户龟种资产（列表）

- `GET /api/pet/owned`

响应：

- `equippedPetId`
- `list[]`
  - `pet`：宠物展示信息统一结构，包含宠物能力、宠物名字、稀有度、ID、图片。
  - `petId` / `petKey` / `petName` / `rarity`：兼容旧字段。
  - `abilities` / `abilityDescriptions` / `image` / `icon`：兼容旧字段。
  - `level` / `xp`
  - `isEquipped`：bool
  - `obtainedAt`：时间戳

响应示例：

```json
{
  "equippedPetId": 12,
  "list": [
    {
      "petId": 12,
      "petKey": "phoenix",
      "petName": "凤凰龟",
      "rarity": 4,
      "abilities": {
        "signin_bonus": {
          "enabled": true,
          "bonusCoins": 100,
          "capPerDay": 500
        }
      },
      "abilityDescriptions": [
        {
          "featureKey": "signin_bonus",
          "name": "每日登录加成",
          "description": "每日登录额外获得100龟币，每日上限500龟币。",
          "enabled": true
        }
      ],
      "image": "/res/pets/phoenix.png",
      "icon": "/res/pets/phoenix.png",
      "level": 3,
      "xp": 120,
      "isEquipped": true,
      "obtainedAt": 1760000000,
      "pet": {
        "id": 12,
        "petId": 12,
        "petKey": "phoenix",
        "petCode": "phoenix",
        "name": "凤凰龟",
        "rarity": 4,
        "rarityKey": "S",
        "image": "/res/pets/phoenix.png",
        "icon": "/res/pets/phoenix.png",
        "abilities": {
          "signin_bonus": {
            "enabled": true,
            "bonusCoins": 100,
            "capPerDay": 500
          }
        },
        "abilityDescriptions": [
          {
            "featureKey": "signin_bonus",
            "name": "每日登录加成",
            "description": "每日登录额外获得100龟币，每日上限500龟币。",
            "enabled": true
          }
        ]
      }
    }
  ]
}
```

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
