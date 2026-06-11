# YAML 配置说明补充（运营维护与监控）

> 本页补充 09-配置YAML说明 中的运营维护流程、监控指标和回滚方案。

## 1. 配置加载与热重载

### 1.1 环境变量覆盖

```bash
# 可通过环境变量覆盖 YAML 配置
export AI_CHAT_MAX_STAMINA=150
export AI_PUSH_SETTLEMENT_ENABLED=false
export DEEPSEEK_API_KEY="sk-xxx"
```

### 1.2 配置优先级

```
环境变量 > YAML 配置 > 默认值
```

### 1.3 热重载接口

```bash
# 重载配置（管理端）
POST /api/admin/ai/config/reload
{
  "configType": "ai_config",
  "version": "v1.20260611"
}

# 查看当前配置
GET /api/admin/ai/config
```

---

## 2. 运营维护流程

### 2.1 配置变更流程

1. **发起变更**：运营在管理端修改配置参数
2. **验证参数**：系统验证参数有效性（范围、类型等）
3. **保存版本**：新配置版本入库，记录变更者和变更理由
4. **触发重载**：调用热重载接口（单机或分布式）
5. **审计日志**：记录完整的变更链路
6. **团队通知**：推送变更通知到技术 & 产品团队

### 2.2 变更审批（可选）

若需要二级审批：

```
1. 运营提交变更
2. 技术负责人审核
3. 核准或驳回
4. 如核准，触发变更流程
```

### 2.3 变更历史

```sql
SELECT * FROM ai_config_history 
WHERE updated_at > DATE_SUB(NOW(), INTERVAL 7 DAY)
ORDER BY updated_at DESC;
```

---

## 3. 监控与告警

### 3.1 关键指标

| 指标 | 阈值 | 告警等级 | 说明 |
| --- | --- | --- | --- |
| `deepseek_error_rate` | > 5% | P1 | DeepSeek 错误率过高 |
| `deepseek_latency_p99` | > 5s | P2 | 响应延迟过高 |
| `stamina_query_failures` | > 1% | P2 | 体力查询失败 |
| `template_load_errors` | > 0 | P3 | 模板加载异常 |
| `config_reload_latency` | > 1s | P3 | 配置重载耗时 |
| `push_delivery_success_rate` | < 95% | P2 | 推送投递成功率 |
| `kill_switch_activated` | any | P1 | 止血开关被触发 |

### 3.2 告警规则

#### 深度查询 API 故障

```
IF deepseek_error_rate > 10% for 5 minutes
THEN
  - 发送 P1 告警
  - 自动触发 ai_chat_enabled=false
  - 通知技术负责人钉钉群
```

#### 推送系统故障

```
IF push_delivery_success_rate < 90% for 10 minutes
THEN
  - 发送 P2 告警
  - 记录故障日志
  - 通知产品团队
```

#### 配置加载异常

```
IF config_reload_latency > 2s
THEN
  - 发送 P3 告警
  - 切回上一个配置版本（自动）
  - 记录错误日志
```

### 3.3 监控仪表板

建议在运营后台显示：

- 实时 API 调用成功率
- 体力消耗分布
- 苹果购买数
- 推送投递统计
- 配置版本历史
- 止血开关状态

---

## 4. 回滚方案

### 4.1 配置回滚（正常场景）

```
若配置变更导致业务异常：

1. 在管理端"配置历史"页面找到历史版本
2. 点击版本号预览配置内容
3. 确认无误后点击"回滚"按钮
4. 系统自动重载新配置
5. 触发通知（钉钉/邮件）
6. 写回滚审计日志
```

### 4.2 自动回滚（故障场景）

```
当触发止血条件时，系统自动回滚：

示例：
IF deepseek_error_rate > 20% for 2 minutes
THEN
  - 自动触发 ai_chat_enabled = false
  - 将 rolloutPercentage 设为 0
  - 发送紧急告警
  - 管理员需手动恢复
```

### 4.3 模板回滚

```sql
-- 快速回滚所有模板到某个历史版本
UPDATE ai_message_template
SET enabled = CASE 
  WHEN template_key IN (SELECT template_key FROM ai_template_version 
                         WHERE version_id = ?)
  THEN enabled_in_version
  ELSE FALSE
END
WHERE scene = 'settle_push';
```

---

## 5. 版本管理

### 5.1 版本命名规范

```
v{MAJOR}.{MINOR}.{PATCH}.{TIMESTAMP}

示例：
- v1.0.0.20260611：主版本 1，功能版本 0，修复版本 0，日期 2026-06-11
- v1.1.0.20260615：新增闲置推送功能
- v1.1.1.20260616：修复模板渲染 bug
```

### 5.2 发布清单

每个版本发布时需要：

```markdown
## v1.0.0.20260611 Release Notes

### 新增
- [ ] 推送模板 10+ 条
- [ ] 止血开关功能
- [ ] 灰度配置支持

### 修复
- [ ] 体力恢复精度问题

### 配置变更
- [ ] maxStamina: 100 -> 120
- [ ] appleCostInCoin: 5 -> 4

### 验收清单
- [ ] 测试环境验收
- [ ] 灰度 10% 用户验证
- [ ] 灰度 50% 用户验证
- [ ] 全量发布
```

### 5.3 版本快照

```sql
-- 保存每个版本的完整配置快照
INSERT INTO ai_config_snapshot (
  version_id,
  config_json,
  templates_json,
  kill_switches_json,
  published_at,
  published_by
) VALUES (
  'v1.0.0.20260611',
  '{...}',
  '[...]',
  '[...]',
  NOW(),
  'admin_user_id'
);
```

---

## 6. 故障排查清单

### 6.1 用户无法聊天

```
1. 检查 ai_chat_enabled 是否为 true
2. 检查用户体力是否为 0
3. 查看 DeepSeek 错误日志
4. 验证 DEEPSEEK_API_KEY 是否有效
5. 检查网络连接
```

### 6.2 推送未投递

```
1. 检查 ai_push_enabled 是否为 true
2. 检查 SSE 连接是否正常
3. 查看未读消息是否写入数据库
4. 验证 WebSocket 或 SSE 握手
5. 检查浏览器控制台是否有错误
```

### 6.3 模板渲染错误

```
1. 检查占位符 {amount} / {n} 是否被正确替换
2. 验证模板 enabled 字段
3. 查看渲染日志中的错误信息
4. 确认占位符与模板定义匹配
```

---

## 7. 性能优化建议

| 优化项 | 当前 | 目标 | 说明 |
| --- | --- | --- | --- |
| DeepSeek 缓存 | 无 | LRU 1000 | 缓存高频问题的回复 |
| 配置缓存 | 内存 | 本地 + Redis | 减少配置查询 |
| 模板加载 | 实时查询 | 启动时加载 | 减少数据库压力 |
| 推送队列 | 同步 | 异步 MQ | 提高投递吞吐 |

---

## 8. 灾备恢复

### 8.1 数据备份

```bash
# 每日自动备份配置
0 2 * * * mysqldump -u root ai_config > /backup/ai_config_$(date +\%Y\%m\%d).sql
```

### 8.2 快速恢复

```bash
# 从备份恢复（生产前必须在备机验证）
mysql -u root < /backup/ai_config_20260611.sql

# 重启应用
systemctl restart bbs-go
```

### 8.3 验证恢复

```
1. 查询配置版本是否正确
2. 测试聊天接口
3. 测试推送接收
4. 检查监控指标

---

## 9. Prompt 与体力参数维护约束

### 9.1 Prompt 维护规范

1. 系统 Prompt 必须强制“小龟身份”，禁止暴露底层模型名。
2. 场景 Prompt 仅允许追加，不覆盖系统 Prompt 基础约束。
3. 对“保证稳赢、鼓励重仓”类表达必须做禁用词校验。

### 9.2 体力参数变更约束

1. `appleCostInCoin`、`defaultStaminaCost` 变更必须记录变更原因。
2. `maxStamina` 下调时不得导致现有用户体力出现负值。
3. 体力配置变更后，需执行抽样校验：查询体力、聊天扣减、苹果恢复三条链路。

### 9.3 模板维护约束

1. 模板必须声明 `scene` 和占位符列表。
2. 新模板上线前必须通过占位符渲染检查。
3. 模板启停和权重调整必须保留审计日志。
```

