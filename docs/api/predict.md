# 预测事件系统（PredictMarket / PredictContext）

本模块接口当前由 `internal/controllers/api/football_controller.go` 提供，并在 `internal/server/router.go` 中注册到：

- 路由组：`/api/football`
- 说明：挂在 `/api` 下，默认经过 `AuthMiddleware`（需要登录）

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
      }
    }
  ],
  "total": 1
}
```

#### 可能错误
- 未登录：`errs.NotLogin()`（由中间件/通用错误处理返回）

---

### 2) 修改/创建 PredictContext（按 marketId upsert）

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

### 3) 热度榜（heat 前 N）

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
