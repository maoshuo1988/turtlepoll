# 撕裂带YAML配置说明

## 1. 配置样例

```yaml
tear:
  enabled: true
  dark:
    close_countdown_seconds: 60
    mint_ratio: 0.1
    mint_floor: 10
  round:
    duration_seconds: 259200
    cooldown_seconds: 600
  heat:
    comment_cap: 20
    coin_factor: 0.02
  settle:
    retry_max: 3
    retry_backoff_seconds: [10, 30, 60]
    lock_ttl_seconds: 300
  external:
    result_source_timeout_ms: 1500
    result_source_retry: 2
    coin_timeout_ms: 1000
    coin_retry: 2
    user_profile_timeout_ms: 500
    user_profile_batch_limit: 200
```

## 2. 字段说明

| 路径 | 类型 | 约束 |
|------|------|------|
| tear.enabled | bool | 必填 |
| tear.dark.close_countdown_seconds | int | `>=0` |
| tear.dark.mint_ratio | float | `0<ratio<=1` |
| tear.dark.mint_floor | int | `>=0` |
| tear.round.duration_seconds | int | 固定 `259200` |
| tear.round.cooldown_seconds | int | 固定 `600`，仅用于对立PK业务冷却期，不作为热度冻结条件 |
| tear.heat.comment_cap | int | 固定 `20` |
| tear.heat.coin_factor | float | 固定 `0.02` |
| tear.settle.retry_max | int | `1~5` |
| tear.settle.lock_ttl_seconds | int | `60~900` |
| tear.external.user_profile_timeout_ms | int | `100~2000` |
| tear.external.user_profile_batch_limit | int | `1~500` |

## 3. 外部特性依赖详情

### 1.1 EXT-01 事件赛果拉取

- 接口名: `ResultProvider.GetEventResult`
- 调用方向: TearSettleJob -> 外部赛事/事件数据源
- 输入参数:
  - `bizId` (string, 必填)
  - `sceneType` (string, 必填)
- 输出参数:
  - `status` (string)
  - `winnerCamp` (string)
  - `officialTime` (int64)
- 错误码:
  - `RESULT_NOT_READY`: 结果未就绪
  - `RESULT_NOT_FOUND`: 事件不存在
- 固定策略:
  - 超时 `1500ms`
  - 重试 `2次`
  - 间隔 `200ms` 固定退避
  - 熔断窗口 `60s`, 失败阈值 `20次`
- 验收标准:
  - 99% 请求在 `2s` 内返回

### 1.2 EXT-02 开放通知推送

- 接口名: `NotifyCenter.SendEventOpen`
- 调用方向: NotifyService -> 站内通知中心
- 输入参数:
  - `eventId` (int64, 必填)
  - `userIds` (array<int64>, 必填)
  - `templateCode` (string, 必填)
- 输出参数:
  - `successCount` (int)
  - `failedCount` (int)
- 错误码:
  - `NOTIFY_RATE_LIMIT`: 触发限流
- 固定策略:
  - 超时 `800ms`
  - 重试 `1次`
  - 并发 `20`
  - 限速 `500 req/min`
- 验收标准:
  - 单批 1000 用户通知在 `30s` 内完成

### 1.3 EXT-03 风控账号状态查询

- 接口名: `RiskService.GetUserRiskState`
- 调用方向: TearInteractService -> 用户风控中心
- 输入参数:
  - `userId` (int64, 必填)
- 输出参数:
  - `riskState` (string)
  - `canBet` (bool)
  - `canInteract` (bool)
- 错误码:
  - `RISK_UNAVAILABLE`: 风控服务不可用
- 固定策略:
  - 超时 `500ms`
  - 重试 `1次`
  - 熔断窗口 `30s`, 失败阈值 `50次`
- 验收标准:
  - 查询成功率 `>=99.9%`

### 1.4 EXT-04 资产入账服务

- 接口名: `CoinAccountService.Credit`
- 调用方向: TearSettleService -> CoinAccountService
- 输入参数:
  - `userId` (int64, 必填)
  - `amount` (int64, 必填)
  - `bizType` (string, 必填)
  - `requestId` (string, 必填)
- 输出参数:
  - `txId` (int64)
  - `balance` (int64)
- 错误码:
  - `COIN_DUPLICATE_REQUEST`: 幂等命中
  - `COIN_SERVICE_TIMEOUT`: 入账超时
- 固定策略:
  - 超时 `1000ms`
  - 重试 `2次`
  - 并发 `10`
  - requestId 幂等窗口 `48h`
- 验收标准:
  - 重放同一 requestId 不产生重复入账

### 1.5 EXT-05 用户资料批量查询

- 接口名: `UserProfileService.BatchGetUserProfile`
- 调用方向: TearQueryService -> UserProfileService
- 输入参数:
  - `userIds` (array<int64>, 必填)
- 输出参数:
  - `profiles[]` (array, 每项包含 `userId/username/avatar`)
- 错误码:
  - `PROFILE_PARTIAL_MISS`: 部分用户资料缺失
  - `PROFILE_SERVICE_TIMEOUT`: 服务超时
- 固定策略:
  - 超时 `500ms`
  - 重试 `1次`
  - 单批上限 `200`
  - 并发 `10`
- 验收标准:
  - 榜单接口中用户名与头像回填成功率 `>=99.9%`
