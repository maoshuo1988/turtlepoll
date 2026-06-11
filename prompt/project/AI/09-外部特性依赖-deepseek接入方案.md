# DeepSeek 接入方案

> 依据：`docs/api/deepseek.md`。
>
> 本文只做方案设计，不开发代码。

## 1. 总体架构

```text
前端
  -> /api/ai/chat
  -> /api/ai/presence
  -> /api/ai/messages

后端
  -> AIChatController
  -> AIChatService
  -> AIMemoryService
  -> AIRiskService
  -> DeepSeekClient

外部
  -> DeepSeek Chat Completions
```

关键原则：

- DeepSeek 只在服务端调用。
- 主动推送不实时调用 DeepSeek。
- 体力不足、敏感兜底、配置关闭时不调用 DeepSeek。
- 预测、下注、结算仍由业务规则决定，LLM 只做辅助表达。

## 2. DeepSeek 配置

建议配置：

```yaml
deepseek:
  enabled: false
  baseURL: "https://api.deepseek.com"
  apiKey: "" # 生产环境优先读取 DEEPSEEK_API_KEY
  defaultModel: "deepseek-v4-flash"
  reasoningModel: "deepseek-v4-pro"
  timeoutSeconds: 120
  maxRetries: 2

aiChat:
  enabled: false
  defaultStaminaCost: 1
  maxInputChars: 1000
  maxHistoryMessages: 12
  dailyUserMessageLimit: 100
  idlePushCooldownMinutes: 60
  idlePushDailyLimit: 2
  idleTriggerMinutes: 10
```

环境变量：

```bash
export DEEPSEEK_API_KEY="sk-..."
export DEEPSEEK_BASE_URL="https://api.deepseek.com"
export DEEPSEEK_DEFAULT_MODEL="deepseek-v4-flash"
```

## 3. 模型策略

| 用途 | 模型 | thinking | 说明 |
| --- | --- | --- | --- |
| 普通聊天 | `deepseek-v4-flash` | disabled | 默认模型，低成本。 |
| 记忆问答 | `deepseek-v4-flash` | disabled | 事实由后端提供。 |
| 简单预测分析 | `deepseek-v4-flash` | disabled | 简短分析即可。 |
| 复杂预测分析 | `deepseek-v4-pro` | enabled | 使用 `reasoning_effort=high`。 |
| 异步模板补池 | `deepseek-v4-flash` | disabled | 后续可选，本期不做。 |

不建议新接入使用：

- `deepseek-chat`
- `deepseek-reasoner`

## 4. 对话调用流程

```text
用户发送消息
  -> 登录校验
  -> AI 功能开关校验
  -> 输入长度校验
  -> 敏感/辱骂初筛
  -> 体力校验
  -> 构建 Prompt
  -> 选择模型
  -> 调用 DeepSeek
  -> 保存消息和 usage
  -> 扣体力
  -> 返回小龟回复
```

体力处理建议：

- DeepSeek 调用失败不扣体力。
- 如果实现上先扣体力，需要失败回滚或补偿。
- 用户消息和 AI 回复建议同事务保存。

## 5. Prompt 组装

### 5.1 普通聊天

包含：

- 基础人格 Prompt。
- 最近 `N` 轮对话。
- 用户当前输入。

### 5.2 记忆问答

包含：

- 基础人格 Prompt。
- 记忆问答约束。
- 后端读取的 `memory_facts`。
- 用户当前问题。

要求：

- 只能基于事实回答。
- 缺少事实时兜底。
- 不提数据库、Flag、系统。

### 5.3 预测分析

包含：

- 基础人格 Prompt。
- 预测分析约束。
- 站内 `PredictMarket / PredictContext` 上下文。
- 用户当前问题。

要求：

- 不承诺稳赚。
- 不鼓励重仓。
- 数据不足时说明不足。
- LLM 输出不参与结算。

## 6. 稳定性策略

| 场景 | 策略 |
| --- | --- |
| 普通请求超时 | 建议 `60s`。 |
| 思考模式超时 | 建议 `120s-300s`。 |
| 429 / 5xx | 指数退避，最多重试 `2` 次。 |
| 400 / 401 / 402 / 422 | 不重试，记录错误。 |
| DeepSeek 不可用 | 返回降级文案，不扣体力。 |
| 模板池为空 | 不推送主动消息。 |

降级文案示例：

```text
小龟刚才走神了，等下再试试。
```

## 7. 成本控制

1. 主动推送不实时调用 DeepSeek。
2. 普通聊天优先使用 `deepseek-v4-flash`。
3. 复杂预测分析才使用 `deepseek-v4-pro`。
4. 限制输入长度，例如 `1000` 字。
5. 对话历史最多带最近 `12` 条。
6. 用户每日主动聊天次数限制。
7. 记录 `prompt_tokens`、`completion_tokens`、`total_tokens`。

## 8. 安全策略

1. API Key 只保存在服务端配置或环境变量。
2. 日志禁止打印 `Authorization` 和 API Key。
3. 不向前端返回 `reasoning_content`。
4. 不保存 DeepSeek 的 `reasoning_content`。
5. 用户输入进入 LLM 前做长度和敏感初筛。
6. 管理后台展示 AI 配置时必须脱敏。

## 9. 后续能力

后续可以接入：

- 流式输出。
- JSON Output。
- Tool Calls。
- 异步模板补池。

Tool Calls 仅允许访问服务端白名单函数，不能让模型直接操作下注、结算、派奖。
