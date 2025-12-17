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
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	// 确保配置已设置
	if cfg == nil {
		log.Fatal("配置未正确加载")
	}

	// 初始化数据库
	if err := database.InitDatabase(); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	db := database.GetDB()

	// 查找所有管理员账户
	var admins []models.User
	if err := db.Where("is_admin = ?", true).Find(&admins).Error; err != nil {
		log.Fatalf("查询管理员失败: %v", err)
	}

	if len(admins) == 0 {
		fmt.Println("❌ 未找到任何管理员账户")
		fmt.Println("\n💡 请先创建管理员账户:")
		fmt.Println("   go run scripts/create_admin.go")
		os.Exit(1)
	}

	fmt.Printf("✅ 找到 %d 个管理员账户:\n\n", len(admins))

	for i, admin := range admins {
		fmt.Printf("=== 管理员 #%d ===\n", i+1)
		fmt.Printf("ID:        %d\n", admin.ID)
		fmt.Printf("用户名:    %s\n", admin.Username)
		fmt.Printf("邮箱:      %s\n", admin.Email)
		fmt.Printf("IsAdmin:   %v\n", admin.IsAdmin)
		fmt.Printf("IsActive:  %v\n", admin.IsActive)
		fmt.Printf("IsVerified: %v\n", admin.IsVerified)
		fmt.Printf("密码哈希:  %s\n", admin.Password[:20]+"...")

		// 检查密码哈希格式
		if len(admin.Password) < 7 {
			fmt.Printf("⚠️  警告: 密码哈希长度异常 (%d 字符)\n", len(admin.Password))
		} else if admin.Password[:4] != "$2a$" && admin.Password[:4] != "$2b$" && admin.Password[:4] != "$2y$" {
			fmt.Printf("⚠️  警告: 密码哈希格式异常 (不是 bcrypt 格式)\n")
			fmt.Printf("   前4个字符: %s\n", admin.Password[:4])
		} else {
			fmt.Printf("✅ 密码哈希格式正确 (bcrypt)\n")
		}

		// 测试密码验证
		if len(os.Args) > 1 {
			testPassword := os.Args[1]
			fmt.Printf("\n🔐 测试密码验证:\n")
			fmt.Printf("   测试密码: %s\n", testPassword)
			if auth.VerifyPassword(testPassword, admin.Password) {
				fmt.Printf("   ✅ 密码验证成功\n")
			} else {
				fmt.Printf("   ❌ 密码验证失败\n")
			}
		}

		fmt.Println()
	}

	fmt.Println("💡 登录提示:")
	fmt.Println("   1. 可以使用用户名或邮箱登录")
	fmt.Println("   2. 确保账户状态: IsActive=true, IsVerified=true, IsAdmin=true")
	fmt.Println("   3. 如果密码验证失败，请重新创建管理员账户:")
	fmt.Println("      ADMIN_PASSWORD=your_password go run scripts/create_admin.go")
	fmt.Println()
	fmt.Println("🔍 测试密码验证:")
	fmt.Println("   go run scripts/check_admin.go your_password")
}
