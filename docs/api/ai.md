# AI 聊天

## 模块说明

- 基础路径：`/api/ai`
- 认证：需要登录。
- DeepSeek 与 AI 聊天都需要在配置中开启：`deepseek.enabled=true`、`aiChat.enabled=true`。
- 用户主动聊天会消耗 `aiChat.defaultStaminaCost` 点 AI 体力。
- DeepSeek 调用失败时返回降级文案，不扣体力，也不保存成功对话。
- 体力不足时不调用 DeepSeek，可通过自然恢复或苹果恢复体力。

## 配置示例

```yaml
deepseek:
  enabled: true
  baseURL: "https://api.deepseek.com"
  apiKey: ""
  defaultModel: "deepseek-v4-flash"
  reasoningModel: "deepseek-v4-pro"
  timeoutSeconds: 120
  maxRetries: 2

aiChat:
  enabled: true
  defaultStaminaCost: 1
  defaultMaxStamina: 5
  staminaRecoverMinutes: 60
  appleCoinCost: 5
  maxInputChars: 500
  maxHistoryMessages: 8
  dailyUserMessageLimit: 50
  idlePushCooldownMinutes: 120
  idlePushDailyLimit: 2
  idleTriggerMinutes: 10
```

生产环境建议通过环境变量注入密钥：

```bash
export DEEPSEEK_API_KEY="sk-..."
export DEEPSEEK_BASE_URL="https://api.deepseek.com"
export DEEPSEEK_DEFAULT_MODEL="deepseek-v4-flash"
```

## 1. 用户聊天

- 接口：`POST /api/ai/chat`
- 功能：用户主动和小龟 AI 聊天。
- 请求格式：JSON。

### 请求体

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `content` | string | 是 | 用户输入，默认最多 `aiChat.maxInputChars` 个字符 |
| `scene` | string | 否 | 场景，默认 `chat` |
| `contextType` | string | 否 | 业务上下文类型，如 `predict_market` |
| `contextId` | int64 | 否 | 业务上下文 ID |

示例：

```json
{
  "content": "今天适合参加哪个预测话题？",
  "scene": "chat"
}
```

### 返回 data

```json
{
  "message": {
    "id": 12,
    "userId": 1,
    "role": "assistant",
    "scene": "chat",
    "content": "可以先看你熟悉的赛事，小龟建议别重仓，稳稳玩更开心。",
    "model": "deepseek-v4-flash",
    "promptTokens": 120,
    "completionTokens": 32,
    "totalTokens": 152,
    "staminaCost": 0,
    "contextType": "",
    "contextId": 0,
    "requestId": "chatcmpl-xxx",
    "createTime": 1760000000
  },
  "userMessage": {},
  "staminaLeft": 4,
  "maxStamina": 5,
  "nextRecoverAt": 1760003600,
  "staminaCost": 1,
  "promptTokens": 120,
  "completionTokens": 32,
  "totalTokens": 152,
  "degraded": false,
  "dailyRemaining": 49,
  "dailyMessageLimit": 50
}
```

### 体力不足返回

当前接口为了方便前端直接展示兜底文案，体力不足时仍返回 `data`：

```json
{
  "staminaLeft": 0,
  "maxStamina": 5,
  "nextRecoverAt": 1760003600,
  "staminaCost": 1,
  "degraded": false,
  "dailyRemaining": 49,
  "dailyMessageLimit": 50,
  "insufficientPrompt": "小龟睡着啦~ 喂它一颗苹果(5 龟币)就能继续聊咯"
}
```

### 降级返回

DeepSeek 超时或报错时：

```json
{
  "message": {
    "role": "assistant",
    "content": "小龟现在有点困，等会儿再来找我聊吧。",
    "staminaCost": 0
  },
  "degraded": true,
  "staminaLeft": 5,
  "maxStamina": 5,
  "staminaCost": 1
}
```

### 可能错误

- `ai chat is disabled`
- `content is required`
- `content exceeds max length 500`
- `daily ai chat limit reached`
- `AI_STAMINA_NOT_ENOUGH`

## 2. 查询 AI 体力

- 接口：`GET /api/ai/stamina`
- 功能：查询当前 AI 体力、上限、下次恢复时间和每日主动聊天次数。

### 返回 data

```json
{
  "userId": 1,
  "stamina": 3,
  "maxStamina": 5,
  "nextRecoverAt": 1760003600,
  "dailyUsedCount": 2,
  "dailyLimit": 50,
  "recoverMinutes": 60,
  "appleCoinCost": 5,
  "lastRecoverAt": 1760000000,
  "dailyRemaining": 48
}
```

说明：

- 查询前会先做自然恢复懒结算。
- `nextRecoverAt=0` 表示当前体力已满。

## 3. 苹果恢复体力

- 接口：`POST /api/ai/stamina/apple`
- 功能：消耗龟币恢复 AI 体力。
- 请求格式：JSON 或表单。

### 请求体

```json
{
  "count": 1
}
```

### 返回 data

```json
{
  "userId": 1,
  "stamina": 4,
  "maxStamina": 5,
  "nextRecoverAt": 1760003600,
  "dailyUsedCount": 2,
  "dailyLimit": 50,
  "recoverMinutes": 60,
  "appleCoinCost": 5,
  "lastRecoverAt": 1760000000,
  "dailyRemaining": 48,
  "requestedCount": 1,
  "recoveredCount": 1,
  "coinCost": 5,
  "balanceAfter": 95
}
```

### 可能错误

- `count must be positive`
- `AI_STAMINA_FULL`
- `insufficient balance`

## 4. 拉取未读 AI 推送

- 接口：`GET /api/ai/pushes/unread`
- 功能：拉取离线、断线或尚未展示的 AI 主动推送。

### 查询参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `limit` | int | 否 | 默认 20，最大 100 |

### 返回 data

```json
{
  "results": [
    {
      "id": 20001,
      "scene": "win_streak",
      "content": "3 连了。小龟先帮你记一笔。",
      "contextType": "predict_market",
      "contextId": 123,
      "createTime": 1760000000
    }
  ]
}
```

## 5. 标记 AI 推送已读

- 接口：`POST /api/ai/pushes/read`
- 功能：将当前用户自己的未读 AI 推送标记为已读。
- 请求格式：JSON。

### 请求体

```json
{
  "ids": [20001, 20002]
}
```

### 返回 data

```json
{
  "updated": 2
}
```

说明：`updated` 表示本次真实从未读改为已读的数量；重复提交已读 ID 不视为错误。

## 6. AI presence 上报

- 接口：`POST /api/ai/presence`
- 功能：上报用户在线、页面和活跃状态，用于闲置推送判断。
- 请求格式：JSON 或表单。

### 请求体

```json
{
  "page": "predict_market",
  "active": true
}
```

### 返回 data

```json
{
  "userId": 1,
  "page": "predict_market",
  "active": true,
  "lastSeenAt": 1760000000
}
```

## 7. AI 推送 SSE

- 接口：`GET /api/ai/pushes/stream`
- 功能：建立 SSE 长连接，在线接收 `ai_push` 事件。

### 事件格式

```text
id: 20001
event: ai_push
data: {"id":20001,"scene":"win","content":"120 入账。低调。","contextType":"predict_market","contextId":123,"createTime":1760000000}
```

说明：

- SSE 需要登录态。
- 服务端每 30 秒发送 `: ping` 心跳。
- 浏览器重连携带 `Last-Event-ID` 时，服务端会补发仍未读且 `id > Last-Event-ID` 的推送。
- SSE 投递失败不影响消息落库；前端仍应通过 `GET /api/ai/pushes/unread` 兜底补拉。
