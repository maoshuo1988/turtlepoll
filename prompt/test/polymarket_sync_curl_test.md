# Polymarket 预测市场同步接口 curl 测试

本文用于手工测试 Polymarket 预测市场同步链路：登录 -> Discovery -> Tracking -> 查询结果。

## 1. 准备变量

```bash
BASE_URL="http://127.0.0.1:8082"
USERNAME="你的管理员用户名或邮箱"
PASSWORD="你的密码"
```

## 2. 登录并获取 token

> 说明：项目支持以下鉴权方式：
> - `Authorization: Bearer <token>`
> - `X-User-Token: <token>`
> - `Cookie: bbsgo_token=<token>`

```bash
LOGIN_RESP=$(curl -sS -X POST "$BASE_URL/api/login/signin" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "username=$USERNAME" \
  --data-urlencode "password=$PASSWORD" \
  --data-urlencode "captchaProtocol=4")

echo "$LOGIN_RESP"
```

提取 token（依赖 jq）：

```bash
TOKEN=$(echo "$LOGIN_RESP" | jq -r '.data.token // .token // empty')
echo "TOKEN=$TOKEN"
```

## 3. 触发同步（管理端）

curl -k -sS -X POST "https://52.220.192.18/api/admin/polymarket/discovery_sync" \
  -H "Authorization: Bearer ecffc9914ec942a19ac8b3af5cb7e96e"

curl -k -sS -X POST "https://52.220.192.18/api/admin/polymarket/tracking_sync" \
  -H "Authorization: Bearer ecffc9914ec942a19ac8b3af5cb7e96e"


curl -k -sS -X POST "http://127.0.0.1:8082/api/admin/polymarket/tracking_sync" \
  -H "Authorization: Bearer 749c3b8056af4499ac276758760a5a58"

### 3.1 Discovery 同步

```bash
curl -sS -X POST "$BASE_URL/api/admin/polymarket/discovery_sync" \
  -H "Authorization: Bearer $TOKEN"
```

### 3.2 Tracking 同步

```bash
curl -sS -X POST "$BASE_URL/api/admin/polymarket/tracking_sync" \
  -H "Authorization: Bearer $TOKEN"
```

## 4. 兼容旧入口（用户侧）

```bash
curl -sS -X POST "$BASE_URL/api/football/sync_polymarket" \
  -H "Authorization: Bearer $TOKEN"
```

## 5. 查询同步结果

### 5.1 查询 Polymarket 市场

```bash
curl -sS "$BASE_URL/api/football/markets?page=1&limit=20&sourceModel=Polymarket" \
  -H "Authorization: Bearer $TOKEN"
```

### 5.2 查询自动结算问题（admin）

```bash
curl -sS "$BASE_URL/api/admin/polymarket/issues?status=OPEN&page=1&limit=20" \
  -H "Authorization: Bearer $TOKEN"
```

## 6. 常见运维动作（可选）

### 6.1 重试 Tracking

```bash
curl -sS -X POST "$BASE_URL/api/admin/polymarket/tracking_retry" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "marketId=123"
```

### 6.2 更新 Outcome 映射

```bash
curl -sS -X POST "$BASE_URL/api/admin/polymarket/outcome_update" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "marketId": 123,
    "externalOutcomeId": "0",
    "option": "A",
    "displayName": "YES",
    "locked": true
  }'
```

## 7. Cookie 鉴权示例

```bash
curl -sS -X POST "$BASE_URL/api/admin/polymarket/discovery_sync" \
  -H "Cookie: bbsgo_token=$TOKEN"
```

## 8. 常见问题

- 如果登录接口提示验证码错误，说明当前环境未放开 `captchaProtocol=4`，请先通过前端登录后复制 token 再执行以上命令。
- 如果返回无权限，确认登录账号是 `owner/admin` 角色。
- 若同步结果为空，检查 `bbs-go.yaml` 中 `polymarket.enabled` 是否开启，并确认数据库标签配置表已启用可用标签。

## 9. 直接访问 Gamma API（只读）

下面这些接口可以直接请求 `https://gamma-api.polymarket.com`，用于验证 Polymarket 侧市场数据是否可拉取。

### 9.1 拉取标签列表

```bash
curl -sS "https://gamma-api.polymarket.com/tags?limit=200"
```

### 9.2 按 keyset 拉取市场列表

```bash
curl -sS "https://gamma-api.polymarket.com/markets/keyset?limit=20"
```

带筛选参数示例：

```bash
curl -sS "https://gamma-api.polymarket.com/markets/keyset?limit=20&tag_id=123"
```

### 9.3 按 offset 拉取市场列表

```bash
curl -sS "https://gamma-api.polymarket.com/markets?limit=20&offset=0"
```

### 9.4 按 market id 查询单个市场

```bash
curl -sS "https://gamma-api.polymarket.com/markets/123456"
```

### 9.5 解析结果时建议关注的字段

- `id`: Gamma 市场 ID
- `slug`: 市场 slug
- `question` / `title`: 市场标题
- `active`: 是否活跃
- `closed`: 是否关闭
- `resolved`: 是否已结算
- `resolution`: 外部赢家文本或 key
- `outcomes`: 选项列表
- `clobTokenIds`: 每个 outcome 对应的 token id

### 9.6 常见排查

- 如果 curl 直接请求超时，先检查本机到公网的网络连通性。
- 如果返回空数组，通常是筛选条件不匹配，先去掉 `tag_id` 再试。
- 如果返回字段结构变化，优先看 `outcomes`、`resolution`、`clobTokenIds` 三个字段，基本能判断市场是否可同步。
