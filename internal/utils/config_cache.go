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
}
