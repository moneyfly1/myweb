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

func findMaxSequenceFromTable(db *gorm.DB, tableName string, prefix string) int {
	var maxSeq int
	dateStr := GetBeijingTime().Format("20060102")
	fullPrefix := fmt.Sprintf("%s%s", prefix, dateStr)

	validTableNames := map[string]bool{
		"orders":           true,
		"recharge_records": true,
	}
	if !validTableNames[tableName] {
		return 0
	}

	var orderNos []string
	query := fmt.Sprintf("SELECT order_no FROM %s WHERE order_no LIKE ? ORDER BY order_no DESC LIMIT 100", tableName)
	if err := db.Raw(query, fullPrefix+"%").Scan(&orderNos).Error; err != nil {
		return 0
	}

	for _, orderNo := range orderNos {
		if len(orderNo) >= len(fullPrefix)+3 {
			var seq int
			if _, err := fmt.Sscanf(orderNo[len(fullPrefix):], "%d", &seq); err == nil && seq > maxSeq {
				maxSeq = seq
			}
		}
	}
	return maxSeq
}

func checkOrderNoExistsInTable(db *gorm.DB, tableName string, orderNo string) bool {
	validTableNames := map[string]bool{
		"orders":           true,
		"recharge_records": true,
	}
	if !validTableNames[tableName] {
		return false
	}

	var count int64
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE order_no = ?", tableName)
	if err := db.Raw(query, orderNo).Scan(&count).Error; err != nil {
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

func generateOrderNoWithPrefix(prefix string, tableName string, db interface{}) (string, error) {
	now := GetBeijingTime()
	dateStr := now.Format("20060102")
	fullPrefix := fmt.Sprintf("%s%s", prefix, dateStr)

	maxSeq := 0
	if db != nil {
		if gormDB, ok := db.(*gorm.DB); ok {
			maxSeq = findMaxSequenceFromTable(gormDB, tableName, prefix)
		}
	}

	maxSeq = incrementSequence(maxSeq)
	orderNo := fmt.Sprintf("%s%03d", fullPrefix, maxSeq)

	if db != nil {
		if gormDB, ok := db.(*gorm.DB); ok {
			for i := 0; i < 10; i++ {
				if !checkOrderNoExistsInTable(gormDB, tableName, orderNo) {
					break
				}
				maxSeq = incrementSequence(maxSeq)
				orderNo = fmt.Sprintf("%s%03d", fullPrefix, maxSeq)
			}
		}
	}

	return orderNo, nil
}

func GenerateOrderNo(db interface{}) (string, error) {
	return generateOrderNoWithPrefix("ORD", "orders", db)
}

func GenerateRechargeOrderNo(userID uint, db interface{}) (string, error) {
	return generateOrderNoWithPrefix("RCH", "recharge_records", db)
}

func GenerateDeviceUpgradeOrderNo(db interface{}) (string, error) {
	return generateOrderNoWithPrefix("UPG", "orders", db)
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
