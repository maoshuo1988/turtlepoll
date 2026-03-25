# Upload / 图片上传 API

> 说明：本文档基于实际代码实现整理（`internal/controllers/api/upload_controller.go`），路由挂载于 `/api/upload`。
>
> - 认证：是（`AuthMiddleware`）。当前线上/前端通常使用 cookie `bbsgo_token` 维持登录态。
> - 上传字段名：固定为 `image`（见 `Ctx.FormFile("image")`）。
> - 大小限制：`constants.UploadMaxM = 10`（约 10MB）。超过限制会返回 `upload.image_too_large`。

## 上传图片

- 方法：POST
- 路径：`/api/upload`
- 认证：是
- Content-Type：`multipart/form-data`

### Form 参数

- `image` (file, 必填)：要上传的图片文件

### 返回

成功时返回 `url`：

```json
{
  "success": true,
  "message": "",
  "url": "https://example.com/uploads/xxx.png"
}
```

> 注意：返回结构基于 `web.NewEmptyRspBuilder().Put("url", url).JsonResult()` 以及项目通用 `web.JsonResult` 包装；实际字段名以线上返回为准，但 `url` 字段一定存在。

### 错误

- 未登录：通常返回 `NotLogin`
- 图片过大：提示 `upload.image_too_large`
- 表单缺少字段或解析失败：直接返回 `FormFile` 的错误信息（例如未找到 `image`）
- 上传存储失败：返回 UploadService/uploader 的错误

### curl 示例（推荐写法）

> 不要手写 boundary，让 curl 自动生成 multipart。

```bash
curl -X POST "http://localhost:8082/api/upload" \
  -b "bbsgo_token=<YOUR_TOKEN>" \
  -F "image=@/absolute/path/to/image.png;type=image/png"
```
