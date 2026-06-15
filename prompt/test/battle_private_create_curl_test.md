# 开战广场：创建私人局 curl 测试

本文档用于手工测试接口：POST /api/battle/create（创建私人开战广场）。

约定：
- 成功返回：{"success":true,"data":...}
- 失败返回：{"success":false,"message":"..."}

## 0. 测试前准备

### 0.1 环境变量

```bash
export BASE_URL="https://52.220.192.18"
export TOKEN="d552657e68444e0296d59e81a35c0bbd"
export NOW_TS=$(date +%s)
export SETTLE_TS=$((NOW_TS + 3600))
```

### 0.2 鉴权方式

本项目常见写法是 Cookie：bbsgo_token。

```bash
export AUTH_COOKIE="bbsgo_token=${TOKEN}"
```

如果你的环境使用 Bearer Token，可改成：

```bash
export AUTH_HEADER="Authorization: Bearer ${TOKEN}"
```

## 1) 创建私人开战广场

请求说明：
- isPublic 必须为 false
- 邀请码 inviteCode 由服务端生成，客户端不要传
- stakeAmount 最小 100
- settleTime 使用秒级时间戳

### 1.1 使用 Cookie 鉴权（推荐）

```bash
export BASE_URL="https://52.220.192.18"
export TOKEN="fc7315c2cd57450a98791ace63a6a42e"
export NOW_TS=$(date +%s)
export SETTLE_TS=$((NOW_TS + 3600))
export AUTH_COOKIE="bbsgo_token=${TOKEN}"

curl -sS -k -X POST "${BASE_URL}/api/battle/create" \
  -b "${AUTH_COOKIE}" \
  -H "Content-Type: application/json" \
  --data-raw "{\
    \"title\": \"私人局测试：主队能否净胜21球\",\
    \"bankerSide\": \"能\",\
    \"challengerSide\": \"不能\",\
    \"stakeAmount\": 1000,\
    \"isPublic\": false,\
    \"settleTime\": ${SETTLE_TS},\
    \"requestId\": \"create-private-$(date +%s)\"\
  }" | jq
```

### 1.2 使用 Bearer 鉴权（可选）

```bash
curl -sS -k -X POST "${BASE_URL}/api/battle/create" \
  -H "${AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  --data-raw "{\
    \"title\": \"私人局测试：客队是否不败\",\
    \"bankerSide\": \"是\",\
    \"challengerSide\": \"否\",\
    \"stakeAmount\": 1000,\
    \"isPublic\": false,\
    \"settleTime\": ${SETTLE_TS},\
    \"requestId\": \"create-private-bearer-$(date +%s)\"\
  }" | jq
```

## 2) 成功响应检查点

执行成功后重点确认：
- success 为 true
- data.battle.isPublic 为 false
- data.battle.status 为 open
- data.inviteCode 存在，且是 4 位字母数字
- data.inviteExpireAt 为有效时间戳（约 48 小时有效）

示例响应（节选）：

```json
{
  "success": true,
  "data": {
    "battle": {
      "id": 123,
      "isPublic": false,
      "status": "open"
    },
    "inviteCode": "A9K2",
    "inviteExpireAt": 1760000000
  }
}
```

## 3) 常见失败与排查

- title is required：title 为空
- sides are required：bankerSide 或 challengerSide 为空
- stakeAmount must be >= 100：本金小于 100
- settleTime is required：settleTime 未传或小于等于 0
- insufficient balance：用户余额不足
- 未登录/401：TOKEN 无效、过期，或鉴权方式不匹配

## 4) 可选：快速校验返回的邀请码

从创建结果中拿到 battle.id 和 inviteCode 后，可用详情接口验证私人局可见性：

```bash
export BATTLE_ID="<上一步返回的 battle.id>"
export INVITE_CODE="<上一步返回的 inviteCode>"

curl -sS "${BASE_URL}/api/battle/by?battleId=${BATTLE_ID}&inviteCode=${INVITE_CODE}" \
  -b "${AUTH_COOKIE}" | jq
```