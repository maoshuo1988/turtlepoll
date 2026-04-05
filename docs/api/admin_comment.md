# 评论管理（Admin Comment）

本模块接口由 `internal/controllers/admin/comment_controller.go` 提供，并在 `internal/server/router.go` 中注册到：

- `/api/admin/comment`

说明：

- **认证**：需要管理员权限（`AdminMiddleware`）
- **用途**：运营后台按用户/实体维度检索评论、执行删除。

---

## 接口列表（/api/admin/comment）

### 1) 评论列表/检索（分页）

- **接口**：`GET /api/admin/comment/list`
- **认证**：需要管理员权限
- **参数（query）**（至少填写一个筛选条件，否则返回空成功）：
  - `id`: int64（可选）
  - `userId`: int64（可选）
  - `entityType`: string（可选；例如 `topic/article/battle/...`）
  - `entityId`: int64（可选）
  - `status`: int（可选）
  - `page`: int（可选）
  - `limit`: int（可选）
- **返回**：分页 `web.PageResult`

返回列表每项会补充：

- `user`: `UserInfo`（评论作者）
- `content`: string（若是 markdown，会转为 HTML）
- `imageList`: array（若存在图片）

对应实现：`CommentController.AnyList()`。

---

### 2) 删除评论

- **接口**：`POST /api/admin/comment/delete/{id}`
- **认证**：需要管理员权限
- **路径参数**：
  - `id`: int64
- **返回**：成功 `web.JsonSuccess()`

对应实现：`CommentController.PostDeleteBy(id)`。
