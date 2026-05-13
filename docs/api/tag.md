# 标签系统

本文档基于 `internal/controllers/api/tag_controller.go` 整理，路由前缀为 `/api/tag`。

说明：

- 标签主表为 `t_tag`
- 话题与标签关系表为 `t_topic_tag`
- 评论表为 `t_comment`
- 本次新增的“标签评论统计”接口，实际关联链路为 `t_tag -> t_topic_tag -> t_topic -> t_comment`
- 虽然 `t_comment.entity_id` 对应 `t_topic.id`，但如果要按标签统计评论数，必须经过 `t_topic_tag`

---

## 1. 新建标签

- 方法：`POST`
- 路径：`/api/tag/create`
- 认证：需要登录
- Content-Type：`application/x-www-form-urlencoded`

### Form 参数

- `name`：标签名称，必填，最大 32 字符
- `description`：标签描述，选填，最大 1024 字符

### 行为说明

- `name` 会先做首尾空白裁剪
- 如果同名标签已存在，则直接返回该标签
- 如果同名标签存在但状态不是正常，接口会恢复为正常状态

### 示例

```bash
curl -X POST "http://localhost:8082/api/tag/create" \
  -b "bbsgo_token=<YOUR_TOKEN>" \
  -d "name=英超" \
  -d "description=英超相关讨论"
```

### 返回示例

```json
{
  "id": 1,
  "name": "英超"
}
```

---

## 2. 标签分页列表

- 方法：`GET`
- 路径：`/api/tag/tags`
- 认证：否

### Query 参数

- `page`：页码，默认 `1`
- `limit`：每页数量，默认 `20`，最大 `200`
- `keyword`：标签名称模糊搜索，选填

### 示例

```bash
curl "http://localhost:8082/api/tag/tags?page=1&limit=20&keyword=英"
```

### 返回示例

```json
{
  "results": [
    {
      "id": 1,
      "name": "英超"
    }
  ],
  "page": 1,
  "limit": 20,
  "total": 1
}
```

---

## 3. 标签详情

- 方法：`GET`
- 路径：`/api/tag/{tagId}`
- 认证：否

### 返回说明

- 成功返回 `TagResponse`
- 标签不存在时返回：`tag not found`

---

## 4. 标签自动完成

- 方法：`POST`
- 路径：`/api/tag/autocomplete`
- 认证：否

### Form 参数

- `input`：输入关键字，必填

### 返回说明

- 最多返回 6 条状态正常的标签

---

## 5. 标签下评论数统计分页

- 方法：`GET`
- 路径：`/api/tag/comment_stats`
- 认证：否

### 需求说明

统计每个标签下所有用户的评论总数，并按评论数降序分页返回。

统计口径：

- 只统计 `t_comment.entity_type = 'topic'` 的评论
- `t_comment.entity_id = t_topic.id`
- 通过 `t_topic_tag.topic_id = t_topic.id` 找到话题所属标签
- 最终按 `t_tag.id`、`t_tag.name` 聚合
- 排序规则：`commentCount DESC, tagId ASC`
- 为了保证没有评论的标签也能返回，查询采用 `LEFT JOIN`

### Query 参数

- `page`：页码，默认 `1`
- `limit`：每页数量，默认 `20`，最大 `200`
- `keyword`：标签名称模糊搜索，选填

### 返回字段

- `tagId`：标签 ID
- `tagName`：标签名称
- `commentCount`：该标签下所有话题的评论总数

### 示例

```bash
curl "http://localhost:8082/api/tag/comment_stats?page=1&limit=10"
```

### 返回示例

```json
{
  "results": [
    {
      "tagId": 1,
      "tagName": "英超",
      "commentCount": 12
    },
    {
      "tagId": 2,
      "tagName": "西甲",
      "commentCount": 8
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
    tg.id   AS tag_id,
    tg.name AS tag_name,
    COUNT(c.id) AS comment_count
FROM t_tag tg
LEFT JOIN t_topic_tag tt
    ON tt.tag_id = tg.id
   AND tt.status = 0
LEFT JOIN t_topic tp
    ON tp.id = tt.topic_id
   AND tp.status = 0
LEFT JOIN t_comment c
    ON c.entity_id = tp.id
   AND c.entity_type = 'topic'
   AND c.status = 0
WHERE tg.status = 0
GROUP BY tg.id, tg.name
ORDER BY comment_count DESC, tg.id ASC;
```
