package handlers

import (
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// 用户活跃度分析
func GetUserAnalytics(c *gin.Context) {
	db := database.GetDB()
	now := utils.GetBeijingTime()

	// 获取时间范围参数
	timeRange := c.DefaultQuery("range", "day")

	var currentStart, currentEnd time.Time
	var weekStart, monthStart time.Time
	var activityStats struct {
		ActivityCount int64
		DAU           int64
		WAU           int64
		MAU           int64
		TotalUsers    int64
	}

	switch timeRange {
	case "month":
		// 本月
		currentStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		currentEnd = currentStart.AddDate(0, 1, 0)
		// 本月的周活跃和月活跃都是本月范围
		weekStart = currentStart
		monthStart = currentStart
	case "year":
		// 本年
		currentStart = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		currentEnd = currentStart.AddDate(1, 0, 0)
		// 本年的周活跃和月活跃都是本年范围
		weekStart = currentStart
		monthStart = currentStart
	default: // day
		// 今日
		currentStart, currentEnd = utils.GetDayRange(now)
		// 今日的周活跃是最近7天，月活跃是最近30天
		weekStart = now.AddDate(0, 0, -7)
		monthStart = now.AddDate(0, -1, 0)
	}

	// 优先使用 user_activities 表，如果为空则使用 users.last_login。
	db.Raw(`
		SELECT
			COUNT(*) AS activity_count,
			COUNT(DISTINCT CASE WHEN created_at >= ? AND created_at < ? THEN user_id END) AS dau,
			COUNT(DISTINCT CASE WHEN created_at >= ? THEN user_id END) AS wau,
			COUNT(DISTINCT CASE WHEN created_at >= ? THEN user_id END) AS mau,
			(SELECT COUNT(*) FROM users) AS total_users
		FROM user_activities
	`, currentStart, currentEnd, weekStart, monthStart).Scan(&activityStats)

	if activityStats.ActivityCount > 0 {
		if timeRange != "day" {
			activityStats.WAU = activityStats.DAU
			activityStats.MAU = activityStats.DAU
		}
	} else {
		db.Raw(`
			SELECT
				0 AS activity_count,
				COALESCE(SUM(CASE WHEN last_login >= ? AND last_login < ? THEN 1 ELSE 0 END), 0) AS dau,
				COALESCE(SUM(CASE WHEN last_login >= ? THEN 1 ELSE 0 END), 0) AS wau,
				COALESCE(SUM(CASE WHEN last_login >= ? THEN 1 ELSE 0 END), 0) AS mau,
				COUNT(*) AS total_users
			FROM users
		`, currentStart, currentEnd, weekStart, monthStart).Scan(&activityStats)
		if timeRange != "day" {
			activityStats.WAU = activityStats.DAU
			activityStats.MAU = activityStats.DAU
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"dau":         activityStats.DAU,
		"wau":         activityStats.WAU,
		"mau":         activityStats.MAU,
		"total_users": activityStats.TotalUsers,
		"time_range":  timeRange,
	})
}

// 用户留存分析 - 基于订阅业务维度
func GetRetentionAnalytics(c *gin.Context) {
	db := database.GetDB()
	now := utils.GetBeijingTime()

	type RetentionData struct {
		Label    string  `json:"label"`
		Retained int64   `json:"retained"`
		Total    int64   `json:"total"`
		Rate     float64 `json:"rate"`
	}

	// 1. 7日留存：7天前注册的用户中，之后仍登录过的比例
	d7Start := now.AddDate(0, 0, -8)
	d7End := now.AddDate(0, 0, -7)
	// 2. 30日留存：30天前注册的用户中，之后仍登录过的比例
	d30Start := now.AddDate(0, 0, -31)
	d30End := now.AddDate(0, 0, -30)

	var stats struct {
		Total7     int64
		Retained7  int64
		Total30    int64
		Retained30 int64
		TotalUsers int64
		PaidUsers  int64
		ActiveSubs int64
		RenewUsers int64
	}

	db.Raw(`
		SELECT
			COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? THEN 1 ELSE 0 END), 0) AS total7,
			COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? AND last_login >= ? THEN 1 ELSE 0 END), 0) AS retained7,
			COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? THEN 1 ELSE 0 END), 0) AS total30,
			COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? AND last_login >= ? THEN 1 ELSE 0 END), 0) AS retained30,
			COUNT(*) AS total_users,
			(SELECT COUNT(DISTINCT user_id) FROM orders WHERE status = ?) AS paid_users,
			(SELECT COUNT(DISTINCT user_id) FROM subscriptions WHERE is_active = ? AND expire_time > ?) AS active_subs,
			(SELECT COUNT(*) FROM (SELECT user_id FROM orders WHERE status = ? GROUP BY user_id HAVING COUNT(*) >= 2) renew_user_set) AS renew_users
		FROM users
	`, d7Start, d7End, d7Start, d7End, d7End, d30Start, d30End, d30Start, d30End, d30End, "paid", true, now, "paid").Scan(&stats)

	rate := func(retained int64, total int64) float64 {
		if total <= 0 {
			return 0
		}
		return float64(retained) / float64(total) * 100
	}

	result := []RetentionData{
		{Label: "7日留存", Retained: stats.Retained7, Total: stats.Total7, Rate: rate(stats.Retained7, stats.Total7)},
		{Label: "30日留存", Retained: stats.Retained30, Total: stats.Total30, Rate: rate(stats.Retained30, stats.Total30)},
		{Label: "付费转化", Retained: stats.PaidUsers, Total: stats.TotalUsers, Rate: rate(stats.PaidUsers, stats.TotalUsers)},
		{Label: "订阅活跃", Retained: stats.ActiveSubs, Total: stats.TotalUsers, Rate: rate(stats.ActiveSubs, stats.TotalUsers)},
		{Label: "续费率", Retained: stats.RenewUsers, Total: stats.PaidUsers, Rate: rate(stats.RenewUsers, stats.PaidUsers)},
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// 用户流失预警
func GetChurnWarning(c *gin.Context) {
	db := database.GetDB()
	now := utils.GetBeijingTime()

	// 7天未活跃且订阅即将到期的用户
	sevenDaysAgo := now.AddDate(0, 0, -7)
	sevenDaysLater := now.AddDate(0, 0, 7)

	type ChurnUser struct {
		ID         uint      `json:"id"`
		Username   string    `json:"username"`
		Email      string    `json:"email"`
		LastLogin  time.Time `json:"last_login"`
		ExpireTime time.Time `json:"expire_time"`
	}

	var users []ChurnUser
	db.Raw(`SELECT u.id, u.username, u.email, u.last_login, s.expire_time
		FROM users u
		JOIN subscriptions s ON s.user_id = u.id AND s.is_active = 1
		WHERE (u.last_login < ? OR u.last_login IS NULL)
		AND s.expire_time BETWEEN ? AND ?
		ORDER BY s.expire_time ASC LIMIT 50`, sevenDaysAgo, now, sevenDaysLater).Scan(&users)

	utils.SuccessResponse(c, http.StatusOK, "", users)
}

// 设备分析
func GetDeviceAnalytics(c *gin.Context) {
	db := database.GetDB()

	type DeviceStat struct {
		DeviceType string `json:"device_type"`
		Count      int64  `json:"count"`
	}

	var stats []DeviceStat
	db.Model(&models.Device{}).Select("device_type, COUNT(*) as count").
		Group("device_type").Order("count DESC").Scan(&stats)

	var osStat []DeviceStat
	db.Model(&models.Device{}).Select("os as device_type, COUNT(*) as count").
		Where("os != ''").Group("os").Order("count DESC").Scan(&osStat)

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"device_types": stats,
		"os_stats":     osStat,
	})
}

// 收入统计分析
func GetRevenueAnalytics(c *gin.Context) {
	db := database.GetDB()
	now := utils.GetBeijingTime()

	// 获取时间范围参数
	timeRange := c.DefaultQuery("range", "day")

	var currentStart, currentEnd, previousStart, previousEnd time.Time

	switch timeRange {
	case "month":
		// 本月
		currentStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		currentEnd = currentStart.AddDate(0, 1, 0)
		// 上月
		previousStart = currentStart.AddDate(0, -1, 0)
		previousEnd = currentStart
	case "year":
		// 本年
		currentStart = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		currentEnd = currentStart.AddDate(1, 0, 0)
		// 去年
		previousStart = currentStart.AddDate(-1, 0, 0)
		previousEnd = currentStart
	default: // day
		// 今日
		currentStart, currentEnd = utils.GetDayRange(now)
		// 昨日
		yesterday := now.AddDate(0, 0, -1)
		previousStart, previousEnd = utils.GetDayRange(yesterday)
	}

	var revenueStats struct {
		CurrentRevenue  float64
		CurrentOrders   int64
		PreviousRevenue float64
	}
	db.Raw(`
		SELECT
			COALESCE(SUM(current_revenue), 0) AS current_revenue,
			COALESCE(SUM(current_orders), 0) AS current_orders,
			COALESCE(SUM(previous_revenue), 0) AS previous_revenue
		FROM (
			SELECT
				COALESCE(SUM(CASE WHEN status = ? AND created_at >= ? AND created_at < ? THEN
					CASE WHEN final_amount IS NOT NULL THEN final_amount ELSE amount END
				ELSE 0 END), 0) AS current_revenue,
				COALESCE(SUM(CASE WHEN status = ? AND created_at >= ? AND created_at < ? THEN 1 ELSE 0 END), 0) AS current_orders,
				COALESCE(SUM(CASE WHEN status = ? AND created_at >= ? AND created_at < ? THEN
					CASE WHEN final_amount IS NOT NULL THEN final_amount ELSE amount END
				ELSE 0 END), 0) AS previous_revenue
			FROM orders
			UNION ALL
			SELECT
				COALESCE(SUM(CASE WHEN status = ? AND created_at >= ? AND created_at < ? THEN amount ELSE 0 END), 0) AS current_revenue,
				COALESCE(SUM(CASE WHEN status = ? AND created_at >= ? AND created_at < ? THEN 1 ELSE 0 END), 0) AS current_orders,
				COALESCE(SUM(CASE WHEN status = ? AND created_at >= ? AND created_at < ? THEN amount ELSE 0 END), 0) AS previous_revenue
			FROM recharge_records
		) revenue_summary
	`, "paid", currentStart, currentEnd, "paid", currentStart, currentEnd, "paid", previousStart, previousEnd,
		"paid", currentStart, currentEnd, "paid", currentStart, currentEnd, "paid", previousStart, previousEnd).Scan(&revenueStats)

	currentRevenue := utils.RoundFloat(revenueStats.CurrentRevenue, 2)
	previousRevenue := utils.RoundFloat(revenueStats.PreviousRevenue, 2)
	orderCount := revenueStats.CurrentOrders

	// 平均订单金额
	avgOrder := float64(0)
	if orderCount > 0 {
		avgOrder = currentRevenue / float64(orderCount)
	}

	// 计算变化率
	changeRate := float64(0)
	if previousRevenue > 0 {
		changeRate = ((currentRevenue - previousRevenue) / previousRevenue) * 100
	} else if currentRevenue > 0 {
		changeRate = 100
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"current":       formatMoney(currentRevenue),
		"previous":      formatMoney(previousRevenue),
		"change_rate":   changeRate,
		"order_count":   orderCount,
		"avg_order":     formatMoney(avgOrder),
		"time_range":    timeRange,
		"current_start": currentStart.Format("2006-01-02 15:04:05"),
		"current_end":   currentEnd.Format("2006-01-02 15:04:05"),
	})
}

// formatMoney 格式化金额为两位小数
func formatMoney(amount float64) string {
	return fmt.Sprintf("%.2f", amount)
}
