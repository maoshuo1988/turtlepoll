# 资讯系统（News）

> 路由前缀：`/api/news`
> 说明：该模块对外提供虎扑资讯的浏览、搜索、分类/标签查询接口，无需登录。

---

## 目录

- [获取资讯列表](#获取资讯列表)
- [获取资讯详情](#获取资讯详情)
- [搜索资讯](#搜索资讯)
- [获取分类列表](#获取分类列表)
- [获取标签列表](#获取标签列表)

---

## 获取资讯列表

`GET /api/news/list`

### 请求参数

| 参数名     | 类型   | 必填 | 默认值            | 说明                                                  |
|------------|--------|------|-------------------|-------------------------------------------------------|
| page       | int    | 否   | 1                 | 页码（从 1 开始）                                     |
| pageSize   | int    | 否   | 20                | 每页条数                                              |
| q          | string | 否   | ""                | 关键词搜索（标题/摘要）                               |
| category   | string | 否   | ""                | 分类 key，如 `nba`、`football`                        |
| tag        | string | 否   | ""                | 标签 key                                              |
| source     | string | 否   | `hupu`            | 数据源，目前仅支持 `hupu`                             |
| sort       | string | 否   | `publishedAt_desc`| 排序方式：`publishedAt_desc` / `publishedAt_asc` / `hotScore_desc` |

### 响应示例

```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "id": 1001,
        "title": "湖人队大胜勇士",
        "summary": "昨晚湖人队以 120-98 大胜勇士...",
        "coverUrl": "https://img.hupu.com/xxx.jpg",
        "source": "hupu",
        "sourceName": "虎扑",
        "sourceUrl": "https://www.hupu.com/nba/123456.html",
        "channel": "nba",
        "category": "nba",
        "tags": ["湖人", "勇士", "NBA"],
        "publishedAt": 1748880000,
        "hotScore": 9800
      }
    ],
    "count": 320,
    "page": 1,
    "pageSize": 20
  }
}
```

### 响应字段说明

| 字段        | 类型     | 说明                      |
|-------------|----------|---------------------------|
| id          | int64    | 资讯 ID                   |
| title       | string   | 标题                      |
| summary     | string   | 摘要（og:description）    |
| coverUrl    | string   | 封面图 URL                |
| source      | string   | 数据源标识（`hupu`）      |
| sourceName  | string   | 数据源名称（`虎扑`）      |
| sourceUrl   | string   | 原文链接                  |
| channel     | string   | 频道（nba / football 等） |
| category    | string   | 分类 key                  |
| tags        | []string | 标签列表                  |
| publishedAt | int64    | 发布时间（Unix 秒）       |
| hotScore    | int64    | 热度分（越高越热门）      |

---

## 获取资讯详情

`GET /api/news/detail`

### 请求参数（三选一，优先级：id > sourceId > slug）

| 参数名   | 类型   | 必填 | 说明              |
|----------|--------|------|-------------------|
| id       | int64  | 否   | 资讯 ID           |
| sourceId | string | 否   | 原文来源 ID       |
| slug     | string | 否   | URL slug          |

### 响应示例

```json
{
  "code": 0,
  "data": {
    "news": {
      "id": 1001,
      "source": "hupu",
      "sourceId": "123456",
      "sourceUrl": "https://www.hupu.com/nba/123456.html",
      "title": "湖人队大胜勇士",
      "summary": "昨晚湖人队以 120-98 大胜勇士...",
      "content": "<p>昨晚湖人队...</p>",
      "contentImages": [
        "https://img.hupu.com/img1.jpg",
        "https://img.hupu.com/img2.jpg"
      ],
      "coverUrl": "https://img.hupu.com/xxx.jpg",
      "sourceName": "虎扑",
      "channel": "nba",
      "category": "nba",
      "tags": ["湖人", "勇士", "NBA"],
      "publishedAt": 1748880000,
      "fetchedAt": 1748883600,
      "hotScore": 9800
    }
  }
}
```

### 响应字段说明（额外字段）

| 字段          | 类型     | 说明                                    |
|---------------|----------|-----------------------------------------|
| content       | string   | 正文 HTML（完整正文，由详情页解析）      |
| contentImages | []string | 正文中提取的图片 URL 列表               |
| fetchedAt     | int64    | 采集时间（Unix 秒）                     |

### 错误码

| code | 说明              |
|------|-------------------|
| 400  | PARAM_INVALID     |
| 404  | NEWS_NOT_FOUND    |

---

## 搜索资讯

`GET /api/news/search`

与 `/api/news/list` 参数相同，`q` 参数作为关键词进行全文搜索。

---

## 获取分类列表

`GET /api/news/categories`

### 响应示例

```json
{
  "code": 0,
  "data": {
    "list": [
      { "key": "nba",      "name": "NBA",  "sort": 1 },
      { "key": "football", "name": "足球", "sort": 2 },
      { "key": "esports",  "name": "电竞", "sort": 3 }
    ]
  }
}
```

---

## 获取标签列表

`GET /api/news/tags`

### 响应示例

```json
{
  "code": 0,
  "data": {
    "list": [
      { "key": "lakers",   "name": "湖人" },
      { "key": "warriors", "name": "勇士" }
    ]
  }
}
```

---

## curl 测试示例

```bash
# 列表（第1页，每页10条，NBA分类）
curl "http://127.0.0.1:8082/api/news/list?page=1&pageSize=10&category=nba"

# 详情（按 id）
curl "http://127.0.0.1:8082/api/news/detail?id=1001"

# 详情（按 sourceId）
curl "http://127.0.0.1:8082/api/news/detail?sourceId=123456"

# 关键词搜索
curl "http://127.0.0.1:8082/api/news/search?q=湖人&sort=hotScore_desc"

# 分类列表
curl "http://127.0.0.1:8082/api/news/categories"

# 标签列表
curl "http://127.0.0.1:8082/api/news/tags"
```
