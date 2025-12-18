package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/services/email"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetNotifications 获取通知列表
func GetNotifications(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	db := database.GetDB()
	var notifications []models.Notification
	if err := db.Where("user_id = ? OR user_id IS NULL", user.ID).Order("created_at DESC").Find(&notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取通知列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    notifications,
	})
}

// GetUnreadCount 获取未读通知数量
func GetUnreadCount(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	db := database.GetDB()
	var count int64
	if err := db.Model(&models.Notification{}).Where("user_id = ? AND is_read = ?", user.ID, false).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取未读数量失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"count": count,
		},
	})
}

// MarkAsRead 标记通知为已读
func MarkAsRead(c *gin.Context) {
	id := c.Param("id")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	db := database.GetDB()
	var notification models.Notification
	if err := db.Where("id = ? AND user_id = ?", id, user.ID).First(&notification).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "通知不存在",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "更新通知失败",
			})
		}
		return
	}

	notification.IsRead = true
	if err := db.Save(&notification).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新通知失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "已标记为已读",
	})
}

// MarkAllAsRead 标记所有通知为已读
func MarkAllAsRead(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	db := database.GetDB()
	if err := db.Model(&models.Notification{}).Where("user_id = ?", user.ID).Update("is_read", true).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新通知失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "已全部标记为已读",
	})
}

// DeleteNotification 删除通知
func DeleteNotification(c *gin.Context) {
	id := c.Param("id")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	db := database.GetDB()
	if err := db.Where("id = ? AND user_id = ?", id, user.ID).Delete(&models.Notification{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除通知失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "删除成功",
	})
}

// GetAdminNotifications 管理员获取通知列表
func GetAdminNotifications(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.Notification{})

	// 分页参数
	page := 1
	size := 20
	if pageStr := c.Query("page"); pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
	}
	if sizeStr := c.Query("size"); sizeStr != "" {
		fmt.Sscanf(sizeStr, "%d", &size)
	}

	var total int64
	query.Count(&total)

	var notifications []models.Notification
	offset := (page - 1) * size
	if err := query.Preload("User").Offset(offset).Limit(size).Order("created_at DESC").Find(&notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取通知列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"notifications": notifications,
			"total":         total,
			"page":          page,
			"size":          size,
		},
	})
}

// GetUserNotifications 获取用户通知（限制数量）
func GetUserNotifications(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	db := database.GetDB()
	var notifications []models.Notification
	if err := db.Where("user_id = ? OR user_id IS NULL", user.ID).Order("created_at DESC").Limit(limit).Find(&notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取通知列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    notifications,
	})
}

// CreateAdminNotification 管理员创建通知
func CreateAdminNotification(c *gin.Context) {
	var req struct {
		UserID    *uint  `json:"user_id"` // 可选，如果为空则发送给所有用户
		Title     string `json:"title" binding:"required"`
		Content   string `json:"content" binding:"required"`
		Type      string `json:"type"`
		Status    string `json:"status"`
		IsActive  *bool  `json:"is_active"`
		SendEmail *bool  `json:"send_email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// 不向客户端返回详细错误信息，防止信息泄露
		utils.LogError("CreateNotification: bind request", err, nil)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误，请检查输入格式",
		})
		return
	}

	db := database.GetDB()
	notification := models.Notification{
		Title:   req.Title,
		Content: req.Content,
		Type:    req.Type,
		IsRead:  false,
	}

	if req.Type == "" {
		notification.Type = "announcement"
	}

	// 处理状态和is_active
	if req.Status == "published" || (req.IsActive != nil && *req.IsActive) {
		notification.IsActive = true
	} else if req.IsActive != nil {
		notification.IsActive = *req.IsActive
	} else {
		notification.IsActive = true // 默认发布
	}

	// 如果指定了用户ID，则发送给特定用户
	if req.UserID != nil {
		// 验证用户是否存在
		var user models.User
		if err := db.First(&user, *req.UserID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "用户不存在",
			})
			return
		}
		notification.UserID = sql.NullInt64{Int64: int64(*req.UserID), Valid: true}
	} else {
		// 发送给所有用户（user_id 为 NULL）
		notification.UserID = sql.NullInt64{Valid: false}
	}

	if err := db.Create(&notification).Error; err != nil {
		// 不向客户端返回详细错误信息，防止信息泄露
		utils.LogError("CreateNotification: create notification", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建通知失败，请稍后重试",
		})
		return
	}

	// 如果设置了发送邮件，则发送营销邮件
	if req.SendEmail != nil && *req.SendEmail {
		go func() {
			emailService := email.NewEmailService()
			templateBuilder := email.NewEmailTemplateBuilder()

			if req.UserID != nil {
				// 发送给特定用户
				var user models.User
				if err := db.First(&user, *req.UserID).Error; err == nil {
					content := templateBuilder.GetMarketingEmailTemplate(req.Title, req.Content)
					_ = emailService.QueueEmail(user.Email, req.Title, content, "marketing")
				}
			} else {
				// 发送给所有用户
				var users []models.User
				if err := db.Where("is_active = ?", true).Find(&users).Error; err == nil {
					content := templateBuilder.GetMarketingEmailTemplate(req.Title, req.Content)
					for _, user := range users {
						_ = emailService.QueueEmail(user.Email, req.Title, content, "marketing")
					}
				}
			}
		}()
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "通知创建成功",
		"data":    notification,
	})
}

// UpdateAdminNotification 管理员更新通知
func UpdateAdminNotification(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()

	var notification models.Notification
	if err := db.First(&notification, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "通知不存在",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "获取通知失败",
			})
		}
		return
	}

	var req struct {
		Title     string `json:"title"`
		Content   string `json:"content"` // 内容可能包含HTML，由前端DOMPurify处理
		Type      string `json:"type"`
		Status    string `json:"status"`
		IsActive  *bool  `json:"is_active"`
		SendEmail *bool  `json:"send_email"`
		UserID    *uint  `json:"user_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// 不向客户端返回详细错误信息，防止信息泄露
		utils.LogError("UpdateNotification: bind request", err, nil)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误，请检查输入格式",
		})
		return
	}

	// 更新字段
	if req.Title != "" {
		notification.Title = req.Title
	}
	if req.Content != "" {
		notification.Content = req.Content
	}
	if req.Type != "" {
		notification.Type = req.Type
	}
	if req.Status != "" {
		if req.Status == "published" {
			notification.IsActive = true
		} else if req.Status == "draft" {
			notification.IsActive = false
		}
	}
	if req.IsActive != nil {
		notification.IsActive = *req.IsActive
	}
	if req.UserID != nil {
		// 验证用户是否存在
		var user models.User
		if err := db.First(&user, *req.UserID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "用户不存在",
			})
			return
		}
		notification.UserID = sql.NullInt64{Int64: int64(*req.UserID), Valid: true}
	}

	if err := db.Save(&notification).Error; err != nil {
		utils.LogError("UpdateNotification: save notification failed", err, map[string]interface{}{
			"notification_id": notification.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新通知失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "通知更新成功",
		"data":    notification,
	})
}

// DeleteAdminNotification 管理员删除通知
func DeleteAdminNotification(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()

	if err := db.Delete(&models.Notification{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除通知失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "删除成功",
	})
}
