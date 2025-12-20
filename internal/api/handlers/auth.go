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

func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "message": "请求格式错误"})
		return
	}
	db := database.GetDB(); var count int64; db.Model(&models.User{}).Where("email = ? OR username = ?", req.Email, req.Username).Count(&count)
	if count > 0 { c.JSON(400, gin.H{"success": false, "message": "用户名或邮箱已存在"}); return }
	hashed, _ := auth.HashPassword(req.Password)
	user := models.User{Username: req.Username, Email: req.Email, Password: hashed, IsActive: true}
	if err := db.Create(&user).Error; err != nil { c.JSON(500, gin.H{"success": false, "message": "创建用户失败"}); return }
	_ = createDefaultSubscription(db, user.ID)
	if req.InviteCode != "" { processInviteCode(db, req.InviteCode, user.ID) }
	c.JSON(201, gin.H{"success": true, "message": "注册成功", "data": gin.H{"id": user.ID, "email": user.Email}})
}

func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "message": "请求参数错误"})
		return
	}
	db := database.GetDB()
	user, err := auth.AuthenticateUser(db, req.Email, req.Password)
	if err != nil {
		c.JSON(401, gin.H{"success": false, "message": "邮箱或密码错误"})
		return
	}
	if !user.IsActive {
		c.JSON(403, gin.H{"success": false, "message": "账号已禁用"})
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
	
	// 创建登录历史记录（包含User-Agent和IP地址）
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
			utils.LogError("Login: 创建登录历史失败", err, nil)
		}
	}()
	
	// 设置用户ID到上下文，以便审计日志可以获取
	c.Set("user_id", user.ID)
	utils.SetResponseStatus(c, http.StatusOK)
	
	// 记录登录审计日志
	utils.CreateAuditLogSimple(c, "login", "auth", user.ID, fmt.Sprintf("用户登录: %s", user.Username))
	
	c.JSON(200, gin.H{"success": true, "data": gin.H{"access_token": atk, "refresh_token": rtk, "user": user}})
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

	// 检查维护模式：维护模式下只允许管理员登录
	var maintenanceConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "maintenance_mode", "system").First(&maintenanceConfig).Error; err == nil {
		if maintenanceConfig.Value == "true" {
			// 维护模式下，先验证用户身份
			var tempUser models.User
			if err := db.Where("email = ? OR username = ?", req.Username, req.Username).First(&tempUser).Error; err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"message": "用户名或密码错误",
				})
				return
			}
			if !auth.VerifyPassword(req.Password, tempUser.Password) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"message": "用户名或密码错误",
				})
				return
			}
			// 维护模式下，只有管理员可以登录
			if !tempUser.IsAdmin {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"success":          false,
					"message":          "系统维护中，请稍后再试",
					"maintenance_mode": true,
				})
				return
			}
			// 管理员可以继续登录流程
		}
	}

	var user models.User
	if err := db.Where("email = ? OR username = ?", req.Username, req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "用户名或密码错误",
		})
		return
	}

	// 检查用户是否激活
	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "账户已被禁用，无法使用服务。如有疑问，请联系管理员。",
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

// processInviteCode 处理邀请码（注册时使用）
func processInviteCode(db *gorm.DB, inviteCodeStr string, newUserID uint) {
	if inviteCodeStr == "" {
		return
	}

	// 查找邀请码
	var inviteCode models.InviteCode
	if err := db.Where("code = ? AND is_active = ?", inviteCodeStr, true).First(&inviteCode).Error; err != nil {
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

	// 如果邀请者奖励大于0，立即发放（注册奖励）
	if inviteCode.InviterReward > 0 {
		var inviter models.User
		if err := db.First(&inviter, inviteCode.UserID).Error; err == nil {
			inviter.Balance += inviteCode.InviterReward
			inviter.TotalInviteReward += inviteCode.InviterReward
			inviter.TotalInviteCount++
			if err := db.Save(&inviter).Error; err == nil {
				inviteRelation.InviterRewardGiven = true
				db.Save(&inviteRelation)
			}
		}
	}

	// 如果被邀请者奖励大于0，立即发放（注册奖励）
	if inviteCode.InviteeReward > 0 {
		var invitee models.User
		if err := db.First(&invitee, newUserID).Error; err == nil {
			invitee.Balance += inviteCode.InviteeReward
			if err := db.Save(&invitee).Error; err == nil {
				inviteRelation.InviteeRewardGiven = true
				db.Save(&inviteRelation)
			}
		}
	}
}
