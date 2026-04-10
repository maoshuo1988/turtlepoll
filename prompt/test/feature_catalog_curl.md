# FeatureCatalog（特性模板库）接口 curl 访问测试

> 覆盖 `docs/api/pet.md` 中章节「接口：特性模板库（FeatureCatalog）」的 4 个接口：
>
> - `GET /api/admin/pet/features`
> - `GET /api/admin/pet/features/:featureKey`
> - `POST /api/admin/pet/features`
> - `DELETE /api/admin/pet/features/:featureKey`
>
> 说明：这些接口挂在 `/api/admin` 下，通常需要管理员权限（`Authorization: Bearer <ADMIN_TOKEN>`）。

---

## 0) 通用变量（建议先在终端里 export）

```bash
export BASE_URL="http://localhost:8082"
export ADMIN_TOKEN="<YOUR_ADMIN_TOKEN>"
```

下面命令默认使用：`$BASE_URL` + `Authorization: Bearer $ADMIN_TOKEN`。

---

## 1) FeatureCatalog 列表

### 1.1 列表（默认）

```bash
curl -sS "${BASE_URL}/api/admin/pet/features" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Accept: application/json"
```

### 1.2 按 enabled 过滤

```bash
curl -sS "${BASE_URL}/api/admin/pet/features?enabled=1" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Accept: application/json"
```

### 1.3 按 scope 过滤（PET / GLOBAL）

```bash
curl -sS "${BASE_URL}/api/admin/pet/features?scope=PET" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Accept: application/json"

curl -sS "${BASE_URL}/api/admin/pet/features?scope=GLOBAL" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Accept: application/json"
```

### 1.4 模糊搜索（q 匹配 feature_key / 名称）

```bash
curl -sS "${BASE_URL}/api/admin/pet/features?q=signin" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Accept: application/json"
```

### 1.5 分页（page/size）

```bash
curl -sS "${BASE_URL}/api/admin/pet/features?page=1&size=20" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Accept: application/json"
```

---

## 2) 获取单个 FeatureCatalogItem

> 先用列表接口拿到一个 `featureKey`，然后替换掉下面示例中的 `signin_bonus`。

```bash
curl -sS "${BASE_URL}/api/admin/pet/features/signin_bonus" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Accept: application/json"
```

---

## 3) 创建/更新 FeatureCatalogItem（幂等 upsert）

> 注意：文档字段是 snake_case（`feature_key` / `effective_event` / `params_schema`），而代码实现里可能使用 camelCase 或结构体标签；
> 若你发现 400/字段不识别，以实际后端实现为准（本文件先按 `docs/api/pet.md` 的契约写）。

### 3.1 创建或更新：first_bet_bonus

```bash
curl -sS "${BASE_URL}/api/admin/pet/features" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  --data-raw ' {
    "feature_key": "first_bet_bonus",
    "name": {"zh-CN": "首次下注奖励", "en-US": "First Bet Bonus"},
    "scope": "PET",
    "effective_event": "BET_SETTLE",
    "params_schema": {
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
  } '
```

### 3.2 创建或更新：signin_bonus（示例）

```bash
curl -sS "${BASE_URL}/api/admin/pet/features" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  --data-raw ' {
    "feature_key": "signin_bonus",
    "name": {"zh-CN": "每日登录加成", "en-US": "Signin Bonus"},
    "scope": "PET",
    "effective_event": "DAILY_SIGNIN",
    "params_schema": {
      "type": "object",
      "required": ["bonusCoins"],
      "properties": {
        "bonusCoins": {"type": "integer", "minimum": 0}
      }
    },
    "enabled": true
  } '
```

---

## 4) 删除 FeatureCatalogItem

> 建议在测试环境先把你刚创建的 `first_bet_bonus` 删除掉，避免污染测试数据。

```bash
curl -sS -X DELETE "${BASE_URL}/api/admin/pet/features/first_bet_bonus" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -H "Accept: application/json"
```

---

## 5) 快速排错

- **401/未登录**：token 不对，或接口实际需要 cookie（看你们 AuthMiddleware 的实现）。
- **403/无权限**：需要管理员账号。
- **400/字段校验失败**：以服务端返回 message 为准；必要时把实际后端字段名（snake/camel）对齐。
- **504/超时**：服务没启动或端口不对。
