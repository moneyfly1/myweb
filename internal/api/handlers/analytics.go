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
	var dau, wau, mau int64

	switch timeRange {
	case "month":
		// 本月
		currentStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		currentEnd = currentStart.AddDate(0, 1, 0)
	case "year":
		// 本年
		currentStart = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		currentEnd = currentStart.AddDate(1, 0, 0)
	default: // day
		// 今日
		currentStart, currentEnd = utils.GetDayRange(now)
	}

	// 当前期间活跃用户
	db.Model(&models.UserActivity{}).
		Where("created_at >= ? AND created_at < ?", currentStart, currentEnd).
		Distinct("user_id").Count(&dau)

	// 周活跃（最近7天）
	weekStart := now.AddDate(0, 0, -7)
	db.Model(&models.UserActivity{}).Where("created_at >= ?", weekStart).
		Distinct("user_id").Count(&wau)

	// 月活跃（最近30天）
	monthStart := now.AddDate(0, -1, 0)
	db.Model(&models.UserActivity{}).Where("created_at >= ?", monthStart).
		Distinct("user_id").Count(&mau)

	var totalUsers int64
	db.Model(&models.User{}).Count(&totalUsers)

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"dau":         dau,
		"wau":         wau,
		"mau":         mau,
		"total_users": totalUsers,
		"time_range":  timeRange,
	})
}

// 用户留存分析
func GetRetentionAnalytics(c *gin.Context) {
	db := database.GetDB()
	now := utils.GetBeijingTime()

	type RetentionData struct {
		Day       int     `json:"day"`
		Retained  int64   `json:"retained"`
		Total     int64   `json:"total"`
		Rate      float64 `json:"rate"`
	}

	var result []RetentionData
	for _, days := range []int{1, 3, 7, 14, 30} {
		registerStart := now.AddDate(0, 0, -days-1)
		registerEnd := now.AddDate(0, 0, -days)

		var totalNew int64
		db.Model(&models.User{}).Where("created_at >= ? AND created_at < ?", registerStart, registerEnd).Count(&totalNew)

		var retained int64
		if totalNew > 0 {
			db.Model(&models.UserActivity{}).
				Where("created_at >= ? AND user_id IN (?)",
					registerEnd,
					db.Model(&models.User{}).Select("id").Where("created_at >= ? AND created_at < ?", registerStart, registerEnd),
				).Distinct("user_id").Count(&retained)
		}

		rate := float64(0)
		if totalNew > 0 {
			rate = float64(retained) / float64(totalNew) * 100
		}
		result = append(result, RetentionData{Day: days, Retained: retained, Total: totalNew, Rate: rate})
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
		ID       uint      `json:"id"`
		Username string    `json:"username"`
		Email    string    `json:"email"`
		LastLogin time.Time `json:"last_login"`
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

	// 当前期间收入
	var currentRevenue float64
	db.Model(&models.Order{}).
		Where("status = ? AND created_at >= ? AND created_at < ?", "paid", currentStart, currentEnd).
		Select("COALESCE(SUM(amount), 0)").Scan(&currentRevenue)

	// 上期收入
	var previousRevenue float64
	db.Model(&models.Order{}).
		Where("status = ? AND created_at >= ? AND created_at < ?", "paid", previousStart, previousEnd).
		Select("COALESCE(SUM(amount), 0)").Scan(&previousRevenue)

	// 订单数量
	var orderCount int64
	db.Model(&models.Order{}).
		Where("status = ? AND created_at >= ? AND created_at < ?", "paid", currentStart, currentEnd).
		Count(&orderCount)

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
		"current":      formatMoney(currentRevenue),
		"previous":     formatMoney(previousRevenue),
		"change_rate":  changeRate,
		"order_count":  orderCount,
		"avg_order":    formatMoney(avgOrder),
	})
}

// formatMoney 格式化金额为两位小数
func formatMoney(amount float64) string {
	return fmt.Sprintf("%.2f", amount)
}

