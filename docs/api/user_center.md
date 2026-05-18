# 用户中心

本文档为新增需求接口文档，描述用户中心“登录用户帖子列表”接口。

---

## 1. 登录用户帖子列表

- 方法：`GET`
- 路径：`/api/user/center/topics`
- 认证：是

### 需求说明

用于用户中心分页查询当前登录用户发布的帖子列表。

接口说明：

- 后端以登录态 token 解析出的当前用户 ID 为准
- `user` 参数可兼容保留，但即使传入，与 token 对应用户不一致时也不会按该参数查询

### Query 参数

- `user`：用户 ID，选填，仅兼容保留，实际以后端从 token 解析出的当前登录用户 ID 为准
- `page`：页码，默认 `1`
- `limit`：每页数量，默认 `20`，最大 `200`

### 返回字段

- `id`：帖子主键
- `userId`：用户 ID
- `title`：帖子标题
- `content`：帖子内容
- `create_time`：帖子创建时间

### 示例

```bash
curl "http://localhost:8082/api/user/center/topics?page=1&limit=10" \
  -b "bbsgo_token=<YOUR_TOKEN>"
```

### 返回示例

```json
{
  "results": [
    {
      "id": 101,
      "userId": 1,
      "title": "英超今晚怎么看",
      "content": "聊聊这场比赛的几个关键点。",
      "create_time": "2026-05-15 20:30:00"
    },
    {
      "id": 102,
      "userId": 1,
      "title": "西甲本轮观察",
      "content": "这轮更看好主场一方。",
      "create_time": "2026-05-14 18:20:00"
    }
  ],
  "page": 1,
  "limit": 10,
  "total": 2
}
```

---

## 2. 登录用户评论别人的帖子列表

- 方法：`GET`
- 路径：`/api/user/center/comments`
- 认证：是

### 需求说明

用于用户中心分页查询当前登录用户“评论过的别人帖子”列表。

接口说明：

- 后端以登录态 token 解析出的当前用户 ID 为准
- `user` 参数可兼容保留，但即使传入，与 token 对应用户不一致时也不会按该参数查询

统计口径：

- 查询表：`t_comment`
- 只统计 `t_comment.entity_type = 'topic'` 的评论
- 通过 `t_comment.entity_id = t_topic.id` 与帖子表 `t_topic` 关联，获取帖子标题 `title`
- 只返回“登录用户自己发表的评论”
- 只返回“评论的是别人发的帖子”
- 判定条件：`t_comment.user_id <> t_topic.user_id`
- 按 `t_comment.create_time DESC` 排序，最新评论排在最前面

### Query 参数

- `user`：登录用户 ID，选填，仅兼容保留，实际以后端从 token 解析出的当前登录用户 ID 为准
- `page`：页码，默认 `1`
- `limit`：每页数量，默认 `20`，最大 `200`

### 返回字段

- `id`：评论主键 ID
- `user_id`：评论用户 ID
- `content`：评论内容
- `title`：帖子标题
- `create_time`：评论创建时间

### 示例

```bash
curl "http://localhost:8082/api/user/center/comments?page=1&limit=10" \
  -b "bbsgo_token=<YOUR_TOKEN>"
```

### 返回示例

```json
{
  "results": [
    {
      "id": 201,
      "user_id": 1,
      "content": "这个观点我比较认同。",
      "title": "英超今晚怎么看？",
      "create_time": "2026-05-15 21:10:00"
    },
    {
      "id": 202,
      "user_id": 1,
      "content": "我觉得这场比赛主队更稳一些。",
      "title": "西甲本轮观察",
      "create_time": "2026-05-14 19:05:00"
    }
  ],
  "page": 1,
  "limit": 10,
  "total": 2
}
```

### 参考 SQL

```sql
SELECT
    c.id,
    c.user_id,
    c.content,
    t.title,
    c.create_time
FROM t_comment c
INNER JOIN t_topic t
    ON t.id = c.entity_id
WHERE c.user_id = :user_id
  AND c.entity_type = 'topic'
  AND c.status = 0
  AND t.status = 0
  AND c.user_id <> t.user_id
ORDER BY c.create_time DESC
LIMIT :limit OFFSET (:page - 1) * :limit;
```

---

## 3. 登录用户收藏别人的帖子列表

- 方法：`GET`
- 路径：`/api/user/center/favorites`
- 认证：是

### 需求说明

用于用户中心分页查询当前登录用户“收藏过的别人帖子”列表。

接口说明：

- 后端以登录态 token 解析出的当前用户 ID 为准
- 参数只保留 `page`、`limit`
- 只查询帖子收藏，不查询文章收藏

统计口径：

- 查询表：`t_favorite`
- 只统计 `t_favorite.entity_type = 'topic'` 的收藏记录
- 通过 `t_favorite.entity_id = t_topic.id` 与帖子表 `t_topic` 关联，获取帖子标题 `title`、帖子内容 `content`
- 只返回“当前登录用户自己收藏的帖子”
- 只返回“收藏的是别人发的帖子”
- 判定条件：`t_favorite.user_id <> t_topic.user_id`
- 只返回正常状态帖子：`t_topic.status = 0`
- 按 `t_favorite.create_time DESC, t_favorite.id DESC` 排序，最新收藏排在最前面

### 数据表

帖子收藏表：`t_favorite`

```sql
CREATE TABLE "public"."t_favorite" (
  "id" int8 NOT NULL DEFAULT nextval('t_favorite_id_seq'::regclass),
  "user_id" int8 NOT NULL,
  "entity_type" varchar(32) NOT NULL,
  "entity_id" int8 NOT NULL,
  "create_time" int8,
  CONSTRAINT "t_favorite_pkey" PRIMARY KEY ("id")
);
```

帖子表：`t_topic`

```sql
CREATE TABLE "public"."t_topic" (
  "id" int8 NOT NULL DEFAULT nextval('t_topic_id_seq'::regclass),
  "type" int8 NOT NULL DEFAULT 0,
  "node_id" int8 NOT NULL,
  "user_id" int8 NOT NULL,
  "title" varchar(128),
  "content_type" varchar(32) DEFAULT 'markdown',
  "content" text,
  "image_list" text,
  "hide_content" text,
  "vote_id" int8 NOT NULL DEFAULT 0,
  "recommend" bool NOT NULL,
  "recommend_time" int8 NOT NULL,
  "sticky" bool NOT NULL,
  "sticky_time" int8 NOT NULL,
  "view_count" int8 NOT NULL,
  "comment_count" int8 NOT NULL,
  "like_count" int8 NOT NULL,
  "status" int8,
  "last_comment_time" int8,
  "last_comment_user_id" int8,
  "user_agent" varchar(1024),
  "ip" varchar(128),
  "ip_location" varchar(64),
  "create_time" int8,
  "extra_data" text,
  CONSTRAINT "t_topic_pkey" PRIMARY KEY ("id")
);
```

### Query 参数

- `page`：页码，默认 `1`
- `limit`：每页数量，默认 `20`，最大 `200`

### 返回字段

- `id`：收藏记录主键 ID
- `user_id`：收藏用户 ID
- `entity_id`：帖子 ID
- `title`：帖子标题
- `content`：帖子内容
- `create_time`：收藏时间

### 示例

```bash
curl "http://localhost:8082/api/user/center/favorites?page=1&limit=10" \
  -b "bbsgo_token=<YOUR_TOKEN>"
```

### 返回示例

```json
{
  "results": [
    {
      "id": 301,
      "user_id": 1,
      "entity_id": 2001,
      "title": "英超今晚怎么看？",
      "content": "聊聊这场比赛的几个关键点。",
      "create_time": "2026-05-16 10:20:00"
    },
    {
      "id": 302,
      "user_id": 1,
      "entity_id": 2002,
      "title": "西甲本轮观察",
      "content": "这轮更看好主场一方。",
      "create_time": "2026-05-15 19:30:00"
    }
  ],
  "page": 1,
  "limit": 10,
  "total": 2
}
```

### 参考 SQL

```sql
SELECT
    f.id,
    f.user_id,
    f.entity_id,
    t.title,
    t.content,
    f.create_time
FROM t_favorite f
INNER JOIN t_topic t
    ON t.id = f.entity_id
WHERE f.user_id = :current_user_id
  AND f.entity_type = 'topic'
  AND f.user_id <> t.user_id
  AND t.status = 0
ORDER BY f.create_time DESC, f.id DESC
LIMIT :limit OFFSET (:page - 1) * :limit;
```
