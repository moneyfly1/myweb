package utils

import (
	"database/sql"

	"gorm.io/gorm"
)

// CalculateTotalRevenue 计算总收入
// 使用final_amount，如果为NULL或0则使用amount
func CalculateTotalRevenue(db *gorm.DB, status string) float64 {
	var result struct {
		Total sql.NullFloat64
	}

	err := db.Raw(`
		SELECT COALESCE(SUM(
			CASE 
				WHEN final_amount IS NOT NULL AND final_amount != 0 THEN final_amount
				ELSE amount
			END
		), 0) as total
		FROM orders 
		WHERE status = ?
	`, status).Scan(&result).Error

	if err != nil || !result.Total.Valid {
		return 0
	}

	return result.Total.Float64
}

// CalculateTotalRevenueWithDateRange 计算指定日期范围内的总收入
func CalculateTotalRevenueWithDateRange(db *gorm.DB, status string, startDate, endDate string) float64 {
	var result struct {
		Total sql.NullFloat64
	}

	var sqlQuery string
	var args []interface{}

	// 构建SQL查询
	if startDate != "" && endDate != "" {
		sqlQuery = `
			SELECT COALESCE(SUM(
				CASE 
					WHEN final_amount IS NOT NULL AND final_amount != 0 THEN final_amount
					ELSE amount
				END
			), 0) as total
			FROM orders 
			WHERE status = ? AND DATE(created_at) >= ? AND DATE(created_at) <= ?
		`
		args = []interface{}{status, startDate, endDate}
	} else if startDate != "" {
		sqlQuery = `
			SELECT COALESCE(SUM(
				CASE 
					WHEN final_amount IS NOT NULL AND final_amount != 0 THEN final_amount
					ELSE amount
				END
			), 0) as total
			FROM orders 
			WHERE status = ? AND DATE(created_at) >= ?
		`
		args = []interface{}{status, startDate}
	} else if endDate != "" {
		sqlQuery = `
			SELECT COALESCE(SUM(
				CASE 
					WHEN final_amount IS NOT NULL AND final_amount != 0 THEN final_amount
					ELSE amount
				END
			), 0) as total
			FROM orders 
			WHERE status = ? AND DATE(created_at) <= ?
		`
		args = []interface{}{status, endDate}
	} else {
		// 没有日期范围，使用基础函数
		return CalculateTotalRevenue(db, status)
	}

	err := db.Raw(sqlQuery, args...).Scan(&result).Error
	if err != nil || !result.Total.Valid {
		return 0
	}

	return result.Total.Float64
}

// CalculateTodayRevenue 计算今日收入
func CalculateTodayRevenue(db *gorm.DB, status string) float64 {
	today := GetBeijingTime().Format("2006-01-02")
	var result struct {
		Total sql.NullFloat64
	}

	err := db.Raw(`
		SELECT COALESCE(SUM(
			CASE 
				WHEN final_amount IS NOT NULL AND final_amount != 0 THEN final_amount
				ELSE amount
			END
		), 0) as total
		FROM orders 
		WHERE status = ? AND DATE(created_at) = ?
	`, status, today).Scan(&result).Error

	if err != nil || !result.Total.Valid {
		return 0
	}

	return result.Total.Float64
}

// CalculateUserOrderAmount 计算用户订单金额（支持绝对值）
func CalculateUserOrderAmount(db *gorm.DB, userID uint, status string, useAbs bool) float64 {
	var result struct {
		Total sql.NullFloat64
	}

	var sqlQuery string
	if useAbs {
		sqlQuery = `
			SELECT COALESCE(SUM(
				CASE 
					WHEN final_amount IS NOT NULL AND final_amount != 0 THEN ABS(final_amount)
					ELSE ABS(amount)
				END
			), 0) as total
			FROM orders 
			WHERE user_id = ?
		`
	} else {
		sqlQuery = `
			SELECT COALESCE(SUM(
				CASE 
					WHEN final_amount IS NOT NULL AND final_amount != 0 THEN final_amount
					ELSE amount
				END
			), 0) as total
			FROM orders 
			WHERE user_id = ?
		`
	}

	args := []interface{}{userID}
	if status != "" {
		sqlQuery += " AND status = ?"
		args = append(args, status)
	}

	err := db.Raw(sqlQuery, args...).Scan(&result).Error
	if err != nil || !result.Total.Valid {
		return 0
	}

	return result.Total.Float64
}
