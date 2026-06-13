# Redis 邀请码管理 - 实现完成

## 概述

已成功实现 Redis 作为邀请码的存储和验证层。系统采用**分层验证**策略，Redis 作为快速验证层，数据库作为可靠性回退层。

## 实现架构

### 1. Redis 客户端管理 - `internal/pkg/cache/redis.go`

**关键特性：**
- 连接池配置：PoolSize=10, MinIdleConns=2
- 超时配置：DialTimeout=5s, ReadTimeout=3s, WriteTimeout=3s
- 优雅降级：Redis 不可用时，系统仍可正常运行（所有操作返回 nil 或 false）
- 基础操作：Set, Get, Exists, Delete, SetNX

**初始化：**
```go
func InitRedis() error  // 127.0.0.1:6379, DB=0
func CloseRedis() error
```

**关键函数：**
- `Set(ctx, key, value, ttl)` - 设置键值对
- `Get(ctx, key)` - 获取值
- `Delete(ctx, ...keys)` - 删除键
- `SetNX(ctx, key, value, ttl)` - 原子化设置（存在时返回 false）

### 2. 邀请码生成与存储流程

**位置：** `generateUniqueInviteCode()` 函数

**流程：**
1. 生成随机 4 字符邀请码（`[0-9A-Z]`）
2. 查询数据库检查唯一性（最多重试 64 次）
3. 找到唯一邮请码后，**异步存入 Redis**：
   - Key: `battle:invite:code:{CODE}`
   - Value: `battleId`
   - TTL: 172800 秒（48 小时）
4. 同时存储到数据库的 `battle.invite_code` 和 `battle.invite_code_expire_at` 字段

**特点：**
- Redis 存储是异步的，不阻塞邀请码生成和战局创建
- 若 Redis 不可用或失败，系统继续工作（数据库仍是可靠数据源）

### 3. 邀请码验证流程

**位置：** `canJoinPrivateBattleWithInvite()` 函数

**验证策略（两层）：**

**第一层 - Redis 验证（快速路径）：**
```
1. 格式检查：4 字符大小写不敏感 [0-9A-Z]
2. 查询 Redis: battle:invite:code:{CODE}
3. 验证返回的 battleId 与当前战局 ID 匹配
4. 若匹配则返回 true（有效）
```

**第二层 - 数据库验证（可靠性回退）：**
```
若 Redis 查询失败或不存在，回退到：
1. 比对 battle.invite_code（大小写不敏感）
2. 检查 battle.invite_code_expire_at > now
3. 若都符合则验证通过
```

**返回值：**
- Redis 中有效 → 直接通过
- Redis 不可用但数据库有效 → 通过（并考虑异步同步到 Redis）
- 两者都无效 → 拒绝

### 4. 邀请码刷新流程

**位置：** `RefreshInviteCode()` 函数

**步骤：**
1. 验证当前用户是庄家
2. **异步删除旧邀请码的 Redis key**（不阻塞更新）
   - Key: `battle:invite:code:{OLD_CODE}`
3. 生成新邀请码（同时存入 Redis 和数据库）
4. 更新数据库 `battle.invite_code` 和 `battle.invite_code_expire_at`

**特点：**
- 旧邀请码**立即失效**（Redis key 被删除）
- 新邀请码立即生效
- 即使 Redis 删除失败，48 小时后过期（TTL 保证）

## Redis 键设计

### 键值映射

| 键 | 值 | TTL | 用途 |
|----|-----|-----|------|
| `battle:invite:code:{CODE}` | `battleId` | 172800s (48h) | 邀请码→战局映射，快速查询 |

### 示例

创建私人战局：
```
生成邀请码：ABC1
Redis: SET battle:invite:code:ABC1 "123" EX 172800
```

刷新邀请码：
```
旧邀请码：ABC1 → Redis: DEL battle:invite:code:ABC1
新邀请码：XYZ9 → Redis: SET battle:invite:code:XYZ9 "123" EX 172800
```

## 项目集成

### 1. 依赖包

已有：`github.com/go-redis/redis/v8` (间接依赖，现已直接使用)

### 2. 初始化流程

**在 `internal/install/install.go` 的 `InitOthers()` 函数中：**

```go
func InitOthers() error {
    // ... 其他初始化 ...
    
    // 初始化 Redis（可选，若连接失败仅记录警告）
    if err := cache.InitRedis(); err != nil {
        slog.Warn("Redis 初始化失败，邀请码管理将使用数据库存储", "error", err)
    }
    return nil
}
```

**特点：**
- Redis 初始化失败不会中断应用启动
- 系统自动降级到纯数据库模式
- 日志记录可观察性

### 3. 优雅降级

当 Redis 不可用时：

```go
// 在 redis.go 中的所有函数都检查 RedisClient != nil
func Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
    if RedisClient == nil {
        return nil // 直接返回 nil
    }
    return RedisClient.Set(ctx, key, value, ttl).Err()
}
```

结果：
- ✅ CreateBattle：生成邀请码成功（仅数据库）
- ✅ GetBy：验证邀请码成功（回退到数据库）
- ✅ RefreshInviteCode：刷新邀请码成功（仅数据库）
- ⚠️ 性能：没有 Redis 加速的快速验证路径

## 文件清单

### 新增文件
- **[internal/pkg/cache/redis.go](internal/pkg/cache/redis.go)** - Redis 客户端和基础操作

### 修改文件
- **[internal/services/battle_service.go](internal/services/battle_service.go)**
  - 添加导入：`context`, `time`, `cache`
  - 修改 `generateUniqueInviteCode()` - 异步存储到 Redis
  - 修改 `canJoinPrivateBattleWithInvite()` - 两层验证策略
  - 修改 `RefreshInviteCode()` - 异步删除旧 Redis key
  - 新增 `storeInviteCodeToRedis()` - 异步存储函数
  - 新增 `validateInviteCodeWithRedis()` - Redis 验证函数
  - 新增 `deleteInviteCodeFromRedis()` - 异步删除函数

- **[internal/install/install.go](internal/install/install.go)**
  - 添加导入：`cache`
  - 修改 `InitOthers()` - 添加 Redis 初始化调用

## 验证清单

- ✅ Redis 客户端包创建
- ✅ 邀请码生成时存储到 Redis
- ✅ 邀请码验证时优先查 Redis（快速路径）
- ✅ 邀请码验证时回退到数据库（可靠性保证）
- ✅ 邀请码刷新时删除旧 Redis key（立即失效）
- ✅ 优雅降级：Redis 不可用时系统仍可运行
- ✅ 所有单元测试通过（4 个测试套件）
- ✅ 编译无错误
- ✅ 上下文超时控制（所有 Redis 操作 2-3s 超时）

## 性能优化点

1. **异步操作**：
   - 邀请码存储到 Redis 不阻塞战局创建
   - 删除旧邀请码不阻塞邀请码刷新

2. **快速验证**：
   - Redis 命中时直接返回，避免数据库查询
   - 平均验证时间从数据库查询（几十毫秒）降低到 Redis GET（几毫秒）

3. **可靠回退**：
   - Redis 不可用不影响功能
   - 数据库总是最终数据源

4. **超时保护**：
   - 所有 Redis 操作有上下文超时（2-3s）
   - 避免 Redis 超时导致的应用卡住

## 后续优化方向

1. **Redis 配置化**
   - 将 127.0.0.1:6379 改为配置文件驱动
   - 支持哨兵/集群模式

2. **Redis 集群支持**
   - 支持多个 Redis 实例
   - 支持高可用部署

3. **监控和告警**
   - Redis 连接数监控
   - 邀请码命中率统计
   - 性能指标导出

4. **邀请码使用统计**
   - 记录邀请码被使用次数
   - 记录邀请码最后使用时间

## 部署说明

### 前置条件
- Redis 6.0+ 安装在 127.0.0.1:6379
- 无认证密码（如需密码，修改 `cache/redis.go` 的 `Options.Password`）

### 部署步骤
1. 更新代码
2. 编译：`go build ./...`
3. 启动应用：应用启动时会自动初始化 Redis
4. 若 Redis 不可用，应用会记录警告但继续运行

### 验证
```bash
# 观察应用日志
# 成功：INFO "Redis 连接成功"
# 失败：WARN "Redis 初始化失败，邀请码管理将使用数据库存储"

# 测试邀请码生成
curl -X POST http://localhost:8080/api/battle/create \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"title": "Test", "bankerSide": "A", "challengerSide": "B", "stakeAmount": 100, "settleTime": 1718300000, "isPublic": false}'

# 应返回
# {
#   "battle": { ... },
#   "inviteCode": "ABC1",
#   "inviteExpireAt": 1718280000
# }
```

---

**完成时间**：2026-06-13
**状态**：✅ 完成并验证
**测试状态**：✅ 4/4 单元测试通过
