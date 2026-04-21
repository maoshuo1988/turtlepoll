# 宠物（Pet）运营侧接口（Admin Pet）

说明：本文件记录运营侧对“龟种（Pet）”相关的接口契约；**不对外暴露配置版本、快照、回滚、灰度/生效策略等概念与接口**。对外仅提供：龟种定义的增删改查，以及紧急止血开关。

## 设计原则（简要）

- 不在公开 API 层暴露配置版本/快照/回滚接口（由内部实现负责历史留存与回滚策略）。
- 运营对龟种的变更必须做强校验（概率/价格/返还/可抽性），失败返回 400 并包含明确错误码与说明。
- 每次变更应写入 operate-log（由后端实现负责）；开蛋/结算时订单应记录当时读取的配置标识（内部字段）。

---

## 资源：PetDefinition（龟种定义）

说明：运营维护的单个龟种定义用于展示、开蛋池抽取与能力挂载。后端在开蛋/结算时读取该资源决定行为。

字段清单（详细说明）：

- `pet_id` (string, required)
	- 含义：内部唯一标识（不可变）。例：`basic`, `lava`, `space`。
	- 约束：只允许小写字母/下划线/数字；创建后不可修改。

- `name` (object, required)
	- 含义：多语言显示名称。
	- 结构：{ "zh-CN": "火山龟", "en-US": "Lava Turtle" }
	- 约束：至少包含一门语言；长度 <= 64

- `rarity` (string, required)
	- 含义：稀有度档位，用于抽取概率映射。可取值：`C`,`B`,`A`,`S`,`SS`,`SSS`。
	- 校验：必须是预定义枚举之一。

- `enabled` (boolean, required)
	- 含义：运营是否允许该龟种在产品侧被展示/出现在开蛋池中（展示与可抽两者受此开关控制，具体实现可分离）。

- `obtainable_by_egg` (boolean, required)
	- 含义：是否可通过开蛋获得（true：可被抽到；false：只能通过活动/发放/兑换获得）。

- `display` (object, optional)
	- 含义：展示资源与前端渲染信息。
	- 结构示例：
		- `icon` (string) — 小图标资源路径或资源 id
		- `cover` (string) — 封面/大图资源
		- `thumbnail` (string) — 列表缩略图（可选）
	- 约束：资源 id 长度 <= 256

- `description` (object, optional)
	- 含义：多语言描述文本，用于详情页。
	- 结构示例：{ "zh-CN": "每日登录加成龟", "en-US": "Daily bonus turtle" }
	- 约束：每个语言文本长度 <= 1000

- `abilities` (object, optional)
	- 含义：能力/特性集合（能力由 featureKey 指定并使用参数化配置）。
	- 结构说明（示例）：
		```json
		{
			"signin_bonus": { "enabled": true, "base_amount": 100, "level_step": 10, "daily_cap": 500 },
			"spark_multiplier": { "enabled": true, "base": 1.3, "per_level": 0.03, "cap": 400 }
		}
		```
	- 约定：abilities 的键必须来自运营侧统一的 featureKey 白名单；每个 ability 的参数结构需后端校验。

- `pricing` (object, optional)
	- 含义：与开蛋/商城相关的定价与折扣信息。
	- 结构示例：
		- `egg_price` (integer) — 默认开蛋价格（单位：系统币），如 500
		- `egg_discount` (object|null) — 若该龟种享有开蛋折扣（由运营配置），格式：{ "type": "rate|fixed", "value": 0.05 }
	- 校验：egg_price >= 0；若 discount.type=="rate" 则 0 <= value < 1

- `metadata` (object, read-only)
	- `created_at` (datetime)
	- `created_by` (string)
	- `updated_at` (datetime)
	- `updated_by` (string)

校验/约束（运营端提交时后端必须校验）：

- `pet_id` 唯一且格式合法。
- `rarity` 必须为定义枚举。
- 如果 `obtainable_by_egg == true`，后端在抽取/发放逻辑中应确保对应稀有度档位存在（由业务实现保证）。
- `abilities` 中每项参数需满足对应 feature 的约束（例：refundRate 在 [0,1]）。

---

## 接口设计

说明：下列接口为运营侧对 `PetDefinition` 的管理接口；对外不区分“保存但不生效/发布生效”等阶段性接口。

同时，本文件也包含“给宠物配置特性（Feature）”所需的接口：

- **特性模板库（FeatureCatalog）**：运营维护 featureKey、作用域、参数 schema（用于校验与表单渲染）。
- **龟种能力挂载（PetDefinition.abilities）**：把某个 featureKey 以参数化方式挂到某个龟种上。

### GET /api/admin/pet/defs

- 描述：列出所有龟种定义（支持分页/过滤）。
- 参数：
	- `?enabled` (bool, optional)
	- `?rarity` (string, optional)
	- `?page`/`?size`
- 返回：200 OK
	- body: { "items": [PetDefinition], "total": N }

### GET /api/admin/pet/defs/:petId

- 描述：获取单个龟种定义详情。
- 返回：200 OK 或 404

### POST /api/admin/pet/defs

- 描述：创建或更新龟种定义（幂等：若 pet_id 存在则更新，否则创建）。
- Body: PetDefinition（见上字段说明）
- 行为：
	- 服务端对 body 做强校验，失败返回 400（带错误码与说明）。
	- 校验通过后保存记录（并写 operate-log）。
	- 返回：200 OK（含 `id`/metadata）或 201 Created（创建时）

### DELETE /api/admin/pet/defs/:petId

- 描述：下架或删除某个龟种定义（建议软删，设置 `enabled=false`）。
- 返回：204 No Content 或 404

### POST /api/admin/pet/kill-switch

- 描述：紧急开关，支持直接关闭开蛋入口或单项能力。示例 body：

```json
{ "action": "disable_pool", "scope": "global", "reason": "emergency: exploit" }
```

- 返回：200 OK

- 返回：200 OK

---

## 资源：FeatureCatalogItem（特性模板）

说明：FeatureCatalog 是运营侧“特性/能力”的统一注册中心。

- 一方面提供 **featureKey 白名单**；避免 `abilities` 填任意 key。
- 另一方面提供 **参数 schema**；用于服务端强校验，也便于后台动态生成表单。

字段清单（建议）：

- `feature_key` (string, required)
	- 含义：特性唯一键（不可变）。例：`signin_bonus`、`spark_multiplier`。
	- 约束：小写字母/下划线/数字；创建后不可修改。

- `name` (object, required)
	- 含义：多语言中文/英文名称。
	- 结构：{ "zh-CN": "每日登录加成", "en-US": "Signin Bonus" }

- `scope` (string, required)
	- 含义：作用域。
	- 可取值（建议枚举）：
		- `PET`：单龟种能力（挂到某个 pet 上）
		- `GLOBAL`：全局规则（不依赖某个 pet；也可以通过 `pet/kill-switch` 紧急禁用）

- `effective_event` (string, required)
	- 含义：生效时机（仅描述业务归属，不代表“版本/发布接口”）。
	- 可取值（建议枚举）：
		- `DAILY_SIGNIN`
		- `EGG_PURCHASE`
		- `EGG_RESOLVE`
		- `BET_SETTLE`
		- `CHAT_STAMINA`
		- `MINIGAME`

- `params_schema` (object, required)
	- 含义：参数 schema（用于校验与 UI 表单）。建议兼容 JSON Schema 子集。
	- 约束：必须包含 type/object/required/properties。

- `enabled` (boolean, required)
	- 含义：该 featureKey 是否允许在运营侧被使用。禁用后：
		- 不影响已保存的 pet ability（实现可选择拒绝保存 / 容忍但不执行）；
		- 新增/修改时应返回 400。

- `metadata` (object, read-only)
	- `created_at` (datetime)
	- `created_by` (string)
	- `updated_at` (datetime)
	- `updated_by` (string)

---

## 接口：特性模板库（FeatureCatalog）

### GET /api/admin/pet/features

- 描述：列出特性模板（支持分页/过滤）。
- 参数：
	- `?enabled` (bool, optional)
	- `?scope` (string, optional) — `PET|GLOBAL`
	- `?q` (string, optional) — 模糊搜索 feature_key / 名称
	- `?page`/`?size`
- 返回：200 OK
	- body: { "items": [FeatureCatalogItem], "total": N }

### GET /api/admin/pet/features/:featureKey

- 描述：获取单个特性模板详情。
- 返回：200 OK 或 404

### POST /api/admin/pet/features

- 描述：创建或更新特性模板（幂等：若 feature_key 存在则更新，否则创建）。
- Body: FeatureCatalogItem（见上字段说明）
- 行为：
	- 服务端校验：feature_key 格式、scope 枚举、params_schema 合法性。
	- 返回：200 OK 或 201 Created

### DELETE /api/admin/pet/features/:featureKey

- 描述：删除特性模板（建议软删：`enabled=false`，避免历史配置无法解释）。
- 返回：204 No Content 或 404

---

## 接口：给龟种配置特性（abilities）

说明：`PetDefinition.abilities` 是一个 dict：`featureKey -> params`。

建议后端做两层校验：

1) `featureKey` 必须存在于 FeatureCatalog 且 `enabled=true`
2) `params` 必须满足该 feature 的 `params_schema`

### PUT /api/admin/pet/defs/:petId/abilities

- 描述：整体替换某个龟种的 abilities（推荐用于“保存整页表单”）。
- Body:
	- `abilities` (object, required)
		- 结构：{ "signin_bonus": { ... }, "spark_multiplier": { ... } }
- 行为：
	- 校验通过后写 operate-log。
- 返回：200 OK（返回更新后的 PetDefinition）

### PATCH /api/admin/pet/defs/:petId/abilities/:featureKey

- 描述：更新/新增单个特性到某龟种（推荐用于“单行编辑/弹窗编辑”）。
- Body:
	- `params` (object, required)
- 行为：
	- 约定：是否启用由 `params.enabled` 控制。
- 返回：200 OK

### DELETE /api/admin/pet/defs/:petId/abilities/:featureKey

- 描述：从某龟种移除某个特性（等价于彻底删除该 key）。
- 返回：204 No Content 或 404

---

## 常用 abilities 入参与示例（附 curl）

说明：下列能力参数是“当前产品侧约定的常用 key”，用于运营配置/后端校验对齐。

- 写入方式 A（整页保存）：`PUT /api/admin/pet/defs/:petId/abilities`（一次性提交 `abilities` 整体对象）
- 写入方式 B（单项调整）：`PATCH /api/admin/pet/defs/:petId/abilities/:featureKey`（提交单个 ability 的 `params`）

下面每个能力都给出：参数说明 + JSON 示例 + PATCH curl 示例。

### 1) signin_bonus（每日登录加成）

params 示例：

```json
{
	"enabled": true,
	"base_amount": 100,
	"level_step": 10,
	"daily_cap": 500
}
```

- `enabled` (bool)
- `base_amount` (int) 基础发放金币
- `level_step` (int) 等级加成的步进（示例口径：每级 +level_step）
- `daily_cap` (int) 每日封顶

PATCH curl：

```bash
export PET_ID="lava"
export FEATURE_KEY="signin_bonus"

curl -sS -X PATCH "${BASE_URL}/api/admin/pet/defs/${PET_ID}/abilities/${FEATURE_KEY}" \
	-H "${ADMIN_AUTH_HEADER}" \
	-H "Content-Type: application/json" \
	-d '{
		"params": {"enabled": true, "base_amount": 100, "level_step": 10, "daily_cap": 500}
	}' | jq
```

### 2) spark_multiplier（火花倍率加成）

params 示例：

```json
{
	"enabled": true,
	"base": 1.3,
	"per_level": 0.03,
	"cap": 400
}
```

- `enabled` (bool)
- `base` (number) 基础倍率
- `per_level` (number) 每级提升倍率
- `cap` (int) 上限（示例：某个内部计算的封顶）

PATCH curl：

```bash
export PET_ID="lava"
export FEATURE_KEY="spark_multiplier"

curl -sS -X PATCH "${BASE_URL}/api/admin/pet/defs/${PET_ID}/abilities/${FEATURE_KEY}" \
	-H "${ADMIN_AUTH_HEADER}" \
	-H "Content-Type: application/json" \
	-d '{
		"params": {"enabled": true, "base": 1.3, "per_level": 0.03, "cap": 400}
	}' | jq
```

### 3) debt（欠账能力：允许余额为负）

params 示例：

```json
{
	"enabled": true,
	"debtFloor": -300,
	"forbidEquipWhenDebt": true,
	"errorCode": "DEBT_UNPAID"
}
```

- `enabled` (bool)
- `debtFloor` (int, <=0) 最低余额（例如 -300）
- `forbidEquipWhenDebt` (bool) 是否启用“欠账禁止切龟”
- `errorCode` (string) 装备被拒绝时的错误码（默认 `DEBT_UNPAID`）

PATCH curl：

```bash
export PET_ID="lightning"
export FEATURE_KEY="debt"

curl -sS -X PATCH "${BASE_URL}/api/admin/pet/defs/${PET_ID}/abilities/${FEATURE_KEY}" \
	-H "${ADMIN_AUTH_HEADER}" \
	-H "Content-Type: application/json" \
	-d '{
		"params": {"enabled": true, "debtFloor": -300, "forbidEquipWhenDebt": true, "errorCode": "DEBT_UNPAID"}
	}' | jq
```

### 4) debt_subsidy（欠款补贴）

params 示例：

```json
{
	"enabled": true,
	"subsidyRate": 0.2,
	"capPerDay": 500,
	"rounding": "floor"
}
```

- `enabled` (bool)
- `subsidyRate` (number, 0..1) 补贴比例
- `capPerDay` (int, optional) 每日补贴封顶
- `rounding` (string) 取整方式（示例：`floor`）

PATCH curl：

```bash
export PET_ID="lightning"
export FEATURE_KEY="debt_subsidy"

curl -sS -X PATCH "${BASE_URL}/api/admin/pet/defs/${PET_ID}/abilities/${FEATURE_KEY}" \
	-H "${ADMIN_AUTH_HEADER}" \
	-H "Content-Type: application/json" \
	-d '{
		"params": {"enabled": true, "subsidyRate": 0.2, "capPerDay": 500, "rounding": "floor"}
	}' | jq
```

### 5) debt_lock（欠账锁龟标记/联动）

params 示例：

```json
{
	"enabled": true,
	"lock_switch_when_debt": true
}
```

- `enabled` (bool)
- `lock_switch_when_debt` (bool) 是否在欠账时锁定切换

PATCH curl：

```bash
export PET_ID="lightning"
export FEATURE_KEY="debt_lock"

curl -sS -X PATCH "${BASE_URL}/api/admin/pet/defs/${PET_ID}/abilities/${FEATURE_KEY}" \
	-H "${ADMIN_AUTH_HEADER}" \
	-H "Content-Type: application/json" \
	-d '{
		"params": {"enabled": true, "lock_switch_when_debt": true}
	}' | jq
```

### 6) egg_discount（开蛋折扣，按龟种生效）

params 示例（折扣率）：

```json
{
	"enabled": true,
	"discountRate": 0.05,
	"rounding": "floor"
}
```

- `enabled` (bool)
- `discountRate` (number, 0..1) 折扣率（示例 0.05 表示 95 折）
- `rounding` (string) 取整方式（示例：`floor`）

PATCH curl：

```bash
export PET_ID="ninja"
export FEATURE_KEY="egg_discount"

curl -sS -X PATCH "${BASE_URL}/api/admin/pet/defs/${PET_ID}/abilities/${FEATURE_KEY}" \
	-H "${ADMIN_AUTH_HEADER}" \
	-H "Content-Type: application/json" \
	-d '{
		"params": {"enabled": true, "discountRate": 0.05, "rounding": "floor"}
	}' | jq
```

### 7) egg_duplicate_refund（重复返还，全局规则口径）

params 示例：

```json
{
	"enabled": true,
	"refundRate": 0.3,
	"rounding": "floor"
}
```

- `enabled` (bool)
- `refundRate` (number, 0..1) 返还比例（示例：0.3）
- `rounding` (string) 取整方式（示例：`floor`）

PATCH curl：

```bash
export PET_ID="basic"
export FEATURE_KEY="egg_duplicate_refund"

curl -sS -X PATCH "${BASE_URL}/api/admin/pet/defs/${PET_ID}/abilities/${FEATURE_KEY}" \
	-H "${ADMIN_AUTH_HEADER}" \
	-H "Content-Type: application/json" \
	-d '{
		"params": {"enabled": true, "refundRate": 0.3, "rounding": "floor"}
	}' | jq
```

> 备注：上面示例以“挂在某个 pet 上”的方式演示；若要作为全局规则生效，建议最终落一个 GLOBAL scope 的配置入口。

## 例子（完整请求/响应示例）

POST /api/admin/pet/defs

Request JSON:

```json
{
	"pet_id": "lava",
	"name": { "zh-CN": "熔岩龟", "en-US": "Lava Turtle" },
	"rarity": "S",
	"enabled": true,
	"obtainable_by_egg": true,
	"display": { "icon": "pet:lava:icon_v2", "cover": "pet:lava:cover" },
	"description": { "zh-CN": "每日火花倍率加成", "en-US": "Spark multiplier" },
	"abilities": {
		"spark_multiplier": { "enabled": true, "base": 1.3, "per_level": 0.03, "cap": 400 }
	},
	"pricing": { "egg_price": 500 }
}
```

Success response (200):

```json
{
	"pet_id": "lava",
	"created_at": "2026-04-08T10:00:00Z",
	"updated_at": "2026-04-08T10:00:00Z",
	"updated_by": "operator_1"
}
```

---

如果你愿意，我可以把上面的 `abilities` 字段里常用的 featureKey（例如 `signin_bonus`, `spark_multiplier`, `debt`, `egg_duplicate_refund` 等）也写成 schema 例子并放到同一文档下，便于后端实现与前端对齐。

> 本文档用于“宠物系统”的运营侧接口设计（仅接口契约，暂不实现）。
>
> 相关文档：
>
> - 运营侧流程与口径：`prompt/project/宠物-运营侧宠物维护.md`
> - 用户侧开蛋流程：`prompt/project/宠物-用户侧宠物使用.md`（「开蛋流程」）

---

## 统一响应与错误约定

项目统一使用 JSON 包装返回（见 `github.com/mlogclub/simple/web`）：

- 成功：`{"success":true,"data":...}`
- 失败：`{"success":false,"message":"..."}`

> 备注：当前项目多数接口以 `message` 文案区分错误类型；如需稳定错误码（`code`），后续可统一扩展。

---

---

## 说明（非接口契约）


- 配置的“是否立即对线上生效”、以及内部如何留痕/审计/回滚，均属于后端实现细节，不作为对外接口的一部分。
