# DeepSeek API 接入说明

> 整理日期：2026-05-09。DeepSeek 模型、价格和兼容字段变化较快，上线前请以官方文档为准再核一次。

## 1. 官方入口

- 中文文档：https://api-docs.deepseek.com/zh-cn/
- API Key 控制台：https://platform.deepseek.com/api_keys
- OpenAI 兼容 Base URL：`https://api.deepseek.com`
- Anthropic 兼容 Base URL：`https://api.deepseek.com/anthropic`

DeepSeek API 当前兼容 OpenAI Chat Completions 和 Anthropic Messages 形式。项目服务端优先建议使用 OpenAI 兼容接口，因为 Go 侧可以用普通 HTTP 客户端或 OpenAI 兼容 SDK 快速接入。

## 2. 模型选择

| 模型 | 建议用途 | 说明 |
| --- | --- | --- |
| `deepseek-v4-flash` | 默认首选、成本敏感、高频问答、分类、摘要、轻量 AI 对话 | 支持非思考和思考模式，适合先作为项目默认模型。 |
| `deepseek-v4-pro` | 复杂推理、代码、长上下文分析、Agent / 工具调用 | 质量更强，成本更高，建议只给高价值任务或后台分析使用。 |
| `deepseek-chat` | 不建议新接入使用 | 官方标注将于 2026-07-24 弃用；兼容映射到 `deepseek-v4-flash` 的非思考模式。 |
| `deepseek-reasoner` | 不建议新接入使用 | 官方标注将于 2026-07-24 弃用；兼容映射到 `deepseek-v4-flash` 的思考模式。 |

推荐默认策略：

- 用户侧轻量聊天：`deepseek-v4-flash` + `thinking.disabled`
- 需要推理的预测分析：`deepseek-v4-pro` + `thinking.enabled` + `reasoning_effort=high`
- 后台批处理、摘要、标签提取：`deepseek-v4-flash` + `response_format={"type":"json_object"}`

## 3. 配置建议

不要把 API Key 写进仓库。建议生产配置文件只保留结构，密钥通过环境变量注入。

```yaml
deepseek:
  enabled: false
  baseURL: "https://api.deepseek.com"
  apiKey: "" # 生产环境建议从 DEEPSEEK_API_KEY 读取
  defaultModel: "deepseek-v4-flash"
  reasoningModel: "deepseek-v4-pro"
  timeoutSeconds: 120
  maxRetries: 2
```

建议环境变量：

```bash
export DEEPSEEK_API_KEY="sk-..."
export DEEPSEEK_BASE_URL="https://api.deepseek.com"
export DEEPSEEK_DEFAULT_MODEL="deepseek-v4-flash"
```

## 4. curl 验证

非流式调用：

```bash
curl -sS https://api.deepseek.com/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${DEEPSEEK_API_KEY}" \
  -d '{
    "model": "deepseek-v4-flash",
    "messages": [
      {"role": "system", "content": "你是龟投论坛的 AI 助手，回答要简洁。"},
      {"role": "user", "content": "用一句话解释预测市场是什么。"}
    ],
    "thinking": {"type": "disabled"},
    "stream": false
  }'
```

思考模式调用：

```bash
curl -sS https://api.deepseek.com/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${DEEPSEEK_API_KEY}" \
  -d '{
    "model": "deepseek-v4-pro",
    "messages": [
      {"role": "user", "content": "分析一场预测事件结算前需要检查哪些风险点。"}
    ],
    "thinking": {"type": "enabled"},
    "reasoning_effort": "high",
    "stream": false
  }'
```

流式调用只需要加上：

```json
{"stream": true}
```

服务端读取 SSE 时要支持 `data: ...` 增量片段，并处理 keep-alive 注释或空行。

## 5. Go HTTP 调用示例

项目当前没有 DeepSeek 客户端，最小实现可以先用标准库 HTTP。

```go
package deepseek

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model           string         `json:"model"`
	Messages        []Message      `json:"messages"`
	Thinking        map[string]any `json:"thinking,omitempty"`
	ReasoningEffort string         `json:"reasoning_effort,omitempty"`
	Stream          bool           `json:"stream"`
	ResponseFormat  map[string]any `json:"response_format,omitempty"`
}

type ChatResponse struct {
	Choices []struct {
		Message struct {
			ReasoningContent string `json:"reasoning_content,omitempty"`
			Content          string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage map[string]any `json:"usage,omitempty"`
}

func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{Timeout: 120 * time.Second}
	}
	if c.BaseURL == "" {
		c.BaseURL = "https://api.deepseek.com"
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("deepseek api status: %s", resp.Status)
	}

	var out ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
```

调用示例：

```go
resp, err := client.Chat(ctx, deepseek.ChatRequest{
	Model: "deepseek-v4-flash",
	Messages: []deepseek.Message{
		{Role: "system", Content: "你是龟投论坛的 AI 助手。"},
		{Role: "user", Content: "帮我生成一条预测事件摘要。"},
	},
	Thinking: map[string]any{"type": "disabled"},
	Stream:   false,
})
```

## 6. JSON 输出

当业务需要稳定结构化结果，例如预测事件摘要、标签提取、风控结论，使用 JSON Output：

```json
{
  "model": "deepseek-v4-flash",
  "messages": [
    {
      "role": "system",
      "content": "请只输出 json。格式：{\"title\":\"\",\"riskLevel\":\"LOW|MEDIUM|HIGH\",\"reasons\":[]}"
    },
    {
      "role": "user",
      "content": "分析这个预测事件：世界杯决赛 A 队 vs B 队。"
    }
  ],
  "response_format": {"type": "json_object"},
  "thinking": {"type": "disabled"}
}
```

注意：

- prompt 中必须明确出现 `json` 字样，并给出目标 JSON 示例。
- `max_tokens` 要给足，否则 JSON 可能被截断。
- 后端仍要做 `json.Valid` / schema 校验，不要直接信任模型输出。

## 7. Tool Calls

DeepSeek 支持 OpenAI 风格 `tools`。模型只会返回要调用的工具和参数，真正的函数执行必须由业务服务端完成。

适合本项目的工具场景：

- 查询预测市场详情
- 查询用户龟币余额
- 查询赛事状态或 Polymarket 只读数据
- 生成结算前检查清单

工具调用闭环：

1. 用户提问。
2. 请求中传入 `tools`。
3. 模型返回 `tool_calls`。
4. 服务端执行对应内部函数。
5. 把工具结果作为 `role=tool` 消息传回模型。
6. 模型生成最终回答。

严格工具模式目前属于 Beta，需要使用 Beta Base URL，并且每个 function 设置 `strict: true`。生产接入前建议先不用 strict，等工具 schema 稳定后再切。

## 8. 思考模式

DeepSeek V4 支持思考模式：

```json
{
  "thinking": {"type": "enabled"},
  "reasoning_effort": "high"
}
```

关闭思考模式：

```json
{
  "thinking": {"type": "disabled"}
}
```

建议：

- 普通聊天、摘要、分类默认关闭思考，降低延迟和成本。
- 复杂预测分析、风控判断、代码生成开启思考。
- `reasoning_effort` 优先使用 `high`；极复杂任务再评估 `max`。
- 如果响应里有 `reasoning_content`，前端默认不要展示；除非产品明确要“推理过程”视图。
- 多轮对话回放时谨慎处理 `reasoning_content`。如果后续请求报 400，优先检查是否把不该回传的思考字段带回了 `messages`。

## 9. 速率、超时和重试

官方说明 DeepSeek 会根据服务器负载动态限制并发，达到并发限制会返回 HTTP 429。请求发出后，如果服务端还在排队，连接可能保持一段时间；非流式请求可能持续返回空行，流式请求可能返回 SSE keep-alive 注释。

建议服务端策略：

- HTTP timeout：普通请求 `60s`，思考模式或长上下文 `120s-300s`。
- 429 / 5xx：指数退避重试，最多 `2-3` 次。
- 400 / 401 / 402 / 422：不要重试，直接记录错误原因。
- 记录 request id、模型、耗时、token usage、HTTP status；不要记录 API Key。
- 用户侧接口要做限流，避免一个用户触发大量 LLM 成本。

## 10. 项目落地建议

后端建议新增：

```text
internal/services/ai/
  deepseek_client.go
  deepseek_types.go
  ai_service.go
```

配置结构建议新增：

```go
type DeepSeekConfig struct {
	Enabled        bool   `yaml:"enabled"`
	BaseURL        string `yaml:"baseURL"`
	APIKey         string `yaml:"apiKey"`
	DefaultModel   string `yaml:"defaultModel"`
	ReasoningModel string `yaml:"reasoningModel"`
	TimeoutSeconds int    `yaml:"timeoutSeconds"`
	MaxRetries     int    `yaml:"maxRetries"`
}
```

优先接入顺序：

1. 后台 curl 验证 API Key、模型和余额。
2. 增加服务端 DeepSeek Client，先支持非流式。
3. 给用户侧 AI 对话或预测摘要接一个最小接口。
4. 增加 token usage 记录和每日成本上限。
5. 再补流式输出、Tool Calls、JSON Schema 校验。

## 11. 安全注意事项

- API Key 只能放服务端，禁止下发给浏览器。
- 管理后台展示配置时必须脱敏。
- 日志不要打印 `Authorization` header。
- 用户输入进入 LLM 前要做长度限制和敏感内容过滤。
- LLM 输出不能直接作为结算依据，只能作为辅助分析；预测、下注、结算仍以业务规则和链路数据为准。
