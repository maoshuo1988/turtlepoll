# 预测事件系统（/api/predict）

路由前缀：`/api/predict`

认证说明：

- `/api/predict/**` 默认需要登录。
- `/api/predict/admin/**` 需要管理员权限。

说明：

- 本文统一使用 `/api/predict` 作为对外契约路径。
- 当前代码实现仍分布在 `football/comment/like/coin/admin predict` 等 controller 中，后续以路由迁移为准。
- 本文档与 `prompt/project/预测市场公共部分/06-接口.md` 保持同构，作为 `docs/api` 下的实现对齐版本。

## 接口总览

| 分类 | 方法 | 路径 | 认证 | 说明 | 当前实现映射 |
|------|------|------|------|------|------|
| 市场查询 | GET | `/api/predict/markets` | 是 | 查询预测市场聚合列表 | `/api/predict/markets`（内部复用 football 查询逻辑） |
| 市场查询 | GET | `/api/predict/markets/by-name` | 是 | 按名称模糊查询市场 | `/api/predict/markets/by-name`（兼容旧：`/api/football/markets/by_name`） |
| 市场查询 | GET | `/api/predict/markets/by-tag` | 是 | 按标签查询市场 | `/api/predict/markets/by-tag`（兼容旧：`/api/football/markets/by_tag`） |
| 市场查询 | GET | `/api/predict/bet-settle-result` | 是 | 查询当前用户某市场下注结算结果 | `/api/predict/bet-settle-result`（兼容旧：`/api/football/bet_settle_result`） |
| 撕裂带结算 | POST | `/api/predict/tear/settle` | 是 | 结算预测市场撕裂带奖励 | `/api/predict/tear/settle` |
| 热度 | GET | `/api/predict/heat` | 是 | 当前预测市场热度快照 | `/api/predict/heat` |
| 评论 | GET | `/api/predict/comments` | 是 | 预测市场不同阵营评论列表 | `/api/predict/comments` |
| 评论 | GET | `/api/predict/comment/replies` | 是 | 评论回复列表 | `/api/predict/comment/replies` |
| 热度 | GET | `/api/predict/heat/rank` | 是 | 事件热度榜 | `/api/predict/heat/rank` |
| 热度 | GET | `/api/predict/heat/me` | 是 | 我的热度数据 | `/api/predict/heat/me` |
| 赔率 | GET | `/api/predict/odds/current` | 是 | 当前赔率 | `/api/predict/odds/current` |
| 上下文 | POST | `/api/predict/context/update` | 是 | 按 marketId upsert PredictContext | `/api/predict/context/update`（内部复用 football context 逻辑） |
| 上下文 | GET | `/api/predict/context/hot` | 是 | 热度榜 Top N | `/api/predict/context/hot`（兼容旧：`/api/football/predict_context/hot`） |
| 标签 | GET | `/api/predict/tags/hot` | 是 | 热门标签 Top N | `/api/predict/tags/hot`（兼容旧：`/api/football/predict_tags/hot`） |
| 下注 | POST | `/api/predict/coin/bet` | 是 | 下注、锁赔率、更新池子 | `/api/predict/coin/bet`（兼容旧：`/api/coin/bet`） |
| 结算 | POST | `/api/predict/coin/settle` | 是 | 用户下注结算 | `/api/predict/coin/settle`（兼容旧：`/api/coin/settle`） |
| 管理结算 | POST | `/api/predict/admin/market/settle` | 管理员 | 运营人工结算市场 | `/api/predict/admin/market/settle`（兼容旧：`/api/admin/predict/market/settle`） |
| 评论 | POST | `/api/predict/comment/create` | 是 | 预测市场一级评论 | `/api/predict/comment/create`（兼容旧：`/api/comment`） |
| 回复 | POST | `/api/predict/comment/reply` | 是 | 预测市场回复评论 | `/api/predict/comment/reply`（兼容旧：`/api/comment`） |
| 点赞 | POST | `/api/predict/like` | 是 | 评论/回复点赞 | `/api/predict/like`（兼容旧：`/api/like/like`） |
| 取消点赞 | POST | `/api/predict/unlike` | 是 | 评论/回复取消点赞 | `/api/predict/unlike`（兼容旧：`/api/like/unlike`） |
| 奖励 | POST | `/api/predict/admin/comment-reward/run` | 管理员 | 手动触发开撕台评论奖励 | `/api/predict/admin/comment-reward/run`（兼容旧：`/api/admin/predict/comment_reward/run`） |
| 奖励 | POST | `/api/predict/admin/comment-reward/retry` | 管理员 | 重试失败奖励批次 | `/api/predict/admin/comment-reward/retry`（兼容旧：`/api/admin/predict/comment_reward/retry`） |
| 奖励 | GET | `/api/predict/admin/comment-reward/logs` | 管理员 | 查询奖励批次与明细 | `/api/predict/admin/comment-reward/logs`（兼容旧：`/api/admin/predict/comment_reward/logs`） |

## 1. 市场查询接口

统一约定：以下市场查询接口都返回 `tearSettlement`，用于前端直接判断“是否可触发撕裂带结算”与“当前结算状态”。

- `GET /api/predict/markets`
- `GET /api/predict/markets/by-name`
- `GET /api/predict/markets/by-tag`
- `GET /api/predict/bet-settle-result`

### 1.1 查询预测市场聚合列表

```text
GET /api/predict/markets
```

认证：需要登录。

查询参数：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `page` | int | 否 | 页码，默认 1 |
| `limit` | int | 否 | 每页条数，默认 20，最大 100 |
| `sourceModel` | string | 否 | 来源模型，例如 `MatchSchedule` |
| `sourceModelId` | int64 | 否 | 来源模型 ID |

返回值（data）：

- `list`
- `total`

`list` 每项包含：

- `market`
- `context`
- `hasBet`
- `betSettleResult`
- `tearSettlement`
- `schedule`
- `matchPhase`

`tearSettlement` 字段：

- `canSettle`: 是否允许触发撕裂带结算
- `status`: 撕裂带结算状态（`NONE/PENDING/PROCESSING/PAID/EXPIRED/FAILED`）
- `reason`: 不可结算原因（如 `MARKET_NOT_SETTLED/ALREADY_PAID/DEADLINE_EXPIRED`）
- `settledAt`: 市场结算时间（秒）
- `deadlineAt`: 撕裂带结算截止时间（秒）
- `remainSeconds`: 距截止剩余秒数
- `rewardLogId`: 奖励批次 ID（存在时）
- `winnerOption`: 结算胜方

返回示例：

```json
{
  "list": [
    {
      "market": {
        "id": 1,
        "sourceModel": "MatchSchedule",
        "sourceModelId": 1001,
        "title": "阿根廷 vs 法国",
        "marketType": "1x2",
        "status": "OPEN",
        "closeTime": 1734012345,
        "result": "",
        "createTime": 1734010000,
        "updateTime": 1734010000
      },
      "context": {
        "id": 10,
        "marketId": 1,
        "eventName": "世界杯决赛",
        "heat": 999,
        "tags": "wc,final"
      },
      "schedule": {
        "id": 1001,
        "matchPhase": "GROUP_STAGE",
        "status": "FINISHED"
      },
      "matchPhase": "GROUP_STAGE",
      "betSettleResult": "WIN",
      "hasBet": true,
      "tearSettlement": {
        "canSettle": true,
        "status": "PENDING",
        "reason": "",
        "settledAt": 1782700000,
        "deadlineAt": 1782703600,
        "remainSeconds": 1200,
        "rewardLogId": 0,
        "winnerOption": "A"
      }
    }
  ],
  "total": 1
}
```

可能错误：

- 未登录：`errs.NotLogin()`

### 1.2 按名称模糊查询市场

```text
GET /api/predict/markets/by-name
```

认证：需要登录。

查询参数：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `q` | string | 是 | 关键词 |
| `page` | int | 否 | 页码，默认 1 |
| `limit` | int | 否 | 每页条数，默认 20，最大 100 |

返回值（data）：

- `list`
- `total`
- `q`
- `list[].tearSettlement`

说明：

- 命中 `PredictMarket.title`
- 或命中 `PredictContext.eventName`

可能错误：

- `q is required`

### 1.3 按标签查询市场

```text
GET /api/predict/markets/by-tag
```

认证：需要登录。

查询参数：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `tag` | string | 是 | 标签 |
| `page` | int | 否 | 页码，默认 1 |
| `limit` | int | 否 | 每页条数，默认 20，最大 100 |

返回值（data）：

- `list`
- `total`
- `page`
- `limit`
- `tag`
- `list[].tearSettlement`

说明：

- `list` 为 market + context 聚合结果。
- 排序优先级：标签命中强度、非 TBD、状态、收盘时间、热度。

可能错误：

- `tag is required`

### 1.4 查询用户在某市场的下注结算结果

```text
GET /api/predict/bet-settle-result
```

认证：需要登录。

权限：仅允许查询本人。

查询参数：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `userId` | int64 | 是 | 当前登录用户 ID |
| `marketId` | int64 | 是 | 市场 ID |

返回值（data）：

- `userId`
- `marketId`
- `betSettleResult`
- `tearSettlement`

说明：

- 无下注返回空字符串。
- 多笔下注时优先返回 `WIN`，其次 `LOSE`，否则返回最新一条非空值。

可能错误：

- 未登录：`errs.NotLogin()`
- 非本人：`errs.NoPermission()`

## 2. 上下文与标签接口

### 2.1 创建或修改 PredictContext

```text
POST /api/predict/context/update
```

认证：需要登录。

请求格式：表单。

参数：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `marketId` | int64 | 是 | 市场 ID |
| `eventName` | string | 是 | 展示名称 |
| `imageUrl` | string | 否 | 大图 |
| `listImage` | string | 否 | 列表图 |
| `sideABgImage` | string | 否 | A 阵营背景图 |
| `sideBBgImage` | string | 否 | B 阵营背景图 |
| `sideABgColor` | string | 否 | A 阵营背景色 |
| `sideBBgColor` | string | 否 | B 阵营背景色 |
| `participantCount` | int64 | 否 | 参与人数 |
| `proText` | string | 否 | A 阵营文案 |
| `conText` | string | 否 | B 阵营文案 |
| `proVoteCount` | int64 | 否 | A 阵营票数 |
| `conVoteCount` | int64 | 否 | B 阵营票数 |
| `heat` | int64 | 否 | 热度 |
| `detail` | string | 否 | 详情 |
| `tags` | string | 否 | 标签 |

返回值（data）：

- 更新后的 `PredictContext`

可能错误：

- 未登录：`errs.NotLogin()`
- `marketId is required`
- `eventName is required`

### 2.2 热度榜

```text
GET /api/predict/context/hot
```

认证：需要登录。

查询参数：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `limit` | int | 否 | 返回数量，默认 10，最大 100 |

返回值（data）：

- `list`
- `limit`

### 2.3 热门标签榜

```text
GET /api/predict/tags/hot
```

认证：需要登录。

查询参数：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `limit` | int | 否 | 返回数量，默认 10，最大 100 |

返回值（data）：

- `list`
- `limit`

## 3. 热度、评论、赔率与撕裂带结算接口

### 3.1 当前预测市场热度

```text
GET /api/predict/heat
```

认证：需要登录。

查询参数：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `marketId` | int64 | 是 | 市场 ID |

返回值（data）：

- `marketId`
- `marketType`
- `status`
- `options`（每项含 `option/hLike/hComment/hCoin/hTotal/snapshotType/snapshotTime`）
- `snapshotTime`
- `snapshotType`
- `leaderOption`
- `totalHeatValue`

请求参数 JSON 示例（Query）：

```json
{
  "marketId": 101
}
```

返回 JSON 示例（data）：

```json
{
  "marketId": 101,
  "marketType": "1x2",
  "status": "OPEN",
  "options": [
    {
      "option": "A",
      "hLike": 12,
      "hComment": 36.5,
      "hCoin": 18.2,
      "hTotal": 66.7,
      "snapshotType": "CHECKPOINT",
      "snapshotTime": 1782700000
    },
    {
      "option": "B",
      "hLike": 9,
      "hComment": 28.3,
      "hCoin": 20.0,
      "hTotal": 57.3,
      "snapshotType": "CHECKPOINT",
      "snapshotTime": 1782700000
    },
    {
      "option": "DRAW",
      "hLike": 2,
      "hComment": 4.1,
      "hCoin": 3.8,
      "hTotal": 9.9,
      "snapshotType": "CHECKPOINT",
      "snapshotTime": 1782700000
    }
  ],
  "snapshotTime": 1782700000,
  "snapshotType": "CHECKPOINT",
  "leaderOption": "A",
  "totalHeatValue": 66.7
}
```

### 3.2 结算预测市场撕裂带奖励

```text
POST /api/predict/tear/settle
```

认证：需要登录。

说明：

- 该接口触发“预测市场公共部分”的撕裂带评论奖励结算。
- 结算成功后会生成或更新一条 `PredictCommentRewardLog`。

请求参数：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `marketId` | int64 | 是 | 市场 ID |

返回值（data）：

- `marketId`
- `rewardLog`
- `tearSettlement`

请求 JSON 示例：

```json
{
  "marketId": 101
}
```

返回 JSON 示例（data）：

```json
{
  "marketId": 101,
  "rewardLog": {
    "id": 501,
    "marketId": 101,
    "winnerOption": "A",
    "marketBetTotal": 6900,
    "rewardPool": 690,
    "winnerTotalCommentHeat": 88.4,
    "winnerCommentUserCount": 17,
    "perUserReward": 40,
    "remainder": 0,
    "status": "PAID",
    "reason": "",
    "settledAt": 1782700000,
    "deadlineAt": 1782703600,
    "paidAt": 1782700500
  },
  "tearSettlement": {
    "canSettle": false,
    "status": "PAID",
    "reason": "ALREADY_PAID",
    "settledAt": 1782700000,
    "deadlineAt": 1782703600,
    "remainSeconds": 0,
    "rewardLogId": 501,
    "winnerOption": "A"
  }
}
```

### 3.3 预测市场不同阵营评论列表

```text
GET /api/predict/comments
```

认证：需要登录。

查询参数：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `marketId` | int64 | 是 | 市场 ID |
| `option` | string | 否 | 阵营过滤：`A/B/DRAW` |
| `cursor` | int64 | 否 | 游标（评论 ID） |
| `pageSize` | int | 否 | 每页数量，默认 20，最大 100 |

返回：游标分页结构（`results/cursor/hasMore`），`results` 为评论渲染对象。

请求参数 JSON 示例（Query）：

```json
{
  "marketId": 101,
  "option": "A",
  "cursor": 0,
  "pageSize": 20
}
```

返回 JSON 示例（data）：

```json
{
  "results": [
    {
      "id": 90001,
      "user": {
        "id": 2001,
        "nickname": "Alice",
        "avatar": "https://cdn.example.com/avatar/alice.png"
      },
      "optionAtAction": "A",
      "entityType": "predict_market",
      "entityId": 101,
      "content": "我看好 A 阵营今晚反超。",
      "likeCount": 12,
      "commentCount": 2,
      "createTime": 1782699800
    }
  ],
  "cursor": "90001",
  "hasMore": true
}
```

### 3.4 评论回复列表

```text
GET /api/predict/comment/replies
```

认证：需要登录。

查询参数：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `commentId` | int64 | 是 | 父评论 ID |
| `cursor` | int64 | 否 | 游标 |

返回：游标分页结构（`results/cursor/hasMore`）。

请求参数 JSON 示例（Query）：

```json
{
  "commentId": 90001,
  "cursor": 0
}
```

返回 JSON 示例（data）：

```json
{
  "results": [
    {
      "id": 90011,
      "user": {
        "id": 2010,
        "nickname": "Bob",
        "avatar": "https://cdn.example.com/avatar/bob.png"
      },
      "optionAtAction": "A",
      "entityType": "comment",
      "entityId": 90001,
      "content": "同意，这场 A 的盘口很稳。",
      "likeCount": 3,
      "commentCount": 0,
      "createTime": 1782699900
    }
  ],
  "cursor": "90011",
  "hasMore": false
}
```

### 3.5 事件热度榜

```text
GET /api/predict/heat/rank
```

认证：需要登录。

查询参数：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `marketId` | int64 | 是 | 市场 ID |
| `scope` | string | 否 | `ALL/MY_SIDE` |
| `page` | int | 否 | 页码，默认 1 |
| `pageSize` | int | 否 | 每页数量，默认 20，最大 100 |

返回值（data）：

- `marketId`
- `scope`
- `myOption`
- `list`（每项含 `rank/userId/nickname/avatar/option/totalHeat/commentHeat/likeHeat/coinHeat`）
- `count/page/pageSize`

请求参数 JSON 示例（Query）：

```json
{
  "marketId": 101,
  "scope": "ALL",
  "page": 1,
  "pageSize": 20
}
```

返回 JSON 示例（data）：

```json
{
  "marketId": 101,
  "scope": "ALL",
  "myOption": "A",
  "list": [
    {
      "rank": 1,
      "userId": 2001,
      "nickname": "Alice",
      "avatar": "https://cdn.example.com/avatar/alice.png",
      "option": "A",
      "totalHeat": 88.4,
      "commentHeat": 52.4,
      "likeHeat": 16,
      "coinHeat": 20
    }
  ],
  "count": 36,
  "page": 1,
  "pageSize": 20
}
```

### 3.6 我的热度数据

```text
GET /api/predict/heat/me
```

认证：需要登录。

查询参数：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `marketId` | int64 | 是 | 市场 ID |

返回值（data）：

- `marketId`
- `userId`
- `myOption`
- `myHeat`
- `myRank`
- `commentHeat`
- `likeHeat`
- `coinHeat`
- `myActionCount`
- `myCommentCount`
- `receivedLikeCount`
- `myBetAmount`

请求参数 JSON 示例（Query）：

```json
{
  "marketId": 101
}
```

返回 JSON 示例（data）：

```json
{
  "marketId": 101,
  "userId": 2001,
  "myOption": "A",
  "myHeat": 88.4,
  "myRank": 1,
  "commentHeat": 52.4,
  "likeHeat": 16,
  "coinHeat": 20,
  "myActionCount": 34,
  "myCommentCount": 11,
  "receivedLikeCount": 16,
  "myBetAmount": 1000
}
```

### 3.7 当前赔率

```text
GET /api/predict/odds/current
```

认证：需要登录。

查询参数：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `marketId` | int64 | 是 | 市场 ID |

返回值（data）：

- `marketId`
- `marketType`
- `status`
- `oddsA`
- `oddsB`
- `oddsDraw`（仅 1x2）
- `effA/effB/effDraw`
- `totalEffPool`

请求参数 JSON 示例（Query）：

```json
{
  "marketId": 101
}
```

返回 JSON 示例（data）：

```json
{
  "marketId": 101,
  "marketType": "1x2",
  "status": "OPEN",
  "oddsA": 1.93,
  "oddsB": 2.15,
  "oddsDraw": 3.42,
  "effA": 2800,
  "effB": 2500,
  "effDraw": 1600,
  "totalEffPool": 6900
}
```

## 4. 下注与结算接口

### 4.1 下注

```text
POST /api/predict/coin/bet
```

认证：需要登录。

参数：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `marketId` | int64 | 是 | 市场 ID |
| `option` | string | 是 | `A/B/DRAW` |
| `amount` | int64 | 是 | 下注金额 |

规则：

- 下注成功后锁定赔率并更新市场池子。
- 下注前会建立或校验阵营锁边。
- 锁边冲突时返回 `TEAR_CAMP_LOCKED_BY_BET` 或 `TEAR_CAMP_CONFLICT`。

返回：

- `bet`
- `market`
- `userCoin`
- `lockedOdds`

### 4.2 用户结算

```text
POST /api/predict/coin/settle
```

认证：需要登录。

参数：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `marketId` | int64 | 是 | 市场 ID |

规则：

- 市场必须是 `SETTLED`。
- `PredictMarket.result` 必须合法。
- 只结算当前用户未结算下注单。
- 结算成功后幂等更新用户胜率和最新连胜。

### 4.3 管理员结算市场

```text
POST /api/predict/admin/market/settle
```

认证：管理员。

请求：

```json
{
  "marketId": 1,
  "result": "A",
  "requestId": "manual-001",
  "remark": "manual settle",
  "allowReset": false
}
```

规则：

- `binary` 允许 `A/B`。
- `1x2` 允许 `A/B/DRAW`。
- 默认只允许 `CLOSED/CLOSE -> SETTLED`。
- `allowReset=true` 可重设已结算市场。

## 5. 评论、回复与点赞接口

### 5.1 预测市场一级评论

```text
POST /api/predict/comment/create
```

认证：需要登录。

公共影响：

- 一级评论会写入预测评论元数据。
- 评论热度进入热度快照。
- 评论选项用于后续奖励聚合。

参数：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `entityType` | string | 是 | 预测市场评论实体类型 |
| `entityId` | int64 | 是 | `marketId` |
| `content` | string | 是 | 评论内容 |
| `option` | string | 是 | `A/B/DRAW`，必须匹配市场类型 |
| `requestId` | string | 否 | 幂等请求 ID，推荐传 |

写入：

- `Comment`
- `PredictCommentMeta.optionAtAction`
- `PredictCampMember(lockType=INTERACT)`

可能错误：

- `PREDICT_400_MARKET_ID_REQUIRED`
- `PREDICT_404_MARKET_NOT_FOUND`
- `PREDICT_422_OPTION_INVALID`
- `TEAR_CAMP_OPTION_REQUIRED`

### 5.2 预测市场回复评论

```text
POST /api/predict/comment/reply
```

认证：需要登录。

参数：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `commentId` | int64 | 是 | 父评论 ID |
| `content` | string | 是 | 回复内容 |
| `marketId` | int64 | 是 | 市场 ID |
| `option` | string | 否 | `A/B/DRAW`，未锁边场景可传 |
| `requestId` | string | 否 | 幂等请求 ID，推荐传 |

规则：

- 回复会按写入时推导 `optionAtAction`。
- 推导顺序：用户锁边优先 -> 请求 `option` -> 父评论归属兜底。
- 回复热度进入评论热度统计。

可能错误：

- `TEAR_CAMP_CONFLICT`
- `TEAR_CAMP_OPTION_REQUIRED`

### 5.3 评论点赞与取消点赞

```text
POST /api/predict/like
POST /api/predict/unlike
```

认证：需要登录。

参数：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `entityType` | string | 是 | `comment` |
| `entityId` | int64 | 是 | 评论或回复 ID |

规则：

- 点赞成功后增加点赞热度。
- 取消点赞不回滚历史热度。
- 点赞前会校验评论归属与锁边一致性。

## 6. 开撕台评论奖励接口

### 6.1 手动触发发放

```text
POST /api/predict/admin/comment-reward/run
```

认证：管理员。

参数：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `marketId` | int64 | 是 | 市场 ID |

规则：

- 仅 `SETTLED` 市场可发放。
- 必须在结算后 1 小时内执行。
- 奖金池固定为单市场真实池的 10%。
- 奖励按用户评论热度占比发放。

返回值（data）：

- `PredictCommentRewardLog`

### 6.2 重试失败批次

```text
POST /api/predict/admin/comment-reward/retry
```

认证：管理员。

参数：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `rewardLogId` | int64 | 是 | 奖励批次 ID |

规则：

- 只允许重试 `FAILED` 状态。
- `EXPIRED` 默认不允许重试。

### 6.3 奖励批次查询

```text
GET /api/predict/admin/comment-reward/logs
```

认证：管理员。

查询参数：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `market_id` | int64 | 否 | 市场 ID |
| `status` | string | 否 | 批次状态 |
| `page` | int | 否 | 页码 |
| `limit` | int | 否 | 每页条数 |

返回：

- `results`
- `page`

关键审计字段：

- 批次：`winnerOption`、`marketBetTotal`、`rewardPool`、`winnerTotalCommentHeat`、`remainder`、`status`、`reason`、`settledAt`、`deadlineAt`、`paidAt`
- 明细：`userId`、`amount`、`userCommentHeat`、`commentCount`、`firstCommentId`、`coinLogId`
