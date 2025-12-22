package main

import (
	"log"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	if cfg == nil {
		log.Fatal("配置未正确加载")
	}

	// 初始化数据库
	if err := database.InitDatabase(); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	db := database.GetDB()

	// 运行迁移以添加 location 字段
	log.Println("正在运行数据库迁移...")
	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// 检查 audit_logs 表是否有 location 字段
	var hasLocation bool
	if db.Migrator().HasColumn(&models.AuditLog{}, "location") {
		hasLocation = true
		log.Println("✅ audit_logs 表已有 location 字段")
	} else {
		log.Println("⚠️  audit_logs 表缺少 location 字段，正在添加...")
		// 手动添加字段
		if err := db.Migrator().AddColumn(&models.AuditLog{}, "location"); err != nil {
			log.Printf("添加 location 字段失败: %v", err)
		} else {
			hasLocation = true
			log.Println("✅ location 字段已添加")
		}
	}

	if hasLocation {
		log.Println("\n✅ 数据库迁移完成！")
		log.Println("现在可以运行分析脚本: go run scripts/analyze_user_distribution.go")
	} else {
		log.Println("\n⚠️  请手动运行数据库迁移或重启服务器")
	}
}

