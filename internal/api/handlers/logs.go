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
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// PaginationParams 分页参数
type PaginationParams struct {
	Page     int
	PageSize int
	Offset   int
}

// parseLogsPaginationParams 解析日志分页参数
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

// applyAuditLogFilters 应用审计日志筛选条件
func applyAuditLogFilters(query *gorm.DB, c *gin.Context) *gorm.DB {
	// 日志级别筛选
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

	// 模块筛选
	if module := strings.TrimSpace(c.Query("module")); module != "" {
		query = query.Where("resource_type LIKE ?", "%"+module+"%")
	}

	// 用户名筛选
	if username := strings.TrimSpace(c.Query("username")); username != "" {
		query = query.Joins("JOIN users ON audit_logs.user_id = users.id").
			Where("users.username LIKE ?", "%"+username+"%")
	}

	// 关键词筛选
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		query = query.Where("action_description LIKE ? OR action_type LIKE ? OR resource_type LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 开始时间筛选
	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", startTime); err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}

	// 结束时间筛选
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", endTime); err == nil {
			query = query.Where("created_at <= ?", t)
		}
	}

	return query
}

// getLogLevel 获取日志级别
func getLogLevel(log models.AuditLog) string {
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

// getLogLevelCN 获取日志级别（中文）
func getLogLevelCN(log models.AuditLog) string {
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

// getLogUsername 获取日志用户名
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

// formatAuditLogForAPI 格式化审计日志为API响应格式
func formatAuditLogForAPI(db *gorm.DB, log models.AuditLog) gin.H {
	username := getLogUsername(db, log)
	level := getLogLevel(log)

	// 获取消息
	message := ""
	if log.ActionDescription.Valid {
		message = log.ActionDescription.String
	} else {
		message = log.ActionType
	}

	// 解析BeforeData中的JSON数据（用于安全日志）
	var failureReason string
	var additionalInfo map[string]interface{}
	if log.BeforeData.Valid && strings.HasPrefix(log.ActionType, "security_") {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(log.BeforeData.String), &data); err == nil {
			additionalInfo = data
			// 提取失败原因
			if reason, ok := data["reason"].(string); ok {
				failureReason = reason
			}
			// 如果是登录失败，显示更详细的信息
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

	// 构建详情
	details := ""
	if log.BeforeData.Valid || log.AfterData.Valid {
		var detailParts []string
		if log.BeforeData.Valid {
			if failureReason != "" {
				// 如果有解析出的失败原因，优先显示
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

	// 构建上下文
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

	result := gin.H{
		"id":          log.ID,
		"timestamp":   log.CreatedAt.Format("2006-01-02 15:04:05"),
		"level":       level,
		"module":      getNullableStringValue(log.ResourceType),
		"message":     message,
		"username":    username,
		"ip_address":  getNullableStringValue(log.IPAddress),
		"user_agent":  getNullableStringValue(log.UserAgent),
		"action_type": log.ActionType,
		"details":     details,
		"context":     context,
	}

	// 如果是安全日志，添加失败原因字段
	if failureReason != "" {
		result["failure_reason"] = failureReason
	}
	if additionalInfo != nil {
		result["additional_info"] = additionalInfo
	}

	return result
}

// getNullableStringValue 获取可空字符串值
func getNullableStringValue(v sql.NullString) string {
	if v.Valid {
		return v.String
	}
	return ""
}

// formatLogForCSV 格式化日志为CSV格式
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

	// 转义CSV中的特殊字符
	message = strings.ReplaceAll(message, "\"", "\"\"")
	message = strings.ReplaceAll(message, "\n", " ")
	message = strings.ReplaceAll(message, "\r", " ")

	return fmt.Sprintf("%s,%s,%s,%s,%s,%s,\"%s\"\n",
		log.CreatedAt.Format("2006-01-02 15:04:05"),
		level,
		getNullableStringValue(log.ResourceType),
		username,
		getNullableStringValue(log.IPAddress),
		log.ActionType,
		message,
	)
}

// GetAuditLogs 获取审计日志
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

// GetLoginAttempts 获取登录尝试记录
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

// GetSystemLogs 获取系统日志
func GetSystemLogs(c *gin.Context) {
	pagination := parseLogsPaginationParams(c, 1, 20)
	db := database.GetDB()

	// 构建查询（带Preload）
	query := db.Model(&models.AuditLog{}).Preload("User")
	query = applyAuditLogFilters(query, c)

	// 构建Count查询（不带Preload，避免JOIN影响计数）
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

	// 转换为前端期望的格式
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

// GetLogsStats 获取日志统计
func GetLogsStats(c *gin.Context) {
	db := database.GetDB()

	var stats struct {
		Total   int64 `json:"total"`
		Error   int64 `json:"error"`
		Warning int64 `json:"warning"`
		Info    int64 `json:"info"`
	}

	// 总日志数
	db.Model(&models.AuditLog{}).Count(&stats.Total)

	// 错误日志（响应状态 >= 400）
	db.Model(&models.AuditLog{}).Where("response_status >= ?", 400).Count(&stats.Error)

	// 警告日志（响应状态 300-399）
	db.Model(&models.AuditLog{}).Where("response_status >= ? AND response_status < ?", 300, 400).Count(&stats.Warning)

	// 信息日志（响应状态 < 300 或 NULL）
	db.Model(&models.AuditLog{}).Where("response_status < ? OR response_status IS NULL", 300).Count(&stats.Info)

	utils.SuccessResponse(c, http.StatusOK, "", stats)
}

// ExportLogs 导出日志
func ExportLogs(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.AuditLog{}).Preload("User")

	// 应用筛选条件
	query = applyAuditLogFilters(query, c)

	var logs []models.AuditLog
	if err := query.Order("created_at DESC").Limit(10000).Find(&logs).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "导出日志失败", err)
		return
	}

	// 生成CSV内容
	var csvContent strings.Builder
	csvContent.WriteString("\xEF\xBB\xBF") // UTF-8 BOM
	csvContent.WriteString("时间,级别,模块,用户,IP地址,操作类型,日志内容\n")

	for _, log := range logs {
		csvContent.WriteString(formatLogForCSV(db, log))
	}

	// 设置响应头
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=system_logs_%s.csv", time.Now().Format("20060102")))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", []byte(csvContent.String()))
}

// ClearLogs 清空日志
func ClearLogs(c *gin.Context) {
	db := database.GetDB()

	// 删除所有审计日志
	result := db.Where("1 = 1").Delete(&models.AuditLog{})

	if result.Error != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "清空日志失败", result.Error)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("已清空 %d 条日志", result.RowsAffected), gin.H{
		"deleted_count": result.RowsAffected,
	})
}
