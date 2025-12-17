package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"cboard-go/internal/core/auth"
	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/services/email"
	"cboard-go/internal/services/notification"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username         string `json:"username" binding:"required"`
	Email            string `json:"email" binding:"required,email"`
	Password         string `json:"password" binding:"required,min=8"`
	VerificationCode string `json:"verification_code"` // 验证码（可选，根据系统配置决定是否必填）
	InviteCode       string `json:"invite_code"`       // 邀请码（可选）
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginJSONRequest 兼容用户名或邮箱的登录（前端 /auth/login-json）
type LoginJSONRequest struct {
	Username string `json:"username" binding:"required"` // 可填写用户名或邮箱
	Password string `json:"password" binding:"required"`
}

// Register 注册
func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 不向客户端返回详细错误信息，防止信息泄露
		utils.LogError("Register: bind request", err, nil)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误，请检查输入格式",
		})
		return
	}

	// 验证密码强度
	valid, msg := auth.ValidatePasswordStrength(req.Password, 8)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": msg,
		})
		return
	}

	db := database.GetDB()

	// 检查系统配置：是否需要邮箱验证
	var emailVerificationRequired bool
	var emailVerificationConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "email_verification_required", "registration").First(&emailVerificationConfig).Error; err == nil {
		emailVerificationRequired = emailVerificationConfig.Value == "true"
	} else {
		// 默认需要邮箱验证
		emailVerificationRequired = true
	}

	// 如果系统要求邮箱验证，则验证验证码
	if emailVerificationRequired {
		if req.VerificationCode == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "系统要求邮箱验证，请先获取并输入验证码",
			})
			return
		}

		var verificationCode models.VerificationCode
		if err := db.Where("email = ? AND code = ? AND used = ? AND purpose = ?", req.Email, req.VerificationCode, 0, "register").Order("created_at DESC").First(&verificationCode).Error; err != nil {
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
				"message": "验证码已过期，请重新获取",
			})
			return
		}

		// 标记验证码为已使用（使用ID直接更新，确保更新成功）
		verificationCode.Used = 1
		if err := db.Model(&verificationCode).Where("id = ?", verificationCode.ID).Update("used", 1).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": fmt.Sprintf("标记验证码失败: %v", err),
			})
			return
		}
	}

	// 检查用户是否已存在
	var existingUser models.User
	if err := db.Where("email = ? OR username = ?", req.Email, req.Username).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "邮箱或用户名已存在",
		})
		return
	} else if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "服务器错误",
		})
		return
	}

	// 哈希密码
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "密码加密失败",
		})
		return
	}

	// 创建用户
	// 如果通过验证码注册，邮箱已验证，设置 IsVerified 为 true
	isVerified := false
	if emailVerificationRequired {
		isVerified = true // 通过验证码注册的用户，邮箱已验证
	}
	
	user := models.User{
		Username:   req.Username,
		Email:      req.Email,
		Password:   hashedPassword,
		IsActive:   true,
		IsVerified: isVerified,
		IsAdmin:    false,
	}

	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建用户失败",
		})
		return
	}

	// 为新用户创建默认订阅（忽略错误，不影响注册流程）
	_ = createDefaultSubscription(db, user.ID)

	// 设置用户ID到上下文，以便审计日志可以获取
	c.Set("user_id", user.ID)
	utils.SetResponseStatus(c, http.StatusCreated)

	// 记录注册审计日志
	utils.CreateAuditLogSimple(c, "register", "auth", user.ID, fmt.Sprintf("用户注册: %s (%s)", user.Username, user.Email))

	// 发送欢迎邮件（检查客户通知开关）
	go func() {
		if notification.ShouldSendCustomerNotification("new_user") {
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
			content := templateBuilder.GetWelcomeTemplate(user.Username, user.Email, loginURL, false, "")
			subject := "欢迎加入我们！"
			_ = emailService.QueueEmail(user.Email, subject, content, "welcome")
		}
	}()

	// 发送管理员通知
	go func() {
		notificationService := notification.NewNotificationService()
		registerTime := utils.GetBeijingTime().Format("2006-01-02 15:04:05")
		_ = notificationService.SendAdminNotification("user_registered", map[string]interface{}{
			"username":      user.Username,
			"email":         user.Email,
			"register_time": registerTime,
		})
	}()

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "注册成功",
		"data": gin.H{
			"id":    user.ID,
			"email": user.Email,
		},
	})
}

// Login 登录
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	user, err := auth.AuthenticateUser(db, req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "邮箱或密码错误",
		})
		return
	}

	// 生成令牌
	accessToken, err := utils.CreateAccessToken(user.ID, user.Email, user.IsAdmin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "生成令牌失败",
		})
		return
	}

	refreshToken, err := utils.CreateRefreshToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "生成刷新令牌失败",
		})
		return
	}

	// 更新最后登录时间
	now := utils.GetBeijingTime()
	user.LastLogin = database.NullTime(now)
	if saveErr := db.Save(&user).Error; saveErr != nil {
		// 记录错误但不影响登录流程
		utils.LogError("更新最后登录时间失败", saveErr, nil)
	}

	// 创建登录历史记录
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	loginHistory := models.LoginHistory{
		UserID:      user.ID,
		LoginTime:   now,
		IPAddress:   database.NullString(ipAddress),
		UserAgent:   database.NullString(userAgent),
		LoginStatus: "success",
	}
	// 异步创建登录历史，不阻塞登录流程
	go func() {
		if err := db.Create(&loginHistory).Error; err != nil {
			utils.LogError("创建登录历史失败", err, nil)
		}
	}()

	// 设置用户ID到上下文，以便审计日志可以获取
	c.Set("user_id", user.ID)
	utils.SetResponseStatus(c, http.StatusOK)

	// 记录登录审计日志
	utils.CreateAuditLogSimple(c, "login", "auth", user.ID, fmt.Sprintf("用户登录: %s", user.Email))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"token_type":    "bearer",
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
				"email":    user.Email,
				"is_admin": user.IsAdmin,
			},
		},
	})
}

// LoginJSON 兼容用户名或邮箱的登录，供前端 /auth/login-json 使用
func LoginJSON(c *gin.Context) {
	var req LoginJSONRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.Where("email = ? OR username = ?", req.Username, req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "用户名或密码错误",
		})
		return
	}

	if !auth.VerifyPassword(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "用户名或密码错误",
		})
		return
	}

	accessToken, err := utils.CreateAccessToken(user.ID, user.Email, user.IsAdmin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "生成令牌失败",
		})
		return
	}

	refreshToken, err := utils.CreateRefreshToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "生成刷新令牌失败",
		})
		return
	}

	// 更新最后登录时间
	now := utils.GetBeijingTime()
	user.LastLogin = database.NullTime(now)
	if saveErr := db.Save(&user).Error; saveErr != nil {
		// 记录错误但不影响登录流程
		utils.LogError("更新最后登录时间失败", saveErr, nil)
	}

	// 创建登录历史记录
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	loginHistory := models.LoginHistory{
		UserID:      user.ID,
		LoginTime:   now,
		IPAddress:   database.NullString(ipAddress),
		UserAgent:   database.NullString(userAgent),
		LoginStatus: "success",
	}
	// 异步创建登录历史，不阻塞登录流程
	go func() {
		if err := db.Create(&loginHistory).Error; err != nil {
			utils.LogError("创建登录历史失败", err, nil)
		}
	}()

	// 设置用户ID到上下文，以便审计日志可以获取
	c.Set("user_id", user.ID)
	utils.SetResponseStatus(c, http.StatusOK)

	// 记录登录审计日志
	utils.CreateAuditLogSimple(c, "login", "auth", user.ID, fmt.Sprintf("用户登录: %s", user.Username))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"token_type":    "bearer",
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
				"email":    user.Email,
				"is_admin": user.IsAdmin,
			},
		},
	})
}

// RefreshToken 刷新令牌
func RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	claims, err := utils.VerifyToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "无效的刷新令牌",
		})
		return
	}

	if claims.Type != "refresh" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "令牌类型错误",
		})
		return
	}

	// 生成新的访问令牌
	accessToken, err := utils.CreateAccessToken(claims.UserID, claims.Email, claims.IsAdmin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "生成令牌失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"access_token": accessToken,
			"token_type":   "bearer",
		},
	})
}

// Logout 登出
func Logout(c *gin.Context) {
	// 获取当前用户
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	// 获取Token
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未提供认证令牌",
		})
		return
	}

	// 提取 Bearer token
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "无效的认证格式",
		})
		return
	}

	token := parts[1]

	// 验证Token并获取过期时间
	claims, err := utils.VerifyToken(token)
	if err != nil {
		// Token无效或已过期，仍然返回成功（避免信息泄露）
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "登出成功",
		})
		return
	}

	// 将Token添加到黑名单
	db := database.GetDB()
	tokenHash := utils.HashToken(token)

	// 获取Token的过期时间
	var expiresAt time.Time
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	} else {
		// 如果没有过期时间，使用默认的过期时间（24小时）
		expiresAt = time.Now().Add(24 * time.Hour)
	}

	// 添加到黑名单
	if err := models.AddToBlacklist(db, tokenHash, user.ID, expiresAt); err != nil {
		// 记录错误但不影响登出流程
		utils.LogError("Logout: failed to add token to blacklist", err, map[string]interface{}{
			"user_id": user.ID,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "登出成功",
	})
}
