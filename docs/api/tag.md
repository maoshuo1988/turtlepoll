# 标签评论统计

本文档基于 `internal/controllers/api/tag_controller.go` 整理，路由前缀为 `/api/tag`。

---

## 1. 标签下评论数统计分页

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
