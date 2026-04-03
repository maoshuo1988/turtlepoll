# 评论系统

本模块接口由 `internal/controllers/api/comment_controller.go` 提供，并在 `internal/server/router.go` 中注册到：

- `/api/comment`

说明：挂在 `/api` 下，默认经过 `AuthMiddleware`（需要登录）。

---

## 功能

- 评论列表（按 entityType/entityId）
- 回复列表（按 commentId）
- 发布评论（含反垃圾检测）

---

## 接口列表（/api/comment）

### 1) 评论列表
- **接口**：`GET /api/comment/comments`
- **功能**：获取评论列表
- **参数（query）**（来自 `CommentController.GetComments`）：
  - `cursor`: int64（可选）
  - `entityType`: string
  - `entityId`:（从请求中读取 id，见 `common.GetID(c.Ctx, "entityId")`）
- **返回**：`web.JsonCursorData(...)`
  - `data` 为 `render.BuildComments(comments, currentUser, true, false)`
  - `cursor`/`hasMore` 游标分页

### 2) 回复列表
- **接口**：`GET /api/comment/replies`
- **功能**：获取某条评论的回复列表
- **参数（query）**：
  - `cursor`: int64（可选）
  - `commentId`: int64
- **返回**：`web.JsonCursorData(...)`

#### cursor 参数有什么用？

`/api/comment/replies` 的 `cursor` 用于 **二级回复的游标分页**，它的实现逻辑在 `CommentService.GetReplies`：

- 回复列表排序：`Asc("id")`（按 id 从小到大）
- 当 `cursor > 0` 时：追加条件 `id > cursor`
- 服务端返回里的 `cursor`：本次结果最后一条回复的 `id`（即 `nextCursor = comments[len-1].Id`）
- `hasMore`：当返回条数 `>= limit`（控制器里固定传 `limit=10`）就认为后面可能还有

也就是说：

- **第一次请求**：不传 `cursor`（或传 0）→ 拿到最早的一批回复（前 10 条）
- **加载更多**：把上一次返回的 `cursor` 原样带上 → 拿到 `id` 更大的下一批回复

注意：`/api/comment/comments`（一级评论列表）是 `Desc("id")` + `id < cursor`，属于“向下翻更早的评论”；而 replies 是 `Asc("id")` + `id > cursor`，属于“向后追加更新的回复”。

### 3) 发布评论
- **接口**：`POST /api/comment/create`
- **功能**：发布评论
- **前置**：
  - 登录校验由 AuthMiddleware 处理
  - 额外发帖状态校验：`UserService.CheckPostStatus(user)`
  - 反垃圾：`spam.CheckComment(user, form)`
- **请求体**：`req.GetCreateCommentForm(c.Ctx)`（以该函数实现为准，当前实现为「表单参数」解析）

#### 参数明细（CreateCommentForm）

对应结构：`req.CreateCommentForm`（`internal/models/req/request.go`）

- `entityType` (string，必填)
发布评论的入参来自 `req.GetCreateCommentForm`，当前实现**从表单字段读取**（`params.FormValue(...)`），字段如下：

- `entityType` (string，必填)：评论挂载的目标类型
  - 常见取值：
    - `article`（`constants.EntityArticle`）
    - `topic`（`constants.EntityTopic`）
    - `comment`（`constants.EntityComment`，用于二级回复）

补充约定（turtlepoll 定制）：

- 当 `entityType=topic` 时，服务端除了更新 `t_topic.comment_count` 外，还会尝试把 **同 ID 的预测市场**（`PredictMarket.id == topicId`）对应的 `PredictContext.heat` 做 `+1`，用于驱动热榜 `/api/football/predict_context/hot`。
- 如果该 `PredictMarket` 尚未生成 `PredictContext`（例如尚未同步/创建预测上下文），热度更新会被跳过，不影响评论发布成功。
- `entityId` (int64，必填)：目标实体 id
  - 例如：评论话题时传 topicId；回复某条评论时传 commentId
- `content` (string，必填)：评论内容（服务端会 `TrimSpace`）
- `imageList` (string，可选)：图片列表（**注意：这里是一个 JSON 数组字符串**）
- `quoteId` (int64，可选，默认 0)：引用评论 id

下面给一个更可读的示例（以 JSON 展示字段结构；实际传参可用 `application/x-www-form-urlencoded` / `multipart/form-data`，其中 `imageList` 字段的值就是这段 JSON 数组的字符串）：

```json
{
  "entityType": "topic",
  "entityId": 123,
  "content": "这场比赛我支持正方",
  "imageList": [
    { "url": "https://example.com/img1.png" },
    { "url": "https://example.com/img2.png" }
  ],
  "quoteId": 0
}
```

补充说明：

- `imageList` 在服务端是通过 `GetImageList(ctx, "imageList")` 解析的，它期望你传入的 `imageList` 是一个 JSON 数组字符串，元素格式为 `{ "url": "..." }`。
- `userAgent` / `ip` 字段由服务端从请求头和请求地址自动采集，你不需要传。

服务端自动补充（无需传）：

- `userAgent`：从请求头读取
- `ip`：从请求获取
- **返回**：`render.BuildComment(comment)`

---

## 可能错误

- 发帖状态异常（禁言/限制等）：`UserService.CheckPostStatus` 返回的错误
- 命中垃圾策略：`spam.CheckComment` 返回的错误
- 发布失败：`CommentService.Publish` 返回的错误
