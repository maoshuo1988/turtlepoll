# AI 系统 YAML 配置说明

> AI 相关配置统一写在项目根目录的 `bbs-go.yaml` 中，由 `internal/pkg/config/config.go` 中的 `Config` 结构体解析。
> 不存在独立的 `ai_config.yaml`、`ai_templates.yaml` 等文件。

## 1. 配置来源

- 配置文件：`bbs-go.yaml`（项目根目录，示例见 `bbs-go.example.yaml`）
- 读取时机：服务启动时一次性加载，运行期通过 `POST /api/admin/ai/config/reload` 热重载
- API Key 必须通过环境变量 `DEEPSEEK_API_KEY` 注入，不得明文写入版本仓库

---

## 2. bbs-go.yaml 中 AI 相关配置

### 2.1 完整示例（与代码 struct 字段一一对应）

```yaml
# DeepSeek API（服务端调用；生产环境建议通过 DEEPSEEK_API_KEY 注入密钥）
deepseek:
  enabled: true
  baseURL: "https://api.deepseek.com"
  apiKey: ""                  # 明文留空，生产用 DEEPSEEK_API_KEY 环境变量
  defaultModel: "deepseek-v4-flash"
  reasoningModel: "deepseek-v4-pro"
  timeoutSeconds: 120
  maxRetries: 2

# AI 聊天
aiChat:
  enabled: true
  defaultStaminaCost: 1        # 每次聊天消耗体力
  defaultMaxStamina: 5         # 体力上限（当前生产值）
  staminaRecoverMinutes: 60    # 每 60 分钟自然恢复 1 点
  appleCoinCost: 5             # 1 个苹果 = 5 龟币
  maxInputChars: 500           # 用户输入字符上限
  maxHistoryMessages: 8        # 对话上下文最大条数
  dailyUserMessageLimit: 50    # 每自然日主动聊天上限
  idlePushCooldownMinutes: 120 # 两次闲置推送最短间隔（分钟）
  idlePushDailyLimit: 2        # 每日闲置推送上限
  idleTriggerMinutes: 10       # 闲置判定阈值（分钟）
```

### 2.2 参数说明

#### DeepSeek 结构体字段（`internal/pkg/config/config.go`）

| yaml 键 | Go 字段 | 类型 | 当前值 | 说明 |
| --- | --- | --- | --- | --- |
| `deepseek.enabled` | `Enabled` | bool | true | 是否启用 DeepSeek 调用 |
| `deepseek.baseURL` | `BaseURL` | string | `https://api.deepseek.com` | API 地址 |
| `deepseek.apiKey` | `APIKey` | string | — | 优先读 `DEEPSEEK_API_KEY` 环境变量 |
| `deepseek.defaultModel` | `DefaultModel` | string | `deepseek-v4-flash` | 普通对话模型 |
| `deepseek.reasoningModel` | `ReasoningModel` | string | `deepseek-v4-pro` | 复杂推理模型 |
| `deepseek.timeoutSeconds` | `TimeoutSeconds` | int | 120 | HTTP 超时（秒） |
| `deepseek.maxRetries` | `MaxRetries` | int | 2 | 最大重试次数 |

#### AIChat 结构体字段

| yaml 键 | Go 字段 | 类型 | 当前值 | 说明 |
| --- | --- | --- | --- | --- |
| `aiChat.enabled` | `Enabled` | bool | true | 功能总开关 |
| `aiChat.defaultStaminaCost` | `DefaultStaminaCost` | int | 1 | 每次聊天消耗体力点数 |
| `aiChat.defaultMaxStamina` | `DefaultMaxStamina` | int | 5 | 体力上限 |
| `aiChat.staminaRecoverMinutes` | `StaminaRecoverMinutes` | int | 60 | 自然恢复周期（分钟/点） |
| `aiChat.appleCoinCost` | `AppleCoinCost` | int | 5 | 苹果价格（龟币/个） |
| `aiChat.maxInputChars` | `MaxInputChars` | int | 500 | 用户输入字符上限 |
| `aiChat.maxHistoryMessages` | `MaxHistoryMessages` | int | 8 | 上下文历史条数 |
| `aiChat.dailyUserMessageLimit` | `DailyUserMessageLimit` | int | 50 | 每自然日主动聊天次数上限 |
| `aiChat.idlePushCooldownMinutes` | `IdlePushCooldownMinutes` | int | 120 | 两次闲置推送最短间隔（分钟） |
| `aiChat.idlePushDailyLimit` | `IdlePushDailyLimit` | int | 2 | 每日闲置推送上限 |
| `aiChat.idleTriggerMinutes` | `IdleTriggerMinutes` | int | 10 | 闲置判定阈值（分钟） |

> **不存在的字段**：`aiPush`、`featureFlags`、`operations`、`prompt`、`aiPush.settlement`、`maxAppleUsagePerDay`、`staminaRecoverPerUnit`、`petStaminaReductionEnabled` 均不在代码结构体中，不要写入 bbs-go.yaml。

---

## 3. 推送模板（数据库表，非 YAML）

推送模板存储在数据库表 `ai_message_template` 中，通过管理端接口维护，**不**通过 YAML 文件配置。

原 `ai_templates.yaml` 章节内容已废弃，下方仅保留字段说明供参考。

### 3.1 模板字段说明（数据库）

| 字段 | 说明 |
| --- | --- |
| `template_key` | 唯一标识，如 `settle_win_1` |
| `scene` | `settle_push` / `idle_push` |
| `content` | 支持 `{amount}`、`{n}` 等占位符 |
| `enabled` | 是否启用 |
| `weight` | 权重（0-100） |

---

## 4. 原 ai_templates.yaml 种子模板（仅供参考）

原

```yaml
templates:
  # ===== 结算赢推送 =====
  - key: "settle_win_1"
    scene: "settle_push"
    category: "win"
    content: "赢了呢!{amount}龟币入账"
    placeholders: ["amount"]
    enabled: true
    weight: 100
    dailyLimit: 0
    createdBy: "system"
    createdAt: "2026-06-01T00:00:00Z"
  
  - key: "settle_win_2"
    scene: "settle_push"
    category: "win"
    content: "💚 恭喜赢了，{amount}龟币到账~继续加油!"
    placeholders: ["amount"]
    enabled: true
    weight: 100
  
  - key: "settle_win_3"
    scene: "settle_push"
    category: "win"
    content: "好运来啦，{amount}龟币收入囊中"
    placeholders: ["amount"]
    enabled: true
    weight: 100
  
  # ===== 结算输推送 =====
  - key: "settle_lose_1"
    scene: "settle_push"
    category: "lose"
    content: "这把没看对，-{amount}龟币"
    placeholders: ["amount"]
    enabled: true
    weight: 100
  
  - key: "settle_lose_2"
    scene: "settle_push"
    category: "lose"
    content: "失手了呢，-{amount}龟币，下把努力哦"
    placeholders: ["amount"]
    enabled: true
    weight: 100
  
  - key: "settle_lose_3"
    scene: "settle_push"
    category: "lose"
    content: "不过没关系，看看你的战绩，这只是小挫折~"
    placeholders: []
    enabled: true
    weight: 80
  
  # ===== 连胜推送 =====
  - key: "win_streak_3"
    scene: "settle_push"
    category: "win_streak"
    content: "连胜{n}场啦!感觉你最近状态不错呢~"
    placeholders: ["n"]
    enabled: true
    weight: 100
  
  - key: "win_streak_5"
    scene: "settle_push"
    category: "win_streak"
    content: "{n}连胜!你就是这个平台的预测之神!"
    placeholders: ["n"]
    enabled: true
    weight: 100
  
  - key: "win_streak_7"
    scene: "settle_push"
    category: "win_streak"
    content: "天呢，{n}连胜!你绝对是这个平台最强的!"
    placeholders: ["n"]
    enabled: true
    weight: 100
  
  # ===== 连败推送 =====
  - key: "lose_streak_5"
    scene: "settle_push"
    category: "lose_streak"
    content: "连败{n}场了，休息一下再来吧~"
    placeholders: ["n"]
    enabled: true
    weight: 100
  
  # ===== 闲置推送 =====
  - key: "idle_1"
    scene: "idle_push"
    category: "idle_recall"
    content: "好久没见你了，想来聊聊最近的比赛吗?"
    placeholders: []
    enabled: true
    weight: 100
  
  - key: "idle_2"
    scene: "idle_push"
    category: "idle_encourage"
    content: "你还好吗?最近没来看比赛，想听听我的分析吗?"
    placeholders: []
    enabled: true
    weight: 100
  
  - key: "idle_3"
    scene: "idle_push"
    category: "idle_challenge"
    content: "大佬，来一把?我有新发现想和你讨论!"
    placeholders: []
    enabled: true
    weight: 80

# 模板选择策略
templateSelectionStrategy:
  deduplicationEnabled: true         # 是否启用去重
  prioritizeUnseenTemplate: true     # 优先选未见过的
  useWeightedRandom: true            # 按权重随机
  topCandidateCount: 3               # 前 3 个候选中随机选择
```

---

## 5. 止血开关（`aiChat.enabled` + 管理端接口）

止血通过以下两种方式实现，**不**存在独立的 `ai_kill_switches.yaml`：

1. **静态关闭**：修改 `bbs-go.yaml` 中 `aiChat.enabled: false` 或 `deepseek.enabled: false`，重启生效。
2. **运行时关闭**：通过管理端接口 `POST /api/admin/ai/kill-switch` 修改内存中的开关状态，立即生效，详见 [06-接口-管理与内部触发.md](./06-接口-管理与内部触发.md#13-止血开关管理)。

> 以下为原文档保留的 `killSwitches` 字段说明（仅供理解业务意图，不要写入 bbs-go.yaml）：

```yaml
# 仅供参考，非真实配置文件结构
killSwitches:
  # ===== 功能禁用开关 =====
  - key: "ai_chat_enabled"
    enabled: true
    level: "critical"
    description: "禁用主动聊天"
    affectedEndpoints: ["/api/ai/chat"]
    rolloutPercentage: 100
    lastUpdatedAt: "2026-06-11T12:00:00Z"
    updatedBy: "admin"
    reason: ""
  
  - key: "ai_push_enabled"
    enabled: true
    level: "high"
    description: "禁用主动推送"
    affectedEndpoints: ["/api/ai/pushes/stream", "/api/ai/pushes/unread"]
    rolloutPercentage: 100
    updatedBy: "system"
    reason: ""
  
  - key: "ai_stamina_deduction"
    enabled: true
    level: "medium"
    description: "禁用体力消耗（调试用）"
    affectedServices: ["AIStaminaService"]
    rolloutPercentage: 0
    updatedBy: "system"
  
  - key: "ai_apple_purchase"
    enabled: true
    level: "medium"
    description: "禁用苹果购买"
    affectedEndpoints: ["/api/ai/stamina/apple"]
    rolloutPercentage: 100
    updatedBy: "system"
  
  # ===== 灰度控制开关 =====
  - key: "ai_idle_push_rollout"
    enabled: true
    level: "medium"
    description: "闲置推送灰度"
    rolloutPercentage: 50                # 50% 用户
    rolloutStrategy: "user_id_hash"      # 按 user_id 哈希
    affectedFeatures: ["idle_push"]
    updatedBy: "product_manager"
  
  - key: "ai_memory_qa_rollout"
    enabled: false
    level: "low"
    description: "记忆问答灰度（未上线）"
    rolloutPercentage: 0
    rolloutStrategy: "user_id_hash"
    updatedBy: "system"

# 止血规则
killSwitchRules:
  # 自动止血条件
  autoBreaker:
    enabled: true
    triggers:
      - metric: "deepseek_error_rate"
        threshold: 0.1                   # 错误率 > 10%
        window: 60000                    # 时间窗口 60s
        action: "set_ai_chat_enabled=false"
      
      - metric: "api_response_time"
        threshold: 5000                  # 响应时间 > 5s
        window: 300000                   # 时间窗口 5分钟
        action: "trigger_alert"
  
  # 手动止血通知
  notificationChannels:
    - type: "email"
      recipients: ["tech-lead@company.com"]
    
    - type: "dingtalk"
      webhookUrl: "${DINGTALK_WEBHOOK_URL}"
    
    - type: "dashboard"
      enabled: true
```

---

## 6. 配置加载与优先级

`bbs-go.yaml` 字段 < 环境变量覆盖（仅 `DEEPSEEK_API_KEY` 支持环境变量注入）

```bash
# 生产环境必须通过环境变量注入 API Key，不得明文写入 yaml
export DEEPSEEK_API_KEY="sk-xxx"
```

> `AI_CHAT_MAX_STAMINA` 等环境变量覆盖不存在，体力参数只能改 `bbs-go.yaml`。

详见补充文档：[09-yaml配置说明-运营维护.md](./09-yaml配置说明-运营维护.md)

