# 节点管理（Admin TopicNode）

本模块接口由 `internal/controllers/admin/topic_node_controller.go` 提供，并在 `internal/server/router.go` 中注册到：

- `/api/admin/topic-node`

说明：

- **认证**：需要管理员权限（`AdminMiddleware`）
- **用途**：运营后台维护帖子节点（板块）信息与排序。

---

## 接口列表（/api/admin/topic-node）

### 1) 节点列表（用于后台管理）

- **接口**：`GET /api/admin/topic-node/list`
- **认证**：需要管理员权限
- **参数（query）**：
  - `name`: string（可选，模糊匹配）
- **返回**：节点数组（按 `sort_no asc, id desc`）

对应实现：`TopicNodeController.AnyList()`。

---

### 2) 节点详情

- **接口**：`GET /api/admin/topic-node/by/{id}`
- **认证**：需要管理员权限
- **路径参数**：
  - `id`: int64
- **返回**：`TopicNode`

对应实现：`TopicNodeController.GetBy(id)`。

---

### 3) 创建节点

- **接口**：`POST /api/admin/topic-node/create`
- **认证**：需要管理员权限
- **请求格式**：表单
- **参数（form）**：以 `TopicNode` 为准（常见：`name/description/logo/status` 等）
- **返回**：创建后的 `TopicNode`

说明：服务端会自动：

- 设置 `sortNo = GetNextSortNo()`
- 设置 `createTime = now`

对应实现：`TopicNodeController.PostCreate()`。

---

### 4) 更新节点

- **接口**：`POST /api/admin/topic-node/update`
- **认证**：需要管理员权限
- **请求格式**：表单
- **参数（form）**：
  - `id`: int64（必填）
  - 其他字段以 `TopicNode` 为准
- **返回**：更新后的 `TopicNode`

对应实现：`TopicNodeController.PostUpdate()`。

---

### 5) 节点导航（给前台使用的节点树/列表）

- **接口**：`GET /api/admin/topic-node/nodes`
- **认证**：需要管理员权限
- **返回**：`services.TopicNodeService.GetNodes()` 的结果

对应实现：`TopicNodeController.GetNodes()`。

---

### 6) 更新排序

- **接口**：`POST /api/admin/topic-node/update_sort`
- **认证**：需要管理员权限
- **请求格式**：JSON 数组（节点 id 列表，按新顺序排列）
- **请求体示例**：

```json
[3, 8, 1, 10]
```

- **返回**：成功 `web.JsonSuccess()`

对应实现：`TopicNodeController.PostUpdate_sort()`。
