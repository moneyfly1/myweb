package main

import (
	"fmt"
	"os"
	"strings"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run scripts/unlock_user.go <用户名或邮箱>")
		fmt.Println("示例: go run scripts/unlock_user.go admin")
		fmt.Println("示例: go run scripts/unlock_user.go admin@example.com")
		fmt.Println("示例: go run scripts/unlock_user.go user@example.com")
		os.Exit(1)
	}

	identifier := strings.TrimSpace(os.Args[1])
	if identifier == "" {
		fmt.Println("❌ 错误: 用户名或邮箱不能为空")
		os.Exit(1)
	}

	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("❌ 配置加载失败: %v\n", err)
		os.Exit(1)
	}

	// 确保配置已设置
	if cfg == nil {
		fmt.Println("❌ 配置未正确加载")
		os.Exit(1)
	}

	// 初始化数据库
	if err := database.InitDatabase(); err != nil {
		fmt.Printf("❌ 数据库连接失败: %v\n", err)
		os.Exit(1)
	}

	db := database.GetDB()

	// 查找用户（支持管理员和普通用户）
	var user models.User
	query := db.Model(&models.User{})
	if strings.Contains(identifier, "@") {
		query = query.Where("email = ?", identifier)
	} else {
		query = query.Where("username = ?", identifier)
	}

	if err := query.First(&user).Error; err != nil {
		fmt.Printf("❌ 未找到用户账户: %s\n", identifier)
		fmt.Println("\n💡 提示:")
		fmt.Println("   1. 请确认用户名或邮箱是否正确")
		fmt.Println("   2. 请检查数据库连接是否正常")
		os.Exit(1)
	}

	userType := "普通用户"
	if user.IsAdmin {
		userType = "管理员"
	}

	fmt.Printf("✅ 找到用户账户:\n")
	fmt.Printf("   ID: %d\n", user.ID)
	fmt.Printf("   用户名: %s\n", user.Username)
	fmt.Printf("   邮箱: %s\n", user.Email)
	fmt.Printf("   类型: %s\n", userType)
	fmt.Printf("   当前状态: IsActive=%v, IsVerified=%v\n", user.IsActive, user.IsVerified)

	// 检查登录失败记录
	var failedAttempts int64
	db.Model(&models.LoginAttempt{}).
		Where("(username = ? OR username = ?) AND success = ?", user.Username, user.Email, false).
		Count(&failedAttempts)

	fmt.Printf("\n📊 登录失败记录统计:\n")
	fmt.Printf("   - 失败记录数: %d 条\n", failedAttempts)

	// 显示最近的失败记录
	var recentAttempts []models.LoginAttempt
	db.Where("(username = ? OR username = ?) AND success = ?", user.Username, user.Email, false).
		Order("created_at DESC").
		Limit(5).
		Find(&recentAttempts)

	if len(recentAttempts) > 0 {
		fmt.Printf("   - 最近的失败记录:\n")
		for i, attempt := range recentAttempts {
			ipAddr := ""
			if attempt.IPAddress.Valid {
				ipAddr = attempt.IPAddress.String
			}
			fmt.Printf("     %d. %s (IP: %s, 时间: %s)\n",
				i+1,
				attempt.Username,
				ipAddr,
				attempt.CreatedAt.Format("2006-01-02 15:04:05"))
		}
	}

	// 清除所有登录失败记录（包括成功和失败的）
	result := db.Where("username = ? OR username = ?", user.Username, user.Email).
		Delete(&models.LoginAttempt{})

	fmt.Printf("\n🗑️  清除登录记录: %d 条（包括成功和失败的记录）\n", result.RowsAffected)

	// 确保用户是激活状态
	user.IsActive = true
	user.IsVerified = true

	if err := db.Save(&user).Error; err != nil {
		fmt.Printf("❌ 解锁失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✅ 用户账户已成功解锁!")
	fmt.Println("\n📝 操作摘要:")
	fmt.Printf("   - 清除了 %d 条登录记录\n", result.RowsAffected)
	fmt.Printf("   - 账户状态: IsActive=true, IsVerified=true\n")

	fmt.Println("\n⚠️  重要提示:")
	fmt.Println("   1. 如果仍然无法登录，可能是 IP 地址被速率限制器锁定")
	fmt.Println("   2. 速率限制器基于 IP 地址，锁定时间为 15 分钟")
	fmt.Println("   3. 解决方案:")
	fmt.Println("      a) 等待 15 分钟后重试")
	fmt.Println("      b) 更换 IP 地址（使用 VPN 或移动网络）")
	fmt.Println("      c) 重启服务器以清除内存中的速率限制记录")
	fmt.Println("      d) 使用其他未锁定的 IP 地址登录")

	fmt.Println("\n💡 验证步骤:")
	fmt.Println("   1. 确认账户状态: IsActive=true, IsVerified=true")
	fmt.Println("   2. 确认密码正确")
	if user.IsAdmin {
		fmt.Println("   3. 如果是管理员，可以使用: go run scripts/update_admin_password.go <新密码> 重置密码")
	}
	fmt.Println("   4. 如果 IP 被锁定，等待 15 分钟或更换 IP")
	fmt.Println("   5. 清除浏览器缓存和 Cookie 后重试登录")
}
