package handlers

import (
	"net/http"
	"strconv"

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
			"logs": logs,
			"total": total,
			"page": page,
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
			"attempts": attempts,
			"total": total,
			"page": page,
			"page_size": pageSize,
		},
	})
}

