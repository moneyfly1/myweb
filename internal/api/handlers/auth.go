package handlers

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"cboard-go/internal/core/auth"
	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/services/device"
	"cboard-go/internal/services/email"
	"cboard-go/internal/services/geoip"
	"cboard-go/internal/services/notification"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RegisterRequest struct {
	Username         string `json:"username" binding:"required"`
	Email            string `json:"email" binding:"required,email,max=255"`
	Password         string `json:"password" binding:"required,min=8"`
	VerificationCode string `json:"verification_code"`
	InviteCode       string `json:"invite_code"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required"`
}

type LoginJSONRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func refreshCookieName(isAdmin bool) string {
	if isAdmin {
		return "cboard_admin_refresh_token"
	}
	return "cboard_user_refresh_token"
}

func isSecureRequest(c *gin.Context) bool {
	if c.Request != nil && c.Request.TLS != nil {
		return true
	}
	return strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
}

func setRefreshTokenCookie(c *gin.Context, isAdmin bool, token string) {
	days := 30
	if cfg := config.AppConfig; cfg != nil && cfg.RefreshTokenExpireDays > 0 {
		days = cfg.RefreshTokenExpireDays
	}
	maxAge := days * 24 * 3600
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(refreshCookieName(isAdmin), token, maxAge, "/", "", isSecureRequest(c), true)
}

func clearRefreshTokenCookies(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	secure := isSecureRequest(c)
	c.SetCookie(refreshCookieName(false), "", -1, "/", "", secure, true)
	c.SetCookie(refreshCookieName(true), "", -1, "/", "", secure, true)
}

func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleValidationError(c, err)
		return
	}

	req.Email = utils.NormalizeEmail(req.Email)
	db := database.GetDB()

	var count int64
	if db.Model(&models.User{}).Where("LOWER(email) = ?", req.Email).Count(&count); count > 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "该邮箱已注册，请直接登录。如忘记密码，请点击找回密码。", nil)
		return
	}
	if db.Model(&models.User{}).Where("username = ?", req.Username).Count(&count); count > 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "用户名已被使用，请选择其他用户名", nil)
		return
	}

	if valid, msg := auth.ValidatePasswordStrength(req.Password, 8); !valid {
		utils.ErrorResponse(c, http.StatusBadRequest, msg, nil)
		return
	}

	if err := verifyRegisterCode(db, req.Email, req.VerificationCode); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	var user models.User
	err := utils.WithTransaction(db, func(tx *gorm.DB) error {
		hashed, hashErr := auth.HashPassword(req.Password)
		if hashErr != nil {
			return fmt.Errorf("密码加密失败: %v", hashErr)
		}
		user = models.User{
			Username:   req.Username,
			Email:      req.Email,
			Password:   hashed,
			IsActive:   true,
			IsVerified: true,
		}
		if err := tx.Create(&user).Error; err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint") || strings.Contains(err.Error(), "Duplicate entry") {
				if strings.Contains(err.Error(), "email") || strings.Contains(err.Error(), "Email") {
					return fmt.Errorf("该邮箱已注册，请直接登录。如忘记密码，请点击找回密码。")
				}
				if strings.Contains(err.Error(), "username") || strings.Contains(err.Error(), "Username") {
					return fmt.Errorf("用户名已被使用，请选择其他用户名")
				}
				return fmt.Errorf("邮箱或用户名已被使用，请检查后重试")
			}
			return fmt.Errorf("创建用户失败: %v", err)
		}
		if err := createDefaultSubscription(tx, user.ID); err != nil {
			return fmt.Errorf("创建默认订阅失败: %v", err)
		}
		if req.InviteCode != "" {
			processInviteCode(tx, req.InviteCode, user.ID)
		}
		return nil
	})

	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "邮箱已注册") || strings.Contains(errMsg, "用户名已被使用") {
			utils.ErrorResponse(c, http.StatusBadRequest, errMsg, nil)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, errMsg, err)
		}
		return
	}

	db.Where("id = ?", user.ID).First(&user)

	ipAddress := utils.GetRealClientIP(c)

	atk, err := utils.CreateAccessToken(user.ID, user.Email, user.IsAdmin)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成令牌失败", err)
		return
	}
	rtk, err := utils.CreateRefreshToken(user.ID, user.Email, user.IsAdmin)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成刷新令牌失败", err)
		return
	}
	setRefreshTokenCookie(c, user.IsAdmin, rtk)

	now := utils.GetBeijingTime()
	user.LastLogin = database.NullTime(now)
	if saveErr := db.Save(user).Error; saveErr != nil {
		utils.LogError("Register: 更新最后登录时间失败", saveErr, nil)
	}

	// 异步记录登录历史，不阻塞注册流程
	go func(userID uint, ip, ua string, loginTime time.Time) {
		var location sql.NullString
		if geoip.IsEnabled() {
			location = geoip.GetLocationWithCache(ip)
		}

		loginHistory := models.LoginHistory{
			UserID:      userID,
			LoginTime:   loginTime,
			IPAddress:   database.NullString(ip),
			UserAgent:   database.NullString(ua),
			Location:    location,
			LoginStatus: "success",
		}
		if err := database.GetDB().Create(&loginHistory).Error; err != nil {
			utils.LogError("Register: 创建登录历史失败", err, map[string]interface{}{"user_id": userID, "ip": ip})
		}
	}(user.ID, ipAddress, c.GetHeader("User-Agent"), now)

	utils.CreateSecurityLog(c, "register_success", "INFO",
		fmt.Sprintf("注册成功: 用户 %s (IP: %s)", user.Username, ipAddress),
		map[string]interface{}{"user_id": user.ID, "username": user.Username, "ip": ipAddress})

	// 记录注册日志
	var inviterID *uint
	if user.InvitedBy.Valid {
		id := utils.MustSafeInt64ToUint(user.InvitedBy.Int64)
		inviterID = &id
	}
	go func() {
		if err := utils.CreateRegistrationLog(
			user.ID,
			user.Username,
			user.Email,
			ipAddress,
			c.GetHeader("User-Agent"),
			req.InviteCode,
			inviterID,
		); err != nil {
			log.Printf("failed to create registration log: %v", err)
		}
	}()

	handleRegisterNotification(user)
	utils.SuccessResponse(c, http.StatusCreated, "注册成功", gin.H{
		"access_token":  atk,
		"refresh_token": rtk,
		"token_type":    "bearer",
		"user": gin.H{
			"id":          user.ID,
			"username":    user.Username,
			"email":       user.Email,
			"is_admin":    user.IsAdmin,
			"is_verified": user.IsVerified,
			"is_active":   user.IsActive,
		},
	})
}

func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	req.Email = utils.NormalizeEmail(req.Email)
	db := database.GetDB()
	ipAddress := utils.GetRealClientIP(c)
	checkSuspiciousLogin(c, req.Email, ipAddress)

	user, err := auth.AuthenticateUser(db, req.Email, req.Password)
	if err != nil {
		handleLoginFailure(c, ipAddress, req.Email, "密码错误或用户不存在", err)
		return
	}

	finalizeLogin(c, db, user, ipAddress)
}

func LoginJSON(c *gin.Context) {
	var req LoginJSONRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	ipAddress := utils.GetRealClientIP(c)

	if err := checkMaintenanceMode(c, db, req.Username, req.Password, ipAddress); err != nil {
		return
	}

	checkSuspiciousLogin(c, req.Username, ipAddress)

	normalizedInput := utils.NormalizeEmail(req.Username)
	var user models.User
	if err := db.Where("LOWER(email) = ? OR username = ?", normalizedInput, req.Username).First(&user).Error; err != nil {
		handleLoginFailure(c, ipAddress, req.Username, "用户不存在或密码错误", err)
		return
	}

	if !auth.VerifyPassword(req.Password, user.Password) {
		handleLoginFailure(c, ipAddress, req.Username, "密码错误", nil)
		return
	}

	finalizeLogin(c, db, &user, ipAddress)
}

func RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", nil)
		return
	}

	refreshToken := strings.TrimSpace(req.RefreshToken)
	if refreshToken == "" {
		role := strings.ToLower(strings.TrimSpace(c.GetHeader("X-Auth-Role")))
		if role == "admin" {
			if cookie, err := c.Cookie(refreshCookieName(true)); err == nil && cookie != "" {
				refreshToken = cookie
			}
		} else if role == "user" {
			if cookie, err := c.Cookie(refreshCookieName(false)); err == nil && cookie != "" {
				refreshToken = cookie
			}
		}
		if refreshToken == "" {
			if cookie, err := c.Cookie(refreshCookieName(true)); err == nil && cookie != "" {
				refreshToken = cookie
			} else if cookie, err := c.Cookie(refreshCookieName(false)); err == nil && cookie != "" {
				refreshToken = cookie
			}
		}
	}
	if refreshToken == "" {
		utils.ErrorResponse(c, http.StatusUnauthorized, "缺少刷新令牌", nil)
		return
	}

	tokenHash := utils.HashToken(refreshToken)
	if models.IsTokenBlacklisted(database.GetDB(), tokenHash) {
		utils.ErrorResponse(c, http.StatusUnauthorized, "刷新令牌已失效，请重新登录", nil)
		return
	}

	claims, err := utils.VerifyToken(refreshToken)
	if err != nil {
		// token 过期是正常客户端行为，不记录安全日志
		utils.ErrorResponse(c, http.StatusUnauthorized, "刷新令牌已过期", nil)
		return
	}
	if claims.Type != "refresh" {
		utils.ErrorResponse(c, http.StatusUnauthorized, "令牌类型错误", nil)
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.Where("id = ?", claims.UserID).First(&user).Error; err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, "用户不存在或已被删除", nil)
		return
	}
	if !user.IsActive {
		utils.ErrorResponse(c, http.StatusForbidden, "账户已被禁用", nil)
		return
	}

	if claims.ExpiresAt != nil {
		_ = models.AddToBlacklist(db, tokenHash, user.ID, claims.ExpiresAt.Time)
	}

	accessToken, err := utils.CreateAccessToken(user.ID, user.Email, user.IsAdmin)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成令牌失败", err)
		return
	}
	newRefreshToken, err := utils.CreateRefreshToken(user.ID, user.Email, user.IsAdmin)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成刷新令牌失败", err)
		return
	}
	setRefreshTokenCookie(c, user.IsAdmin, newRefreshToken)

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
		"token_type":    "bearer",
	})
}

func Logout(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未提供认证令牌", nil)
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		utils.ErrorResponse(c, http.StatusUnauthorized, "无效的认证格式", nil)
		return
	}

	token := parts[1]
	claims, err := utils.VerifyToken(token)
	if err != nil {
		clearRefreshTokenCookies(c)
		utils.SuccessResponse(c, http.StatusOK, "登出成功", nil)
		return
	}

	expiresAt := utils.GetBeijingTime().Add(24 * time.Hour)
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}

	if err := models.AddToBlacklist(database.GetDB(), utils.HashToken(token), user.ID, expiresAt); err != nil {
		utils.LogError("Logout: failed to add token to blacklist", err, map[string]interface{}{"user_id": user.ID})
	}
	if refreshCookie, err := c.Cookie(refreshCookieName(user.IsAdmin)); err == nil && refreshCookie != "" {
		if refreshClaims, verifyErr := utils.VerifyToken(refreshCookie); verifyErr == nil && refreshClaims.ExpiresAt != nil {
			_ = models.AddToBlacklist(database.GetDB(), utils.HashToken(refreshCookie), user.ID, refreshClaims.ExpiresAt.Time)
		}
	}
	clearRefreshTokenCookies(c)

	utils.SuccessResponse(c, http.StatusOK, "登出成功", nil)
}

func handleValidationError(c *gin.Context, err error) {
	if validationErr, ok := err.(validator.ValidationErrors); ok {
		for _, fieldErr := range validationErr {
			switch fieldErr.Field() {
			case "Email":
				if fieldErr.Tag() == "email" {
					utils.ErrorResponse(c, http.StatusBadRequest, "邮箱格式不正确，请输入有效的邮箱地址", nil)
				} else {
					utils.ErrorResponse(c, http.StatusBadRequest, "邮箱不能为空，请输入您的邮箱地址", nil)
				}
				return
			case "Username":
				if fieldErr.Tag() == "required" {
					utils.ErrorResponse(c, http.StatusBadRequest, "用户名不能为空，请输入用户名", nil)
				} else {
					utils.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("用户名验证失败: %s", fieldErr.Tag()), nil)
				}
				return
			case "Password":
				if fieldErr.Tag() == "min" {
					utils.ErrorResponse(c, http.StatusBadRequest, "密码长度至少8位，请设置更长的密码", nil)
				} else if fieldErr.Tag() == "required" {
					utils.ErrorResponse(c, http.StatusBadRequest, "密码不能为空，请输入密码", nil)
				} else {
					utils.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("密码验证失败: %s", fieldErr.Tag()), nil)
				}
				return
			}
		}
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数验证失败，请检查输入信息", err)
		return
	}
	utils.ErrorResponse(c, http.StatusBadRequest, "请求格式错误，请检查输入信息", err)
}

func verifyRegisterCode(db *gorm.DB, emailStr, code string) error {
	var emailVerificationConfig models.SystemConfig
	required := true
	if err := db.Where("key = ? AND category = ?", "email_verification_required", "registration").First(&emailVerificationConfig).Error; err == nil {
		required = emailVerificationConfig.Value == "true"
	}

	if !required {
		return nil
	}

	if code == "" {
		return fmt.Errorf("请输入邮箱验证码")
	}
	if len(code) != 6 {
		return fmt.Errorf("验证码格式错误，请输入6位数字验证码")
	}

	var codeCount int64
	db.Model(&models.VerificationCode{}).Where("LOWER(email) = ? AND purpose = ?", emailStr, "register").Count(&codeCount)
	if codeCount == 0 {
		return fmt.Errorf("未找到该邮箱的验证码，请先获取验证码")
	}

	var usedCode models.VerificationCode
	if err := db.Where("LOWER(email) = ? AND code = ? AND used = ? AND purpose = ?", emailStr, code, 1, "register").First(&usedCode).Error; err == nil {
		return fmt.Errorf("验证码已使用，请重新获取验证码")
	}

	var verificationCode models.VerificationCode
	if err := db.Where("LOWER(email) = ? AND code = ? AND used = ? AND purpose = ?", emailStr, code, 0, "register").Order("created_at DESC").First(&verificationCode).Error; err != nil {
		return fmt.Errorf("验证码错误，请检查后重新输入")
	}

	if verificationCode.IsExpired() {
		return fmt.Errorf("验证码已过期，请重新获取验证码")
	}

	verificationCode.MarkAsUsed()
	db.Save(&verificationCode)
	return nil
}

func handleRegisterNotification(user models.User) {
	go func() {
		notificationService := notification.NewNotificationService()
		registerTime := utils.FormatBeijingTime(utils.GetBeijingTime())
		_ = notificationService.SendAdminNotification("user_registered", map[string]interface{}{
			"username":      user.Username,
			"email":         user.Email,
			"register_time": registerTime,
		})
	}()

	go func() {
		if !notification.ShouldSendCustomerNotification("new_user") {
			utils.LogInfo("欢迎邮件未发送: email=%s, 客户通知已禁用", user.Email)
			return
		}
		emailService := email.NewEmailService()
		templateBuilder := email.NewEmailTemplateBuilder()
		loginURL := fmt.Sprintf("%s/login", templateBuilder.GetBaseURL())
		content := templateBuilder.GetWelcomeTemplate(user.Username, user.Email, loginURL, false, "")

		if err := emailService.QueueEmail(user.Email, "欢迎加入我们！", content, "welcome"); err != nil {
			utils.LogErrorMsg("发送欢迎邮件失败: email=%s, error=%v", user.Email, err)
		} else {
			utils.LogInfo("欢迎邮件已加入队列: email=%s", user.Email)
		}
	}()
}

func checkSuspiciousLogin(c *gin.Context, identifier, ipAddress string) {
	isSuspicious, reason := utils.CheckBruteForcePattern(c, identifier)
	if isSuspicious {
		utils.CreateSecurityLog(c, "login_attempt", "HIGH",
			fmt.Sprintf("检测到可疑登录行为: %s", reason),
			map[string]interface{}{"identifier": identifier, "ip": ipAddress, "reason": reason})
	}
	utils.CreateSecurityLog(c, "login_attempt", "INFO",
		fmt.Sprintf("登录尝试: %s", identifier),
		map[string]interface{}{"identifier": identifier, "ip": ipAddress})
}

func handleLoginFailure(c *gin.Context, ip, identifier, reason string, err error) {
	middleware.IncrementLoginAttempt(ip)
	_, _, locked := middleware.GetLoginAttemptStatus(ip)
	severity := "MEDIUM"
	if locked {
		severity = "HIGH"
	}

	utils.CreateSecurityLog(c, "login_failed", severity,
		fmt.Sprintf("登录失败: %s (IP: %s)", reason, ip),
		map[string]interface{}{
			"identifier": identifier,
			"ip":         ip,
			"reason":     reason,
			"locked":     locked,
		})

	utils.ErrorResponse(c, http.StatusUnauthorized, "用户名或密码错误", err)
}

// checkAndSendAbnormalLoginAlert 在 goroutine 中执行：检测新设备/异地登录并发送邮件与站内通知
func checkAndSendAbnormalLoginAlert(db *gorm.DB, userID uint, current *models.LoginHistory, ipAddress, userAgent string) {
	var cfg models.SystemConfig
	if err := db.Where("category = ? AND key = ?", "security", "abnormal_login_alert_enabled").First(&cfg).Error; err == nil && cfg.Value == "false" {
		return
	}

	var prev models.LoginHistory
	if err := db.Where("user_id = ? AND login_status = ? AND id != ?", userID, "success", current.ID).Order("login_time DESC").Limit(1).First(&prev).Error; err != nil {
		return
	}

	currCountry, _ := current.GetLocationInfo()
	prevCountry, _ := prev.GetLocationInfo()
	isNewLocation := currCountry != "" && prevCountry != "" && currCountry != prevCountry

	deviceHash := ""
	if current.DeviceFingerprint.Valid {
		deviceHash = current.DeviceFingerprint.String
	}
	isNewDevice := false
	if deviceHash != "" {
		ninetyDaysAgo := utils.GetBeijingTime().Add(-90 * 24 * time.Hour)
		var count int64
		db.Model(&models.LoginHistory{}).Where("user_id = ? AND login_status = ? AND id != ? AND login_time >= ? AND device_fingerprint = ?",
			userID, "success", current.ID, ninetyDaysAgo, deviceHash).Count(&count)
		isNewDevice = count == 0
	}

	// 同一个 IP 不触发告警（即使设备或地区不同）
	if prev.IPAddress.Valid && prev.IPAddress.String == ipAddress {
		return
	}

	if !isNewDevice && !isNewLocation {
		return
	}

	// 查询完整的用户信息（包括 email, username, is_admin, abnormal_login_alert_enabled）
	var user models.User
	if err := db.Select("id, email, username, is_admin, abnormal_login_alert_enabled").First(&user, userID).Error; err != nil {
		utils.LogErrorMsg("查询用户信息失败: user_id=%d, error=%v", userID, err)
		return
	}

	// 尊重用户通知设置：仅当用户开启「异常登录/设备告警」时发送邮件和站内通知
	if !user.AbnormalLoginAlertEnabled {
		return
	}
	// 管理员账户：同时受系统开关「管理员异常登录告警」控制
	if user.IsAdmin {
		var adminCfg models.SystemConfig
		if err := db.Where("category = ? AND key = ?", "admin_notification", "admin_abnormal_login_alert_enabled").First(&adminCfg).Error; err == nil && adminCfg.Value == "false" {
			return
		}
	}

	locationStr := ""
	if current.Location.Valid {
		locationStr = current.Location.String
	}
	loginTimeStr := utils.FormatBeijingTime(current.LoginTime)
	templateBuilder := email.NewEmailTemplateBuilder()
	content := templateBuilder.GetAbnormalLoginAlertTemplate(user.Username, loginTimeStr, ipAddress, locationStr, isNewDevice, isNewLocation)
	emailSvc := email.NewEmailService()
	if err := emailSvc.QueueEmail(user.Email, "账户登录安全提醒", content, "abnormal_login_alert"); err != nil {
		utils.LogErrorMsg("异常登录告警邮件入队失败: email=%s, error=%v", user.Email, err)
	}

	notifContent := "检测到您的账户在新设备或新地点登录。如非本人操作请尽快修改密码。"
	if isNewDevice && isNewLocation {
		notifContent = "检测到您的账户在新设备且新地点登录。如非本人操作请尽快修改密码。"
	} else if isNewDevice {
		notifContent = "检测到您的账户在新设备登录。如非本人操作请尽快修改密码。"
	} else {
		notifContent = "检测到您的账户在异地登录。如非本人操作请尽快修改密码。"
	}
	uid := utils.MustSafeUintToInt64(user.ID)
	if err := db.Create(&models.Notification{UserID: sql.NullInt64{Int64: uid, Valid: true}, Title: "账户登录安全提醒", Content: notifContent, Type: "security"}).Error; err != nil {
		utils.LogErrorMsg("创建异常登录站内通知失败: user_id=%d, error=%v", user.ID, err)
	}
}

func checkMaintenanceMode(c *gin.Context, db *gorm.DB, username, password, ip string) error {
	var maintenanceConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "maintenance_mode", "system").First(&maintenanceConfig).Error; err != nil || maintenanceConfig.Value != "true" {
		return nil
	}

	normalizedInput := utils.NormalizeEmail(username)
	var tempUser models.User
	if err := db.Where("LOWER(email) = ? OR username = ?", normalizedInput, username).First(&tempUser).Error; err != nil {
		handleLoginFailure(c, ip, username, "维护模式下用户不存在", err)
		return fmt.Errorf("auth error")
	}

	if !auth.VerifyPassword(password, tempUser.Password) {
		handleLoginFailure(c, ip, username, "维护模式下密码错误", nil)
		return fmt.Errorf("auth error")
	}

	if !tempUser.IsAdmin {
		middleware.IncrementLoginAttempt(ip)
		utils.CreateSecurityLog(c, "login_blocked", "MEDIUM",
			fmt.Sprintf("登录被阻止: 维护模式下非管理员尝试登录 (用户: %s, IP: %s)", tempUser.Username, ip),
			map[string]interface{}{
				"user_id": tempUser.ID, "username": tempUser.Username, "ip": ip, "reason": "维护模式下非管理员无法登录",
			})
		utils.ErrorResponse(c, http.StatusServiceUnavailable, "系统维护中，请稍后再试", nil)
		return fmt.Errorf("maintenance")
	}
	return nil
}

func finalizeLogin(c *gin.Context, db *gorm.DB, user *models.User, ipAddress string) {
	if !user.IsActive {
		middleware.IncrementLoginAttempt(ipAddress)
		utils.CreateSecurityLog(c, "login_blocked", "HIGH",
			fmt.Sprintf("登录被阻止: 账号已禁用 (用户: %s, IP: %s)", user.Username, ipAddress),
			map[string]interface{}{"user_id": user.ID, "username": user.Username, "ip": ipAddress, "reason": "账号已禁用"})
		utils.ErrorResponse(c, http.StatusForbidden, "账户已被禁用，无法使用服务。如有疑问，请联系管理员。", nil)
		return
	}

	atk, err := utils.CreateAccessToken(user.ID, user.Email, user.IsAdmin)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成令牌失败", err)
		return
	}
	rtk, err := utils.CreateRefreshToken(user.ID, user.Email, user.IsAdmin)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成刷新令牌失败", err)
		return
	}
	setRefreshTokenCookie(c, user.IsAdmin, rtk)

	now := utils.GetBeijingTime()
	user.LastLogin = database.NullTime(now)
	if saveErr := db.Save(user).Error; saveErr != nil {
		utils.LogError("Login: 更新最后登录时间失败", saveErr, nil)
	}

	middleware.ResetLoginAttempt(ipAddress)

	ua := c.GetHeader("User-Agent")
	deviceHash := device.NewDeviceManager().GenerateDeviceHash(ua, ipAddress, "")

	// 异步记录登录历史，不阻塞登录流程
	go func(userID uint, ip, userAgent, devHash string, loginTime time.Time) {
		var location sql.NullString
		if geoip.IsEnabled() {
			location = geoip.GetLocationWithCache(ip)
		}

		loginHistory := models.LoginHistory{
			UserID:            userID,
			LoginTime:         loginTime,
			IPAddress:         database.NullString(ip),
			UserAgent:         database.NullString(userAgent),
			Location:          location,
			LoginStatus:       "success",
			DeviceFingerprint: database.NullString(devHash),
		}
		if err := database.GetDB().Create(&loginHistory).Error; err != nil {
			utils.LogError("Login: 创建登录历史失败", err, map[string]interface{}{"user_id": userID, "ip": ip})
		} else {
			// 检查异常登录 - 传入 userID 而不是不完整的 User 对象
			checkAndSendAbnormalLoginAlert(database.GetDB(), userID, &loginHistory, ip, userAgent)
		}
	}(user.ID, ipAddress, ua, deviceHash, now)

	c.Set("user_id", user.ID)
	utils.SetResponseStatus(c, http.StatusOK)

	utils.CreateSecurityLog(c, "login_success", "INFO",
		fmt.Sprintf("登录成功: 用户 %s (IP: %s)", user.Username, ipAddress),
		map[string]interface{}{"user_id": user.ID, "username": user.Username, "ip": ipAddress})
	if user.IsAdmin {
		utils.CreateSecurityLog(c, "admin_login_success", "INFO",
			fmt.Sprintf("管理员登录: %s (IP: %s)", user.Username, ipAddress),
			map[string]interface{}{"user_id": user.ID, "username": user.Username, "ip": ipAddress})
	}
	utils.CreateAuditLogSimpleFast(c, "login", "auth", user.ID, fmt.Sprintf("用户登录: %s", user.Username))

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"access_token":  atk,
		"refresh_token": rtk,
		"token_type":    "bearer",
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"is_admin": user.IsAdmin,
		},
	})
}

func processInviteCode(db *gorm.DB, inviteCodeStr string, newUserID uint) {
	if inviteCodeStr == "" {
		return
	}
	inviteCodeStr = strings.ToUpper(strings.TrimSpace(inviteCodeStr))

	txErr := db.Transaction(func(tx *gorm.DB) error {
		// FOR UPDATE 行锁防止并发超用
		var inviteCode models.InviteCode
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("UPPER(code) = ? AND is_active = ?", inviteCodeStr, true).
			First(&inviteCode).Error; err != nil {
			return nil // 邀请码不存在或未激活，静默返回
		}

		now := utils.GetBeijingTime()
		if inviteCode.ExpiresAt.Valid && inviteCode.ExpiresAt.Time.Before(now) {
			return nil // 已过期
		}
		if inviteCode.MaxUses.Valid && inviteCode.UsedCount >= int(inviteCode.MaxUses.Int64) {
			return nil // 已达上限
		}

		var existingRelation models.InviteRelation
		if err := tx.Where("invitee_id = ?", newUserID).First(&existingRelation).Error; err == nil {
			return nil // 已有邀请关系
		}

		inviteRelation := models.InviteRelation{
			InviteCodeID:        inviteCode.ID,
			InviterID:           inviteCode.UserID,
			InviteeID:           newUserID,
			InviterRewardGiven:  false,
			InviteeRewardGiven:  false,
			InviterRewardAmount: inviteCode.InviterReward,
			InviteeRewardAmount: inviteCode.InviteeReward,
		}

		if err := tx.Create(&inviteRelation).Error; err != nil {
			utils.LogError("processInviteCode: create invite relation failed", err, map[string]interface{}{"invite_code_id": inviteCode.ID, "new_user_id": newUserID})
			return err
		}

		// 原子递增使用次数
		if err := tx.Model(&models.InviteCode{}).Where("id = ?", inviteCode.ID).
			Update("used_count", gorm.Expr("used_count + 1")).Error; err != nil {
			return err
		}

		var newUser models.User
		if err := tx.First(&newUser, newUserID).Error; err == nil {
			newUser.InviteCodeUsed = database.NullString(inviteCodeStr)
			tx.Save(&newUser)
		}

		// 无最低消费要求时立即发放奖励（事务外异步处理）
		if inviteCode.MinOrderAmount == 0 {
			go func() {
				freshDB := database.GetDB()
				distributeReward(freshDB, inviteCode.UserID, inviteCode.InviterReward, newUserID, &inviteRelation, true)
				distributeReward(freshDB, newUserID, inviteCode.InviteeReward, newUserID, &inviteRelation, false)
			}()
		} else if utils.AppLogger != nil {
			utils.AppLogger.Info("processInviteCode: ⏳ 等待订单支付后发放奖励 - invitee_id=%d, min_order_amount=%.2f", newUserID, inviteCode.MinOrderAmount)
		}

		return nil
	})

	if txErr != nil {
		utils.LogError("processInviteCode: transaction failed", txErr, map[string]interface{}{"invite_code": inviteCodeStr, "new_user_id": newUserID})
	}
}

func distributeReward(db *gorm.DB, userID uint, amount float64, relatedUserID uint, relation *models.InviteRelation, isInviter bool) {
	if amount <= 0 {
		return
	}
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return
	}

	oldBalance := user.Balance
	updates := map[string]interface{}{
		"balance": gorm.Expr("balance + ?", amount),
	}
	if isInviter {
		updates["total_invite_reward"] = gorm.Expr("total_invite_reward + ?", amount)
		updates["total_invite_count"] = gorm.Expr("total_invite_count + 1")
	}

	if err := db.Model(&user).Updates(updates).Error; err == nil {
		// 刷新余额用于日志记录
		db.First(&user, user.ID)
		if isInviter {
			relation.InviterRewardGiven = true
		} else {
			relation.InviteeRewardGiven = true
		}
		db.Save(relation)
		if utils.AppLogger != nil {
			utils.AppLogger.Info("processInviteCode: ✅ 发放奖励 - user_id=%d, amount=%.2f, related_id=%d", userID, amount, relatedUserID)
		}

		// 记录余额日志和佣金日志
		// 注意：这里是在异步 goroutine 中，无法获取 gin.Context，所以 IP 为空
		// 这是系统内部操作（邀请奖励），不是用户直接操作，所以 IP 为空是合理的
		go func() {
			// 余额日志
			if err := utils.CreateBalanceLog(
				userID,
				"commission",
				amount,
				oldBalance,
				user.Balance,
				nil,
				nil,
				fmt.Sprintf("邀请奖励: %s", map[bool]string{true: "邀请人奖励", false: "被邀请人奖励"}[isInviter]),
				"system",
				nil,
				"", // 系统内部操作，无客户端 IP
			); err != nil {
				log.Printf("failed to create balance log: %v", err)
			}

			// 佣金日志
			commissionType := "register_reward"
			inviterID := userID
			inviteeID := relatedUserID
			if !isInviter {
				inviterID = relatedUserID
				inviteeID = userID
			}
			relationID := uint(relation.ID)
			if err := utils.CreateCommissionLog(
				inviterID,
				inviteeID,
				commissionType,
				amount,
				&relationID,
				nil,
				fmt.Sprintf("邀请奖励: %s", map[bool]string{true: "邀请人奖励", false: "被邀请人奖励"}[isInviter]),
			); err != nil {
				log.Printf("failed to create commission log: %v", err)
			}
		}()
	} else {
		utils.LogError("processInviteCode: failed to give reward", err, map[string]interface{}{"user_id": userID, "amount": amount})
	}
}

// ==========================================
// 密码管理（从 password.go 合并）
// ==========================================

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
		utils.CreateSecurityLog(c, "password_change_failed", "MEDIUM",
			fmt.Sprintf("修改密码失败: 原密码错误 (用户: %s)", user.Username),
			map[string]interface{}{"user_id": user.ID, "username": user.Username, "reason": "原密码错误"})
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
		changeTime := utils.FormatBeijingTime(utils.GetBeijingTime())
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

	utils.CreateSecurityLog(c, "admin_reset_password", "MEDIUM",
		fmt.Sprintf("管理员重置用户密码: %s (ID: %s)", user.Username, userID),
		map[string]interface{}{"target_user_id": user.ID, "target_username": user.Username})
	utils.CreateAuditLogSimple(c, "reset_password", "user", user.ID,
		fmt.Sprintf("管理员重置用户密码: %s (%s)", user.Username, user.Email))

	go func() {
		notificationService := notification.NewNotificationService()
		resetTime := utils.FormatBeijingTime(utils.GetBeijingTime())
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
	Email string `json:"email" binding:"required,email,max=255"`
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

	req.Email = utils.NormalizeEmail(req.Email)
	db := database.GetDB()
	ipAddress := utils.GetRealClientIP(c)
	var user models.User
	if err := db.Where("LOWER(email) = ?", req.Email).First(&user).Error; err != nil {
		utils.SuccessResponse(c, http.StatusOK, "如果该邮箱存在，验证码已发送", nil)
		return
	}
	utils.CreateSecurityLog(c, "password_reset_requested", "INFO",
		"用户请求密码重置验证码",
		map[string]interface{}{"email": req.Email, "ip": ipAddress})

	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成验证码失败", err)
		return
	}
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

	sendErr := emailService.SendEmail(user.Email, subject, content)
	if sendErr != nil {
		utils.LogError("RequestPasswordReset: send email failed", sendErr, map[string]interface{}{
			"user_id": user.ID,
		})
	}

	queueErr := emailService.QueueEmail(user.Email, subject, content, "password_reset")
	if queueErr != nil {
		utils.LogError("RequestPasswordReset: queue email failed", queueErr, map[string]interface{}{
			"user_id": user.ID,
		})
		if sendErr != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "发送验证码邮件失败", sendErr)
			return
		}
	} else if sendErr == nil {
		// 直接发送成功，将队列记录标记为已发送
		db := database.GetDB()
		var emailQueue models.EmailQueue
		if err := db.Where("to_email = ? AND subject = ? AND email_type = ? AND status = ?",
			user.Email, subject, "password_reset", "pending").
			Order("created_at DESC").First(&emailQueue).Error; err == nil {
			emailQueue.Status = "sent"
			emailQueue.SentAt = database.NullTime(utils.GetBeijingTime())
			db.Save(&emailQueue)
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "验证码已发送，请查收邮箱", nil)
}

type ResetPasswordByCodeRequest struct {
	Email            string `json:"email" binding:"required,email,max=255"`
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

	req.Email = utils.NormalizeEmail(req.Email)
	db := database.GetDB()
	var user models.User
	if err := db.Where("LOWER(email) = ?", req.Email).First(&user).Error; err != nil {
		utils.CreateSecurityLog(c, "reset_code_failed", "MEDIUM",
			"重置密码失败: 该邮箱未注册（可能撞库/探测）",
			map[string]interface{}{"email": req.Email, "reason": "该邮箱未注册"})
		utils.ErrorResponse(c, http.StatusNotFound, "该邮箱未注册，请检查邮箱地址是否正确", nil)
		return
	}

	if len(req.VerificationCode) != 6 {
		utils.ErrorResponse(c, http.StatusBadRequest, "验证码格式错误，请输入6位数字验证码", nil)
		return
	}

	var codeCount int64
	db.Model(&models.VerificationCode{}).Where("LOWER(email) = ? AND purpose = ?", req.Email, "reset_password").Count(&codeCount)
	if codeCount == 0 {
		utils.CreateSecurityLog(c, "reset_code_failed", "MEDIUM",
			"重置密码失败: 未找到该邮箱的验证码",
			map[string]interface{}{"email": req.Email, "reason": "未找到验证码"})
		utils.ErrorResponse(c, http.StatusBadRequest, "未找到该邮箱的验证码，请先获取验证码", nil)
		return
	}

	var usedCode models.VerificationCode
	if err := db.Where("LOWER(email) = ? AND code = ? AND used = ? AND purpose = ?", req.Email, req.VerificationCode, 1, "reset_password").First(&usedCode).Error; err == nil {
		utils.CreateSecurityLog(c, "reset_code_failed", "MEDIUM",
			"重置密码失败: 验证码已使用",
			map[string]interface{}{"email": req.Email, "reason": "验证码已使用"})
		utils.ErrorResponse(c, http.StatusBadRequest, "验证码已使用，请重新获取验证码", nil)
		return
	}

	var verificationCode models.VerificationCode
	if err := db.Where("LOWER(email) = ? AND code = ? AND used = ? AND purpose = ?", req.Email, req.VerificationCode, 0, "reset_password").Order("created_at DESC").First(&verificationCode).Error; err != nil {
		utils.CreateSecurityLog(c, "reset_code_failed", "MEDIUM",
			"重置密码失败: 验证码错误",
			map[string]interface{}{"email": req.Email, "reason": "验证码错误"})
		utils.ErrorResponse(c, http.StatusBadRequest, "验证码错误，请检查后重新输入", nil)
		return
	}

	if verificationCode.IsExpired() {
		utils.CreateSecurityLog(c, "reset_code_failed", "MEDIUM",
			"重置密码失败: 验证码已过期",
			map[string]interface{}{"email": req.Email, "reason": "验证码已过期"})
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
		resetTime := utils.FormatBeijingTime(utils.GetBeijingTime())
		_ = notificationService.SendAdminNotification("password_reset", map[string]interface{}{
			"username":   user.Username,
			"email":      user.Email,
			"reset_time": resetTime,
		})
	}()

	utils.SetResponseStatus(c, http.StatusOK)
	utils.SuccessResponse(c, http.StatusOK, "密码重置成功", nil)
}

// ==========================================
// 验证码管理（从 verification.go 合并）
// ==========================================

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
		req.Email = utils.NormalizeEmail(req.Email)

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
	ipAddress := utils.GetRealClientIP(c)
	identifier := utils.NormalizeEmail(req.Email)

	fiveMinutesAgo := utils.GetBeijingTime().Add(-5 * time.Minute)
	var failedAttempts int64
	db.Model(&models.VerificationAttempt{}).
		Where("LOWER(email) = ? AND success = ? AND created_at > ?", identifier, false, fiveMinutesAgo).
		Count(&failedAttempts)

	if failedAttempts >= 5 {
		utils.CreateSecurityLog(c, "verify_code_rate_limit", "MEDIUM",
			"验证码尝试次数过多（5分钟内失败≥5次）",
			map[string]interface{}{"email": identifier, "ip": ipAddress, "purpose": "register"})
		utils.ErrorResponse(c, http.StatusTooManyRequests, "验证码尝试次数过多，请5分钟后再试", nil)
		return
	}

	var verificationCode models.VerificationCode
	if err := db.Where("LOWER(email) = ? AND code = ? AND used = ?", identifier, req.Code, 0).Order("created_at DESC").First(&verificationCode).Error; err != nil {
		attempt := models.VerificationAttempt{
			Email:     identifier,
			IPAddress: database.NullString(ipAddress),
			Success:   false,
			Purpose:   "register",
		}
		db.Create(&attempt)
		utils.CreateSecurityLog(c, "verification_code_failed", "MEDIUM",
			"验证码校验失败: 验证码错误或已使用",
			map[string]interface{}{"email": identifier, "ip": ipAddress, "purpose": "register"})
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
		utils.CreateSecurityLog(c, "verification_code_failed", "MEDIUM",
			"验证码校验失败: 验证码已过期",
			map[string]interface{}{"email": identifier, "ip": ipAddress, "purpose": "register"})
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
