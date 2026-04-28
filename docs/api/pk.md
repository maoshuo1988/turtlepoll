# 对立PK（PK）

> 设计稿文档：本模块尚未在 `internal/server/router.go` 注册。建议新增用户端路由组 `/api/pk`，默认经过 `AuthMiddleware`。

## 模块说明

对立PK是“永恒话题 + A/B 阵营 + 72小时回合制”的玩法模块。

核心口径：

- 每个话题长期存在。
- 每局分为下注期、锁局期、冷却期。
- 用户每局只能下注一次，固定 100 龟币。
- 评论、点赞、回复、拉踩和下注共同影响阵营热度。
- 每局结束后按热度判胜负，胜方瓜分败方奖池。

## 通用约定

- 基础路径：`/api/pk`
- 认证：需要登录，除非后续明确允许游客只读。
- 时间戳：秒级 Unix 时间戳。
- 阵营：`A` / `B`。
- 阶段：`betting` / `locked` / `cooldown` / `settled`。
- 返回结构沿用项目通用 `web.JsonResult`。

## 数据结构摘要

### PKTopic

| 字段 | 说明 |
|------|------|
| `id` | 话题ID |
| `slug` | 业务标识 |
| `title` | 话题标题 |
| `sideAName` / `sideBName` | A/B 阵营名称 |
| `status` | `enabled/disabled` |
| `currentRoundId` | 当前局ID |
| `currentSeasonId` | 当前赛季ID |

### PKRound

| 字段 | 说明 |
|------|------|
| `id` | 回合ID |
| `topicId` | 话题ID |
| `seasonId` | 赛季ID |
| `roundNo` | 局号 |
| `phase` | 当前阶段 |
| `startTime/lockTime/endTime/nextRoundTime` | 阶段时间 |
| `heatA/heatB` | A/B 热度 |
| `poolA/poolB` | A/B 奖池 |
| `betCountA/betCountB` | A/B 下注人数 |
| `winner` | `A/B/draw` |

### PKBet

| 字段 | 说明 |
|------|------|
| `id` | 下注ID |
| `topicId` / `roundId` | 话题与回合 |
| `userId` | 用户ID |
| `side` | `A/B` |
| `amount` | 下注额，P0 固定 100 |
| `requestId` | 幂等键 |
| `settleResult` | `win/lose/draw` |
| `payout` | 派奖金额 |

## 接口列表

### 1. PK话题列表

- 接口：`GET /api/pk/topics`
- 功能：获取所有启用的对立PK话题。

#### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `page` | int | 否 | 默认 1 |
| `pageSize` | int | 否 | 默认 20，最大 100 |

#### 返回 data

```json
{
  "list": [
    {
      "topic": {},
      "round": {},
      "season": {},
      "oddsA": 1.8,
      "oddsB": 2.2,
      "leader": "A",
      "streakStatus": "defending",
      "countdownSeconds": 3600,
      "mySide": "A",
      "myBet": {}
    }
  ],
  "count": 16
}
```

#### 后端工作

- 查询启用的 `PKTopic`。
- 批量加载当前 `PKRound`、`PKSeason`。
- 计算赔率、领先方、倒计时、守擂/翻盘状态。
- 当前用户已下注时返回 `mySide/myBet`。

### 2. PK话题详情

- 接口：`GET /api/pk/topic`
- 功能：获取单个PK详情页首屏数据。

#### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `topicId` | int64 | 与 `slug` 二选一 | 话题ID |
| `slug` | string | 与 `topicId` 二选一 | 话题标识 |

#### 返回 data

```json
{
  "topic": {},
  "round": {},
  "season": {},
  "stats": {
    "totalRounds": 20,
    "winsA": 11,
    "winsB": 9,
    "currentStreakSide": "A",
    "currentStreak": 3
  },
  "recentRounds": [],
  "myBet": {},
  "oddsA": 1.8,
  "oddsB": 2.2,
  "countdownSeconds": 3600
}
```

#### 可能错误

- `topicId or slug is required`
- `pk topic not found`

### 3. 当前局下注

- 接口：`POST /api/pk/bet`
- 功能：用户在下注期选择阵营下注。
- 请求格式：JSON。

#### 请求体

```json
{
  "topicId": 1,
  "side": "A",
  "requestId": "client-uuid"
}
```

#### 返回 data

```json
{
  "bet": {},
  "round": {},
  "userCoin": {},
  "oddsA": 1.8,
  "oddsB": 2.2
}
```

#### 后端工作

- 校验登录、话题启用、当前阶段为 `betting`。
- 校验 `side=A/B`。
- 同一用户同一局只能下注一次。
- 相同 `requestId` 重试返回已有记录，不重复扣款。
- 固定扣除 100 龟币，复用 `UserCoinService.SpendToPool`。
- 写入 `PKBet`，更新奖池、下注人数、下注热度。

#### 可能错误

- `pk topic not found`
- `pk round is not betting`
- `invalid side`
- `already bet in this round`
- `insufficient balance`
- `requestId is required`

### 4. 实时热度

- 接口：`GET /api/pk/heat`
- 功能：获取当前局实时热度。

#### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `topicId` | int64 | 是 | 话题ID |

#### 返回 data

```json
{
  "roundId": 10,
  "phase": "betting",
  "heatA": 128.5,
  "heatB": 110.2,
  "leader": "A",
  "streakStatus": "defending",
  "countdownSeconds": 3600
}
```

### 5. PK评论列表

- 接口：`GET /api/pk/comments`
- 功能：获取指定阵营的评论战场列表。

#### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `topicId` | int64 | 是 | 话题ID |
| `side` | string | 是 | `A/B` |
| `cursor` | int64 | 否 | 游标 |
| `sort` | string | 否 | `time/heat`，默认 `time` |

#### 返回 data

返回 `web.JsonCursorData`：

- `data`: 评论列表，每项包含现有评论渲染字段、`side`、`heatScore`、`downvoteCount`。
- `cursor`: 下一页游标。
- `hasMore`: 是否还有更多。

### 6. 发布PK评论

- 接口：`POST /api/pk/comment/create`
- 功能：发布带阵营归属的PK评论。
- 请求格式：表单或 JSON，建议实现时与现有评论接口保持一致。

#### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `topicId` | int64 | 是 | 话题ID |
| `side` | string | 是 | `A/B` |
| `content` | string | 是 | 评论内容 |
| `imageList` | string | 否 | JSON数组字符串 |
| `quoteId` | int64 | 否 | 引用评论ID |

#### 后端工作

- 校验登录、发帖权限、spam。
- 校验当前阶段不是 `cooldown`。
- 创建通用 `Comment`。
- 写入 `PKCommentMeta` 绑定话题、回合、阵营。
- 写入 `PKAction(actionType=comment)` 并更新热度。

### 7. PK评论回复

- 接口：`POST /api/pk/comment/reply`
- 功能：回复PK评论，回复归属父评论阵营。

#### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `commentId` | int64 | 是 | 父评论ID |
| `content` | string | 是 | 回复内容 |
| `imageList` | string | 否 | JSON数组字符串 |
| `quoteId` | int64 | 否 | 引用评论ID |

#### 后端工作

- 查询父评论 `PKCommentMeta`。
- 校验当前回合允许互动。
- 创建 `entityType=comment` 的回复。
- 写入 `PKAction(actionType=reply)` 并更新热度。

### 8. 拉踩评论

- 接口：`POST /api/pk/downvote`
- 功能：拉踩对方评论，降低目标评论热度。
- 请求格式：JSON。

#### 请求体

```json
{
  "commentId": 1,
  "requestId": "client-uuid"
}
```

#### 后端工作

- 校验目标评论属于当前PK回合。
- 校验用户本局阵营，且只能拉踩对方阵营评论。
- 校验同一用户不能重复拉踩同一评论。
- 写入 `PKAction(actionType=downvote)` 或独立拉踩表。
- 更新评论热度和回合热度。

### 9. 历史对局

- 接口：`GET /api/pk/history`
- 功能：分页查看某话题历史每局结果。

#### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `topicId` | int64 | 是 | 话题ID |
| `page` | int | 否 | 默认 1 |
| `pageSize` | int | 否 | 默认 20 |

#### 返回 data

- `list`: `PKRound[]`
- `count`: 总数

### 10. 赛季历史

- 接口：`GET /api/pk/seasons`
- 功能：分页查看某话题赛季记录。

#### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `topicId` | int64 | 是 | 话题ID |
| `page` | int | 否 | 默认 1 |
| `pageSize` | int | 否 | 默认 20 |

#### 返回 data

- `list`: `PKSeason[]`
- `count`: 总数

### 11. 我的PK记录

- 接口：`GET /api/pk/my/bets`
- 功能：分页查看当前用户参与过的PK下注记录。

#### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `page` | int | 否 | 默认 1 |
| `pageSize` | int | 否 | 默认 20 |

#### 返回 data

```json
{
  "list": [
    {
      "bet": {},
      "topic": {},
      "round": {}
    }
  ],
  "count": 1
}
```

## 实现注意

- 下注、拉踩、派奖都必须幂等。
- 评论和回复建议封装 PK 专用接口，不直接扩展通用评论参数。
- 点赞可先复用 `/api/like/like`，通过事件或热度重算识别 PK 评论。
- 金币 API 不需要新增用户端接口，服务层复用 `UserCoinService`。
