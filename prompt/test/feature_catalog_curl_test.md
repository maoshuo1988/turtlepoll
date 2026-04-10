# FeatureCatalog（特性模板库）接口访问测试（curl）

> 来源：`docs/api/pet.md` → 「接口：特性模板库（FeatureCatalog）」章节。
>
> 目标：提供一套“可复制粘贴”的 curl 命令，用于手工冒烟测试 FeatureCatalog 的增删改查。
>
> 说明：本项目接口返回使用统一 JSON 包装：
>
> - 成功：`{"success":true,"data":...}`
> - 失败：`{"success":false,"message":"..."}`

---

## 使用前准备

把下面变量替换成你自己的环境：

- `BASE_URL`：服务地址（例如 `http://localhost:8082`）
- `ADMIN_TOKEN`：管理员 token（接口前缀 `/api/admin` 一般需要管理员权限）

建议先在当前 shell 里导出：

```bash
export BASE_URL="http://localhost:8082"
export ADMIN_TOKEN="<YOUR_ADMIN_TOKEN>"
```

后续命令都假设你已设置了这两个环境变量。

---

## 0) Health check（可选）

确认服务可访问（不一定需要登录）：

```bash
curl -sS "$BASE_URL/" | head
```

---

## 1) GET /api/admin/pet/features（列表）

### 1.1 默认分页（不带过滤）

```bash
curl -sS "$BASE_URL/api/admin/pet/features?page=1&size=20" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### 1.2 过滤：enabled=true

```bash
curl -sS "$BASE_URL/api/admin/pet/features?page=1&size=50&enabled=true" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### 1.3 过滤：scope=PET（可选值：PET|GLOBAL）

```bash
curl -sS "$BASE_URL/api/admin/pet/features?page=1&size=50&scope=PET" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### 1.4 模糊搜索：q=signin

```bash
curl -sS "$BASE_URL/api/admin/pet/features?page=1&size=50&q=signin" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

---

## 2) GET /api/admin/pet/features/:featureKey（详情）

以 `signin_bonus` 为例（请替换为真实已存在的 feature_key）：

```bash
curl -sS "$BASE_URL/api/admin/pet/features/signin_bonus" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

---

## 3) POST /api/admin/pet/features（创建或更新：幂等 upsert）

> 说明：文档约定：若 `feature_key` 存在则更新，否则创建。
>
> 注意：以下示例的 schema 是“示例 shape”，具体强校验规则以服务端实现为准。

### 3.1 创建/更新：first_bet_bonus（示例）

```bash
curl -sS -X POST "$BASE_URL/api/admin/pet/features" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "featureKey": "first_bet_bonus",
    "name": {"zh-CN": "首次下注奖励", "en-US": "First Bet Bonus"},
    "scope": "PET",
    "effectiveEvent": "BET_SETTLE",
    "paramsSchema": {
      "type": "object",
      "required": ["bonusCoins"],
      "properties": {
        "bonusCoins": {"type": "integer", "minimum": 0},
        "levelScale": {"type": "number", "minimum": 0},
        "capPerDay": {"type": "integer", "minimum": 0},
        "minBetAmount": {"type": "integer", "minimum": 0}
      }
    },
    "enabled": true
  }'
```

### 3.2 更新：禁用该 featureKey（enabled=false）

```bash
curl -sS -X POST "$BASE_URL/api/admin/pet/features" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "featureKey": "first_bet_bonus",
    "name": {"zh-CN": "首次下注奖励", "en-US": "First Bet Bonus"},
    "scope": "PET",
    "effectiveEvent": "BET_SETTLE",
    "paramsSchema": {"type": "object", "required": ["bonusCoins"], "properties": {"bonusCoins": {"type": "integer", "minimum": 0}}},
    "enabled": false
  }'
```

---

## 4) DELETE /api/admin/pet/features/:featureKey（删除/下架）

> 文档建议：软删 `enabled=false`；但接口形态是 DELETE。

```bash
curl -sS -X DELETE "$BASE_URL/api/admin/pet/features/first_bet_bonus" \
  -H "Authorization: Bearer $ADMIN_TOKEN" -i
```

---

## 常见排查

### 1) 401/403

- token 不是管理员
- 或服务端中间件要求额外 cookie/header

### 2) 400（校验失败）

- `featureKey` 格式不合法（需小写字母/下划线/数字）
- `scope` / `effectiveEvent` 枚举值不合法
- `paramsSchema` 不是 object 或缺 required/properties

### 3) 404

- featureKey 不存在

---

## 最小回归脚本（按顺序执行）

> 这个顺序用于快速验证：创建 -> 查询 -> 列表过滤 -> 删除。

```bash
# 1) upsert
curl -sS -X POST "$BASE_URL/api/admin/pet/features" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"featureKey":"first_bet_bonus","name":{"zh-CN":"首次下注奖励"},"scope":"PET","effectiveEvent":"BET_SETTLE","paramsSchema":{"type":"object","required":["bonusCoins"],"properties":{"bonusCoins":{"type":"integer","minimum":0}}},"enabled":true}'

# 2) get
curl -sS "$BASE_URL/api/admin/pet/features/first_bet_bonus" \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# 3) list
curl -sS "$BASE_URL/api/admin/pet/features?page=1&size=20&q=first_bet" \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# 4) delete
curl -sS -X DELETE "$BASE_URL/api/admin/pet/features/first_bet_bonus" \
  -H "Authorization: Bearer $ADMIN_TOKEN" -i
```
