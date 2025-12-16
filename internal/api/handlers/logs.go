package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"

	"github.com/gin-gonic/gin"
)

// GetAuditLogs 获取审计日志
func GetAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	db := database.GetDB()

	var logs []models.AuditLog
	var total int64

	offset := (page - 1) * pageSize

	db.Model(&models.AuditLog{}).Count(&total)
	db.Preload("User").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"logs":      logs,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetLoginAttempts 获取登录尝试记录
func GetLoginAttempts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	db := database.GetDB()

	var attempts []models.LoginAttempt
	var total int64

	offset := (page - 1) * pageSize

	db.Model(&models.LoginAttempt{}).Count(&total)
	db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&attempts)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"attempts":  attempts,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetSystemLogs 获取系统日志（兼容前端API）
func GetSystemLogs(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.AuditLog{}).Preload("User")

	// 分页参数
	page := 1
	size := 20
	if pageStr := c.Query("page"); pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
	}
	if sizeStr := c.Query("size"); sizeStr != "" {
		fmt.Sscanf(sizeStr, "%d", &size)
	}

	// 筛选参数
	if logLevel := c.Query("log_level"); logLevel != "" {
		logLevel = strings.TrimSpace(logLevel)
		// 将日志级别映射到审计日志的ActionType或ResponseStatus
		if logLevel == "error" {
			query = query.Where("response_status >= ?", 400)
		} else if logLevel == "warning" {
			query = query.Where("response_status >= ? AND response_status < ?", 300, 400)
		} else if logLevel == "info" {
			query = query.Where("response_status < ? OR response_status IS NULL", 300)
		}
	}

	if module := c.Query("module"); module != "" {
		module = strings.TrimSpace(module)
		if module != "" {
			// 根据模块筛选ResourceType
			query = query.Where("resource_type LIKE ?", "%"+module+"%")
		}
	}

	if username := c.Query("username"); username != "" {
		username = strings.TrimSpace(username)
		if username != "" {
			query = query.Joins("JOIN users ON audit_logs.user_id = users.id").
				Where("users.username LIKE ?", "%"+username+"%")
		}
	}

	if keyword := c.Query("keyword"); keyword != "" {
		keyword = strings.TrimSpace(keyword)
		if keyword != "" {
			query = query.Where("action_description LIKE ? OR action_type LIKE ? OR resource_type LIKE ?",
				"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
		}
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

	var total int64
	// 先执行Count查询（在应用Preload之前，避免JOIN影响计数）
	countQuery := db.Model(&models.AuditLog{})

	// 应用相同的筛选条件到Count查询
	if logLevel := c.Query("log_level"); logLevel != "" {
		logLevel = strings.TrimSpace(logLevel)
		if logLevel == "error" {
			countQuery = countQuery.Where("response_status >= ?", 400)
		} else if logLevel == "warning" {
			countQuery = countQuery.Where("response_status >= ? AND response_status < ?", 300, 400)
		} else if logLevel == "info" {
			countQuery = countQuery.Where("response_status < ? OR response_status IS NULL", 300)
		}
	}

	if module := c.Query("module"); module != "" {
		module = strings.TrimSpace(module)
		if module != "" {
			countQuery = countQuery.Where("resource_type LIKE ?", "%"+module+"%")
		}
	}

	if username := c.Query("username"); username != "" {
		username = strings.TrimSpace(username)
		if username != "" {
			countQuery = countQuery.Joins("JOIN users ON audit_logs.user_id = users.id").
				Where("users.username LIKE ?", "%"+username+"%")
		}
	}

	if keyword := c.Query("keyword"); keyword != "" {
		keyword = strings.TrimSpace(keyword)
		if keyword != "" {
			countQuery = countQuery.Where("action_description LIKE ? OR action_type LIKE ? OR resource_type LIKE ?",
				"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
		}
	}

	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", startTime); err == nil {
			countQuery = countQuery.Where("created_at >= ?", t)
		}
	}

	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", endTime); err == nil {
			countQuery = countQuery.Where("created_at <= ?", t)
		}
	}

	if err := countQuery.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取系统日志总数失败",
		})
		return
	}

	var logs []models.AuditLog
	offset := (page - 1) * size

	// 执行Find查询（带Preload）
	if err := query.Order("created_at DESC").Offset(offset).Limit(size).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取系统日志失败",
		})
		return
	}

	// 转换为前端期望的格式
	logList := make([]gin.H, 0)
	for _, log := range logs {
		// 确定日志级别
		level := "info"
		if log.ResponseStatus.Valid {
			if log.ResponseStatus.Int64 >= 500 {
				level = "error"
			} else if log.ResponseStatus.Int64 >= 400 {
				level = "error"
			} else if log.ResponseStatus.Int64 >= 300 {
				level = "warning"
			}
		}

		// 获取用户名
		username := ""
		if log.UserID.Valid {
			// 如果Preload失败，尝试单独查询
			if log.User.ID == 0 {
				var user models.User
				if db.First(&user, log.UserID.Int64).Error == nil {
					username = user.Username
				}
			} else {
				username = log.User.Username
			}
		}

		// 获取IP地址
		ipAddress := ""
		if log.IPAddress.Valid {
			ipAddress = log.IPAddress.String
		}

		// 获取User-Agent
		userAgent := ""
		if log.UserAgent.Valid {
			userAgent = log.UserAgent.String
		}

		// 获取模块
		module := ""
		if log.ResourceType.Valid {
			module = log.ResourceType.String
		}

		// 获取消息
		message := ""
		if log.ActionDescription.Valid {
			message = log.ActionDescription.String
		} else {
			message = log.ActionType
		}

		logList = append(logList, gin.H{
			"id":          log.ID,
			"timestamp":   log.CreatedAt.Format("2006-01-02 15:04:05"),
			"level":       level,
			"module":      module,
			"message":     message,
			"username":    username,
			"ip_address":  ipAddress,
			"user_agent":  userAgent,
			"action_type": log.ActionType,
			"details": func() string {
				if log.BeforeData.Valid || log.AfterData.Valid {
					var details []string
					if log.BeforeData.Valid {
						details = append(details, "Before: "+log.BeforeData.String)
					}
					if log.AfterData.Valid {
						details = append(details, "After: "+log.AfterData.String)
					}
					return strings.Join(details, "\n")
				}
				return ""
			}(),
			"context": func() gin.H {
				ctx := gin.H{}
				if log.RequestMethod.Valid {
					ctx["method"] = log.RequestMethod.String
				}
				if log.RequestPath.Valid {
					ctx["path"] = log.RequestPath.String
				}
				if log.RequestParams.Valid {
					ctx["params"] = log.RequestParams.String
				}
				if log.ResponseStatus.Valid {
					ctx["status"] = log.ResponseStatus.Int64
				}
				return ctx
			}(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"logs":  logList,
			"total": total,
			"page":  page,
			"size":  size,
		},
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// ExportLogs 导出日志
func ExportLogs(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.AuditLog{}).Preload("User")

	// 应用筛选（与GetSystemLogs相同的筛选逻辑）
	if logLevel := c.Query("log_level"); logLevel != "" {
		logLevel = strings.TrimSpace(logLevel)
		if logLevel == "error" {
			query = query.Where("response_status >= ?", 400)
		} else if logLevel == "warning" {
			query = query.Where("response_status >= ? AND response_status < ?", 300, 400)
		} else if logLevel == "info" {
			query = query.Where("response_status < ? OR response_status IS NULL", 300)
		}
	}

	if module := c.Query("module"); module != "" {
		module = strings.TrimSpace(module)
		if module != "" {
			query = query.Where("resource_type LIKE ?", "%"+module+"%")
		}
	}

	if username := c.Query("username"); username != "" {
		username = strings.TrimSpace(username)
		if username != "" {
			query = query.Joins("JOIN users ON audit_logs.user_id = users.id").
				Where("users.username LIKE ?", "%"+username+"%")
		}
	}

	if keyword := c.Query("keyword"); keyword != "" {
		keyword = strings.TrimSpace(keyword)
		if keyword != "" {
			query = query.Where("action_description LIKE ? OR action_type LIKE ? OR resource_type LIKE ?",
				"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
		}
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

	var logs []models.AuditLog
	if err := query.Order("created_at DESC").Limit(10000).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "导出日志失败",
		})
		return
	}

	// 生成CSV内容
	var csvContent strings.Builder
	csvContent.WriteString("\xEF\xBB\xBF") // UTF-8 BOM
	csvContent.WriteString("时间,级别,模块,用户,IP地址,操作类型,日志内容\n")

	for _, log := range logs {
		level := "信息"
		if log.ResponseStatus.Valid {
			if log.ResponseStatus.Int64 >= 400 {
				level = "错误"
			} else if log.ResponseStatus.Int64 >= 300 {
				level = "警告"
			}
		}

		username := ""
		if log.UserID.Valid && log.User.ID > 0 {
			username = log.User.Username
		}

		ipAddress := ""
		if log.IPAddress.Valid {
			ipAddress = log.IPAddress.String
		}

		module := ""
		if log.ResourceType.Valid {
			module = log.ResourceType.String
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

		csvContent.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,\"%s\"\n",
			log.CreatedAt.Format("2006-01-02 15:04:05"),
			level,
			module,
			username,
			ipAddress,
			log.ActionType,
			message,
		))
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "清空日志失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("已清空 %d 条日志", result.RowsAffected),
		"data": gin.H{
			"deleted_count": result.RowsAffected,
		},
	})
}
