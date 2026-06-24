# 对立PK管理（Admin PK）

路由前缀：/api/admin/pk
认证：管理员登录

## 接口总览

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/admin/pk/topic/list | 话题列表 |
| POST | /api/admin/pk/topic/save | 新增/编辑话题 |
| POST | /api/admin/pk/topic/status | 话题启停 |
| GET | /api/admin/pk/round/list | 回合列表 |
| GET | /api/admin/pk/season/list | 赛季列表 |
| POST | /api/admin/pk/recalc/heat | 重算回合热度 |

## 详细说明

### 1) GET /api/admin/pk/topic/list
请求参数：page, pageSize, status, q

请求参数 JSON 示例（Query）：
```json
{
  "page": 1,
  "pageSize": 20,
  "status": "enabled",
  "q": "足球"
}
```

返回 data：
- list（buildTopicItem 结构，含 topic/round/season/个人态字段占位）
- count, page, pageSize

返回 JSON 示例（data）：
```json
{
  "list": [
    {
      "topic": {
        "id": 1,
        "slug": "pk-hero",
        "title": "足球GOAT之争",
        "sideAName": "梅西",
        "sideBName": "C罗",
        "status": "enabled"
      },
      "round": {
        "id": 101,
        "phase": "betting",
        "winner": "",
        "settledAt": 0
      },
      "season": {
        "id": 11,
        "seasonNo": 3,
        "status": "active"
      }
    }
  ],
  "count": 1,
  "page": 1,
  "pageSize": 20
}
```

**请求字段说明：**
| 字段 | 说明 |
|------|------|
| page / pageSize | 分页参数 |
| status | 筛选话题状态：enabled=启用，disabled=停用，不传则返回全部 |
| q | 搜索关键词，匹配标题、slug、阵营名称 |

**返回字段说明（list 每项）：**
| 字段 | 说明 |
|------|------|
| topic.status | 话题状态：enabled=启用，disabled=停用 |
| round.phase | 当前回合阶段：betting=下注期，locked=锁局期，cooldown=冷却期 |
| round.winner | 胜方，回合结算前为空字符串 |
| round.settledAt | 结算完成时间戳，0 表示未结算 |
| season.status | 赛季状态：active=进行中，finished=已结束 |

### 2) POST /api/admin/pk/topic/save
请求体字段：
- id（编辑时传）
- slug, title, sideAName, sideBName
- status(enabled/disabled), sort
- cover, listImage
- sideABgImage, sideBBgImage
- sideABgColor, sideBBgColor

请求 JSON 示例：
```json
{
  "id": 1,
  "slug": "pk-hero",
  "title": "足球GOAT之争",
  "sideAName": "梅西",
  "sideBName": "C罗",
  "status": "enabled",
  "sort": 100,
  "cover": "https://example.com/pk/cover.png",
  "listImage": "https://example.com/pk/list.png",
  "sideABgImage": "https://example.com/pk/a-bg.png",
  "sideBBgImage": "https://example.com/pk/b-bg.png",
  "sideABgColor": "#E23D3D",
  "sideBBgColor": "#276EF1"
}
```

返回 data：topic

返回 JSON 示例（data）：
```json
{
  "id": 1,
  "slug": "pk-hero",
  "title": "足球GOAT之争",
  "sideAName": "梅西",
  "sideBName": "C罗",
  "status": "enabled",
  "sort": 100,
  "cover": "https://example.com/pk/cover.png",
  "listImage": "https://example.com/pk/list.png",
  "sideABgImage": "https://example.com/pk/a-bg.png",
  "sideBBgImage": "https://example.com/pk/b-bg.png",
  "sideABgColor": "#E23D3D",
  "sideBBgColor": "#276EF1",
  "currentRoundId": 101,
  "currentSeasonId": 11
}
```

**请求字段说明：**
| 字段 | 说明 |
|------|------|
| id | 话题 ID，编辑时必传，新建时不传 |
| slug | 话题唯一标识（URL 友好），全局唯一，不可重复 |
| title | 话题标题，必填 |
| sideAName / sideBName | A/B 阵营名称，必填 |
| status | 话题状态：enabled=启用，disabled=停用 |
| sort | 排序权重，数值越大越靠前 |
| cover | 话题封面图 URL |
| listImage | 列表展示图 URL |
| sideABgImage / sideBBgImage | A/B 阵营撕裂带背景图 URL |
| sideABgColor / sideBBgColor | A/B 阵营背景色（十六进制色值） |

**返回字段说明：**
| 字段 | 说明 |
|------|------|
| currentRoundId | 新建话题时自动生成的首个回合 ID |
| currentSeasonId | 新建话题时自动生成的首个赛季 ID |

常见错误：
- title is required
- sides are required
- invalid status
- slug already exists
- pk topic not found

### 3) POST /api/admin/pk/topic/status
请求参数或请求体：topicId, status

请求 JSON 示例：
```json
{
  "topicId": 1,
  "status": "disabled"
}
```

返回 data：topic

返回 JSON 示例（data）：
```json
{
  "id": 1,
  "slug": "pk-hero",
  "title": "足球GOAT之争",
  "status": "disabled"
}
```

**请求字段说明：**
| 字段 | 说明 |
|------|------|
| topicId | 话题 ID |
| status | 目标状态：enabled=启用，disabled=停用 |

**返回字段说明：**
返回更新后的完整 topic 对象。

常见错误：
- invalid status
- pk topic not found

### 4) GET /api/admin/pk/round/list
请求参数：page, pageSize, topicId, phase, winner

请求参数 JSON 示例（Query）：
```json
{
  "page": 1,
  "pageSize": 20,
  "topicId": 1,
  "phase": "betting",
  "winner": ""
}
```

返回 data：
- list（PKRound）
- count, page, pageSize

返回 JSON 示例（data）：
```json
{
  "list": [
    {
      "id": 101,
      "topicId": 1,
      "seasonId": 11,
      "roundNo": 11,
      "phase": "betting",
      "heatA": 140.5,
      "heatB": 132.2,
      "poolA": 2400,
      "poolB": 2100,
      "winner": "",
      "settledAt": 0
    }
  ],
  "count": 1,
  "page": 1,
  "pageSize": 20
}
```

**请求字段说明：**
| 字段 | 说明 |
|------|------|
| topicId | 按话题筛选，不传则返回全部 |
| phase | 按阶段筛选：betting / locked / cooldown |
| winner | 按胜方筛选：A / B / draw，不传则返回全部 |

**返回字段说明（list 每项）：**
| 字段 | 说明 |
|------|------|
| roundNo | 回合序号，从 1 开始递增 |
| phase | 当前阶段 |
| heatA / heatB | A/B 阵营热度（来自撕裂带快照或本地重算） |
| poolA / poolB | A/B 阵营奖池，单位龟币 |
| winner | 胜方，未结算为空字符串 |
| settledAt | 结算完成时间戳，0 表示未结算 |

### 5) GET /api/admin/pk/season/list
请求参数：page, pageSize, topicId, status

请求参数 JSON 示例（Query）：
```json
{
  "page": 1,
  "pageSize": 20,
  "topicId": 1,
  "status": "active"
}
```

返回 data：
- list（PKSeason）
- count, page, pageSize

返回 JSON 示例（data）：
```json
{
  "list": [
    {
      "id": 11,
      "topicId": 1,
      "seasonNo": 3,
      "status": "active",
      "totalRounds": 10,
      "winsA": 6,
      "winsB": 4,
      "champion": ""
    }
  ],
  "count": 1,
  "page": 1,
  "pageSize": 20
}
```

**请求字段说明：**
| 字段 | 说明 |
|------|------|
| topicId | 按话题筛选，不传则返回全部 |
| status | 赛季状态筛选：active=进行中，finished=已结束 |

**返回字段说明（list 每项）：**
| 字段 | 说明 |
|------|------|
| seasonNo | 赛季序号，从 1 开始递增 |
| status | 赛季状态：active=进行中，finished=已结束 |
| totalRounds | 本赛季总回合数 |
| winsA / winsB | 本赛季 A/B 阵营胜场数 |
| champion | 赛季冠军（赛季结束后填写），进行中为空字符串 |

### 6) POST /api/admin/pk/recalc/heat
请求参数或请求体：roundId

请求 JSON 示例：
```json
{
  "roundId": 101
}
```

返回 data：
- round
- heatA
- heatB

返回 JSON 示例（data）：
```json
{
  "round": {
    "id": 101,
    "topicId": 1,
    "phase": "betting",
    "heatA": 141.5,
    "heatB": 132.2
  },
  "heatA": 141.5,
  "heatB": 132.2
}
```

**请求字段说明：**
| 字段 | 说明 |
|------|------|
| roundId | 需要重算热度的回合 ID |

**返回字段说明：**
| 字段 | 说明 |
|------|------|
| round | 重算后的回合对象，含最新 heatA / heatB |
| heatA / heatB | 重算后 A/B 阵营热度（与 round.heatA / heatB 相同，方便直接读取） |

常见错误：
- pk round not found

## 备注

- 管理端当前没有 retry_settle 接口，实现中仍是 recalc heat。
- 文档路径已按控制器方法 PostRecalcHeat 对应的 /recalc/heat 同步。