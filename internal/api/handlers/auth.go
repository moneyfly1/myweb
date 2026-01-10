package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cboard-go/internal/core/auth"
	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/services/email"
	"cboard-go/internal/services/geoip"
	"cboard-go/internal/services/notification"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
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

// LoginJSONRequest 登录请求
type LoginJSONRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 提供更详细的错误信息
		if validationErr, ok := err.(validator.ValidationErrors); ok {
			for _, fieldErr := range validationErr {
				switch fieldErr.Field() {
				case "Email":
					utils.ErrorResponse(c, http.StatusBadRequest, "邮箱格式不正确，请输入有效的邮箱地址", err)
					return
				case "Username":
					utils.ErrorResponse(c, http.StatusBadRequest, "用户名不能为空", err)
					return
				case "Password":
					if fieldErr.Tag() == "min" {
						utils.ErrorResponse(c, http.StatusBadRequest, "密码长度至少8位", err)
					} else {
						utils.ErrorResponse(c, http.StatusBadRequest, "密码不能为空", err)
					}
					return
				}
			}
		}
		utils.ErrorResponse(c, http.StatusBadRequest, "请求格式错误，请检查输入信息", err)
		return
	}

	db := database.GetDB()

	// 分别检查邮箱和用户名，给出更明确的错误提示
	var emailUser models.User
	if err := db.Where("email = ?", req.Email).First(&emailUser).Error; err == nil {
		// 邮箱已注册，提示用户直接登录
		utils.ErrorResponse(c, http.StatusBadRequest, "该邮箱已被注册，请直接登录或使用其他邮箱", nil)
		return
	}

	var usernameUser models.User
	if err := db.Where("username = ?", req.Username).First(&usernameUser).Error; err == nil {
		// 用户名已存在
		utils.ErrorResponse(c, http.StatusBadRequest, "用户名已被使用，请选择其他用户名", nil)
		return
	}

	// 验证密码强度
	valid, msg := auth.ValidatePasswordStrength(req.Password, 8)
	if !valid {
		utils.ErrorResponse(c, http.StatusBadRequest, msg, nil)
		return
	}

	// 检查邮箱验证开关
	var emailVerificationConfig models.SystemConfig
	emailVerificationRequired := true // 默认需要验证
	if err := db.Where("key = ? AND category = ?", "email_verification_required", "registration").First(&emailVerificationConfig).Error; err == nil {
		emailVerificationRequired = emailVerificationConfig.Value == "true"
	}

	// 如果邮箱验证开关打开，需要验证验证码
	if emailVerificationRequired {
		if req.VerificationCode == "" {
			utils.ErrorResponse(c, http.StatusBadRequest, "请输入邮箱验证码", nil)
			return
		}

		// 验证验证码格式（6位数字）
		if len(req.VerificationCode) != 6 {
			utils.ErrorResponse(c, http.StatusBadRequest, "验证码格式错误，请输入6位数字验证码", nil)
			return
		}

		// 先检查是否有该邮箱的验证码（用于区分验证码不存在和验证码错误）
		var codeCount int64
		db.Model(&models.VerificationCode{}).Where("email = ? AND purpose = ?", req.Email, "register").Count(&codeCount)
		if codeCount == 0 {
			utils.ErrorResponse(c, http.StatusBadRequest, "未找到该邮箱的验证码，请先获取验证码", nil)
			return
		}

		// 检查验证码是否已使用
		var usedCode models.VerificationCode
		if err := db.Where("email = ? AND code = ? AND used = ? AND purpose = ?", req.Email, req.VerificationCode, 1, "register").First(&usedCode).Error; err == nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "验证码已使用，请重新获取验证码", nil)
			return
		}

		// 验证验证码
		var verificationCode models.VerificationCode
		if err := db.Where("email = ? AND code = ? AND used = ? AND purpose = ?", req.Email, req.VerificationCode, 0, "register").Order("created_at DESC").First(&verificationCode).Error; err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "验证码错误，请检查后重新输入", nil)
			return
		}

		// 检查验证码是否过期
		if verificationCode.IsExpired() {
			utils.ErrorResponse(c, http.StatusBadRequest, "验证码已过期，请重新获取验证码", nil)
			return
		}

		// 标记验证码为已使用
		verificationCode.MarkAsUsed()
		db.Save(&verificationCode)
	}

	// 创建用户
	hashed, _ := auth.HashPassword(req.Password)
	user := models.User{
		Username:   req.Username,
		Email:      req.Email,
		Password:   hashed,
		IsActive:   true,
		IsVerified: true, // 注册成功即视为已验证（无论是否通过验证码）
	}

	if err := db.Create(&user).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建用户失败", err)
		return
	}

	// 创建默认订阅
	_ = createDefaultSubscription(db, user.ID)

	// 处理邀请码
	if req.InviteCode != "" {
		processInviteCode(db, req.InviteCode, user.ID)
	}

	// 发送管理员通知（新用户注册）
	go func() {
		notificationService := notification.NewNotificationService()
		registerTime := utils.GetBeijingTime().Format("2006-01-02 15:04:05")
		_ = notificationService.SendAdminNotification("user_registered", map[string]interface{}{
			"username":      user.Username,
			"email":         user.Email,
			"register_time": registerTime,
		})
	}()

	// 发送客户欢迎邮件（新用户注册通知）
	go func() {
		if notification.ShouldSendCustomerNotification("new_user") {
			emailService := email.NewEmailService()
			templateBuilder := email.NewEmailTemplateBuilder()
			baseURL := templateBuilder.GetBaseURL()
			loginURL := fmt.Sprintf("%s/login", baseURL)
			
			content := templateBuilder.GetWelcomeTemplate(
				user.Username,
				user.Email,
				loginURL,
				false, // 不包含密码（用户自己设置的密码）
				"",    // 不显示密码
			)
			
			if err := emailService.QueueEmail(user.Email, "欢迎加入我们！", content, "welcome"); err != nil {
				utils.LogErrorMsg("发送欢迎邮件失败: email=%s, error=%v", user.Email, err)
			} else {
				utils.LogInfo("欢迎邮件已加入队列: email=%s", user.Email)
			}
		} else {
			utils.LogInfo("欢迎邮件未发送: email=%s, 客户通知已禁用", user.Email)
		}
	}()

	utils.SuccessResponse(c, http.StatusCreated, "注册成功", gin.H{"id": user.ID, "email": user.Email})
}

func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}
	db := database.GetDB()
	ipAddress := utils.GetRealClientIP(c)

	// 检测批量登录和撞库行为
	isSuspicious, reason := utils.CheckBruteForcePattern(c, req.Email)
	if isSuspicious {
		// 记录可疑行为
		utils.CreateSecurityLog(c, "login_attempt", "HIGH",
			fmt.Sprintf("检测到可疑登录行为: %s", reason),
			map[string]interface{}{
				"email":  req.Email,
				"ip":     ipAddress,
				"reason": reason,
			})
	}

	// 记录登录尝试
	utils.CreateSecurityLog(c, "login_attempt", "INFO",
		fmt.Sprintf("登录尝试: 邮箱 %s", req.Email),
		map[string]interface{}{
			"email": req.Email,
			"ip":    ipAddress,
		})

	user, err := auth.AuthenticateUser(db, req.Email, req.Password)
	if err != nil {
		// 登录失败，增加计数
		middleware.IncrementLoginAttempt(ipAddress)

		// 检查是否被封禁（增加计数后检查）
		_, _, locked := middleware.GetLoginAttemptStatus(ipAddress)

		// 记录密码错误日志
		severity := "MEDIUM"
		if locked {
			severity = "HIGH"
		}

		utils.CreateSecurityLog(c, "login_failed", severity,
			fmt.Sprintf("登录失败: 邮箱或密码错误 (IP: %s)", ipAddress),
			map[string]interface{}{
				"email":  req.Email,
				"ip":     ipAddress,
				"reason": "密码错误或用户不存在",
				"locked": locked,
			})

		utils.ErrorResponse(c, http.StatusUnauthorized, "邮箱或密码错误", err)
		return
	}
	if !user.IsActive {
		// 账号被禁用，增加计数
		middleware.IncrementLoginAttempt(ipAddress)

		// 记录账号被禁用日志
		utils.CreateSecurityLog(c, "login_blocked", "HIGH",
			fmt.Sprintf("登录被阻止: 账号已禁用 (用户: %s, IP: %s)", user.Username, ipAddress),
			map[string]interface{}{
				"user_id":  user.ID,
				"email":    req.Email,
				"username": user.Username,
				"ip":       ipAddress,
				"reason":   "账号已禁用",
			})

		utils.ErrorResponse(c, http.StatusForbidden, "账号已禁用", nil)
		return
	}

	atk, _ := utils.CreateAccessToken(user.ID, user.Email, user.IsAdmin)
	rtk, _ := utils.CreateRefreshToken(user.ID, user.Email)

	// 更新最后登录时间
	now := utils.GetBeijingTime()
	user.LastLogin = database.NullTime(now)
	if saveErr := db.Save(&user).Error; saveErr != nil {
		utils.LogError("Login: 更新最后登录时间失败", saveErr, nil)
	}

	// 登录成功，重置限流计数
	middleware.ResetLoginAttempt(ipAddress)

	// 创建登录历史记录（包含User-Agent和IP地址）
	userAgent := c.GetHeader("User-Agent")

	// 解析地理位置（如果 GeoIP 已启用）
	var location sql.NullString
	if geoip.IsEnabled() {
		location = geoip.GetLocationString(ipAddress)
	}

	loginHistory := models.LoginHistory{
		UserID:      user.ID,
		LoginTime:   now,
		IPAddress:   database.NullString(ipAddress),
		UserAgent:   database.NullString(userAgent),
		Location:    location,
		LoginStatus: "success",
	}
	// 创建登录历史（同步创建，确保记录被保存）
	if err := db.Create(&loginHistory).Error; err != nil {
		utils.LogError("Login: 创建登录历史失败", err, map[string]interface{}{
			"user_id": user.ID,
			"ip":      ipAddress,
		})
		// 即使创建失败也不影响登录流程
	}

	// 设置用户ID到上下文，以便审计日志可以获取
	c.Set("user_id", user.ID)
	utils.SetResponseStatus(c, http.StatusOK)

	// 记录登录成功日志
	utils.CreateSecurityLog(c, "login_success", "INFO",
		fmt.Sprintf("登录成功: 用户 %s (IP: %s)", user.Username, ipAddress),
		map[string]interface{}{
			"user_id":  user.ID,
			"email":    user.Email,
			"username": user.Username,
			"ip":       ipAddress,
		})

	// 记录登录审计日志
	utils.CreateAuditLogSimple(c, "login", "auth", user.ID, fmt.Sprintf("用户登录: %s", user.Username))

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{"access_token": atk, "refresh_token": rtk, "user": user})
}

// LoginJSON 登录
func LoginJSON(c *gin.Context) {
	var req LoginJSONRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	ipAddress := utils.GetRealClientIP(c)

	// 检查维护模式：维护模式下只允许管理员登录
	var maintenanceConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "maintenance_mode", "system").First(&maintenanceConfig).Error; err == nil {
		if maintenanceConfig.Value == "true" {
			// 维护模式下，先验证用户身份
			var tempUser models.User
			if err := db.Where("email = ? OR username = ?", req.Username, req.Username).First(&tempUser).Error; err != nil {
				// 登录失败，增加计数
				middleware.IncrementLoginAttempt(ipAddress)

				// 记录登录失败日志
				utils.CreateSecurityLog(c, "login_failed", "MEDIUM",
					fmt.Sprintf("登录失败: 维护模式下用户不存在 (IP: %s)", ipAddress),
					map[string]interface{}{
						"username": req.Username,
						"ip":       ipAddress,
						"reason":   "维护模式下用户不存在",
					})

				utils.ErrorResponse(c, http.StatusUnauthorized, "用户名或密码错误", err)
				return
			}
			if !auth.VerifyPassword(req.Password, tempUser.Password) {
				// 登录失败，增加计数
				middleware.IncrementLoginAttempt(ipAddress)

				// 记录密码错误日志
				utils.CreateSecurityLog(c, "login_failed", "MEDIUM",
					fmt.Sprintf("登录失败: 维护模式下密码错误 (IP: %s)", ipAddress),
					map[string]interface{}{
						"username": req.Username,
						"ip":       ipAddress,
						"reason":   "维护模式下密码错误",
					})

				utils.ErrorResponse(c, http.StatusUnauthorized, "用户名或密码错误", nil)
				return
			}
			// 维护模式下，只有管理员可以登录
			if !tempUser.IsAdmin {
				// 非管理员在维护模式下无法登录，增加计数
				middleware.IncrementLoginAttempt(ipAddress)

				// 记录非管理员在维护模式下尝试登录
				utils.CreateSecurityLog(c, "login_blocked", "MEDIUM",
					fmt.Sprintf("登录被阻止: 维护模式下非管理员尝试登录 (用户: %s, IP: %s)", tempUser.Username, ipAddress),
					map[string]interface{}{
						"user_id":  tempUser.ID,
						"username": tempUser.Username,
						"ip":       ipAddress,
						"reason":   "维护模式下非管理员无法登录",
					})

				utils.ErrorResponse(c, http.StatusServiceUnavailable, "系统维护中，请稍后再试", nil)
				return
			}
			// 管理员可以继续登录流程
		}
	}

	// 检测批量登录和撞库行为
	isSuspicious, reason := utils.CheckBruteForcePattern(c, req.Username)
	if isSuspicious {
		// 记录可疑行为
		utils.CreateSecurityLog(c, "login_attempt", "HIGH",
			fmt.Sprintf("检测到可疑登录行为: %s", reason),
			map[string]interface{}{
				"username": req.Username,
				"ip":       ipAddress,
				"reason":   reason,
			})
	}

	// 记录登录尝试
	utils.CreateSecurityLog(c, "login_attempt", "INFO",
		fmt.Sprintf("登录尝试: 用户名 %s", req.Username),
		map[string]interface{}{
			"username": req.Username,
			"ip":       ipAddress,
		})

	var user models.User
	if err := db.Where("email = ? OR username = ?", req.Username, req.Username).First(&user).Error; err != nil {
		// 登录失败，增加计数
		middleware.IncrementLoginAttempt(ipAddress)

		// 检查是否被封禁
		_, _, locked := middleware.GetLoginAttemptStatus(ipAddress)

		// 记录用户不存在或密码错误日志
		severity := "MEDIUM"
		if locked {
			severity = "HIGH"
		}

		utils.CreateSecurityLog(c, "login_failed", severity,
			fmt.Sprintf("登录失败: 用户名或密码错误 (IP: %s)", ipAddress),
			map[string]interface{}{
				"username": req.Username,
				"ip":       ipAddress,
				"reason":   "用户不存在或密码错误",
				"locked":   locked,
			})

		utils.ErrorResponse(c, http.StatusUnauthorized, "用户名或密码错误", err)
		return
	}

	// 检查用户是否激活
	if !user.IsActive {
		// 账号被禁用，增加计数
		middleware.IncrementLoginAttempt(ipAddress)

		// 记录账号被禁用日志
		utils.CreateSecurityLog(c, "login_blocked", "HIGH",
			fmt.Sprintf("登录被阻止: 账号已禁用 (用户: %s, IP: %s)", user.Username, ipAddress),
			map[string]interface{}{
				"user_id":  user.ID,
				"username": user.Username,
				"email":    user.Email,
				"ip":       ipAddress,
				"reason":   "账号已禁用",
			})

		utils.ErrorResponse(c, http.StatusForbidden, "账户已被禁用，无法使用服务。如有疑问，请联系管理员。", nil)
		return
	}

	if !auth.VerifyPassword(req.Password, user.Password) {
		// 登录失败，增加计数
		middleware.IncrementLoginAttempt(ipAddress)

		// 检查是否被封禁
		_, _, locked := middleware.GetLoginAttemptStatus(ipAddress)

		// 记录密码错误日志
		severity := "MEDIUM"
		if locked {
			severity = "HIGH"
		}

		utils.CreateSecurityLog(c, "login_failed", severity,
			fmt.Sprintf("登录失败: 密码错误 (用户: %s, IP: %s)", user.Username, ipAddress),
			map[string]interface{}{
				"user_id":  user.ID,
				"username": user.Username,
				"email":    user.Email,
				"ip":       ipAddress,
				"reason":   "密码错误",
				"locked":   locked,
			})

		utils.ErrorResponse(c, http.StatusUnauthorized, "用户名或密码错误", nil)
		return
	}

	accessToken, err := utils.CreateAccessToken(user.ID, user.Email, user.IsAdmin)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成令牌失败", err)
		return
	}

	refreshToken, err := utils.CreateRefreshToken(user.ID, user.Email)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成刷新令牌失败", err)
		return
	}

	// 更新最后登录时间
	now := utils.GetBeijingTime()
	user.LastLogin = database.NullTime(now)
	if saveErr := db.Save(&user).Error; saveErr != nil {
		// 记录错误但不影响登录流程
		utils.LogError("更新最后登录时间失败", saveErr, nil)
	}

	// 登录成功，重置限流计数
	middleware.ResetLoginAttempt(ipAddress)

	// 创建登录历史记录
	userAgent := c.GetHeader("User-Agent")

	// 解析地理位置（如果 GeoIP 已启用）
	var location sql.NullString
	if geoip.IsEnabled() {
		location = geoip.GetLocationString(ipAddress)
	}

	loginHistory := models.LoginHistory{
		UserID:      user.ID,
		LoginTime:   now,
		IPAddress:   database.NullString(ipAddress),
		UserAgent:   database.NullString(userAgent),
		Location:    location,
		LoginStatus: "success",
	}
	// 创建登录历史（同步创建，确保记录被保存）
	if err := db.Create(&loginHistory).Error; err != nil {
		utils.LogError("LoginJSON: 创建登录历史失败", err, map[string]interface{}{
			"user_id": user.ID,
			"ip":      ipAddress,
		})
		// 即使创建失败也不影响登录流程
	}

	// 设置用户ID到上下文，以便审计日志可以获取
	c.Set("user_id", user.ID)
	utils.SetResponseStatus(c, http.StatusOK)

	// 记录登录成功日志
	utils.CreateSecurityLog(c, "login_success", "INFO",
		fmt.Sprintf("登录成功: 用户 %s (IP: %s)", user.Username, ipAddress),
		map[string]interface{}{
			"user_id":  user.ID,
			"username": user.Username,
			"email":    user.Email,
			"ip":       ipAddress,
		})

	// 记录登录审计日志
	utils.CreateAuditLogSimple(c, "login", "auth", user.ID, fmt.Sprintf("用户登录: %s", user.Username))

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "bearer",
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"is_admin": user.IsAdmin,
		},
	})
}

// RefreshToken 刷新令牌
func RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	claims, err := utils.VerifyToken(req.RefreshToken)
	if err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, "无效的刷新令牌", err)
		return
	}

	if claims.Type != "refresh" {
		utils.ErrorResponse(c, http.StatusUnauthorized, "令牌类型错误", nil)
		return
	}

	// 生成新的访问令牌
	accessToken, err := utils.CreateAccessToken(claims.UserID, claims.Email, claims.IsAdmin)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成令牌失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"access_token": accessToken,
		"token_type":   "bearer",
	})
}

// Logout 登出
func Logout(c *gin.Context) {
	// 获取当前用户
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	// 获取Token
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未提供认证令牌", nil)
		return
	}

	// 提取 Bearer token
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		utils.ErrorResponse(c, http.StatusUnauthorized, "无效的认证格式", nil)
		return
	}

	token := parts[1]

	// 验证Token并获取过期时间
	claims, err := utils.VerifyToken(token)
	if err != nil {
		// Token无效或已过期，仍然返回成功（避免信息泄露）
		utils.SuccessResponse(c, http.StatusOK, "登出成功", nil)
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

	utils.SuccessResponse(c, http.StatusOK, "登出成功", nil)
}

// processInviteCode 处理邀请码（注册时使用）
func processInviteCode(db *gorm.DB, inviteCodeStr string, newUserID uint) {
	if inviteCodeStr == "" {
		return
	}

	// 统一转换为大写进行查询（大小写不敏感）
	inviteCodeStr = strings.ToUpper(strings.TrimSpace(inviteCodeStr))

	// 查找邀请码（使用 UPPER() 函数进行大小写不敏感查询）
	var inviteCode models.InviteCode
	if err := db.Where("UPPER(code) = ? AND is_active = ?", inviteCodeStr, true).First(&inviteCode).Error; err != nil {
		// 邀请码不存在或已停用，忽略错误（不影响注册流程）
		return
	}

	// 检查邀请码是否有效
	now := utils.GetBeijingTime()
	if inviteCode.ExpiresAt.Valid && inviteCode.ExpiresAt.Time.Before(now) {
		// 邀请码已过期，忽略
		return
	}

	if inviteCode.MaxUses.Valid && inviteCode.UsedCount >= int(inviteCode.MaxUses.Int64) {
		// 邀请码使用次数已达上限，忽略
		return
	}

	// 检查是否已经存在邀请关系（防止重复使用）
	var existingRelation models.InviteRelation
	if err := db.Where("invitee_id = ?", newUserID).First(&existingRelation).Error; err == nil {
		// 该用户已经使用过邀请码，忽略
		return
	}

	// 创建邀请关系
	inviteRelation := models.InviteRelation{
		InviteCodeID:        inviteCode.ID,
		InviterID:           inviteCode.UserID,
		InviteeID:           newUserID,
		InviterRewardGiven:  false,
		InviteeRewardGiven:  false,
		InviterRewardAmount: inviteCode.InviterReward,
		InviteeRewardAmount: inviteCode.InviteeReward,
	}

	if err := db.Create(&inviteRelation).Error; err != nil {
		// 创建邀请关系失败，记录错误但不影响注册流程
		utils.LogError("processInviteCode: create invite relation failed", err, map[string]interface{}{
			"invite_code_id": inviteCode.ID,
			"new_user_id":    newUserID,
		})
		return
	}

	// 更新邀请码使用次数
	inviteCode.UsedCount++
	if err := db.Save(&inviteCode).Error; err != nil {
		utils.LogError("processInviteCode: update invite code used count failed", err, map[string]interface{}{
			"invite_code_id": inviteCode.ID,
		})
	}

	// 更新用户的邀请码使用记录
	var newUser models.User
	if err := db.First(&newUser, newUserID).Error; err == nil {
		newUser.InviteCodeUsed = database.NullString(inviteCodeStr)
		db.Save(&newUser)
	}

	// 根据 MinOrderAmount 决定是否立即发放奖励
	// 如果 MinOrderAmount = 0，注册时立即发放奖励
	if inviteCode.MinOrderAmount == 0 {
		// 立即发放邀请者奖励（如果大于0）
		if inviteCode.InviterReward > 0 {
			var inviter models.User
			if err := db.First(&inviter, inviteCode.UserID).Error; err == nil {
				inviter.Balance += inviteCode.InviterReward
				inviter.TotalInviteReward += inviteCode.InviterReward
				inviter.TotalInviteCount++
				if err := db.Save(&inviter).Error; err == nil {
					inviteRelation.InviterRewardGiven = true
					db.Save(&inviteRelation)
					if utils.AppLogger != nil {
						utils.AppLogger.Info("processInviteCode: ✅ 发放邀请者注册奖励 - inviter_id=%d, amount=%.2f, invitee_id=%d",
							inviter.ID, inviteCode.InviterReward, newUserID)
					}
				} else {
					utils.LogError("processInviteCode: failed to give inviter reward", err, map[string]interface{}{
						"inviter_id": inviter.ID,
						"amount":     inviteCode.InviterReward,
					})
				}
			}
		}

		// 立即发放被邀请者奖励（如果大于0）
		if inviteCode.InviteeReward > 0 {
			var invitee models.User
			if err := db.First(&invitee, newUserID).Error; err == nil {
				invitee.Balance += inviteCode.InviteeReward
				if err := db.Save(&invitee).Error; err == nil {
					inviteRelation.InviteeRewardGiven = true
					db.Save(&inviteRelation)
					if utils.AppLogger != nil {
						utils.AppLogger.Info("processInviteCode: ✅ 发放被邀请者注册奖励 - invitee_id=%d, amount=%.2f",
							invitee.ID, inviteCode.InviteeReward)
					}
				} else {
					utils.LogError("processInviteCode: failed to give invitee reward", err, map[string]interface{}{
						"invitee_id": invitee.ID,
						"amount":     inviteCode.InviteeReward,
					})
				}
			}
		}
	} else {
		// MinOrderAmount > 0，需要等待订单支付成功后发放奖励
		if utils.AppLogger != nil {
			utils.AppLogger.Info("processInviteCode: ⏳ 等待订单支付后发放奖励 - invitee_id=%d, min_order_amount=%.2f",
				newUserID, inviteCode.MinOrderAmount)
		}
	}
}
