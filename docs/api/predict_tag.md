# 预测标签（PredictTag / PredictTagStat）

本模块接口由 `internal/controllers/api/predict_tag_controller.go` 提供，并在 `internal/server/router.go` 中注册到：

- 路由组：`/api/predict-tag`
- 说明：挂在 `/api` 下，默认经过 `AuthMiddleware`（需要登录）

> 背景：标签的事实来源是 `t_predict_context.tags`（逗号分隔）。
> 
> 为了避免在线请求时扫描/拆分 `t_predict_context` 造成慢查询，本项目会将 tags **物化**到：
> - `t_predict_tag`：标签维表（slug + lastSeenAt）
> - `t_predict_tag_stat`：标签统计表（marketCount + refreshedAt）
> 
> 物化刷新支持：
> - 定时刷新（cron）
> - 手动刷新（API）

## 数据模型

### PredictTag（标签维表）
代码定义：`internal/models/models.go` -> `type PredictTag`

核心字段：
- `id`: int64
- `slug`: string（唯一，建议小写）
- `name`: string（可选，当前实现默认与 slug 一致）
- `lastSeenAt`: int64（该标签最近一次在 `PredictContext.tags` 中出现的时间，秒级时间戳）

### PredictTagStat（标签统计表）
代码定义：`internal/models/models.go` -> `type PredictTagStat`

核心字段：
- `id`: int64
- `tagId`: int64（唯一，对应 PredictTag.id）
- `marketCount`: int64（去重后的 market 数量）
- `refreshedAt`: int64（本次统计刷新时间，秒级时间戳）

---

## 接口列表

### 1) 查询标签列表（可选带统计）

- **接口**：`GET /api/predict-tag/list`
- **功能**：分页查询物化后的标签维表；可选关联返回统计信息（marketCount、refreshedAt）。
- **认证**：需要登录（`AuthMiddleware`）

#### 请求参数（query）
- `page`：int，默认 1
- `limit`：int，默认 20（建议最大 200）
- `q`：string，可选，按 slug 模糊匹配
- `slugs`：string，可选，逗号分隔（仅查询指定 slugs）
- `sort`：string，可选
  - `lastSeenAt_desc`（默认）
  - `slug_asc`
- `includeCounts`：bool，可选，默认 false
  - true：LEFT JOIN `t_predict_tag_stat` 返回统计信息

#### 返回值（data）
- `list`：数组，每个元素包含：
  - `tag`：PredictTag
  - `stat`：PredictTagStat（当 includeCounts=false 时可能为空对象/零值）
- `total`：总数
- `page/limit`：分页信息

返回示例（字段会随实际模型演进，这里仅展示结构）：

```json
{
  "list": [
    {
      "tag": {
        "id": 1,
        "slug": "wc",
        "name": "wc",
        "lastSeenAt": 1734012345
      },
      "stat": {
        "tagId": 1,
        "marketCount": 128,
        "refreshedAt": 1734019999
      }
    }
  ],
  "total": 1,
  "page": 1,
  "limit": 20
}
```

#### 可能错误
- 未登录：`errs.NotLogin()`

---

### 2) 手动触发标签物化刷新（全量）

- **接口**：`POST /api/predict-tag/refresh`
- **功能**：从 `t_predict_context.tags` 全量聚合，并 upsert 写入：
  - `t_predict_tag`
  - `t_predict_tag_stat`
- **认证**：需要登录（`AuthMiddleware`）

#### 请求参数
- 无

#### 返回值（data）
- `{ "ok": true }`（以实际实现为准）

#### 可能错误
- 未登录：`errs.NotLogin()`
- 数据库错误：返回通用错误（以实际错误包装为准）

---

## 备注与运维建议

- 刷新是全量聚合：当 `t_predict_context` 量级非常大时，会带来一定数据库压力；可以通过降低频率/在低峰执行。
- 后续可演进为增量刷新（watermark），但需要额外记录 last processed 的时间戳或 id。
