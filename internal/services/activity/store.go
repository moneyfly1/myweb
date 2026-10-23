package activity

import (
	"context"
	"strconv"
	"sync"
	"time"

	"cboard-go/internal/core/cache"

	"github.com/redis/go-redis/v9"
)

// 存储实现的单一实例（由 GetStore 按 Redis 可用性自动切换）
var (
	storeMu   sync.Mutex
	storeImpl store
)

func itoa(v uint) string    { return strconv.FormatUint(uint64(v), 10) }
func itoa64(v int64) string { return strconv.FormatInt(v, 10) }

func atoiDef(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func opCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 2*time.Second)
}

// ---------------- Redis 实现 ----------------

type redisStore struct{}

func (r *redisStore) zadd(key string, member uint, score int64) {
	client := cache.GetRedisClient()
	if client == nil {
		return
	}
	ctx, cancel := opCtx()
	defer cancel()
	_ = client.ZAdd(ctx, key, redis.Z{Score: float64(score), Member: itoa(member)}).Err()
}

func (r *redisStore) zrem(key string, member uint) {
	client := cache.GetRedisClient()
	if client == nil {
		return
	}
	ctx, cancel := opCtx()
	defer cancel()
	_ = client.ZRem(ctx, key, itoa(member)).Err()
}

func (r *redisStore) membersSince(key string, since int64) []uint {
	client := cache.GetRedisClient()
	if client == nil {
		return nil
	}
	ctx, cancel := opCtx()
	defer cancel()
	vals, err := client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: strconv.FormatInt(since, 10),
		Max: "+inf",
	}).Result()
	if err != nil {
		return nil
	}
	out := make([]uint, 0, len(vals))
	for _, v := range vals {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			out = append(out, uint(n))
		}
	}
	return out
}

func (r *redisStore) zcountSince(key string, since int64) int64 {
	client := cache.GetRedisClient()
	if client == nil {
		return 0
	}
	ctx, cancel := opCtx()
	defer cancel()
	n, err := client.ZCount(ctx, key, strconv.FormatInt(since, 10), "+inf").Result()
	if err != nil {
		return 0
	}
	return n
}

func (r *redisStore) scoreOf(key string, member uint) (int64, bool) {
	client := cache.GetRedisClient()
	if client == nil {
		return 0, false
	}
	ctx, cancel := opCtx()
	defer cancel()
	score, err := client.ZScore(ctx, key, itoa(member)).Result()
	if err != nil {
		return 0, false
	}
	return int64(score), true
}

func (r *redisStore) prune(key string, cutoff int64) int64 {
	client := cache.GetRedisClient()
	if client == nil {
		return 0
	}
	ctx, cancel := opCtx()
	defer cancel()
	n, err := client.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(cutoff, 10)).Result()
	if err != nil {
		return 0
	}
	return n
}

func (r *redisStore) setStr(key, val string, ttl time.Duration) {
	client := cache.GetRedisClient()
	if client == nil {
		return
	}
	ctx, cancel := opCtx()
	defer cancel()
	_ = client.Set(ctx, key, val, ttl).Err()
}

func (r *redisStore) getStr(key string) (string, bool) {
	client := cache.GetRedisClient()
	if client == nil {
		return "", false
	}
	ctx, cancel := opCtx()
	defer cancel()
	v, err := client.Get(ctx, key).Result()
	if err != nil {
		return "", false
	}
	return v, true
}

func (r *redisStore) setNX(key string, ttl time.Duration) bool {
	client := cache.GetRedisClient()
	if client == nil {
		return true
	}
	ctx, cancel := opCtx()
	defer cancel()
	ok, err := client.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		// Redis 出错时宁可放行：写库节流失效，总比丢掉信号好
		return true
	}
	return ok
}

func (r *redisStore) del(keys ...string) {
	client := cache.GetRedisClient()
	if client == nil || len(keys) == 0 {
		return
	}
	ctx, cancel := opCtx()
	defer cancel()
	_ = client.Del(ctx, keys...).Err()
}

// ---------------- 内存实现（无 Redis 时兜底） ----------------

type memKV struct {
	val string
	exp time.Time
}

type memStore struct {
	mu sync.Mutex
	zs map[string]map[uint]int64
	kv map[string]memKV
}

func newMemStore() *memStore {
	return &memStore{zs: make(map[string]map[uint]int64), kv: make(map[string]memKV)}
}

func (m *memStore) zadd(key string, member uint, score int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	set, ok := m.zs[key]
	if !ok {
		set = make(map[uint]int64)
		m.zs[key] = set
	}
	set[member] = score
}

func (m *memStore) zrem(key string, member uint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.zs[key], member)
}

func (m *memStore) membersSince(key string, since int64) []uint {
	m.mu.Lock()
	defer m.mu.Unlock()
	set := m.zs[key]
	out := make([]uint, 0, len(set))
	for member, score := range set {
		if score >= since {
			out = append(out, member)
		}
	}
	return out
}

func (m *memStore) zcountSince(key string, since int64) int64 {
	var n int64
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, score := range m.zs[key] {
		if score >= since {
			n++
		}
	}
	return n
}

func (m *memStore) scoreOf(key string, member uint) (int64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	score, ok := m.zs[key][member]
	return score, ok
}

func (m *memStore) prune(key string, cutoff int64) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	removed := int64(0)
	for member, score := range m.zs[key] {
		if score < cutoff {
			delete(m.zs[key], member)
			removed++
		}
	}
	return removed
}

func (m *memStore) setStr(key, val string, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.kv[key] = memKV{val: val, exp: time.Now().Add(ttl)}
}

func (m *memStore) getStr(key string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.kv[key]
	if !ok {
		return "", false
	}
	if !e.exp.IsZero() && time.Now().After(e.exp) {
		delete(m.kv, key)
		return "", false
	}
	return e.val, true
}

func (m *memStore) setNX(key string, ttl time.Duration) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.kv[key]; ok && (e.exp.IsZero() || time.Now().Before(e.exp)) {
		return false
	}
	m.kv[key] = memKV{val: "1", exp: time.Now().Add(ttl)}
	return true
}

func (m *memStore) del(keys ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.kv, k)
	}
}
