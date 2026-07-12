# 主站消息通知（Message Notify）

> 路由前缀：`/api/message-notify`
>
> 说明：该模块提供 PC 主站消息中心接口，用于展示暗盘、开撕台、线报、系统、奖励、地下钱庄等业务推送消息。
>
> 认证：需要登录。未登录时返回项目统一未登录错误。

---

## 目录

- [消息列表](#消息列表)
- [未读数量](#未读数量)
- [消息详情](#消息详情)
- [标记单条已读](#标记单条已读)
- [业务分类](#业务分类)
- [消息状态](#消息状态)

---

## 消息列表

`GET /api/message-notify/list`

### 请求参数

| 参数名 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| businessCode | string | 否 | "" | 业务分类，见 [业务分类](#业务分类) |
| status | int | 否 | 不过滤 | 消息状态：`0` 未读，`1` 已读 |
| cursor | int64 | 否 | 0 | 游标，首次请求传 0；续页传上次返回的 `cursor` |
| limit | int | 否 | 20 | 每页条数，默认 20，最大 100 |

### 响应示例

```json
{
  "success": true,
  "data": {
    "results": [
      {
        "id": 101,
        "businessCode": "dark_market",
        "templateCode": "predict_settle_win",
        "templateId": 1,
        "userId": 10001,
        "subject": "你参与的 BTC 周末方向 已结算",
        "body": "预测命中，奖励 180 龟币已进入余额。",
        "detailUrl": "/predict/markets/20001",
        "status": 0,
        "templateParams": "{\"marketId\":\"20001\",\"marketTitle\":\"BTC 周末方向\",\"payout\":\"180\"}",
        "extraData": "{\"betId\":90001,\"marketId\":20001,\"settleResult\":\"WIN\",\"payout\":180}",
        "bizId": "90001",
        "idempotencyKey": "predict_settle:90001:win",
        "createTime": 1783770000,
        "updateTime": 1783770000
      }
    ],
    "cursor": "101",
    "hasMore": false,
    "unreadCount": 3
  }
}
```

### 响应字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| results | array | 当前页消息列表 |
| cursor | string | 下一页游标；当 `hasMore=true` 时续页传回 |
| hasMore | bool | 是否还有下一页 |
| unreadCount | int64 | 当前用户未读消息总数 |

`results[]` 字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int64 | 消息记录 ID |
| businessCode | string | 业务分类 |
| templateCode | string | 模板编码，用于识别具体业务场景 |
| templateId | int64 | 模板 ID |
| userId | int64 | 接收用户 ID |
| subject | string | 消息标题 |
| body | string | 消息正文 |
| detailUrl | string | 点击消息后的跳转地址 |
| status | int | 消息状态：`0` 未读，`1` 已读 |
| templateParams | string | 本次模板参数 JSON 快照 |
| extraData | string | 扩展业务数据 JSON |
| bizId | string | 关联业务对象 ID，如 betId、dayName、orderNo |
| idempotencyKey | string | 幂等键 |
| createTime | int64 | 创建时间，Unix 秒 |
| updateTime | int64 | 更新时间，Unix 秒 |

### 说明

- 列表只返回当前登录用户自己的消息。
- 默认按 `id desc` 返回最新消息。
- 前端筛选 tab 可通过 `businessCode` 请求不同业务分类。

---

## 未读数量

`GET /api/message-notify/unread-count`

### 请求参数

无。

### 响应示例

```json
{
  "success": true,
  "data": {
    "totalUnread": 5,
    "businessUnread": {
      "dark_market": 2,
      "reward": 1,
      "system": 2
    }
  }
}
```

### 响应字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| totalUnread | int64 | 当前用户未读总数 |
| businessUnread | object | 按 `businessCode` 聚合的未读数 |

---

## 消息详情

`GET /api/message-notify/by/{id}`

### 路径参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | int64 | 是 | 消息记录 ID |

### 响应示例

```json
{
  "success": true,
  "data": {
    "id": 101,
    "businessCode": "reward",
    "templateCode": "daily_login_reward",
    "templateId": 3,
    "userId": 10001,
    "subject": "今日登录奖励已到账",
    "body": "领取 80 龟币，连续登录 3 天。",
    "detailUrl": "/rewards/daily",
    "status": 0,
    "templateParams": "{\"amount\":\"80\",\"loginStreak\":\"3\"}",
    "extraData": "{\"date\":\"2026-07-11\",\"balanceAfter\":1080}",
    "bizId": "20260711",
    "idempotencyKey": "daily_login_reward:10001:20260711",
    "createTime": 1783770000,
    "updateTime": 1783770000
  }
}
```

### 说明

- 只允许读取当前登录用户自己的消息。
- 详情接口不会自动标记已读。
- 前端需要“点击即已读”时，应先调用 [标记单条已读](#标记单条已读)，再跳转 `detailUrl`。

---

## 标记单条已读

`POST /api/message-notify/read`

### 请求格式

JSON 或表单。

### 请求体

```json
{
  "id": 101
}
```

### 请求参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | int64 | 是 | 消息记录 ID |

### 响应示例

```json
{
  "success": true,
  "data": {
    "updated": true,
    "record": {
      "id": 101,
      "businessCode": "dark_market",
      "templateCode": "predict_settle_win",
      "subject": "你参与的 BTC 周末方向 已结算",
      "body": "预测命中，奖励 180 龟币已进入余额。",
      "detailUrl": "/predict/markets/20001",
      "status": 1,
      "createTime": 1783770000,
      "updateTime": 1783770300
    }
  }
}
```

### 响应字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| updated | bool | 本次是否真实从未读更新为已读 |
| record | object | 更新后的消息记录 |

### 说明

- 已读消息重复提交不报错，`updated=false`。
- 消息不存在或不属于当前用户时返回失败，`message` 通常为 `MESSAGE_NOT_FOUND`。

---

## 业务分类

| businessCode | 中文名 | 说明 |
|--------------|--------|------|
| dark_market | 暗盘 | 预测、盘口、结算、开盘提醒 |
| tear_square | 开撕台 | 评论、回复、引用、观点互动 |
| intel | 线报 | 关注线报、来源补充、热点更新 |
| system | 系统 | 规则、维护、审核、仲裁、风控 |
| reward | 奖励 | 登录、任务、宠物、评论奖励 |
| underground_bank | 地下钱庄 | 黑市订单、交易、宠物购买 |

---

## 消息状态

| status | 名称 | 说明 |
|--------|------|------|
| 0 | 未读 | 默认状态 |
| 1 | 已读 | 用户点击或主动标记后状态 |

---

## 常见错误

| message | 说明 |
|---------|------|
| 未登录 | 当前请求未携带有效登录态 |
| USER_ID_REQUIRED | 用户 ID 非法 |
| MESSAGE_NOT_FOUND | 消息不存在，或不属于当前登录用户 |

