# Tag / 标签 API

> 本文档基于 `internal/controllers/api/tag_controller.go` 整理，路由挂载在 `/api/tag`。
>
> 标签数据落在 `t_tag` 表。话题发布时的 `tags` 字段使用标签名称数组，后端会通过标签名称关联或创建标签。

## 新建标签

- 方法：`POST`
- 路径：`/api/tag/create`
- 认证：是
- Content-Type：`application/x-www-form-urlencoded`

### Form 参数

- `name` (string, 必填)：标签名称，最长 32 个字符。
- `description` (string, 可选)：标签描述，最长 1024 个字符。

### 行为

- `name` 会去除首尾空白。
- 如果同名标签已存在，则返回已有标签。
- 如果同名标签状态不是正常，接口会恢复为正常状态。
- 当前用户必须可发帖，否则返回对应错误。

### 示例

```bash
curl -X POST "http://localhost:8082/api/tag/create" \
  -b "bbsgo_token=<YOUR_TOKEN>" \
  -d "name=世界杯" \
  -d "description=世界杯相关话题"
```

### 返回

成功返回 `TagResponse`：

```json
{
  "id": 1,
  "name": "世界杯"
}
```

### 常见错误

- `NotLogin`：未登录。
- `tag name is required`：标签名称为空。
- `tag name length must be <= 32`：标签名称过长。
- `tag description length must be <= 1024`：标签描述过长。

## 分页查询标签

- 方法：`GET`
- 路径：`/api/tag/tags`
- 认证：否

### Query 参数

- `page` (int, 可选)：页码，默认 `1`。
- `limit` (int, 可选)：每页数量，默认 `20`，最大 `200`。
- `keyword` (string, 可选)：按标签名称模糊搜索。

### 示例

```bash
curl "http://localhost:8082/api/tag/tags?page=1&limit=20&keyword=世界"
```

### 返回

分页结构，`results` 为 `TagResponse` 数组：

```json
{
  "results": [
    {
      "id": 1,
      "name": "世界杯"
    }
  ],
  "page": 1,
  "limit": 20,
  "total": 1
}
```

## 标签详情

- 方法：`GET`
- 路径：`/api/tag/{tagId}`
- 认证：否

### 返回

成功返回 `TagResponse`；标签不存在返回 `tag not found`。

## 标签自动完成

- 方法：`POST`
- 路径：`/api/tag/autocomplete`
- 认证：否

### Form 参数

- `input` (string, 必填)：输入关键字。

### 返回

最多返回 6 个正常状态的标签，用于话题编辑页输入联想。
