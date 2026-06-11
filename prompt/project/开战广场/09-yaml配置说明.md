# 开战广场 YAML 配置说明

## 1. 配置示例

```yaml
battle:
  minStakeAmount: 100
  publicEntryFeeRate: 0.05
  privateEntryFeeRate: 0

  inviteCode:
    length: 4
    charset: "A-Za-z0-9"
    caseInsensitive: true
    ttlSeconds: 86400

  deadlines:
    pendingSeconds: 86400
    confirmSeconds: 86400
    disputeSeconds: 86400

  cron:
    tickSeconds: 60

  accounts:
    poolUserId: -1
    burnUserId: -2

redis:
  host: 127.0.0.1
  port: 6379
  db: 0
```

## 2. 字段说明

| 路径 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| `battle.minStakeAmount` | int | 100 | 创建赌局最小庄家押注 |
| `battle.publicEntryFeeRate` | float | 0.05 | 公开局入场费率 |
| `battle.privateEntryFeeRate` | float | 0 | 私人局入场费率 |
| `battle.inviteCode.length` | int | 4 | 邀请码长度 |
| `battle.inviteCode.charset` | string | A-Za-z0-9 | 邀请码字符集 |
| `battle.inviteCode.caseInsensitive` | bool | true | 邀请码大小写不敏感 |
| `battle.inviteCode.ttlSeconds` | int | 86400 | 邀请码过期时间（24h） |
| `battle.deadlines.pendingSeconds` | int | 86400 | 庄家宣判窗口 |
| `battle.deadlines.confirmSeconds` | int | 86400 | 挑战者确认窗口 |
| `battle.deadlines.disputeSeconds` | int | 86400 | 管理员仲裁窗口 |
| `battle.cron.tickSeconds` | int | 60 | 状态轮巡频率 |
| `battle.accounts.poolUserId` | int64 | -1 | 资金池账户 |
| `battle.accounts.burnUserId` | int64 | -2 | 销毁账户 |
| `redis.host` | string | 127.0.0.1 | Redis 主机 |
| `redis.port` | int | 6379 | Redis 端口 |
| `redis.db` | int | 0 | Redis 库编号 |

## 3. 生效规则

- 邀请码生成与校验必须读取 `battle.inviteCode` 配置。
- 邀请码唯一写入必须使用 Redis `SET NX EX`，过期时间取 `ttlSeconds`。
- 轮巡任务频率由 `battle.cron.tickSeconds` 控制，不允许小于 10 秒。
