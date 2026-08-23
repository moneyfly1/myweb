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

	settingCacheMu.Lock()
	settingCache[ck] = cachedSetting{value: config.Value, fetchedAt: time.Now()}
	settingCacheMu.Unlock()
	return config.Value, nil
}

// InvalidateSettingCache 使指定配置项的缓存失效（配置更新后调用）。
func InvalidateSettingCache(key, category string) {
	settingCacheMu.Lock()
	delete(settingCache, settingCacheKey(category, key))
	settingCacheMu.Unlock()
}

// InvalidateAllSettingCache 全量失效（批量设置更新时调用）。
func InvalidateAllSettingCache() {
	settingCacheMu.Lock()
	settingCache = make(map[string]cachedSetting)
	settingCacheMu.Unlock()
}
