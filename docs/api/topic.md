# Topic / 帖子系统 API

> 说明：本文档基于实际代码实现整理（`internal/controllers/api/topic_controller.go`），路由挂载于 `/api/topic`。
>
> - 认证：除少量公开接口外，涉及发帖/编辑/删除/收藏/置顶/推荐等需要登录（cookie `token`），未登录通常返回 `NotLogin`。
> - 分页：列表类接口普遍使用 cursor 分页（`cursor` + `hasMore`）。
> - 主题 id：路径参数通常为编码后的字符串 id（`idcodec.Encode/Decode`）。

## 获取节点导航

- 方法：GET
- 路径：`/api/topic/node_navs`
- 认证：否

### 返回

返回节点数组（包含内置节点 + 自定义节点）：

- 内置节点：
  - `id=0`：最新
  - `id=-1`：推荐
  - `id=-2`：关注

## 获取节点列表

- 方法：GET
- 路径：`/api/topic/nodes`
- 认证：否

### 返回

`[]NodeResponse`

## 获取节点信息

- 方法：GET
- 路径：`/api/topic/node`
- 认证：否

### Query 参数

- `nodeId` (int64, 必填)：节点 id；支持内置节点 id（0/-1/-2）。

### 返回

- node 存在：返回 `NodeResponse`
- node 不存在：返回错误 `common.not_found`

## 发表帖子

- 方法：POST
- 路径：`/api/topic/create`
- 认证：是
- Content-Type：`application/json`

### Body(JSON)

对应结构：`req.CreateTopicForm`（`internal/models/req/request.go`）

下面用 JSON 示例展示字段结构（字段含义以代码/前端约定为准）：

```json
{
  "type": 0,
  "nodeId": 1,
  "title": "标题",
  "content": "正文内容（markdown 或 text）",
  "contentType": "markdown",
  "hideContent": "隐藏内容（可选）",
  "tags": ["go", "世界杯"],
  "imageList": [
    { "url": "https://example.com/1.png" }
  ],
  "vote": null,
  "captchaId": "",
  "captchaCode": "",
  "captchaProtocol": 0
}
```

其中：

- `type`：帖子类型（代码里会把 tweet 强制为纯文本）
- `imageList`：数组，每项结构为 `{ "url": "..." }`
- `vote`：投票信息（可选），结构来自 `req.CreateVoteForm`，示例：

```json
{
  "type": 0,
  "title": "你支持哪一方？",
  "expiredAt": 0,
  "voteNum": 1,
  "options": [
    { "content": "正方" },
    { "content": "反方" }
  ]
}
```

服务端会补充：

- `ip`：请求 IP
- `userAgent`

### 返回

成功：返回 `SimpleTopic`（`render.BuildSimpleTopic`）。

### 常见错误

- `NotLogin`：未登录
- `UserService.CheckPostStatus`：禁言/限制发帖
- `spam.CheckTopic`：反垃圾校验失败

## 编辑前获取详情

- 方法：GET
- 路径：`/api/topic/edit/{topicId}`
- 认证：是

### Path 参数

- `topicId`：编码后的 topic id（string）

### 返回

返回编辑所需字段：

- `id`(string)：编码 id
- `nodeId`(int64)
- `title`(string)
- `content`(string)
- `contentType`
- `hideContent`(string)
- `tags`([]string)

### 常见错误

- `common.not_found`：帖子不存在/状态非 OK
- `topic.type_not_supported`：非普通帖子类型不支持编辑
- `topic.no_permission`：非作者且非管理员/站长

## 编辑帖子

- 方法：POST
- 路径：`/api/topic/edit/{topicId}`
- 认证：是
- Content-Type：`application/json`

### Body(JSON)

对应结构：`req.EditTopicForm`

```json
{
  "nodeId": 1,
  "title": "更新后的标题",
  "content": "更新后的内容",
  "hideContent": "更新后的隐藏内容（可选）",
  "tags": ["tag1", "tag2"]
}
```

### 返回

成功：返回 `SimpleTopic`。

### 常见错误

- `topic.no_permission`：无权限
- `common.not_found`：帖子不存在

## 删除帖子

- 方法：POST
- 路径：`/api/topic/delete/{topicId}`
- 认证：是

### 返回

成功：`success=true`（帖子不存在也会直接 success）。

### 常见错误

- `topic.no_permission`：无权限

## 设置推荐（管理员）

- 方法：POST
- 路径：`/api/topic/recommend/{topicId}`
- 认证：是（且需要 owner/admin）

### Form 参数

- `recommend` (bool, 必填)：`true/false`

### 返回

成功：`success=true`

### 常见错误

- `NotLogin`
- `topic.no_permission`

## 帖子详情

- 方法：GET
- 路径：`/api/topic/{topicId}`
- 认证：否（但审核状态可能限制）

### 行为

- 自动增加浏览量：`TopicService.IncrViewCount(topicId)`

### 常见错误

- `common.not_found`：被删除/不存在
- `403 topic.under_review`：审核中，仅作者或管理员可见

## 点赞用户（最近 5 个）

- 方法：GET
- 路径：`/api/topic/recentlikes/{topicId}`
- 认证：否

### 返回

`[]UserInfo`（最多 5 个）

## 最新帖子（固定 10 条）

- 方法：GET
- 路径：`/api/topic/recent`
- 认证：否

### 返回

`[]SimpleTopic`（按 id desc，limit=10）

## 用户帖子列表（cursor 分页）

- 方法：GET
- 路径：`/api/topic/user_topics`
- 认证：否

### Query/Form 参数

- `userId` (int64, 必填)
- `cursor` (int64, 可选，默认 0)

### 返回

`JsonCursorData([]SimpleTopic, cursor, hasMore)`

## 帖子列表（cursor 分页）

- 方法：GET
- 路径：`/api/topic/topics`
- 认证：部分（当 `nodeId=-2` 关注节点时需要登录）

### Query/Form 参数

- `nodeId` (int64, 可选，默认 0)：节点 id，支持内置节点（0/-1/-2）
- `cursor` (int64, 可选，默认 0)

### 返回

- 首页（cursor<=0）会拼接最多 3 条置顶帖（`GetStickyTopics(nodeId, 3)`）
- 正常列表会把 `Sticky=false`，避免渲染置顶标记

返回：`JsonCursorData([]SimpleTopic, cursor, hasMore)`

### 常见错误

- `NotLogin`：nodeId=-2(关注) 且未登录

## 标签帖子列表（cursor 分页）

- 方法：GET
- 路径：`/api/topic/tag_topics`
- 认证：否

### Query/Form 参数

- `tagId` (int64, 必填)
- `cursor` (int64, 可选，默认 0)

### 返回

`JsonCursorData([]SimpleTopic, cursor, hasMore)`

## 收藏帖子

- 方法：GET
- 路径：`/api/topic/favorite/{topicId}`
- 认证：是

### 返回

成功：`success=true`

## 设置置顶（管理员）

- 方法：POST
- 路径：`/api/topic/sticky/{topicId}`
- 认证：是（且需要 owner/admin）

### Form 参数

- `sticky` (bool, 可选，默认 false)

### 返回

成功：`success=true`

## 获取隐藏内容

- 方法：GET
- 路径：`/api/topic/hide_content`
- 认证：否

### Query/Form 参数

- `topicId` (int64, 必填)

### 返回

```json
{
  "exists": true,
  "show": false,
  "content": "<p>...</p>"
}
```

含义：

- `exists`：是否存在隐藏内容
- `show`：是否有权限查看（作者或已评论过该帖子）
- `content`：当 `show=true` 时返回 HTML（服务端 markdown 渲染后的内容）
