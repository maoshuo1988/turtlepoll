# AI 接口访问测试（curl）

本文档基于 `docs/api/ai.md` 和 `internal/controllers/api/ai_controller.go`，提供一套可复制的 curl 命令，用于对 AI 聊天、AI 体力和 AI 主动推送接口做手工冒烟/回归。

> 说明：本文档只负责“怎么打接口”。AI 聊天开关、DeepSeek 模型与推送规则口径以 `docs/api/ai.md`、`prompt/project/AI/` 下设计文档和当前配置为准。

## 0. 测试前准备

### 0.1 环境变量

```bash
export BASE_URL="http://127.0.0.1:8082"
export USER_TOKEN="<YOUR_USER_TOKEN>"
```

### 0.2 通用 Header

```bash
export USER_AUTH_HEADER="Authorization: Bearer ${USER_TOKEN}"
```

> 如果你的项目鉴权 header 不是 `Authorization: Bearer ...`，把下面示例里的 header 名改掉即可。

### 0.3 配置检查

AI 聊天依赖服务端配置同时开启：

```yaml
deepseek:
  enabled: true

aiChat:
  enabled: true
```

生产或联调环境还需要有效的 DeepSeek Key：

```bash
export DEEPSEEK_API_KEY="sk-..."
```

### 0.4 可选：jq

示例里很多命令带了 `| jq` 方便查看；未安装 jq 时可以先去掉。

---

## 1) 查询 AI 体力

### 1.1 GET /api/ai/stamina

```bash
curl -sS "${BASE_URL}/api/ai/stamina" \
  -H "${USER_AUTH_HEADER}" | jq
```

输出检查点：

- `stamina/maxStamina` 有值。
- `dailyUsedCount/dailyLimit/dailyRemaining` 有值。
- `nextRecoverAt=0` 表示当前 AI 体力已满。
- 未登录时应返回 401 或等价未登录错误。

---

## 2) 用户主动聊天

### 2.1 POST /api/ai/chat（基础聊天）

```bash
curl -sS "${BASE_URL}/api/ai/chat" \
  -H "${USER_AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "今天适合参加哪个预测话题？",
    "scene": "chat"
  }' | jq
```

输出检查点：

- `message.role=="assistant"`。
- `message.content` 非空。
- `userMessage.role=="user"`。
- `staminaCost` 通常为 `aiChat.defaultStaminaCost`。
- 成功调用 DeepSeek 时，`degraded==false`，并扣减 AI 体力。
- DeepSeek 超时或报错时，`degraded==true`，返回兜底文案，不扣体力。

### 2.2 POST /api/ai/chat（带业务上下文）

```bash
curl -sS "${BASE_URL}/api/ai/chat" \
  -H "${USER_AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "帮我看看这个预测市场下注前要注意什么。",
    "scene": "chat",
    "contextType": "predict_market",
    "contextId": 123
  }' | jq
```

输出检查点：

- `message.contextType=="predict_market"`。
- `message.contextId==123`。

### 2.3 常见错误：content 为空

```bash
curl -sS "${BASE_URL}/api/ai/chat" \
  -H "${USER_AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d '{"content":"   ","scene":"chat"}' | jq
```

期望错误：`content is required`。

### 2.4 常见错误：content 超长

> 默认最大长度来自 `aiChat.maxInputChars`，示例配置通常为 500。

```bash
LONG_CONTENT="$(printf '小龟%.0s' {1..300})"

curl -sS "${BASE_URL}/api/ai/chat" \
  -H "${USER_AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d "{\"content\":\"${LONG_CONTENT}\",\"scene\":\"chat\"}" | jq
```

期望错误：`content exceeds max length 500` 或当前配置对应的最大长度。

### 2.5 常见错误：体力不足

当 `stamina` 不足以支付 `aiChat.defaultStaminaCost` 时：

```bash
curl -sS "${BASE_URL}/api/ai/chat" \
  -H "${USER_AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "再聊一句。",
    "scene": "chat"
  }' | jq
```

输出检查点：

- 返回 `data.insufficientPrompt`，可直接给前端展示。
- 不调用 DeepSeek。
- 不保存成功对话。

### 2.6 常见错误：每日次数上限

当 `dailyUsedCount >= dailyLimit` 时：

```bash
curl -sS "${BASE_URL}/api/ai/chat" \
  -H "${USER_AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d '{"content":"今天还能聊吗？","scene":"chat"}' | jq
```

期望错误：`daily ai chat limit reached`。

---

## 3) 苹果恢复 AI 体力

### 3.1 POST /api/ai/stamina/apple（JSON）

```bash
curl -sS "${BASE_URL}/api/ai/stamina/apple" \
  -H "${USER_AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d '{"count":1}' | jq
```

输出检查点：

- `requestedCount==1`。
- `recoveredCount` 为实际恢复点数。
- `coinCost == recoveredCount * appleCoinCost`。
- `balanceAfter` 反映扣费后的龟币余额。

### 3.2 POST /api/ai/stamina/apple（表单）

```bash
curl -sS "${BASE_URL}/api/ai/stamina/apple" \
  -H "${USER_AUTH_HEADER}" \
  -d "count=1" | jq
```

### 3.3 常见错误：count 非法

```bash
curl -sS "${BASE_URL}/api/ai/stamina/apple" \
  -H "${USER_AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d '{"count":0}' | jq
```

期望错误：`count must be positive`。

### 3.4 常见错误：体力已满

```bash
curl -sS "${BASE_URL}/api/ai/stamina/apple" \
  -H "${USER_AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d '{"count":1}' | jq
```

期望错误：`AI_STAMINA_FULL`。

### 3.5 常见错误：龟币不足

```bash
curl -sS "${BASE_URL}/api/ai/stamina/apple" \
  -H "${USER_AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d '{"count":99}' | jq
```

期望错误：`insufficient balance`。

---

## 4) AI presence 上报

### 4.1 POST /api/ai/presence（JSON）

```bash
curl -sS "${BASE_URL}/api/ai/presence" \
  -H "${USER_AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d '{
    "page": "predict_market",
    "active": true
  }' | jq
```

输出检查点：

- `userId` 为当前用户。
- `page=="predict_market"`。
- `active==true`。
- `lastSeenAt` 为当前时间附近的秒级时间戳。

### 4.2 POST /api/ai/presence（表单）

```bash
curl -sS "${BASE_URL}/api/ai/presence" \
  -H "${USER_AUTH_HEADER}" \
  -d "page=predict_market&active=false" | jq
```

输出检查点：

- `active==false`。

---

## 5) 拉取未读 AI 推送

### 5.1 GET /api/ai/pushes/unread

```bash
curl -sS "${BASE_URL}/api/ai/pushes/unread" \
  -H "${USER_AUTH_HEADER}" | jq
```

### 5.2 GET /api/ai/pushes/unread?limit=5

```bash
curl -sS "${BASE_URL}/api/ai/pushes/unread?limit=5" \
  -H "${USER_AUTH_HEADER}" | jq
```

输出检查点：

- `results` 为数组。
- 推送对象包含 `id/scene/content/contextType/contextId/createTime`。
- `limit<=0` 或 `limit>100` 时，服务端会回退默认 `20`。

---

## 6) 标记 AI 推送已读

### 6.1 准备：提取未读推送 ID

```bash
AI_PUSH_IDS="$(
  curl -sS "${BASE_URL}/api/ai/pushes/unread?limit=5" \
    -H "${USER_AUTH_HEADER}" \
  | jq -r '[.data.results[].id] | @json'
)"

echo "${AI_PUSH_IDS}"
```

> 如果没有安装 jq，可以手动从 `GET /api/ai/pushes/unread` 返回里复制 ID，例如 `[20001,20002]`。

### 6.2 POST /api/ai/pushes/read

```bash
curl -sS "${BASE_URL}/api/ai/pushes/read" \
  -H "${USER_AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d "{\"ids\":${AI_PUSH_IDS:-[]}}" | jq
```

输出检查点：

- `updated` 表示本次真实从未读改为已读的数量。
- 重复提交已读 ID 不应报错，`updated` 可能为 0。

### 6.3 空 ID

```bash
curl -sS "${BASE_URL}/api/ai/pushes/read" \
  -H "${USER_AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d '{"ids":[]}' | jq
```

输出检查点：

- `updated==0`。

---

## 7) AI 推送 SSE

### 7.1 GET /api/ai/pushes/stream

> 这是长连接接口，命令会持续等待。验证完成后按 `Ctrl+C` 结束。

```bash
curl -N -sS "${BASE_URL}/api/ai/pushes/stream" \
  -H "${USER_AUTH_HEADER}"
```

输出检查点：

- 连接成功后立即看到 `: connected`。
- 服务端约每 30 秒发送 `: ping`。
- 有 AI 推送时，收到类似：

```text
id: 20001
event: ai_push
data: {"id":20001,"scene":"win","content":"120 入账。低调。","contextType":"predict_market","contextId":123,"createTime":1760000000}
```

### 7.2 带 Last-Event-ID 补拉

```bash
curl -N -sS "${BASE_URL}/api/ai/pushes/stream" \
  -H "${USER_AUTH_HEADER}" \
  -H "Last-Event-ID: 20000"
```

输出检查点：

- 仍未读且 `id > Last-Event-ID` 的推送会在连接后补发。

---

## 最小回归序列（建议）

1. `GET /api/ai/stamina`
2. `POST /api/ai/presence`
3. `POST /api/ai/chat`
4. `GET /api/ai/stamina`，确认聊天后体力与每日次数变化
5. 如体力不足，`POST /api/ai/stamina/apple`
6. `GET /api/ai/pushes/unread`
7. `POST /api/ai/pushes/read`
8. `curl -N GET /api/ai/pushes/stream`，观察连接、心跳和推送事件

---

## 常见问题排查

- 401：token 不存在/过期，或 header 名不匹配。
- `ai chat is disabled`：检查 `deepseek.enabled` 和 `aiChat.enabled`。
- `content is required`：`content` 为空或全空格。
- `content exceeds max length ...`：超过 `aiChat.maxInputChars`。
- `daily ai chat limit reached`：当前用户当天主动聊天次数已达 `aiChat.dailyUserMessageLimit`。
- `AI_STAMINA_NOT_ENOUGH`：AI 体力不足，接口会附带 `insufficientPrompt`。
- `AI_STAMINA_FULL`：当前 AI 体力已满，不能继续用苹果恢复。
- `insufficient balance`：用户龟币余额不足以支付苹果恢复体力。
- SSE 一直没有 `event: ai_push`：先确认是否存在未读推送，或通过结算/闲置推送链路制造一条 AI 推送。
