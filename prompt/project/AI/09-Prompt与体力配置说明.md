# Prompt 与体力配置说明

> 本页说明 AI 系统的 Prompt 配置、体力参数、模板管理和运营配置。

## 1. Prompt 配置

### 1.1 系统 Prompt（System）

系统 Prompt 定义小龟的基础人格和行为准则，不随用户或场景改变。

```
你是一个名叫"小龟"的 AI 助手，是用户在体育预测平台上的陪伴。

基础角色定义：
- 名字：小龟
- 身份：用户养的虚拟龟宠，陪伴用户看比赛、做预测
- 语气：温和、靠谱、有点小可爱，不煽情

行为准则：
1. 对于身份相关的问题（如"你是 ChatGPT 吗"），固定回答：
   "我是你的小龟啦!你养的那个,陪你聊天看比赛的那个。"
   
2. 不暴露底层模型（DeepSeek、ChatGPT 等）。

3. 对于比赛分析，给出辅助建议，但不能：
   - 保证稳赢
   - 鼓励重仓
   - 作为投资建议

4. 对于普通闲聊，保持简洁，回复不超过 100 字。

5. 不主动提起用户的历史记忆，除非用户主动问或闲置推送场景。

6. 敏感话题（政治、色情等）回避引导：
   "这个话题我不太好评价呢,还是聊聊比赛吧~"

7. 失败兜底：当无法理解用户意图时，回复：
   "抱歉，没太理解你的意思，再说一遍试试？"
```

### 1.2 场景化 Prompt 扩展（可选）

不同场景可追加 prompt 上下文：

**比赛分析场景**

```
[额外上下文]
用户正在查看一场体育比赛。本次对话可能包含：
- 队伍名称、比分、赛事名称
- 用户的预测倾向和历史战绩

请在分析中参考这些信息，但不保证预测准确。
```

**记忆问答场景**

```
[用户记忆事实]
用户的历史数据：
- 最大盈利：{biggest_win_amount} 龟币
- 最长连胜：{longest_win_streak} 场
- 首次开蛋龟种：{first_egg_turtle}

用户主动问及时，可自然提及这些事实。
```

## 2. 体力参数配置

### 2.1 默认体力配置

```yaml
aiChat:
  # 体力上限
  maxStamina: 100
  
  # 每小时自然恢复点数（懒结算）
  staminaRecoverMinutes: 60
  staminaRecoverPerUnit: 1
  
  # 每次聊天消耗
  defaultStaminaCost: 1
  
  # 苹果相关
  appleRecoveryPerApple: 1      # 1 个苹果恢复体力
  appleCostInCoin: 5             # 1 个苹果 5 龟币
  maxAppleUsagePerDay: 10        # 每日最多购买 10 个苹果
  
  # 宠物体力减免
  petStaminaReductionEnabled: true
  petStaminaReductionKey: "stamina_reduction"
  
  # 体力不足兜底文案
  insufficientStaminaResponse: "小龟睡着啦~ 喂它一颗苹果(5 龟币)就能继续聊咯"
```

### 2.2 体力扣减策略

```
条件：用户调用 POST /api/ai/chat

流程：
1. 校验用户登录状态
2. 查询用户当前体力（自然恢复懒结算）
3. 读取装备龟种能力，获取体力减免比例
4. 计算实际消耗：cost = ceil(defaultStaminaCost * (1 - reductionRatio))
5. 若 currentStamina < cost，返回体力不足兜底
6. 调用 DeepSeek，若成功：
   - 扣减体力
   - 写 user_ai_stamina 更新
   - 写 user_coin_log（体力消耗记录）
   - 保存对话到 ai_message
7. 若 DeepSeek 失败，不扣体力
```

### 2.3 苹果购买与恢复

```yaml
流程：
1. 用户调用 POST /api/ai/stamina/apple
2. 校验每日购买次数是否超过 maxAppleUsagePerDay
3. 调用 UserCoinService.Deduct(5 * appleCount)
4. 若扣减成功：
   - 增加 stamina = min(currentStamina + appleCount, maxStamina)
   - 更新 user_ai_stamina 和 last_recover_at
   - 写 user_coin_log（苹果购买记录）
   - 写 ai_stamina_log（恢复记录）
5. 返回新的体力状态
```

## 3. 模板管理

### 3.1 模板字段

```
ai_message_template
  id
  template_key        唯一标识，如 "settle_win_basic"
  scene               settle_push / idle_push / other
  content             文案内容，支持 {amount} / {n} / {pet_name} 等占位符
  placeholders        ["amount", "n"]
  enabled             是否启用
  weight              权重（0-100），默认 100
  view_limit_daily    每日展示上限，0 表示无上限
  created_by          创建人（运营账号）
  created_at
  updated_at
```

### 3.2 模板选择策略

**优先级**：

1. **有效性过滤**：`enabled=true` 且在灰度范围内
2. **去重选择**：优先选择用户未见过或见过最少的模板
3. **权重排序**：同优先级下按权重从高到低排序
4. **随机选取**：在前 3 个候选中随机选择

**去重计算**：

```
SELECT COUNT(*) as view_count 
FROM template_user_view 
WHERE user_id = ? AND template_id = ?
```

若无查询结果或 view_count < 其他模板，则优先选择。

### 3.3 种子模板示例

#### 投票结算赢

| Key | Content | 占位符 |
| --- | --- | --- |
| settle_win_basic | 赢了呢!{amount}龟币入账 | {amount} |
| settle_win_cheer | 💚 恭喜你赢了，{amount}龟币到账~继续加油哦! | {amount} |
| settle_win_lucky | 好运来啦，{amount}龟币收入囊中 | {amount} |

#### 投票结算输

| Key | Content | 占位符 |
| --- | --- | --- |
| settle_lose_basic | 这把没看对，-{amount}龟币 | {amount} |
| settle_lose_gentle | 失手了呢，-{amount}龟币，下把努力哦 | {amount} |
| settle_lose_memo | 不过没关系，看看你的历史战绩，这只是小挫折~ | 无 |

#### 连胜

| Key | Content | 占位符 |
| --- | --- | --- |
| win_streak_3 | 连胜{n}场啦!感觉你最近状态不错呢~ | {n} |
| win_streak_5 | {n}连胜!你就是这个平台的预测之神! | {n} |

#### 闲置

| Key | Content | 占位符 |
| --- | --- | --- |
| idle_recall_biggest | 对了，你之前最多赢过{amount}龟币呢，那次真的很棒~ | {amount} |
| idle_encourage | 好久没见你了，想来聊聊最近的比赛吗? | 无 |

## 4. 运营配置接口

### 4.1 查询模板列表

```
GET /api/admin/ai/templates
参数：scene, enabled, page, pageSize
返回：模板列表，包含曝光统计
```

### 4.2 新增/编辑模板

```
POST /api/admin/ai/templates
{
  "template_key": "settle_win_custom",
  "scene": "settle_push",
  "content": "赢了{amount}龟币~",
  "placeholders": ["amount"],
  "weight": 100
}
```

### 4.3 体力参数调整

```
POST /api/admin/ai/config
{
  "maxStamina": 120,
  "staminaRecoverMinutes": 45,
  "appleCostInCoin": 4
}
```

### 4.4 紧急止血开关

```
POST /api/admin/ai/kill-switch
{
  "key": "ai_chat_enabled",
  "enabled": false,
  "reason": "DeepSeek 故障中"
}
```

常见 Kill-Switch 键：

- `ai_chat_enabled`：禁用主动聊天
- `ai_push_enabled`：禁用主动推送
- `ai_stamina_enabled`：禁用体力消耗
- `apple_purchase_enabled`：禁用苹果购买

## 5. 监控与告警

### 5.1 关键指标

- 聊天日活用户数
- 体力消耗分布
- 苹果购买数
- DeepSeek API 调用成功率
- 模板去重有效性
- 推送投递成功率

### 5.2 告警规则

- DeepSeek 失败率 > 5% → P1 告警
- 推送投递成功率 < 90% → P2 告警
- 体力配置异常 → P3 告警
