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

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

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

	if !auth.VerifyPassword(req.CurrentPassword, user.Password) {
		utils.ErrorResponse(c, http.StatusBadRequest, "原密码错误", nil)
		return
	}

	valid, msg := auth.ValidatePasswordStrength(req.NewPassword, 8)
	if !valid {
		utils.ErrorResponse(c, http.StatusBadRequest, msg, nil)
		return
	}

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

	utils.CreateAuditLogSimple(c, "change_password", "user", user.ID,
		fmt.Sprintf("用户修改密码: %s", user.Email))

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

type ResetPasswordRequest struct {
	Password string `json:"password" binding:"required,min=8"`
}

func ResetPassword(c *gin.Context) {
	userID := c.Param("id")

	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

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

	utils.CreateAuditLogSimple(c, "reset_password", "user", user.ID,
		fmt.Sprintf("管理员重置用户密码: %s (%s)", user.Username, user.Email))

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

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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
		utils.SuccessResponse(c, http.StatusOK, "如果该邮箱存在，验证码已发送", nil)
		return
	}

	b := make([]byte, 4)
	rand.Read(b)
	codeInt := int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
	codeInt = 100000 + (codeInt % 900000)
	code := fmt.Sprintf("%06d", codeInt)

	expiresAt := utils.GetBeijingTime().Add(10 * time.Minute)

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

	emailService := email.NewEmailService()
	templateBuilder := email.NewEmailTemplateBuilder()
	content := templateBuilder.GetPasswordResetVerificationCodeTemplate(user.Username, code)
	subject := "密码重置验证码"

	if err := emailService.SendEmail(user.Email, subject, content); err != nil {
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

type ResetPasswordByCodeRequest struct {
	Email            string `json:"email" binding:"required,email"`
	VerificationCode string `json:"verification_code" binding:"required"`
	NewPassword      string `json:"new_password" binding:"required,min=8"`
}

func ResetPasswordByCode(c *gin.Context) {
	var req ResetPasswordByCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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

	if len(req.VerificationCode) != 6 {
		utils.ErrorResponse(c, http.StatusBadRequest, "验证码格式错误，请输入6位数字验证码", nil)
		return
	}

	var codeCount int64
	db.Model(&models.VerificationCode{}).Where("email = ? AND purpose = ?", req.Email, "reset_password").Count(&codeCount)
	if codeCount == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "未找到该邮箱的验证码，请先获取验证码", nil)
		return
	}

	var usedCode models.VerificationCode
	if err := db.Where("email = ? AND code = ? AND used = ? AND purpose = ?", req.Email, req.VerificationCode, 1, "reset_password").First(&usedCode).Error; err == nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "验证码已使用，请重新获取验证码", nil)
		return
	}

	var verificationCode models.VerificationCode
	if err := db.Where("email = ? AND code = ? AND used = ? AND purpose = ?", req.Email, req.VerificationCode, 0, "reset_password").Order("created_at DESC").First(&verificationCode).Error; err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "验证码错误，请检查后重新输入", nil)
		return
	}

	if verificationCode.IsExpired() {
		utils.ErrorResponse(c, http.StatusBadRequest, "验证码已过期，请重新获取验证码", nil)
		return
	}

	verificationCode.Used = 1
	if err := db.Model(&verificationCode).Where("id = ?", verificationCode.ID).Update("used", 1).Error; err != nil {
		utils.LogError("ResetPasswordByCode: mark verification code as used failed", err, map[string]interface{}{
			"code_id": verificationCode.ID,
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "标记验证码失败", err)
		return
	}

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

	c.Set("user_id", user.ID)
	utils.SetResponseStatus(c, http.StatusOK)

	utils.CreateAuditLogSimple(c, "reset_password", "user", user.ID,
		fmt.Sprintf("用户通过验证码重置密码: %s (%s)", user.Username, user.Email))

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
