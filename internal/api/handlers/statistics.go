package handlers

import (
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GetStatistics 获取统计数据
func GetStatistics(c *gin.Context) {
	db := database.GetDB()

	var stats struct {
		TotalUsers          int64   `json:"total_users"`
		ActiveUsers         int64   `json:"active_users"`
		TotalOrders         int64   `json:"total_orders"`
		PaidOrders          int64   `json:"paid_orders"`
		TotalRevenue        float64 `json:"total_revenue"`
		TotalSubscriptions  int64   `json:"total_subscriptions"`
		ActiveSubscriptions int64   `json:"active_subscriptions"`
		TodayRevenue        float64 `json:"today_revenue"`
		TodayOrders         int64   `json:"today_orders"`
	}

	// 用户统计
	db.Model(&models.User{}).Count(&stats.TotalUsers)
	db.Model(&models.User{}).Where("is_active = ?", true).Count(&stats.ActiveUsers)

	// 订单统计
	db.Model(&models.Order{}).Count(&stats.TotalOrders)
	db.Model(&models.Order{}).Where("status = ?", "paid").Count(&stats.PaidOrders)

	// 收入统计
	var totalRevenue sql.NullFloat64
	db.Model(&models.Order{}).Where("status = ?", "paid").Select("COALESCE(SUM(final_amount), 0)").Scan(&totalRevenue)
	if totalRevenue.Valid {
		stats.TotalRevenue = totalRevenue.Float64
	} else {
		stats.TotalRevenue = 0
	}

	// 今日统计
	today := time.Now().Format("2006-01-02")
	db.Model(&models.Order{}).Where("status = ? AND DATE(created_at) = ?", "paid", today).Count(&stats.TodayOrders)
	var todayRevenue sql.NullFloat64
	db.Model(&models.Order{}).Where("status = ? AND DATE(created_at) = ?", "paid", today).Select("COALESCE(SUM(final_amount), 0)").Scan(&todayRevenue)
	if todayRevenue.Valid {
		stats.TodayRevenue = todayRevenue.Float64
	} else {
		stats.TodayRevenue = 0
	}

	// 订阅统计
	db.Model(&models.Subscription{}).Count(&stats.TotalSubscriptions)
	db.Model(&models.Subscription{}).Where("is_active = ? AND status = ?", true, "active").Count(&stats.ActiveSubscriptions)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetRevenueChart 获取收入图表数据
func GetRevenueChart(c *gin.Context) {
	_ = c.DefaultQuery("days", "30")

	// 按日期分组的收入统计
	type RevenueStat struct {
		Date    string  `json:"date"`
		Revenue float64 `json:"revenue"`
	}

	var stats []RevenueStat
	days := 30 // 默认30天
	if daysParam := c.Query("days"); daysParam != "" {
		fmt.Sscanf(daysParam, "%d", &days)
	}

	// 使用原生 SQL 查询（兼容 SQLite 和 MySQL）
	db := database.GetDB()
	var rows *sql.Rows
	var err error

	// 检测数据库类型（简化处理，使用 SQLite 语法）
	rows, err = db.Raw(`
		SELECT DATE(created_at) as date, COALESCE(SUM(final_amount), 0) as revenue
		FROM orders 
		WHERE status = ? AND created_at >= datetime('now', '-' || ? || ' days')
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`, "paid", days).Rows()

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var stat RevenueStat
			rows.Scan(&stat.Date, &stat.Revenue)
			stats = append(stats, stat)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetUserStatistics 获取用户统计
func GetUserStatistics(c *gin.Context) {
	db := database.GetDB()

	var stats struct {
		TotalUsers        int64 `json:"total_users"`
		NewUsersToday     int64 `json:"new_users_today"`
		NewUsersThisWeek  int64 `json:"new_users_this_week"`
		NewUsersThisMonth int64 `json:"new_users_this_month"`
		VerifiedUsers     int64 `json:"verified_users"`
		UnverifiedUsers   int64 `json:"unverified_users"`
	}

	db.Model(&models.User{}).Count(&stats.TotalUsers)
	db.Model(&models.User{}).Where("is_verified = ?", true).Count(&stats.VerifiedUsers)
	db.Model(&models.User{}).Where("is_verified = ?", false).Count(&stats.UnverifiedUsers)

	today := time.Now().Format("2006-01-02")
	db.Model(&models.User{}).Where("DATE(created_at) = ?", today).Count(&stats.NewUsersToday)

	// 本周统计（从周一开始）
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekStart := now.AddDate(0, 0, -weekday+1)
	weekStartStr := weekStart.Format("2006-01-02")
	db.Model(&models.User{}).Where("DATE(created_at) >= ?", weekStartStr).Count(&stats.NewUsersThisWeek)

	// 本月统计
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthStartStr := monthStart.Format("2006-01-02")
	db.Model(&models.User{}).Where("DATE(created_at) >= ?", monthStartStr).Count(&stats.NewUsersThisMonth)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}
