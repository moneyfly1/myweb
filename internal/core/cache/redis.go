package cache

import (
	"context"
	"fmt"
	"os"
	"strconv"
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

// ==========================================
// 防"陈旧回填"（stale repopulate）
// ==========================================
//
// 背景：读路径的典型写法是"缓存未命中 → 查库 → 写缓存"。若期间管理员改了数据
// 并清除了缓存，读路径随后会把清除前查到的旧数据写回，旧数据就一直被缓存到 TTL
// 到期（套餐价格、支付方式开关都属于这种，曾出现改成新价格后前台仍显示旧价格）。
//
// 做法：每个缓存键带一个版本号 <key>:gen。清除缓存时版本 +1；
// 读路径在查库前先取版本号，写回时用 Lua 原子校验版本未变，变了就放弃这次写入。

// setIfGenerationScript 原子校验版本号后写入
const setIfGenerationScript = `
local cur = redis.call('GET', KEYS[1] .. ':gen')
if (cur or '0') ~= ARGV[1] then return 0 end
redis.call('SET', KEYS[1], ARGV[2], 'PX', ARGV[3])
return 1
`

// delWithGenerationScript 原子删除缓存并递增版本号
const delWithGenerationScript = `
for i = 1, #KEYS do
  redis.call('DEL', KEYS[i])
  redis.call('INCR', KEYS[i] .. ':gen')
end
return 1
`

// Generation 读取缓存键的版本号（键不存在时为 0）。
// 调用方应在查询数据源之前调用，并把结果传给 SetIfGeneration。
func Generation(key string) int64 {
	if !IsRedisEnabled() {
		return 0
	}
	n, err := redisClient.Get(ctx, key+":gen").Int64()
	if err != nil {
		return 0
	}
	return n
}

// SetIfGeneration 仅当版本号未变时写入缓存，返回是否真的写入。
// 返回 false 表示期间有人清除过缓存（数据已变更），本次写入被丢弃。
func SetIfGeneration(key string, generation int64, value interface{}, expiration time.Duration) (bool, error) {
	if !IsRedisEnabled() {
		return false, nil
	}

	ms := expiration.Milliseconds()
	if ms <= 0 {
		ms = int64(time.Minute / time.Millisecond)
	}

	res, err := redisClient.Eval(ctx, setIfGenerationScript, []string{key},
		strconv.FormatInt(generation, 10), value, ms).Int64()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}

// DelWithGeneration 删除缓存并递增其版本号（原子），用于"数据变更后使缓存失效"。
// 比起单纯 Del，它还能让并发的陈旧回填失效。
func DelWithGeneration(keys ...string) error {
	if !IsRedisEnabled() || len(keys) == 0 {
		return nil
	}
	return redisClient.Eval(ctx, delWithGenerationScript, keys).Err()
}

// Del 删除缓存
func Del(keys ...string) error {
	if !IsRedisEnabled() {
		return fmt.Errorf("redis not enabled")
	}
	return redisClient.Del(ctx, keys...).Err()
}

// FlushAll 清空所有缓存（谨慎使用）
func FlushAll() error {
	if !IsRedisEnabled() {
		return fmt.Errorf("redis not enabled")
	}
	return redisClient.FlushDB(ctx).Err()
}

// SubscriptionConfigFormats 订阅配置缓存的全部格式。
// 写入侧（config_update）与清除侧必须共用这一份清单：
// 历史上两侧各自手写 key，清除侧只删了 clash 与 base64，
// 导致设备踢下线 / 改订阅后，ssr、v2rayN 客户端仍能拿到变更前的节点。
var SubscriptionConfigFormats = []string{
	"clash",
	"base64",
	"ssr",
	"base64_v2rayn",
	"ssr_v2rayn",
}

// SubscriptionConfigKey 生成订阅配置缓存 key（写入侧调用）
func SubscriptionConfigKey(subscriptionURL, format string) string {
	return fmt.Sprintf("subscription:config:%s:%s", subscriptionURL, format)
}

// SubscriptionConfigKeys 生成某订阅全部格式的缓存 key（清除侧调用）
func SubscriptionConfigKeys(subscriptionURL string) []string {
	keys := make([]string, 0, len(SubscriptionConfigFormats))
	for _, format := range SubscriptionConfigFormats {
		keys = append(keys, SubscriptionConfigKey(subscriptionURL, format))
	}
	return keys
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

	keys := SubscriptionConfigKeys(subscriptionURL)
	if err := redisClient.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("failed to delete subscription config cache: %w", err)
	}
	return nil
}
