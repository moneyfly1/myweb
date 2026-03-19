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

func buildRequestParams(c *gin.Context) sql.NullString {
	if c == nil || c.Request == nil {
		return sql.NullString{Valid: false}
	}

	params := make(map[string]interface{})

	if len(c.Params) > 0 {
		pathParams := make(map[string]string, len(c.Params))
		for _, p := range c.Params {
			pathParams[p.Key] = p.Value
		}
		params["path"] = pathParams
	}

	queryParams := c.Request.URL.Query()
	if len(queryParams) > 0 {
		params["query"] = queryParams
	}

	if len(params) == 0 {
		return sql.NullString{Valid: false}
	}

	b, err := json.Marshal(params)
	if err != nil {
		return sql.NullString{Valid: false}
	}

	return sql.NullString{String: string(b), Valid: true}
}

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
			userID = sql.NullInt64{Int64: MustSafeUintToInt64(u), Valid: true}
		}
	}

	ipAddress := GetRealClientIP(c)

	userAgent := c.GetHeader("User-Agent")

	var location sql.NullString
	if ipAddress != "" {
		location = geoip.GetLocationWithCache(ipAddress)
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
		ResourceID:        sql.NullInt64{Int64: MustSafeUintToInt64(resourceID), Valid: resourceID > 0},
		ActionDescription: sql.NullString{String: description, Valid: description != ""},
		IPAddress:         sql.NullString{String: ipAddress, Valid: ipAddress != ""},
		UserAgent:         sql.NullString{String: userAgent, Valid: userAgent != ""},
		Location:          location,
		RequestMethod:     sql.NullString{String: c.Request.Method, Valid: true},
		RequestPath:       sql.NullString{String: c.Request.URL.Path, Valid: true},
		RequestParams:     buildRequestParams(c),
		ResponseStatus:    responseStatus,
		BeforeData:        beforeDataJSON,
		AfterData:         afterDataJSON,
	}

	if err := db.Create(&auditLog).Error; err != nil {
		if userID.Valid {
			LogAudit(MustSafeInt64ToUint(userID.Int64), actionType, resourceType, resourceID, description)
		}
		if AppLogger != nil {
			AppLogger.Error("保存审计日志失败: %v", err)
		}
	}
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
		location = geoip.GetLocationWithCache(ipAddress)
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
			userID = sql.NullInt64{Int64: MustSafeUintToInt64(u), Valid: true}
		}
	}

	var responseStatus sql.NullInt64
	switch eventType {
	case "login_success", "admin_login_success", "register_success", "user_unlock", "user_enabled":
		responseStatus = sql.NullInt64{Int64: http.StatusOK, Valid: true}
	case "login_attempt":
		responseStatus = sql.NullInt64{Int64: http.StatusOK, Valid: true}
	case "login_failed", "login_blocked", "ip_blocked", "password_change_failed", "reset_code_failed":
		responseStatus = sql.NullInt64{Int64: http.StatusUnauthorized, Valid: true}
	case "login_rate_limit", "register_rate_limit", "verify_code_rate_limit":
		responseStatus = sql.NullInt64{Int64: http.StatusTooManyRequests, Valid: true}
	case "register_ip_blocked":
		responseStatus = sql.NullInt64{Int64: http.StatusTooManyRequests, Valid: true}
	case "admin_login_as", "user_disabled":
		responseStatus = sql.NullInt64{Int64: http.StatusOK, Valid: true} // 管理员操作 200
	case "auth_token_invalid", "auth_token_blacklisted", "admin_forbidden":
		responseStatus = sql.NullInt64{Int64: http.StatusUnauthorized, Valid: true}
	case "csrf_validation_failed":
		responseStatus = sql.NullInt64{Int64: http.StatusForbidden, Valid: true}
	case "refresh_token_invalid":
		responseStatus = sql.NullInt64{Int64: http.StatusUnauthorized, Valid: true}
	case "verification_code_failed":
		responseStatus = sql.NullInt64{Int64: http.StatusBadRequest, Valid: true}
	case "password_reset_requested":
		responseStatus = sql.NullInt64{Int64: http.StatusOK, Valid: true}
	case "admin_reset_password":
		responseStatus = sql.NullInt64{Int64: http.StatusOK, Valid: true}
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
		RequestParams:     buildRequestParams(c),
		ResponseStatus:    responseStatus,
		BeforeData:        additionalDataJSON, // 将附加数据存储在 BeforeData 中
		AfterData:         sql.NullString{Valid: false},
	}

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
				AppLogger.Error("%s", logMsg)
			case "HIGH":
				AppLogger.Error("%s", logMsg)
			case "MEDIUM":
				AppLogger.Warn("%s", logMsg)
			default:
				AppLogger.Info("%s", logMsg)
			}
		}
	}
}

// CreateBusinessLog 记录业务/操作类日志（支付回调失败、订阅拒绝、Token 无效等），便于在系统日志中排查问题
// c 可为 nil（如定时任务内调用），此时不记录 IP/Path
func CreateBusinessLog(c *gin.Context, actionType, description, level string, data map[string]interface{}) {
	db := database.GetDB()
	if db == nil {
		if AppLogger != nil {
			AppLogger.Warn("[业务日志] %s: %s", actionType, description)
		}
		return
	}

	var ipAddress, userAgent, method, path string
	var location sql.NullString
	var userID sql.NullInt64
	requestParams := sql.NullString{Valid: false}
	if c != nil {
		ipAddress = GetRealClientIP(c)
		if ipAddress == "" {
			ipAddress = c.ClientIP()
		}
		userAgent = c.GetHeader("User-Agent")
		method = c.Request.Method
		path = c.Request.URL.Path
		if ipAddress != "" {
			location = geoip.GetLocationWithCache(ipAddress)
		}
		if uid, exists := c.Get("user_id"); exists {
			if u, ok := uid.(uint); ok {
				userID = sql.NullInt64{Int64: MustSafeUintToInt64(u), Valid: true}
			}
		}
		requestParams = buildRequestParams(c)
	}

	var dataJSON sql.NullString
	if data != nil && len(data) > 0 {
		if b, err := json.Marshal(data); err == nil {
			dataJSON = sql.NullString{String: string(b), Valid: true}
		}
	}

	var responseStatus sql.NullInt64
	switch level {
	case "error":
		responseStatus = sql.NullInt64{Int64: http.StatusInternalServerError, Valid: true}
	case "warning":
		responseStatus = sql.NullInt64{Int64: http.StatusBadRequest, Valid: true}
	default:
		responseStatus = sql.NullInt64{Int64: http.StatusOK, Valid: true}
	}

	auditLog := models.AuditLog{
		UserID:            userID,
		ActionType:        "business_" + actionType,
		ResourceType:      sql.NullString{String: "system", Valid: true},
		ResourceID:        sql.NullInt64{Valid: false},
		ActionDescription: sql.NullString{String: description, Valid: true},
		IPAddress:         sql.NullString{String: ipAddress, Valid: ipAddress != ""},
		UserAgent:         sql.NullString{String: userAgent, Valid: userAgent != ""},
		Location:          location,
		RequestMethod:     sql.NullString{String: method, Valid: method != ""},
		RequestPath:       sql.NullString{String: path, Valid: path != ""},
		RequestParams:     requestParams,
		ResponseStatus:    responseStatus,
		BeforeData:        dataJSON,
		AfterData:         sql.NullString{Valid: false},
	}

	if err := db.Create(&auditLog).Error; err != nil && AppLogger != nil {
		AppLogger.Error("[业务日志保存失败] %s: %s, 错误: %v", actionType, description, err)
	}
}

// CreateBusinessLogFast 快速记录业务日志，跳过 GeoIP 查询（用于高频操作如签到）
func CreateBusinessLogFast(c *gin.Context, actionType, description, level string, data map[string]interface{}) {
	db := database.GetDB()
	if db == nil {
		if AppLogger != nil {
			AppLogger.Warn("[业务日志] %s: %s", actionType, description)
		}
		return
	}

	var ipAddress, userAgent, method, path string
	var userID sql.NullInt64
	requestParams := sql.NullString{Valid: false}
	if c != nil {
		ipAddress = GetRealClientIP(c)
		if ipAddress == "" {
			ipAddress = c.ClientIP()
		}
		userAgent = c.GetHeader("User-Agent")
		method = c.Request.Method
		path = c.Request.URL.Path
		if uid, exists := c.Get("user_id"); exists {
			if u, ok := uid.(uint); ok {
				userID = sql.NullInt64{Int64: MustSafeUintToInt64(u), Valid: true}
			}
		}
		requestParams = buildRequestParams(c)
	}

	var dataJSON sql.NullString
	if data != nil && len(data) > 0 {
		if b, err := json.Marshal(data); err == nil {
			dataJSON = sql.NullString{String: string(b), Valid: true}
		}
	}

	var responseStatus sql.NullInt64
	switch level {
	case "error":
		responseStatus = sql.NullInt64{Int64: http.StatusInternalServerError, Valid: true}
	case "warning":
		responseStatus = sql.NullInt64{Int64: http.StatusBadRequest, Valid: true}
	default:
		responseStatus = sql.NullInt64{Int64: http.StatusOK, Valid: true}
	}

	auditLog := models.AuditLog{
		UserID:            userID,
		ActionType:        "business_" + actionType,
		ResourceType:      sql.NullString{String: "system", Valid: true},
		ResourceID:        sql.NullInt64{Valid: false},
		ActionDescription: sql.NullString{String: description, Valid: true},
		IPAddress:         sql.NullString{String: ipAddress, Valid: ipAddress != ""},
		UserAgent:         sql.NullString{String: userAgent, Valid: userAgent != ""},
		Location:          sql.NullString{Valid: false}, // 跳过 GeoIP 查询
		RequestMethod:     sql.NullString{String: method, Valid: method != ""},
		RequestPath:       sql.NullString{String: path, Valid: path != ""},
		RequestParams:     requestParams,
		ResponseStatus:    responseStatus,
		BeforeData:        dataJSON,
		AfterData:         sql.NullString{Valid: false},
	}

	if err := db.Create(&auditLog).Error; err != nil && AppLogger != nil {
		AppLogger.Error("[业务日志保存失败] %s: %s, 错误: %v", actionType, description, err)
	}
}

// CreateBusinessLogAsync 异步记录业务日志（用于超高频操作如订阅拉取）
func CreateBusinessLogAsync(c *gin.Context, actionType, description, level string, data map[string]interface{}) {
	// 复制必要的上下文信息
	var ipAddress, userAgent, method, path string
	var userID sql.NullInt64
	if c != nil {
		ipAddress = GetRealClientIP(c)
		if ipAddress == "" {
			ipAddress = c.ClientIP()
		}
		userAgent = c.GetHeader("User-Agent")
		method = c.Request.Method
		path = c.Request.URL.Path
		if uid, exists := c.Get("user_id"); exists {
			if u, ok := uid.(uint); ok {
				userID = sql.NullInt64{Int64: MustSafeUintToInt64(u), Valid: true}
			}
		}
	}

	// 异步执行
	go func() {
		db := database.GetDB()
		if db == nil {
			if AppLogger != nil {
				AppLogger.Warn("[业务日志] %s: %s", actionType, description)
			}
			return
		}

		var dataJSON sql.NullString
		if data != nil && len(data) > 0 {
			if b, err := json.Marshal(data); err == nil {
				dataJSON = sql.NullString{String: string(b), Valid: true}
			}
		}

		var responseStatus sql.NullInt64
		switch level {
		case "error":
			responseStatus = sql.NullInt64{Int64: http.StatusInternalServerError, Valid: true}
		case "warning":
			responseStatus = sql.NullInt64{Int64: http.StatusBadRequest, Valid: true}
		default:
			responseStatus = sql.NullInt64{Int64: http.StatusOK, Valid: true}
		}

		auditLog := models.AuditLog{
			UserID:            userID,
			ActionType:        "business_" + actionType,
			ResourceType:      sql.NullString{String: "system", Valid: true},
			ResourceID:        sql.NullInt64{Valid: false},
			ActionDescription: sql.NullString{String: description, Valid: true},
			IPAddress:         sql.NullString{String: ipAddress, Valid: ipAddress != ""},
			UserAgent:         sql.NullString{String: userAgent, Valid: userAgent != ""},
			Location:          sql.NullString{Valid: false}, // 跳过 GeoIP 查询
			RequestMethod:     sql.NullString{String: method, Valid: method != ""},
			RequestPath:       sql.NullString{String: path, Valid: path != ""},
			RequestParams:     sql.NullString{Valid: false},
			ResponseStatus:    responseStatus,
			BeforeData:        dataJSON,
			AfterData:         sql.NullString{Valid: false},
		}

		if err := db.Create(&auditLog).Error; err != nil && AppLogger != nil {
			AppLogger.Error("[业务日志保存失败] %s: %s, 错误: %v", actionType, description, err)
		}
	}()
}

// CreateAuditLogSimpleFast 快速创建简单审计日志，跳过 GeoIP 查询
func CreateAuditLogSimpleFast(c *gin.Context, actionType, resourceType string, resourceID uint, description string) {
	db := database.GetDB()
	if db == nil {
		return
	}

	var ipAddress, userAgent string
	var userID sql.NullInt64
	if c != nil {
		ipAddress = GetRealClientIP(c)
		if ipAddress == "" {
			ipAddress = c.ClientIP()
		}
		userAgent = c.GetHeader("User-Agent")
		if uid, exists := c.Get("user_id"); exists {
			if u, ok := uid.(uint); ok {
				userID = sql.NullInt64{Int64: MustSafeUintToInt64(u), Valid: true}
			}
		}
	}

	auditLog := models.AuditLog{
		UserID:            userID,
		ActionType:        actionType,
		ResourceType:      sql.NullString{String: resourceType, Valid: true},
		ResourceID:        sql.NullInt64{Int64: MustSafeUintToInt64(resourceID), Valid: resourceID > 0},
		ActionDescription: sql.NullString{String: description, Valid: true},
		IPAddress:         sql.NullString{String: ipAddress, Valid: ipAddress != ""},
		UserAgent:         sql.NullString{String: userAgent, Valid: userAgent != ""},
		Location:          sql.NullString{Valid: false}, // 跳过 GeoIP 查询
		ResponseStatus:    sql.NullInt64{Int64: http.StatusOK, Valid: true},
	}

	if err := db.Create(&auditLog).Error; err != nil && AppLogger != nil {
		AppLogger.Error("[审计日志保存失败] %s: %s, 错误: %v", actionType, description, err)
	}
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
	if c != nil {
		c.Set("system_error_logged", true)
	}

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
		location = geoip.GetLocationWithCache(ipAddress)
	}

	var userID sql.NullInt64
	if uid, exists := c.Get("user_id"); exists {
		if u, ok := uid.(uint); ok {
			userID = sql.NullInt64{Int64: MustSafeUintToInt64(u), Valid: true}
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
		RequestParams:     buildRequestParams(c),
		ResponseStatus:    sql.NullInt64{Int64: int64(statusCode), Valid: true},
		BeforeData:        errorDetailsJSON,
		AfterData:         sql.NullString{Valid: false},
	}

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
			AppLogger.Error("%s", errorMsg)
		}
	}
}

// ==========================================
// 系统日志记录
// ==========================================

// CreateSystemLog 创建系统级别的日志（定时任务、系统状态等）
func CreateSystemLog(actionType, description string, level string, data map[string]interface{}) error {
	db := database.GetDB()
	if db == nil {
		if AppLogger != nil {
			AppLogger.Info("[系统日志] %s: %s", actionType, description)
		}
		return fmt.Errorf("数据库未初始化")
	}

	var dataJSON sql.NullString
	if data != nil {
		if dataBytes, err := json.Marshal(data); err == nil {
			dataJSON = sql.NullString{String: string(dataBytes), Valid: true}
		}
	}

	// 根据级别设置响应状态码
	var responseStatus sql.NullInt64
	switch level {
	case "error", "ERROR":
		responseStatus = sql.NullInt64{Int64: 500, Valid: true}
	case "warning", "WARNING":
		responseStatus = sql.NullInt64{Int64: 300, Valid: true}
	default:
		responseStatus = sql.NullInt64{Int64: 200, Valid: true}
	}

	auditLog := models.AuditLog{
		UserID:            sql.NullInt64{Valid: false}, // 系统日志没有用户ID
		ActionType:        actionType,
		ResourceType:      sql.NullString{String: "system", Valid: true},
		ResourceID:        sql.NullInt64{Valid: false},
		ActionDescription: sql.NullString{String: description, Valid: true},
		IPAddress:         sql.NullString{Valid: false},
		UserAgent:         sql.NullString{Valid: false},
		Location:          sql.NullString{Valid: false},
		RequestMethod:     sql.NullString{Valid: false},
		RequestPath:       sql.NullString{Valid: false},
		RequestParams:     sql.NullString{Valid: false},
		ResponseStatus:    responseStatus,
		BeforeData:        dataJSON,
		AfterData:         sql.NullString{Valid: false},
	}

	if err := db.Create(&auditLog).Error; err != nil {
		if AppLogger != nil {
			AppLogger.Error("[系统日志保存失败] %s: %s, 错误: %v", actionType, description, err)
		}
	}

	return nil
}

// CreateSchedulerLog 创建定时任务日志
func CreateSchedulerLog(taskName, status, message string, data map[string]interface{}) error {
	actionType := fmt.Sprintf("scheduler_%s", taskName)
	description := fmt.Sprintf("[定时任务] %s: %s", taskName, message)

	level := "info"
	if status == "error" || status == "failed" {
		level = "error"
	} else if status == "warning" {
		level = "warning"
	}

	return CreateSystemLog(actionType, description, level, data)
}
