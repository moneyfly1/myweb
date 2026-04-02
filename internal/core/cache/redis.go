package cache

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	redisClient  *redis.Client
	ctx          = context.Background()
	redisEnabled bool
)

// InitRedis 初始化 Redis 连接
func InitRedis() error {
	// 从环境变量读取配置
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB := 0 // 默认使用 DB 0

	redisClient = redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		Password:     redisPassword,
		DB:           redisDB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	// 测试连接
	if err := redisClient.Ping(ctx).Err(); err != nil {
		fmt.Printf("Redis 连接失败: %v (将禁用缓存功能)\n", err)
		redisEnabled = false
		return err
	}

	redisEnabled = true
	fmt.Println("Redis 连接成功")
	return nil
}

// GetRedisClient 获取 Redis 客户端
func GetRedisClient() *redis.Client {
	return redisClient
}

// IsRedisEnabled 检查 Redis 是否可用
func IsRedisEnabled() bool {
	return redisEnabled && redisClient != nil
}

// Close 关闭 Redis 连接
func Close() error {
	if redisClient != nil {
		return redisClient.Close()
	}
	return nil
}

// Get 获取缓存
func Get(key string) (string, error) {
	if !IsRedisEnabled() {
		return "", fmt.Errorf("redis not enabled")
	}
	return redisClient.Get(ctx, key).Result()
}

// Set 设置缓存
func Set(key string, value interface{}, expiration time.Duration) error {
	if !IsRedisEnabled() {
		return fmt.Errorf("redis not enabled")
	}
	return redisClient.Set(ctx, key, value, expiration).Err()
}

// Del 删除缓存
func Del(keys ...string) error {
	if !IsRedisEnabled() {
		return fmt.Errorf("redis not enabled")
	}
	return redisClient.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func Exists(keys ...string) (int64, error) {
	if !IsRedisEnabled() {
		return 0, fmt.Errorf("redis not enabled")
	}
	return redisClient.Exists(ctx, keys...).Result()
}

// FlushAll 清空所有缓存（谨慎使用）
func FlushAll() error {
	if !IsRedisEnabled() {
		return fmt.Errorf("redis not enabled")
	}
	return redisClient.FlushDB(ctx).Err()
}

// ClearSubscriptionConfigCache 清除指定订阅的配置缓存（所有格式）
func ClearSubscriptionConfigCache(subscriptionURL string) error {
	return ClearSubscriptionConfigCacheWithContext(ctx, subscriptionURL)
}

// ClearSubscriptionConfigCacheWithContext 带上下文的缓存清除
func ClearSubscriptionConfigCacheWithContext(ctx context.Context, subscriptionURL string) error {
	if !IsRedisEnabled() {
		return nil
	}
	
	keys := []string{
		fmt.Sprintf("subscription:config:%s:clash", subscriptionURL),
		fmt.Sprintf("subscription:config:%s:base64", subscriptionURL),
	}
	
	if err := redisClient.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("failed to delete subscription config cache: %w", err)
	}
	return nil
}
