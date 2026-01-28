package utils

import (
	crand "crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Base64Encode(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}

func GenerateUUID() string {
	return uuid.New().String()
}

func GenerateSubscriptionURL() string {
	b := make([]byte, 16)
	crand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

type orderNoRecord struct {
	OrderNo string `gorm:"column:order_no"`
}

func findMaxSequence(db *gorm.DB, prefix string) int {
	var records []orderNoRecord
	if err := db.Model(&orderNoRecord{}).Where("order_no LIKE ?", prefix+"%").Select("order_no").Find(&records).Error; err != nil {
		return 0
	}

	maxSeq := 0
	for _, record := range records {
		if len(record.OrderNo) >= len(prefix)+3 {
			var seq int
			if _, err := fmt.Sscanf(record.OrderNo[len(prefix):], "%d", &seq); err == nil && seq > maxSeq {
				maxSeq = seq
			}
		}
	}
	return maxSeq
}

func checkOrderNoExists(db *gorm.DB, orderNo string) bool {
	var count int64
	if err := db.Model(&orderNoRecord{}).Where("order_no = ?", orderNo).Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func incrementSequence(seq int) int {
	seq++
	if seq > 999 {
		return 1
	}
	return seq
}

func generateOrderNoWithPrefix(prefix string, db interface{}) (string, error) {
	now := GetBeijingTime()
	dateStr := now.Format("20060102")
	fullPrefix := fmt.Sprintf("%s%s", prefix, dateStr)

	maxSeq := 0
	if db != nil {
		if gormDB, ok := db.(*gorm.DB); ok {
			maxSeq = findMaxSequence(gormDB, fullPrefix)
		}
	}

	maxSeq = incrementSequence(maxSeq)
	orderNo := fmt.Sprintf("%s%03d", fullPrefix, maxSeq)

	if db != nil {
		if gormDB, ok := db.(*gorm.DB); ok {
			if checkOrderNoExists(gormDB, orderNo) {
				maxSeq = incrementSequence(maxSeq)
				orderNo = fmt.Sprintf("%s%03d", fullPrefix, maxSeq)
			}
		}
	}

	return orderNo, nil
}

func GenerateOrderNo(db interface{}) (string, error) {
	return generateOrderNoWithPrefix("ORD", db)
}

func GenerateRechargeOrderNo(userID uint, db interface{}) (string, error) {
	return generateOrderNoWithPrefix("RCH", db)
}

func GenerateDeviceUpgradeOrderNo(db interface{}) (string, error) {
	return generateOrderNoWithPrefix("UPG", db)
}

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

func GenerateTicketNo(userID uint) string {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 2)
	crand.Read(randomBytes)
	randomStr := base64.URLEncoding.EncodeToString(randomBytes)[:3]
	return fmt.Sprintf("TKT%d%d%s", timestamp, userID, randomStr)
}
