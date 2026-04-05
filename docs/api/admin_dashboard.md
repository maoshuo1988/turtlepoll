# 运营总览看板（Admin Dashboard）

## 1) 全站统计

### 1.1 获取全站基础统计

- **接口**：`GET /api/admin/dashboard/stats`
- **认证**：需要管理员（`AdminMiddleware`）
- **说明**：用于运营侧总览看板的“全站概览”卡片，返回总用户数、总评论数、总帖子数。

#### 返回值（data）

- `totalUsers`: int64（总用户数）
- `totalComments`: int64（总评论数，仅统计 `status=ok`）
- `totalTopics`: int64（总帖子数，仅统计 `status=ok`）

#### 示例

请求：

```http
GET /api/admin/dashboard/stats
```

响应：

```json
{
  "success": true,
  "data": {
    "totalUsers": 12345,
    "totalComments": 67890,
    "totalTopics": 24680
  }
}
```
