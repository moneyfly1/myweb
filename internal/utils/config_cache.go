package utils

import (
	"sync"
	"time"

	"cboard-go/internal/models"

	"gorm.io/gorm"
)

// 配置项短 TTL 缓存：避免热路径（注册、改密、创建用户、建订阅等）每次请求重复查 SystemConfig 表。
// 配置修改频率极低，30 秒缓存 + 更新时显式失效即可。

const settingCacheTTL = 30 * time.Second

type cachedSetting struct {
	value     string
	fetchedAt time.Time
}

var (
	settingCache   = make(map[string]cachedSetting)
	settingCacheMu sync.RWMutex
)

func settingCacheKey(category, key string) string {
	return category + "\x00" + key
}

// GetCachedSetting 读取 SystemConfig 单键值；TTL 内命中缓存，未命中回源并填充。
//
// 返回前统一做一层「敏感值还原」（见 ResolveConfigSecret）：线上出现过配置里存
// "enc:v1:密文"的情况（由外部系统/旧密钥写入），若原样返回，调用方会把密文当
// Token/密码用，报出来的是 `401 Bad credentials` 这类误导性错误。
// 解不开时返回错误（调用方能看到可读原因），同时值按空处理，避免把密文当明文用。
func GetCachedSetting(db *gorm.DB, key, category string) (string, error) {
	ck := settingCacheKey(category, key)

	settingCacheMu.RLock()
	if c, ok := settingCache[ck]; ok && time.Since(c.fetchedAt) < settingCacheTTL {
		settingCacheMu.RUnlock()
		return c.value, nil
	}
	settingCacheMu.RUnlock()

	var config models.SystemConfig
	if err := db.Where("key = ? AND category = ?", key, category).First(&config).Error; err != nil {
		return "", err
	}

	// 缓存里存**还原后**的值：TTL 内多次读取不必重复解密
	value, resolveErr := ResolveConfigSecret(config.Value)
	if resolveErr != nil {
		LogWarn("配置项 %s/%s 无法还原为明文: %v", category, key, resolveErr)
		value = ""
	}

	settingCacheMu.Lock()
	settingCache[ck] = cachedSetting{value: value, fetchedAt: time.Now()}
	settingCacheMu.Unlock()
	if resolveErr != nil {
		return "", resolveErr
	}
	return value, nil
}

// InvalidateAllSettingCache 全量失效（批量设置更新时调用）。
func InvalidateAllSettingCache() {
	settingCacheMu.Lock()
	settingCache = make(map[string]cachedSetting)
	settingCacheMu.Unlock()
	// 公开设置快照同源，一并失效，否则后台改完配置前台最多 30 秒不生效
	InvalidatePublicSettingsCache()
}

// ---------------- 计数缓存 ----------------
//
// 后台列表的 total 都靠 COUNT(*) 得到，而 COUNT 在大表（audit_logs、
// subscription_logs）上是第二次全量扫描 —— 它是列表页里最贵的一条查询，
// 却只用于显示"共 N 条"。这里给"总数"加一层很短的 TTL 缓存：
// 用户翻页时不必每次都重扫，新增日志最多滞后十几秒（可接受）。

const defaultCountCacheTTL = 15 * time.Second

type cachedCount struct {
	value     int64
	fetchedAt time.Time
}

var (
	countCache   = make(map[string]cachedCount)
	countCacheMu sync.RWMutex
)

// CachedCount 返回 key 对应的计数；TTL 内命中缓存，否则调用 fn 取数并填充。
// fn 出错时不写缓存，错误原样返回。
func CachedCount(key string, ttl time.Duration, fn func() (int64, error)) (int64, error) {
	if ttl <= 0 {
		ttl = defaultCountCacheTTL
	}
	countCacheMu.RLock()
	if c, ok := countCache[key]; ok && time.Since(c.fetchedAt) < ttl {
		countCacheMu.RUnlock()
		return c.value, nil
	}
	countCacheMu.RUnlock()

	v, err := fn()
	if err != nil {
		return 0, err
	}
	countCacheMu.Lock()
	if len(countCache) > 512 { // 简单上限，避免过滤条件组合把内存撑爆
		countCache = make(map[string]cachedCount)
	}
	countCache[key] = cachedCount{value: v, fetchedAt: time.Now()}
	countCacheMu.Unlock()
	return v, nil
}

// InvalidateCountCache 清空计数缓存（写操作后可调用，让 total 立刻反映新增/删除）。
func InvalidateCountCache() {
	countCacheMu.Lock()
	countCache = make(map[string]cachedCount)
	countCacheMu.Unlock()
}

// ---------------- 公开设置快照缓存 ----------------
//
// GET /api/v1/settings/public-settings 是匿名高频接口（每个页面首次加载都会请求），
// 内容来自 system_configs 全表筛选 + 大量类型转换，属于"读多写极少"。
// 这里缓存 30 秒；配置更新走 InvalidateAllSettingCache（本文件）时一并清掉。

type publicSettingsSnapshot struct {
	data      map[string]interface{}
	fetchedAt time.Time
}

var (
	publicSettingsMu   sync.RWMutex
	publicSettingsSnap *publicSettingsSnapshot
)

const publicSettingsTTL = 30 * time.Second

// GetPublicSettingsCache 取公开设置快照（未命中返回 nil,false）。
func GetPublicSettingsCache() (map[string]interface{}, bool) {
	publicSettingsMu.RLock()
	defer publicSettingsMu.RUnlock()
	if publicSettingsSnap == nil || time.Since(publicSettingsSnap.fetchedAt) >= publicSettingsTTL {
		return nil, false
	}
	return publicSettingsSnap.data, true
}

// SetPublicSettingsCache 写入公开设置快照。
func SetPublicSettingsCache(data map[string]interface{}) {
	publicSettingsMu.Lock()
	publicSettingsSnap = &publicSettingsSnapshot{data: data, fetchedAt: time.Now()}
	publicSettingsMu.Unlock()
}

// InvalidatePublicSettingsCache 失效公开设置快照。
func InvalidatePublicSettingsCache() {
	publicSettingsMu.Lock()
	publicSettingsSnap = nil
	publicSettingsMu.Unlock()
}
