package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"cboard-go/internal/core/auth"
	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"

	"gorm.io/gorm"
)

func main() {
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

	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	db := database.GetDB()

	username := os.Getenv("ADMIN_USERNAME")
	if username == "" {
		username = "admin"
		log.Println("提示: 未设置 ADMIN_USERNAME 环境变量，使用默认用户名 'admin'")
	}

	email := os.Getenv("ADMIN_EMAIL")
	if email == "" {
		email = "admin@example.com"
		log.Println("提示: 未设置 ADMIN_EMAIL 环境变量，使用默认邮箱 'admin@example.com'")
	}

	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		if os.Getenv("ENV") == "production" {
			log.Fatalf("错误: 生产环境必须设置 ADMIN_PASSWORD 环境变量")
		}
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
			var existingAdmin models.User
			if err := db.Where("is_admin = ?", true).First(&existingAdmin).Error; err == nil {
				updates := map[string]interface{}{
					"username":    username,
					"email":       email,
					"password":    hashed,
					"is_admin":    true,
					"is_verified": true,
					"is_active":   true,
				}
				if err := db.Model(&existingAdmin).Updates(updates).Error; err != nil {
					log.Fatalf("更新管理员失败: %v", err)
				}
				fmt.Printf("管理员已更新: 用户名=%s 邮箱=%s\n", username, email)
			} else {
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
			}
		} else {
			log.Fatalf("查询用户失败: %v", result.Error)
		}
	} else {
		updates := map[string]interface{}{
			"username":    username,
			"email":       email,
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

	fmt.Println("\n✅ 管理员账户准备就绪！")
	fmt.Println("\n📋 账号信息：")
	fmt.Printf("  用户名: %s\n", username)
	fmt.Printf("  邮箱:   %s\n", email)
	if os.Getenv("ADMIN_PASSWORD") == "" {
		fmt.Printf("  密码:   %s (默认密码，请尽快修改！)\n", password)
	} else {
		fmt.Printf("  密码:   [已从环境变量读取]\n")
	}

	fmt.Println("\n🔍 验证信息：")
	fmt.Printf("  密码哈希长度: %d 字符\n", len(hashed))
	if len(hashed) >= 4 {
		fmt.Printf("  哈希格式: %s\n", hashed[:4])
		if hashed[:4] == "$2a$" || hashed[:4] == "$2b$" || hashed[:4] == "$2y$" {
			fmt.Printf("  ✅ 密码哈希格式正确 (bcrypt)\n")
		} else {
			fmt.Printf("  ⚠️  警告: 密码哈希格式异常\n")
		}
	}

	if auth.VerifyPassword(password, hashed) {
		fmt.Printf("  ✅ 密码验证测试通过\n")
	} else {
		fmt.Printf("  ❌ 密码验证测试失败！请检查密码哈希\n")
	}

	fmt.Println("\n💡 登录提示：")
	fmt.Println("  1. 访问管理员登录页面: /admin/login")
	fmt.Println("  2. 可以使用用户名或邮箱登录")
	fmt.Println("  3. 如果无法登录，可以:")
	fmt.Println("     - 修改密码: go run scripts/update_admin_password.go <新密码>")
	fmt.Println("     - 解锁账户: go run scripts/unlock_user.go admin")
}
