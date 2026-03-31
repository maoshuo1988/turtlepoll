# 服务接口文档索引

> 说明：本文档按“业务领域”拆分维护，全部基于代码中已注册的真实接口（Iris MVC 路由）。

## 文档列表

- [用户系统](./user.md)
- [评论系统](./comment.md)
- [帖子系统（Topic）](./topic.md)
- [图片上传（Upload）](./upload.md)
- [预测事件系统（PredictMarket / PredictContext / 标签统计）](./predict.md)
- [预测标签（PredictTag / PredictTagStat）](./predict_tag.md)
- [下注/结算系统（金币与预测下注）](./coin.md)
- [开战广场（Battle Square）](./battle.md)

## 约定

- 基础路径：默认以服务端路由注册为准（见 `internal/server/router.go`）。
- 认证：`/api/**` 默认经过 `AuthMiddleware`，通常需要登录（cookie `token`）。
- 返回结构：接口通常返回 `web.JsonResult`（字段以实际实现为准）。
