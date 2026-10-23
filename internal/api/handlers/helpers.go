package handlers

import (
	"context"
	"log"
	"time"

	"cboard-go/internal/core/cache"
	"cboard-go/internal/models"
)

// handlers 包公共常量。

const (
	// TimeLayout 通用时间格式（北京时区展示/解析）
	TimeLayout = "2006-01-02 15:04:05"
	// DateFormat 仅日期格式
	DateFormat = "2006-01-02"
)

// clearSubConfigCacheAsync 异步清除单个订阅的配置缓存（后台执行，不阻塞响应）。
//
// 必须用 context.Background() 而不是 c.Request.Context()：handler 返回后请求
// context 立即 cancel，异步任务里带它清缓存会必然失败（此前 6 处重复实现有 3 处踩了这个坑）。
func clearSubConfigCacheAsync(subURL string) {
	if subURL == "" {
		return
	}
	go func(url string) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := cache.ClearSubscriptionConfigCacheWithContext(ctx, url); err != nil {
			log.Printf("failed to clear subscription config cache: %v", err)
		}
	}(subURL)
}

// clearSubConfigCachesAsync 批量版：共用一个 10s 超时，超时即停。
func clearSubConfigCachesAsync(subURLs []string) {
	if len(subURLs) == 0 {
		return
	}
	go func(urls []string) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, url := range urls {
			if url == "" {
				continue
			}
			if err := cache.ClearSubscriptionConfigCacheWithContext(ctx, url); err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("failed to clear subscription config cache: %v", err)
			}
		}
	}(subURLs)
}

// subscriptionURLs 提取订阅地址列表（供批量清缓存使用）。
func subscriptionURLs(subs []models.Subscription) []string {
	urls := make([]string, 0, len(subs))
	for _, s := range subs {
		if s.SubscriptionURL != "" {
			urls = append(urls, s.SubscriptionURL)
		}
	}
	return urls
}
