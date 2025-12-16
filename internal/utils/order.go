package utils

import (
	"fmt"
	"time"
)

// GenerateOrderNo 生成订单号
func GenerateOrderNo(userID uint) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("ORD%d%06d", timestamp, userID)
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

