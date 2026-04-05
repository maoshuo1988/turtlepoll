# 用户管理（Admin User）

本模块接口由 `internal/controllers/admin/user_controller.go` 提供，并在 `internal/server/router.go` 中注册到：

- `/api/admin/user`

说明：

- **认证**：需要管理员权限（`AdminMiddleware`）
- **owner-only**：涉及“授权/取消授权”等高危权限变更时，额外要求 `owner`（见 `internal/middleware/admin_middleware.go` 的路径规则）
- **审计**：关键操作会写入操作日志（`/api/admin/operate-log/**`）

---

## 接口列表（/api/admin/user）

### 1) 用户列表/搜索（分页）

- **接口**：`GET /api/admin/user/list`
- **认证**：需要管理员权限
- **参数（query）**（以实现为准）：
  - `id`: int64（可选）
  - `nickname`: string（可选，模糊匹配）
  - `email`: string（可选）
  - `username`: string（可选）
  - `type`: int（可选）
  - `page`: int（可选）
  - `limit`: int（可选）
- **返回**：分页 `web.PageResult`

> 对应实现：`UserController.AnyList()`。

---

### 2) 授权为管理员（owner-only）

- **接口**：`POST /api/admin/user/grant_admin`
- **认证**：需要 `owner`
- **请求格式**：表单
- **参数（form）**：
  - `userId`: int64（必填）
- **幂等性**：重复授权会直接成功（不重复插入角色关系）
- **限制**：不允许给自己授权（避免误操作）
- **返回**：成功 `web.JsonSuccess()`

会记录 operate-log：

- `opType=update`
- `dataType=user`
- `dataId=userId`
- `description=grant_admin`

> 对应实现：`UserController.PostGrant_admin()`。

---

### 3) 取消管理员（owner-only）

- **接口**：`POST /api/admin/user/revoke_admin`
- **认证**：需要 `owner`
- **请求格式**：表单
- **参数（form）**：
  - `userId`: int64（必填）
- **幂等性**：目标用户没有 admin 角色时直接成功
- **限制**：不允许对自己执行取消管理员（避免把自己锁死）
- **返回**：成功 `web.JsonSuccess()`

会记录 operate-log：

- `opType=update`
- `dataType=user`
- `dataId=userId`
- `description=revoke_admin`

> 对应实现：`UserController.PostRevoke_admin()`。

---

## 关联接口（不在 /api/admin/user 下，但常用于“用户管理”页面）

- 禁言/解禁：`POST /api/user/forbidden`
  - 详见：`docs/api/user.md#11-禁言管理员`

- 后台铸币（加金币）：`POST /api/admin/coin/mint`
  - 详见：`docs/api/coin.md#4-管理员铸币给用户加金币`
