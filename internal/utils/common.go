package utils

import (
	crand "crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Base64Encode(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}

// NormalizeEmail 将邮箱转为小写并去空格，用于注册/登录等场景防止同一邮箱不同写法重复注册
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func GenerateUUID() string {
	return uuid.New().String()
}

func GenerateSubscriptionURL() string {
	b := make([]byte, 16)
	if _, err := crand.Read(b); err != nil {
		log.Printf("failed to generate random bytes: %v", err)
		// Fallback to timestamp-based generation
		return fmt.Sprintf("%032x", time.Now().UnixNano())
	}
	// 使用十六进制编码生成32位随机字符串，例如: a8f3c9e1b7d4f6a2c8e9d0b1a3c5e7f9
	return hex.EncodeToString(b)
}

// 订单号生成器接口
type orderNoGenerator interface {
	getMaxSequence() int
	checkExists(orderNo string) bool
	getTableName() string
	getPrefix() string
}

// 订单生成器
type orderGenerator struct {
	db *gorm.DB
}

func (og *orderGenerator) getTableName() string { return "orders" }
func (og *orderGenerator) getPrefix() string    { return "ORD" }
func (og *orderGenerator) getMaxSequence() int {
	if og.db == nil {
		return 0
	}
	return findMaxOrderSequence(og.db)
}
func (og *orderGenerator) checkExists(orderNo string) bool {
	if og.db == nil {
		return false
	}
	return checkOrderNoExists(og.db, orderNo)
}

// 充值订单生成器
type rechargeOrderGenerator struct {
	db *gorm.DB
}

func (og *rechargeOrderGenerator) getTableName() string { return "recharge_records" }
func (og *rechargeOrderGenerator) getPrefix() string    { return "RCH" }
func (og *rechargeOrderGenerator) getMaxSequence() int {
	if og.db == nil {
		return 0
	}
	return findMaxRechargeOrderSequence(og.db)
}
func (og *rechargeOrderGenerator) checkExists(orderNo string) bool {
	if og.db == nil {
		return false
	}
	return checkRechargeOrderNoExists(og.db, orderNo)
}

// 设备升级订单生成器
type deviceUpgradeOrderGenerator struct {
	db *gorm.DB
}

func (og *deviceUpgradeOrderGenerator) getTableName() string { return "orders" }
func (og *deviceUpgradeOrderGenerator) getPrefix() string    { return "UPG" }
func (og *deviceUpgradeOrderGenerator) getMaxSequence() int {
	if og.db == nil {
		return 0
	}
	return findMaxOrderSequence(og.db)
}
func (og *deviceUpgradeOrderGenerator) checkExists(orderNo string) bool {
	if og.db == nil {
		return false
	}
	return checkOrderNoExists(og.db, orderNo)
}

// 专门查询订单序列号（硬编码表名，避免SQL注入）
func findMaxOrderSequence(db *gorm.DB) int {
	var maxSeq int
	dateStr := GetBeijingTime().Format("20060102")
	fullPrefix := fmt.Sprintf("ORD%s", dateStr)

	var orderNos []string
	// 使用硬编码表名，无SQL注入风险
	if err := db.Table("orders").Where("order_no LIKE ?", fullPrefix+"%").Order("order_no DESC").Limit(100).Pluck("order_no", &orderNos).Error; err != nil {
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

// 专门查询充值订单序列号（硬编码表名）
func findMaxRechargeOrderSequence(db *gorm.DB) int {
	var maxSeq int
	dateStr := GetBeijingTime().Format("20060102")
	fullPrefix := fmt.Sprintf("RCH%s", dateStr)

	var orderNos []string
	// 使用硬编码表名，无SQL注入风险
	if err := db.Table("recharge_records").Where("order_no LIKE ?", fullPrefix+"%").Order("order_no DESC").Limit(100).Pluck("order_no", &orderNos).Error; err != nil {
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

// 检查订单号是否存在（硬编码表名）
func checkOrderNoExists(db *gorm.DB, orderNo string) bool {
	var count int64
	if err := db.Table("orders").Where("order_no = ?", orderNo).Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

// 检查充值订单号是否存在（硬编码表名）
func checkRechargeOrderNoExists(db *gorm.DB, orderNo string) bool {
	var count int64
	if err := db.Table("recharge_records").Where("order_no = ?", orderNo).Count(&count).Error; err != nil {
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

func generateOrderNo(gen orderNoGenerator) (string, error) {
	now := GetBeijingTime()
	dateStr := now.Format("20060102")
	fullPrefix := fmt.Sprintf("%s%s", gen.getPrefix(), dateStr)

	maxSeq := gen.getMaxSequence()
	maxSeq = incrementSequence(maxSeq)
	orderNo := fmt.Sprintf("%s%03d", fullPrefix, maxSeq)

	// 最多重试10次避免重复
	for i := 0; i < 10; i++ {
		if !gen.checkExists(orderNo) {
			break
		}
		maxSeq = incrementSequence(maxSeq)
		orderNo = fmt.Sprintf("%s%03d", fullPrefix, maxSeq)
	}

	return orderNo, nil
}

func GenerateOrderNo(db interface{}) (string, error) {
	var gen orderNoGenerator
	if db != nil {
		if gormDB, ok := db.(*gorm.DB); ok {
			gen = &orderGenerator{db: gormDB}
		}
	}
	if gen == nil {
		gen = &orderGenerator{}
	}
	return generateOrderNo(gen)
}

func GenerateRechargeOrderNo(userID uint, db interface{}) (string, error) {
	var gen orderNoGenerator
	if db != nil {
		if gormDB, ok := db.(*gorm.DB); ok {
			gen = &rechargeOrderGenerator{db: gormDB}
		}
	}
	if gen == nil {
		gen = &rechargeOrderGenerator{}
	}
	return generateOrderNo(gen)
}

func GenerateDeviceUpgradeOrderNo(db interface{}) (string, error) {
	var gen orderNoGenerator
	if db != nil {
		if gormDB, ok := db.(*gorm.DB); ok {
			gen = &deviceUpgradeOrderGenerator{db: gormDB}
		}
	}
	if gen == nil {
		gen = &deviceUpgradeOrderGenerator{}
	}
	return generateOrderNo(gen)
}

func GenerateCouponCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	for i := range b {
		randBytes := make([]byte, 1)
		if _, err := crand.Read(randBytes); err != nil {
			log.Printf("failed to generate random bytes: %v", err)
			// Fallback to simple random
			b[i] = charset[int(time.Now().UnixNano())%len(charset)]
			continue
		}
		b[i] = charset[int(randBytes[0])%len(charset)]
	}
	return string(b)
}

func GenerateTicketNo(userID uint) string {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 2)
	if _, err := crand.Read(randomBytes); err != nil {
		log.Printf("failed to generate random bytes: %v", err)
		// Fallback to timestamp-based random
		// #nosec G115 - Safe conversion: timestamp % 256 is always in byte range [0, 255]
		randomBytes = []byte{byte(timestamp % 256), byte((timestamp / 256) % 256)} // #nosec G115
	}
	randomStr := base64.URLEncoding.EncodeToString(randomBytes)[:3]
	return fmt.Sprintf("TKT%d%d%s", timestamp, userID, randomStr)
}

// ========== 常量定义 ==========

const (
	DefaultDeviceLimit    = 0
	DefaultDurationMonths = 0
)

const (
	SubscriptionStatusActive   = "active"
	SubscriptionStatusInactive = "inactive"
	SubscriptionStatusExpired  = "expired"
)

const (
	OrderStatusPending  = "pending"
	OrderStatusPaid     = "paid"
	OrderStatusFailed   = "failed"
	OrderStatusCanceled = "canceled"
)

const (
	VerificationPurposeRegister      = "register"
	VerificationPurposeResetPassword = "reset_password"
	VerificationPurposeChangeEmail   = "change_email"
)

// ========== 时区相关 ==========

var BeijingTZ = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}()

func GetBeijingTime() time.Time {
	return time.Now().In(BeijingTZ)
}

func ToBeijingTime(t time.Time) time.Time {
	return t.In(BeijingTZ)
}

func GetDayRange(t time.Time) (time.Time, time.Time) {
	loc := t.Location()
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	return start, start.Add(24 * time.Hour)
}

func FormatBeijingTime(t time.Time) string {
	return t.In(BeijingTZ).Format("2006-01-02 15:04:05")
}

func FormatBeijingDate(t time.Time) string {
	return t.In(BeijingTZ).Format("2006-01-02")
}

func FormatBeijingRFC3339(t time.Time) string {
	return t.In(BeijingTZ).Format(time.RFC3339)
}

func FormatNullTimeBeijing(nt sql.NullTime) string {
	if !nt.Valid {
		return ""
	}
	return FormatBeijingTime(nt.Time)
}

func ResolveTimezone(timezone string) *time.Location {
	tz := strings.TrimSpace(timezone)
	if tz == "" {
		return BeijingTZ
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return BeijingTZ
	}
	return loc
}

func FormatTimeInTimezone(t time.Time, timezone string) string {
	loc := ResolveTimezone(timezone)
	return t.In(loc).Format("2006-01-02 15:04:05")
}

func FormatNullTimeInTimezone(nt sql.NullTime, timezone string) string {
	if !nt.Valid {
		return ""
	}
	return FormatTimeInTimezone(nt.Time, timezone)
}

// ========== Token哈希 ==========

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// ========== 事务处理 ==========

func WithTransaction(db *gorm.DB, fn func(*gorm.DB) error) error {
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// ========== SQL辅助函数 ==========

func GetNullStringValue(ns sql.NullString) interface{} {
	if ns.Valid {
		return ns.String
	}
	return nil
}

func GetNullInt64Value(ni sql.NullInt64) interface{} {
	if ni.Valid {
		return ni.Int64
	}
	return nil
}

func GetNullFloat64Value(nf sql.NullFloat64) interface{} {
	if nf.Valid {
		return nf.Float64
	}
	return nil
}

func GetNullTimeValue(nt sql.NullTime) interface{} {
	if nt.Valid {
		return FormatBeijingTime(nt.Time)
	}
	return nil
}

func GetStringValue(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}

// ========== 订单查询相关 ==========

func orderRevenueQuery(db *gorm.DB) *gorm.DB {
	return db.Table("orders")
}

func applyStatusFilter(query *gorm.DB, status string) *gorm.DB {
	if status != "" {
		return query.Where("status = ?", status)
	}
	return query
}

func CalculateOrderRevenue(db *gorm.DB, status string) float64 {
	var total float64
	query := applyStatusFilter(orderRevenueQuery(db), status)

	query.Select("COALESCE(SUM(CASE WHEN final_amount IS NOT NULL THEN final_amount ELSE amount END), 0)").
		Scan(&total)

	return total
}

func CalculateRechargeRevenue(db *gorm.DB, status string) float64 {
	var total float64
	query := applyStatusFilter(db.Table("recharge_records"), status)

	query.Select("COALESCE(SUM(amount), 0)").
		Scan(&total)

	return total
}

func CalculateTotalRevenue(db *gorm.DB, status string) float64 {
	return RoundFloat(CalculateOrderRevenue(db, status)+CalculateRechargeRevenue(db, status), 2)
}

type PaymentSummary struct {
	Total        int64
	Pending      int64
	Paid         int64
	Cancelled    int64
	PaidRevenue  float64
	RangePaid    int64
	RangeRevenue float64
}

func CalculatePaymentSummary(db *gorm.DB, rangeStart time.Time, rangeEnd time.Time) PaymentSummary {
	var summary PaymentSummary
	db.Raw(`
		SELECT
			COALESCE(SUM(total), 0) AS total,
			COALESCE(SUM(pending), 0) AS pending,
			COALESCE(SUM(paid), 0) AS paid,
			COALESCE(SUM(cancelled), 0) AS cancelled,
			COALESCE(SUM(paid_revenue), 0) AS paid_revenue,
			COALESCE(SUM(range_paid), 0) AS range_paid,
			COALESCE(SUM(range_revenue), 0) AS range_revenue
		FROM (
			SELECT
				COUNT(*) AS total,
				COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0) AS pending,
				COALESCE(SUM(CASE WHEN status = 'paid' THEN 1 ELSE 0 END), 0) AS paid,
				COALESCE(SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END), 0) AS cancelled,
				COALESCE(SUM(CASE WHEN status = 'paid' THEN
					CASE WHEN final_amount IS NOT NULL THEN final_amount ELSE amount END
				ELSE 0 END), 0) AS paid_revenue,
				COALESCE(SUM(CASE WHEN status = 'paid' AND created_at >= ? AND created_at < ? THEN 1 ELSE 0 END), 0) AS range_paid,
				COALESCE(SUM(CASE WHEN status = 'paid' AND created_at >= ? AND created_at < ? THEN
					CASE WHEN final_amount IS NOT NULL THEN final_amount ELSE amount END
				ELSE 0 END), 0) AS range_revenue
			FROM orders
			UNION ALL
			SELECT
				COUNT(*) AS total,
				COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0) AS pending,
				COALESCE(SUM(CASE WHEN status = 'paid' THEN 1 ELSE 0 END), 0) AS paid,
				COALESCE(SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END), 0) AS cancelled,
				COALESCE(SUM(CASE WHEN status = 'paid' THEN amount ELSE 0 END), 0) AS paid_revenue,
				COALESCE(SUM(CASE WHEN status = 'paid' AND created_at >= ? AND created_at < ? THEN 1 ELSE 0 END), 0) AS range_paid,
				COALESCE(SUM(CASE WHEN status = 'paid' AND created_at >= ? AND created_at < ? THEN amount ELSE 0 END), 0) AS range_revenue
			FROM recharge_records
		) payment_summary
	`, rangeStart, rangeEnd, rangeStart, rangeEnd, rangeStart, rangeEnd, rangeStart, rangeEnd).Scan(&summary)

	summary.PaidRevenue = RoundFloat(summary.PaidRevenue, 2)
	summary.RangeRevenue = RoundFloat(summary.RangeRevenue, 2)
	return summary
}

func FindEnabledPaymentConfig(db *gorm.DB, payType string) (models.PaymentConfig, error) {
	if payType == "" {
		payType = "alipay"
	}

	queryPayType := payType
	if strings.HasPrefix(payType, "yipay_") {
		queryPayType = "yipay"
	} else if strings.HasPrefix(payType, "codepay_") {
		queryPayType = "codepay"
	}

	var paymentConfig models.PaymentConfig
	err := db.Where("pay_type = ? AND status = ?", queryPayType, 1).
		Order("sort_order ASC").
		First(&paymentConfig).Error
	if err == nil {
		return paymentConfig, nil
	}

	err = db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", queryPayType, 1).
		Order("sort_order ASC").
		First(&paymentConfig).Error
	if err == nil {
		return paymentConfig, nil
	}

	if queryPayType != payType {
		err = db.Where("pay_type = ? AND status = ?", payType, 1).
			Order("sort_order ASC").
			First(&paymentConfig).Error
		if err == nil {
			return paymentConfig, nil
		}
		err = db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", payType, 1).
			Order("sort_order ASC").
			First(&paymentConfig).Error
	}
	return paymentConfig, err
}

// ==========================================
// 日志记录器
// ==========================================

type beijingTimeWriter struct {
	writer io.Writer
}

func (w *beijingTimeWriter) Write(p []byte) (n int, err error) {
	beijingTime := GetBeijingTime().Format("2006/01/02 15:04:05")
	timestamp := fmt.Sprintf("[%s] ", beijingTime)
	_, err = w.writer.Write([]byte(timestamp))
	if err != nil {
		return 0, err
	}
	return w.writer.Write(p)
}

type Logger struct {
	infoLog  *log.Logger
	errorLog *log.Logger
	warnLog  *log.Logger
}

var AppLogger *Logger

func InitLogger(logDir string) error {
	// 清理并验证日志目录路径
	cleanLogDir := filepath.Clean(logDir)
	if strings.Contains(cleanLogDir, "..") {
		return fmt.Errorf("不安全的日志目录路径: %s", logDir)
	}

	if err := os.MkdirAll(cleanLogDir, 0750); err != nil {
		return err
	}

	infoLogPath := filepath.Join(cleanLogDir, "app.log")
	errorLogPath := filepath.Join(cleanLogDir, "error.log")

	infoFile, err := os.OpenFile(infoLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}

	errorFile, err := os.OpenFile(errorLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}

	infoWriter := &beijingTimeWriter{writer: infoFile}
	errorWriter := &beijingTimeWriter{writer: errorFile}
	warnWriter := &beijingTimeWriter{writer: os.Stdout}

	AppLogger = &Logger{
		infoLog:  log.New(infoWriter, "[INFO] ", log.Lshortfile),
		errorLog: log.New(errorWriter, "[ERROR] ", log.Lshortfile),
		warnLog:  log.New(warnWriter, "[WARN] ", 0),
	}

	return nil
}

func (l *Logger) Info(format string, v ...interface{}) {
	if l != nil && l.infoLog != nil {
		l.infoLog.Printf(format, v...)
	}
}

func (l *Logger) Error(format string, v ...interface{}) {
	if l != nil && l.errorLog != nil {
		l.errorLog.Printf(format, v...)
	}
}

func (l *Logger) Warn(format string, v ...interface{}) {
	if l != nil && l.warnLog != nil {
		l.warnLog.Printf(format, v...)
	}
}

func LogUserActivity(userID uint, activityType, description string) {
	if AppLogger != nil {
		AppLogger.Info("用户活动: user_id=%d, type=%s, description=%s", userID, activityType, description)
	}
}

func LogAudit(userID uint, actionType, resourceType string, resourceID uint, description string) {
	if AppLogger != nil {
		AppLogger.Info("审计日志: user_id=%d, action=%s, resource=%s:%d, description=%s",
			userID, actionType, resourceType, resourceID, description)
	}
}

func LogInfo(format string, v ...interface{}) {
	if AppLogger != nil {
		AppLogger.Info(format, v...)
	} else {
		log.Printf("[INFO] "+format, v...)
	}
}

func LogWarn(format string, v ...interface{}) {
	if AppLogger != nil {
		AppLogger.Warn(format, v...)
	} else {
		log.Printf("[WARN] "+format, v...)
	}
}

func LogErrorMsg(format string, v ...interface{}) {
	if AppLogger != nil {
		AppLogger.Error(format, v...)
	} else {
		log.Printf("[ERROR] "+format, v...)
	}
}

// ==========================================
// JWT Token 相关
// ==========================================

type JWTClaims struct {
	UserID  uint   `json:"sub"`
	Email   string `json:"email"`
	IsAdmin bool   `json:"is_admin"`
	Type    string `json:"type"`
	jwt.RegisteredClaims
}

// GetSessionTimeout 获取会话超时时间（分钟），优先级：每用户设置 > 全局设置 > .env 配置
func GetSessionTimeout(userID uint) int {
	db := database.GetDB()
	if db != nil {
		// 1. 检查每用户设置
		var userCfg models.SystemConfig
		userKey := fmt.Sprintf("user_%d_session_timeout", userID)
		if err := db.Where("category = ? AND key = ?", "user_security", userKey).First(&userCfg).Error; err == nil {
			if v, err := strconv.Atoi(userCfg.Value); err == nil && v > 0 {
				return v
			}
		}

		// 2. 检查全局安全设置
		var globalCfg models.SystemConfig
		if err := db.Where("category = ? AND key = ?", "security", "session_timeout").First(&globalCfg).Error; err == nil {
			if v, err := strconv.Atoi(globalCfg.Value); err == nil && v > 0 {
				return v
			}
		}
	}

	// 3. 回退到 .env 配置
	cfg := config.AppConfig
	if cfg != nil && cfg.AccessTokenExpireMinutes > 0 {
		return cfg.AccessTokenExpireMinutes
	}
	return 60
}

func CreateAccessToken(userID uint, email string, isAdmin bool) (string, error) {
	cfg := config.AppConfig
	if cfg == nil {
		return "", errors.New("配置未初始化")
	}

	timeoutMinutes := GetSessionTimeout(userID)
	expiresAt := time.Now().Add(time.Duration(timeoutMinutes) * time.Minute)

	claims := JWTClaims{
		UserID:  userID,
		Email:   email,
		IsAdmin: isAdmin,
		Type:    "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.SecretKey))
}

func CreateRefreshToken(userID uint, email string, isAdmin bool) (string, error) {
	cfg := config.AppConfig
	if cfg == nil {
		return "", errors.New("配置未初始化")
	}

	expiresAt := time.Now().Add(time.Duration(cfg.RefreshTokenExpireDays) * 24 * time.Hour)

	claims := JWTClaims{
		UserID:  userID,
		Email:   email,
		IsAdmin: isAdmin,
		Type:    "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.SecretKey))
}

func VerifyToken(tokenString string) (*JWTClaims, error) {
	cfg := config.AppConfig
	if cfg == nil {
		return nil, errors.New("配置未初始化")
	}

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("无效的签名方法")
		}
		return []byte(cfg.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("无效的令牌")
}

func CalculateTodayRevenue(db *gorm.DB, status string) float64 {
	start, end := GetDayRange(GetBeijingTime())
	return CalculatePaymentSummary(db, start, end).RangeRevenue
}

type UserPaymentSummary struct {
	Total      int64
	Pending    int64
	Paid       int64
	Cancelled  int64
	PaidAmount float64
}

func CalculateUserPaymentSummary(db *gorm.DB, userID uint) UserPaymentSummary {
	var summary UserPaymentSummary
	db.Raw(`
		SELECT
			COALESCE(SUM(total), 0) AS total,
			COALESCE(SUM(pending), 0) AS pending,
			COALESCE(SUM(paid), 0) AS paid,
			COALESCE(SUM(cancelled), 0) AS cancelled,
			COALESCE(SUM(paid_amount), 0) AS paid_amount
		FROM (
			SELECT
				COUNT(*) AS total,
				COALESCE(SUM(CASE WHEN LOWER(status) = 'pending' THEN 1 ELSE 0 END), 0) AS pending,
				COALESCE(SUM(CASE WHEN LOWER(status) = 'paid' THEN 1 ELSE 0 END), 0) AS paid,
				COALESCE(SUM(CASE WHEN LOWER(status) = 'cancelled' THEN 1 ELSE 0 END), 0) AS cancelled,
				COALESCE(SUM(CASE WHEN LOWER(status) = 'paid' THEN
					ABS(CASE WHEN final_amount IS NOT NULL THEN final_amount ELSE amount END)
				ELSE 0 END), 0) AS paid_amount
			FROM orders
			WHERE user_id = ?
			UNION ALL
			SELECT
				COUNT(*) AS total,
				COALESCE(SUM(CASE WHEN LOWER(status) = 'pending' THEN 1 ELSE 0 END), 0) AS pending,
				COALESCE(SUM(CASE WHEN LOWER(status) = 'paid' THEN 1 ELSE 0 END), 0) AS paid,
				COALESCE(SUM(CASE WHEN LOWER(status) = 'cancelled' THEN 1 ELSE 0 END), 0) AS cancelled,
				COALESCE(SUM(CASE WHEN LOWER(status) = 'paid' THEN amount ELSE 0 END), 0) AS paid_amount
			FROM recharge_records
			WHERE user_id = ?
		) user_payment_summary
	`, userID, userID).Scan(&summary)

	summary.PaidAmount = RoundFloat(summary.PaidAmount, 2)
	return summary
}

// GenerateRandomString 生成指定长度的随机字符串
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, err := crand.Int(crand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			panic(fmt.Sprintf("随机数生成失败: %v", err))
		}
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

// ParseFloat 解析字符串为float64，失败返回默认值
func ParseFloat(s string, defaultValue float64) float64 {
	if s == "" {
		return defaultValue
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return defaultValue
	}
	return f
}

// ParseInt 解析字符串为int，失败返回默认值
func ParseInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}
	return i
}

// RoundFloat 四舍五入到指定小数位
func RoundFloat(f float64, decimals int) float64 {
	multiplier := math.Pow(10, float64(decimals))
	return math.Round(f*multiplier) / multiplier
}
