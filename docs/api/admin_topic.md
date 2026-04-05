# 帖子管理（Admin Topic）

本模块接口由 `internal/controllers/admin/topic_controller.go` 提供，并在 `internal/server/router.go` 中注册到：

- `/api/admin/topic`

说明：

- **认证**：需要管理员权限（`AdminMiddleware`）
- **ID 规则**：admin 端多数接口使用 **明文 int64 id**（不使用 encode id），以兼容后台列表/管理操作。

---

## 接口列表（/api/admin/topic）

### 1) 帖子列表/搜索（分页）

- **接口**：`GET /api/admin/topic/list`
- **认证**：需要管理员权限
- **参数（query）**（以实现为准）：
  - `id`: string（可选；支持 encode id，会自动 decode 为 int64）
  - `userId`: string（可选；支持 encode id，会自动 decode 为 int64）
  - `status`: int（可选）
  - `recommend`: bool（可选）
  - `title`: string（可选，模糊匹配）
  - `page`: int（可选）
  - `limit`: int（可选）
- **返回**：分页 `web.PageResult`

返回列表每项在 `Topic` 基础上额外补充：

- `id`: int64（明文）
- `idEncode`: string（encode 后 id，便于和用户侧接口联动排查）
- `status`: int
- `vote`: object（可选；如果该 Topic 绑定投票）

对应实现：`TopicController.AnyList()`。

---

### 2) 帖子详情

- **接口**：`GET /api/admin/topic/by/{id}`
- **认证**：需要管理员权限
- **路径参数**：
  - `id`: int64
- **返回**：`Topic`

对应实现：`TopicController.GetBy(id)`。

---

### 3) 删除帖子

- **接口**：`POST /api/admin/topic/delete`
- **认证**：需要管理员权限
- **请求格式**：表单
- **参数（form）**：
  - `id`: int64（必填；明文 id）

对应实现：`TopicController.PostDelete()`。

---

### 4) 恢复删除（取消删除）

- **接口**：`POST /api/admin/topic/undelete`
- **认证**：需要管理员权限
- **请求格式**：表单
- **参数（form）**：
  - `id`: int64（必填）

对应实现：`TopicController.PostUndelete()`。

---

### 5) 审核通过（将状态置为 OK）

- **接口**：`POST /api/admin/topic/audit`
- **认证**：需要管理员权限
- **请求格式**：表单
- **参数（form）**：
  - `id`: int64（必填）

对应实现：`TopicController.PostAudit()`。

---

### 6) 推荐 / 取消推荐

- **推荐**：`POST /api/admin/topic/recommend`
- **取消推荐**：`DELETE /api/admin/topic/recommend`
- **认证**：需要管理员权限
- **请求格式**：表单
- **参数（form）**：
  - `id`: int64（必填）

对应实现：`TopicController.PostRecommend()` / `TopicController.DeleteRecommend()`。

---

## 与用户侧 Topic API 的关系

- 用户侧推荐/置顶能力主要在 `docs/api/topic.md`（路由 `/api/topic`）。
- admin 侧提供的是“后台管理风格”的 CRUD（路由 `/api/admin/topic`）。
- 置顶目前仍复用用户侧管理员接口（`POST /api/topic/sticky/{topicId}`），admin 侧暂无 `/api/admin/topic/pin|unpin`。
