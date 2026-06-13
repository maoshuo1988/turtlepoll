package cache

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	// RedisClient 全局 Redis 客户端
	RedisClient *redis.Client
)

// InitRedis 初始化 Redis 客户端
// 配置：localhost:6379，无密码，数据库 0
func InitRedis() error {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:6379",
		Password:     "",
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := RedisClient.Ping(ctx).Err(); err != nil {
		slog.Error("Redis 连接失败", "error", err)
		return err
	}

	slog.Info("Redis 连接成功", "addr", "127.0.0.1:6379")
	return nil
}

// CloseRedis 关闭 Redis 连接
func CloseRedis() error {
	if RedisClient != nil {
		return RedisClient.Close()
	}
	return nil
}

// Set 设置键值对（带 TTL）
func Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if RedisClient == nil {
		return nil // Redis 未初始化时优雅降级
	}
	return RedisClient.Set(ctx, key, value, ttl).Err()
}

// Get 获取键值
func Get(ctx context.Context, key string) (string, error) {
	if RedisClient == nil {
		return "", nil // Redis 未初始化时返回空
	}
	return RedisClient.Get(ctx, key).Result()
}

// Exists 检查键是否存在
func Exists(ctx context.Context, key string) (bool, error) {
	if RedisClient == nil {
		return false, nil
	}
	result, err := RedisClient.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// Delete 删除键
func Delete(ctx context.Context, keys ...string) error {
	if RedisClient == nil {
		return nil
	}
	return RedisClient.Del(ctx, keys...).Err()
}

// SetNX 原子性地设置键值对（仅当键不存在时）
// 返回 true 表示设置成功，false 表示键已存在
func SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	if RedisClient == nil {
		return true, nil // Redis 未初始化时当做成功
	}
	result, err := RedisClient.SetNX(ctx, key, value, ttl).Result()
	return result, err
}
