package utils

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/geoip"

	"gorm.io/gorm"
)

// ==========================================
// 注册日志记录
// ==========================================

// CreateRegistrationLog 创建注册日志
func CreateRegistrationLog(userID uint, username, email, ipAddress, userAgent string, inviteCode string, inviterID *uint) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	var location sql.NullString
	if ipAddress != "" && geoip.IsEnabled() {
		location = geoip.GetLocationWithCache(ipAddress)
	}

	log := models.RegistrationLog{
		UserID:         userID,
		Username:       username,
		Email:          email,
		IPAddress:      database.NullString(ipAddress),
		UserAgent:      database.NullString(userAgent),
		Location:       location,
		Status:         "success",
		RegisterSource: database.NullString("direct"),
	}

	if inviteCode != "" {
		log.InviteCode = database.NullString(inviteCode)
		log.RegisterSource = database.NullString("invite_code")
	}

	if inviterID != nil {
		log.InviterID = database.NullInt64(MustSafeUintToInt64(*inviterID))
	}

	return db.Create(&log).Error
}

// CreateRegistrationLogFailed 创建注册失败日志
func CreateRegistrationLogFailed(email, ipAddress, userAgent, reason string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	var location sql.NullString
	if ipAddress != "" && geoip.IsEnabled() {
		location = geoip.GetLocationWithCache(ipAddress)
	}

	log := models.RegistrationLog{
		Email:         email,
		IPAddress:     database.NullString(ipAddress),
		UserAgent:     database.NullString(userAgent),
		Location:      location,
		Status:        "failed",
		FailureReason: database.NullString(reason),
	}

	return db.Create(&log).Error
}

// ==========================================
// 订阅日志记录
// ==========================================

// CreateSubscriptionLog 创建订阅日志
func CreateSubscriptionLog(subscriptionID, userID uint, actionType, actionBy string, actionByUserID *uint, ipAddress string, beforeData, afterData map[string]interface{}, description string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	var beforeDataJSON, afterDataJSON sql.NullString
	if beforeData != nil {
		if data, err := json.Marshal(beforeData); err == nil {
			beforeDataJSON = sql.NullString{String: string(data), Valid: true}
		}
	}
	if afterData != nil {
		if data, err := json.Marshal(afterData); err == nil {
			afterDataJSON = sql.NullString{String: string(data), Valid: true}
		}
	}

	// 获取地理位置信息（如果 GeoIP 已启用）
	var location sql.NullString
	if ipAddress != "" && geoip.IsEnabled() {
		location = geoip.GetLocationWithCache(ipAddress)
	}

	log := models.SubscriptionLog{
		SubscriptionID: subscriptionID,
		UserID:         userID,
		ActionType:     actionType,
		ActionBy:       database.NullString(actionBy),
		IPAddress:      database.NullString(ipAddress),
		Location:       location, // 保存地理位置信息
		BeforeData:     beforeDataJSON,
		AfterData:      afterDataJSON,
		Description:    database.NullString(description),
	}

	if actionByUserID != nil {
		log.ActionByUserID = database.NullInt64(MustSafeUintToInt64(*actionByUserID))
	}

	return db.Create(&log).Error
}

// ==========================================
// 余额日志记录
// ==========================================

// CreateBalanceLog 创建余额日志
func CreateBalanceLog(userID uint, changeType string, amount, balanceBefore, balanceAfter float64, relatedOrderID, relatedRecordID *uint, description, operator string, operatorUserID *uint, ipAddress string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	return CreateBalanceLogWithDB(db, userID, changeType, amount, balanceBefore, balanceAfter, relatedOrderID, relatedRecordID, description, operator, operatorUserID, ipAddress)
}

func CreateBalanceLogWithDB(db *gorm.DB, userID uint, changeType string, amount, balanceBefore, balanceAfter float64, relatedOrderID, relatedRecordID *uint, description, operator string, operatorUserID *uint, ipAddress string) error {
	// 获取地理位置信息（如果 GeoIP 已启用）
	var location sql.NullString
	if ipAddress != "" && geoip.IsEnabled() {
		location = geoip.GetLocationWithCache(ipAddress)
	}

	log := models.BalanceLog{
		UserID:        userID,
		ChangeType:    changeType,
		Amount:        amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
		Description:   database.NullString(description),
		Operator:      database.NullString(operator),
		IPAddress:     database.NullString(ipAddress),
		Location:      location, // 保存地理位置信息
	}

	if relatedOrderID != nil {
		log.RelatedOrderID = database.NullInt64(MustSafeUintToInt64(*relatedOrderID))
	}

	if relatedRecordID != nil {
		log.RelatedRecordID = database.NullInt64(MustSafeUintToInt64(*relatedRecordID))
	}

	if operatorUserID != nil {
		log.OperatorUserID = database.NullInt64(MustSafeUintToInt64(*operatorUserID))
	}

	return db.Create(&log).Error
}

// ==========================================
// 佣金日志记录
// ==========================================

// CreateCommissionLog 创建佣金日志
func CreateCommissionLog(inviterID, inviteeID uint, commissionType string, amount float64, inviteRelationID, relatedOrderID *uint, description string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	return CreateCommissionLogWithDB(db, inviterID, inviteeID, commissionType, amount, inviteRelationID, relatedOrderID, description)
}

func CreateCommissionLogWithDB(db *gorm.DB, inviterID, inviteeID uint, commissionType string, amount float64, inviteRelationID, relatedOrderID *uint, description string) error {
	log := models.CommissionLog{
		InviterID:      inviterID,
		InviteeID:      inviteeID,
		CommissionType: commissionType,
		Amount:         amount,
		Status:         "pending",
		Description:    database.NullString(description),
	}

	if inviteRelationID != nil {
		log.InviteRelationID = database.NullInt64(MustSafeUintToInt64(*inviteRelationID))
	}

	if relatedOrderID != nil {
		log.RelatedOrderID = database.NullInt64(MustSafeUintToInt64(*relatedOrderID))
	}

	return db.Create(&log).Error
}

// UpdateCommissionLogStatus 更新佣金日志状态
func UpdateCommissionLogStatus(logID uint, status string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	return db.Model(&models.CommissionLog{}).Where("id = ?", logID).Updates(map[string]interface{}{
		"status": status,
	}).Error
}
