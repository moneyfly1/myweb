package utils

import (
	crand "crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base64Encode 对字符串进行Base64编码
func Base64Encode(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}

// GenerateUUID 生成UUID
func GenerateUUID() string {
	return uuid.New().String()
}

// GenerateSubscriptionURL 生成订阅URL（Base64编码的随机字符串）
// 使用密码学安全的随机数生成器，生成16字节（128位）随机数
// 返回约22字符的Base64 URL编码字符串
func GenerateSubscriptionURL() string {
	b := make([]byte, 16) // 16字节 = 128位
	crand.Read(b)         // 使用密码学安全的随机数生成器
	return base64.URLEncoding.EncodeToString(b)
}

// GenerateOrderNo 生成订单号
// 格式：ORD + timestamp + 随机数
// 如果提供了数据库，会检查唯一性
func GenerateOrderNo(db interface{}) (string, error) {
	// 生成订单号：ORD + 时间戳 + 随机数
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 4)
	crand.Read(randomBytes)
	randomStr := base64.URLEncoding.EncodeToString(randomBytes)[:6] // 取前6个字符

	orderNo := fmt.Sprintf("ORD%d%s", timestamp, randomStr)

	// 如果提供了数据库，检查唯一性
	if db != nil {
		if gormDB, ok := db.(*gorm.DB); ok {
			var count int64
			if err := gormDB.Model(&struct {
				OrderNo string `gorm:"column:order_no"`
			}{}).Where("order_no = ?", orderNo).Count(&count).Error; err == nil && count > 0 {
				// 如果重复，重新生成
				return GenerateOrderNo(db)
			}
		}
	}

	return orderNo, nil
}

// GenerateCouponCode 生成优惠券码（8位大写字母和数字）
func GenerateCouponCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	for i := range b {
		randBytes := make([]byte, 1)
		crand.Read(randBytes)
		b[i] = charset[int(randBytes[0])%len(charset)]
	}
	return string(b)
}

// GenerateRechargeOrderNo 生成充值订单号
// 格式：RCH + timestamp + userID + 随机数
func GenerateRechargeOrderNo(userID uint) string {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 2)
	crand.Read(randomBytes)
	randomStr := base64.URLEncoding.EncodeToString(randomBytes)[:3] // 取前3个字符
	return fmt.Sprintf("RCH%d%d%s", timestamp, userID, randomStr)
}

// GenerateTicketNo 生成工单号
// 格式：TKT + timestamp + userID + 随机数
func GenerateTicketNo(userID uint) string {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 2)
	crand.Read(randomBytes)
	randomStr := base64.URLEncoding.EncodeToString(randomBytes)[:3] // 取前3个字符
	return fmt.Sprintf("TKT%d%d%s", timestamp, userID, randomStr)
}
