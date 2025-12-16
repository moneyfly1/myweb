package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"cboard-go/internal/core/auth"
	"cboard-go/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 简单的初始化脚本：确保存在管理员账号
// 密码从环境变量 ADMIN_PASSWORD 读取，如果未设置则使用默认值（仅用于开发环境）
func main() {
	db, err := gorm.Open(sqlite.Open("cboard.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}

	username := "admin"
	email := "admin@example.com"

	// 从环境变量读取密码，如果未设置则使用默认值（仅开发环境）
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		// 检查是否为生产环境
		if os.Getenv("ENV") == "production" {
			log.Fatalf("错误: 生产环境必须设置 ADMIN_PASSWORD 环境变量")
		}
		// 开发环境使用默认密码，但给出警告
		password = "admin123"
		log.Println("警告: 未设置 ADMIN_PASSWORD 环境变量，使用默认密码 'admin123'")
		log.Println("警告: 生产环境请务必设置强密码！")
	}

	hashed, err := auth.HashPassword(password)
	if err != nil {
		log.Fatalf("生成密码哈希失败: %v", err)
	}

	var user models.User
	result := db.Where("username = ? OR email = ?", username, email).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			user = models.User{
				Username:   username,
				Email:      email,
				Password:   hashed,
				IsAdmin:    true,
				IsVerified: true,
				IsActive:   true,
			}
			if err := db.Create(&user).Error; err != nil {
				log.Fatalf("创建管理员失败: %v", err)
			}
			fmt.Printf("管理员已创建: 用户名=%s 邮箱=%s\n", username, email)
		} else {
			log.Fatalf("查询用户失败: %v", result.Error)
		}
	} else {
		updates := map[string]interface{}{
			"password":    hashed,
			"is_admin":    true,
			"is_verified": true,
			"is_active":   true,
		}
		if err := db.Model(&user).Updates(updates).Error; err != nil {
			log.Fatalf("更新管理员失败: %v", err)
		}
		fmt.Printf("管理员已更新: 用户名=%s 邮箱=%s\n", username, email)
	}

	fmt.Println("管理员账户准备就绪。账号信息：")
	fmt.Printf("  用户名: %s\n", username)
	fmt.Printf("  邮箱:   %s\n", email)
	if os.Getenv("ADMIN_PASSWORD") == "" {
		fmt.Printf("  密码:   %s (默认密码，请尽快修改！)\n", password)
	} else {
		fmt.Printf("  密码:   [已从环境变量读取]\n")
	}
}
