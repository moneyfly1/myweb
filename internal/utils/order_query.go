package utils

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

func CalculateTotalRevenue(db *gorm.DB, status string) float64 {
	var total float64
	query := db.Table("orders")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Select("COALESCE(SUM(CASE WHEN final_amount IS NOT NULL AND final_amount != 0 THEN final_amount ELSE amount END), 0)").
		Scan(&total)

	return total
}

func CalculateTodayRevenue(db *gorm.DB, status string) float64 {
	var total float64
	today := time.Now().Format("2006-01-02")
	query := db.Table("orders").Where("DATE(created_at) = ?", today)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Select("COALESCE(SUM(CASE WHEN final_amount IS NOT NULL AND final_amount != 0 THEN final_amount ELSE amount END), 0)").
		Scan(&total)

	return total
}

func CalculateUserOrderAmount(db *gorm.DB, userID uint, status string, useAbsolute bool) float64 {
	var total float64
	query := db.Table("orders").Where("user_id = ?", userID)

	if status != "" {
		query = query.Where("LOWER(status) = ?", strings.ToLower(status))
	}

	selectExpr := "COALESCE(SUM(CASE WHEN final_amount IS NOT NULL AND final_amount != 0 THEN final_amount ELSE amount END), 0)"
	if useAbsolute {
		selectExpr = "COALESCE(SUM(ABS(CASE WHEN final_amount IS NOT NULL AND final_amount != 0 THEN final_amount ELSE amount END)), 0)"
	}

	query.Select(selectExpr).Scan(&total)

	return total
}
