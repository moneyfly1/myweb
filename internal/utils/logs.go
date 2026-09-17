package utils

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/geoip"

	"gorm.io/gorm"
)

// ==========================================
// 注册日志记录
// ==========================================

// RegistrationLogInput 注册成功日志的写入参数。
// 用结构体而不是一长串位置参数：此前失败日志正是"少传一个字段"导致用户名常年为空，
// 字段命名后调用方漏传会一眼看出来。
type RegistrationLogInput struct {
	UserID     uint
	Username   string
	Email      string
	IPAddress  string
	UserAgent  string
	Source     string // direct（自助注册）/ invite_code（邀请码注册）/ admin（管理员创建）
	InviteCode string
	InviterID  *uint
}

// 注册来源常量（写入 registration_logs.register_source）
const (
	RegisterSourceDirect     = "direct"
	RegisterSourceInviteCode = "invite_code"
	RegisterSourceAdmin      = "admin"
)

// CreateRegistrationLog 创建注册成功日志
func CreateRegistrationLog(in RegistrationLogInput) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	var location sql.NullString
	if in.IPAddress != "" && geoip.IsEnabled() {
		location = geoip.GetLocationWithCache(in.IPAddress)
	}

	source := in.Source
	if source == "" {
		source = RegisterSourceDirect
	}

	log := models.RegistrationLog{
		UserID:         in.UserID,
		Username:       in.Username,
		Email:          in.Email,
		IPAddress:      database.NullString(in.IPAddress),
		UserAgent:      database.NullString(in.UserAgent),
		Location:       location,
		Status:         "success",
		RegisterSource: database.NullString(source),
	}

	if in.InviteCode != "" {
		log.InviteCode = database.NullString(in.InviteCode)
	}

	if in.InviterID != nil {
		log.InviterID = database.NullInt64(MustSafeUintToInt64(*in.InviterID))
	}

	return db.Create(&log).Error
}

// CreateRegistrationLogFailed 创建注册失败日志。
//
// 失败日志里的 username/email 是"注册时提交的内容"，对应用户并不存在（没有建号）；
// 此前只记邮箱、不记用户名，导致注册日志列表里失败行的"用户名"列全为空，
// 管理员看不出被尝试占用的究竟是哪个用户名（也无法从别处找回）。
func CreateRegistrationLogFailed(username, email, ipAddress, userAgent, reason string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	var location sql.NullString
	if ipAddress != "" && geoip.IsEnabled() {
		location = geoip.GetLocationWithCache(ipAddress)
	}

	log := models.RegistrationLog{
		Username:      strings.TrimSpace(username),
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
	// 佣金余额在发放点即时到账（见 auth.go/order.go 的 balance + reward），
	// 故这里直接记"已结算"，避免日志长期显示"待结算"与实际资金状态矛盾。
	now := GetBeijingTime()
	log := models.CommissionLog{
		InviterID:      inviterID,
		InviteeID:      inviteeID,
		CommissionType: commissionType,
		Amount:         amount,
		Status:         "paid",
		SettledAt:      sql.NullTime{Time: now, Valid: true},
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
