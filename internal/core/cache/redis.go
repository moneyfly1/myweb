package cache

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

var (
	redisMu     sync.RWMutex
	redisClient *redis.Client
	redisReady  atomic.Bool
	ctx         = context.Background()

	// keyPrefix 多环境共用一个 Redis 时必须隔离，否则测试环境会把生产缓存冲掉。
	keyPrefix = "cboard:"

	reconnectStop chan struct{}
	reconnectOnce sync.Once

	// recoverHooks Redis 从不可用恢复时执行（例如把任务队列 worker 拉起来）。
	recoverMu    sync.Mutex
	recoverHooks []func()
)

// RegisterRecoverHook 注册"Redis 恢复可用"的回调（只注册一次即可，恢复时会全部调用）。
func RegisterRecoverHook(fn func()) {
	if fn == nil {
		return
	}
	recoverMu.Lock()
	recoverHooks = append(recoverHooks, fn)
	recoverMu.Unlock()
}

func runRecoverHooks() {
	recoverMu.Lock()
	hooks := append([]func(){}, recoverHooks...)
	recoverMu.Unlock()
	for _, fn := range hooks {
		fn()
	}
}

var (
	redisOpsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cboard_redis_ops_total",
		Help: "Redis 操作次数（按操作类型与结果）",
	}, []string{"op", "result"})
	redisCacheLookups = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cboard_cache_lookups_total",
		Help: "缓存查询命中情况（hit / miss / error）",
	}, []string{"result"})
)

func init() {
	prometheus.MustRegister(redisOpsTotal, redisCacheLookups)
}

func envMillis(key string, defMs int) time.Duration {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Millisecond
		}
	}
	return time.Duration(defMs) * time.Millisecond
}

// opTimeout 单条 Redis 命令的超时上限（读/写/删除都在这个预算内）。
const opTimeout = 800 * time.Millisecond

func envInt(key string, def int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// Key 给缓存键加统一前缀（多环境隔离）。所有直接使用 GetRedisClient() 的模块
// 都应通过它拼 key，避免一部分 key 带前缀、一部分不带。
func Key(name string) string { return keyPrefix + name }

func k(name string) string { return keyPrefix + name }

// InitRedis 初始化 Redis 连接。
//
// 连不上时不会永久放弃：会起一个后台协程每 30s 重试一次（Redis 常与面板同时重启，
// 谁先起来并不确定）。重试成功前 IsRedisEnabled() 为 false，所有缓存调用自动回落到数据库。
func InitRedis() error {
	redisMu.Lock()
	if redisClient != nil {
		_ = redisClient.Close()
		redisClient = nil
	}
	if p := strings.TrimSpace(os.Getenv("REDIS_KEY_PREFIX")); p != "" {
		keyPrefix = p
	}
	addr := strings.TrimSpace(os.Getenv("REDIS_ADDR"))
	if addr == "" {
		addr = "localhost:6379"
	}
	opts := &redis.Options{
		Addr:         addr,
		Password:     os.Getenv("REDIS_PASSWORD"),
		DB:           envInt("REDIS_DB", 0),
		ClientName:   "cboard",
		DialTimeout:  envMillis("REDIS_DIAL_TIMEOUT_MS", 2000),
		ReadTimeout:  envMillis("REDIS_TIMEOUT_MS", 800),
		WriteTimeout: envMillis("REDIS_TIMEOUT_MS", 800),
		PoolSize:     envInt("REDIS_POOL_SIZE", 20),
		MinIdleConns: envInt("REDIS_MIN_IDLE", 5),
		// -1 = 关闭命令级重试：Redis 不可用时让调用立刻失败并降级，
		// 而不是把"连不上"放大成多轮重试（既拖慢请求又刷屏日志）。
		MaxRetries: envInt("REDIS_MAX_RETRIES", -1),
		// 让每条命令受 ctx deadline 约束：否则 Redis 主机被丢包（不是立刻 ECONNREFUSED）时，
		// 一次 cache.Get 会按 拨号超时×重试 挂住好几秒，而调用方以为"缓存很快"。
		ContextTimeoutEnabled: true,
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("REDIS_TLS")), "true") {
		opts.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	client := redis.NewClient(opts)
	redisMu.Unlock()

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		redisMu.Lock()
		_ = client.Close()
		redisMu.Unlock()
		supervise()
		return err
	}

	redisMu.Lock()
	redisClient = client
	redisMu.Unlock()
	redisReady.Store(true)
	fmt.Printf("Redis 连接成功 (addr=%s db=%d prefix=%s)\n", addr, opts.DB, keyPrefix)
	runRecoverHooks()
	return nil
}

// supervise 后台探活：Redis 不可用时把 ready 置 false（所有缓存调用立即短路，
// 不再每个请求都去付一次连接超时），恢复后自动重新启用，不必重启进程。
//
// 背景：原实现"启动时 Ping 失败就永久禁用"，而 Redis 往往与面板同时被重启，
// 谁先起来并不确定 —— 于是整个进程生命周期内缓存/队列全部失效。
func supervise() {
	reconnectOnce.Do(func() {
		reconnectStop = make(chan struct{})
		go func() {
			// 退避探测：无 Redis 的部署（或长时间故障）不该每 10 秒刷一批
			// "failed to dial" 日志；恢复后再回到 10s 节奏，保证及时发现。
			backoff := 10 * time.Second
			for {
				select {
				case <-reconnectStop:
					return
				case <-time.After(backoff):
				}
				client := GetRedisClient()
				ok := false
				if client == nil {
					if err := InitRedis(); err == nil {
						log.Println("Redis 已恢复，缓存重新启用")
						ok = true
					}
				} else {
					pingCtx, cancel := context.WithTimeout(ctx, time.Second)
					err := client.Ping(pingCtx).Err()
					cancel()
					if err == nil {
						if !redisReady.Load() {
							redisReady.Store(true)
							log.Println("Redis 连接恢复，缓存重新启用")
							runRecoverHooks()
						}
						ok = true
					} else if redisReady.Swap(false) {
						log.Printf("Redis 不可用，缓存降级为直连数据库: %v", err)
					}
				}
				if ok {
					backoff = 10 * time.Second
					continue
				}
				if backoff < 5*time.Minute {
					backoff *= 2
					if backoff > 5*time.Minute {
						backoff = 5 * time.Minute
					}
				}
			}
		}()
	})
}

// markDown 命令级失败时的快速降级：只对"连不上/超时"降级，
// 业务错误（如 redis.Nil、WRONGTYPE）不降级。
func markDown(err error) {
	if err == nil {
		return
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded) {
		redisReady.Store(false)
		return
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		redisReady.Store(false)
	}
}

// RedisConn 返回当前 Redis 连接参数（供 asynq 等需要自建连接的组件复用同一套配置，
// 避免"缓存连得上、队列连不上"这种因为密码/DB 没传导致的静默失败）。
type RedisConn struct {
	Addr     string
	Password string
	DB       int
	TLS      *tls.Config
}

func CurrentRedisConn() RedisConn {
	addr := strings.TrimSpace(os.Getenv("REDIS_ADDR"))
	if addr == "" {
		addr = "localhost:6379"
	}
	c := RedisConn{Addr: addr, Password: os.Getenv("REDIS_PASSWORD"), DB: envInt("REDIS_DB", 0)}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("REDIS_TLS")), "true") {
		c.TLS = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	return c
}

// GetRedisClient 获取 Redis 客户端（未就绪时为 nil，调用方必须判空）
func GetRedisClient() *redis.Client {
	redisMu.RLock()
	defer redisMu.RUnlock()
	return redisClient
}

// IsRedisEnabled 检查 Redis 是否可用
func IsRedisEnabled() bool {
	return redisReady.Load() && GetRedisClient() != nil
}

// Close 关闭 Redis 连接并停止重连
func Close() error {
	reconnectOnce.Do(func() {}) // 避免 Close 早于 Init 时 startReconnectLoop 再起协程
	if reconnectStop != nil {
		select {
		case <-reconnectStop:
		default:
			close(reconnectStop)
		}
	}
	redisMu.Lock()
	defer redisMu.Unlock()
	redisReady.Store(false)
	if redisClient != nil {
		err := redisClient.Close()
		redisClient = nil
		return err
	}
	return nil
}

// Get 获取缓存
func Get(key string) (string, error) {
	client := GetRedisClient()
	if client == nil || !redisReady.Load() {
		return "", fmt.Errorf("redis not enabled")
	}
	opCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()
	v, err := client.Get(opCtx, k(key)).Result()
	markDown(err)
	redisOpsTotal.WithLabelValues("get", resultLabel(err)).Inc()
	if err != nil {
		if err == redis.Nil {
			redisCacheLookups.WithLabelValues("miss").Inc()
		} else {
			redisCacheLookups.WithLabelValues("error").Inc()
		}
		return "", err
	}
	redisCacheLookups.WithLabelValues("hit").Inc()
	return v, nil
}

func resultLabel(err error) string {
	if err == nil {
		return "ok"
	}
	if err == redis.Nil {
		return "nil"
	}
	return "error"
}

// Set 设置缓存
func Set(key string, value interface{}, expiration time.Duration) error {
	client := GetRedisClient()
	if client == nil || !redisReady.Load() {
		return fmt.Errorf("redis not enabled")
	}
	opCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()
	err := client.Set(opCtx, k(key), value, expiration).Err()
	markDown(err)
	redisOpsTotal.WithLabelValues("set", resultLabel(err)).Inc()
	return err
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
	client := GetRedisClient()
	if client == nil || !redisReady.Load() {
		return 0
	}
	n, err := client.Get(ctx, k(key)+":gen").Int64()
	redisOpsTotal.WithLabelValues("get", resultLabel(err)).Inc()
	if err != nil {
		return 0
	}
	return n
}

// SetIfGeneration 仅当版本号未变时写入缓存，返回是否真的写入。
// 返回 false 表示期间有人清除过缓存（数据已变更），本次写入被丢弃。
func SetIfGeneration(key string, generation int64, value interface{}, expiration time.Duration) (bool, error) {
	ms := expiration.Milliseconds()
	if ms <= 0 {
		ms = int64(time.Minute / time.Millisecond)
	}

	client := GetRedisClient()
	if client == nil || !redisReady.Load() {
		return false, nil
	}
	opCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()
	res, err := client.Eval(opCtx, setIfGenerationScript, []string{k(key)},
		strconv.FormatInt(generation, 10), value, ms).Int64()
	redisOpsTotal.WithLabelValues("setif", resultLabel(err)).Inc()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}

// DelWithGeneration 删除缓存并递增其版本号（原子），用于"数据变更后使缓存失效"。
// 比起单纯 Del，它还能让并发的陈旧回填失效。
func DelWithGeneration(keys ...string) error {
	client := GetRedisClient()
	if client == nil || !redisReady.Load() || len(keys) == 0 {
		return nil
	}
	prefixed := make([]string, len(keys))
	for i, key := range keys {
		prefixed[i] = k(key)
	}
	opCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()
	err := client.Eval(opCtx, delWithGenerationScript, prefixed).Err()
	markDown(err)
	redisOpsTotal.WithLabelValues("delgen", resultLabel(err)).Inc()
	return err
}

// Del 删除缓存
func Del(keys ...string) error {
	client := GetRedisClient()
	if client == nil || !redisReady.Load() {
		return fmt.Errorf("redis not enabled")
	}
	prefixed := make([]string, len(keys))
	for i, key := range keys {
		prefixed[i] = k(key)
	}
	opCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()
	err := client.Del(opCtx, prefixed...).Err()
	markDown(err)
	redisOpsTotal.WithLabelValues("del", resultLabel(err)).Inc()
	return err
}

// FlushAll 清空**本应用命名空间**下的缓存键。
//
// 注意不能用 FlushDB/FlushAll：同一个 Redis 可能还装着任务队列（asynq:*）、
// 活跃度 zset，甚至同一个 DB 被别的环境共用 —— 全库清空会把别人的数据一起抹掉，
// 也会把待发邮件任务删掉。这里只按本应用前缀 SCAN 删除。
func FlushAll() error {
	_, err := ClearPrefix("*")
	return err
}

// ClearPrefix 按本应用前缀删除匹配的键（pattern 不需要带前缀），返回删除数量。
func ClearPrefix(pattern string) (int, error) {
	client := GetRedisClient()
	if client == nil || !redisReady.Load() {
		return 0, fmt.Errorf("redis not enabled")
	}
	var cursor uint64
	deleted := 0
	for {
		opCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		keys, next, err := client.Scan(opCtx, cursor, k(pattern), 500).Result()
		cancel()
		if err != nil {
			markDown(err)
			return deleted, err
		}
		if len(keys) > 0 {
			// 任务队列（asynq）可能与前缀重叠，永远不删
			keep := keys[:0]
			for _, key := range keys {
				if strings.HasPrefix(strings.TrimPrefix(key, keyPrefix), "asynq:") {
					continue
				}
				keep = append(keep, key)
			}
			if len(keep) > 0 {
				opCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
				if err := client.Del(opCtx, keep...).Err(); err != nil {
					cancel()
					markDown(err)
					return deleted, err
				}
				cancel()
				deleted += len(keep)
			}
		}
		cursor = next
		if cursor == 0 {
			return deleted, nil
		}
	}
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

// subscriptionKeyID 把订阅 URL（含 token，属凭据）压成短哈希。
// 直接把 URL 拼进 key 会让 token 出现在 Redis 键名、MONITOR/slowlog/ACL 日志与运维截图里。
func subscriptionKeyID(subscriptionURL string) string {
	sum := sha256.Sum256([]byte(subscriptionURL))
	return hex.EncodeToString(sum[:8])
}

// SubscriptionConfigKey 生成订阅配置缓存 key（写入侧调用）
func SubscriptionConfigKey(subscriptionURL, format string) string {
	return fmt.Sprintf("subscription:config:%s:%s", subscriptionKeyID(subscriptionURL), format)
}

// SubscriptionConfigKeys 生成某订阅全部格式的缓存 key（清除侧调用）
func SubscriptionConfigKeys(subscriptionURL string) []string {
	keys := make([]string, 0, len(SubscriptionConfigFormats))
	for _, format := range SubscriptionConfigFormats {
		keys = append(keys, SubscriptionConfigKey(subscriptionURL, format))
	}
	return keys
}

// SubscriptionConfigGeneration 读取某订阅某格式缓存的版本号（调用方应在查库之前取，
// 写回时交给 SetSubscriptionConfigIfUnchanged 校验；期间若有人清过缓存，这次写回会被丢弃）。
func SubscriptionConfigGeneration(subscriptionURL, format string) int64 {
	return Generation(SubscriptionConfigKey(subscriptionURL, format))
}

// SetSubscriptionConfigIfUnchanged 版本号未变才写入（防陈旧回填）。
// 返回 false 表示期间缓存已被清除（数据已变更），本次写回被丢弃。
func SetSubscriptionConfigIfUnchanged(subscriptionURL, format string, generation int64, value string, ttl time.Duration) (bool, error) {
	if !IsRedisEnabled() {
		return false, nil
	}
	return SetIfGeneration(SubscriptionConfigKey(subscriptionURL, format), generation, value, ttl)
}

// ClearSubscriptionConfigCache 清除指定订阅的配置缓存（所有格式）
func ClearSubscriptionConfigCache(subscriptionURL string) error {
	return ClearSubscriptionConfigCacheWithContext(ctx, subscriptionURL)
}

// ClearSubscriptionConfigCacheWithContext 带上下文的缓存清除
func ClearSubscriptionConfigCacheWithContext(ctx context.Context, subscriptionURL string) error {
	client := GetRedisClient()
	if client == nil || !redisReady.Load() {
		return nil
	}

	keys := SubscriptionConfigKeys(subscriptionURL)
	// 用 DelWithGeneration（Lua：DEL + INCR <key>:gen）：单纯 DEL 无法阻止
	// "清缓存之前已开始生成、清完之后才写回"的陈旧条目被重新塞进缓存。
	if err := DelWithGeneration(keys...); err != nil {
		return fmt.Errorf("failed to delete subscription config cache: %w", err)
	}
	return nil
}
