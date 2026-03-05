package main

import (
	"fmt"
	"log"
	"os"

	"cboard-go/internal/core/cache"
	"cboard-go/internal/core/config"
	"cboard-go/internal/services/cache_service"
)

func main() {
	// 加载配置
	if _, err := config.LoadConfig(); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化 Redis
	if err := cache.InitRedis(); err != nil {
		log.Fatalf("Redis 初始化失败: %v", err)
	}
	defer cache.Close()

	fmt.Println("========================================")
	fmt.Println("清除所有 Redis 缓存")
	fmt.Println("========================================")
	fmt.Println("⚠️  警告：此操作将清空所有缓存数据！")
	fmt.Print("确认继续？(yes/no): ")

	var confirm string
	fmt.Scanln(&confirm)

	if confirm != "yes" {
		fmt.Println("操作已取消")
		os.Exit(0)
	}

	if err := cache_service.FlushAllCache(); err != nil {
		log.Fatalf("清除缓存失败: %v", err)
	}

	fmt.Println("✅ 所有缓存已清除")
	fmt.Println("建议：重启服务后缓存将自动预热")
}
