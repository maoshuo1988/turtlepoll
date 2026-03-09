# TurtlePoll 页面接口文档（按当前项目风格整理）

## 1. 文档范围
本文件面向截图对应的社区首页/话题流页面，整理：

1. 现有后端可直接复用的接口。
2. 每个接口的请求字段与响应字段说明。
3. 与项目一致的返回包裹格式、鉴权方式、分页方式。
4. 现有接口不足时，给出风格一致的建议新增接口。

## 2. 统一约定（与项目一致）

### 2.1 基础路径
- 业务接口统一前缀：`/api`

### 2.2 鉴权
- 已登录请求可通过以下任一方式携带 Token：
1. Header: `Authorization: Bearer <token>` 或 `Authorization: <token>`
2. Query/Form 参数：`_user_token=<token>`
3. Cookie：`bbsgo_token=<token>`

### 2.3 统一响应结构 `JsonResult`
所有接口外层均为：

| 字段 | 类型 | 说明 |
|---|---|---|
| `errorCode` | number | 错误码。`0` 通常表示成功；业务错误如未登录可能返回 `1` 等。 |
| `message` | string | 错误或提示信息。成功时可能为空。 |
| `data` | any | 实际业务数据。 |
| `success` | boolean | 是否成功。 |

### 2.4 游标分页结构 `CursorResult`
当 `data` 是游标分页时：

| 字段 | 类型 | 说明 |
|---|---|---|
| `results` | array | 当前页数据列表。 |
| `cursor` | string | 下一页游标。首屏通常传 `0`，返回值继续透传。 |
| `hasMore` | boolean | 是否还有下一页。 |

### 2.5 ID 规则
- 路径上的资源 ID 常为**编码字符串**（如 `topic.id`）。
- 多数 `entityId/userId` 参数支持“数字 ID”或“编码 ID”两种输入。

## 3. 页面模块与接口映射

| 页面模块 | 接口 |
|---|---|
| 顶部站点信息/公告 | `GET /api/config/configs` |
| 顶部消息提醒 | `GET /api/user/msg_recent` |
| 左侧用户卡片 | `GET /api/user/current` |
| 左侧勋章展示 | `GET /api/badge/badges` |
| 签到面板 | `GET /api/checkin/checkin` `POST /api/checkin/checkin` `GET /api/checkin/rank` |
| 节点导航（最新/推荐/关注/节点） | `GET /api/topic/node_navs` |
| 主列表（话题卡片/预测卡片） | `GET /api/topic/topics` |
| 卡片点赞 | `POST /api/like/like` `POST /api/like/unlike` |
| 投票提交 | `POST /api/vote/cast` |
| 右侧积分榜 | `GET /api/user/score/rank` |
| 右侧友链 | `GET /api/link/top_links` |
| 搜索页结果 | `GET /api/search/topic` |
| 最新热点（可复用） | `GET /api/topic/recent` |
| 用户认证（登录/注册） | `GET /api/captcha/request_angle` `POST /api/login/signup` `POST /api/login/signin` `POST /api/login/login_sms_code` `POST /api/login/login_sms` `GET /api/login/signout` |

## 4. 公共对象字段定义

### 4.1 `UserInfo`
| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | string | 用户 ID（编码字符串）。 |
| `type` | number | 用户类型。`0` 普通用户，`1` 员工用户。 |
| `nickname` | string | 昵称。 |
| `avatar` | string | 头像原图 URL。 |
| `smallAvatar` | string | 小图头像 URL。 |
| `gender` | string | 性别：`Male`/`Female`。 |
| `birthday` | string\|null | 生日（时间格式）。 |
| `topicCount` | number | 发帖数。 |
| `commentCount` | number | 评论数。 |
| `fansCount` | number | 粉丝数。 |
| `followCount` | number | 关注数。 |
| `score` | number | 积分。 |
| `exp` | number | 经验值。 |
| `level` | number | 等级。 |
| `levelTitle` | string | 等级称号。 |
| `description` | string | 个性签名。 |
| `createTime` | number | 创建时间（Unix 秒时间戳）。 |
| `forbidden` | boolean | 是否禁言。 |
| `followed` | boolean | 当前登录用户是否已关注该用户。 |
| `expProgress` | object | 经验进度对象，见下表。 |

### 4.2 `ExpProgress`
| 字段 | 类型 | 说明 |
|---|---|---|
| `currentExp` | number | 当前累计经验值。 |
| `level` | number | 当前等级。 |
| `levelTitle` | string | 当前等级称号。 |
| `expInCurrentLevel` | number | 当前等级区间内已获得经验。 |
| `expNeedForNextLevel` | number | 升到下一等级所需经验。 |
| `expProgressPercent` | number | 当前等级经验进度百分比（0-100）。 |
| `isMaxLevel` | boolean | 是否满级。 |

### 4.3 `NodeResponse`
| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | number | 节点 ID。内置节点：`0`最新，`-1`推荐，`-2`关注。 |
| `name` | string | 节点名称。 |
| `logo` | string | 节点图标 URL（内置节点可能为空）。 |
| `description` | string | 节点描述（内置节点可能为空）。 |

### 4.4 `TagResponse`
| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | number | 标签 ID。 |
| `name` | string | 标签名。 |

### 4.5 `ImageInfo`
| 字段 | 类型 | 说明 |
|---|---|---|
| `url` | string | 图片详情图 URL。 |
| `preview` | string | 图片预览图 URL。 |

### 4.6 `VoteResponse`
| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | number | 投票 ID。 |
| `type` | number | 投票类型：`1` 单选，`2` 多选。 |
| `title` | string | 投票标题。 |
| `expiredAt` | number | 截止时间（Unix 秒时间戳）。 |
| `voteNum` | number | 最多可选项数量。 |
| `optionCount` | number | 选项总数。 |
| `voteCount` | number | 总投票数。 |
| `expired` | boolean | 是否已过期。 |
| `voted` | boolean | 当前用户是否已投票。 |
| `optionIds` | number[] | 当前用户已选项 ID 列表。 |
| `options` | object[] | 选项列表。 |

`options[]` 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | number | 选项 ID。 |
| `content` | string | 选项内容。 |
| `sortNo` | number | 排序号。 |
| `voteCount` | number | 当前选项票数。 |
| `percent` | number | 当前选项得票占比（百分比）。 |
| `voted` | boolean | 当前用户是否选择该选项。 |

### 4.7 `TopicResponse`（列表卡片）
| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | string | 话题 ID（编码字符串）。 |
| `type` | number | 话题类型：`0` 帖子，`1` 动态。 |
| `user` | UserInfo | 作者信息。 |
| `node` | NodeResponse\|null | 节点信息。 |
| `tags` | TagResponse[]\|null | 标签列表。 |
| `title` | string | 标题（帖子类型有值）。 |
| `summary` | string | 摘要（帖子类型常用）。 |
| `content` | string | 内容（动态或详情常用）。 |
| `imageList` | ImageInfo[] | 图片列表。 |
| `lastCommentTime` | number | 最后评论时间。 |
| `viewCount` | number | 浏览量。 |
| `commentCount` | number | 评论数。 |
| `likeCount` | number | 点赞数。 |
| `liked` | boolean | 当前用户是否已点赞。 |
| `createTime` | number | 创建时间（Unix 秒时间戳）。 |
| `recommend` | boolean | 是否推荐。 |
| `recommendTime` | number | 推荐时间。 |
| `sticky` | boolean | 是否置顶。 |
| `stickyTime` | number | 置顶时间。 |
| `status` | number | 状态：`0` 正常，`1` 删除，`2` 审核中。 |
| `favorited` | boolean | 当前用户是否已收藏。 |
| `ipLocation` | string | IP 属地。 |
| `vote` | VoteResponse\|null | 投票信息。 |

## 5. 具体接口清单（字段级说明）

### 5.1 获取站点配置
- `GET /api/config/configs`
- 鉴权：否
- 请求参数：无

`data` 字段（页面常用）：

| 字段 | 类型 | 说明 |
|---|---|---|
| `siteTitle` | string | 站点标题。 |
| `siteDescription` | string | 站点描述。 |
| `siteLogo` | string | 站点 Logo。 |
| `siteNotification` | string | 站点公告（HTML 文本）。 |
| `recommendTags` | string[] | 推荐标签。 |
| `modules` | object | 模块开关。 |
| `modules.tweet` | boolean | 是否启用动态模块。 |
| `modules.topic` | boolean | 是否启用帖子模块。 |
| `modules.article` | boolean | 是否启用文章模块。 |
| `installed` | boolean | 是否已安装完成。 |
| `language` | string | 当前语言。 |
| `loginConfig` | object | 登录配置（公开部分）。 |
| `loginConfig.passwordLogin.enabled` | boolean | 是否允许账号密码登录。 |
| `loginConfig.weixinLogin.enabled` | boolean | 是否允许微信登录。 |
| `loginConfig.smsLogin.enabled` | boolean | 是否允许短信登录。 |
| `loginConfig.googleLogin.enabled` | boolean | 是否允许 Google 登录。 |

---

### 5.2 获取当前登录用户
- `GET /api/user/current`
- 鉴权：可选（未登录时返回 `success=true,data=null`）
- 请求参数：无
- 响应：`data` 为用户完整资料（`UserProfile`），其中页面高频使用 `UserInfo` 字段与 `expProgress`。

---

### 5.3 获取用户勋章列表
- `GET /api/badge/badges`
- 鉴权：可选
- Query 参数：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `userId` | string\|number | 否 | 用户 ID（编码或数字）。传入后会标记勋章是否已拥有。 |

响应 `data` 为数组，元素字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | number | 勋章 ID。 |
| `name` | string | 勋章唯一名称。 |
| `title` | string | 勋章标题。 |
| `description` | string | 勋章描述。 |
| `icon` | string | 勋章图标 URL。 |
| `sortNo` | number | 排序号。 |
| `status` | number | 状态。 |
| `owned` | boolean | 指定用户是否拥有。 |
| `worn` | boolean | 是否佩戴中。 |
| `obtainTime` | number | 获得时间（Unix 秒时间戳，未获得为 0）。 |

---

### 5.4 签到状态
- `GET /api/checkin/checkin`
- 鉴权：是（未登录返回 `data=null`）
- 请求参数：无

响应 `data` 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | number | 签到记录 ID。 |
| `userId` | number | 用户 ID。 |
| `dayName` | number | 最后签到日期（如 `20260307`）。 |
| `consecutiveDays` | number | 连续签到天数。 |
| `createTime` | number | 创建时间。 |
| `updateTime` | number | 更新时间。 |
| `checkIn` | boolean | 今日是否已签到。 |

---

### 5.5 执行签到
- `POST /api/checkin/checkin`
- 鉴权：是
- 请求参数：无
- 成功返回：`data=null`

---

### 5.6 今日签到排行
- `GET /api/checkin/rank`
- 鉴权：否
- 请求参数：无

响应 `data` 为数组，元素字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | number | 签到记录 ID。 |
| `userId` | number | 用户 ID。 |
| `dayName` | number | 最后签到日期。 |
| `consecutiveDays` | number | 连续签到天数。 |
| `createTime` | number | 创建时间。 |
| `updateTime` | number | 更新时间。 |
| `user` | UserInfo | 用户信息。 |

---

### 5.7 节点导航
- `GET /api/topic/node_navs`
- 鉴权：否
- 请求参数：无
- 响应：`data` 为 `NodeResponse[]`

---

### 5.8 话题列表（首页主流）
- `GET /api/topic/topics`
- 鉴权：可选（`nodeId=-2` 关注流时需要登录）
- Query 参数：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `cursor` | number | 否 | 游标，首屏传 `0`。 |
| `nodeId` | number | 否 | 节点 ID：`0` 最新，`-1` 推荐，`-2` 关注，`>0` 普通节点。 |

响应 `data` 为 `CursorResult`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `results` | TopicResponse[] | 话题卡片列表。 |
| `cursor` | string | 下一页游标。 |
| `hasMore` | boolean | 是否有更多。 |

---

### 5.9 最新帖子（可用于“最新热点”）
- `GET /api/topic/recent`
- 鉴权：否
- 请求参数：无
- 响应：`data` 为 `TopicResponse[]`（最多 10 条）

---

### 5.10 点赞
- `POST /api/like/like`
- 鉴权：是
- 请求参数（Form 或 Query）：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `entityType` | string | 是 | 实体类型：`topic`/`article`/`comment`。 |
| `entityId` | string\|number | 是 | 实体 ID（编码或数字）。 |

- 成功返回：`data=null`

---

### 5.11 取消点赞
- `POST /api/like/unlike`
- 鉴权：是
- 请求参数同 5.10
- 成功返回：`data=null`

---

### 5.12 投票提交
- `POST /api/vote/cast`
- 鉴权：是
- Content-Type：`application/json`
- 请求体：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `voteId` | number | 是 | 投票 ID。 |
| `optionIds` | number[] | 是 | 选项 ID 列表。单选时长度应为 1。 |

- 响应：`data` 为最新 `VoteResponse`

---

### 5.13 积分排行榜
- `GET /api/user/score/rank`
- 鉴权：否
- 请求参数：无
- 响应：`data` 为 `UserInfo[]`

---

### 5.14 友链（首页右侧）
- `GET /api/link/top_links`
- 鉴权：否
- 请求参数：无

响应 `data` 为数组，元素字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | number | 友链主键 ID。 |
| `linkId` | number | 友链 ID（与 `id` 相同，前端兼容字段）。 |
| `url` | string | 站点链接。 |
| `title` | string | 站点标题。 |
| `summary` | string | 站点简介。 |
| `createTime` | number | 创建时间。 |

---

### 5.15 顶部未读消息摘要
- `GET /api/user/msg_recent`
- 鉴权：可选（未登录时 `count=0,messages=[]`）
- 请求参数：无

响应 `data` 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `count` | number | 未读消息总数。 |
| `messages` | object[] | 最近未读消息（最多 3 条）。 |

`messages[]` 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | number | 消息 ID。 |
| `from` | UserInfo | 发送者。系统消息可能是系统用户。 |
| `userId` | number | 接收用户 ID。 |
| `title` | string | 消息标题。 |
| `content` | string | 消息内容。 |
| `quoteContent` | string | 引用内容。 |
| `type` | number | 消息类型。 |
| `detailUrl` | string | 点击跳转链接。 |
| `extraData` | string | 扩展 JSON 字符串。 |
| `status` | number | 状态：`0` 未读，`1` 已读。 |
| `createTime` | number | 创建时间。 |

---

### 5.16 搜索结果
- `GET /api/search/topic`
- 鉴权：否
- Query 参数：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `keyword` | string | 否 | 搜索关键词。 |
| `nodeId` | number | 否 | 节点过滤。 |
| `timeRange` | number | 否 | 时间范围：`0` 全部，`1` 日，`2` 周，`3` 月，`4` 年。 |
| `cursor` | number | 否 | 页码型游标，默认 `1`。 |

响应 `data` 为 `CursorResult`，`results[]` 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | number | 话题数字 ID。 |
| `user` | UserInfo | 作者信息。 |
| `node` | NodeResponse\|null | 节点信息。 |
| `tags` | TagResponse[]\|null | 标签。 |
| `title` | string | 标题（含高亮片段）。 |
| `summary` | string | 摘要（含高亮片段）。 |
| `createTime` | number | 创建时间。 |

---

### 5.17 滑块验证码（推荐登录/注册使用）
- `GET /api/captcha/request_angle`
- 鉴权：否
- 请求参数：无

响应 `data` 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | string | 验证码 ID（后续作为 `captchaId` 提交）。 |
| `imageBase64` | string | 背景图 base64。 |
| `thumbBase64` | string | 滑块图 base64。 |
| `thumbSize` | number | 滑块尺寸（像素）。 |

说明：
- 前端完成滑块后，提交给登录/注册接口的 `captchaCode` 一般为角度值字符串。
- 对应提交时建议 `captchaProtocol=2`。

---

### 5.18 图片验证码（兼容）
- `GET /api/captcha/request`
- 鉴权：否
- 请求参数：无

响应 `data` 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `captchaId` | string | 验证码 ID。 |
| `captchaBase64` | string | 验证码图片 base64。 |

---

### 5.19 图片验证码校验（兼容）
- `GET /api/captcha/verify`
- 鉴权：否
- Query 参数：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `captchaId` | string | 是 | 验证码 ID。 |
| `captchaCode` | string | 是 | 用户输入验证码。 |

响应 `data` 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `success` | boolean | 是否校验成功。 |

---

### 5.20 用户注册
- `POST /api/login/signup`
- 鉴权：否
- Content-Type：`application/x-www-form-urlencoded`（项目现状）

请求参数：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `captchaId` | string | 是 | 验证码 ID。 |
| `captchaCode` | string | 是 | 验证码结果。 |
| `captchaProtocol` | number | 否 | 验证码协议版本。`2` 表示滑块验证码；其他值走旧图形验证码。 |
| `email` | string | 是 | 邮箱。 |
| `username` | string | 是 | 用户名（5-12 位，字母开头，允许数字/`_`/`-`）。 |
| `password` | string | 是 | 密码。 |
| `rePassword` | string | 是 | 确认密码。 |
| `nickname` | string | 是 | 昵称。 |
| `redirect` | string | 否 | 登录成功后的跳转地址。 |

成功响应 `data`（与登录成功结构一致）：

| 字段 | 类型 | 说明 |
|---|---|---|
| `token` | string | 登录态 token。 |
| `user` | UserProfile | 用户信息（含 `roles`、`email`、`emailVerified`、`passwordSet` 等字段）。 |
| `redirect` | string | 回传跳转地址。 |

副作用：
- 服务端会同时写入 Cookie：`bbsgo_token=<token>`。

---

### 5.21 用户名/邮箱 + 密码登录
- `POST /api/login/signin`
- 鉴权：否
- Content-Type：`application/x-www-form-urlencoded`

请求参数：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `captchaId` | string | 是 | 验证码 ID。 |
| `captchaCode` | string | 是 | 验证码结果。 |
| `captchaProtocol` | number | 否 | `2` 使用滑块验证码。 |
| `username` | string | 是 | 用户名或邮箱。 |
| `password` | string | 是 | 密码。 |
| `redirect` | string | 否 | 登录成功跳转地址。 |

成功响应 `data` 同 5.20。

---

### 5.22 请求短信验证码（短信登录）
- `POST /api/login/login_sms_code`
- 鉴权：否
- Content-Type：`application/x-www-form-urlencoded`

请求参数：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `phone` | string | 是 | 手机号。 |
| `captchaId` | string | 是 | 滑块验证码 ID。 |
| `captchaCode` | string | 是 | 滑块验证码结果。 |

响应 `data` 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `smsId` | string | 短信会话 ID（后续短信登录提交）。 |

---

### 5.23 短信登录
- `POST /api/login/login_sms`
- 鉴权：否
- Content-Type：`application/x-www-form-urlencoded`

请求参数：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `smsId` | string | 是 | 短信会话 ID。 |
| `smsCode` | string | 是 | 短信验证码。 |
| `redirect` | string | 否 | 登录成功跳转地址。 |

成功响应 `data` 同 5.20。

---

### 5.24 退出登录
- `GET /api/login/signout`
- 鉴权：是（无有效 token 时也可安全调用）
- 请求参数：无
- 成功返回：`data=null`

副作用：
- 当前 token 会在服务端失效，并清除 `bbsgo_token` Cookie。

---

### 5.25 发送找回密码邮件
- `POST /api/login/send_reset_password_email`
- 鉴权：否
- Content-Type：`application/x-www-form-urlencoded`

请求参数：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `captchaId` | string | 是 | 验证码 ID。 |
| `captchaCode` | string | 是 | 验证码结果。 |
| `captchaProtocol` | number | 否 | `2` 使用滑块验证码。 |
| `email` | string | 是 | 邮箱。 |

成功返回：`data=null`

---

### 5.26 重置密码
- `POST /api/login/reset_password`
- 鉴权：否
- Content-Type：`application/x-www-form-urlencoded`

请求参数：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `token` | string | 是 | 邮件中的重置 token。 |
| `password` | string | 是 | 新密码。 |
| `rePassword` | string | 是 | 确认新密码。 |

成功返回：`data=null`

---

### 5.27 微信登录配置
- `GET /api/login/wx_login_config`
- 鉴权：否
- Query 参数：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `redirect` | string | 否 | 登录成功后跳转地址。 |
| `bind` | boolean | 否 | 是否绑定流程。 |

响应 `data` 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `appid` | string | 微信应用 ID。 |
| `scope` | string | 授权范围。 |
| `redirect_uri` | string | 微信回调地址。 |
| `state` | string | 防 CSRF 状态值（30 分钟内有效）。 |

---

### 5.28 微信登录回调提交
- `POST /api/login/wx_login_submit`
- 鉴权：否
- Content-Type：`application/x-www-form-urlencoded`

请求参数：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `code` | string | 是 | 微信授权码。 |
| `state` | string | 是 | 与 5.27 返回一致。 |

成功响应 `data` 同 5.20。

---

### 5.29 Google 登录配置
- `GET /api/login/google_login_config`
- 鉴权：否
- Query 参数：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `redirect` | string | 否 | 登录成功跳转地址。 |
| `bind` | boolean | 否 | 是否绑定流程。 |

响应 `data` 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `clientId` | string | Google Client ID。 |
| `authUrl` | string | Google 授权地址（前端直接跳转）。 |
| `redirectUri` | string | 回调地址。 |
| `state` | string | 防 CSRF 状态值。 |
| `redirect` | string | 回传的跳转地址。 |

---

### 5.30 Google 登录回调提交
- `POST /api/login/google_login_submit`
- 鉴权：否
- Content-Type：`application/x-www-form-urlencoded`

请求参数：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `code` | string | 是 | Google 授权码。 |
| `state` | string | 是 | 与 5.29 返回一致。 |

成功响应 `data` 同 5.20。

---

### 5.31 Google One Tap 登录
- `POST /api/login/google_one_tap`
- 鉴权：否
- Content-Type：`application/x-www-form-urlencoded`

请求参数：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `credential` | string | 是 | Google One Tap 返回的 JWT 凭证。 |

成功响应 `data` 同 5.20。

## 6. 建议新增接口（可选，保持现有风格）

如果首页需要更接近截图的“热点榜 + 焦点大图”能力，建议在不破坏现有接口的前提下补充：

### 6.1 热点榜（建议）
- `GET /api/topic/hot_topics`
- Query 参数：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `limit` | number | 否 | 返回条数，默认 `10`。 |
| `timeRange` | number | 否 | 统计时间范围：`1`日 `2`周 `3`月。 |

响应 `data`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | string | 话题 ID（编码字符串）。 |
| `title` | string | 标题。 |
| `heatScore` | number | 热度分（后端计算）。 |
| `viewCount` | number | 浏览数。 |
| `commentCount` | number | 评论数。 |
| `likeCount` | number | 点赞数。 |
| `voteCount` | number | 投票参与数。 |
| `rank` | number | 排名（1 开始）。 |

### 6.2 首页焦点卡片（建议）
- `GET /api/topic/focus_topics`
- Query 参数：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `limit` | number | 否 | 返回条数，默认 `3`。 |

响应 `data` 为数组，元素字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `topic` | TopicResponse | 话题卡片主数据。 |
| `cover` | string | 焦点图 URL。 |
| `subTitle` | string | 副标题。 |
| `badge` | string | 左上角角标文案（如“热门”“精选”）。 |
| `heatScore` | number | 热度分。 |

## 7. 错误码建议（沿用项目）

| errorCode | 说明 |
|---|---|
| `0` | 通用成功或普通错误（通过 `success` 与 `message` 判断）。 |
| `1` | 未登录。 |
| `2` | 无权限。 |
| `1000` | 验证码错误。 |
| `1001` | 禁止操作。 |
| `1002` | 用户被禁用。 |
| `1004` | 邮箱未验证。 |
