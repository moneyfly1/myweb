package main

import (
	"errors"
	"fmt"
	"log"

	"cboard-go/internal/core/auth"
	"cboard-go/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 简单的初始化脚本：确保存在管理员账号 admin/admin123
func main() {
	db, err := gorm.Open(sqlite.Open("cboard.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}

	username := "admin"
	email := "admin@example.com"
	password := "admin123"

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
	fmt.Printf("  密码:   %s\n", password)
}
