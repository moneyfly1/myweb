package handlers

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	"cboard-go/internal/core/auth"
	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/services/email"
	"cboard-go/internal/services/notification"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	OldPassword     string `json:"old_password"` // 兼容字段
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// ChangePassword 修改密码
func ChangePassword(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()

	// 支持 current_password 和 old_password 两种字段名
	oldPassword := req.CurrentPassword
	if oldPassword == "" {
		oldPassword = req.OldPassword
	}

	// 验证原密码
	if !auth.VerifyPassword(oldPassword, user.Password) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "原密码错误",
		})
		return
	}

	// 验证新密码强度
	valid, msg := auth.ValidatePasswordStrength(req.NewPassword, 8)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": msg,
		})
		return
	}

	// 更新密码
	hashedPassword, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "密码加密失败",
		})
		return
	}

	user.Password = hashedPassword
	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新密码失败",
		})
		return
	}

	// 记录审计日志
	utils.CreateAuditLogSimple(c, "change_password", "user", user.ID,
		fmt.Sprintf("用户修改密码: %s", user.Email))

	// 发送密码修改成功邮件
	go func() {
		emailService := email.NewEmailService()
		templateBuilder := email.NewEmailTemplateBuilder()
		baseURL := func() string {
			scheme := "http"
			if proto := c.Request.Header.Get("X-Forwarded-Proto"); proto != "" {
				scheme = proto
			} else if c.Request.TLS != nil {
				scheme = "https"
			}
			return fmt.Sprintf("%s://%s", scheme, c.Request.Host)
		}()
		loginURL := fmt.Sprintf("%s/login", baseURL)
		changeTime := utils.GetBeijingTime().Format("2006-01-02 15:04:05")
		content := templateBuilder.GetPasswordChangedTemplate(user.Username, changeTime, loginURL)
		subject := "密码修改成功"
		_ = emailService.QueueEmail(user.Email, subject, content, "password_changed")
	}()

	utils.SetResponseStatus(c, http.StatusOK)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "密码修改成功",
	})
}

// ResetPasswordRequest 重置密码请求（管理员）
type ResetPasswordRequest struct {
	Password string `json:"password" binding:"required,min=8"`
}

// ResetPassword 重置用户密码（管理员）
func ResetPassword(c *gin.Context) {
	userID := c.Param("id")

	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	// 验证密码强度（最小长度8位，要求包含大小写字母、数字和特殊字符中的至少三种）
	valid, msg := auth.ValidatePasswordStrength(req.Password, 8)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": msg,
		})
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "用户不存在",
		})
		return
	}

	// 更新密码
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "密码加密失败",
		})
		return
	}

	user.Password = hashedPassword
	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "重置密码失败",
		})
		return
	}

	// 记录审计日志
	utils.CreateAuditLogSimple(c, "reset_password", "user", user.ID,
		fmt.Sprintf("管理员重置用户密码: %s (%s)", user.Username, user.Email))

	// 发送管理员通知
	go func() {
		notificationService := notification.NewNotificationService()
		resetTime := utils.GetBeijingTime().Format("2006-01-02 15:04:05")
		_ = notificationService.SendAdminNotification("password_reset", map[string]interface{}{
			"username":   user.Username,
			"email":      user.Email,
			"reset_time": resetTime,
		})
	}()

	utils.SetResponseStatus(c, http.StatusOK)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "密码重置成功",
	})
}

// ForgotPasswordRequest 忘记密码请求
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ForgotPassword 忘记密码（发送验证码）
func ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		// 为了安全，即使用户不存在也返回成功
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "如果该邮箱存在，验证码已发送",
		})
		return
	}

	// 生成6位验证码（使用 crypto/rand 生成更安全的随机数）
	b := make([]byte, 4)
	rand.Read(b)
	codeInt := int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
	codeInt = 100000 + (codeInt % 900000)
	code := fmt.Sprintf("%06d", codeInt)

	expiresAt := utils.GetBeijingTime().Add(10 * time.Minute)

	// 保存验证码（purpose 为 reset_password）
	verificationCode := models.VerificationCode{
		Email:     req.Email,
		Code:      code,
		ExpiresAt: expiresAt,
		Used:      0,
		Purpose:   "reset_password",
	}

	if err := db.Create(&verificationCode).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "保存验证码失败",
		})
		return
	}

	// 发送密码重置验证码邮件（验证码邮件是必须的，不受客户通知开关影响）
	emailService := email.NewEmailService()
	templateBuilder := email.NewEmailTemplateBuilder()
	content := templateBuilder.GetPasswordResetVerificationCodeTemplate(user.Username, code)
	subject := "密码重置验证码"

	// 验证码邮件立即发送，不加入队列（验证码需要实时性）
	if err := emailService.SendEmail(user.Email, subject, content); err != nil {
		// 如果立即发送失败，尝试加入队列作为备选方案
		if queueErr := emailService.QueueEmail(user.Email, subject, content, "verification"); queueErr != nil {
			utils.LogError("RequestPasswordReset: send email failed", err, map[string]interface{}{
				"user_id": user.ID,
			})
			utils.LogError("RequestPasswordReset: queue email also failed", queueErr, map[string]interface{}{
				"user_id": user.ID,
			})
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "发送验证码邮件失败",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "验证码已发送，请查收邮箱",
	})
}

// ResetPasswordByCodeRequest 通过验证码重置密码请求
type ResetPasswordByCodeRequest struct {
	Email            string `json:"email" binding:"required,email"`
	VerificationCode string `json:"verification_code" binding:"required"`
	NewPassword      string `json:"new_password" binding:"required,min=8"`
}

// ResetPasswordByCode 通过验证码重置密码
func ResetPasswordByCode(c *gin.Context) {
	var req ResetPasswordByCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	// 验证密码强度
	valid, msg := auth.ValidatePasswordStrength(req.NewPassword, 8)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": msg,
		})
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "用户不存在",
		})
		return
	}

	// 验证验证码
	var verificationCode models.VerificationCode
	if err := db.Where("email = ? AND code = ? AND used = ? AND purpose = ?", req.Email, req.VerificationCode, 0, "reset_password").Order("created_at DESC").First(&verificationCode).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "验证码错误或已使用",
		})
		return
	}

	// 检查验证码是否过期
	if verificationCode.IsExpired() {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "验证码已过期，请重新获取",
		})
		return
	}

	// 标记验证码为已使用（防止重复使用）
	verificationCode.Used = 1
	if err := db.Model(&verificationCode).Where("id = ?", verificationCode.ID).Update("used", 1).Error; err != nil {
		utils.LogError("ResetPasswordByCode: mark verification code as used failed", err, map[string]interface{}{
			"code_id": verificationCode.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "标记验证码失败",
		})
		return
	}

	// 更新密码
	hashedPassword, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "密码加密失败",
		})
		return
	}

	user.Password = hashedPassword
	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "重置密码失败",
		})
		return
	}

	// 设置用户ID到上下文，以便审计日志可以获取
	c.Set("user_id", user.ID)
	utils.SetResponseStatus(c, http.StatusOK)

	// 记录审计日志（用户自己重置密码）
	utils.CreateAuditLogSimple(c, "reset_password", "user", user.ID,
		fmt.Sprintf("用户通过验证码重置密码: %s (%s)", user.Username, user.Email))

	// 发送管理员通知
	go func() {
		notificationService := notification.NewNotificationService()
		resetTime := utils.GetBeijingTime().Format("2006-01-02 15:04:05")
		_ = notificationService.SendAdminNotification("password_reset", map[string]interface{}{
			"username":   user.Username,
			"email":      user.Email,
			"reset_time": resetTime,
		})
	}()

	utils.SetResponseStatus(c, http.StatusOK)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "密码重置成功",
	})
}
