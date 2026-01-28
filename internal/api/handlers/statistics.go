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

	db.Model(&models.User{}).Count(&stats.TotalUsers)
	db.Model(&models.User{}).Where("is_active = ?", true).Count(&stats.ActiveUsers)

	db.Model(&models.Order{}).Count(&stats.TotalOrders)
	db.Model(&models.Order{}).Where("status = ?", "paid").Count(&stats.PaidOrders)

	stats.TotalRevenue = utils.CalculateTotalRevenue(db, "paid")

	today := time.Now().Format("2006-01-02")
	db.Model(&models.Order{}).Where("status = ? AND DATE(created_at) = ?", "paid", today).Count(&stats.TodayOrders)
	stats.TodayRevenue = utils.CalculateTodayRevenue(db, "paid")

	db.Model(&models.Subscription{}).Count(&stats.TotalSubscriptions)
	now := time.Now()
	db.Model(&models.Subscription{}).
		Where("is_active = ?", true).
		Where("(status = ? OR status = '' OR status IS NULL)", "active").
		Where("expire_time > ?", now).
		Count(&stats.ActiveSubscriptions)

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
		"userStats":         userStatsList,
		"subscriptionStats": subscriptionStatsList,
		"recentActivities":  recentActivitiesList,
	})
}

func GetRevenueChart(c *gin.Context) {
	_ = c.DefaultQuery("days", "30")

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

	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekStart := now.AddDate(0, 0, -weekday+1)
	weekStartStr := weekStart.Format("2006-01-02")
	db.Model(&models.User{}).Where("DATE(created_at) >= ?", weekStartStr).Count(&stats.NewUsersThisWeek)

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthStartStr := monthStart.Format("2006-01-02")
	db.Model(&models.User{}).Where("DATE(created_at) >= ?", monthStartStr).Count(&stats.NewUsersThisMonth)

	utils.SuccessResponse(c, http.StatusOK, "", stats)
}

func GetRegionStats(c *gin.Context) {
	db := database.GetDB()

	var auditLogs []models.AuditLog
	db.Select("DISTINCT user_id, location, ip_address, created_at").
		Where("user_id IS NOT NULL AND (location IS NOT NULL AND location != '' OR ip_address IS NOT NULL AND ip_address != '')").
		Order("created_at DESC").
		Find(&auditLogs)

	var activities []models.UserActivity
	db.Select("DISTINCT user_id, location, ip_address").
		Where("location IS NOT NULL AND location != ''").
		Find(&activities)

	type RegionStat struct {
		Region     string `json:"region"`
		Country    string `json:"country"`
		City       string `json:"city"`
		UserCount  int    `json:"userCount"`
		LoginCount int    `json:"loginCount"`
		Percentage string `json:"percentage"`
		LastLogin  string `json:"lastLogin"`
	}

	statsMap := make(map[string]*RegionStat)
	userRegionMap := make(map[uint]string)

	parseLocation := func(locationStr string) (country, city string) {
		if locationStr == "" {
			return "", ""
		}
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

	processEntry := func(userID uint, locationStr, ipStr string, createdAt time.Time) {
		var country, city string
		if locationStr != "" {
			country, city = parseLocation(locationStr)
		} else if ipStr != "" && ipStr != "127.0.0.1" && ipStr != "::1" && geoip.IsEnabled() {
			if location, err := geoip.GetLocation(ipStr); err == nil && location != nil {
				country = location.Country
				city = location.City
			}
		}

		if country == "" {
			return
		}

		regionKey := country
		if city != "" {
			regionKey = country + " - " + city
		}

		if _, exists := statsMap[regionKey]; !exists {
			statsMap[regionKey] = &RegionStat{
				Region:    regionKey,
				Country:   country,
				City:      city,
				LastLogin: "-",
			}
		}

		stat := statsMap[regionKey]
		stat.LoginCount++

		if !createdAt.IsZero() {
			currentLastLogin := time.Time{}
			if stat.LastLogin != "-" {
				currentLastLogin, _ = time.Parse("2006-01-02 15:04:05", stat.LastLogin)
			}
			if createdAt.After(currentLastLogin) {
				stat.LastLogin = createdAt.Format("2006-01-02 15:04:05")
			}
		}

		if _, exists := userRegionMap[userID]; !exists {
			userRegionMap[userID] = regionKey
			stat.UserCount++
		}
	}

	for _, log := range auditLogs {
		if log.UserID.Valid {
			processEntry(uint(log.UserID.Int64), log.Location.String, log.IPAddress.String, log.CreatedAt)
		}
	}

	for _, activity := range activities {
		processEntry(activity.UserID, activity.Location.String, "", time.Time{})
	}

	totalUsers := len(userRegionMap)
	regions := make([]*RegionStat, 0, len(statsMap))
	for _, stat := range statsMap {
		percentage := 0.0
		if totalUsers > 0 {
			percentage = float64(stat.UserCount) / float64(totalUsers) * 100
		}
		stat.Percentage = fmt.Sprintf("%.1f", percentage)
		regions = append(regions, stat)
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"regions": regions,
	})
}
