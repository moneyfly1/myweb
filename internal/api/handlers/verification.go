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

	// 生成6位验证码
	code := generateVerificationCode()

	// 验证码有效期10分钟
	expiresAt := utils.GetBeijingTime().Add(10 * time.Minute)

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

	var verificationCode models.VerificationCode
	if err := db.Where("email = ? AND code = ? AND used = ?", identifier, req.Code, 0).Order("created_at DESC").First(&verificationCode).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "验证码错误或已使用",
		})
		return
	}

	// 检查是否过期
	if verificationCode.IsExpired() {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "验证码已过期",
		})
		return
	}

	// 标记为已使用
	verificationCode.MarkAsUsed()
	db.Save(&verificationCode)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "验证成功",
	})
}

// generateVerificationCode 生成6位数字验证码
func generateVerificationCode() string {
	b := make([]byte, 3)
	rand.Read(b)
	code := int(b[0])<<16 | int(b[1])<<8 | int(b[2])
	return fmt.Sprintf("%06d", code%1000000)
}

