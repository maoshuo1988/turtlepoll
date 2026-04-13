# 开蛋池（Gacha Pool）配置（Admin）

> 说明：本文件只定义“开蛋池配置”的**接口契约与校验规则**，用于后台运营侧页面维护。
> 
> 注意：本文档不代表当前代码一定已实现；以实际后端实现为准。

---

## 资源：GachaPoolConfig

用于描述全局开蛋池的基础配置。

字段：

- `enabled` (boolean, required)
  - 含义：开蛋池总开关。
  - `false` 时：用户侧所有开蛋入口应直接返回不可用（实现可结合 kill-switch）。

- `base_cost` (integer, required)
  - 含义：开蛋基础费用（单位：系统币）。
  - 校验：`base_cost >= 0`。

- `rarity_weights` (object, required)
  - 含义：稀有度概率配置（权重）。
  - key：稀有度枚举 `C|B|A|S|SS|SSS`
  - value：概率（浮点数），范围 `[0,1]`
  - 示例：
    ```json
    {
      "C": 0.4,
      "B": 0.3,
      "A": 0.15,
      "S": 0.1,
      "SS": 0.04,
      "SSS": 0.01
    }
    ```

### 核心校验规则（保存时必须校验）

当运营侧调用“保存/更新开蛋池配置”接口时，后端必须校验：

1. **稀有度 key 合法**：只允许 `C|B|A|S|SS|SSS`。
2. **概率范围合法**：每个 value 必须满足 `0 <= p <= 1`。
3. **概率累加必须等于 1**：
   - 计算：`sum(p_i)`
   - 规则：`sum == 1`（推荐后端允许极小误差，例如 `abs(sum-1) <= 1e-9`，但对外语义仍是“必须等于 1”）
   - 若不满足：保存失败（HTTP 400），返回明确错误信息。

---

## 接口（建议）

> 路由组建议挂在：`/api/admin/pet/gacha/*` 或 `/api/admin/gacha/*`。
> 若你们已有统一的 admin 模块路由风格，可按实际调整。

### 1) 获取配置

- **GET** `/api/admin/pet/gacha/config`
- 返回：200
  - `data`: `GachaPoolConfig`

### 2) 保存（全量覆盖）

- **POST** `/api/admin/pet/gacha/config`
- Body：`GachaPoolConfig`
- 校验失败：400
  - 推荐错误信息：`rarity_weights_sum_must_equal_1`

#### 失败示例（概率和不为 1）

- Request（示例）：
  ```json
  {
    "enabled": true,
    "base_cost": 500,
    "rarity_weights": {
      "C": 0.4,
      "B": 0.3,
      "A": 0.15,
      "S": 0.1,
      "SS": 0.04,
      "SSS": 0.02
    }
  }
  ```

- Response（示例）：
  ```json
  {
    "success": false,
    "message": "rarity_weights 概率累加必须等于 1，当前 sum=1.01"
  }
  ```

### 3) 重置为默认配置（可选）

- **POST** `/api/admin/pet/gacha/config/reset`
- 返回：200
  - `data`: `GachaPoolConfig`

---

## 默认推荐值（与截图一致）

- `base_cost`: 500
- `enabled`: true
- `rarity_weights`：
  - C: 0.40
  - B: 0.30
  - A: 0.15
  - S: 0.10
  - SS: 0.04
  - SSS: 0.01

---

## 用户开蛋时“概率依据”的口径说明

用户侧每次开蛋的抽取概率应**直接依据**本配置中的 `rarity_weights`：

1) **先抽稀有度（Rarity）**

- 按 `rarity_weights` 对稀有度 `C/B/A/S/SS/SSS` 进行一次随机抽样。
- 因此某稀有度被抽中的概率 = `rarity_weights[rarity]`。

2) **再在该稀有度内抽取具体龟种（PetDefinition）**

- 候选集合 = 所有满足：
  - `PetDefinition.rarity == 抽中的稀有度`
  - 且 `PetDefinition.obtainable_by_egg == true`
  - 且（可选）`PetDefinition.enabled == true`（是否要求 enabled 由产品口径决定，但建议抽取时也约束）

- 稀有度内分配口径：
  - **默认：均分**（候选集合内等概率）
  - 若未来支持“稀有度内权重”，则在该稀有度内按权重二次抽样（本文档先按“均分”描述）。

3) 边界与强约束（推荐在保存/发布时校验）

- 若某稀有度 `rarity_weights[x] > 0`，则该稀有度必须至少存在 1 只可抽取的龟种（`obtainable_by_egg=true` 且稀有度匹配）；否则用户开蛋时可能出现“抽中稀有度但无候选”的运行时错误。
- 若某稀有度无候选，建议有两种处理策略（二选一，避免线上随机失败）：
  - A. **保存失败**：在运营侧保存配置时直接拒绝（推荐）
  - B. **运行时降级**：把该稀有度概率临时归零并重新归一化（不推荐，会导致口径漂移且难审计）
