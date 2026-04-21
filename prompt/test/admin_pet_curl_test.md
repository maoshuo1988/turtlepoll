# Admin Pet（宠物运营侧接口）curl 访问测试

本文档基于 `docs/api/pet.md` 中「宠物（Pet）运营侧接口（Admin Pet）」章节，提供一套可复制的 curl 命令，用于对章节内接口做手工访问测试/回归。

> 约定：项目统一 JSON 包装返回：成功 `{"success":true,"data":...}`；失败 `{"success":false,"message":"..."}`。

## 0. 测试前准备

### 0.1 环境变量

```bash
export BASE_URL="http://127.0.0.1:8082"
export ADMIN_TOKEN="<YOUR_ADMIN_TOKEN>"
```

### 0.2 通用 Header

```bash
export ADMIN_AUTH_HEADER="Authorization: Bearer ${ADMIN_TOKEN}"
```

> 如果你的项目实际使用的是 `X-Token` / `X-Auth-Token` / Cookie 等方式，把下面示例里的 header 换一下即可。

---

## 1) PetDefinition（龟种定义）

### 1.1 列表：GET /api/admin/pet/defs

基础列表（分页）：

```bash
curl -sS "${BASE_URL}/api/admin/pet/defs?page=1&size=20" \
  -H "${ADMIN_AUTH_HEADER}"
```

按 enabled 过滤：

```bash
curl -sS "${BASE_URL}/api/admin/pet/defs?enabled=true&page=1&size=20" \
  -H "${ADMIN_AUTH_HEADER}" | jq
```

按 rarity 过滤：

```bash
# 注意：当前后端实现的 rarity 过滤参数使用数字（1..6），对应 C/B/A/S/SS/SSS。
# 例如：S=4
curl -sS "${BASE_URL}/api/admin/pet/defs?rarity=4&page=1&size=20" \
  -H "${ADMIN_AUTH_HEADER}" | jq
```

### 1.2 详情：GET /api/admin/pet/defs/:petId

```bash
export PET_ID="lava"

curl -sS "${BASE_URL}/api/admin/pet/defs/${PET_ID}" \
  -H "${ADMIN_AUTH_HEADER}" | jq
```

404 验证：

```bash
curl -sS "${BASE_URL}/api/admin/pet/defs/not_exists_123" \
  -H "${ADMIN_AUTH_HEADER}" | jq
```

### 1.3 创建/更新（Upsert）：POST /api/admin/pet/defs

说明：当前后端实现对 Upsert 同时兼容 `pet_id`（推荐）与 `petKey`（兼容旧客户端）。

创建/更新一个示例龟种：

```bash
curl -sS "${BASE_URL}/api/admin/pet/defs" \
  -H "${ADMIN_AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d '{
  "pet_id": "lava",
  "petKey": "lava",
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
}' | jq
```

### 1.4 删除/下架：DELETE /api/admin/pet/defs/:petId

```bash
export PET_ID="lava"

curl -sS -X DELETE "${BASE_URL}/api/admin/pet/defs/${PET_ID}" \
  -H "${ADMIN_AUTH_HEADER}" -i
```

---

## 2) 紧急开关：POST /api/admin/pet/kill-switch

关闭开蛋池示例：

```bash
curl -sS "${BASE_URL}/api/admin/pet/kill-switch" \
  -H "${ADMIN_AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d '{"action":"disable_pool","scope":"global","reason":"emergency: exploit"}' | jq
```

> `action/scope` 的取值以服务端实现为准；此处仅按契约文档示例。

---

## 3) FeatureCatalog（特性模板库）

> 本节接口与 `prompt/test/feature_catalog_curl_test.md` 内容有重叠；这里放一份“全章节一体化”的版本，便于一次性回归。

### 3.1 列表：GET /api/admin/pet/features

```bash
curl -sS "${BASE_URL}/api/admin/pet/features?page=1&size=20" \
  -H "${ADMIN_AUTH_HEADER}" | jq
```

过滤 enabled / scope：

```bash
curl -sS "${BASE_URL}/api/admin/pet/features?enabled=true&scope=PET&page=1&size=20" \
  -H "${ADMIN_AUTH_HEADER}" | jq
```

模糊搜索 q：

```bash
curl -sS "${BASE_URL}/api/admin/pet/features?q=signin&page=1&size=20" \
  -H "${ADMIN_AUTH_HEADER}" | jq
```

### 3.2 详情：GET /api/admin/pet/features/:featureKey

```bash
export FEATURE_KEY="spark_multiplier"

curl -sS "${BASE_URL}/api/admin/pet/features/${FEATURE_KEY}" \
  -H "${ADMIN_AUTH_HEADER}" | jq
```

### 3.3 创建/更新（Upsert）：POST /api/admin/pet/features

```bash
curl -sS "${BASE_URL}/api/admin/pet/features" \
  -H "${ADMIN_AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d '{
  "feature_key": "spark_multiplier",
  "name": { "zh-CN": "火花倍率", "en-US": "Spark Multiplier" },
  "scope": "PET",
  "effective_event": "DAILY_SIGNIN",
  "params_schema": {
    "type": "object",
    "required": ["enabled", "base", "per_level", "cap"],
    "properties": {
      "enabled": {"type": "boolean"},
      "base": {"type": "number"},
      "per_level": {"type": "number"},
      "cap": {"type": "integer"}
    }
  },
  "enabled": true
}' | jq
```

### 3.4 删除：DELETE /api/admin/pet/features/:featureKey

```bash
export FEATURE_KEY="spark_multiplier"

curl -sS -X DELETE "${BASE_URL}/api/admin/pet/features/${FEATURE_KEY}" \
  -H "${ADMIN_AUTH_HEADER}" -i
```

---

## 4) abilities（给龟种配置特性）

### 4.1 整体替换：PUT /api/admin/pet/defs/:petId/abilities

```bash
export PET_ID="lava"

curl -sS -X PUT "${BASE_URL}/api/admin/pet/defs/${PET_ID}/abilities" \
  -H "${ADMIN_AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d '{
    "abilities": {
      "spark_multiplier": {"enabled": true, "base": 1.3, "per_level": 0.03, "cap": 400},
      "signin_bonus": {"enabled": true, "base_amount": 100, "level_step": 10, "daily_cap": 500}
    }
  }' | jq
```

### 4.2 单项更新/新增：PATCH /api/admin/pet/defs/:petId/abilities/:featureKey

说明：当前后端实现的 PATCH body 只接收 `params`（不支持顶层 `enabled`）。

启用并更新 params（以 spark_multiplier 为例）：

```bash
export PET_ID="lava"
export FEATURE_KEY="spark_multiplier"

curl -sS -X PATCH "${BASE_URL}/api/admin/pet/defs/${PET_ID}/abilities/${FEATURE_KEY}" \
  -H "${ADMIN_AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d '{"params":{"base":1.3,"per_level":0.03,"cap":400}}' | jq
```

禁用（不阻塞保留 params 的实现）：

```bash
curl -sS -X PATCH "${BASE_URL}/api/admin/pet/defs/${PET_ID}/abilities/${FEATURE_KEY}" \
  -H "${ADMIN_AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d '{"params":{"enabled":false}}' | jq
```

#### 4.2.1 更多能力示例（复制即用）

signin_bonus：

```bash
export PET_ID="lava"
export FEATURE_KEY="signin_bonus"

curl -sS -X PATCH "${BASE_URL}/api/admin/pet/defs/${PET_ID}/abilities/${FEATURE_KEY}" \
  -H "${ADMIN_AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d '{"params":{"enabled":true,"base_amount":100,"level_step":10,"daily_cap":500}}' | jq
```

debt：

```bash
export PET_ID="lightning"
export FEATURE_KEY="debt"

curl -sS -X PATCH "${BASE_URL}/api/admin/pet/defs/${PET_ID}/abilities/${FEATURE_KEY}" \
  -H "${ADMIN_AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d '{"params":{"enabled":true,"debtFloor":-300,"forbidEquipWhenDebt":true,"errorCode":"DEBT_UNPAID"}}' | jq
```

### 4.3 单项移除：DELETE /api/admin/pet/defs/:petId/abilities/:featureKey

```bash
export PET_ID="lava"
export FEATURE_KEY="spark_multiplier"

curl -sS -X DELETE "${BASE_URL}/api/admin/pet/defs/${PET_ID}/abilities/${FEATURE_KEY}" \
  -H "${ADMIN_AUTH_HEADER}" -i
```

---

## 5) 最小回归序列（建议按顺序执行）

1) 创建/更新 FeatureCatalog：`POST /api/admin/pet/features`
2) 创建/更新 PetDefinition：`POST /api/admin/pet/defs`
3) 替换 abilities：`PUT /api/admin/pet/defs/:petId/abilities`
4) abilities 单项 patch + 禁用：`PATCH ...`
5) 查询 defs 列表/详情确认返回：`GET /api/admin/pet/defs` + `GET /api/admin/pet/defs/:petId`
6) （可选）触发 kill-switch：`POST /api/admin/pet/kill-switch`
7) 清理：删除 abilities / defs / feature（按你的环境策略决定是否软删）

---

## 常见问题排查

- 401/403：检查 token 是否为管理员、header 名是否匹配服务端鉴权实现。
- 400：一般是字段校验失败（特别是 `rarity/scope/effective_event/params_schema`）。把返回 `message` 打印出来对齐文档约束。
- 404：确认 `petId/featureKey` 是否存在；注意大小写与下划线。
- `jq: command not found`：可以先不管 `| jq`，或在系统里安装 jq。
