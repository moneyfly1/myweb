package utils

import (
	"fmt"
	"math/rand"
	"time"

	"gorm.io/gorm"
)

// GenerateCouponCode 生成优惠券码
func GenerateCouponCode() string {
	// 使用 rand.New 替代已弃用的 rand.Seed (Go 1.20+)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 8)
	for i := range code {
		code[i] = charset[r.Intn(len(charset))]
	}
	return string(code)
}

// GenerateOrderNo 生成订单号
// 格式：ORD + 时间（YYYYMMDDHHmmss）+ 当天订单序号（3位数字，如001）
// 例如：ORD20240101123456001
func GenerateOrderNo(db *gorm.DB) (string, error) {
	// 获取当前时间（北京时间）
	now := GetBeijingTime()

	// 格式化时间为：YYYYMMDDHHmmss
	timeStr := now.Format("20060102150405")

	// 计算当天订单序号
	// 查询当天已创建的订单数量（包括当前正在创建的）
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.AddDate(0, 0, 1)

	var count int64
	if err := db.Table("orders").
		Where("created_at >= ? AND created_at < ?", todayStart, todayEnd).
		Count(&count).Error; err != nil {
		return "", fmt.Errorf("查询当天订单数量失败: %v", err)
	}

	// 序号从1开始，所以当前订单序号 = count + 1
	sequence := count + 1

	// 格式化为3位数字（001, 002, ...）
	sequenceStr := fmt.Sprintf("%03d", sequence)

	// 组合订单号：ORD + 时间 + 序号
	orderNo := fmt.Sprintf("ORD%s%s", timeStr, sequenceStr)

	return orderNo, nil
}

// GenerateRechargeOrderNo 生成充值订单号
func GenerateRechargeOrderNo(userID uint) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("RCH%d%06d", timestamp, userID)
}

// GenerateTicketNo 生成工单号
func GenerateTicketNo(userID uint) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("TKT%d%06d", timestamp, userID)
}
