package geoip

import (
	"cboard-go/internal/core/cache"
	"cboard-go/internal/core/netutil"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// GetLocationWithCache 带缓存的地理位置查询
func GetLocationWithCache(ipAddress string) sql.NullString {
	// 本地/内网 IP 判定统一走 netutil.Normalize + IsPrivateOrReservedString
	// （此前只挡三个字面量，导致 192.168.x.x 等内网地址被当成公网继续外发查询）
	ipAddress = netutil.Normalize(ipAddress, false)
	if ipAddress == "" {
		return sql.NullString{}
	}
	if ipAddress == "127.0.0.1" {
		return sql.NullString{String: "本地", Valid: true}
	}
	if netutil.IsPrivateOrReservedString(ipAddress) {
		return sql.NullString{String: "内网", Valid: true}
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
			if err := cache.Set(cacheKey, cacheValue, 24*time.Hour); err != nil {
				log.Printf("failed to set geoip cache: %v", err)
			}
		}(ipAddress, location)
	}

	return location
}

// GetLocationSimpleWithCache 带缓存的简单地理位置查询，返回**人类可读文本**（如"中国, 杭州"）。
//
// 历史缺陷：此前直接返回 GetLocationWithCache(...).String，而该值是 JSON 串
// （{"country":"中国","city":"杭州"}），却被日志列表当作展示文本返回给前端，
// 于回退路径下把原始 JSON 显示给了管理员。这里统一解析为可读文本。
func GetLocationSimpleWithCache(ipAddress string) string {
	location := GetLocationWithCache(ipAddress)
	if !location.Valid || location.String == "" {
		return ""
	}
	return LocationDisplay(location.String)
}

// LocationDisplay 把落库的 location 值（JSON / "本地" / "内网" / "国家, 城市"）
// 统一转成人类可读文本。geoip 包不能 import utils（会与 utils→geoip 形成循环依赖），
// 因此在此提供唯一实现，供本包与上层复用。
func LocationDisplay(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if raw == "本地" || raw == "内网" || raw == "localhost" {
		return raw
	}
	if strings.HasPrefix(raw, "{") {
		var loc struct {
			Country string `json:"country"`
			City    string `json:"city"`
			Region  string `json:"region"`
		}
		if err := json.Unmarshal([]byte(raw), &loc); err == nil && loc.Country != "" {
			return LocationDisplayOf(loc.Country, loc.City, loc.Region)
		}
	}
	return raw
}

// LocationDisplayOf 由结构化字段拼装展示文本 —— 全站唯一的"国家, 城市"拼接规则。
// 此前后端 4 处、前端 3 处各写一遍，其中统计模块用了 " - " 分隔符，
// 导致同一地区在列表显示"中国, 杭州"、在地区统计显示"中国 - 杭州"。
func LocationDisplayOf(country, city, region string) string {
	country = strings.TrimSpace(country)
	if country == "" {
		return ""
	}
	if city = strings.TrimSpace(city); city != "" {
		return country + ", " + city
	}
	if region = strings.TrimSpace(region); region != "" {
		return country + ", " + region
	}
	return country
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

	// 有界并发预热：限制同时进行的地理查询数量，避免 goroutine 风暴
	const maxConcurrent = 10
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	for _, ip := range ipAddresses {
		ip := ip
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			GetLocationWithCache(ip)
		}()
	}
	wg.Wait()
}

// GetLocationWithFallbackCached 带缓存的详细地理位置查询（包含 Fallback）
func GetLocationWithFallbackCached(ipAddress string) (*LocationInfo, error) {
	// 本地/内网 IP 必须先拦截：此分支之后的 Fallback 链会请求第三方接口，
	// 若不拦截会把内网地址（192.168/10/172.16 等）外发给 ping0.cc 等外部服务
	ipAddress = netutil.Normalize(ipAddress, false)
	if ipAddress == "" {
		return &LocationInfo{}, nil
	}
	if ipAddress == "127.0.0.1" {
		return &LocationInfo{Country: "本地"}, nil
	}
	if netutil.IsPrivateOrReservedString(ipAddress) {
		return &LocationInfo{Country: "内网"}, nil
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
			if err := cache.Set(cacheKey, cacheValue, 24*time.Hour); err != nil {
				log.Printf("failed to set geoip detail cache: %v", err)
			}
		}(ipAddress, location, err)
	}

	return location, err
}
