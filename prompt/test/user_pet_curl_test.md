# 用户侧宠物（user_pet）接口访问测试（curl）

本文档基于 `docs/api/user_pet.md`，提供一套可复制的 curl 命令，用于对用户侧宠物系统接口做手工冒烟/回归。

> 说明：本文档只负责“怎么打接口”。玩法/规则口径以 `prompt/project/宠物-用户侧宠物使用.md` 为准。

## 0. 测试前准备

### 0.1 环境变量

```bash
export BASE_URL="http://127.0.0.1:8082"
export USER_TOKEN="<your_user_token>"
```

### 0.2 通用 Header

```bash
export USER_AUTH_HEADER="Authorization: Bearer ${USER_TOKEN}"
```

> 如果你的项目鉴权 header 不是 `Authorization: Bearer ...`，把下面示例里的 header 名改掉即可。

### 0.3 可选：jq

示例里很多命令带了 `| jq` 方便查看；未安装 jq 时可以先去掉。

---

## 1) 登录接口即每日结算入口（主入口）

> 文档约定：登录成功时触发每日结算；结算失败不应阻断登录。
> 
> 注意：本项目实际登录路径可能不是 `/api/user/login`，以你当前服务为准。

### 1.1 POST /api/user/login（示例）

> 如果你已有 token，可以跳过本节，直接从第 2 节开始。

```bash
curl -sS "${BASE_URL}/api/user/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"<username>","password":"<password>"}' | jq
```

输出检查点（建议人工确认）：

- `success==true`
- `data.dailySettle` 存在
- `data.dailySettle.alreadySettled` 随同日重复登录变化
- 结算失败时：`data.dailySettle.errorCode/errorMsg` 有值，但登录仍 success

---

## 2) 当前装备龟种（读）

### 2.1 GET /api/pet/equip

```bash
curl -sS "${BASE_URL}/api/pet/equip" \
  -H "${USER_AUTH_HEADER}" | jq
```

---

## 3) 切换龟种（写）

### 3.1 POST /api/pet/equip

按文档：`petId` 或 `petKey` 二选一（建议只用一种口径）。这里以 `petId` 为例：

```bash
curl -sS "${BASE_URL}/api/pet/equip" \
  -H "${USER_AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d '{"petId":"basic"}' | jq
```

输出检查点：

- `ok==true`
- `nextEffectiveAt` 为“北京时间次日 0 点”

常见错误码（示例，具体以服务端 message/code 为准）：

- `EQUIP_DAILY_LIMIT`
- `DEBT_UNPAID`
- `PARAM_INVALID`（或 `PET_NOT_OWNED`）

---

## 4) 用户龟种资产（列表）

### 4.1 GET /api/pet/owned

```bash
curl -sS "${BASE_URL}/api/pet/owned" \
  -H "${USER_AUTH_HEADER}" | jq
```

---

## 5) 体力（查询 + 消耗/恢复）

### 5.1 查询：GET /api/pet/stamina

```bash
curl -sS "${BASE_URL}/api/pet/stamina" \
  -H "${USER_AUTH_HEADER}" | jq
```

### 5.2 （可选）消耗体力：POST /api/pet/stamina/consume

```bash
curl -sS "${BASE_URL}/api/pet/stamina/consume" \
  -H "${USER_AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d '{"amount":1}' | jq
```

期望错误：体力不足时应返回 `STAMINA_NOT_ENOUGH`（或等价 message）。

### 5.3 （可选）喂食恢复：POST /api/pet/stamina/feed

```bash
curl -sS "${BASE_URL}/api/pet/stamina/feed" \
  -H "${USER_AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d '{"count":1}' | jq
```

期望错误：余额不足时应返回 `INSUFFICIENT_COINS`（或等价 message）。

---

## 6) 开蛋（抽龟）

### 6.1 POST /api/pet/egg/hatch

```bash
curl -sS "${BASE_URL}/api/pet/egg/hatch" \
  -H "${USER_AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d '{}' | jq
```

输出检查点：

- `cost` / `refund` / `isDuplicate`
- `pet`（petId/petKey/rarity/name）
- `balanceBefore/balanceAfter`

---

## 7) 状态页聚合

### 7.1 GET /api/pet/status

```bash
curl -sS "${BASE_URL}/api/pet/status" \
  -H "${USER_AUTH_HEADER}" | jq
```

---

## 最小回归序列（建议）

1)（如需）登录一次，观察 `dailySettle`
2) `GET /api/pet/equip`
3) `GET /api/pet/owned`
4) `GET /api/pet/stamina`
5)（可选）`POST /api/pet/stamina/consume`
6)（可选）`POST /api/pet/stamina/feed`
7) `POST /api/pet/egg/hatch`
8) `GET /api/pet/status`

---

## 常见问题排查

- 401：token 不存在/过期，或 header 名不匹配。
- 400：参数结构不对（例如 `petId` vs `pet_id` / `petKey` vs `pet_key`）。
- 业务错误：对照文档错误码（EQUIP_DAILY_LIMIT / DEBT_UNPAID / STAMINA_NOT_ENOUGH / INSUFFICIENT_COINS）。
