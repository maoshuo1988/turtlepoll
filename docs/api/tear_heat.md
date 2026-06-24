# Tear Heat Service

撕裂带热度服务用于把上游业务动作转换为热度快照，并维护用户在事件维度的累计热度。

当前代码仅提供 service/repository 能力；业务入口接入可在独立分支完成。

## 业务类型

| 值 | 含义 |
|----|------|
| 1 | 暗盘 |
| 2 | 开撕台 |

对应常量：

- `services.TearBusinessTypeDark`
- `services.TearBusinessTypeArena`

## TearHeatService.AddPKHeat

```go
func (s *tearHeatService) AddPKHeat(
    tx *gorm.DB,
    roundId int64,
    userId int64,
    side string,
    businessType int,
    isLike bool,
    isComment bool,
    isBet bool,
    amount float64,
) error
```

### 参数

| 参数 | 说明 |
|------|------|
| `tx` | 事务对象；传 `nil` 时使用 `sqls.DB()` |
| `roundId` | 开撕台 PK round ID，写入热度表的 `event_id` |
| `userId` | 贡献热度的用户 ID |
| `side` | 阵营/下注对象，通常为 `A` 或 `B` |
| `businessType` | 业务类型，开撕台传 `2` |
| `isLike` | 是否点赞动作 |
| `isComment` | 是否评论/回复动作 |
| `isBet` | 是否下注动作 |
| `amount` | 下注金额；仅 `isBet=true` 时参与计算 |

### 热度计算

| 动作 | 字段 | 规则 |
|------|------|------|
| 点赞 | `h_like` | `1` |
| 评论/回复 | `h_comment` | `2` |
| 下注 | `h_coin` | `amount * 0.02` |

### 写入行为

每次调用会写入一条 `T_TEAR_HEAT_SNAPSHOT` 动作快照。

同时会按以下唯一维度累计更新 `T_TEAR_EVENT_HEAT`：

```text
event_id + user_id + option + business_type
```

累计总热度：

```text
h_total = h_like + h_comment + h_coin
```

## TearEventHeatService.AddHeat

```go
func (s *tearEventHeatService) AddHeat(
    tx *gorm.DB,
    eventId int64,
    userId int64,
    option string,
    businessType int,
    hLike float64,
    hComment float64,
    hCoin float64,
) error
```

用于直接累计写入 `T_TEAR_EVENT_HEAT`。如果目标记录不存在则创建，存在则累加 `h_like`、`h_comment`、`h_coin` 并重算 `h_total`。

## TearEventHeatService.SetHeat

```go
func (s *tearEventHeatService) SetHeat(
    tx *gorm.DB,
    eventId int64,
    userId int64,
    option string,
    businessType int,
    hLike float64,
    hComment float64,
    hCoin float64,
) error
```

用于复算场景覆盖写入 `T_TEAR_EVENT_HEAT`。如果目标记录不存在则创建，存在则覆盖热度字段并重算 `h_total`。

## 上游调用示例

评论：

```go
err := services.TearHeatService.AddPKHeat(tx, roundId, userId, side, services.TearBusinessTypeArena, false, true, false, 0)
```

下注：

```go
err := services.TearHeatService.AddPKHeat(tx, roundId, userId, side, services.TearBusinessTypeArena, false, false, true, float64(amount))
```

点赞：

```go
err := services.TearHeatService.AddPKHeat(tx, roundId, userId, side, services.TearBusinessTypeArena, true, false, false, 0)
```
