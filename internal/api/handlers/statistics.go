package handlers

import (
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/geoip"
	"cboard-go/internal/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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

	// 收入统计（使用公共函数）
	stats.TotalRevenue = utils.CalculateTotalRevenue(db, "paid")

	// 今日统计
	today := time.Now().Format("2006-01-02")
	db.Model(&models.Order{}).Where("status = ? AND DATE(created_at) = ?", "paid", today).Count(&stats.TodayOrders)
	stats.TodayRevenue = utils.CalculateTodayRevenue(db, "paid")

	// 订阅统计
	db.Model(&models.Subscription{}).Count(&stats.TotalSubscriptions)
	// 统计活跃订阅数（状态为active且is_active为true，并且未过期）
	now := time.Now()
	db.Model(&models.Subscription{}).
		Where("is_active = ?", true).
		Where("(status = ? OR status = '' OR status IS NULL)", "active").
		Where("expire_time > ?", now).
		Count(&stats.ActiveSubscriptions)

	// 生成用户统计列表
	var inactiveUsers int64
	db.Model(&models.User{}).Where("is_active = ?", false).Count(&inactiveUsers)
	var verifiedUsers int64
	db.Model(&models.User{}).Where("is_verified = ?", true).Count(&verifiedUsers)
	var unverifiedUsers int64
	db.Model(&models.User{}).Where("is_verified = ?", false).Count(&unverifiedUsers)

	userStatsList := []gin.H{
		{
			"name":       "总用户数",
			"value":      stats.TotalUsers,
			"percentage": 100,
		},
		{
			"name":  "活跃用户",
			"value": stats.ActiveUsers,
			"percentage": func() float64 {
				if stats.TotalUsers > 0 {
					return float64(stats.ActiveUsers) / float64(stats.TotalUsers) * 100
				}
				return 0
			}(),
		},
		{
			"name":  "未激活用户",
			"value": inactiveUsers,
			"percentage": func() float64 {
				if stats.TotalUsers > 0 {
					return float64(inactiveUsers) / float64(stats.TotalUsers) * 100
				}
				return 0
			}(),
		},
		{
			"name":  "已验证用户",
			"value": verifiedUsers,
			"percentage": func() float64 {
				if stats.TotalUsers > 0 {
					return float64(verifiedUsers) / float64(stats.TotalUsers) * 100
				}
				return 0
			}(),
		},
		{
			"name":  "未验证用户",
			"value": unverifiedUsers,
			"percentage": func() float64 {
				if stats.TotalUsers > 0 {
					return float64(unverifiedUsers) / float64(stats.TotalUsers) * 100
				}
				return 0
			}(),
		},
	}

	// 生成订阅统计列表
	var expiredSubscriptions int64
	db.Model(&models.Subscription{}).
		Where("expire_time <= ?", now).
		Count(&expiredSubscriptions)
	var inactiveSubscriptions int64
	db.Model(&models.Subscription{}).
		Where("is_active = ?", false).
		Count(&inactiveSubscriptions)

	subscriptionStatsList := []gin.H{
		{
			"name":       "总订阅数",
			"value":      stats.TotalSubscriptions,
			"percentage": 100,
		},
		{
			"name":  "活跃订阅",
			"value": stats.ActiveSubscriptions,
			"percentage": func() float64 {
				if stats.TotalSubscriptions > 0 {
					return float64(stats.ActiveSubscriptions) / float64(stats.TotalSubscriptions) * 100
				}
				return 0
			}(),
		},
		{
			"name":  "已过期订阅",
			"value": expiredSubscriptions,
			"percentage": func() float64 {
				if stats.TotalSubscriptions > 0 {
					return float64(expiredSubscriptions) / float64(stats.TotalSubscriptions) * 100
				}
				return 0
			}(),
		},
		{
			"name":  "未激活订阅",
			"value": inactiveSubscriptions,
			"percentage": func() float64 {
				if stats.TotalSubscriptions > 0 {
					return float64(inactiveSubscriptions) / float64(stats.TotalSubscriptions) * 100
				}
				return 0
			}(),
		},
	}

	// 获取最近活动（最近10条订单）
	var recentOrders []models.Order
	db.Preload("User").Order("created_at DESC").Limit(10).Find(&recentOrders)
	recentActivitiesList := make([]gin.H, 0)
	for _, order := range recentOrders {
		amount := order.Amount
		if order.FinalAmount.Valid {
			amount = order.FinalAmount.Float64
		}
		activityType := "primary"
		if order.Status == "paid" {
			activityType = "success"
		} else if order.Status == "pending" {
			activityType = "warning"
		} else if order.Status == "cancelled" {
			activityType = "danger"
		}
		recentActivitiesList = append(recentActivitiesList, gin.H{
			"id":          order.ID,
			"type":        activityType,
			"description": fmt.Sprintf("订单 %s - 用户 %s", order.OrderNo, order.User.Username),
			"amount":      amount,
			"status":      order.Status,
			"time":        order.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		// 直接返回统计数据
		"total_users":          stats.TotalUsers,
		"active_users":         stats.ActiveUsers,
		"total_orders":         stats.TotalOrders,
		"paid_orders":          stats.PaidOrders,
		"total_revenue":        stats.TotalRevenue,
		"total_subscriptions":  stats.TotalSubscriptions,
		"active_subscriptions": stats.ActiveSubscriptions,
		"today_revenue":        stats.TodayRevenue,
		"today_orders":         stats.TodayOrders,
		"overview": gin.H{
			"totalUsers":          stats.TotalUsers,
			"activeSubscriptions": stats.ActiveSubscriptions,
			"totalOrders":         stats.TotalOrders,
			"totalRevenue":        stats.TotalRevenue,
		},
		// 用户统计列表
		"userStats": userStatsList,
		// 订阅统计列表
		"subscriptionStats": subscriptionStatsList,
		// 最近活动列表
		"recentActivities": recentActivitiesList,
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

	db := database.GetDB()
	var rows *sql.Rows
	var err error
	rows, err = db.Raw(`
		SELECT DATE(created_at) as date, COALESCE(SUM(
			CASE 
				WHEN final_amount IS NOT NULL AND final_amount != 0 THEN final_amount
				ELSE amount
			END
		), 0) as revenue
		FROM orders 
		WHERE status = ? AND created_at >= datetime('now', '-' || ? || ' days')
		GROUP BY DATE(created_at)
		ORDER BY date ASC
	`, "paid", days).Rows()

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var stat RevenueStat
			rows.Scan(&stat.Date, &stat.Revenue)
			stats = append(stats, stat)
		}
	}

	// 转换为前端期望的格式
	labels := make([]string, 0)
	data := make([]float64, 0)
	for _, stat := range stats {
		labels = append(labels, stat.Date)
		data = append(data, stat.Revenue)
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"labels": labels,
		"data":   data,
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

	utils.SuccessResponse(c, http.StatusOK, "", stats)
}

// GetRegionStats 获取地区统计
func GetRegionStats(c *gin.Context) {
	db := database.GetDB()

	// 从审计日志中获取地区信息
	var auditLogs []models.AuditLog
	db.Select("DISTINCT user_id, location, ip_address").
		Where("user_id IS NOT NULL AND (location IS NOT NULL AND location != '' OR ip_address IS NOT NULL AND ip_address != '' AND ip_address != '127.0.0.1' AND ip_address != '::1')").
		Find(&auditLogs)

	// 从用户活动中获取地区信息
	var activities []models.UserActivity
	db.Select("DISTINCT user_id, location, ip_address").
		Where("location IS NOT NULL AND location != ''").
		Find(&activities)

	// 统计地区分布
	regionMap := make(map[string]map[string]interface{})
	userRegionMap := make(map[uint]string)
	regionDetailsMap := make(map[string]map[string]interface{})

	// 解析位置信息的辅助函数
	parseLocation := func(locationStr string) (country, city string) {
		if locationStr == "" {
			return "", ""
		}
		// 尝试解析JSON格式
		var locationData map[string]interface{}
		if err := json.Unmarshal([]byte(locationStr), &locationData); err == nil {
			if c, ok := locationData["country"].(string); ok {
				country = c
			}
			if c, ok := locationData["city"].(string); ok {
				city = c
			}
			return
		}
		// 尝试解析逗号分隔格式
		if strings.Contains(locationStr, ",") {
			parts := strings.Split(locationStr, ",")
			if len(parts) >= 1 {
				country = strings.TrimSpace(parts[0])
			}
			if len(parts) >= 2 {
				city = strings.TrimSpace(parts[1])
			}
			return
		}
		country = strings.TrimSpace(locationStr)
		return
	}

	// 处理审计日志
	for _, log := range auditLogs {
		if !log.UserID.Valid {
			continue
		}
		userID := uint(log.UserID.Int64)

		var country, city string
		if log.Location.Valid && log.Location.String != "" {
			country, city = parseLocation(log.Location.String)
		} else if log.IPAddress.Valid && log.IPAddress.String != "" {
			// 如果没有location，尝试使用GeoIP解析
			ip := log.IPAddress.String
			if geoip.IsEnabled() {
				location, err := geoip.GetLocation(ip)
				if err == nil && location != nil {
					country = location.Country
					city = location.City
				}
			}
		} else {
			continue
		}

		if country == "" {
			continue
		}

		regionKey := country
		if city != "" {
			regionKey = country + " - " + city
		}

		// 统计地区
		if _, exists := regionMap[regionKey]; !exists {
			regionMap[regionKey] = map[string]interface{}{
				"region":     regionKey,
				"country":    country,
				"city":       city,
				"userCount":  0,
				"loginCount": 0,
			}
		}
		regionMap[regionKey]["loginCount"] = regionMap[regionKey]["loginCount"].(int) + 1

		// 记录用户地区
		if _, exists := userRegionMap[userID]; !exists {
			userRegionMap[userID] = regionKey
			regionMap[regionKey]["userCount"] = regionMap[regionKey]["userCount"].(int) + 1
		}

		// 详细统计
		detailKey := country + "|" + city
		if _, exists := regionDetailsMap[detailKey]; !exists {
			regionDetailsMap[detailKey] = map[string]interface{}{
				"country":    country,
				"city":       city,
				"userCount":  0,
				"loginCount": 0,
				"lastLogin":  log.CreatedAt.Format("2006-01-02 15:04:05"),
			}
		}
		regionDetailsMap[detailKey]["loginCount"] = regionDetailsMap[detailKey]["loginCount"].(int) + 1
		lastLoginStr := regionDetailsMap[detailKey]["lastLogin"].(string)
		lastLoginTime, _ := time.Parse("2006-01-02 15:04:05", lastLoginStr)
		if log.CreatedAt.After(lastLoginTime) {
			regionDetailsMap[detailKey]["lastLogin"] = log.CreatedAt.Format("2006-01-02 15:04:05")
		}
	}

	// 处理用户活动
	for _, activity := range activities {
		if !activity.Location.Valid || activity.Location.String == "" {
			continue
		}

		country, city := parseLocation(activity.Location.String)
		if country == "" {
			continue
		}

		regionKey := country
		if city != "" {
			regionKey = country + " - " + city
		}

		if _, exists := regionMap[regionKey]; !exists {
			regionMap[regionKey] = map[string]interface{}{
				"region":     regionKey,
				"country":    country,
				"city":       city,
				"userCount":  0,
				"loginCount": 0,
			}
		}

		if _, exists := userRegionMap[activity.UserID]; !exists {
			userRegionMap[activity.UserID] = regionKey
			regionMap[regionKey]["userCount"] = regionMap[regionKey]["userCount"].(int) + 1
		}
	}

	// 转换为数组并计算百分比
	totalUsers := len(userRegionMap)
	regions := make([]map[string]interface{}, 0, len(regionMap))
	for _, region := range regionMap {
		userCount := region["userCount"].(int)
		percentage := 0.0
		if totalUsers > 0 {
			percentage = float64(userCount) / float64(totalUsers) * 100
		}
		region["percentage"] = fmt.Sprintf("%.1f", percentage)
		regions = append(regions, region)
	}

	// 转换为详细统计数组
	details := make([]map[string]interface{}, 0, len(regionDetailsMap))
	for _, detail := range regionDetailsMap {
		details = append(details, detail)
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"regions": regions,
		"details": details,
	})
}
