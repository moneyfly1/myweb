package utils

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"

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
