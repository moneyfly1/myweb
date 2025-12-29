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
	"github.com/go-playground/validator/v10"
)

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// ChangePassword 修改密码
func ChangePassword(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()

	// 验证原密码
	if !auth.VerifyPassword(req.CurrentPassword, user.Password) {
		utils.ErrorResponse(c, http.StatusBadRequest, "原密码错误", nil)
		return
	}

	// 验证新密码强度
	valid, msg := auth.ValidatePasswordStrength(req.NewPassword, 8)
	if !valid {
		utils.ErrorResponse(c, http.StatusBadRequest, msg, nil)
		return
	}

	// 更新密码
	hashedPassword, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "密码加密失败", err)
		return
	}

	user.Password = hashedPassword
	if err := db.Save(&user).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新密码失败", err)
		return
	}

	// 记录审计日志
	utils.CreateAuditLogSimple(c, "change_password", "user", user.ID,
		fmt.Sprintf("用户修改密码: %s", user.Email))

	// 发送密码修改成功邮件
	go func() {
		emailService := email.NewEmailService()
		templateBuilder := email.NewEmailTemplateBuilder()
		baseURL := utils.GetBuildBaseURL(c.Request, database.GetDB())
		loginURL := fmt.Sprintf("%s/login", baseURL)
		changeTime := utils.GetBeijingTime().Format("2006-01-02 15:04:05")
		content := templateBuilder.GetPasswordChangedTemplate(user.Username, changeTime, loginURL)
		subject := "密码修改成功"
		_ = emailService.QueueEmail(user.Email, subject, content, "password_changed")
	}()

	utils.SetResponseStatus(c, http.StatusOK)
	utils.SuccessResponse(c, http.StatusOK, "密码修改成功", nil)
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
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	// 验证密码强度（最小长度8位，要求包含大小写字母、数字和特殊字符中的至少三种）
	valid, msg := auth.ValidatePasswordStrength(req.Password, 8)
	if !valid {
		utils.ErrorResponse(c, http.StatusBadRequest, msg, nil)
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "用户不存在", err)
		return
	}

	// 更新密码
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "密码加密失败", err)
		return
	}

	user.Password = hashedPassword
	if err := db.Save(&user).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "重置密码失败", err)
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
	utils.SuccessResponse(c, http.StatusOK, "密码重置成功", nil)
}

// ForgotPasswordRequest 忘记密码请求
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ForgotPassword 忘记密码（发送验证码）
func ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 提供更详细的错误信息
		if validationErr, ok := err.(validator.ValidationErrors); ok {
			for _, fieldErr := range validationErr {
				if fieldErr.Field() == "Email" {
					if fieldErr.Tag() == "required" {
						utils.ErrorResponse(c, http.StatusBadRequest, "请输入邮箱地址", err)
					} else if fieldErr.Tag() == "email" {
						utils.ErrorResponse(c, http.StatusBadRequest, "邮箱格式不正确，请输入有效的邮箱地址", err)
					} else {
						utils.ErrorResponse(c, http.StatusBadRequest, "邮箱格式不正确", err)
					}
					return
				}
			}
		}
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误，请检查输入信息", err)
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		// 为了安全，即使用户不存在也返回成功
		utils.SuccessResponse(c, http.StatusOK, "如果该邮箱存在，验证码已发送", nil)
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
		utils.ErrorResponse(c, http.StatusInternalServerError, "保存验证码失败", err)
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
			utils.ErrorResponse(c, http.StatusInternalServerError, "发送验证码邮件失败", err)
			return
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "验证码已发送，请查收邮箱", nil)
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
		// 提供更详细的错误信息
		if validationErr, ok := err.(validator.ValidationErrors); ok {
			for _, fieldErr := range validationErr {
				switch fieldErr.Field() {
				case "Email":
					if fieldErr.Tag() == "required" {
						utils.ErrorResponse(c, http.StatusBadRequest, "请输入邮箱地址", err)
					} else if fieldErr.Tag() == "email" {
						utils.ErrorResponse(c, http.StatusBadRequest, "邮箱格式不正确，请输入有效的邮箱地址", err)
					}
					return
				case "VerificationCode":
					utils.ErrorResponse(c, http.StatusBadRequest, "请输入验证码", err)
					return
				case "NewPassword":
					if fieldErr.Tag() == "required" {
						utils.ErrorResponse(c, http.StatusBadRequest, "请输入新密码", err)
					} else if fieldErr.Tag() == "min" {
						utils.ErrorResponse(c, http.StatusBadRequest, "密码长度至少8位", err)
					}
					return
				}
			}
		}
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误，请检查输入信息", err)
		return
	}

	// 验证密码强度
	valid, msg := auth.ValidatePasswordStrength(req.NewPassword, 8)
	if !valid {
		utils.ErrorResponse(c, http.StatusBadRequest, msg, nil)
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "该邮箱未注册，请检查邮箱地址是否正确", nil)
		return
	}

	// 验证验证码格式（6位数字）
	if len(req.VerificationCode) != 6 {
		utils.ErrorResponse(c, http.StatusBadRequest, "验证码格式错误，请输入6位数字验证码", nil)
		return
	}

	// 先检查是否有该邮箱的验证码（用于区分验证码不存在和验证码错误）
	var codeCount int64
	db.Model(&models.VerificationCode{}).Where("email = ? AND purpose = ?", req.Email, "reset_password").Count(&codeCount)
	if codeCount == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "未找到该邮箱的验证码，请先获取验证码", nil)
		return
	}

	// 检查验证码是否已使用
	var usedCode models.VerificationCode
	if err := db.Where("email = ? AND code = ? AND used = ? AND purpose = ?", req.Email, req.VerificationCode, 1, "reset_password").First(&usedCode).Error; err == nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "验证码已使用，请重新获取验证码", nil)
		return
	}

	// 验证验证码
	var verificationCode models.VerificationCode
	if err := db.Where("email = ? AND code = ? AND used = ? AND purpose = ?", req.Email, req.VerificationCode, 0, "reset_password").Order("created_at DESC").First(&verificationCode).Error; err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "验证码错误，请检查后重新输入", nil)
		return
	}

	// 检查验证码是否过期
	if verificationCode.IsExpired() {
		utils.ErrorResponse(c, http.StatusBadRequest, "验证码已过期，请重新获取验证码", nil)
		return
	}

	// 标记验证码为已使用（防止重复使用）
	verificationCode.Used = 1
	if err := db.Model(&verificationCode).Where("id = ?", verificationCode.ID).Update("used", 1).Error; err != nil {
		utils.LogError("ResetPasswordByCode: mark verification code as used failed", err, map[string]interface{}{
			"code_id": verificationCode.ID,
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "标记验证码失败", err)
		return
	}

	// 更新密码
	hashedPassword, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "密码加密失败", err)
		return
	}

	user.Password = hashedPassword
	if err := db.Save(&user).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "重置密码失败", err)
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
	utils.SuccessResponse(c, http.StatusOK, "密码重置成功", nil)
}
