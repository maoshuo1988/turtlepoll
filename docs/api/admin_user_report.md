# 举报审核（Admin UserReport）

本模块接口由 `internal/controllers/admin/user_report_controller.go` 提供，并在 `internal/server/router.go` 中注册到：

- `/api/admin/user-report`

说明：

- **认证**：需要管理员权限（`AdminMiddleware`）
- **用途**：运营后台对用户举报进行列表查看与审核处理。

---

## 数据模型摘要

字段以 `internal/models/models.go` 中 `UserReport` 为准。常见字段（可能随版本演进）：

- `id`: int64
- `userId`: int64（举报人；允许为空/0，表示匿名举报）
- `dataType`: string（举报数据类型，例如 `topic/comment/user/...`）
- `dataId`: int64
- `reason`: string
- `status`: int（可选；若模型存在该字段）

审核相关字段在当前实现里由 `POST /update` 直接写表单字段，推荐约定：

- `auditStatus`: int（例如：0=待处理，1=通过，2=驳回，3=忽略；具体值你们可在后台枚举）
- `auditUserId`: int64（审核人 admin）
- `auditTime`: int64（秒级时间戳）
- `auditRemark`: string（可选）

---

## 接口列表（/api/admin/user-report）

### 1) 举报列表（分页）

- **接口**：`GET /api/admin/user-report/list`
- **认证**：需要管理员权限
- **参数**：
  - `page`: int（可选）
  - `limit`: int（可选）
- **返回**：分页 `web.PageResult`

对应实现：`UserReportController.AnyList()`（按 `id desc`）。

---

### 2) 举报详情

- **接口**：`GET /api/admin/user-report/by/{id}`
- **认证**：需要管理员权限
- **路径参数**：
  - `id`: int64
- **返回**：`UserReport`

对应实现：`UserReportController.GetBy(id)`。

---

### 3) 更新举报（审核/处理）

- **接口**：`POST /api/admin/user-report/update`
- **认证**：需要管理员权限
- **请求格式**：表单
- **参数（form）**：
  - `id`: int64（必填）
  - 其他：任意 `UserReport` 表字段（按需更新）

对应实现：`UserReportController.PostUpdate()`。

> 提示：该接口是“通用更新”，你们可以在后台 UI 层面限制只允许更新审核相关字段，避免误改举报原始内容（如 reason/dataId）。

---

## 用户侧举报入口（对照）

- 用户提交举报：`POST /api/user-report/submit`
- 详见：`docs/api/admin.md` 的「用户侧举报」条目。
