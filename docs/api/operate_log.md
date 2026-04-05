# 操作审计日志（OperateLog）

本模块接口由 `internal/controllers/admin/operate_log_controller.go` 提供，并在 `internal/server/router.go` 中注册到：

- `/api/admin/operate-log`

说明：

- **认证**：需要管理员权限（`AdminMiddleware`）
- **用途**：运营后台「最近操作 / 审计追溯」。

---

## 数据模型摘要

字段以 `internal/models/models.go` 中 `OperateLog` 为准，常见字段：

- `id`: int64
- `userId`: int64（操作者）
- `opType`: string（操作类型，如 `create/update/delete` 等；以写入方为准）
- `dataType`: string（实体类型，如 `user/predictMarket/battle/...`）
- `dataId`: int64（实体 id）
- `description`: string（操作描述，如 `grant_admin/revoke_admin/market_settle/...`）
- `ip`: string
- `userAgent`: string
- `referer`: string
- `createTime`: int64（秒级时间戳）

---

## 接口列表（/api/admin/operate-log）

### 1) 操作日志列表（分页）

- **接口**：`GET /api/admin/operate-log/list`
- **认证**：需要管理员权限
- **参数（query）**：
  - `user_id`: int64（可选）
  - `op_type`: string（可选）
  - `data_type`: string（可选）
  - `data_id`: int64（可选）
  - `page`: int（可选，默认 1）
  - `limit`: int（可选，默认 20）
- **返回**：分页列表 `web.PageResult`

> 说明：该接口由 `OperateLogController.AnyList()` 实现，按 `id desc` 排序。

---

### 2) 操作日志详情

- **接口**：`GET /api/admin/operate-log/by/{id}`
- **认证**：需要管理员权限
- **路径参数**：
  - `id`: int64
- **返回**：单条 `OperateLog`

> 说明：该接口由 `OperateLogController.GetBy(id)` 实现。
