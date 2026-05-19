# 用户中心

本文档为用户中心相关接口文档，主要描述登录用户自己的帖子列表、评论帖子列表、收藏帖子列表，以及帖子隐藏相关接口。

---

## 1. 登录用户帖子列表

- 方法：`GET`
- 路径：`/api/user/center/topics`
- 认证：是

### 需求说明

用于用户中心分页查询当前登录用户发布的帖子列表。

接口说明：

- 后端以登录态 `token` 解析出的当前用户 ID 为准
- `page` 默认 `1`
- `limit` 默认 `20`，最大 `200`

### Query 参数

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
      "title": "英超今晚怎么看？",
      "content": "聊聊这场比赛的几个关键点。",
      "create_time": "2026-05-15 20:30:00"
    }
  ],
  "page": 1,
  "limit": 10,
  "total": 1
}
```

---

## 2. 登录用户评论别人的帖子列表

- 方法：`GET`
- 路径：`/api/user/center/comments`
- 认证：是

### 需求说明

用于用户中心分页查询当前登录用户“评论过的别人的帖子”列表。

统计口径：

- 查询表：`t_comment`
- 只统计 `t_comment.entity_type = 'topic'` 的评论
- 通过 `t_comment.entity_id = t_topic.id` 关联帖子表 `t_topic`
- 只返回“当前登录用户自己发表的评论”
- 只返回“评论的是别人发的帖子”
- 判定条件：`t_comment.user_id <> t_topic.user_id`
- 按 `t_comment.create_time DESC` 排序

### Query 参数

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
    }
  ],
  "page": 1,
  "limit": 10,
  "total": 1
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

用于用户中心分页查询当前登录用户“收藏过的别人的帖子”列表。

统计口径：

- 查询表：`t_favorite`
- 只统计 `t_favorite.entity_type = 'topic'` 的收藏记录
- 通过 `t_favorite.entity_id = t_topic.id` 关联帖子表 `t_topic`
- 只返回“当前登录用户自己收藏的帖子”
- 只返回“收藏的是别人发的帖子”
- 判定条件：`t_favorite.user_id <> t_topic.user_id`
- 只返回正常状态帖子：`t_topic.status = 0`
- 按 `t_favorite.create_time DESC, t_favorite.id DESC` 排序

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
    }
  ],
  "page": 1,
  "limit": 10,
  "total": 1
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

---

## 4. 帖子显示状态设计

本次隐藏帖子需求不修改 `t_topic.status` 语义，新增字段 `display_status` 用于控制“用户自己隐藏的帖子”。

### 字段建议

表：`t_topic`

- `display_status = 0`：显示
- `display_status = 1`：隐藏

### 设计说明

- `status` 继续表示帖子业务状态，例如正常、删除、待审核
- `display_status` 单独表示帖子是否被作者隐藏
- “隐藏帖子列表”查询条件按 `user_id + display_status = 1` 查询
- 隐藏/取消隐藏只更新 `display_status`，不改 `status`

### 建议 DDL

```sql
ALTER TABLE t_topic
ADD COLUMN display_status int2 NOT NULL DEFAULT 0;

COMMENT ON COLUMN t_topic.display_status IS '显示状态：0-显示，1-隐藏';
```

### 本次维护脚本

```sql
ALTER TABLE t_topic
ADD COLUMN display_status int2 NOT NULL DEFAULT 0;
```

---

## 5. 隐藏帖子

- 方法：`POST`
- 路径：`/api/user/topic/hide/{topicId}`
- 认证：是

### 需求说明

登录用户将自己的帖子设为隐藏状态。

接口行为：

- 仅允许操作当前登录用户自己发布的帖子
- 更新 `t_topic.display_status = 1`
- 不修改 `t_topic.status`

### Path 参数

- `topicId`：帖子 ID

### 返回

成功返回：

```json
{
  "success": true
}
```

### 可能错误

- 未登录
- 帖子不存在
- 无权限操作他人帖子

---

## 6. 取消隐藏帖子

- 方法：`POST`
- 路径：`/api/user/topic/unhide/{topicId}`
- 认证：是

### 需求说明

登录用户将自己已隐藏的帖子恢复为显示状态。

接口行为：

- 仅允许操作当前登录用户自己发布的帖子
- 更新 `t_topic.display_status = 0`
- 不修改 `t_topic.status`

### Path 参数

- `topicId`：帖子 ID

### 返回

成功返回：

```json
{
  "success": true
}
```

### 可能错误

- 未登录
- 帖子不存在
- 无权限操作他人帖子

---

## 7. 登录用户自己隐藏的帖子列表

- 方法：`POST`
- 路径：`/api/user/topic/hide/list?page=1&limit=10`
- 认证：是

### 需求说明

分页查询当前登录用户自己隐藏的帖子列表。

查询口径：

- 查询表：`t_topic`
- 按当前登录用户 `user_id` 查询
- 只查询 `display_status = 1` 的帖子
- 建议按 `create_time DESC, id DESC` 排序

### Query 参数

- `page`：页码，默认 `1`
- `limit`：每页数量，默认 `20`，最大 `200`

### 返回字段

- `id`：帖子 ID
- `user_id`：用户 ID
- `content`：帖子内容
- `title`：帖子标题
- `create_time`：帖子创建时间
- `display_status`：显示状态，`0` 表示显示，`1` 表示隐藏

### 示例

```bash
curl -X POST "http://localhost:8082/api/user/topic/hide/list?page=1&limit=10" \
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
      "create_time": "2026-05-15 21:10:00",
      "display_status": 1
    }
  ],
  "page": 1,
  "limit": 10,
  "total": 1
}
```

### 参考 SQL

```sql
SELECT
    id,
    user_id,
    content,
    title,
    create_time,
    display_status
FROM t_topic
WHERE user_id = :current_user_id
  AND display_status = 1
ORDER BY create_time DESC, id DESC
LIMIT :limit OFFSET (:page - 1) * :limit;
```
