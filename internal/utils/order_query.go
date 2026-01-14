package utils

import (
	"time"

	"gorm.io/gorm"
)

// CalculateTotalRevenue 计算总收入
// status: 订单状态筛选（如 "paid"），空字符串表示所有订单
func CalculateTotalRevenue(db *gorm.DB, status string) float64 {
	var total float64
	query := db.Table("orders")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 使用 COALESCE 优先使用 final_amount，如果为 NULL 或 0 则使用 amount
	query.Select("COALESCE(SUM(CASE WHEN final_amount IS NOT NULL AND final_amount != 0 THEN final_amount ELSE amount END), 0)").
		Scan(&total)

	return total
}

// CalculateTodayRevenue 计算今日收入
// status: 订单状态筛选（如 "paid"），空字符串表示所有订单
func CalculateTodayRevenue(db *gorm.DB, status string) float64 {
	var total float64
	today := time.Now().Format("2006-01-02")
	query := db.Table("orders").Where("DATE(created_at) = ?", today)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 使用 COALESCE 优先使用 final_amount，如果为 NULL 或 0 则使用 amount
	query.Select("COALESCE(SUM(CASE WHEN final_amount IS NOT NULL AND final_amount != 0 THEN final_amount ELSE amount END), 0)").
		Scan(&total)

	return total
}

// CalculateUserOrderAmount 计算用户订单金额
// userID: 用户ID
// status: 订单状态筛选（如 "paid"），空字符串表示所有订单
// useAbsolute: 是否使用绝对值
func CalculateUserOrderAmount(db *gorm.DB, userID uint, status string, useAbsolute bool) float64 {
	var total float64
	query := db.Table("orders").Where("user_id = ?", userID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 使用 COALESCE 优先使用 final_amount，如果为 NULL 或 0 则使用 amount
	selectExpr := "COALESCE(SUM(CASE WHEN final_amount IS NOT NULL AND final_amount != 0 THEN final_amount ELSE amount END), 0)"
	if useAbsolute {
		selectExpr = "COALESCE(SUM(ABS(CASE WHEN final_amount IS NOT NULL AND final_amount != 0 THEN final_amount ELSE amount END)), 0)"
	}

	query.Select(selectExpr).Scan(&total)

	return total
}
