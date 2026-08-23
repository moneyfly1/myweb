package main

import (
	"fmt"
	"os"
	"strings"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
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

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("❌ 配置加载失败: %v\n", err)
		os.Exit(1)
	}

	if cfg == nil {
		fmt.Println("❌ 配置未正确加载")
		os.Exit(1)
	}

	if err := database.InitDatabase(); err != nil {
		fmt.Printf("❌ 数据库连接失败: %v\n", err)
		os.Exit(1)
	}

	db := database.GetDB()

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

	var failedAttempts int64
	db.Model(&models.LoginAttempt{}).
		Where("(username = ? OR username = ?) AND success = ?", user.Username, user.Email, false).
		Count(&failedAttempts)

	fmt.Printf("\n📊 登录失败记录统计:\n")
	fmt.Printf("   - 失败记录数: %d 条\n", failedAttempts)

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

	result := db.Where("username = ? OR username = ?", user.Username, user.Email).
		Delete(&models.LoginAttempt{})

	fmt.Printf("\n🗑️  清除登录记录: %d 条（包括成功和失败的记录）\n", result.RowsAffected)

	var loginHistories []models.LoginHistory
	db.Where("user_id = ? AND ip_address IS NOT NULL", user.ID).
		Order("login_time DESC").
		Limit(10).
		Find(&loginHistories)

	var auditLogs []models.AuditLog
	db.Where("user_id = ? AND ip_address IS NOT NULL AND action_type LIKE ?",
		user.ID, "security_login%").
		Order("created_at DESC").
		Limit(10).
		Find(&auditLogs)

	ipSet := make(map[string]bool)
	for _, history := range loginHistories {
		if history.IPAddress.Valid && history.IPAddress.String != "" {
			ipSet[history.IPAddress.String] = true
		}
	}
	for _, log := range auditLogs {
		if log.IPAddress.Valid && log.IPAddress.String != "" {
			ipSet[log.IPAddress.String] = true
		}
	}

	for _, attempt := range recentAttempts {
		if attempt.IPAddress.Valid && attempt.IPAddress.String != "" {
			ipSet[attempt.IPAddress.String] = true
		}
	}

	ipCount := 0
	for ip := range ipSet {
		middleware.ResetLoginAttempt(ip)
		ipCount++
	}

	if ipCount > 0 {
		fmt.Printf("🔓 清除IP速率限制: %d 个IP地址\n", ipCount)
		fmt.Printf("   - 已清除的IP地址:\n")
		ipList := make([]string, 0, len(ipSet))
		for ip := range ipSet {
			ipList = append(ipList, ip)
		}
		for i, ip := range ipList {
			if i < 10 { // 最多显示10个IP
				fmt.Printf("     %d. %s\n", i+1, ip)
			}
		}
		if len(ipList) > 10 {
			fmt.Printf("     ... 还有 %d 个IP地址\n", len(ipList)-10)
		}
	} else {
		fmt.Printf("ℹ️  未找到相关的IP地址记录\n")
	}

	user.IsActive = true
	user.IsVerified = true

	if err := db.Save(&user).Error; err != nil {
		fmt.Printf("❌ 解锁失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✅ 用户账户已成功解锁!")
	fmt.Println("\n📝 操作摘要:")
	fmt.Printf("   - 清除了 %d 条登录记录\n", result.RowsAffected)
	if ipCount > 0 {
		fmt.Printf("   - 清除了 %d 个IP地址的速率限制\n", ipCount)
	}
	fmt.Printf("   - 账户状态: IsActive=true, IsVerified=true\n")

	fmt.Println("\n⚠️  重要提示:")
	if ipCount > 0 {
		fmt.Println("   ✅ 已自动清除相关IP地址的速率限制")
		fmt.Println("   ✅ 用户现在应该可以正常登录了")
	} else {
		fmt.Println("   ℹ️  未找到相关的IP地址记录，可能该用户没有登录历史")
		fmt.Println("   ℹ️  如果用户仍然无法登录，请检查:")
		fmt.Println("      a) 确认密码是否正确")
		fmt.Println("      b) 确认账户状态是否为激活状态")
		fmt.Println("      c) 如果IP被锁定，等待15分钟后重试或更换IP地址")
	}

	fmt.Println("\n💡 验证步骤:")
	fmt.Println("   1. 确认账户状态: IsActive=true, IsVerified=true")
	fmt.Println("   2. 确认密码正确")
	if user.IsAdmin {
		fmt.Println("   3. 如果是管理员，可以使用: go run scripts/admin_tool.go <新密码> 重置密码")
	}
	fmt.Println("   4. 如果 IP 被锁定，等待 15 分钟或更换 IP")
	fmt.Println("   5. 清除浏览器缓存和 Cookie 后重试登录")
}
