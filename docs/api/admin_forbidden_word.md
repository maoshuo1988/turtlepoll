# 敏感词管理（Admin ForbiddenWord）

本模块接口由 `internal/controllers/admin/forbidden_word_controller.go` 提供，并在 `internal/server/router.go` 中注册到：

- `/api/admin/forbidden-word`

说明：

- **认证**：需要管理员权限（`AdminMiddleware`）
- **用途**：运营后台维护敏感词/违禁词库。

---

## 接口列表（/api/admin/forbidden-word）

### 1) 敏感词列表（分页）

- **接口**：`GET /api/admin/forbidden-word/list`
- **认证**：需要管理员权限
- **参数（query）**：
  - `type`: int（可选）
  - `word`: string（可选，模糊匹配）
  - `status`: int（可选）
  - `page`: int（可选）
  - `limit`: int（可选）
- **返回**：分页 `web.PageResult`

对应实现：`ForbiddenWordController.AnyList()`。

---

### 2) 敏感词详情

- **接口**：`GET /api/admin/forbidden-word/by/{id}`
- **认证**：需要管理员权限
- **路径参数**：
  - `id`: int64
- **返回**：`ForbiddenWord`

对应实现：`ForbiddenWordController.GetBy(id)`。

---

### 3) 创建敏感词

- **接口**：`POST /api/admin/forbidden-word/create`
- **认证**：需要管理员权限
- **请求格式**：表单
- **参数**：以 `models.ForbiddenWord` 为准（常见：`type/word/status`）
- **返回**：创建后的 `ForbiddenWord`

说明：服务端会自动设置 `createTime = now`。

对应实现：`ForbiddenWordController.PostCreate()`。

---

### 4) 更新敏感词

- **接口**：`POST /api/admin/forbidden-word/update`
- **认证**：需要管理员权限
- **请求格式**：表单
- **参数（form）**：
  - `id`: int64（必填）
  - 其他字段以 `models.ForbiddenWord` 为准
- **返回**：更新后的 `ForbiddenWord`

对应实现：`ForbiddenWordController.PostUpdate()`。

---

### 5) 删除敏感词

- **接口**：`POST /api/admin/forbidden-word/delete`
- **认证**：需要管理员权限
- **请求格式**：表单
- **参数（form）**：
  - `id`: int64
- **返回**：成功 `web.JsonSuccess()`

对应实现：`ForbiddenWordController.PostDelete()`。
