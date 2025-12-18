package handlers

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/email"
	"cboard-go/internal/services/sms"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

// SendVerificationCodeRequest 发送验证码请求
type SendVerificationCodeRequest struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
	Type  string `json:"type" binding:"required"` // email, sms
}

// SendVerificationCode 发送验证码
func SendVerificationCode(c *gin.Context) {
	var req SendVerificationCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()

	// 检查系统配置：注册是否启用（仅对注册验证码进行检查）
	if req.Type == "email" || req.Type == "sms" {
		var registrationConfig models.SystemConfig
		if err := db.Where("key = ? AND category = ?", "registration_enabled", "registration").First(&registrationConfig).Error; err == nil {
			if registrationConfig.Value != "true" {
				c.JSON(http.StatusForbidden, gin.H{
					"success": false,
					"message": "注册功能已禁用，请联系管理员",
				})
				return
			}
		}
	}

	// 生成6位验证码
	code := generateVerificationCode()

	// 验证码有效期5分钟（缩短有效期提高安全性）
	expiresAt := utils.GetBeijingTime().Add(5 * time.Minute)

	if req.Type == "email" {
		if req.Email == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "邮箱不能为空",
			})
			return
		}

		// 保存验证码
		verificationCode := models.VerificationCode{
			Email:     req.Email,
			Code:      code,
			ExpiresAt: expiresAt,
			Used:      0,
			Purpose:   "register",
		}

		if err := db.Create(&verificationCode).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "保存验证码失败",
			})
			return
		}

		// 发送邮件
		emailService := email.NewEmailService()
		if err := emailService.SendVerificationEmail(req.Email, code); err != nil {
			utils.LogError("SendVerificationCode: send email failed", err, map[string]interface{}{
				"email": req.Email,
			})
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "发送邮件失败",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "验证码已发送到邮箱",
		})

	} else if req.Type == "sms" {
		if req.Phone == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "手机号不能为空",
			})
			return
		}

		// 保存验证码（使用邮箱字段存储手机号）
		verificationCode := models.VerificationCode{
			Email:     req.Phone,
			Code:      code,
			ExpiresAt: expiresAt,
			Used:      0,
			Purpose:   "register",
		}

		if err := db.Create(&verificationCode).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "保存验证码失败",
			})
			return
		}

		// 发送短信
		smsService := sms.NewAliyunSMSService()
		if err := smsService.SendVerificationCode(req.Phone, code); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "发送短信失败",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "验证码已发送到手机",
		})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "不支持的验证码类型",
		})
	}
}

// VerifyCodeRequest 验证验证码请求
type VerifyCodeRequest struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
	Code  string `json:"code" binding:"required"`
	Type  string `json:"type" binding:"required"`
}

// VerifyCode 验证验证码
func VerifyCode(c *gin.Context) {
	var req VerifyCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()

	identifier := req.Email
	if req.Type == "sms" {
		identifier = req.Phone
	}

	// 检查最近的验证码尝试次数（最多5次）
	// 检查最近5分钟内失败的尝试次数
	fiveMinutesAgo := utils.GetBeijingTime().Add(-5 * time.Minute)
	var failedAttempts int64
	db.Model(&models.VerificationAttempt{}).
		Where("email = ? AND success = ? AND created_at > ?", identifier, false, fiveMinutesAgo).
		Count(&failedAttempts)

	if failedAttempts >= 5 {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"success": false,
			"message": "验证码尝试次数过多，请5分钟后再试",
		})
		return
	}

	// 获取IP地址用于记录
	ipAddress := c.ClientIP()

	// 查找验证码
	var verificationCode models.VerificationCode
	if err := db.Where("email = ? AND code = ? AND used = ?", identifier, req.Code, 0).Order("created_at DESC").First(&verificationCode).Error; err != nil {
		// 记录失败的尝试
		attempt := models.VerificationAttempt{
			Email:     identifier,
			IPAddress: database.NullString(ipAddress),
			Success:   false,
			Purpose:   "register",
		}
		db.Create(&attempt)

		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "验证码错误或已使用",
		})
		return
	}

	// 检查是否过期
	if verificationCode.IsExpired() {
		// 记录失败的尝试
		attempt := models.VerificationAttempt{
			Email:     identifier,
			IPAddress: database.NullString(ipAddress),
			Success:   false,
			Purpose:   "register",
		}
		db.Create(&attempt)

		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "验证码已过期",
		})
		return
	}

	// 验证成功，记录成功的尝试
	attempt := models.VerificationAttempt{
		Email:     identifier,
		IPAddress: database.NullString(ipAddress),
		Success:   true,
		Purpose:   "register",
	}
	db.Create(&attempt)

	// 标记验证码为已使用
	verificationCode.MarkAsUsed()
	db.Save(&verificationCode)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "验证成功",
	})
}

// generateVerificationCode 生成6位数字验证码（使用加密安全的随机数生成器）
func generateVerificationCode() string {
	// 使用 crypto/rand 生成更安全的随机数
	b := make([]byte, 4)
	rand.Read(b)
	// 使用更大的随机数范围，确保更好的随机性
	code := int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
	// 确保是6位数字（100000-999999）
	code = 100000 + (code % 900000)
	return fmt.Sprintf("%06d", code)
}
