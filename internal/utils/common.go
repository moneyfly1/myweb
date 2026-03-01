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
	"os"
	"path/filepath"
	"strings"
	"time"

	"cboard-go/internal/core/config"

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
	if err := db.Table(tableName).Where("order_no LIKE ?", fullPrefix+"%").Order("order_no DESC").Limit(100).Pluck("order_no", &orderNos).Error; err != nil {
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
	if err := db.Table(tableName).Where("order_no = ?", orderNo).Count(&count).Error; err != nil {
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
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	infoFile, err := os.OpenFile(filepath.Join(logDir, "app.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	errorFile, err := os.OpenFile(filepath.Join(logDir, "error.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
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

func CreateAccessToken(userID uint, email string, isAdmin bool) (string, error) {
	cfg := config.AppConfig
	if cfg == nil {
		return "", errors.New("配置未初始化")
	}

	expiresAt := time.Now().Add(time.Duration(cfg.AccessTokenExpireMinutes) * time.Minute)

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

func CreateRefreshToken(userID uint, email string) (string, error) {
	cfg := config.AppConfig
	if cfg == nil {
		return "", errors.New("配置未初始化")
	}

	expiresAt := time.Now().Add(time.Duration(cfg.RefreshTokenExpireDays) * 24 * time.Hour)

	claims := JWTClaims{
		UserID: userID,
		Email:  email,
		Type:   "refresh",
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
	var total float64
	start, end := GetDayRange(GetBeijingTime())
	query := db.Table("orders").Where("created_at >= ? AND created_at < ?", start, end)

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
