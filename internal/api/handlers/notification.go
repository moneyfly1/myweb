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

func requireAuth(c *gin.Context) (*models.User, bool) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return nil, false
	}
	return user, true
}

func errorResponse(c *gin.Context, statusCode int, message string) {
	utils.ErrorResponse(c, statusCode, message, nil)
}

func successResponse(c *gin.Context, statusCode int, message string, data interface{}) {
	utils.SuccessResponse(c, statusCode, message, data)
}

func parsePaginationParams(c *gin.Context) (page, size int) {
	page = 1
	size = 20
	if pageStr := c.Query("page"); pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
	}
	if sizeStr := c.Query("size"); sizeStr != "" {
		fmt.Sscanf(sizeStr, "%d", &size)
	}
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	return page, size
}

func validateUserExists(db *gorm.DB, userID uint) (*models.User, error) {
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func handleGormError(c *gin.Context, err error, notFoundMsg, serverErrorMsg string) bool {
	if err == gorm.ErrRecordNotFound {
		errorResponse(c, http.StatusNotFound, notFoundMsg)
		return true
	}
	utils.LogError(serverErrorMsg, err, nil)
	errorResponse(c, http.StatusInternalServerError, serverErrorMsg)
	return true
}

func sendNotificationEmail(db *gorm.DB, userID *uint, title, content string) {
	go func() {
		emailService := email.NewEmailService()
		templateBuilder := email.NewEmailTemplateBuilder()
		emailContent := templateBuilder.GetMarketingEmailTemplate(title, content)

		if userID != nil {
			var user models.User
			if err := db.First(&user, *userID).Error; err == nil {
				_ = emailService.QueueEmail(user.Email, title, emailContent, "marketing")
			}
		} else {
			var users []models.User
			if err := db.Where("is_active = ?", true).Find(&users).Error; err == nil {
				for _, user := range users {
					_ = emailService.QueueEmail(user.Email, title, emailContent, "marketing")
				}
			}
		}
	}()
}

func GetNotifications(c *gin.Context) {
	user, ok := requireAuth(c)
	if !ok {
		return
	}

	db := database.GetDB()
	var notifications []models.Notification
	if err := db.Where("user_id = ? OR user_id IS NULL", user.ID).
		Order("created_at DESC").
		Find(&notifications).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "获取通知列表失败")
		return
	}

	successResponse(c, http.StatusOK, "", notifications)
}

func GetUnreadCount(c *gin.Context) {
	user, ok := requireAuth(c)
	if !ok {
		return
	}

	db := database.GetDB()
	var count int64
	if err := db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", user.ID, false).
		Count(&count).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "获取未读数量失败")
		return
	}

	successResponse(c, http.StatusOK, "", gin.H{"count": count})
}

func MarkAsRead(c *gin.Context) {
	id := c.Param("id")
	user, ok := requireAuth(c)
	if !ok {
		return
	}

	db := database.GetDB()
	var notification models.Notification
	if err := db.Where("id = ? AND user_id = ?", id, user.ID).
		First(&notification).Error; err != nil {
		handleGormError(c, err, "通知不存在", "更新通知失败")
		return
	}

	notification.IsRead = true
	if err := db.Save(&notification).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "更新通知失败")
		return
	}

	successResponse(c, http.StatusOK, "已标记为已读", nil)
}

func MarkAllAsRead(c *gin.Context) {
	user, ok := requireAuth(c)
	if !ok {
		return
	}

	db := database.GetDB()
	if err := db.Model(&models.Notification{}).
		Where("user_id = ?", user.ID).
		Update("is_read", true).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "更新通知失败")
		return
	}

	successResponse(c, http.StatusOK, "已全部标记为已读", nil)
}

func DeleteNotification(c *gin.Context) {
	id := c.Param("id")
	user, ok := requireAuth(c)
	if !ok {
		return
	}

	db := database.GetDB()
	if err := db.Where("id = ? AND user_id = ?", id, user.ID).
		Delete(&models.Notification{}).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "删除通知失败")
		return
	}

	successResponse(c, http.StatusOK, "删除成功", nil)
}

func GetAdminNotifications(c *gin.Context) {
	db := database.GetDB()
	page, size := parsePaginationParams(c)

	query := db.Model(&models.Notification{})
	var total int64
	query.Count(&total)

	var notifications []models.Notification
	offset := (page - 1) * size
	if err := query.Preload("User").
		Offset(offset).
		Limit(size).
		Order("created_at DESC").
		Find(&notifications).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "获取通知列表失败")
		return
	}

	successResponse(c, http.StatusOK, "", gin.H{
		"notifications": notifications,
		"total":         total,
		"page":          page,
		"size":          size,
	})
}

func GetUserNotifications(c *gin.Context) {
	user, ok := requireAuth(c)
	if !ok {
		return
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}
	if limit < 1 {
		limit = 10
	}

	db := database.GetDB()
	var notifications []models.Notification
	if err := db.Where("user_id = ? OR user_id IS NULL", user.ID).
		Order("created_at DESC").
		Limit(limit).
		Find(&notifications).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "获取通知列表失败")
		return
	}

	successResponse(c, http.StatusOK, "", notifications)
}

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
		utils.LogError("CreateNotification: bind request", err, nil)
		errorResponse(c, http.StatusBadRequest, "请求参数错误，请检查输入格式")
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

	if req.Status == "published" || (req.IsActive != nil && *req.IsActive) {
		notification.IsActive = true
	} else if req.IsActive != nil {
		notification.IsActive = *req.IsActive
	} else {
		notification.IsActive = true // 默认发布
	}

	if req.UserID != nil {
		if _, err := validateUserExists(db, *req.UserID); err != nil {
			errorResponse(c, http.StatusNotFound, "用户不存在")
			return
		}
		notification.UserID = sql.NullInt64{Int64: int64(*req.UserID), Valid: true}
	} else {
		notification.UserID = sql.NullInt64{Valid: false}
	}

	if err := db.Create(&notification).Error; err != nil {
		utils.LogError("CreateNotification: create notification", err, nil)
		errorResponse(c, http.StatusInternalServerError, "创建通知失败，请稍后重试")
		return
	}

	if req.SendEmail != nil && *req.SendEmail {
		sendNotificationEmail(db, req.UserID, req.Title, req.Content)
	}

	successResponse(c, http.StatusCreated, "通知创建成功", notification)
}

func UpdateAdminNotification(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()

	var notification models.Notification
	if err := db.First(&notification, id).Error; err != nil {
		handleGormError(c, err, "通知不存在", "获取通知失败")
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
		utils.LogError("UpdateNotification: bind request", err, nil)
		errorResponse(c, http.StatusBadRequest, "请求参数错误，请检查输入格式")
		return
	}

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
		if _, err := validateUserExists(db, *req.UserID); err != nil {
			errorResponse(c, http.StatusNotFound, "用户不存在")
			return
		}
		notification.UserID = sql.NullInt64{Int64: int64(*req.UserID), Valid: true}
	}

	if err := db.Save(&notification).Error; err != nil {
		utils.LogError("UpdateNotification: save notification failed", err, map[string]interface{}{
			"notification_id": notification.ID,
		})
		errorResponse(c, http.StatusInternalServerError, "更新通知失败")
		return
	}

	successResponse(c, http.StatusOK, "通知更新成功", notification)
}

func DeleteAdminNotification(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()

	if err := db.Delete(&models.Notification{}, id).Error; err != nil {
		errorResponse(c, http.StatusInternalServerError, "删除通知失败")
		return
	}

	successResponse(c, http.StatusOK, "删除成功", nil)
}
