package main

import (
	"fmt"
	"log"
	"os"

	"cboard-go/internal/core/auth"
	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run scripts/update_admin_password.go <新密码>")
		fmt.Println("示例: go run scripts/update_admin_password.go Sikeming001@")
		os.Exit(1)
	}

	newPassword := os.Args[1]
	if len(newPassword) < 6 {
		fmt.Println("❌ 错误: 密码长度至少6位")
		os.Exit(1)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	if cfg == nil {
		log.Fatal("配置未正确加载")
	}

	if err := database.InitDatabase(); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	db := database.GetDB()

	var user models.User
	err = db.Where("username = ? OR email = ?", "admin", "admin@example.com").First(&user).Error
	if err != nil {
		log.Fatalf("未找到管理员账号: %v\n请先创建管理员账号", err)
	}

	hashed, err := auth.HashPassword(newPassword)
	if err != nil {
		log.Fatalf("生成密码哈希失败: %v", err)
	}

	if err := db.Model(&user).Update("password", hashed).Error; err != nil {
		log.Fatalf("更新密码失败: %v", err)
	}

	updates := map[string]interface{}{
		"is_admin":    true,
		"is_verified": true,
		"is_active":   true,
	}
	if err := db.Model(&user).Updates(updates).Error; err != nil {
		log.Fatalf("更新管理员属性失败: %v", err)
	}

	fmt.Println("========================================")
	fmt.Println("✅ 管理员密码已更新成功！")
	fmt.Println("========================================")
	fmt.Printf("用户名: %s\n", user.Username)
	fmt.Printf("邮箱:   %s\n", user.Email)
	fmt.Printf("新密码: %s\n", newPassword)
	fmt.Println("========================================")
	fmt.Println("💡 请使用新密码登录管理员后台")
	fmt.Println("========================================")
}
