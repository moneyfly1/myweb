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

func CreateAuditLog(c *gin.Context, actionType, resourceType string, resourceID uint, description string, beforeData, afterData interface{}) {
	db := database.GetDB()
	if db == nil {
		if userID, exists := c.Get("user_id"); exists {
			if uid, ok := userID.(uint); ok {
				LogAudit(uid, actionType, resourceType, resourceID, description)
			}
		}
		return
	}

	var userID sql.NullInt64
	if uid, exists := c.Get("user_id"); exists {
		if u, ok := uid.(uint); ok {
			userID = sql.NullInt64{Int64: int64(u), Valid: true}
		}
	}

	ipAddress := GetRealClientIP(c)

	userAgent := c.GetHeader("User-Agent")

	var location sql.NullString
	if ipAddress != "" {
		location = geoip.GetLocationString(ipAddress)
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

	var responseStatus sql.NullInt64
	if status, exists := c.Get("response_status"); exists {
		if s, ok := status.(int); ok {
			responseStatus = sql.NullInt64{Int64: int64(s), Valid: true}
		}
	} else {
		responseStatus = sql.NullInt64{Int64: http.StatusOK, Valid: true}
	}

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

	go func() {
		if err := db.Create(&auditLog).Error; err != nil {
			if userID.Valid {
				LogAudit(uint(userID.Int64), actionType, resourceType, resourceID, description)
			}
			if AppLogger != nil {
				AppLogger.Error("保存审计日志失败: %v", err)
			}
		}
	}()
}

func CreateAuditLogSimple(c *gin.Context, actionType, resourceType string, resourceID uint, description string) {
	CreateAuditLog(c, actionType, resourceType, resourceID, description, nil, nil)
}

func CreateAuditLogWithData(c *gin.Context, actionType, resourceType string, resourceID uint, description string, beforeData, afterData interface{}) {
	CreateAuditLog(c, actionType, resourceType, resourceID, description, beforeData, afterData)
}

func SetResponseStatus(c *gin.Context, status int) {
	c.Set("response_status", status)
}

func CreateSecurityLog(c *gin.Context, eventType, severity, description string, additionalData map[string]interface{}) {
	db := database.GetDB()
	if db == nil {
		if AppLogger != nil {
			AppLogger.Warn("[安全日志] %s - %s: %s", severity, eventType, description)
		}
		return
	}

	ipAddress := GetRealClientIP(c)
	if ipAddress == "" {
		ipAddress = c.ClientIP()
	}

	userAgent := c.GetHeader("User-Agent")

	var location sql.NullString
	if ipAddress != "" {
		location = geoip.GetLocationString(ipAddress)
	}

	var additionalDataJSON sql.NullString
	if additionalData != nil && len(additionalData) > 0 {
		if data, err := json.Marshal(additionalData); err == nil {
			additionalDataJSON = sql.NullString{String: string(data), Valid: true}
		}
	}

	var userID sql.NullInt64
	if uid, exists := c.Get("user_id"); exists {
		if u, ok := uid.(uint); ok {
			userID = sql.NullInt64{Int64: int64(u), Valid: true}
		}
	}

	var responseStatus sql.NullInt64
	switch eventType {
	case "login_success":
		responseStatus = sql.NullInt64{Int64: http.StatusOK, Valid: true}
	case "login_attempt":
		responseStatus = sql.NullInt64{Int64: http.StatusOK, Valid: true}
	case "login_failed", "login_blocked", "ip_blocked":
		responseStatus = sql.NullInt64{Int64: http.StatusUnauthorized, Valid: true}
	case "login_rate_limit":
		responseStatus = sql.NullInt64{Int64: http.StatusTooManyRequests, Valid: true}
	default:
		switch severity {
		case "CRITICAL", "HIGH":
			responseStatus = sql.NullInt64{Int64: http.StatusForbidden, Valid: true}
		case "MEDIUM":
			responseStatus = sql.NullInt64{Int64: http.StatusTooManyRequests, Valid: true}
		default:
			responseStatus = sql.NullInt64{Int64: http.StatusUnauthorized, Valid: true}
		}
	}

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

	go func() {
		if err := db.Create(&auditLog).Error; err != nil {
			if AppLogger != nil {
				AppLogger.Error("[安全日志保存失败] %s - %s: %s, 错误: %v", severity, eventType, description, err)
			}
		} else {
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

	var recentAttempts int64
	db.Model(&models.AuditLog{}).
		Where("ip_address = ? AND action_type LIKE ? AND created_at > ?",
			ipAddress, "security_login_attempt%", now.Add(-1*time.Minute)).
		Count(&recentAttempts)

	if recentAttempts >= 10 {
		return true, fmt.Sprintf("检测到批量登录行为：IP %s 在1分钟内尝试登录 %d 次", ipAddress, recentAttempts)
	}

	var uniqueUsernames int64
	db.Model(&models.AuditLog{}).
		Where("ip_address = ? AND action_type LIKE ? AND created_at > ?",
			ipAddress, "security_login_attempt%", now.Add(-5*time.Minute)).
		Group("before_data").
		Count(&uniqueUsernames)

	if uniqueUsernames >= 5 {
		return true, fmt.Sprintf("检测到撞库行为：IP %s 在5分钟内尝试登录 %d 个不同用户名", ipAddress, uniqueUsernames)
	}

	if username != "" {
		var uniqueIPs int64
		db.Model(&models.AuditLog{}).
			Where("action_type LIKE ? AND before_data LIKE ? AND created_at > ?",
				"security_login_attempt%", "%"+username+"%", now.Add(-10*time.Minute)).
			Group("ip_address").
			Count(&uniqueIPs)

		if uniqueIPs >= 3 {
			return true, fmt.Sprintf("检测到账户定向攻击：用户名 %s 在10分钟内被 %d 个不同IP尝试登录", username, uniqueIPs)
		}
	}

	return false, ""
}

func CreateSystemErrorLog(c *gin.Context, statusCode int, message string, err error) {
	db := database.GetDB()
	if db == nil {
		if AppLogger != nil {
			AppLogger.Error("[系统错误] %s: %v", message, err)
		}
		return
	}

	ipAddress := GetRealClientIP(c)
	if ipAddress == "" {
		ipAddress = c.ClientIP()
	}

	userAgent := c.GetHeader("User-Agent")

	var location sql.NullString
	if ipAddress != "" {
		location = geoip.GetLocationString(ipAddress)
	}

	var userID sql.NullInt64
	if uid, exists := c.Get("user_id"); exists {
		if u, ok := uid.(uint); ok {
			userID = sql.NullInt64{Int64: int64(u), Valid: true}
		}
	}

	errorDetails := map[string]interface{}{
		"message": message,
		"path":    c.Request.URL.Path,
		"method":  c.Request.Method,
	}
	if err != nil {
		errorDetails["error"] = err.Error()
	}

	var errorDetailsJSON sql.NullString
	if data, err := json.Marshal(errorDetails); err == nil {
		errorDetailsJSON = sql.NullString{String: string(data), Valid: true}
	}

	auditLog := models.AuditLog{
		UserID:            userID,
		ActionType:        "system_error",
		ResourceType:      sql.NullString{String: "system", Valid: true},
		ResourceID:        sql.NullInt64{Valid: false},
		ActionDescription: sql.NullString{String: fmt.Sprintf("系统错误: %s", message), Valid: true},
		IPAddress:         sql.NullString{String: ipAddress, Valid: ipAddress != ""},
		UserAgent:         sql.NullString{String: userAgent, Valid: userAgent != ""},
		Location:          location,
		RequestMethod:     sql.NullString{String: c.Request.Method, Valid: true},
		RequestPath:       sql.NullString{String: c.Request.URL.Path, Valid: true},
		ResponseStatus:    sql.NullInt64{Int64: int64(statusCode), Valid: true},
		BeforeData:        errorDetailsJSON,
		AfterData:         sql.NullString{Valid: false},
	}

	go func() {
		errDetail := err
		if saveErr := db.Create(&auditLog).Error; saveErr != nil {
			if AppLogger != nil {
				AppLogger.Error("[系统错误日志保存失败] %s: %v, 错误: %v", message, errDetail, saveErr)
			}
		} else {
			if AppLogger != nil {
				errorMsg := fmt.Sprintf("[系统错误] 状态码:%d | 路径:%s | 错误:%s", statusCode, c.Request.URL.Path, message)
				if errDetail != nil {
					errorMsg += fmt.Sprintf(" | 详细错误:%v", errDetail)
				}
				AppLogger.Error(errorMsg)
			}
		}
	}()
}
