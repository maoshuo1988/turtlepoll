# AI 聊天

## 模块说明

- 基础路径：`/api/ai`
- 认证：需要登录。
- DeepSeek 与 AI 聊天都需要在配置中开启：`deepseek.enabled=true`、`aiChat.enabled=true`。
- 用户主动聊天会消耗 `aiChat.defaultStaminaCost` 龟币，当前返回字段沿用产品口径叫 `staminaCost`。
- DeepSeek 调用失败时返回降级文案，不扣龟币，也不保存成功对话。

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
  "balanceAfter": 99,
  "staminaCost": 1,
  "promptTokens": 120,
  "completionTokens": 32,
  "totalTokens": 152,
  "degraded": false,
  "dailyRemaining": 49,
  "dailyMessageLimit": 50
}
```

### 余额不足返回

当前接口为了方便前端直接展示兜底文案，余额不足时仍返回 `data`：

```json
{
  "balanceAfter": 0,
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
  "balanceAfter": 100,
  "staminaCost": 1
}
```

### 可能错误

- `ai chat is disabled`
- `content is required`
- `content exceeds max length 500`
- `daily ai chat limit reached`
