# 用户系统

本模块接口由 `internal/controllers/api/login_controller.go` 与 `internal/controllers/api/user_controller.go` 提供，并在 `internal/server/router.go` 中注册到：

- `/api/login`
- `/api/user`

说明：挂在 `/api` 下，默认经过 `AuthMiddleware`（需要登录）。其中登录/注册相关接口会在内部自行处理未登录场景。

---

## 认证与会话

- 登录成功后，服务端会通过 `UserTokenService` 写 cookie（常见为 `token`），后续请求携带该 cookie 通过 `AuthMiddleware`。

---

## 登录/注册（/api/login）

### 1) 注册
- **接口**：`POST /api/login/signup`
- **功能**：新用户注册
- **参数（form）**（来自 `LoginController.PostSignup`）：
  - `captchaId`: string
  - `captchaCode`: string
  - `captchaProtocol`: int（常见为 2）
  - `email`: string
  - `username`: string
  - `password`: string
  - `rePassword`: string
  - `nickname`: string
  - `redirect`: string（可选）
- **返回**：`web.JsonResult`（登录成功信息，见 `render.BuildLoginSuccess`）
- **可能错误**：
  - `errs.CaptchaError()` 验证码错误
  - `auth.password_login_disabled`（配置禁用密码登录时）
  - `UserService.SignUp` 的业务错误（如用户名/邮箱重复、密码不一致等）

### 2) 用户名密码登录
- **接口**：`POST /api/login/signin`
- **功能**：用户名/密码登录
- **参数（form）**：
  - `captchaId`: string
  - `captchaCode`: string
  - `captchaProtocol`: int
  - `username`: string
  - `password`: string
  - `redirect`: string（可选）
- **返回**：`web.JsonResult`（登录成功信息，见 `render.BuildLoginSuccess`）
- **可能错误**：
  - `errs.CaptchaError()`
  - `auth.password_login_disabled`
  - `UserService.SignIn` 的业务错误

### 3) 发送重置密码邮件
- **接口**：`POST /api/login/send_reset_password_email`
- **功能**：找回密码，发送邮件
- **参数（form）**：
  - `captchaId`: string
  - `captchaCode`: string
  - `captchaProtocol`: int
  - `email`: string
- **返回**：成功 `web.JsonSuccess()`
- **可能错误**：验证码错误、邮件发送失败等（来自 `UserService.SendResetPasswordEmail`）

### 4) 通过 token 重置密码
- **接口**：`POST /api/login/reset_password`
- **功能**：使用邮件中的 token 重置密码
- **参数（form）**：
  - `token`: string
  - `password`: string
  - `rePassword`: string
- **返回**：成功 `web.JsonSuccess()`

### 5) 退出登录
- **接口**：`GET /api/login/signout`
- **功能**：退出当前会话
- **返回**：成功 `web.JsonSuccess()`

### 6) 请求登录短信验证码
- **接口**：`POST /api/login/login_sms_code`
- **功能**：发送短信验证码
- **参数（form/query）**（代码通过 `params.Get` 读取）：
  - `phone`: string
  - `captchaId`: string
  - `captchaCode`: string
- **返回（data）**：

```json
{
  "smsId": "..."
}
```

### 7) 短信登录
- **接口**：`POST /api/login/login_sms`
- **功能**：短信验证码登录
- **参数（form/query）**：
  - `smsId`: string
  - `smsCode`: string
  - `redirect`: string（可选）
- **返回**：`render.BuildLoginSuccess`

### 8) 获取微信登录配置
- **接口**：`GET /api/login/wx_login_config`
- **参数（query）**：
  - `redirect`: string
  - `bind`: bool
- **返回（data）**（字段以实际实现为准，常见为以下结构）：

```json
{
  "appid": "...",
  "scope": "...",
  "redirect_uri": "...",
  "state": "..."
}
```

### 9) 提交微信登录
- **接口**：`POST /api/login/wx_login_submit`
- **参数（form/query）**：
  - `code`: string
  - `state`: string

### 10) 微信绑定/解绑
- **接口**：
  - `POST /api/login/wx_bind`
  - `POST /api/login/wx_unbind`

### 11) 获取 Google 登录配置 / 提交 Google 登录
- **接口**：
  - `GET /api/login/google_login_config`
  - `POST /api/login/google_login_submit`

### 12) Google 绑定/一键登录/解绑
- **接口**：
  - `POST /api/login/google_bind`
  - `POST /api/login/google_one_tap`
  - `POST /api/login/google_unbind`

> 上述微信/Google 相关更多字段与错误信息以 controller 与对应 service 实现为准（本文档基于方法签名与参数读取方式列出）。

---

## 用户资料与用户资产（/api/user）

### 1) 获取当前登录用户
- **接口**：`GET /api/user/current`
- **功能**：返回当前登录用户 Profile（未登录返回 Success 空）
- **返回**：`render.BuildUserProfile(user)`

### 2) 用户详情
- **接口**：`GET /api/user/{userId}`
- **功能**：查询指定用户详情（userId 为加密/编码后的字符串）
- **返回**：`render.BuildUserDetail(user)`
- **可能错误**：`user.not_found`

### 3) 修改用户资料
- **接口**：`POST /api/user/edit/{userId}`
- **参数（form）**：`nickname`, `homePage`, `description`, `gender`
- **返回**：成功 `web.JsonSuccess()`

### 4) 修改头像
- **接口**：`POST /api/user/updateAvatar`
- **参数（form）**：`avatar`

### 5) 修改昵称/简介/性别/生日
- **接口**：
  - `POST /api/user/updateNickname`
  - `POST /api/user/updateDescription`
  - `POST /api/user/updateGender`
  - `POST /api/user/updateBirthday`

### 6) 设置用户名/邮箱/密码、修改密码
- **接口**：
  - `POST /api/user/set_username`
  - `POST /api/user/set_email`
  - `POST /api/user/set_password`
  - `POST /api/user/update_password`

### 7) 设置背景图
- **接口**：`POST /api/user/set_background_image`
- **参数（form）**：`backgroundImage`

### 8) 用户收藏
- **接口**：`GET /api/user/favorites`
- **参数（query）**：`cursor`（可选）
- **返回**：游标分页 `web.JsonCursorData`

### 9) 用户消息
- **接口**：
  - `GET /api/user/msg_recent`（最近 3 条未读）
  - `GET /api/user/messages`（消息列表，读完标记已读）

### 10) 用户积分
- **接口**：
  - `GET /api/user/score_logs`
  - `GET /api/user/scoreRank`

### 11) 禁言（管理员）
- **接口**：`POST /api/user/forbidden`
- **参数（form）**：`userId`, `days`, `reason`
- **权限**：需要 `owner/admin`

### 12) 邮箱验证
- **接口**：
  - `POST /api/user/send_verify_email`
  - `POST /api/user/verify_email`

### 13) 第三方绑定信息
- **接口**：
  - `GET /api/user/wx_bind_info`
  - `GET /api/user/google_bind_info`
