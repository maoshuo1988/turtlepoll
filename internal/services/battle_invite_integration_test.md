// 私人赌局邀请码集成测试文档
// 本文档说明如何在集成环境中验证邀请码的完整生命周期

package services

/*
## 集成测试场景

### 场景 1：创建私人局并获取邀请码
- 操作：POST /api/battle/create with isPublic=false
- 验证：
  1. 返回的 CreateBattleResult 包含 inviteCode 和 inviteExpireAt
  2. inviteCode 为 4 字符大写字母数字组合
  3. inviteExpireAt = now + 172800（48小时）
  4. 数据库中 Battle 记录有相同的 inviteCode 和 inviteExpireAt

### 场景 2：私人局庄家查看自己的战局
- 操作：GET /api/battle/1
- 验证：
  1. 庄家用户 ID 匹配则返回战局详情
  2. 返回结果包含 inviteCode 和 inviteExpireAt 供庄家刷新使用

### 场景 3：非庄家用户在没有邀请码的情况下查看私人局
- 操作：GET /api/battle/1?inviteCode=（空）
- 验证：
  1. 返回 "battle not found" 错误
  2. 不会泄露战局存在的信息

### 场景 4：非庄家用户用正确邀请码查看私人局
- 操作：GET /api/battle/1?inviteCode=ABC1
- 验证：
  1. 返回战局详情（但不包含庄家专用的 inviteCode 字段）
  2. 用户可以看到战局基本信息

### 场景 5：非庄家用户用错误邀请码查看私人局
- 操作：GET /api/battle/1?inviteCode=XXXX
- 验证：
  1. 返回 "battle not found" 错误

### 场景 6：邀请码大小写不敏感
- 操作：GET /api/battle/1?inviteCode=abc1（存储的是 ABC1）
- 验证：
  1. 返回战局详情
  2. normalizeInviteCode 确保大小写不敏感

### 场景 7：庄家刷新邀请码
- 操作：GET /api/battle/1?refreshInvite=1（需要庄家权限）
- 验证：
  1. 返回新的 inviteCode（与原来不同）
  2. 返回新的 inviteExpireAt = now + 172800
  3. 旧邀请码立即失效（数据库中已更新）
  4. 非庄家用旧邀请码无法访问战局

### 场景 8：非庄家用户尝试刷新邀请码
- 操作：GET /api/battle/1?refreshInvite=1（非庄家用户）
- 验证：
  1. 返回错误："not battle banker"

### 场景 9：邀请码过期（48小时后）
- 操作：GET /api/battle/1?inviteCode=ABC1（48小时后）
- 验证：
  1. 返回 "inviteCode expired" 错误
  2. 需要庄家刷新邀请码才能让新用户进入

### 场景 10：加入私人局
- 操作：POST /api/battle/1/join with inviteCode=ABC1
- 验证：
  1. 邀请码有效则加入成功
  2. 用户成为 challenger
  3. inviteCode 校验逻辑与 GetBy 一致

## 单元测试覆盖情况

已在 battle_invite_test.go 中实现以下单元测试：

1. TestBattle_InviteCodeFormat - 邀请码格式验证
   ✓ 正确格式（大写字母数字，长度 4）
   ✓ 错误格式（特殊字符、长度不对、空值）

2. TestBattle_NormalizeInviteCode - 邀请码大小写归一化
   ✓ 小写转大写
   ✓ 首尾空白移除
   ✓ 混合大小写转大写

3. TestBattle_CanJoinWithInviteCode - 邀请码验证完整逻辑
   ✓ 完全匹配通过
   ✓ 大小写不敏感
   ✓ 格式检查
   ✓ 过期检查
   ✓ 空值检查

4. TestBattle_InviteCodeGeneration - 邀请码生成
   ✓ 随机性和格式有效性
   ✓ 多次生成不会失败

## 依赖函数

battle_service.go 中的关键函数：

- normalizeInviteCode(code) - 大小写归一化和空白移除
- isInviteCodeFormatValid(code) - 格式检查
- newRandomInviteCode() - 生成随机 4 字符邀请码
- generateUniqueInviteCode(tx, now, excludeBattleId) - 生成唯一邀请码并存储
- canJoinPrivateBattleWithInvite(b, rawInviteCode, now) - 验证邀请码有效性
- CanViewBattle(tx, userId, b, rawInviteCode, now) - 三层授权检查
- RefreshInviteCode(bankerUserId, battleId) - 庄家刷新邀请码

## 数据库字段

Battle 模型中的相关字段：
- IsPublic bool - 是否公开
- InviteCode string - 邀请码（4 字符）
- InviteCodeExpireAt int64 - 邀请码过期时间戳

迁移脚本：000025_migration_script_battle_add_invite_expire.go
- 添加 invite_code_expire_at 列
- 创建索引用于查询性能

## 已知限制

1. 当前基于数据库字段存储邀请码有效性
2. Redis 层集成为后续优化项（使用键 battle:invite:code:{CODE} 和 battle:invite:battle:{battleId}）
3. 邀请码 TTL 为 48 小时（常数 battleInviteTTLSeconds = 172800）
4. 生成重试上限 64 次（常数 battleInviteMaxRetry = 64）

*/
