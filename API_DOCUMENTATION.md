# BBS-Go 项目接口文档

本文档按业务领域整理 BBS-Go 项目的所有 RESTful API 接口。

## 目录

- [用户认证域](#用户认证域)
- [用户中心域](#用户中心域)
- [内容域 - 文章](#内容域 - 文章)
- [内容域 - 话题/帖子](#内容域 - 话题帖子)
- [内容域 - 评论](#内容域 - 评论)
- [内容域 - 标签](#内容域 - 标签)
- [互动域](#互动域)
- [系统配置域](#系统配置域)
- [文件上传域](#文件上传域)
- [搜索域](#搜索域)
- [任务域](#任务域)
- [徽章域](#徽章域)
- [投票域](#投票域)
- [管理后台 - 用户管理](#管理后台---用户管理)
- [管理后台 - 内容管理](#管理后台---内容管理)
- [管理后台 - 系统配置](#管理后台---系统配置)
- [管理后台 - 日志管理](#管理后台---日志管理)
- [管理后台 - 权限管理](#管理后台---权限管理)
- [管理后台 - 其他管理功能](#管理后台---其他管理功能)

---

## 用户认证域

**基础路径**: `/api/login`

### 1. 用户注册
- **接口**: `POST /api/login/signup`
- **描述**: 新用户注册账号
- **请求参数**:
  ```json
  {
    "captchaId": "string",
    "captchaCode": "string",
    "captchaProtocol": 2,
    "email": "string",
    "username": "string",
    "password": "string",
    "rePassword": "string",
    "nickname": "string",
    "redirect": "string"
  }
  ```
- **返回**: 登录成功后的用户信息和 Token

### 2. 用户名密码登录
- **接口**: `POST /api/login/signin`
- **描述**: 使用用户名和密码登录
- **请求参数**:
  ```json
  {
    "captchaId": "string",
    "captchaCode": "string",
    "captchaProtocol": 2,
    "username": "string",
    "password": "string",
    "redirect": "string"
  }
  ```
- **返回**: 登录成功后的用户信息和 Token

### 3. 发送重置密码邮件
- **接口**: `POST /api/login/send_reset_password_email`
- **描述**: 请求发送找回密码邮件
- **请求参数**:
  ```json
  {
    "captchaId": "string",
    "captchaCode": "string",
    "captchaProtocol": 2,
    "email": "string"
  }
  ```

### 4. 重置密码
- **接口**: `POST /api/login/reset_password`
- **描述**: 通过 token 重置密码
- **请求参数**:
  ```json
  {
    "token": "string",
    "password": "string",
    "rePassword": "string"
  }
  ```

### 5. 退出登录
- **接口**: `GET /api/login/signout`
- **描述**: 退出当前登录

---

## 用户中心域

**基础路径**: `/api/user`

### 1. 获取当前登录用户
- **接口**: `GET /api/user/current`
- **描述**: 获取当前登录用户的详细信息

### 2. 获取用户详情
- **接口**: `GET /api/user/{userId}`
- **描述**: 获取指定用户的详细信息
- **路径参数**: 
  - `userId`: 用户 ID（加密编码）

### 3. 修改用户资料
- **接口**: `POST /api/user/edit/{userId}`
- **描述**: 修改用户基本信息
- **请求参数**:
  ```json
  {
    "nickname": "string",
    "homePage": "string",
    "description": "string",
    "gender": "string"
  }
  ```

### 4. 修改头像
- **接口**: `POST /api/user/updateAvatar`
- **描述**: 更新用户头像
- **请求参数**:
  ```json
  {
    "avatar": "string"
  }
  ```

### 5. 修改昵称
- **接口**: `POST /api/user/updateNickname`
- **描述**: 单独修改用户昵称

### 6. 修改个人描述
- **接口**: `POST /api/user/updateDescription`
- **描述**: 单独修改用户个人描述

### 7. 修改性别
- **接口**: `POST /api/user/updateGender`
- **描述**: 单独修改用户性别

### 8. 获取用户文章列表
- **接口**: `GET /api/user/articles`
- **描述**: 获取指定用户的文章列表
- **查询参数**:
  - `userId`: 用户 ID
  - `cursor`: 游标值

### 9. 检查签到状态
- **接口**: `GET /api/user/checkStatus`
- **描述**: 检查用户签到状态

### 10. 获取用户粉丝列表
- **接口**: `GET /api/user/followers`
- **描述**: 获取用户的粉丝列表

### 11. 获取用户关注列表
- **接口**: `GET /api/user/followings`
- **描述**: 获取用户关注的人列表

### 12. 关注用户
- **接口**: `POST /api/user/follow/{userId}`
- **描述**: 关注指定用户

### 13. 取消关注用户
- **接口**: `POST /api/user/unfollow/{userId}`
- **描述**: 取消关注指定用户

### 14. 举报用户
- **接口**: `POST /api/user/report`
- **描述**: 举报用户
- **请求参数**:
  ```json
  {
    "userId": "number",
    "reason": "string"
  }
  ```

---

## 内容域 - 文章

**基础路径**: `/api/article`

### 1. 获取文章详情
- **接口**: `GET /api/article/{articleId}`
- **描述**: 获取文章详细信息（会增加浏览量）
- **路径参数**: `articleId` - 文章 ID

### 2. 发表文章
- **接口**: `POST /api/article/create`
- **描述**: 发表新文章
- **请求参数**:
  ```json
  {
    "title": "string",
    "content": "string",
    "tags": ["string"],
    "cover": {
      "url": "string",
      "width": "number",
      "height": "number"
    }
  }
  ```

### 3. 获取编辑文章详情
- **接口**: `GET /api/article/edit/{articleId}`
- **描述**: 获取文章编辑时的详细信息
- **返回**: 包含文章标题、内容、标签、封面等信息

### 4. 编辑文章
- **接口**: `POST /api/article/edit/{articleId}`
- **描述**: 编辑已有文章
- **请求参数**:
  ```json
  {
    "title": "string",
    "content": "string",
    "tags": ["string"],
    "cover": {}
  }
  ```

### 5. 删除文章
- **接口**: `POST /api/article/delete/{articleId}`
- **描述**: 删除文章

### 6. 收藏文章
- **接口**: `POST /api/article/favorite/{articleId}`
- **描述**: 收藏指定文章

### 7. 获取文章跳转链接
- **接口**: `GET /api/article/redirect/{articleId}`
- **描述**: 获取文章的跳转链接

### 8. 获取用户文章列表
- **接口**: `GET /api/article/user_articles`
- **描述**: 获取指定用户的文章列表
- **查询参数**:
  - `userId`: 用户 ID
  - `cursor`: 分页游标

---

## 内容域 - 话题/帖子

**基础路径**: `/api/topic`

### 1. 获取节点导航
- **接口**: `GET /api/topic/node_navs`
- **描述**: 获取所有节点导航信息（包括内置节点）

### 2. 获取所有节点
- **接口**: `GET /api/topic/nodes`
- **描述**: 获取所有话题节点列表

### 3. 获取节点信息
- **接口**: `GET /api/topic/node`
- **描述**: 获取指定节点信息
- **查询参数**: `nodeId` - 节点 ID

### 4. 发表话题
- **接口**: `POST /api/topic/create`
- **描述**: 发表新话题/帖子
- **请求参数**:
  ```json
  {
    "type": "number",
    "nodeId": "number",
    "title": "string",
    "content": "string",
    "contentType": "string",
    "hideContent": "string",
    "tags": ["string"]
  }
  ```

### 5. 获取编辑话题详情
- **接口**: `GET /api/topic/edit/{topicId}`
- **描述**: 获取话题编辑时的详细信息
- **路径参数**: `topicId` - 话题 ID（加密编码）

### 6. 编辑话题
- **接口**: `POST /api/topic/edit/{topicId}`
- **描述**: 编辑已有话题
- **请求参数**:
  ```json
  {
    "nodeId": "number",
    "title": "string",
    "content": "string",
    "contentType": "string",
    "hideContent": "string",
    "tags": ["string"]
  }
  ```

### 7. 删除话题
- **接口**: `POST /api/topic/delete/{topicId}`
- **描述**: 删除话题

### 8. 获取话题详情
- **接口**: `GET /api/topic/{topicId}`
- **描述**: 获取话题详细信息

### 9. 获取话题列表
- **接口**: `GET /api/topic/list`
- **描述**: 获取话题列表（支持按节点筛选）
- **查询参数**:
  - `nodeId`: 节点 ID（可选，负数表示内置节点）
  - `cursor`: 分页游标

### 10. 获取话题收藏列表
- **接口**: `GET /api/topic/favorites`
- **描述**: 获取用户收藏的话题列表

### 11. 收藏话题
- **接口**: `POST /api/topic/favorite/{topicId}`
- **描述**: 收藏指定话题

### 12. 取消收藏话题
- **接口**: `POST /api/topic/unfavorite/{topicId}`
- **描述**: 取消收藏话题

### 13. 获取话题标签
- **接口**: `GET /api/topic/tags/{topicId}`
- **描述**: 获取话题的标签列表

### 14. 设置话题标签
- **接口**: `POST /api/topic/set_tags/{topicId}`
- **描述**: 设置话题标签

### 15. 隐藏话题
- **接口**: `POST /api/topic/hide/{topicId}`
- **描述**: 隐藏话题（仅作者或管理员）

### 16. 显示话题
- **接口**: `POST /api/topic/show/{topicId}`
- **描述**: 显示被隐藏的话题

### 17. 推荐话题
- **接口**: `POST /api/topic/recommend/{topicId}`
- **描述**: 推荐话题（仅管理员）

### 18. 取消推荐话题
- **接口**: `POST /api/topic/unrecommend/{topicId}`
- **描述**: 取消话题推荐

### 19. 删除话题（物理删除）
- **接口**: `POST /api/topic/permanent_delete/{topicId}`
- **描述**: 永久删除话题（仅管理员）

### 20. 获取话题跳转链接
- **接口**: `GET /api/topic/redirect/{topicId}`
- **描述**: 获取话题跳转链接

---

## 内容域 - 评论

**基础路径**: `/api/comment`

### 1. 发表评论
- **接口**: `POST /api/comment/create`
- **描述**: 发表评论
- **请求参数**:
  ```json
  {
    "type": "string",
    "objectId": "number",
    "content": "string",
    "quoteId": "number"
  }
  ```

### 2. 删除评论
- **接口**: `POST /api/comment/delete/{commentId}`
- **描述**: 删除评论

### 3. 获取评论列表
- **接口**: `GET /api/comment/list`
- **描述**: 获取指定对象的评论列表
- **查询参数**:
  - `type`: 评论类型（article/topic）
  - `objectId`: 对象 ID
  - `cursor`: 分页游标

### 4. 获取评论详情
- **接口**: `GET /api/comment/{commentId}`
- **描述**: 获取单条评论详情

### 5. 获取子评论
- **接口**: `GET /api/comment/replies/{commentId}`
- **描述**: 获取评论的回复列表

---

## 内容域 - 标签

**基础路径**: `/api/tag`

### 1. 获取标签列表
- **接口**: `GET /api/tag/list`
- **描述**: 获取所有标签列表
- **查询参数**:
  - `cursor`: 分页游标

### 2. 获取热门标签
- **接口**: `GET /api/tag/hot_list`
- **描述**: 获取热门标签列表

### 3. 获取标签详情
- **接口**: `GET /api/tag/{tagId}`
- **描述**: 获取标签详细信息

### 4. 关注标签
- **接口**: `POST /api/tag/follow/{tagId}`
- **描述**: 关注指定标签

### 5. 取消关注标签
- **接口**: `POST /api/tag/unfollow/{tagId}`
- **描述**: 取消关注标签

### 6. 获取用户关注的标签
- **接口**: `GET /api/tag/followed`
- **描述**: 获取当前用户关注的标签列表

---

## 互动域

### 点赞相关
**基础路径**: `/api/like`

#### 1. 点赞
- **接口**: `POST /api/like/create`
- **描述**: 点赞操作
- **请求参数**:
  ```json
  {
    "type": "string",
    "objectId": "number"
  }
  ```

#### 2. 取消点赞
- **接口**: `POST /api/like/cancel`
- **描述**: 取消点赞
- **请求参数**:
  ```json
  {
    "type": "string",
    "objectId": "number"
  }
  ```

#### 3. 获取点赞状态
- **接口**: `GET /api/like/status`
- **描述**: 获取用户对某对象的点赞状态
- **查询参数**:
  - `type`: 类型
  - `objectId`: 对象 ID

### 收藏相关
**基础路径**: `/api/favorite`

#### 1. 获取用户收藏列表
- **接口**: `GET /api/favorite/list`
- **描述**: 获取用户的收藏列表
- **查询参数**:
  - `cursor`: 分页游标
  - `type`: 收藏类型（可选）

### 签到相关
**基础路径**: `/api/checkin`

#### 1. 签到
- **接口**: `POST /api/checkin/create`
- **描述**: 每日签到

#### 2. 获取签到状态
- **接口**: `GET /api/checkin/status`
- **描述**: 获取用户今日签到状态

#### 3. 获取签到排名
- **接口**: `GET /api/checkin/ranking`
- **描述**: 获取签到排行榜

---

## 系统配置域

**基础路径**: `/api/config`

### 1. 获取系统配置
- **接口**: `GET /api/config/configs`
- **描述**: 获取系统公开配置信息
- **返回内容**:
  - 站点基本信息（标题、描述、Logo 等）
  - 登录配置（密码登录、微信登录、短信登录等开关）
  - 功能开关（文章待审、话题验证码、邮箱验证等）
  - 模块配置
  - 脚本注入配置

---

## 文件上传域

**基础路径**: `/api/upload`

### 1. 上传图片
- **接口**: `POST /api/upload/image`
- **描述**: 上传图片文件
- **请求类型**: `multipart/form-data`
- **返回**: 图片 URL 和尺寸信息

### 2. 上传文件
- **接口**: `POST /api/upload/file`
- **描述**: 上传通用文件

---

## 搜索域

**基础路径**: `/api/search`

### 1. 搜索
- **接口**: `GET /api/search/search`
- **描述**: 全局搜索
- **查询参数**:
  - `keyword`: 搜索关键词
  - `type`: 搜索类型（article/topic/user 等）
  - `cursor`: 分页游标

### 2. 搜索建议
- **接口**: `GET /api/search/suggest`
- **描述**: 获取搜索建议
- **查询参数**: `keyword` - 搜索关键词

---

## 任务域

**基础路径**: `/api/task`

### 1. 获取任务列表
- **接口**: `GET /api/task/list`
- **描述**: 获取用户可执行的任务列表

### 2. 执行任务
- **接口**: `POST /api/task/execute`
- **描述**: 执行指定任务
- **请求参数**:
  ```json
  {
    "taskId": "number"
  }
  ```

### 3. 获取任务进度
- **接口**: `GET /api/task/progress`
- **描述**: 获取任务执行进度

---

## 徽章域

**基础路径**: `/api/badge`

### 1. 获取徽章列表
- **接口**: `GET /api/badge/list`
- **描述**: 获取所有可用徽章列表

### 2. 获取用户徽章
- **接口**: `GET /api/badge/user_badges`
- **描述**: 获取用户拥有的徽章

### 3. 佩戴徽章
- **接口**: `POST /api/badge/wear`
- **描述**: 佩戴指定徽章
- **请求参数**:
  ```json
  {
    "badgeId": "number"
  }
  ```

---

## 投票域

**基础路径**: `/api/vote`

### 1. 获取投票详情
- **接口**: `GET /api/vote/{voteId}`
- **描述**: 获取投票活动详情

### 2. 参与投票
- **接口**: `POST /api/vote/vote`
- **描述**: 参与投票
- **请求参数**:
  ```json
  {
    "voteId": "number",
    "optionIds": ["number"]
  }
  ```

### 3. 查看投票结果
- **接口**: `GET /api/vote/result/{voteId}`
- **描述**: 查看投票结果统计

---

## 其他接口

### 验证码
**基础路径**: `/api/captcha`

#### 1. 获取验证码
- **接口**: `GET /api/captcha/image`
- **描述**: 获取图形验证码
- **查询参数**: `id` - 验证码 ID

### 友情链接
**基础路径**: `/api/link`

#### 1. 获取友情链接
- **接口**: `GET /api/link/list`
- **描述**: 获取友情链接列表

### 安装接口
**基础路径**: `/api/install`

#### 1. 检查安装状态
- **接口**: `GET /api/install/check`
- **描述**: 检查系统是否已安装

#### 2. 执行安装
- **接口**: `POST /api/install`
- **描述**: 执行系统安装向导

---

## 管理后台接口

**所有管理后台接口基础路径**: `/api/admin`

**需要权限**: 所有 `/api/admin/*` 的接口都需要管理员权限，通过 `AdminMiddleware` 进行权限验证。

### 管理后台 - 用户管理

**基础路径**: `/api/admin/user`

#### 1. 获取用户列表
- **接口**: `ANY /api/admin/user/list`
- **描述**: 分页获取用户列表（支持多条件筛选）
- **筛选条件**:
  - `id`: 用户 ID
  - `nickname`: 昵称（模糊匹配）
  - `email`: 邮箱
  - `username`: 用户名
  - `type`: 用户类型

#### 2. 获取用户详情
- **接口**: `GET /api/admin/user/{id}`
- **描述**: 获取指定用户详细信息

#### 3. 创建用户
- **接口**: `POST /api/admin/user/create`
- **描述**: 创建新用户
- **请求参数**:
  ```json
  {
    "username": "string",
    "email": "string",
    "nickname": "string",
    "password": "string"
  }
  ```

#### 4. 更新用户
- **接口**: `POST /api/admin/user/update`
- **描述**: 更新用户信息
- **请求参数**:
  ```json
  {
    "id": "number",
    "username": "string",
    "email": "string",
    "nickname": "string",
    "password": "string",
    "status": "number"
  }
  ```

#### 5. 删除用户
- **接口**: `POST /api/admin/user/delete/{id}`
- **描述**: 删除用户

#### 6. 同步用户数据
- **接口**: `GET /api/admin/user/synccount`
- **描述**: 异步同步用户统计数据（话题数、评论数等）

### 管理后台 - 角色管理

**基础路径**: `/api/admin/role`

#### 1. 获取角色列表
- **接口**: `ANY /api/admin/role/list`
- **描述**: 获取所有角色列表

#### 2. 获取角色详情
- **接口**: `GET /api/admin/role/{id}`
- **描述**: 获取角色详细信息

#### 3. 创建角色
- **接口**: `POST /api/admin/role/create`
- **描述**: 创建新角色

#### 4. 更新角色
- **接口**: `POST /api/admin/role/update`
- **描述**: 更新角色信息

#### 5. 删除角色
- **接口**: `POST /api/admin/role/delete/{id}`
- **描述**: 删除角色

### 管理后台 - 菜单管理

**基础路径**: `/api/admin/menu`

#### 1. 获取菜单列表
- **接口**: `ANY /api/admin/menu/list`
- **描述**: 获取所有菜单项

#### 2. 获取菜单详情
- **接口**: `GET /api/admin/menu/{id}`
- **描述**: 获取菜单详情

#### 3. 创建菜单
- **接口**: `POST /api/admin/menu/create`
- **描述**: 创建菜单项

#### 4. 更新菜单
- **接口**: `POST /api/admin/menu/update`
- **描述**: 更新菜单项

#### 5. 删除菜单
- **接口**: `POST /api/admin/menu/delete/{id}`
- **描述**: 删除菜单项

#### 6. 获取菜单 API 映射
- **接口**: `GET /api/admin/menu/api`
- **描述**: 获取菜单与 API 的映射关系

### 管理后台 - 文章管理

**基础路径**: `/api/admin/article`

#### 1. 获取文章列表
- **接口**: `ANY /api/admin/article/list`
- **描述**: 分页获取文章列表（支持筛选）

#### 2. 获取文章详情
- **接口**: `GET /api/admin/article/{id}`
- **描述**: 获取文章详情

#### 3. 审核文章
- **接口**: `POST /api/admin/article/audit/{id}`
- **描述**: 审核文章（通过/拒绝）

#### 4. 推荐文章
- **接口**: `POST /api/admin/article/recommend/{id}`
- **描述**: 推荐文章

#### 5. 删除文章
- **接口**: `POST /api/admin/article/delete/{id}`
- **描述**: 删除文章

### 管理后台 - 话题管理

**基础路径**: `/api/admin/topic`

#### 1. 获取话题列表
- **接口**: `ANY /api/admin/topic/list`
- **描述**: 分页获取话题列表

#### 2. 获取话题详情
- **接口**: `GET /api/admin/topic/{id}`
- **描述**: 获取话题详情

#### 3. 审核话题
- **接口**: `POST /api/admin/topic/audit/{id}`
- **描述**: 审核话题

#### 4. 推荐话题
- **接口**: `POST /api/admin/topic/recommend/{id}`
- **描述**: 推荐话题

#### 5. 删除话题
- **接口**: `POST /api/admin/topic/delete/{id}`
- **描述**: 删除话题

### 管理后台 - 话题节点管理

**基础路径**: `/api/admin/topic-node`

#### 1. 获取节点列表
- **接口**: `ANY /api/admin/topic-node/list`
- **描述**: 获取所有话题节点

#### 2. 创建节点
- **接口**: `POST /api/admin/topic-node/create`
- **描述**: 创建新节点

#### 3. 更新节点
- **接口**: `POST /api/admin/topic-node/update`
- **描述**: 更新节点信息

#### 4. 删除节点
- **接口**: `POST /api/admin/topic-node/delete/{id}`
- **描述**: 删除节点

### 管理后台 - 标签管理

**基础路径**: `/api/admin/tag`

#### 1. 获取标签列表
- **接口**: `ANY /api/admin/tag/list`
- **描述**: 获取标签列表

#### 2. 创建标签
- **接口**: `POST /api/admin/tag/create`
- **描述**: 创建标签

#### 3. 更新标签
- **接口**: `POST /api/admin/tag/update`
- **描述**: 更新标签

#### 4. 删除标签
- **接口**: `POST /api/admin/tag/delete/{id}`
- **描述**: 删除标签

### 管理后台 - 文章标签管理

**基础路径**: `/api/admin/article-tag`

#### 1. 获取文章标签列表
- **接口**: `ANY /api/admin/article-tag/list`
- **描述**: 获取文章标签关联列表

### 管理后台 - 评论管理

**基础路径**: `/api/admin/comment`

#### 1. 获取评论列表
- **接口**: `ANY /api/admin/comment/list`
- **描述**: 分页获取评论列表

#### 2. 删除评论
- **接口**: `POST /api/admin/comment/delete/{id}`
- **描述**: 删除评论

### 管理后台 - 收藏管理

**基础路径**: `/api/admin/favorite`

#### 1. 获取收藏列表
- **接口**: `ANY /api/admin/favorite/list`
- **描述**: 获取收藏记录列表

### 管理后台 - 系统配置管理

**基础路径**: `/api/admin/sys-config`

#### 1. 获取系统配置
- **接口**: `GET /api/admin/sys-config/get`
- **描述**: 获取系统配置项

#### 2. 更新系统配置
- **接口**: `POST /api/admin/sys-config/update`
- **描述**: 更新系统配置
- **请求参数**:
  ```json
  {
    "key": "string",
    "value": "string"
  }
  ```

#### 3. 批量更新配置
- **接口**: `POST /api/admin/sys-config/batch_update`
- **描述**: 批量更新多个配置项

### 管理后台 - 字典管理

**基础路径**: `/api/admin/dict`

#### 1. 获取字典列表
- **接口**: `ANY /api/admin/dict/list`
- **描述**: 获取字典项列表

#### 2. 创建字典
- **接口**: `POST /api/admin/dict/create`
- **描述**: 创建字典项

#### 3. 更新字典
- **接口**: `POST /api/admin/dict/update`
- **描述**: 更新字典项

#### 4. 删除字典
- **接口**: `POST /api/admin/dict/delete/{id}`
- **描述**: 删除字典项

### 管理后台 - 字典类型管理

**基础路径**: `/api/admin/dict-type`

#### 1. 获取字典类型列表
- **接口**: `ANY /api/admin/dict-type/list`
- **描述**: 获取字典类型列表

#### 2. 创建字典类型
- **接口**: `POST /api/admin/dict-type/create`
- **描述**: 创建字典类型

#### 3. 更新字典类型
- **接口**: `POST /api/admin/dict-type/update`
- **描述**: 更新字典类型

#### 4. 删除字典类型
- **接口**: `POST /api/admin/dict-type/delete/{id}`
- **描述**: 删除字典类型

### 管理后台 - 链接管理

**基础路径**: `/api/admin/link`

#### 1. 获取链接列表
- **接口**: `ANY /api/admin/link/list`
- **描述**: 获取友情链接列表

#### 2. 创建链接
- **接口**: `POST /api/admin/link/create`
- **描述**: 创建友情链接

#### 3. 更新链接
- **接口**: `POST /api/admin/link/update`
- **描述**: 更新链接

#### 4. 删除链接
- **接口**: `POST /api/admin/link/delete/{id}`
- **描述**: 删除链接

### 管理后台 - 任务配置管理

**基础路径**: `/api/admin/task-config`

#### 1. 获取任务配置列表
- **接口**: `ANY /api/admin/task-config/list`
- **描述**: 获取任务配置列表

#### 2. 创建任务配置
- **接口**: `POST /api/admin/task-config/create`
- **描述**: 创建任务配置

#### 3. 更新任务配置
- **接口**: `POST /api/admin/task-config/update`
- **描述**: 更新任务配置

#### 4. 删除任务配置
- **接口**: `POST /api/admin/task-config/delete/{id}`
- **描述**: 删除任务配置

### 管理后台 - 徽章管理

**基础路径**: `/api/admin/badge`

#### 1. 获取徽章列表
- **接口**: `ANY /api/admin/badge/list`
- **描述**: 获取所有徽章配置

#### 2. 创建徽章
- **接口**: `POST /api/admin/badge/create`
- **描述**: 创建徽章

#### 3. 更新徽章
- **接口**: `POST /api/admin/badge/update`
- **描述**: 更新徽章配置

#### 4. 删除徽章
- **接口**: `POST /api/admin/badge/delete/{id}`
- **描述**: 删除徽章

### 管理后台 - 等级配置管理

**基础路径**: `/api/admin/level-config`

#### 1. 获取等级配置列表
- **接口**: `ANY /api/admin/level-config/list`
- **描述**: 获取用户等级配置列表

#### 2. 创建等级配置
- **接口**: `POST /api/admin/level-config/create`
- **描述**: 创建等级配置

#### 3. 更新等级配置
- **接口**: `POST /api/admin/level-config/update`
- **描述**: 更新等级配置

#### 4. 删除等级配置
- **接口**: `POST /api/admin/level-config/delete/{id}`
- **描述**: 删除等级配置

### 管理后台 - 禁词管理

**基础路径**: `/api/admin/forbidden-word`

#### 1. 获取禁词列表
- **接口**: `ANY /api/admin/forbidden-word/list`
- **描述**: 获取敏感词列表

#### 2. 添加禁词
- **接口**: `POST /api/admin/forbidden-word/create`
- **描述**: 添加敏感词

#### 3. 更新禁词
- **接口**: `POST /api/admin/forbidden-word/update`
- **描述**: 更新敏感词

#### 4. 删除禁词
- **接口**: `POST /api/admin/forbidden-word/delete/{id}`
- **描述**: 删除敏感词

### 管理后台 - 投票管理

**基础路径**: `/api/admin/vote`

#### 1. 获取投票列表
- **接口**: `ANY /api/admin/vote/list`
- **描述**: 获取投票活动列表

#### 2. 获取投票详情
- **接口**: `GET /api/admin/vote/{id}`
- **描述**: 获取投票详情

#### 3. 删除投票
- **接口**: `POST /api/admin/vote/delete/{id}`
- **描述**: 删除投票活动

### 管理后台 - 投票选项管理

**基础路径**: `/api/admin/vote-option`

#### 1. 获取投票选项列表
- **接口**: `ANY /api/admin/vote-option/list`
- **描述**: 获取投票选项列表

### 管理后台 - 投票记录管理

**基础路径**: `/api/admin/vote-record`

#### 1. 获取投票记录列表
- **接口**: `ANY /api/admin/vote-record/list`
- **描述**: 获取投票记录列表

### 管理后台 - 日志管理

#### 操作日志
**基础路径**: `/api/admin/operate-log`

##### 1. 获取操作日志列表
- **接口**: `ANY /api/admin/operate-log/list`
- **描述**: 分页获取操作日志列表
- **筛选条件**:
  - `userId`: 用户 ID
  - `opType`: 操作类型
  - `entity`: 实体类型
  - `timeRange`: 时间范围

#### 用户积分日志
**基础路径**: `/api/admin/user-score-log`

##### 1. 获取用户积分日志列表
- **接口**: `ANY /api/admin/user-score-log/list`
- **描述**: 获取用户积分变更日志

#### 用户经验日志
**基础路径**: `/api/admin/user-exp-log`

##### 1. 获取用户经验日志列表
- **接口**: `ANY /api/admin/user-exp-log/list`
- **描述**: 获取用户经验值变更日志

#### 用户任务日志
**基础路径**: `/api/admin/user-task-log`

##### 1. 获取用户任务日志列表
- **接口**: `ANY /api/admin/user-task-log/list`
- **描述**: 获取用户任务完成日志

#### 用户任务事件
**基础路径**: `/api/admin/user-task-event`

##### 1. 获取用户任务事件列表
- **接口**: `ANY /api/admin/user-task-event/list`
- **描述**: 获取用户任务事件列表

### 管理后台 - 邮箱管理

#### 邮箱验证码
**基础路径**: `/api/admin/email-code`

##### 1. 获取邮箱验证码列表
- **接口**: `ANY /api/admin/email-code/list`
- **描述**: 获取邮箱验证码记录

#### 邮箱日志
**基础路径**: `/api/admin/email-log`

##### 1. 获取邮箱日志列表
- **接口**: `ANY /api/admin/email-log/list`
- **描述**: 获取邮件发送日志

### 管理后台 - 短信管理

**基础路径**: `/api/admin/sms-code`

#### 1. 获取短信验证码列表
- **接口**: `ANY /api/admin/sms-code/list`
- **描述**: 获取短信验证码记录

### 管理后台 - 用户举报管理

**基础路径**: `/api/admin/user-report`

#### 1. 获取举报列表
- **接口**: `ANY /api/admin/user-report/list`
- **描述**: 获取用户举报列表

#### 2. 处理举报
- **接口**: `POST /api/admin/user-report/handle/{id}`
- **描述**: 处理用户举报

### 管理后台 - 用户徽章管理

**基础路径**: `/api/admin/user-badge`

#### 1. 获取用户徽章列表
- **接口**: `ANY /api/admin/user-badge/list`
- **描述**: 获取用户徽章授予记录

#### 2. 授予徽章
- **接口**: `POST /api/admin/user-badge/grant`
- **描述**: 向用户授予徽章

#### 3. 撤销徽章
- **接口**: `POST /api/admin/user-badge/revoke`
- **描述**: 撤销用户徽章

### 管理后台 - API 路由管理

**基础路径**: `/api/admin/api`

#### 1. 初始化 API 路由
- **接口**: `GET /api/admin/api/init`
- **描述**: 扫描并初始化 API 路由信息

#### 2. 获取 API 列表
- **接口**: `ANY /api/admin/api/list`
- **描述**: 获取所有 API 路由列表
- **筛选条件**:
  - `name`: API 名称（模糊匹配）
  - `path`: 路径（模糊匹配）

#### 3. 获取 API 详情
- **接口**: `GET /api/admin/api/{id}`
- **描述**: 获取 API 路由详情

### 管理后台 - 第三方用户管理

**基础路径**: `/api/admin/third-user`

#### 1. 获取第三方用户列表
- **接口**: `ANY /api/admin/third-user/list`
- **描述**: 获取第三方登录用户记录

### 管理后台 - 消息管理

**基础路径**: `/api/admin/message`

#### 1. 获取消息列表
- **接口**: `ANY /api/admin/message/list`
- **描述**: 获取系统消息列表

### 管理后台 - 签到管理

**基础路径**: `/api/admin/check-in`

#### 1. 获取签到记录列表
- **接口**: `ANY /api/admin/check-in/list`
- **描述**: 获取用户签到记录

### 管理后台 - 用户动态管理

**基础路径**: `/api/admin/user-feed`

#### 1. 获取用户动态列表
- **接口**: `ANY /api/admin/user-feed/list`
- **描述**: 获取用户动态记录

### 管理后台 - 用户关注管理

**基础路径**: `/api/admin/user-follow`

#### 1. 获取用户关注列表
- **接口**: `ANY /api/admin/user-follow/list`
- **描述**: 获取用户关注关系记录

### 管理后台 - 迁移管理

**基础路径**: `/api/admin/migration`

#### 1. 执行数据迁移
- **接口**: `POST /api/admin/migration/execute`
- **描述**: 执行数据库迁移脚本

### 管理后台 - 通用接口

**基础路径**: `/api/admin/common`

#### 1. 获取枚举值
- **接口**: `GET /api/admin/common/enums`
- **描述**: 获取系统枚举值定义

---

## 认证说明

### 前端接口认证
- 大部分 `/api/*` 接口需要通过 `AuthMiddleware` 进行身份验证
- 未登录用户访问需要认证的接口会返回错误
- Token 通过 Cookie 或 Header 传递

### 管理后台接口认证
- 所有 `/api/admin/*` 接口需要双重认证:
  1. `AuthMiddleware`: 验证用户登录状态
  2. `AdminMiddleware`: 验证管理员权限
- 只有拥有管理员角色的用户才能访问管理接口

---

## 公共响应格式

### 成功响应
```json
{
  "code": 0,
  "data": {},
  "message": ""
}
```

### 错误响应
```json
{
  "code": 非 0 错误码,
  "data": null,
  "message": "错误描述"
}
```

### 分页响应
```json
{
  "code": 0,
  "data": {
    "results": [],
    "page": {
      "currentPage": 1,
      "pageSize": 20,
      "total": 100
    }
  },
  "message": ""
}
```

### 游标分页响应
```json
{
  "code": 0,
  "data": [],
  "cursor": "游标值",
  "hasMore": true
}
```

---

## 常见错误码

- `400`: 请求参数错误
- `401`: 未登录或 Token 失效
- `403`: 无权限访问
- `404`: 资源不存在
- `500`: 服务器内部错误

---

## 数据字典

### 用户类型 (UserType)
- `user`: 普通用户
- `owner`: 站长
- `admin`: 管理员

### 用户状态 (Status)
- `0`: 正常
- `1`: 审核中
- `2`: 被封禁
- `-1`: 已删除

### 内容类型 (ContentType)
- `text`: 文本
- `markdown`: Markdown 格式
- `html`: HTML 格式

### 话题类型 (TopicType)
- `topic`: 普通话题
- `tweet`: 推文

### 操作类型 (OpType)
- `create`: 创建
- `update`: 更新
- `delete`: 删除

### 实体类型 (Entity)
- `article`: 文章
- `topic`: 话题
- `comment`: 评论
- `user`: 用户

---

## 版本信息

- **项目名称**: BBS-Go
- **框架**: Iris (Go Web Framework)
- **API 版本**: v1
- **文档更新日期**: 2026-03-04
