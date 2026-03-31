# 预测事件系统（PredictMarket / PredictContext）

本模块接口当前由 `internal/controllers/api/football_controller.go` 提供，并在 `internal/server/router.go` 中注册到：

- 路由组：`/api/football`
- 说明：挂在 `/api` 下，默认经过 `AuthMiddleware`（需要登录）

## 下注与金币接口

预测市场的“金币下注/锁赔率/池子”相关接口已独立整理到：

- `docs/api/coin.md`

## 数据模型

### PredictMarket（预测市场）
代码定义：`internal/models/models.go` -> `type PredictMarket`

核心字段：
- `id`: int64
- `sourceModel`: string（例如 `MatchSchedule`）
- `sourceModelId`: int64（例如 `MatchSchedule.Id`）
- `title`: string
- `marketType`: string（例如 `1x2`）
- `status`: string（`OPEN/CLOSED/SETTLED`）
- `closeTime`: int64（下注截止时间，秒级时间戳）

### PredictContext（预测市场上下文）
代码定义：`internal/models/models.go` -> `type PredictContext`

核心字段：
- `id`: int64
- `marketId`: int64（与 PredictMarket 一对一）
- `eventName`: string（必填）
- `imageUrl`: string
- `participantCount`: int64
- `proText` / `conText`: string
- `proVoteCount` / `conVoteCount`: int64
- `heat`: int64（热度）
- `detail`: string
- `tags`: string（逗号分隔）

---

## 接口列表

### 1) 查询预测市场（聚合返回 market + context）

- **接口**：`GET /api/football/markets`
- **功能**：分页查询 PredictMarket，并批量查询对应的 PredictContext，最终聚合返回。
- **认证**：需要登录（`AuthMiddleware`）

#### 请求参数（query）
- `page`：int，默认 1
- `limit`：int，默认 20
- `sourceModel`：string，可选（例如 `MatchSchedule`）
- `sourceModelId`：int64，可选

#### 返回值（data）
- `list`：数组，每个元素：
  - `market`：PredictMarket
  - `context`：PredictContext（若不存在则为空对象/默认值）
  - `betSettleResult`：string，当前登录用户在该 market 的下注结算结果（来自 PredictBet.SettleResult 聚合）
    - 用户未在该 market 下过注：返回空字符串 `""`
    - 若有多笔下注单：优先返回 `WIN`，其次 `LOSE`，否则返回最新一条非空值
  - `hasBet`：bool，当前登录用户是否在该 market 下过注（存在任意 PredictBet 记录即为 true）
- `total`：总数

返回示例（字段会随实际模型演进，这里仅展示结构）：

```json
{
  "list": [
    {
      "market": {
        "id": 1,
        "sourceModel": "MatchSchedule",
        "sourceModelId": 1001,
        "title": "阿根廷 vs 法国",
        "marketType": "1x2",
        "status": "OPEN",
        "closeTime": 1734012345,
        "externalKey": "wc-2026-1001",
        "createTime": 1734010000,
        "updateTime": 1734010000
      },
      "context": {
        "id": 10,
        "marketId": 1,
        "eventName": "世界杯决赛",
        "imageUrl": "https://example.com/banner.png",
        "participantCount": 123,
        "proText": "支持阿根廷",
        "proVoteCount": 80,
        "conText": "支持法国",
        "conVoteCount": 43,
        "heat": 999,
        "detail": "...",
        "tags": "wc,final",
        "createTime": 1734010000,
        "updateTime": 1734010000
      },
  "betSettleResult": "WIN",
  "hasBet": true
    }
  ],
  "total": 1
}
```

#### 可能错误
- 未登录：`errs.NotLogin()`（由中间件/通用错误处理返回）

---

### 1.1) 通过名称模糊查询预测市场（聚合返回 market + context）

- **接口**：`GET /api/football/markets/by_name`
- **功能**：通过名称关键词 `q` 模糊匹配预测市场列表。
  - 命中 `PredictMarket.title` 或 `PredictContext.eventName`
- **认证**：需要登录（`AuthMiddleware`）

#### 请求参数（query）
- `q`：string，必填。关键词（模糊匹配）
- `page`：int，默认 1
- `limit`：int，默认 20

#### 返回值（data）
- 返回结构与 `GET /api/football/markets` 保持一致：
  - `list`: market + context + betSettleResult + hasBet
  - `total`: 总数
  - `q`: 原样返回关键词

#### 可能错误
- `q is required`

### 2) 查询用户在某个市场的下注结算结果（betSettleResult）

- **接口**：`GET /api/football/bet_settle_result`
- **功能**：根据 `userId + marketId` 查询并聚合该用户在该市场的下注结算结果（基于 PredictBet.SettleResult）。
- **认证**：需要登录（`AuthMiddleware`）
- **权限**：仅允许查询本人（`userId` 必须等于当前登录用户 ID）

#### 请求参数（query）
- `userId`：int64，必填（必须为当前登录用户）
- `marketId`：int64，必填

#### 返回值（data）
- `userId`：int64
- `marketId`：int64
- `betSettleResult`：string
  - 无下注：返回空字符串 `""`
  - 多笔下注：优先返回 `WIN`，其次 `LOSE`，否则返回最新一条非空值

返回示例：

```json
{
  "userId": 100,
  "marketId": 1,
  "betSettleResult": "WIN"
}
```

#### 可能错误
- 未登录：`errs.NotLogin()`
- 无权限（查询非本人）：`errs.NoPermission()`

---

### 3) 修改/创建 PredictContext（按 marketId upsert）

- **接口**：`POST /api/football/predict_context/update`
- **功能**：按 `marketId` upsert PredictContext。
  - 记录不存在：创建
  - 记录存在：更新展示字段（heat、投票数、文案等）
- **认证**：需要登录（方法内也做了 NotLogin 判断）
- **请求格式**：表单（`params.ReadForm`），推荐 `application/x-www-form-urlencoded` 或 `multipart/form-data`

#### 请求参数（form）
- `marketId`：int64，必填
- `eventName`：string，必填
- `imageUrl`：string，可选
- `participantCount`：int64，可选
- `proText`：string，可选
- `proVoteCount`：int64，可选
- `conText`：string，可选
- `conVoteCount`：int64，可选
- `heat`：int64，可选
- `detail`：string，可选
- `tags`：string，可选

请求示例（这里用 JSON 展示字段结构；实际接口用 `params.ReadForm` 读表单）：

```json
{
  "marketId": 1,
  "eventName": "世界杯决赛",
  "imageUrl": "https://example.com/banner.png",
  "participantCount": 123,
  "proText": "支持阿根廷",
  "proVoteCount": 80,
  "conText": "支持法国",
  "conVoteCount": 43,
  "heat": 999,
  "detail": "详细说明...",
  "tags": "wc,final"
}
```

#### 返回值（data）
- PredictContext（更新后的记录）

#### 可能错误
- 未登录：`errs.NotLogin()`
- 参数读取失败：`params.ReadForm` 返回的错误
- 参数校验失败：
  - `marketId is required`
  - `eventName is required`

---

### 4) 热度榜（heat 前 N）

- **接口**：`GET /api/football/predict_context/hot`
- **功能**：按 `heat desc, id desc` 排序返回 PredictContext 列表。
- **认证**：需要登录（`AuthMiddleware`）

#### 请求参数（query）
- `limit`：int，默认 10，最大 100

#### 返回值（data）
- `list`：PredictContext 数组
- `limit`：实际使用的 limit

返回示例（字段会随实际模型演进，这里仅展示结构）：

```json
{
  "list": [
    {
      "id": 10,
      "marketId": 1,
      "eventName": "世界杯决赛",
      "heat": 999,
      "proText": "支持阿根廷",
      "proVoteCount": 80,
      "conText": "支持法国",
      "conVoteCount": 43,
      "tags": "wc,final"
    }
  ],
  "limit": 10
}
```

---

### 5) 热门标签 TOP10（按热度累计）

- **接口**：`GET /api/football/predict_tags/hot`
- **功能**：统计 `PredictContext.tags` 中的标签，并按“标签热度”倒序返回。
  - 当前热度口径：对每条 `PredictContext`，将其 `heat` 累计到它包含的每个 tag 上。
  - tags 存储为逗号分隔字符串（例如 `"wc,final"`），服务端会做 trim 与转小写归一。
- **认证**：需要登录（`AuthMiddleware`）

#### 请求参数（query）
- `limit`：int，默认 10，最大 100

#### 返回值（data）
- `list`：数组，每个元素：
  - `tag`：string
  - `heat`：int64
- `limit`：实际使用的 limit

返回示例：

```json
{
  "list": [
    { "tag": "wc", "heat": 9999 },
    { "tag": "final", "heat": 8888 }
  ],
  "limit": 10
}
```

---

### 6) 按标签查询预测市场列表（聚合返回 market + context）

- **接口**：`GET /api/football/markets/by_tag`
- **功能**：按标签查询拥有该 tag 的 PredictContext，再聚合返回对应 PredictMarket + PredictContext。
- **认证**：需要登录（`AuthMiddleware`）

#### 请求参数（query）
- `tag`：string，必填
- `page`：int，默认 1
- `limit`：int，默认 20，最大 100

返回示例（结构同 markets）：

```json
{
  "list": [
    {
      "market": { "id": 1, "title": "阿根廷 vs 法国", "status": "OPEN" },
      "context": { "marketId": 1, "eventName": "世界杯决赛", "heat": 999, "tags": "wc,final" }
    }
  ],
  "total": 1,
  "page": 1,
  "limit": 20,
  "tag": "final"
}
```

#### 可能错误
- `tag is required`
