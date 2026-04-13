# 运营侧（Admin）接口索引

> 目的：从“运营页面/工作台”的角度，把需要的接口做成一张导航表。
>
> 约定：
> - **接口详细文档仍按业务领域维护**（如 battle/coin/predict/topic 等），本文件只做“页面 → 接口”的索引。
> - 本文件中的每个接口条目都尽量 **直接链接到对应领域的 md 文档锚点**。
>

## 总览看板

### 全站概览

- 总用户
  - `GET /api/admin/dashboard/stats`（已实现）
  - 详见：[运营总览看板（Admin Dashboard）](./admin_dashboard.md#11-获取全站基础统计)
- 总评论
  - `GET /api/admin/dashboard/stats`（已实现）
  - 详见：[运营总览看板（Admin Dashboard）](./admin_dashboard.md#11-获取全站基础统计)
- 总帖子数
  - `GET /api/admin/dashboard/stats`（已实现）
  - 详见：[运营总览看板（Admin Dashboard）](./admin_dashboard.md#11-获取全站基础统计)

> 说明：当前仓库已有各领域明细接口，但“总览聚合”通常更适合做一个 admin 聚合接口，避免前端并发打多次 count。

### 预测市场（看板区块）

- 今日新增市场/赌局
  - `GET /api/admin/predict/stats`（已实现）
  - 详见：[预测市场（Admin Predict）](./admin_predict.md#11-获取预测市场统计)
- 进行中市场统计
  - `GET /api/admin/predict/stats`（已实现）
  - 详见：[预测市场（Admin Predict）](./admin_predict.md#11-获取预测市场统计)
- 已结算市场统计
  - `GET /api/admin/predict/stats`（已实现）
  - 详见：[预测市场（Admin Predict）](./admin_predict.md#11-获取预测市场统计)
- 今日下注额/入场费/燃烧
  - `GET /api/admin/predict/stats`（已实现：todayBetAmount/todayFee/todayBurn）
  - 详见：[预测市场（Admin Predict）](./admin_predict.md#11-获取预测市场统计)
- 7 日内下注趋势柱状图
  - `GET /api/admin/predict/trends?range=7d`（已实现）
  - 详见：[预测市场（Admin Predict）](./admin_predict.md#12-71430-日趋势每日新增市场数)
- 7 日内活跃用户柱状图
  - `GET /api/admin/predict/active_users?range=7d`（已实现）
  - 详见：[预测市场（Admin Predict）](./admin_predict.md#13-71430-日活跃用户下注去重用户数)
- 最近操作
  - `GET /api/admin/operate-log/list`（已存在）
  - 详见：[操作审计日志（OperateLog）](./operate_log.md#1-操作日志列表分页)

### 开战广场（看板区块）

- 今日新增市场/赌局
  - `GET /api/battle/list`（已有，按时间筛选能力建议增强）
  - 详见：[开战广场（Battle Square）](./battle.md#1-赌局列表)
- 进行中赌局统计 / 已结算赌局统计 / 待处理争议
  - `GET /api/battle/stats`（已存在）
  - 详见：[开战广场（Battle Square）](./battle.md#2-赌局统计全局)
- 今日下注额/入场费/燃烧
  - `GET /api/battle/stats`（当前只含未结算聚合；若要“今日”口径建议扩展或新增 admin 聚合接口）
- 7 日内下注趋势柱状图 / 7 日内活跃用户柱状图
  - `GET /api/admin/battle/trends?range=7d`（已实现）
  - `GET /api/admin/battle/active_users?range=7d`（已实现）
  - 详见：[开战广场（Battle Square）](./battle.md#管理员看板接口)
- 最近操作
  - `GET /api/admin/operate-log/list`
  - 详见：[操作审计日志（OperateLog）](./operate_log.md#1-操作日志列表分页)


## 预测市场运营

### 市场列表（OPEN/CLOSED/SETTLED）

- `GET /api/football/markets`
  - 详见：[预测事件系统（PredictMarket / PredictContext / 标签统计）](./predict.md#1-查询预测市场聚合返回-market--context)

> 风险提示：该接口当前在 `/api` 下，默认仅登录校验，不是严格 admin 鉴权；建议迁移或在 admin party 下做只读代理。

### 市场编辑（标题/上下文/封面/标签/置顶/推荐）

- `POST /api/football/predict_context/update`
  - 详见：[预测事件系统（PredictMarket / PredictContext / 标签统计）](./predict.md#3-修改创建-predictcontext按-marketid-upsert)

### 结算（管理员对 CLOSED 市场结算，选择正/反方 outcome）

- 结算入口（当前用户侧实现）：`POST /api/coin/settle`
  - 详见：[下注/结算系统（金币与预测下注）](./coin.md#3-结算用户对自己下注过的预测市场进行结算领取金币)

> 运营侧：`POST /api/admin/predict/market/settle`（已实现），仅允许对 `CLOSED` 市场结算（可选 allowReset 用于纠错），并强制 outcome 为“正方/反方”二选一，同时写入 operate-log。

### 结算前盘口统计（正/反方人数与投注金额）

- `GET /api/admin/predict/market/stats?marketId=...`（已实现）
  - 返回建议字段：
    - `proUserCount` / `conUserCount`
    - `proAmount` / `conAmount`
    - `totalAmount`

### 标签运营

- 标签列表：`GET /api/predict-tag/list`
  - 详见：[预测标签（PredictTag / PredictTagStat）](./predict_tag.md)
- 刷新物化：`POST /api/predict-tag/refresh`
  - 详见：[预测标签（PredictTag / PredictTagStat）](./predict_tag.md)


## 开战广场运营

### 赌局巡检列表（open/sealed/pending/disputed/settled）

- `GET /api/battle/list`
  - 详见：[开战广场（Battle Square）](./battle.md#1-赌局列表)

### 争议仲裁队列（disputed）

- `GET /api/battle/list?status=disputed`
  - 详见：[开战广场（Battle Square）](./battle.md#1-赌局列表)

### 仲裁（resolve）

- `POST /api/admin/battle/resolve`


## 社区管理

## 宠物与开蛋池运营

- 开蛋池配置（概率和必须等于 1，否则保存失败）：
  - 详见：[开蛋池（Gacha Pool）配置（Admin）](./pet_gacha.md)

### 帖子管理（删除 / 置顶 / 推荐，均支持取消）

- 帖子列表/详情
  - `GET /api/topic/list`
  - `GET /api/topic/by?id=...`
  - 详见：[帖子系统（Topic）](./topic.md)

- 删除帖子
  - `POST /api/admin/topic/delete`（已存在；form：id）
  - 说明：Admin 端 `id` 使用明文 int64（不使用 encode id）
  - 详见：[帖子管理（Admin Topic）](./admin_topic.md#3-删除帖子)

- 置顶/取消置顶
  - 现状：仓库当前没有单独的 `/api/admin/topic/pin|unpin`；但已有“管理员置顶”能力：
    - `POST /api/topic/sticky/{topicId}`（已存在；form：sticky=true/false；需要 owner/admin）
    - 详见：[帖子系统（Topic）](./topic.md#设置置顶管理员)
  - 备注：若要统一“admin path 风格”，后续可再补一层 `/api/admin/topic/pin|unpin` 代理（本轮先按既有路径对齐文档）。

- 推荐/取消推荐
  - `POST /api/admin/topic/recommend`（已存在；form：id）
  - `DELETE /api/admin/topic/recommend`（已存在；form：id；用于取消推荐）
  - 详见：[帖子管理（Admin Topic）](./admin_topic.md#6-推荐--取消推荐)

### 评论管理（检索 / 删除）

- 评论列表/检索
  - `GET /api/admin/comment/list`（已存在；至少填写一个筛选条件才返回数据）
  - 详见：[评论管理（Admin Comment）](./admin_comment.md#1-评论列表检索分页)
- 删除评论
  - `POST /api/admin/comment/delete/{id}`（已存在）
  - 详见：[评论管理（Admin Comment）](./admin_comment.md#2-删除评论)

### 节点管理（板块）

- 节点列表/创建/更新
  - `GET /api/admin/topic-node/list`
  - `POST /api/admin/topic-node/create`
  - `POST /api/admin/topic-node/update`
  - 详见：[节点管理（Admin TopicNode）](./admin_topic_node.md)
- 更新排序
  - `POST /api/admin/topic-node/update_sort`（JSON 数组：节点 id 新顺序）
  - 详见：[节点管理（Admin TopicNode）](./admin_topic_node.md#6-更新排序)

### 敏感词管理

- 敏感词列表/新增/更新/删除
  - `GET /api/admin/forbidden-word/list`
  - `POST /api/admin/forbidden-word/create`
  - `POST /api/admin/forbidden-word/update`
  - `POST /api/admin/forbidden-word/delete`
  - 详见：[敏感词管理（Admin ForbiddenWord）](./admin_forbidden_word.md)

### 用户侧举报

- 举报帖子
  - `POST /api/user-report/submit`（已存在，当前提供通用举报：dataType + dataId）

### 举报审核（运营侧处理举报）

- 举报列表/详情
  - `GET /api/admin/user-report/list`（已存在）
  - `GET /api/admin/user-report/by/{id}`（已存在）
  - 详见：[举报审核（Admin UserReport）](./admin_user_report.md#1-举报列表分页)
- 处理动作（通过/驳回/忽略等：自行约定 auditStatus / auditUserId / auditTime）
  - `POST /api/admin/user-report/update`（已存在）
  - 详见：[举报审核（Admin UserReport）](./admin_user_report.md#3-更新举报审核处理)


## 用户管理

- 用户列表/搜索
  - `GET /api/admin/user/list`（已存在；支持按 id/nickname/email/username/type 筛选 + 分页）
  - 详见：[用户管理（Admin User）](./admin_user.md#1-用户列表搜索分页)

- 授权为管理员 / 取消管理员
  - `POST /api/admin/user/grant_admin`（已实现，owner-only）
  - `POST /api/admin/user/revoke_admin`（已实现，owner-only）
  - 详见：[用户管理（Admin User）](./admin_user.md#2-授权为管理员owner-only)

- 金币（后台给用户加金币）
  - `POST /api/admin/coin/mint`（已存在）
  - 详见：[下注/结算系统（金币与预测下注）](./coin.md#4-管理员铸币给用户加金币)

- 禁言/解禁
  - `POST /api/user/forbidden`（已存在；owner/admin；days=0 解禁；days=-1 永久禁言仅 owner）
  - 详见：[用户系统](./user.md#11-禁言管理员)

> 所有权限变更建议强制二次确认，并写入 operate-log。
