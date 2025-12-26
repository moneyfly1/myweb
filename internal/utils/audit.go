package utils

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/geoip"

	"github.com/gin-gonic/gin"
)

// CreateAuditLog 创建审计日志记录到数据库
func CreateAuditLog(c *gin.Context, actionType, resourceType string, resourceID uint, description string, beforeData, afterData interface{}) {
	db := database.GetDB()
	if db == nil {
		// 如果数据库未初始化，只记录到文件日志
		if userID, exists := c.Get("user_id"); exists {
			if uid, ok := userID.(uint); ok {
				LogAudit(uid, actionType, resourceType, resourceID, description)
			}
		}
		return
	}

	// 获取用户ID
	var userID sql.NullInt64
	if uid, exists := c.Get("user_id"); exists {
		if u, ok := uid.(uint); ok {
			userID = sql.NullInt64{Int64: int64(u), Valid: true}
		}
	}

	// 获取IP地址（使用统一的真实IP获取函数）
	ipAddress := GetRealClientIP(c)

	// 获取User-Agent
	userAgent := c.GetHeader("User-Agent")

	// 解析地理位置（如果 GeoIP 已启用）
	var location sql.NullString
	if ipAddress != "" {
		location = geoip.GetLocationString(ipAddress)
	}

	// 序列化前后数据
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

	// 获取响应状态码
	var responseStatus sql.NullInt64
	if status, exists := c.Get("response_status"); exists {
		if s, ok := status.(int); ok {
			responseStatus = sql.NullInt64{Int64: int64(s), Valid: true}
		}
	} else {
		// 如果没有设置，尝试从响应中获取
		responseStatus = sql.NullInt64{Int64: http.StatusOK, Valid: true}
	}

	// 创建审计日志
	auditLog := models.AuditLog{
		UserID:            userID,
		ActionType:        actionType,
		ResourceType:      sql.NullString{String: resourceType, Valid: resourceType != ""},
		ResourceID:        sql.NullInt64{Int64: int64(resourceID), Valid: resourceID > 0},
		ActionDescription: sql.NullString{String: description, Valid: description != ""},
		IPAddress:         sql.NullString{String: ipAddress, Valid: ipAddress != ""},
		UserAgent:         sql.NullString{String: userAgent, Valid: userAgent != ""},
		Location:          location,
		RequestMethod:     sql.NullString{String: c.Request.Method, Valid: true},
		RequestPath:       sql.NullString{String: c.Request.URL.Path, Valid: true},
		ResponseStatus:    responseStatus,
		BeforeData:        beforeDataJSON,
		AfterData:         afterDataJSON,
	}

	// 异步保存，避免影响主流程
	go func() {
		if err := db.Create(&auditLog).Error; err != nil {
			// 如果保存失败，至少记录到文件日志
			if userID.Valid {
				LogAudit(uint(userID.Int64), actionType, resourceType, resourceID, description)
			}
			if AppLogger != nil {
				AppLogger.Error("保存审计日志失败: %v", err)
			}
		}
	}()
}

// CreateAuditLogSimple 创建简单的审计日志（不需要前后数据）
func CreateAuditLogSimple(c *gin.Context, actionType, resourceType string, resourceID uint, description string) {
	CreateAuditLog(c, actionType, resourceType, resourceID, description, nil, nil)
}

// CreateAuditLogWithData 创建带前后数据的审计日志
func CreateAuditLogWithData(c *gin.Context, actionType, resourceType string, resourceID uint, description string, beforeData, afterData interface{}) {
	CreateAuditLog(c, actionType, resourceType, resourceID, description, beforeData, afterData)
}

// SetResponseStatus 设置响应状态码（用于审计日志）
func SetResponseStatus(c *gin.Context, status int) {
	c.Set("response_status", status)
}

// CreateSecurityLog 创建安全日志（记录登录尝试、密码错误、IP封禁、批量登录等安全事件）
func CreateSecurityLog(c *gin.Context, eventType, severity, description string, additionalData map[string]interface{}) {
	db := database.GetDB()
	if db == nil {
		// 如果数据库未初始化，只记录到文件日志
		if AppLogger != nil {
			AppLogger.Warn("[安全日志] %s - %s: %s", severity, eventType, description)
		}
		return
	}

	// 获取IP地址
	ipAddress := GetRealClientIP(c)
	if ipAddress == "" {
		ipAddress = c.ClientIP()
	}

	// 获取User-Agent
	userAgent := c.GetHeader("User-Agent")

	// 解析地理位置
	var location sql.NullString
	if ipAddress != "" {
		location = geoip.GetLocationString(ipAddress)
	}

	// 序列化附加数据
	var additionalDataJSON sql.NullString
	if additionalData != nil && len(additionalData) > 0 {
		if data, err := json.Marshal(additionalData); err == nil {
			additionalDataJSON = sql.NullString{String: string(data), Valid: true}
		}
	}

	// 获取用户ID（如果存在）
	var userID sql.NullInt64
	if uid, exists := c.Get("user_id"); exists {
		if u, ok := uid.(uint); ok {
			userID = sql.NullInt64{Int64: int64(u), Valid: true}
		}
	}

	// 确定响应状态码（根据严重程度）
	var responseStatus sql.NullInt64
	switch severity {
	case "CRITICAL", "HIGH":
		responseStatus = sql.NullInt64{Int64: http.StatusForbidden, Valid: true}
	case "MEDIUM":
		responseStatus = sql.NullInt64{Int64: http.StatusTooManyRequests, Valid: true}
	default:
		responseStatus = sql.NullInt64{Int64: http.StatusUnauthorized, Valid: true}
	}

	// 创建安全审计日志
	auditLog := models.AuditLog{
		UserID:            userID,
		ActionType:        "security_" + eventType, // 使用 security_ 前缀标识安全事件
		ResourceType:      sql.NullString{String: "security", Valid: true},
		ResourceID:        sql.NullInt64{Valid: false}, // 安全事件通常没有资源ID
		ActionDescription: sql.NullString{String: fmt.Sprintf("[%s] %s", severity, description), Valid: true},
		IPAddress:         sql.NullString{String: ipAddress, Valid: ipAddress != ""},
		UserAgent:         sql.NullString{String: userAgent, Valid: userAgent != ""},
		Location:          location,
		RequestMethod:     sql.NullString{String: c.Request.Method, Valid: true},
		RequestPath:       sql.NullString{String: c.Request.URL.Path, Valid: true},
		ResponseStatus:    responseStatus,
		BeforeData:        additionalDataJSON, // 将附加数据存储在 BeforeData 中
		AfterData:         sql.NullString{Valid: false},
	}

	// 异步保存，避免影响主流程
	go func() {
		if err := db.Create(&auditLog).Error; err != nil {
			// 如果保存失败，至少记录到文件日志
			if AppLogger != nil {
				AppLogger.Error("[安全日志保存失败] %s - %s: %s, 错误: %v", severity, eventType, description, err)
			}
		} else {
			// 记录到文件日志（用于实时监控）
			if AppLogger != nil {
				logMsg := fmt.Sprintf("[安全事件] IP:%s | 类型:%s | 严重程度:%s | 描述:%s",
					ipAddress, eventType, severity, description)
				if additionalData != nil {
					if data, err := json.Marshal(additionalData); err == nil {
						logMsg += fmt.Sprintf(" | 附加信息:%s", string(data))
					}
				}

				switch severity {
				case "CRITICAL":
					AppLogger.Error(logMsg)
				case "HIGH":
					AppLogger.Error(logMsg)
				case "MEDIUM":
					AppLogger.Warn(logMsg)
				default:
					AppLogger.Info(logMsg)
				}
			}
		}
	}()
}

// CheckBruteForcePattern 检测批量登录和撞库行为
func CheckBruteForcePattern(c *gin.Context, username string) (isSuspicious bool, reason string) {
	db := database.GetDB()
	if db == nil {
		return false, ""
	}

	ipAddress := GetRealClientIP(c)
	if ipAddress == "" {
		ipAddress = c.ClientIP()
	}

	now := GetBeijingTime()

	// 检查1分钟内同一IP尝试登录不同用户名的次数（撞库检测）
	var recentAttempts int64
	db.Model(&models.AuditLog{}).
		Where("ip_address = ? AND action_type LIKE ? AND created_at > ?",
			ipAddress, "security_login_attempt%", now.Add(-1*time.Minute)).
		Count(&recentAttempts)

	if recentAttempts >= 10 {
		// 1分钟内尝试10次以上，可能是批量登录
		return true, fmt.Sprintf("检测到批量登录行为：IP %s 在1分钟内尝试登录 %d 次", ipAddress, recentAttempts)
	}

	// 检查5分钟内同一IP尝试登录不同用户名的数量（撞库检测）
	var uniqueUsernames int64
	db.Model(&models.AuditLog{}).
		Where("ip_address = ? AND action_type LIKE ? AND created_at > ?",
			ipAddress, "security_login_attempt%", now.Add(-5*time.Minute)).
		Group("before_data").
		Count(&uniqueUsernames)

	if uniqueUsernames >= 5 {
		// 5分钟内尝试5个以上不同用户名，可能是撞库
		return true, fmt.Sprintf("检测到撞库行为：IP %s 在5分钟内尝试登录 %d 个不同用户名", ipAddress, uniqueUsernames)
	}

	// 检查同一用户名在短时间内被多个IP尝试（账户定向攻击）
	if username != "" {
		var uniqueIPs int64
		db.Model(&models.AuditLog{}).
			Where("action_type LIKE ? AND before_data LIKE ? AND created_at > ?",
				"security_login_attempt%", "%"+username+"%", now.Add(-10*time.Minute)).
			Group("ip_address").
			Count(&uniqueIPs)

		if uniqueIPs >= 3 {
			// 10分钟内同一用户名被3个以上不同IP尝试，可能是账户定向攻击
			return true, fmt.Sprintf("检测到账户定向攻击：用户名 %s 在10分钟内被 %d 个不同IP尝试登录", username, uniqueIPs)
		}
	}

	return false, ""
}
