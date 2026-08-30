package handlers

import (
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/cache_service"
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
	cacheService := cache_service.NewCacheService()

	// 尝试从 Redis 缓存获取
	if cached, ok := cacheService.GetStatisticsCache("overview"); ok {
		utils.SuccessResponse(c, http.StatusOK, "", cached)
		return
	}

	now := utils.GetBeijingTime()

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

	var userAggregate struct {
		Total      int64
		Active     int64
		Inactive   int64
		Verified   int64
		Unverified int64
	}
	db.Raw(`
		SELECT
			COUNT(*) AS total,
			COALESCE(SUM(CASE WHEN is_active = ? THEN 1 ELSE 0 END), 0) AS active,
			COALESCE(SUM(CASE WHEN is_active = ? THEN 1 ELSE 0 END), 0) AS inactive,
			COALESCE(SUM(CASE WHEN is_verified = ? THEN 1 ELSE 0 END), 0) AS verified,
			COALESCE(SUM(CASE WHEN is_verified = ? THEN 1 ELSE 0 END), 0) AS unverified
		FROM users
	`, true, false, true, false).Scan(&userAggregate)
	stats.TotalUsers = userAggregate.Total
	stats.ActiveUsers = userAggregate.Active

	dayStart, dayEnd := utils.GetDayRange(now)
	paymentSummary := utils.CalculatePaymentSummary(db, dayStart, dayEnd)
	stats.TotalOrders = paymentSummary.Total
	stats.PaidOrders = paymentSummary.Paid
	stats.TotalRevenue = paymentSummary.PaidRevenue
	stats.TodayOrders = paymentSummary.RangePaid
	stats.TodayRevenue = paymentSummary.RangeRevenue

	var subscriptionAggregate struct {
		Total    int64
		Active   int64
		Expired  int64
		Inactive int64
	}
	db.Raw(`
		SELECT
			COUNT(*) AS total,
			COALESCE(SUM(CASE WHEN is_active = ? AND (status = ? OR status = '' OR status IS NULL) AND expire_time > ? THEN 1 ELSE 0 END), 0) AS active,
			COALESCE(SUM(CASE WHEN expire_time <= ? THEN 1 ELSE 0 END), 0) AS expired,
			COALESCE(SUM(CASE WHEN is_active = ? THEN 1 ELSE 0 END), 0) AS inactive
		FROM subscriptions
	`, true, "active", now, now, false).Scan(&subscriptionAggregate)
	stats.TotalSubscriptions = subscriptionAggregate.Total
	stats.ActiveSubscriptions = subscriptionAggregate.Active

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
			"value": userAggregate.Inactive,
			"percentage": func() float64 {
				if stats.TotalUsers > 0 {
					return float64(userAggregate.Inactive) / float64(stats.TotalUsers) * 100
				}
				return 0
			}(),
		},
		{
			"name":  "已验证用户",
			"value": userAggregate.Verified,
			"percentage": func() float64 {
				if stats.TotalUsers > 0 {
					return float64(userAggregate.Verified) / float64(stats.TotalUsers) * 100
				}
				return 0
			}(),
		},
		{
			"name":  "未验证用户",
			"value": userAggregate.Unverified,
			"percentage": func() float64 {
				if stats.TotalUsers > 0 {
					return float64(userAggregate.Unverified) / float64(stats.TotalUsers) * 100
				}
				return 0
			}(),
		},
	}

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
			"value": subscriptionAggregate.Expired,
			"percentage": func() float64 {
				if stats.TotalSubscriptions > 0 {
					return float64(subscriptionAggregate.Expired) / float64(stats.TotalSubscriptions) * 100
				}
				return 0
			}(),
		},
		{
			"name":  "未激活订阅",
			"value": subscriptionAggregate.Inactive,
			"percentage": func() float64 {
				if stats.TotalSubscriptions > 0 {
					return float64(subscriptionAggregate.Inactive) / float64(stats.TotalSubscriptions) * 100
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
			"time":        utils.FormatBeijingTime(order.CreatedAt),
		})
	}

	payload := gin.H{
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
	}

	// 异步写入 Redis 缓存
	go cacheService.SetStatisticsCache("overview", payload, 30*time.Second)

	utils.SuccessResponse(c, http.StatusOK, "", payload)
}

func GetRevenueChart(c *gin.Context) {
	days := 30
	if daysParam := c.Query("days"); daysParam != "" {
		_, _ = fmt.Sscanf(daysParam, "%d", &days) // Ignore error, use default value
	}

	cacheService := cache_service.NewCacheService()
	cacheKey := fmt.Sprintf("revenue_chart:%d", days)

	// 尝试从缓存获取
	if cached, ok := cacheService.GetStatisticsCache(cacheKey); ok {
		utils.SuccessResponse(c, http.StatusOK, "", cached)
		return
	}

	type RevenueStat struct {
		Date    string  `json:"date"`
		Revenue float64 `json:"revenue"`
	}

	var stats []RevenueStat
	db := database.GetDB()
	startTime := utils.GetBeijingTime().AddDate(0, 0, -days)
	rows, err := db.Raw(`
		SELECT date, COALESCE(SUM(revenue), 0) AS revenue
		FROM (
			SELECT DATE(created_at) AS date, COALESCE(SUM(
				CASE
					WHEN final_amount IS NOT NULL THEN final_amount
					ELSE amount
				END
			), 0) AS revenue
			FROM orders
			WHERE status = ? AND created_at >= ?
			GROUP BY DATE(created_at)
			UNION ALL
			SELECT DATE(created_at) AS date, COALESCE(SUM(amount), 0) AS revenue
			FROM recharge_records
			WHERE status = ? AND created_at >= ?
			GROUP BY DATE(created_at)
		) revenue_records
		GROUP BY date
		ORDER BY date ASC
	`, "paid", startTime, "paid", startTime).Rows()

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var stat RevenueStat
			if scanErr := rows.Scan(&stat.Date, &stat.Revenue); scanErr == nil {
				stats = append(stats, stat)
			}
		}
	}

	labels := make([]string, 0)
	data := make([]float64, 0)
	for _, stat := range stats {
		labels = append(labels, stat.Date)
		data = append(data, stat.Revenue)
	}

	result := gin.H{
		"labels": labels,
		"data":   data,
	}

	// 异步写入缓存
	go cacheService.SetStatisticsCache(cacheKey, result, 5*time.Minute)

	utils.SuccessResponse(c, http.StatusOK, "", result)
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

	now := utils.GetBeijingTime()
	dayStart, dayEnd := utils.GetDayRange(now)

	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekStart := now.AddDate(0, 0, -weekday+1)
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	if err := db.Raw(`
		SELECT
			COUNT(*) AS total_users,
			COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? THEN 1 ELSE 0 END), 0) AS new_users_today,
			COALESCE(SUM(CASE WHEN created_at >= ? THEN 1 ELSE 0 END), 0) AS new_users_this_week,
			COALESCE(SUM(CASE WHEN created_at >= ? THEN 1 ELSE 0 END), 0) AS new_users_this_month,
			COALESCE(SUM(CASE WHEN is_verified = ? THEN 1 ELSE 0 END), 0) AS verified_users,
			COALESCE(SUM(CASE WHEN is_verified = ? THEN 1 ELSE 0 END), 0) AS unverified_users
		FROM users
	`, dayStart, dayEnd, weekStart, monthStart, true, false).Scan(&stats).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取用户统计失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", stats)
}

func GetRegionStats(c *gin.Context) {
	db := database.GetDB()
	// 只统计最近 90 天的活动：地区分布卡不需要无限期历史，
	// 避免每次打开统计页都把整张 audit_logs/user_activities 表装载进内存。
	windowStart := utils.GetBeijingTime().AddDate(0, -3, 0)

	type auditCombo struct {
		UserID   sql.NullInt64  `gorm:"column:user_id"`
		Location sql.NullString `gorm:"column:location"`
		IP       sql.NullString `gorm:"column:ip_address"`
		LastAt   sql.NullString `gorm:"column:last_at"` // SQLite created_at 为 TEXT，不能直接扫入 time.Time
	}
	var auditCombos []auditCombo
	if err := db.Table("audit_logs").
		Select("user_id, location, ip_address, MAX(created_at) AS last_at").
		Where("user_id IS NOT NULL AND created_at >= ? AND (location IS NOT NULL AND location != '' OR ip_address IS NOT NULL AND ip_address != '')", windowStart).
		Group("user_id, location, ip_address").
		Scan(&auditCombos).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取地区统计失败", err)
		return
	}

	// 登录次数在 SQL 侧按 location 精确计数（保持与原语义一致）
	type locationCount struct {
		Location sql.NullString `gorm:"column:location"`
		Cnt      int64          `gorm:"column:cnt"`
	}
	var auditCounts []locationCount
	if err := db.Table("audit_logs").
		Select("location, COUNT(*) AS cnt").
		Where("location IS NOT NULL AND location != '' AND created_at >= ?", windowStart).
		Group("location").
		Scan(&auditCounts).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取地区统计失败", err)
		return
	}

	type activityCombo struct {
		UserID   uint           `gorm:"column:user_id"`
		Location sql.NullString `gorm:"column:location"`
		IP       sql.NullString `gorm:"column:ip_address"`
	}
	var activityCombos []activityCombo
	if err := db.Table("user_activities").
		Select("user_id, location, ip_address").
		Where("location IS NOT NULL AND location != ''").
		Group("user_id, location, ip_address").
		Scan(&activityCombos).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取地区统计失败", err)
		return
	}

	var activityCounts []locationCount
	if err := db.Table("user_activities").
		Select("location, COUNT(*) AS cnt").
		Where("location IS NOT NULL AND location != ''").
		Group("location").
		Scan(&activityCounts).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取地区统计失败", err)
		return
	}

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

	regionKeyOf := func(country, city string) string {
		if city != "" {
			return country + " - " + city
		}
		return country
	}

	ensureRegion := func(country, city string) *RegionStat {
		key := regionKeyOf(country, city)
		stat, ok := statsMap[key]
		if !ok {
			stat = &RegionStat{Region: key, Country: country, City: city, LastLogin: "-"}
			statsMap[key] = stat
		}
		return stat
	}

	// 合并登录/活动计数（按 location 原始字符串 → 解析出地区）
	mergeCounts := func(counts []locationCount) {
		for _, row := range counts {
			country, city := parseLocation(row.Location.String)
			if country == "" {
				continue
			}
			ensureRegion(country, city).LoginCount += int(row.Cnt)
		}
	}
	mergeCounts(auditCounts)
	mergeCounts(activityCounts)

	// 组合数据：用户归属地区 + 最近登录时间
	registerCombo := func(userID uint, locationStr, ipStr string, createdAt time.Time) {
		var country, city string
		if locationStr != "" {
			country, city = parseLocation(locationStr)
		} else if ipStr != "" && ipStr != "127.0.0.1" && ipStr != "::1" && geoip.IsEnabled() {
			locationResult := geoip.GetLocationWithCache(ipStr)
			if locationResult.Valid && locationResult.String != "" {
				country, city = parseLocation(locationResult.String)
			}
		}
		if country == "" {
			return
		}

		stat := ensureRegion(country, city)
		if !createdAt.IsZero() {
			currentLastLogin := time.Time{}
			if stat.LastLogin != "-" {
				currentLastLogin, _ = time.Parse("2006-01-02 15:04:05", stat.LastLogin)
			}
			if createdAt.After(currentLastLogin) {
				stat.LastLogin = utils.FormatBeijingTime(createdAt)
			}
		}
		if _, exists := userRegionMap[userID]; !exists {
			userRegionMap[userID] = stat.Region
			stat.UserCount++
		}
	}

	for _, combo := range auditCombos {
		if combo.UserID.Valid {
			registerCombo(utils.MustSafeInt64ToUint(combo.UserID.Int64), combo.Location.String, combo.IP.String, parseLastAt(combo.LastAt))
		}
	}
	for _, combo := range activityCombos {
		registerCombo(combo.UserID, combo.Location.String, combo.IP.String, time.Time{})
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

// parseLastAt 将 MAX(created_at) 的字符串结果解析为 time.Time。
// SQLite 存的是 "2006-01-02 15:04:05"（或带时区/毫秒），MySQL 可能返回 time.Time 序列化格式，
// 统一按常见格式解析；解析失败返回零值（不影响地区分布统计本身）。
func parseLastAt(s sql.NullString) time.Time {
	if !s.Valid || strings.TrimSpace(s.String) == "" {
		return time.Time{}
	}
	val := strings.TrimSpace(s.String)
	formats := []string{
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.ParseInLocation(f, val, utils.BeijingTZ); err == nil {
			return t
		}
	}
	return time.Time{}
}
