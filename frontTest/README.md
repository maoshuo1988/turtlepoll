# frontTest

前后端分离联调测试前端（独立静态服务）。

## 启动

```bash
cd frontTest
npm run dev
```

默认地址：`http://127.0.0.1:5178`

## 使用说明

1. 先启动后端（例如 `http://172.27.51.133:8082`）。
2. 打开 `login.html` 或 `register.html`。
3. 页面顶部设置后端 API 基地址。
4. 页面会调用：
   - 交互验证码：`GET /api/captcha/request_angle`（拖拽旋转图块）
   - 图片验证码：`GET /api/captcha/request_image`（base64Captcha，captchaProtocol=3）
   - 登录：`POST /api/login/signin`
   - 注册：`POST /api/login/signup`

## 验证码策略（与后端开关联动）

- 「启用旋转验证码」开启时：默认先直接提交；失败后弹出旋转验证码（protocol=2）兜底重试一次。
- 「启用旋转验证码」关闭时：默认弹出图片验证码（protocol=3）并携带 `captchaId/captchaCode/captchaProtocol=3` 提交。
- 如后端配置 `loginCaptcha.disableAllWhenRotateOff=true`，则后端会跳过登录相关验证码校验，此时前端可关闭图片验证码直接提交。
