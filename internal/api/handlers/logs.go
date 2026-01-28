package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/geoip"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PaginationParams struct {
	Page     int
	PageSize int
	Offset   int
}

func parseLogsPaginationParams(c *gin.Context, defaultPage, defaultPageSize int) PaginationParams {
	page, _ := strconv.Atoi(c.DefaultQuery("page", strconv.Itoa(defaultPage)))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(defaultPageSize)))
	if pageSizeStr := c.Query("size"); pageSizeStr != "" {
		fmt.Sscanf(pageSizeStr, "%d", &pageSize)
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = defaultPageSize
	}

	return PaginationParams{
		Page:     page,
		PageSize: pageSize,
		Offset:   (page - 1) * pageSize,
	}
}

func applyAuditLogFilters(query *gorm.DB, c *gin.Context) *gorm.DB {
	if logLevel := strings.TrimSpace(c.Query("log_level")); logLevel != "" {
		switch logLevel {
		case "error":
			query = query.Where("response_status >= ?", 400)
		case "warning":
			query = query.Where("response_status >= ? AND response_status < ?", 300, 400)
		case "info":
			query = query.Where("response_status < ? OR response_status IS NULL", 300)
		}
	}

	if module := strings.TrimSpace(c.Query("module")); module != "" {
		query = query.Where("resource_type LIKE ?", "%"+module+"%")
	}

	if username := strings.TrimSpace(c.Query("username")); username != "" {
		query = query.Joins("JOIN users ON audit_logs.user_id = users.id").
			Where("users.username LIKE ?", "%"+username+"%")
	}

	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		query = query.Where("action_description LIKE ? OR action_type LIKE ? OR resource_type LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", startTime); err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}

	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", endTime); err == nil {
			query = query.Where("created_at <= ?", t)
		}
	}

	return query
}

func getLogLevel(log models.AuditLog) string {
	actionType := log.ActionType

	if actionType == "system_error" {
		return "error"
	}

	if strings.HasPrefix(actionType, "security_") {
		switch actionType {
		case "security_login_success", "security_login_attempt":
			return "info"
		case "security_login_failed", "security_login_blocked", "security_ip_blocked":
			return "error"
		case "security_login_rate_limit":
			return "warning"
		default:
			if log.ActionDescription.Valid {
				desc := log.ActionDescription.String
				if strings.Contains(desc, "[CRITICAL]") || strings.Contains(desc, "[HIGH]") {
					return "error"
				}
				if strings.Contains(desc, "[MEDIUM]") {
					return "warning"
				}
			}
		}
	}

	if actionType == "login" {
		return "info"
	}

	if !log.ResponseStatus.Valid {
		return "info"
	}
	status := log.ResponseStatus.Int64
	if status >= 500 || status >= 400 {
		return "error"
	}
	if status >= 300 {
		return "warning"
	}
	return "info"
}

func getLogLevelCN(log models.AuditLog) string {
	actionType := log.ActionType

	if actionType == "system_error" {
		return "错误"
	}

	if strings.HasPrefix(actionType, "security_") {
		switch actionType {
		case "security_login_success", "security_login_attempt":
			return "信息"
		case "security_login_failed", "security_login_blocked", "security_ip_blocked":
			return "错误"
		case "security_login_rate_limit":
			return "警告"
		default:
			if log.ActionDescription.Valid {
				desc := log.ActionDescription.String
				if strings.Contains(desc, "[CRITICAL]") || strings.Contains(desc, "[HIGH]") {
					return "错误"
				}
				if strings.Contains(desc, "[MEDIUM]") {
					return "警告"
				}
			}
		}
	}

	if actionType == "login" {
		return "信息"
	}

	if !log.ResponseStatus.Valid {
		return "信息"
	}
	status := log.ResponseStatus.Int64
	if status >= 400 {
		return "错误"
	}
	if status >= 300 {
		return "警告"
	}
	return "信息"
}

func getLogUsername(db *gorm.DB, log models.AuditLog) string {
	if !log.UserID.Valid {
		return ""
	}
	if log.User.ID == 0 {
		var user models.User
		if db.First(&user, log.UserID.Int64).Error == nil {
			return user.Username
		}
		return ""
	}
	return log.User.Username
}

func formatAuditLogForAPI(db *gorm.DB, log models.AuditLog) gin.H {
	username := getLogUsername(db, log)
	level := getLogLevel(log)

	message := ""
	if log.ActionDescription.Valid {
		message = log.ActionDescription.String
	} else {
		message = log.ActionType
	}

	var failureReason string
	var additionalInfo map[string]interface{}
	if log.BeforeData.Valid && strings.HasPrefix(log.ActionType, "security_") {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(log.BeforeData.String), &data); err == nil {
			additionalInfo = data
			if reason, ok := data["reason"].(string); ok {
				failureReason = reason
			}
			if log.ActionType == "security_login_failed" {
				if email, ok := data["email"].(string); ok {
					failureReason += fmt.Sprintf(" (邮箱: %s)", email)
				}
				if username, ok := data["username"].(string); ok {
					failureReason += fmt.Sprintf(" (用户名: %s)", username)
				}
				if locked, ok := data["locked"].(bool); ok && locked {
					failureReason += " [IP已被封禁]"
				}
			}
		}
	}

	details := ""
	if log.BeforeData.Valid || log.AfterData.Valid {
		var detailParts []string
		if log.BeforeData.Valid {
			if failureReason != "" {
				detailParts = append(detailParts, "失败原因: "+failureReason)
			} else {
				detailParts = append(detailParts, "Before: "+log.BeforeData.String)
			}
		}
		if log.AfterData.Valid {
			detailParts = append(detailParts, "After: "+log.AfterData.String)
		}
		details = strings.Join(detailParts, "\n")
	}

	context := gin.H{}
	if log.RequestMethod.Valid {
		context["method"] = log.RequestMethod.String
	}
	if log.RequestPath.Valid {
		context["path"] = log.RequestPath.String
	}
	if log.RequestParams.Valid {
		context["params"] = log.RequestParams.String
	}
	if log.ResponseStatus.Valid {
		context["status"] = log.ResponseStatus.Int64
	}

	var locationDisplay string
	var locationInfo map[string]interface{}
	if log.Location.Valid && log.Location.String != "" {
		var location geoip.LocationInfo
		if err := json.Unmarshal([]byte(log.Location.String), &location); err == nil {
			if location.City != "" {
				locationDisplay = fmt.Sprintf("%s, %s", location.Country, location.City)
			} else {
				locationDisplay = location.Country
			}
			locationInfo = map[string]interface{}{
				"country":      location.Country,
				"country_code": location.CountryCode,
				"city":         location.City,
				"region":       location.Region,
			}
		} else {
			locationDisplay = log.Location.String
		}
	} else if log.IPAddress.Valid && log.IPAddress.String != "" {
		if geoip.IsEnabled() {
			location, err := geoip.GetLocationWithFallback(log.IPAddress.String)
			if err == nil && location != nil {
				if location.City != "" {
					locationDisplay = fmt.Sprintf("%s, %s", location.Country, location.City)
				} else if location.Region != "" {
					locationDisplay = fmt.Sprintf("%s, %s", location.Country, location.Region)
				} else {
					locationDisplay = location.Country
				}
				locationInfo = map[string]interface{}{
					"country":      location.Country,
					"country_code": location.CountryCode,
					"city":         location.City,
					"region":       location.Region,
				}
			} else {
				locationStr := geoip.GetLocationSimple(log.IPAddress.String)
				if locationStr != "" {
					locationDisplay = locationStr
				}
			}
		}
	}

	beijingTime := utils.ToBeijingTime(log.CreatedAt)

	result := gin.H{
		"id":          log.ID,
		"timestamp":   beijingTime.Format("2006-01-02 15:04:05"),
		"level":       level,
		"module":      getNullableStringValue(log.ResourceType),
		"message":     message,
		"username":    username,
		"ip_address":  getNullableStringValue(log.IPAddress),
		"location":    locationDisplay, // 添加地理位置显示
		"user_agent":  getNullableStringValue(log.UserAgent),
		"action_type": log.ActionType,
		"details":     details,
		"context":     context,
	}

	if failureReason != "" {
		result["failure_reason"] = failureReason
	}
	if additionalInfo != nil {
		result["additional_info"] = additionalInfo
	}
	if locationInfo != nil {
		result["location_info"] = locationInfo
	}

	return result
}

func getNullableStringValue(v sql.NullString) string {
	if v.Valid {
		return v.String
	}
	return ""
}

func formatLogForCSV(db *gorm.DB, log models.AuditLog) string {
	level := getLogLevelCN(log)
	username := ""
	if log.UserID.Valid && log.User.ID > 0 {
		username = log.User.Username
	}

	message := ""
	if log.ActionDescription.Valid {
		message = log.ActionDescription.String
	} else {
		message = log.ActionType
	}

	message = strings.ReplaceAll(message, "\"", "\"\"")
	message = strings.ReplaceAll(message, "\n", " ")
	message = strings.ReplaceAll(message, "\r", " ")

	beijingTime := utils.ToBeijingTime(log.CreatedAt)

	return fmt.Sprintf("%s,%s,%s,%s,%s,%s,\"%s\"\n",
		beijingTime.Format("2006-01-02 15:04:05"),
		level,
		getNullableStringValue(log.ResourceType),
		username,
		getNullableStringValue(log.IPAddress),
		log.ActionType,
		message,
	)
}

func GetAuditLogs(c *gin.Context) {
	pagination := parseLogsPaginationParams(c, 1, 20)
	db := database.GetDB()

	var logs []models.AuditLog
	var total int64

	db.Model(&models.AuditLog{}).Count(&total)
	db.Preload("User").Order("created_at DESC").
		Offset(pagination.Offset).Limit(pagination.PageSize).Find(&logs)

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"logs":      logs,
		"total":     total,
		"page":      pagination.Page,
		"page_size": pagination.PageSize,
	})
}

func GetLoginAttempts(c *gin.Context) {
	pagination := parseLogsPaginationParams(c, 1, 20)
	db := database.GetDB()

	var attempts []models.LoginAttempt
	var total int64

	db.Model(&models.LoginAttempt{}).Count(&total)
	db.Order("created_at DESC").
		Offset(pagination.Offset).Limit(pagination.PageSize).Find(&attempts)

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"attempts":  attempts,
		"total":     total,
		"page":      pagination.Page,
		"page_size": pagination.PageSize,
	})
}

func GetSystemLogs(c *gin.Context) {
	pagination := parseLogsPaginationParams(c, 1, 20)
	db := database.GetDB()

	query := db.Model(&models.AuditLog{}).Preload("User")
	query = applyAuditLogFilters(query, c)

	countQuery := db.Model(&models.AuditLog{})
	countQuery = applyAuditLogFilters(countQuery, c)

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取系统日志总数失败", err)
		return
	}

	var logs []models.AuditLog
	if err := query.Order("created_at DESC").
		Offset(pagination.Offset).Limit(pagination.PageSize).Find(&logs).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取系统日志失败", err)
		return
	}

	logList := make([]gin.H, 0, len(logs))
	for _, log := range logs {
		logList = append(logList, formatAuditLogForAPI(db, log))
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"logs":  logList,
		"total": total,
		"page":  pagination.Page,
		"size":  pagination.PageSize,
	})
}

func GetLogsStats(c *gin.Context) {
	db := database.GetDB()

	var stats struct {
		Total   int64 `json:"total"`
		Error   int64 `json:"error"`
		Warning int64 `json:"warning"`
		Info    int64 `json:"info"`
	}

	db.Model(&models.AuditLog{}).Count(&stats.Total)

	db.Model(&models.AuditLog{}).Where("response_status >= ?", 400).Count(&stats.Error)

	db.Model(&models.AuditLog{}).Where("response_status >= ? AND response_status < ?", 300, 400).Count(&stats.Warning)

	db.Model(&models.AuditLog{}).Where("response_status < ? OR response_status IS NULL", 300).Count(&stats.Info)

	utils.SuccessResponse(c, http.StatusOK, "", stats)
}

func ExportLogs(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.AuditLog{}).Preload("User")

	query = applyAuditLogFilters(query, c)

	var logs []models.AuditLog
	if err := query.Order("created_at DESC").Limit(10000).Find(&logs).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "导出日志失败", err)
		return
	}

	var csvContent strings.Builder
	csvContent.WriteString("\xEF\xBB\xBF") // UTF-8 BOM
	csvContent.WriteString("时间,级别,模块,用户,IP地址,操作类型,日志内容\n")

	for _, log := range logs {
		csvContent.WriteString(formatLogForCSV(db, log))
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=system_logs_%s.csv", time.Now().Format("20060102")))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", []byte(csvContent.String()))
}

func ClearLogs(c *gin.Context) {
	db := database.GetDB()

	result := db.Where("1 = 1").Delete(&models.AuditLog{})

	if result.Error != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "清空日志失败", result.Error)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("已清空 %d 条日志", result.RowsAffected), gin.H{
		"deleted_count": result.RowsAffected,
	})
}
