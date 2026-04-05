# 预测市场（Admin Predict）

## 1) 统计与看板

### 1.1 获取预测市场统计

- **接口**：`GET /api/admin/predict/stats`
- **认证**：需要管理员（`AdminMiddleware`）
- **说明**：用于运营侧总览看板与预测市场运营列表的概览统计。

#### 返回值（data）

- `openCount`: int64（进行中市场数：`status=OPEN`）
- `closedCount`: int64（已关闭待结算市场数：`status=CLOSED`）
- `settledCount`: int64（已结算市场数：`status=SETTLED`）
- `todayNewMarkets`: int64（今日新增市场数：`create_time >= 今日 00:00(UTC) 的秒级时间戳`）
- `todayBetAmount`: int64（今日下注额：`t_predict_bet.amount` 求和；口径：`create_time >= 今日 00:00(UTC)`）
- `todayFee`: int64（今日入场费：当前实现为 0；预测下注暂未产生 fee 独立流水）
- `todayBurn`: int64（今日燃烧：当前实现为 0；预测下注暂未产生 burn 独立流水）

#### 示例

请求：

```http
GET /api/admin/predict/stats
```

响应：

```json
{
  "success": true,
  "data": {
    "openCount": 12,
    "closedCount": 3,
    "settledCount": 20,
  "todayNewMarkets": 2,
  "todayBetAmount": 1200,
  "todayFee": 0,
  "todayBurn": 0
  }
}
```


### 1.2 7/14/30 日趋势（每日新增市场数）

- **接口**：`GET /api/admin/predict/trends?range=7d`
- **认证**：需要管理员（`AdminMiddleware`）
- **说明**：返回趋势柱状图数据；当前版本口径为“每日新增市场数”（按 `t_predict_market.create_time` 聚合）。

#### 查询参数

- `range`: string，可选，支持：`7d` / `14d` / `30d`（默认 `7d`；最大 `90d`）

#### 返回值（data）

- `range`: string
- `days`: int
- `list`: array
  - `day`: string（`YYYY-MM-DD`，UTC）
  - `count`: int64（当日新增市场数）

#### 示例

请求：

```http
GET /api/admin/predict/trends?range=7d
```

响应：

```json
{
  "success": true,
  "data": {
    "range": "7d",
    "days": 7,
    "list": [
      {"day": "2026-04-01", "count": 3},
      {"day": "2026-04-02", "count": 0},
      {"day": "2026-04-03", "count": 1}
    ]
  }
}
```


### 1.3 7/14/30 日活跃用户（下注去重用户数）
- **说明**：返回趋势柱状图数据；口径为“每日下注去重用户数”（按 `t_predict_bet.create_time` 聚合，`count(distinct user_id)`）。

#### 查询参数

- `range`: string，可选，支持：`7d` / `14d` / `30d`（默认 `7d`；最大 `90d`）

#### 返回值（data）

- `range`: string
- `days`: int
- `list`: array
  - `day`: string（`YYYY-MM-DD`，UTC）
  - `activeUserCount`: int64（当日下注用户去重数）

#### 示例

请求：

```http
GET /api/admin/predict/active_users?range=7d
```

响应：

```json
{
  "success": true,
  "data": {
    "range": "7d",
    "days": 7,
    "list": [
      {"day": "2026-04-01", "activeUserCount": 11},
      {"day": "2026-04-02", "activeUserCount": 8},
      {"day": "2026-04-03", "activeUserCount": 15}
    ]
  }
}
```

---

## 1.4) 获取市场盘口统计（结算前）

- **方法**：GET
- **路径**：`/api/admin/predict/market/stats`
- **认证**：需要管理员（`AdminMiddleware`）

### Query 参数

- `marketId` (int64, 必填)

### 返回字段（data）

- `marketId`: int64
- `status`: string（market.status）
- `result`: string（market.result）
- `closeTime`: int64
- `resolved`: bool
- `resolvedAt`: int64
- `proUserCount`: int64（option=A 去重下注用户数）
- `conUserCount`: int64（option=B 去重下注用户数）
- `proAmount`: int64（option=A 下注金额 sum）
- `conAmount`: int64（option=B 下注金额 sum）
- `totalAmount`: int64（A+B）
- `totalBetCount`: int64（A/B 下单数 sum）

### 示例

`GET /api/admin/predict/market/stats?marketId=1`

---

## 1.5) 管理员结算市场（CLOSED -> SETTLED）

- **方法**：POST
- **路径**：`/api/admin/predict/market/settle`
- **认证**：需要管理员（`AdminMiddleware`）
- **请求格式**：JSON

### 请求体（JSON）

- `marketId` (int64, 必填)
- `result` (string, 必填)：`A` 或 `B`
- `requestId` (string, 必填)：用于审计/幂等标识（当前实现用于日志，不做硬幂等）
- `remark` (string, 可选)
- `allowReset` (bool, 可选，默认 false)：
  - false：仅允许 `CLOSED -> SETTLED`
  - true：允许对已 `SETTLED` 的市场再次写入结果（纠错用，谨慎开启）

### 行为说明

- 将 `t_predict_market.status` 更新为 `SETTLED`。
- 将 `t_predict_market.result` 写入 `A/B`。
- 同时将 `t_predict_market.resolved=true`（兼容外部市场字段）。
- 记录一条 operate-log：`dataType=predictMarket` + `opType=update`。

### 返回（data）

返回更新后的 `PredictMarket`。

### 示例

```json
{
  "marketId": 1,
  "result": "A",
  "requestId": "admin-settle-0001",
  "remark": "manual settle"
}
```
