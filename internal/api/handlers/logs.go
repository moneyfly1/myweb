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

const (
	DefaultPageSize = 20
)

// PaginationParams 分页参数结构
type PaginationParams struct {
	Page     int
	PageSize int
	Offset   int
}

// ==========================================
// 通用辅助函数 (提取并复用)
// ==========================================

// parsePagination 解析分页参数
func parsePagination(c *gin.Context) PaginationParams {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(DefaultPageSize)))

	// 兼容旧参数 "size"
	if pageSizeStr := c.Query("size"); pageSizeStr != "" {
		fmt.Sscanf(pageSizeStr, "%d", &pageSize)
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = DefaultPageSize
	}

	return PaginationParams{
		Page:     page,
		PageSize: pageSize,
		Offset:   (page - 1) * pageSize,
	}
}

// applyTimeRangeFilter 通用时间范围过滤
func applyTimeRangeFilter(query *gorm.DB, c *gin.Context, dbTimeField string) *gorm.DB {
	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(TimeLayout, startTime); err == nil {
			query = query.Where(fmt.Sprintf("%s >= ?", dbTimeField), t)
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(TimeLayout, endTime); err == nil {
			query = query.Where(fmt.Sprintf("%s <= ?", dbTimeField), t)
		}
	}
	return query
}

// applyCommonKeywordFilter 通用关键词过滤 (针对 AuditLog)
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
		escapedModule := utils.EscapeLikePattern(utils.SanitizeSearchKeyword(module))
		query = query.Where("resource_type LIKE ?", "%"+escapedModule+"%")
	}

	if username := strings.TrimSpace(c.Query("username")); username != "" {
		escapedUsername := utils.EscapeLikePattern(utils.SanitizeSearchKeyword(username))
		query = query.Joins("JOIN users ON audit_logs.user_id = users.id").
			Where("users.username LIKE ?", "%"+escapedUsername+"%")
	}

	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		// 这里不需要再次 Escape，因为通常 utils 内部处理，或者简单的 SQL 注入防护
		// 假设 utils.EscapeLikePattern 只是转义 % 和 _
		k := "%" + utils.EscapeLikePattern(utils.SanitizeSearchKeyword(keyword)) + "%"
		query = query.Where("action_description LIKE ? OR action_type LIKE ? OR resource_type LIKE ?", k, k, k)
	}

	return applyTimeRangeFilter(query, c, "created_at")
}

// genericSuccessResponse 统一的分页响应封装
func genericSuccessResponse(c *gin.Context, data interface{}, total int64, p PaginationParams) {
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"logs":        data, // 注意：某些接口返回字段名叫 "attempts" 或其他，如有特定需求需调整
		"total":       total,
		"page":        p.Page,
		"page_size":   p.PageSize,
		"total_pages": (total + int64(p.PageSize) - 1) / int64(p.PageSize),
	})
}

// ==========================================
// 字段获取与格式化辅助函数
// ==========================================

func getNullableString(v sql.NullString) string {
	if v.Valid {
		return v.String
	}
	return ""
}

func getNullableStringPtr(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func getNullableInt64(v sql.NullInt64) interface{} {
	if v.Valid {
		return v.Int64
	}
	return nil
}

func formatTime(t time.Time) string {
	return utils.ToBeijingTime(t).Format(TimeLayout)
}

func getCommonUserName(user *models.User) string {
	if user != nil && user.ID > 0 {
		return user.Username
	}
	return ""
}

func getLocationDisplay(location sql.NullString, ip sql.NullString) (string, map[string]interface{}) {
	// 尝试从 Location 字段解析
	if location.Valid && location.String != "" {
		var loc geoip.LocationInfo
		if err := json.Unmarshal([]byte(location.String), &loc); err == nil {
			display := loc.Country
			if loc.City != "" {
				display = fmt.Sprintf("%s, %s", loc.Country, loc.City)
			}
			return display, map[string]interface{}{
				"country":      loc.Country,
				"country_code": loc.CountryCode,
				"city":         loc.City,
				"region":       loc.Region,
			}
		}
		return location.String, nil
	}

	// 尝试从 IP 实时解析 (Fallback)
	if ip.Valid && ip.String != "" && geoip.IsEnabled() {
		if loc, err := geoip.GetLocationWithFallback(ip.String); err == nil && loc != nil {
			display := loc.Country
			if loc.City != "" {
				display = fmt.Sprintf("%s, %s", loc.Country, loc.City)
			} else if loc.Region != "" {
				display = fmt.Sprintf("%s, %s", loc.Country, loc.Region)
			}
			return display, map[string]interface{}{
				"country":      loc.Country,
				"country_code": loc.CountryCode,
				"city":         loc.City,
				"region":       loc.Region,
			}
		}
		// 简单解析
		if simple := geoip.GetLocationSimple(ip.String); simple != "" {
			return simple, nil
		}
	}
	return "", nil
}

// ==========================================
// 核心日志逻辑
// ==========================================

func getLogLevel(log models.AuditLog, useCN bool) string {
	actionType := log.ActionType

	// 定义返回值的辅助闭包
	ret := func(cn, en string) string {
		if useCN {
			return cn
		}
		return en
	}

	if actionType == "system_error" {
		return ret("错误", "error")
	}

	if strings.HasPrefix(actionType, "security_") {
		switch actionType {
		case "security_login_success", "security_login_attempt":
			return ret("信息", "info")
		case "security_login_failed", "security_login_blocked", "security_ip_blocked":
			return ret("错误", "error")
		case "security_login_rate_limit":
			return ret("警告", "warning")
		default:
			if log.ActionDescription.Valid {
				desc := log.ActionDescription.String
				if strings.Contains(desc, "[CRITICAL]") || strings.Contains(desc, "[HIGH]") {
					return ret("错误", "error")
				}
				if strings.Contains(desc, "[MEDIUM]") {
					return ret("警告", "warning")
				}
			}
		}
	}

	if actionType == "login" {
		return ret("信息", "info")
	}

	if !log.ResponseStatus.Valid {
		return ret("信息", "info")
	}
	status := log.ResponseStatus.Int64
	if status >= 400 {
		return ret("错误", "error")
	}
	if status >= 300 {
		return ret("警告", "warning")
	}
	return ret("信息", "info")
}

func formatAuditLogForAPI(db *gorm.DB, log models.AuditLog) gin.H {
	// 获取用户名 (如果 log.User 未预加载，且 UserID 有效，则查询)
	username := ""
	if log.User.ID > 0 {
		username = log.User.Username
	} else if log.UserID.Valid {
		var u models.User
		if db.Select("username").First(&u, log.UserID.Int64).Error == nil {
			username = u.Username
		}
	}

	level := getLogLevel(log, false)
	message := getNullableString(log.ActionDescription)
	if message == "" {
		message = log.ActionType
	}

	var failureReason string
	var additionalInfo map[string]interface{}

	// 处理 security_ 前缀的 BeforeData
	if log.BeforeData.Valid && strings.HasPrefix(log.ActionType, "security_") {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(log.BeforeData.String), &data); err == nil {
			additionalInfo = data
			if r, ok := data["reason"].(string); ok {
				failureReason = r
			}
			if log.ActionType == "security_login_failed" {
				if email, ok := data["email"].(string); ok {
					failureReason += fmt.Sprintf(" (邮箱: %s)", email)
				}
				if uName, ok := data["username"].(string); ok {
					failureReason += fmt.Sprintf(" (用户名: %s)", uName)
				}
				if locked, ok := data["locked"].(bool); ok && locked {
					failureReason += " [IP已被封禁]"
				}
			}
		}
	}

	details := ""
	if log.BeforeData.Valid || log.AfterData.Valid {
		var parts []string
		if log.BeforeData.Valid {
			if failureReason != "" {
				parts = append(parts, "失败原因: "+failureReason)
			} else {
				parts = append(parts, "Before: "+log.BeforeData.String)
			}
		}
		if log.AfterData.Valid {
			parts = append(parts, "After: "+log.AfterData.String)
		}
		details = strings.Join(parts, "\n")
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

	locationDisplay, locationInfo := getLocationDisplay(log.Location, log.IPAddress)

	result := gin.H{
		"id":          log.ID,
		"timestamp":   formatTime(log.CreatedAt),
		"level":       level,
		"module":      getNullableString(log.ResourceType),
		"message":     message,
		"username":    username,
		"ip_address":  getNullableString(log.IPAddress),
		"location":    locationDisplay,
		"user_agent":  getNullableString(log.UserAgent),
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

func formatLogForCSV(db *gorm.DB, log models.AuditLog) string {
	level := getLogLevel(log, true) // Use CN
	username := getCommonUserName(&log.User)

	message := getNullableString(log.ActionDescription)
	if message == "" {
		message = log.ActionType
	}

	// CSV Escape
	message = strings.ReplaceAll(message, "\"", "\"\"")
	message = strings.ReplaceAll(message, "\n", " ")
	message = strings.ReplaceAll(message, "\r", " ")

	return fmt.Sprintf("%s,%s,%s,%s,%s,%s,\"%s\"\n",
		formatTime(log.CreatedAt),
		level,
		getNullableString(log.ResourceType),
		username,
		getNullableString(log.IPAddress),
		log.ActionType,
		message,
	)
}

// ==========================================
// Handlers (API 接口)
// ==========================================

func GetAuditLogs(c *gin.Context) {
	p := parsePagination(c)
	db := database.GetDB()

	query := db.Model(&models.AuditLog{})
	query = applyAuditLogFilters(query, c) // 复用过滤器逻辑

	var total int64
	query.Count(&total) // 先统计

	var logs []models.AuditLog
	// 再分页查询 (预加载 User)
	query.Preload("User").Order("created_at DESC").
		Offset(p.Offset).Limit(p.PageSize).Find(&logs)

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"logs":      logs,
		"total":     total,
		"page":      p.Page,
		"page_size": p.PageSize,
	})
}

func GetLoginAttempts(c *gin.Context) {
	p := parsePagination(c)
	db := database.GetDB()

	var attempts []models.LoginAttempt
	var total int64

	db.Model(&models.LoginAttempt{}).Count(&total)
	db.Order("created_at DESC").
		Offset(p.Offset).Limit(p.PageSize).Find(&attempts)

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"attempts":  attempts,
		"total":     total,
		"page":      p.Page,
		"page_size": p.PageSize,
	})
}

func GetSystemLogs(c *gin.Context) {
	p := parsePagination(c)
	db := database.GetDB()

	// 基础查询：系统级别日志
	query := db.Model(&models.AuditLog{}).
		Where("action_type = ? OR action_type LIKE ? OR action_type LIKE ? OR resource_type = ?",
			"system_error", "scheduler_%", "system_%", "system")

	// 筛选条件 (Log Level)
	if logLevel := strings.TrimSpace(c.Query("log_level")); logLevel != "" {
		switch logLevel {
		case "error":
			query = query.Where("response_status >= ? OR action_type = ?", 400, "system_error")
		case "warning":
			query = query.Where("response_status >= ? AND response_status < ?", 300, 400)
		case "info":
			query = query.Where("(response_status < ? OR response_status IS NULL) AND action_type != ?", 300, "system_error")
		}
	}

	// 筛选条件 (Keyword)
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		k := "%" + keyword + "%"
		query = query.Where("action_description LIKE ? OR action_type LIKE ?", k, k)
	}

	// 时间筛选
	query = applyTimeRangeFilter(query, c, "created_at")

	// 1. 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取系统日志总数失败", err)
		return
	}

	// 2. 获取数据 (Preload User)
	var logs []models.AuditLog
	if err := query.Preload("User").Order("created_at DESC").
		Offset(p.Offset).Limit(p.PageSize).Find(&logs).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取系统日志失败", err)
		return
	}

	// 3. 格式化
	logList := make([]gin.H, 0, len(logs))
	for _, log := range logs {
		logList = append(logList, formatAuditLogForAPI(db, log))
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"logs":  logList,
		"total": total,
		"page":  p.Page,
		"size":  p.PageSize,
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

	// 这里的查询可以并发执行优化，但简单起见保持顺序执行
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

// ==========================================
// 日志管理 (重构后避免了 duplicate query construction)
// ==========================================

func GetRegistrationLogs(c *gin.Context) {
	p := parsePagination(c)
	db := database.GetDB()

	query := db.Model(&models.RegistrationLog{})

	// 过滤条件
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		escaped := utils.EscapeLikePattern(utils.SanitizeSearchKeyword(keyword))
		query = query.Where("username LIKE ? OR email LIKE ? OR invite_code LIKE ?", "%"+escaped+"%", "%"+escaped+"%", "%"+escaped+"%")
	}
	// 兼容旧参数 (保留以不破坏功能)
	if username := strings.TrimSpace(c.Query("username")); username != "" {
		escaped := utils.EscapeLikePattern(utils.SanitizeSearchKeyword(username))
		query = query.Where("username LIKE ?", "%"+escaped+"%")
	}
	if email := strings.TrimSpace(c.Query("email")); email != "" {
		escaped := utils.EscapeLikePattern(utils.SanitizeSearchKeyword(email))
		query = query.Where("email LIKE ?", "%"+escaped+"%")
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if inviteCode := strings.TrimSpace(c.Query("invite_code")); inviteCode != "" {
		query = query.Where("invite_code = ?", inviteCode)
	}
	query = applyTimeRangeFilter(query, c, "created_at")

	// 1. 获取总数 (基于当前 query 状态)
	var total int64
	query.Count(&total)

	// 2. 获取数据 (Preload + Pagination)
	var logs []models.RegistrationLog
	err := query.Preload("User").Preload("Inviter").
		Order("created_at DESC").
		Offset(p.Offset).Limit(p.PageSize).Find(&logs).Error

	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "查询注册日志失败", err)
		return
	}

	// 3. 格式化
	var logList []gin.H
	for _, log := range logs {
		locDisplay, _ := getLocationDisplay(log.Location, log.IPAddress)
		logList = append(logList, gin.H{
			"id":          log.ID,
			"user_id":     log.UserID,
			"username":    log.Username,
			"email":       log.Email,
			"ip_address":  getNullableString(log.IPAddress),
			"location":    locDisplay,
			"user_agent":  getNullableString(log.UserAgent),
			"status":      log.Status,
			"reason":      getNullableString(log.FailureReason),
			"invite_code": getNullableString(log.InviteCode),
			"inviter_id":  getNullableInt64(log.InviterID),
			"created_at":  log.CreatedAt,
		})
	}
	genericSuccessResponse(c, logList, total, p)
}

func GetSubscriptionLogs(c *gin.Context) {
	p := parsePagination(c)
	db := database.GetDB()

	// 基础 Query，包含 Join
	query := db.Model(&models.SubscriptionLog{}).
		Joins("JOIN users ON subscription_logs.user_id = users.id").
		Joins("LEFT JOIN subscriptions ON subscription_logs.subscription_id = subscriptions.id")

	// 过滤条件
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		escaped := utils.EscapeLikePattern(utils.SanitizeSearchKeyword(keyword))
		query = query.Where("users.username LIKE ? OR users.email LIKE ? OR subscriptions.subscription_url LIKE ?",
			"%"+escaped+"%", "%"+escaped+"%", "%"+escaped+"%")
	}
	if subID := c.Query("subscription_id"); subID != "" {
		query = query.Where("subscription_logs.subscription_id = ?", subID)
	}
	if uid := c.Query("user_id"); uid != "" {
		query = query.Where("subscription_logs.user_id = ?", uid)
	}
	if at := strings.TrimSpace(c.Query("action_type")); at != "" {
		query = query.Where("subscription_logs.action_type = ?", at)
	}
	if ab := strings.TrimSpace(c.Query("action_by")); ab != "" {
		query = query.Where("subscription_logs.action_by = ?", ab)
	}
	query = applyTimeRangeFilter(query, c, "subscription_logs.created_at")

	// 1. 获取总数
	var total int64
	query.Count(&total)

	// 2. 获取数据 (Preload + Pagination)
	// 注意：GORM 的 Count 会修改 query 对象，但在简单的 Chain 中，通常需要重新指定 Select 或者确保没有 Side Effect。
	// 这里直接继续链式调用是安全的，只要没有 Select 聚合函数。
	var logs []models.SubscriptionLog
	// 需要 Preload 关联数据
	err := query.Preload("Subscription").Preload("User").Preload("ActionByUser").
		Order("subscription_logs.created_at DESC").
		Offset(p.Offset).Limit(p.PageSize).
		// 使用 Select 确保返回的是 log 表的数据，避免 Join 造成的字段歧义
		Select("subscription_logs.*").
		Find(&logs).Error

	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "查询订阅日志失败", err)
		return
	}

	var logList []gin.H
	for _, log := range logs {
		var before, after map[string]interface{}
		if log.BeforeData.Valid {
			json.Unmarshal([]byte(log.BeforeData.String), &before)
		}
		if log.AfterData.Valid {
			json.Unmarshal([]byte(log.AfterData.String), &after)
		}

		logList = append(logList, gin.H{
			"id":              log.ID,
			"subscription_id": log.SubscriptionID,
			"user_id":         log.UserID,
			"username":        getCommonUserName(&log.User),
			"action_type":     log.ActionType,
			"action_by":       getNullableString(log.ActionBy), // 操作类型：user, admin, system
			"action_by_user": func() string {
				// 如果有操作人用户对象，返回用户名；否则返回空字符串
				if log.ActionByUser != nil && log.ActionByUser.ID > 0 {
					return log.ActionByUser.Username
				}
				return "" // 没有操作人用户时返回空字符串，而不是返回 action_by 的值
			}(),
			"before_data": before,
			"after_data":  after,
			"description": getNullableString(log.Description),
			"ip_address":  getNullableString(log.IPAddress),
			"created_at":  formatTime(log.CreatedAt),
		})
	}
	genericSuccessResponse(c, logList, total, p)
}

func GetBalanceLogs(c *gin.Context) {
	p := parsePagination(c)
	db := database.GetDB()

	query := db.Model(&models.BalanceLog{}).Joins("JOIN users ON balance_logs.user_id = users.id")

	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		escaped := utils.EscapeLikePattern(utils.SanitizeSearchKeyword(keyword))
		query = query.Where("users.username LIKE ? OR users.email LIKE ?", "%"+escaped+"%", "%"+escaped+"%")
	}
	if uid := c.Query("user_id"); uid != "" {
		query = query.Where("balance_logs.user_id = ?", uid)
	}
	if ct := strings.TrimSpace(c.Query("change_type")); ct != "" {
		query = query.Where("balance_logs.change_type = ?", ct)
	}
	if oid := c.Query("order_id"); oid != "" {
		query = query.Where("balance_logs.related_order_id = ?", oid)
	}
	query = applyTimeRangeFilter(query, c, "balance_logs.created_at")

	var total int64
	query.Count(&total)

	var logs []models.BalanceLog
	err := query.Preload("User").Preload("RelatedOrder").Preload("OperatorUser").
		Order("balance_logs.created_at DESC").
		Offset(p.Offset).Limit(p.PageSize).
		Select("balance_logs.*").
		Find(&logs).Error

	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "查询余额日志失败", err)
		return
	}

	var logList []gin.H
	for _, log := range logs {
		logList = append(logList, gin.H{
			"id":               log.ID,
			"user_id":          log.UserID,
			"username":         getCommonUserName(&log.User),
			"change_type":      log.ChangeType,
			"amount":           log.Amount,
			"balance_before":   log.BalanceBefore,
			"balance_after":    log.BalanceAfter,
			"related_order_id": getNullableInt64(log.RelatedOrderID),
			"order_no": func() string {
				if log.RelatedOrder != nil {
					return log.RelatedOrder.OrderNo
				}
				return ""
			}(),
			"description": getNullableString(log.Description),
			"operator":    getNullableString(log.Operator),
			"operator_user": func() string {
				if log.OperatorUser != nil && log.OperatorUser.ID > 0 {
					return log.OperatorUser.Username
				}
				return getNullableString(log.Operator)
			}(),
			"ip_address": getNullableString(log.IPAddress),
			"created_at": formatTime(log.CreatedAt),
		})
	}
	genericSuccessResponse(c, logList, total, p)
}

func GetCommissionLogs(c *gin.Context) {
	p := parsePagination(c)
	db := database.GetDB()

	query := db.Model(&models.CommissionLog{}).
		Joins("JOIN users AS inviter_user ON commission_logs.inviter_id = inviter_user.id").
		Joins("JOIN users AS invitee_user ON commission_logs.invitee_id = invitee_user.id")

	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		escaped := utils.EscapeLikePattern(utils.SanitizeSearchKeyword(keyword))
		like := "%" + escaped + "%"
		query = query.Where("inviter_user.username LIKE ? OR inviter_user.email LIKE ? OR invitee_user.username LIKE ? OR invitee_user.email LIKE ?",
			like, like, like, like)
	}
	if invID := c.Query("inviter_id"); invID != "" {
		query = query.Where("commission_logs.inviter_id = ?", invID)
	}
	if invteeID := c.Query("invitee_id"); invteeID != "" {
		query = query.Where("commission_logs.invitee_id = ?", invteeID)
	}
	if ct := strings.TrimSpace(c.Query("commission_type")); ct != "" {
		query = query.Where("commission_logs.commission_type = ?", ct)
	}
	if st := strings.TrimSpace(c.Query("status")); st != "" {
		query = query.Where("commission_logs.status = ?", st)
	}
	query = applyTimeRangeFilter(query, c, "commission_logs.created_at")

	var total int64
	query.Count(&total)

	var logs []models.CommissionLog
	err := query.Preload("Inviter").Preload("Invitee").Preload("RelatedOrder").
		Order("commission_logs.created_at DESC").
		Offset(p.Offset).Limit(p.PageSize).
		Select("commission_logs.*").
		Find(&logs).Error

	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "查询佣金日志失败", err)
		return
	}

	var logList []gin.H
	for _, log := range logs {
		settledAt := ""
		if log.SettledAt.Valid {
			settledAt = formatTime(log.SettledAt.Time)
		}

		logList = append(logList, gin.H{
			"id":               log.ID,
			"inviter_id":       log.InviterID,
			"inviter_name":     getCommonUserName(&log.Inviter),
			"invitee_id":       log.InviteeID,
			"invitee_name":     getCommonUserName(&log.Invitee),
			"commission_type":  log.CommissionType,
			"amount":           log.Amount,
			"related_order_id": getNullableInt64(log.RelatedOrderID),
			"order_no": func() string {
				if log.RelatedOrder != nil {
					return log.RelatedOrder.OrderNo
				}
				return ""
			}(),
			"status":      log.Status,
			"settled_at":  settledAt,
			"description": getNullableString(log.Description),
			"created_at":  formatTime(log.CreatedAt),
		})
	}
	genericSuccessResponse(c, logList, total, p)
}

func GetSubscriptionResetLogs(c *gin.Context) {
	p := parsePagination(c)
	db := database.GetDB()

	query := db.Model(&models.SubscriptionReset{}).
		Joins("JOIN users ON subscription_resets.user_id = users.id").
		Joins("LEFT JOIN subscriptions ON subscription_resets.subscription_id = subscriptions.id")

	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		escaped := utils.EscapeLikePattern(utils.SanitizeSearchKeyword(keyword))
		like := "%" + escaped + "%"
		query = query.Where("users.username LIKE ? OR users.email LIKE ? OR subscriptions.subscription_url LIKE ? OR subscription_resets.old_subscription_url LIKE ? OR subscription_resets.new_subscription_url LIKE ?",
			like, like, like, like, like)
	}
	if uid := c.Query("user_id"); uid != "" {
		query = query.Where("subscription_resets.user_id = ?", uid)
	}
	if sid := c.Query("subscription_id"); sid != "" {
		query = query.Where("subscription_resets.subscription_id = ?", sid)
	}
	if rt := strings.TrimSpace(c.Query("reset_type")); rt != "" {
		query = query.Where("subscription_resets.reset_type = ?", rt)
	}
	if rb := strings.TrimSpace(c.Query("reset_by")); rb != "" {
		query = query.Where("subscription_resets.reset_by = ?", rb)
	}
	query = applyTimeRangeFilter(query, c, "subscription_resets.created_at")

	var total int64
	query.Count(&total)

	var logs []models.SubscriptionReset
	err := query.Preload("User").Preload("Subscription").
		Order("subscription_resets.created_at DESC").
		Offset(p.Offset).Limit(p.PageSize).
		Select("subscription_resets.*").
		Find(&logs).Error

	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "查询订阅重置日志失败", err)
		return
	}

	var logList []gin.H
	for _, log := range logs {
		logList = append(logList, gin.H{
			"id":                   log.ID,
			"user_id":              log.UserID,
			"username":             getCommonUserName(&log.User),
			"subscription_id":      log.SubscriptionID,
			"reset_type":           log.ResetType,
			"reason":               log.Reason,
			"old_subscription_url": getNullableStringPtr(log.OldSubscriptionURL),
			"new_subscription_url": getNullableStringPtr(log.NewSubscriptionURL),
			"device_count_before":  log.DeviceCountBefore,
			"device_count_after":   log.DeviceCountAfter,
			"reset_by":             getNullableStringPtr(log.ResetBy),
			"created_at":           formatTime(log.CreatedAt),
		})
	}
	genericSuccessResponse(c, logList, total, p)
}

func GetEmailLogs(c *gin.Context) {
	p := parsePagination(c)
	db := database.GetDB()

	query := db.Model(&models.EmailQueue{})

	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		escaped := utils.EscapeLikePattern(utils.SanitizeSearchKeyword(keyword))
		query = query.Where("to_email LIKE ?", "%"+escaped+"%")
	}
	// 兼容旧参数
	if email := strings.TrimSpace(c.Query("recipient_email")); email != "" {
		query = query.Where("to_email LIKE ?", "%"+utils.EscapeLikePattern(utils.SanitizeSearchKeyword(email))+"%")
	}
	if email := strings.TrimSpace(c.Query("to_email")); email != "" {
		query = query.Where("to_email LIKE ?", "%"+utils.EscapeLikePattern(utils.SanitizeSearchKeyword(email))+"%")
	}
	if et := strings.TrimSpace(c.Query("email_type")); et != "" {
		query = query.Where("email_type = ?", et)
	}
	if st := strings.TrimSpace(c.Query("status")); st != "" {
		query = query.Where("status = ?", st)
	}
	query = applyTimeRangeFilter(query, c, "created_at")

	var total int64
	query.Count(&total)

	var logs []models.EmailQueue
	err := query.Order("created_at DESC").
		Offset(p.Offset).Limit(p.PageSize).Find(&logs).Error

	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "查询邮件日志失败", err)
		return
	}

	var logList []gin.H
	for _, log := range logs {
		sentAt := ""
		if log.SentAt.Valid {
			sentAt = formatTime(log.SentAt.Time)
		}
		logList = append(logList, gin.H{
			"id":              log.ID,
			"to_email":        log.ToEmail,
			"recipient_email": log.ToEmail,
			"subject":         log.Subject,
			"email_type":      log.EmailType,
			"status":          log.Status,
			"retry_count":     log.RetryCount,
			"max_retries":     log.MaxRetries,
			"sent_at":         sentAt,
			"error_message":   getNullableString(log.ErrorMessage),
			"created_at":      formatTime(log.CreatedAt),
		})
	}
	genericSuccessResponse(c, logList, total, p)
}
