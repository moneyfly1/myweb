package config_update

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"cboard-go/internal/core/cache"
)

// CacheService 订阅配置缓存服务
type CacheService struct{}

// 缓存代际统一放在 Redis 侧（<key>:gen，见 core/cache.SetIfGeneration）。
//
// 历史教训：这里曾是**进程内** atomic.Int64。单实例下它就挡不住陈旧回填（写回时才盖章，
// 拿到的已是清理后的新代际）；多实例下更糟——每个实例的计数器天生不同，
// A 写入的条目 B 一读就判定"代际不匹配"并删除，两个实例互相清缓存，命中率≈0。
const systemNodesCacheKey = "nodes:system:all"

// systemNodesCacheEntry 系统节点缓存条目（带代际标记）
type systemNodesCacheEntry struct {
	Gen   int64        `json:"gen"`
	Nodes []*ProxyNode `json:"nodes"`
}

// subscriptionConfigCacheEntry 订阅配置缓存条目（带代际标记）
type subscriptionConfigCacheEntry struct {
	Gen    int64  `json:"gen"`
	Config string `json:"config"`
}

// GetSystemNodesCache 获取系统节点缓存
func (cs *CacheService) GetSystemNodesCache() ([]*ProxyNode, bool) {
	if !cache.IsRedisEnabled() {
		return nil, false
	}

	cached, err := cache.Get(systemNodesCacheKey)
	if err != nil || cached == "" {
		return nil, false
	}
	gen := cache.Generation(systemNodesCacheKey)

	var entry systemNodesCacheEntry
	if err := json.Unmarshal([]byte(cached), &entry); err != nil || entry.Gen != gen {
		// 代际不匹配（或旧格式数据）：视为未命中，并删除过期条目
		_ = cache.Del(systemNodesCacheKey)
		return nil, false
	}

	return entry.Nodes, true
}

// SystemNodesGeneration 查库前调用：拿到当前代际，写回时交给 SetSystemNodesCacheIfUnchanged。
func (cs *CacheService) SystemNodesGeneration() int64 {
	return cache.Generation(systemNodesCacheKey)
}

// SetSystemNodesCache 设置系统节点缓存（写回时再取代际，仍比不校验强，但存在窗口；
// 想彻底避免陈旧回填请用 SetSystemNodesCacheIfUnchanged）
func (cs *CacheService) SetSystemNodesCache(nodes []*ProxyNode) error {
	if !cache.IsRedisEnabled() {
		return nil
	}
	return cs.SetSystemNodesCacheIfUnchanged(nodes, cache.Generation(systemNodesCacheKey))
}

// SetSystemNodesCacheIfUnchanged 仅当代际未变时写入（代际应在查库前取得）
func (cs *CacheService) SetSystemNodesCacheIfUnchanged(nodes []*ProxyNode, generation int64) error {
	if !cache.IsRedisEnabled() {
		return nil
	}
	// #nosec G117 - Password field is proxy node password, not user credential
	data, err := json.Marshal(systemNodesCacheEntry{Gen: generation, Nodes: nodes}) // #nosec G117
	if err != nil {
		return err
	}
	_, err = cache.SetIfGeneration(systemNodesCacheKey, generation, string(data), time.Hour)
	return err
}

// ClearSystemNodesCache 清除系统节点缓存（DelWithGeneration：删除 + 代际自增，原子）
func (cs *CacheService) ClearSystemNodesCache() error {
	if !cache.IsRedisEnabled() {
		return nil
	}
	return cache.DelWithGeneration(systemNodesCacheKey)
}

// ClearCustomNodesCache 清除用户自定义节点缓存
// 注意：专线节点在订阅生成时直接查 DB（appendCustomNodes），不依赖本缓存；
// 因此不需要递增全局 cacheGen（那会使所有用户订阅缓存无意义失效）。
func (cs *CacheService) ClearCustomNodesCache(userID uint) error {
	if !cache.IsRedisEnabled() {
		return nil
	}
	cacheKey := fmt.Sprintf("nodes:custom:user:%d", userID)
	return cache.Del(cacheKey)
}

// ClearAllSubscriptionCache 清除所有订阅配置缓存（节点变更时调用）
func (cs *CacheService) ClearAllSubscriptionCache() error {
	if !cache.IsRedisEnabled() {
		return nil
	}

	// 清除系统节点缓存（同时自增其代际）
	if err := cache.DelWithGeneration(systemNodesCacheKey); err != nil {
		log.Printf("failed to delete nodes cache: %v", err)
	}

	// 清除所有订阅配置缓存（节点变更时全量失效）
	if _, err := cache.ClearPrefix("subscription:config:*"); err != nil {
		log.Printf("failed to delete subscription config cache keys: %v", err)
	}
	return nil
}

// GetSubscriptionConfigCache 获取订阅配置缓存
func (cs *CacheService) GetSubscriptionConfigCache(subscriptionURL, format string) (string, bool) {
	if !cache.IsRedisEnabled() {
		return "", false
	}

	cacheKey := cache.SubscriptionConfigKey(subscriptionURL, format)
	cached, err := cache.Get(cacheKey)
	if err != nil || cached == "" {
		return "", false
	}
	gen := cache.SubscriptionConfigGeneration(subscriptionURL, format)

	var entry subscriptionConfigCacheEntry
	if err := json.Unmarshal([]byte(cached), &entry); err != nil || entry.Gen != gen {
		_ = cache.Del(cacheKey)
		return "", false
	}

	return entry.Config, true
}

// SubscriptionConfigGeneration 生成前调用：拿当前代际，写回时交给 SetSubscriptionConfigCacheIfUnchanged。
func (cs *CacheService) SubscriptionConfigGeneration(subscriptionURL, format string) int64 {
	return cache.SubscriptionConfigGeneration(subscriptionURL, format)
}

// SetSubscriptionConfigCache 设置订阅配置缓存（智能TTL；代际现取，见上）
func (cs *CacheService) SetSubscriptionConfigCache(subscriptionURL, format, config string, ttl time.Duration) error {
	return cs.SetSubscriptionConfigCacheIfUnchanged(subscriptionURL, format, config, ttl,
		cache.SubscriptionConfigGeneration(subscriptionURL, format))
}

// SetSubscriptionConfigCacheIfUnchanged 代际未变才写入（防"清缓存后的旧配置被重新写回"）
func (cs *CacheService) SetSubscriptionConfigCacheIfUnchanged(subscriptionURL, format, config string, ttl time.Duration, generation int64) error {
	if !cache.IsRedisEnabled() || ttl <= 0 {
		return nil
	}
	// #nosec G117 - config contains proxy node links, not user credential
	data, err := json.Marshal(subscriptionConfigCacheEntry{Gen: generation, Config: config}) // #nosec G117
	if err != nil {
		return err
	}
	_, err = cache.SetSubscriptionConfigIfUnchanged(subscriptionURL, format, generation, string(data), ttl)
	return err
}

// ClearSubscriptionConfigCache 清除指定订阅的配置缓存
func (cs *CacheService) ClearSubscriptionConfigCache(subscriptionURL string) error {
	if !cache.IsRedisEnabled() {
		return nil
	}

	// 清除该订阅的所有格式缓存：key 清单来自 cache.SubscriptionConfigKeys，
	// 与写入侧共用一份，避免再次出现"漏删某些格式导致旧节点仍可见"
	if err := cache.ClearSubscriptionConfigCache(subscriptionURL); err != nil {
		log.Printf("failed to clear subscription config cache: %v", err)
	}

	return nil
}
