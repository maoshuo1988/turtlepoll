# 资讯采集管理（Admin News）

> 路由前缀：`/api/admin/news`
> 认证：需要登录（Admin），请求携带 cookie `token`（或 `Authorization: Bearer <token>`）。

---

## 目录

- [触发采集任务](#触发采集任务)
- [刷新文章详情](#刷新文章详情)
- [查看任务列表](#查看任务列表)
- [查看失败日志](#查看失败日志)
- [健康状态报告](#健康状态报告)

---

## 触发采集任务

`POST /api/admin/news/crawl/run`

手动触发一次资讯采集（异步执行，立即返回任务 ID）。

### 请求参数（Query String）

| 参数名   | 类型   | 必填 | 默认值   | 说明                              |
|----------|--------|------|----------|-----------------------------------|
| source   | string | 否   | `hupu`   | 数据源（目前仅支持 `hupu`）        |
| taskType | string | 否   | `list`   | 任务类型：`list`（列表采集）       |
| limit    | int    | 否   | BatchSize（yaml 配置） | 本次最多采集条数 |

### 响应示例

```json
{
  "code": 0,
  "data": {
    "taskId": 42,
    "status": "pending"
  }
}
```

> 任务在后台 goroutine 中执行，通过 [查看任务列表](#查看任务列表) 接口跟踪进度。

### curl 示例

```bash
curl -X POST "http://127.0.0.1:8082/api/admin/news/crawl/run?source=hupu&taskType=list&limit=20" \
  -H "Authorization: Bearer $TOKEN"
```

---

## 刷新文章详情

`POST /api/admin/news/crawl/refresh`

对已入库的文章重新抓取详情页，更新 `content`、`contentImages`、`coverUrl`。

### 请求参数（Form Body）

| 参数名     | 类型     | 必填 | 说明                         |
|------------|----------|------|------------------------------|
| articleIds | []int64  | 是   | 要刷新的文章 ID 列表          |

### 响应示例

```json
{
  "code": 0,
  "data": {
    "taskId": 43,
    "status": "pending"
  }
}
```

### curl 示例

```bash
curl -X POST "http://127.0.0.1:8082/api/admin/news/crawl/refresh" \
  -H "Authorization: Bearer $TOKEN" \
  -d "articleIds=1001&articleIds=1002&articleIds=1003"
```

---

## 查看任务列表

`GET /api/admin/news/crawl/tasks`

分页查看采集任务历史。

### 请求参数

| 参数名   | 类型   | 必填 | 默认值 | 说明                                              |
|----------|--------|------|--------|---------------------------------------------------|
| status   | string | 否   | ""     | 过滤状态：`pending` / `running` / `success` / `failed` |
| page     | int    | 否   | 1      | 页码                                              |
| pageSize | int    | 否   | 20     | 每页条数                                          |

### 响应示例

```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "id": 42,
        "source": "hupu",
        "taskType": "list",
        "status": "success",
        "fetchCount": 48,
        "failCount": 0,
        "errMsg": "",
        "startedAt": 1748880000,
        "finishedAt": 1748880045,
        "retryAfter": null
      }
    ],
    "total": 100,
    "page": 1,
    "limit": 20
  }
}
```

### 任务状态说明

| 状态    | 说明                   |
|---------|------------------------|
| pending | 已创建，等待执行       |
| running | 正在采集中             |
| success | 采集完成               |
| failed  | 采集失败（含熔断跳过） |

### curl 示例

```bash
# 全部任务（最新20条）
curl "http://127.0.0.1:8082/api/admin/news/crawl/tasks?page=1&pageSize=20" \
  -H "Authorization: Bearer $TOKEN"

# 仅查看失败任务
curl "http://127.0.0.1:8082/api/admin/news/crawl/tasks?status=failed" \
  -H "Authorization: Bearer $TOKEN"
```

---

## 查看失败日志

`GET /api/admin/news/crawl/logs`

查看采集过程中失败的单条文章记录。

### 请求参数

| 参数名   | 类型   | 必填 | 说明                             |
|----------|--------|------|----------------------------------|
| taskId   | int64  | 否   | 过滤指定任务的日志；0 表示全部   |
| page     | int    | 否   | 页码                             |
| pageSize | int    | 否   | 每页条数                         |

### 响应示例

```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "id": 5,
        "taskId": 42,
        "source": "hupu",
        "sourceId": "789012",
        "sourceUrl": "https://www.hupu.com/nba/789012.html",
        "errMsg": "HTTP 404",
        "createdAt": 1748880030
      }
    ],
    "total": 3,
    "page": 1,
    "limit": 20
  }
}
```

### curl 示例

```bash
curl "http://127.0.0.1:8082/api/admin/news/crawl/logs?taskId=42&page=1" \
  -H "Authorization: Bearer $TOKEN"
```

---

## 健康状态报告

`GET /api/admin/news/crawl/health`

返回指定时间窗口内的采集健康数据，以及当前熔断器状态。

### 请求参数

| 参数名        | 类型 | 必填 | 默认值 | 说明                     |
|---------------|------|------|--------|--------------------------|
| windowMinutes | int  | 否   | 10     | 统计窗口（分钟）          |

### 响应示例

```json
{
  "code": 0,
  "data": {
    "health": {
      "windowMinutes": 10,
      "taskCount": 1,
      "successCount": 1,
      "failCount": 0,
      "reachRate": 1.0,
      "avgLatencyMs": 1230,
      "parseSuccessRate": 0.96,
      "newArticles": 48
    },
    "circuitOpen": false,
    "circuitUntil": null
  }
}
```

### 健康报告字段说明

| 字段             | 类型    | 说明                                     |
|------------------|---------|------------------------------------------|
| windowMinutes    | int     | 统计窗口（分钟）                          |
| taskCount        | int     | 窗口内任务总数                            |
| successCount     | int     | 成功任务数                                |
| failCount        | int     | 失败任务数                                |
| reachRate        | float64 | 可达率（successCount / taskCount）        |
| avgLatencyMs     | int64   | 平均耗时（毫秒）                          |
| parseSuccessRate | float64 | 解析成功率（成功入库条数 / 抓取条数）      |
| newArticles      | int     | 窗口内新增文章数                          |
| circuitOpen      | bool    | 熔断器是否开路（true=当前停采）           |
| circuitUntil     | int64?  | 熔断恢复时间（Unix 秒）；未熔断时为 null  |

### curl 示例

```bash
# 查看最近10分钟健康状态
curl "http://127.0.0.1:8082/api/admin/news/crawl/health?windowMinutes=10" \
  -H "Authorization: Bearer $TOKEN"

# 查看最近1小时
curl "http://127.0.0.1:8082/api/admin/news/crawl/health?windowMinutes=60" \
  -H "Authorization: Bearer $TOKEN"
```

---

## 熔断机制说明

连续 `maxRetry`（默认3）次任务失败，且均发生在 `circuitBreakMinutes`（默认15分钟）窗口内时，熔断器开路。

- 熔断期间定时任务跳过采集，不创建新任务。
- 超过 `circuitBreakMinutes` 后自动恢复，下一次 cron 触发时重新采集。
- 手动调用 `/crawl/run` 时**同样受熔断保护**，开路期间返回 `failed`（不执行）。

通过健康接口可实时查看熔断状态：`circuitOpen: true` 表示当前处于熔断中。
