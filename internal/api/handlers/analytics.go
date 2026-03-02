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
	var dau, wau, mau int64

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

	// 优先使用 user_activities 表，如果为空则使用 users.last_login
	var activityCount int64
	db.Model(&models.UserActivity{}).Count(&activityCount)

	if activityCount > 0 {
		// 使用 user_activities 表统计
		// DAU - 当前期间活跃用户
		db.Model(&models.UserActivity{}).
			Where("created_at >= ? AND created_at < ?", currentStart, currentEnd).
			Distinct("user_id").Count(&dau)

		// WAU - 根据时间范围计算
		if timeRange == "day" {
			// 今日模式：最近7天活跃用户
			db.Model(&models.UserActivity{}).Where("created_at >= ?", weekStart).
				Distinct("user_id").Count(&wau)
		} else {
			// 本月/本年模式：当前期间活跃用户（与DAU相同）
			wau = dau
		}

		// MAU - 根据时间范围计算
		if timeRange == "day" {
			// 今日模式：最近30天活跃用户
			db.Model(&models.UserActivity{}).Where("created_at >= ?", monthStart).
				Distinct("user_id").Count(&mau)
		} else {
			// 本月/本年模式：当前期间活跃用户（与DAU相同）
			mau = dau
		}
	} else {
		// 使用 users.last_login 统计（备用方案）
		// DAU - 当前期间登录的用户
		db.Model(&models.User{}).
			Where("last_login >= ? AND last_login < ?", currentStart, currentEnd).
			Count(&dau)

		// WAU - 根据时间范围计算
		if timeRange == "day" {
			// 今日模式：最近7天登录的用户
			db.Model(&models.User{}).Where("last_login >= ?", weekStart).Count(&wau)
		} else {
			// 本月/本年模式：当前期间登录的用户（与DAU相同）
			wau = dau
		}

		// MAU - 根据时间范围计算
		if timeRange == "day" {
			// 今日模式：最近30天登录的用户
			db.Model(&models.User{}).Where("last_login >= ?", monthStart).Count(&mau)
		} else {
			// 本月/本年模式：当前期间登录的用户（与DAU相同）
			mau = dau
		}
	}

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

	var result []RetentionData

	// 1. 7日留存：7天前注册的用户中，之后仍登录过的比例
	d7Start := now.AddDate(0, 0, -8)
	d7End := now.AddDate(0, 0, -7)
	var total7 int64
	db.Model(&models.User{}).Where("created_at >= ? AND created_at < ?", d7Start, d7End).Count(&total7)
	var retained7 int64
	if total7 > 0 {
		db.Model(&models.User{}).Where("created_at >= ? AND created_at < ? AND last_login >= ?", d7Start, d7End, d7End).Count(&retained7)
	}
	rate7 := float64(0)
	if total7 > 0 {
		rate7 = float64(retained7) / float64(total7) * 100
	}
	result = append(result, RetentionData{Label: "7日留存", Retained: retained7, Total: total7, Rate: rate7})

	// 2. 30日留存：30天前注册的用户中，之后仍登录过的比例
	d30Start := now.AddDate(0, 0, -31)
	d30End := now.AddDate(0, 0, -30)
	var total30 int64
	db.Model(&models.User{}).Where("created_at >= ? AND created_at < ?", d30Start, d30End).Count(&total30)
	var retained30 int64
	if total30 > 0 {
		db.Model(&models.User{}).Where("created_at >= ? AND created_at < ? AND last_login >= ?", d30Start, d30End, d30End).Count(&retained30)
	}
	rate30 := float64(0)
	if total30 > 0 {
		rate30 = float64(retained30) / float64(total30) * 100
	}
	result = append(result, RetentionData{Label: "30日留存", Retained: retained30, Total: total30, Rate: rate30})

	// 3. 付费转化率：总用户中有过付费订单的比例
	var totalUsers int64
	db.Model(&models.User{}).Count(&totalUsers)
	var paidUsers int64
	db.Model(&models.Order{}).Where("status = ?", "paid").Distinct("user_id").Count(&paidUsers)
	paidRate := float64(0)
	if totalUsers > 0 {
		paidRate = float64(paidUsers) / float64(totalUsers) * 100
	}
	result = append(result, RetentionData{Label: "付费转化", Retained: paidUsers, Total: totalUsers, Rate: paidRate})

	// 4. 订阅活跃率：有活跃订阅的用户占总用户比例
	var activeSubs int64
	db.Model(&models.Subscription{}).Where("is_active = ? AND expire_time > ?", true, now).Distinct("user_id").Count(&activeSubs)
	activeRate := float64(0)
	if totalUsers > 0 {
		activeRate = float64(activeSubs) / float64(totalUsers) * 100
	}
	result = append(result, RetentionData{Label: "订阅活跃", Retained: activeSubs, Total: totalUsers, Rate: activeRate})

	// 5. 续费率：有2笔及以上付费订单的用户占付费用户比例
	var renewUsers int64
	db.Raw("SELECT COUNT(*) FROM (SELECT user_id FROM orders WHERE status = 'paid' GROUP BY user_id HAVING COUNT(*) >= 2)").Scan(&renewUsers)
	renewRate := float64(0)
	if paidUsers > 0 {
		renewRate = float64(renewUsers) / float64(paidUsers) * 100
	}
	result = append(result, RetentionData{Label: "续费率", Retained: renewUsers, Total: paidUsers, Rate: renewRate})

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

	// 当前期间收入 - 使用 final_amount（实际支付金额），如果为空则使用 amount
	var currentRevenue float64
	db.Model(&models.Order{}).
		Where("status = ? AND created_at >= ? AND created_at < ?", "paid", currentStart, currentEnd).
		Select("COALESCE(SUM(CASE WHEN final_amount IS NOT NULL THEN final_amount ELSE amount END), 0)").
		Scan(&currentRevenue)

	// 上期收入
	var previousRevenue float64
	db.Model(&models.Order{}).
		Where("status = ? AND created_at >= ? AND created_at < ?", "paid", previousStart, previousEnd).
		Select("COALESCE(SUM(CASE WHEN final_amount IS NOT NULL THEN final_amount ELSE amount END), 0)").
		Scan(&previousRevenue)

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
		"time_range":   timeRange,
		"current_start": currentStart.Format("2006-01-02 15:04:05"),
		"current_end":   currentEnd.Format("2006-01-02 15:04:05"),
	})
}

// formatMoney 格式化金额为两位小数
func formatMoney(amount float64) string {
	return fmt.Sprintf("%.2f", amount)
}

