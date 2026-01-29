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

type RegisterRequest struct {
	Username         string `json:"username" binding:"required"`
	Email            string `json:"email" binding:"required,email"`
	Password         string `json:"password" binding:"required,min=8"`
	VerificationCode string `json:"verification_code"`
	InviteCode       string `json:"invite_code"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginJSONRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleValidationError(c, err)
		return
	}

	db := database.GetDB()

	var count int64
	if db.Model(&models.User{}).Where("email = ?", req.Email).Count(&count); count > 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "该邮箱已被注册，请直接登录或使用其他邮箱", nil)
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
					return fmt.Errorf("该邮箱已被注册，请直接登录或使用其他邮箱")
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
		if strings.Contains(errMsg, "邮箱已被注册") || strings.Contains(errMsg, "用户名已被使用") {
			utils.ErrorResponse(c, http.StatusBadRequest, errMsg, nil)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, errMsg, err)
		}
		return
	}

	db.Where("id = ?", user.ID).First(&user)

	ipAddress := c.ClientIP()
	if ipAddress == "" {
		ipAddress = c.RemoteIP()
	}

	atk, err := utils.CreateAccessToken(user.ID, user.Email, user.IsAdmin)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成令牌失败", err)
		return
	}
	rtk, err := utils.CreateRefreshToken(user.ID, user.Email)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成刷新令牌失败", err)
		return
	}

	now := utils.GetBeijingTime()
	user.LastLogin = database.NullTime(now)
	if saveErr := db.Save(user).Error; saveErr != nil {
		utils.LogError("Register: 更新最后登录时间失败", saveErr, nil)
	}

	var location sql.NullString
	if geoip.IsEnabled() {
		location = geoip.GetLocationString(ipAddress)
	}

	loginHistory := models.LoginHistory{
		UserID:      user.ID,
		LoginTime:   now,
		IPAddress:   database.NullString(ipAddress),
		UserAgent:   database.NullString(c.GetHeader("User-Agent")),
		Location:    location,
		LoginStatus: "success",
	}
	if err := db.Create(&loginHistory).Error; err != nil {
		utils.LogError("Register: 创建登录历史失败", err, map[string]interface{}{"user_id": user.ID, "ip": ipAddress})
	}

	utils.CreateSecurityLog(c, "register_success", "INFO",
		fmt.Sprintf("注册成功: 用户 %s (IP: %s)", user.Username, ipAddress),
		map[string]interface{}{"user_id": user.ID, "username": user.Username, "ip": ipAddress})

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

	var user models.User
	if err := db.Where("email = ? OR username = ?", req.Username, req.Username).First(&user).Error; err != nil {
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
		utils.SuccessResponse(c, http.StatusOK, "登出成功", nil)
		return
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}

	if err := models.AddToBlacklist(database.GetDB(), utils.HashToken(token), user.ID, expiresAt); err != nil {
		utils.LogError("Logout: failed to add token to blacklist", err, map[string]interface{}{"user_id": user.ID})
	}

	utils.SuccessResponse(c, http.StatusOK, "登出成功", nil)
}

func handleValidationError(c *gin.Context, err error) {
	if validationErr, ok := err.(validator.ValidationErrors); ok {
		for _, fieldErr := range validationErr {
			switch fieldErr.Field() {
			case "Email":
				if fieldErr.Tag() == "email" {
					utils.ErrorResponse(c, http.StatusBadRequest, "邮箱格式不正确，请输入有效的邮箱地址（例如：user@example.com）", nil)
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
	db.Model(&models.VerificationCode{}).Where("email = ? AND purpose = ?", emailStr, "register").Count(&codeCount)
	if codeCount == 0 {
		return fmt.Errorf("未找到该邮箱的验证码，请先获取验证码")
	}

	var usedCode models.VerificationCode
	if err := db.Where("email = ? AND code = ? AND used = ? AND purpose = ?", emailStr, code, 1, "register").First(&usedCode).Error; err == nil {
		return fmt.Errorf("验证码已使用，请重新获取验证码")
	}

	var verificationCode models.VerificationCode
	if err := db.Where("email = ? AND code = ? AND used = ? AND purpose = ?", emailStr, code, 0, "register").Order("created_at DESC").First(&verificationCode).Error; err != nil {
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
		registerTime := utils.GetBeijingTime().Format("2006-01-02 15:04:05")
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

func checkMaintenanceMode(c *gin.Context, db *gorm.DB, username, password, ip string) error {
	var maintenanceConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "maintenance_mode", "system").First(&maintenanceConfig).Error; err != nil || maintenanceConfig.Value != "true" {
		return nil
	}

	var tempUser models.User
	if err := db.Where("email = ? OR username = ?", username, username).First(&tempUser).Error; err != nil {
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
	rtk, err := utils.CreateRefreshToken(user.ID, user.Email)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成刷新令牌失败", err)
		return
	}

	now := utils.GetBeijingTime()
	user.LastLogin = database.NullTime(now)
	if saveErr := db.Save(user).Error; saveErr != nil {
		utils.LogError("Login: 更新最后登录时间失败", saveErr, nil)
	}

	middleware.ResetLoginAttempt(ipAddress)

	var location sql.NullString
	if geoip.IsEnabled() {
		location = geoip.GetLocationString(ipAddress)
	}

	loginHistory := models.LoginHistory{
		UserID:      user.ID,
		LoginTime:   now,
		IPAddress:   database.NullString(ipAddress),
		UserAgent:   database.NullString(c.GetHeader("User-Agent")),
		Location:    location,
		LoginStatus: "success",
	}
	if err := db.Create(&loginHistory).Error; err != nil {
		utils.LogError("Login: 创建登录历史失败", err, map[string]interface{}{"user_id": user.ID, "ip": ipAddress})
	}

	c.Set("user_id", user.ID)
	utils.SetResponseStatus(c, http.StatusOK)

	utils.CreateSecurityLog(c, "login_success", "INFO",
		fmt.Sprintf("登录成功: 用户 %s (IP: %s)", user.Username, ipAddress),
		map[string]interface{}{"user_id": user.ID, "username": user.Username, "ip": ipAddress})
	utils.CreateAuditLogSimple(c, "login", "auth", user.ID, fmt.Sprintf("用户登录: %s", user.Username))

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

	var inviteCode models.InviteCode
	if err := db.Where("UPPER(code) = ? AND is_active = ?", inviteCodeStr, true).First(&inviteCode).Error; err != nil {
		return
	}

	now := utils.GetBeijingTime()
	if (inviteCode.ExpiresAt.Valid && inviteCode.ExpiresAt.Time.Before(now)) ||
		(inviteCode.MaxUses.Valid && inviteCode.UsedCount >= int(inviteCode.MaxUses.Int64)) {
		return
	}

	var existingRelation models.InviteRelation
	if err := db.Where("invitee_id = ?", newUserID).First(&existingRelation).Error; err == nil {
		return
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

	if err := db.Create(&inviteRelation).Error; err != nil {
		utils.LogError("processInviteCode: create invite relation failed", err, map[string]interface{}{"invite_code_id": inviteCode.ID, "new_user_id": newUserID})
		return
	}

	inviteCode.UsedCount++
	db.Save(&inviteCode)

	var newUser models.User
	if err := db.First(&newUser, newUserID).Error; err == nil {
		newUser.InviteCodeUsed = database.NullString(inviteCodeStr)
		db.Save(&newUser)
	}

	if inviteCode.MinOrderAmount == 0 {
		distributeReward(db, inviteCode.UserID, inviteCode.InviterReward, newUserID, &inviteRelation, true)
		distributeReward(db, newUserID, inviteCode.InviteeReward, newUserID, &inviteRelation, false)
	} else if utils.AppLogger != nil {
		utils.AppLogger.Info("processInviteCode: ⏳ 等待订单支付后发放奖励 - invitee_id=%d, min_order_amount=%.2f", newUserID, inviteCode.MinOrderAmount)
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

	user.Balance += amount
	if isInviter {
		user.TotalInviteReward += amount
		user.TotalInviteCount++
	}

	if err := db.Save(&user).Error; err == nil {
		if isInviter {
			relation.InviterRewardGiven = true
		} else {
			relation.InviteeRewardGiven = true
		}
		db.Save(relation)
		if utils.AppLogger != nil {
			utils.AppLogger.Info("processInviteCode: ✅ 发放奖励 - user_id=%d, amount=%.2f, related_id=%d", userID, amount, relatedUserID)
		}
	} else {
		utils.LogError("processInviteCode: failed to give reward", err, map[string]interface{}{"user_id": userID, "amount": amount})
	}
}
