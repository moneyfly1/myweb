package geoip

import (
	"cboard-go/internal/core/cache"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// GetLocationWithCache 带缓存的地理位置查询
func GetLocationWithCache(ipAddress string) sql.NullString {
	// 快速检查本地/内网 IP
	if ipAddress == "127.0.0.1" || ipAddress == "::1" || ipAddress == "localhost" {
		return sql.NullString{String: "本地", Valid: true}
	}

	// 尝试从 Redis 缓存获取
	if cache.IsRedisEnabled() {
		cacheKey := fmt.Sprintf("geoip:%s", ipAddress)
		if cached, err := cache.Get(cacheKey); err == nil && cached != "" {
			// 缓存命中
			if cached == "NULL" {
				return sql.NullString{Valid: false}
			}
			return sql.NullString{String: cached, Valid: true}
		}
	}

	// 缓存未命中，查询 GeoIP 数据库
	location := GetLocationString(ipAddress)

	// 异步写入缓存（不阻塞响应）
	if cache.IsRedisEnabled() {
		go func(ip string, loc sql.NullString) {
			cacheKey := fmt.Sprintf("geoip:%s", ip)
			cacheValue := "NULL"
			if loc.Valid {
				cacheValue = loc.String
			}
			// 缓存 24 小时
			cache.Set(cacheKey, cacheValue, 24*time.Hour)
		}(ipAddress, location)
	}

	return location
}

// GetLocationSimpleWithCache 带缓存的简单地理位置查询
func GetLocationSimpleWithCache(ipAddress string) string {
	location := GetLocationWithCache(ipAddress)
	if location.Valid {
		return location.String
	}
	return ""
}

// ClearLocationCache 清除指定 IP 的缓存
func ClearLocationCache(ipAddress string) error {
	if !cache.IsRedisEnabled() {
		return fmt.Errorf("redis not enabled")
	}
	cacheKey := fmt.Sprintf("geoip:%s", ipAddress)
	return cache.Del(cacheKey)
}

// WarmupCache 预热缓存（批量查询常见 IP）
func WarmupCache(ipAddresses []string) {
	if !cache.IsRedisEnabled() {
		return
	}

	for _, ip := range ipAddresses {
		go func(ipAddr string) {
			GetLocationWithCache(ipAddr)
		}(ip)
		time.Sleep(10 * time.Millisecond) // 避免过载
	}
}

// GetLocationWithFallbackCached 带缓存的详细地理位置查询（包含 Fallback）
func GetLocationWithFallbackCached(ipAddress string) (*LocationInfo, error) {
	// 快速检查本地/内网 IP
	if ipAddress == "127.0.0.1" || ipAddress == "::1" || ipAddress == "localhost" {
		return &LocationInfo{Country: "本地"}, nil
	}

	// 尝试从 Redis 缓存获取
	if cache.IsRedisEnabled() {
		cacheKey := fmt.Sprintf("geoip:detail:%s", ipAddress)
		if cached, err := cache.Get(cacheKey); err == nil && cached != "" {
			// 缓存命中
			if cached == "NULL" {
				return nil, fmt.Errorf("location not found in cache")
			}
			var loc LocationInfo
			if err := json.Unmarshal([]byte(cached), &loc); err == nil {
				return &loc, nil
			}
		}
	}

	// 缓存未命中，查询 GeoIP 数据库（带 Fallback）
	location, err := GetLocationWithFallback(ipAddress)

	// 异步写入缓存（不阻塞响应）
	if cache.IsRedisEnabled() {
		go func(ip string, loc *LocationInfo, queryErr error) {
			cacheKey := fmt.Sprintf("geoip:detail:%s", ip)
			cacheValue := "NULL"
			if queryErr == nil && loc != nil {
				if data, err := json.Marshal(loc); err == nil {
					cacheValue = string(data)
				}
			}
			// 缓存 24 小时
			cache.Set(cacheKey, cacheValue, 24*time.Hour)
		}(ipAddress, location, err)
	}

	return location, err
}
