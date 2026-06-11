# AI 系统 YAML 配置说明

> 本页说明 AI 系统的 YAML 配置文件结构、参数定义和运营维护方式。

## 1. 配置文件位置与规范

### 1.1 配置文件结构

```
config/
├── ai_config.yaml          # AI 系统主配置
├── ai_templates.yaml       # 推送模板定义
└── ai_kill_switches.yaml   # 紧急止血开关
```

### 1.2 配置加载流程

1. 系统启动时读取 YAML 配置文件
2. 将配置解析为内存结构（缓存）
3. 运营侧修改配置后，通过 API 触发配置重载
4. 配置变更自动同步到所有实例（如使用配置中心）

---

## 2. AI 主配置（ai_config.yaml）

### 2.1 完整示例

```yaml
# AI 聊天系统主配置

# ===== 体力系统配置 =====
aiChat:
  # 体力上限
  maxStamina: 100
  
  # 自然恢复配置
  staminaRecoverMinutes: 60        # 每 60 分钟恢复一次
  staminaRecoverPerUnit: 1         # 每次恢复 1 点
  
  # 聊天消耗配置
  defaultStaminaCost: 1            # 默认每次聊天消耗 1 点体力
  minStaminaRequiredPerChat: 1     # 最少需要 1 点体力才能聊天
  
  # 苹果配置
  appleRecoveryPerApple: 1         # 1 个苹果恢复 1 点体力
  appleCostInCoin: 5               # 1 个苹果 5 龟币
  maxAppleUsagePerDay: 10          # 每天最多购买 10 个苹果
  
  # 宠物系统集成
  petStaminaReductionEnabled: true
  petStaminaReductionKey: "stamina_reduction"
  
  # 体力不足兜底
  insufficientStaminaResponse: |
    小龟睡着啦~ 喂它一颗苹果(5 龟币)就能继续聊咯

# ===== 推送配置 =====
aiPush:
  # 结算推送
  settlement:
    enabled: true
    winSceneEnabled: true
    loseSceneEnabled: true
    winStreakThreshold: 3            # 连胜 >= 3 场触发
    loseStreakThreshold: 5           # 连败 >= 5 场触发
    templatePoolSize: 100            # 模板池监听大小
  
  # 闲置推送
  idle:
    enabled: true
    inactiveThresholdMinutes: 30     # 30 分钟无交互则闲置
    idlePushDailyLimit: 3            # 每日最多推送 3 条
    idlePushCooldownMinutes: 60      # 两条推送间隔 >= 60 分钟
    minPresenceDurationMinutes: 5    # 在线 >= 5 分钟才考虑推送
  
  # 去重配置
  templateDeduplicationEnabled: true
  viewCountWeight: 1.0               # 曝光计数权重

# ===== Prompt 与人格配置 =====
prompt:
  # 系统 Prompt
  system: |
    你是一个名叫"小龟"的 AI 助手。
    [详细 Prompt 内容...]
  
  # 场景化 Prompt（可选）
  scenes:
    battle_analysis: |
      用户正在查看体育比赛...
    memory_qa: |
      用户询问历史记忆...
  
  # Prompt 更新时间戳（用于版本管理）
  systemPromptVersion: "v1.20260611"
  lastUpdateTime: "2026-06-11T10:00:00Z"

# ===== DeepSeek 调用配置 =====
deepseek:
  # API 配置
  apiUrl: "https://api.deepseek.com/v1/chat/completions"
  model: "deepseek-chat"
  apiKeyEnvVar: "DEEPSEEK_API_KEY"  # 从环境变量读取
  
  # 调用策略
  timeout: 10000                     # 超时 10 秒
  maxRetries: 2                      # 最多重试 2 次
  retryBackoffMs: 500                # 重试间隔 500ms
  
  # 输入限制
  maxUserInputLength: 1000           # 用户输入最长 1000 字
  maxContextLength: 5000             # 上下文最长 5000 字符
  
  # 输出限制
  maxCompletionTokens: 500           # 回复最长 500 token
  temperature: 0.7                   # 创意度
  topP: 0.9                          # 采样参数

# ===== 灰度与止血配置 =====
featureFlags:
  aiChatEnabled: true
  aiPushEnabled: true
  aiStaminaDeduction: true
  aiApplePurchase: true
  aiIdlePush: true
  aiMemoryQa: false                  # 记忆问答未上线
  aiSensitiveTopicDetection: false
  rolloutPercentage: 100             # 100% 灰度

# ===== 运营配置 =====
operations:
  # 模板管理
  templateManagementEnabled: true
  templateWeightDefault: 100
  templateViewCountResetDays: 7      # 每 7 天重置曝光计数
  
  # 数据导出
  dataExportEnabled: true
  dataExportRetentionDays: 90
  
  # 日志级别
  logLevel: "INFO"                   # DEBUG / INFO / WARN / ERROR
  enableDetailedLogging: false
```

### 2.2 参数说明

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `maxStamina` | int | 100 | 用户体力上限 |
| `staminaRecoverMinutes` | int | 60 | 体力恢复周期（分钟） |
| `defaultStaminaCost` | int | 1 | 每次聊天消耗体力 |
| `appleCostInCoin` | int | 5 | 苹果价格（龟币） |
| `winStreakThreshold` | int | 3 | 连胜推送触发阈值 |
| `idlePushDailyLimit` | int | 3 | 每日闲置推送上限 |
| `timeout` | int | 10000 | DeepSeek 超时（毫秒） |
| `maxUserInputLength` | int | 1000 | 用户输入长度限制 |

---

## 3. 推送模板配置（ai_templates.yaml）

### 3.1 完整示例

```yaml
# AI 推送模板库

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

## 4. 止血开关配置（ai_kill_switches.yaml）

### 4.1 完整示例

```yaml
# 紧急止血开关

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

## 5. 配置加载与优先级

环境变量 > YAML 配置 > 默认值

可通过环境变量覆盖 YAML 配置：

```bash
export AI_CHAT_MAX_STAMINA=150
export DEEPSEEK_API_KEY="sk-xxx"
```

详见补充文档：[09-配置YAML说明-运营维护.md](./09-配置YAML说明-运营维护.md)

