# 标签管理（Admin Tag）

本文档基于 `internal/controllers/admin/tag_controller.go` 整理，路由前缀为 `/api/admin/tag`。

---

## 1. 新建标签

- 方法：`POST`
- 路径：`/api/admin/tag/create`
- 认证：是，且需管理后台权限
- Content-Type：`application/x-www-form-urlencoded`

### Form 参数

- `name`：标签名称，必填
- `description`：标签描述，选填

### 示例

```bash
curl -X POST "http://localhost:8082/api/admin/tag/create" \
  -b "bbsgo_token=<YOUR_TOKEN>" \
  -d "name=英超" \
  -d "description=英超相关讨论"
```

### 返回示例

```json
{
  "id": 1,
  "name": "英超",
  "description": "英超相关讨论",
  "status": 0
}
```

---

## 2. 标签列表

- 方法：`GET`
- 路径：`/api/admin/tag/tags`
- 认证：是，且需管理后台权限

### Query 参数

- `page`：页码，默认 `1`
- `limit`：每页数量，默认 `20`，最大 `200`
- `keyword`：标签名称模糊搜索，选填

### 示例

```bash
curl "http://localhost:8082/api/admin/tag/tags?page=1&limit=20&keyword=英" \
  -b "bbsgo_token=<YOUR_TOKEN>"
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
