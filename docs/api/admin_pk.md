# 对立PK管理（Admin PK）

> 设计稿文档：本模块尚未在 `internal/server/router.go` 注册。建议新增管理端路由组 `/api/admin/pk`，经过 `AuthMiddleware` 与 `AdminMiddleware`。

## 模块说明

管理端负责维护对立PK话题、查看回合与赛季、处理异常热度重算。

默认话题建议通过 migration 幂等初始化；如果需要后台按钮，再补初始化接口。

## 通用约定

- 基础路径：`/api/admin/pk`
- 认证：需要管理员权限。
- 返回结构沿用项目通用 `web.JsonResult`。
- 状态：话题 `enabled/disabled`；赛季 `active/finished`；回合 `betting/locked/cooldown/settled`。

## 接口列表

### 1. 话题管理列表

- 接口：`GET /api/admin/pk/topic/list`
- 功能：分页查看PK话题。

#### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `page` | int | 否 | 默认 1 |
| `pageSize` | int | 否 | 默认 20 |
| `status` | string | 否 | `enabled/disabled` |
| `q` | string | 否 | 搜索标题、slug、阵营名称 |

#### 返回 data

```json
{
  "list": [
    {
      "topic": {},
      "round": {},
      "season": {},
      "stats": {
        "totalRounds": 20,
        "winsA": 11,
        "winsB": 9
      }
    }
  ],
  "count": 16
}
```

#### 后端工作

- 分页查询 `PKTopic`。
- 批量加载当前 `PKRound`、`PKSeason`。
- 聚合总局数、胜场等统计。

### 2. 新增或编辑话题

- 接口：`POST /api/admin/pk/topic/save`
- 功能：创建或编辑PK话题。
- 请求格式：JSON。

#### 请求体

```json
{
  "id": 1,
  "slug": "pk-hero",
  "title": "足球GOAT之争",
  "sideAName": "梅西",
  "sideBName": "C罗",
  "status": "enabled",
  "sort": 100,
  "cover": ""
}
```

#### 后端工作

- 校验 `title`、`sideAName`、`sideBName` 必填。
- 校验 `slug` 格式与唯一性。
- 新建话题时同步创建第一赛季和第一回合。
- 编辑进行中的话题时，限制破坏性字段变更。
- 写入操作日志。

#### 可能错误

- `title is required`
- `sides are required`
- `slug already exists`
- `pk topic not found`

### 3. 启用或停用话题

- 接口：`POST /api/admin/pk/topic/status`
- 功能：启用或停用PK话题。
- 请求格式：JSON。

#### 请求体

```json
{
  "topicId": 1,
  "status": "disabled"
}
```

#### 后端工作

- 校验话题存在。
- 启用时确保存在当前赛季和当前回合。
- 停用后用户端列表隐藏，当前回合不再接受新动作。
- 写入操作日志。

### 4. 回合管理列表

- 接口：`GET /api/admin/pk/round/list`
- 功能：分页查询PK回合记录。

#### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `page` | int | 否 | 默认 1 |
| `pageSize` | int | 否 | 默认 20 |
| `topicId` | int64 | 否 | 按话题筛选 |
| `phase` | string | 否 | 按阶段筛选 |
| `winner` | string | 否 | `A/B/draw` |
| `startTime` | int64 | 否 | 起始时间 |
| `endTime` | int64 | 否 | 结束时间 |

#### 返回 data

```json
{
  "list": [
    {
      "round": {},
      "topic": {}
    }
  ],
  "count": 1
}
```

#### 后端工作

- 查询 `PKRound`。
- 关联 `PKTopic`。
- 返回热度、奖池、下注人数、胜方、结算时间。

### 5. 赛季管理列表

- 接口：`GET /api/admin/pk/season/list`
- 功能：分页查询PK赛季记录。

#### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `page` | int | 否 | 默认 1 |
| `pageSize` | int | 否 | 默认 20 |
| `topicId` | int64 | 否 | 按话题筛选 |
| `status` | string | 否 | `active/finished` |

#### 返回 data

```json
{
  "list": [
    {
      "season": {},
      "topic": {}
    }
  ],
  "count": 1
}
```

### 6. 管理员重算热度

- 接口：`POST /api/admin/pk/recalc_heat`
- 功能：异常修复或规则调整后重算指定回合热度。
- 请求格式：JSON。

#### 请求体

```json
{
  "topicId": 1,
  "roundId": 10,
  "reason": "fix comment heat"
}
```

#### 后端工作

- 校验话题和回合存在。
- 默认只允许未结算回合重算。
- 基于 `PKCommentMeta`、`Comment`、`UserLike`、`PKAction` 重算。
- 更新 `PKCommentMeta.heatScore`。
- 更新 `PKRound.heatA/heatB`。
- 写入操作日志。

#### 可能错误

- `topicId is required`
- `roundId is required`
- `pk round not found`
- `settled round cannot recalc heat`

## 现有接口影响

### 评论管理

现有 `docs/api/admin_comment.md` 可继续管理通用评论。

如需管理PK评论的阵营、热度、拉踩数，建议后续新增：

- `GET /api/admin/pk/comment/list`
- `POST /api/admin/pk/comment/recalc`

### 金币流水

现有管理员金币流水接口可继续按 `bizType` 查询。

建议新增流水类型：

- `PK_BET_STAKE_IN`
- `PK_PAYOUT`
- `PK_DRAW_REFUND`

### 操作日志

话题新增、编辑、启停、热度重算都应写入操作日志，便于回溯运营行为。

## 定时任务

管理端不直接暴露定时任务接口。后端实现时在 `internal/scheduler/cron.go` 注册：

- `PKService.CronTick()`

任务内容：

- `betting -> locked`
- `locked -> cooldown`
- `cooldown -> betting`
- 赛季归档与下一赛季创建

所有任务必须幂等。
