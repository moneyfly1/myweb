package handlers

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/email"

	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

type SendVerificationCodeRequest struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
	Type  string `json:"type" binding:"required"` // email
}

func SendVerificationCode(c *gin.Context) {
	var req SendVerificationCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()

	if req.Type == "email" {
		var registrationConfig models.SystemConfig
		if err := db.Where("key = ? AND category = ?", "registration_enabled", "registration").First(&registrationConfig).Error; err == nil {
			if registrationConfig.Value != "true" {
				utils.ErrorResponse(c, http.StatusForbidden, "注册功能已禁用，请联系管理员", nil)
				return
			}
		}
	}

	code := generateVerificationCode()

	expiresAt := utils.GetBeijingTime().Add(5 * time.Minute)

	if req.Type == "email" {
		if req.Email == "" {
			utils.ErrorResponse(c, http.StatusBadRequest, "邮箱不能为空", nil)
			return
		}

		verificationCode := models.VerificationCode{
			Email:     req.Email,
			Code:      code,
			ExpiresAt: expiresAt,
			Used:      0,
			Purpose:   "register",
		}

		if err := db.Create(&verificationCode).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "保存验证码失败", err)
			return
		}

		emailService := email.NewEmailService()
		if err := emailService.SendVerificationEmail(req.Email, code); err != nil {
			utils.LogError("SendVerificationCode: send email failed", err, map[string]interface{}{
				"email": req.Email,
			})
			utils.ErrorResponse(c, http.StatusInternalServerError, "发送邮件失败", err)
			return
		}

		utils.SuccessResponse(c, http.StatusOK, "验证码已发送到邮箱", nil)

	} else {
		utils.ErrorResponse(c, http.StatusBadRequest, "不支持的验证码类型", nil)
	}

}

type VerifyCodeRequest struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
	Code  string `json:"code" binding:"required"`
	Type  string `json:"type" binding:"required"`
}

func VerifyCode(c *gin.Context) {
	var req VerifyCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()

	identifier := req.Email

	fiveMinutesAgo := utils.GetBeijingTime().Add(-5 * time.Minute)
	var failedAttempts int64
	db.Model(&models.VerificationAttempt{}).
		Where("email = ? AND success = ? AND created_at > ?", identifier, false, fiveMinutesAgo).
		Count(&failedAttempts)

	if failedAttempts >= 5 {
		utils.ErrorResponse(c, http.StatusTooManyRequests, "验证码尝试次数过多，请5分钟后再试", nil)
		return
	}

	ipAddress := utils.GetRealClientIP(c)

	var verificationCode models.VerificationCode
	if err := db.Where("email = ? AND code = ? AND used = ?", identifier, req.Code, 0).Order("created_at DESC").First(&verificationCode).Error; err != nil {
		attempt := models.VerificationAttempt{
			Email:     identifier,
			IPAddress: database.NullString(ipAddress),
			Success:   false,
			Purpose:   "register",
		}
		db.Create(&attempt)

		utils.ErrorResponse(c, http.StatusBadRequest, "验证码错误或已使用", err)
		return
	}

	if verificationCode.IsExpired() {
		attempt := models.VerificationAttempt{
			Email:     identifier,
			IPAddress: database.NullString(ipAddress),
			Success:   false,
			Purpose:   "register",
		}
		db.Create(&attempt)

		utils.ErrorResponse(c, http.StatusBadRequest, "验证码已过期", nil)
		return
	}

	attempt := models.VerificationAttempt{
		Email:     identifier,
		IPAddress: database.NullString(ipAddress),
		Success:   true,
		Purpose:   "register",
	}
	db.Create(&attempt)

	verificationCode.MarkAsUsed()
	db.Save(&verificationCode)

	utils.SuccessResponse(c, http.StatusOK, "验证成功", nil)
}

func generateVerificationCode() string {
	b := make([]byte, 4)
	rand.Read(b)
	code := int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
	code = 100000 + (code % 900000)
	return fmt.Sprintf("%06d", code)
}
