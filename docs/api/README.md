# 服务接口文档索引

> 说明：本文档按“业务领域”拆分维护，全部基于代码中已注册的真实接口（Iris MVC 路由）。

## 文档列表

- [运营侧（Admin）接口索引（按页面视角）](./admin.md)
- [运营总览看板（Admin Dashboard）](./admin_dashboard.md)
- [预测市场（Admin Predict）](./admin_predict.md)
- [操作审计日志（OperateLog）](./operate_log.md)
- [帖子管理（Admin Topic）](./admin_topic.md)
- [评论管理（Admin Comment）](./admin_comment.md)
- [节点管理（Admin TopicNode）](./admin_topic_node.md)
- [敏感词管理（Admin ForbiddenWord）](./admin_forbidden_word.md)
- [举报审核（Admin UserReport）](./admin_user_report.md)
- [用户管理（Admin User）](./admin_user.md)
- [用户系统](./user.md)
- [评论系统](./comment.md)
- [帖子系统（Topic）](./topic.md)
- [图片上传（Upload）](./upload.md)
- [预测事件系统（PredictMarket / PredictContext / 标签统计）](./predict.md)
- [预测标签（PredictTag / PredictTagStat）](./predict_tag.md)
- [下注/结算系统（金币与预测下注）](./coin.md)
- [开战广场（Battle Square）](./battle.md)
- [对立PK（PK）](./pk.md)
- [对立PK管理（Admin PK）](./admin_pk.md)

## 约定

- 基础路径：默认以服务端路由注册为准（见 `internal/server/router.go`）。
- 认证：`/api/**` 默认经过 `AuthMiddleware`，通常需要登录（cookie `token`）。
- 返回结构：接口通常返回 `web.JsonResult`（字段以实际实现为准）。
