package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GenerateInviteCode 生成邀请码
func GenerateInviteCode() string {
	b := make([]byte, 8)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:8]
}

// CreateInviteCode 创建邀请码
func CreateInviteCode(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	var req struct {
		MaxUses        int       `json:"max_uses"`
		ExpiresAt      time.Time `json:"expires_at"`
		ExpiresDays    int       `json:"expires_days"` // 支持通过天数设置
		RewardType     string    `json:"reward_type"`
		InviterReward  float64   `json:"inviter_reward"`
		InviteeReward  float64   `json:"invitee_reward"`
		MinOrderAmount float64   `json:"min_order_amount"`
		NewUserOnly    bool      `json:"new_user_only"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.LogError("CreateInviteCode: bind JSON failed", err, nil)
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()

	// 从系统配置获取默认奖励设置（如果请求中未提供）
	if req.InviterReward == 0 || req.InviteeReward == 0 {
		var configs []models.SystemConfig
		db.Where("category = ? AND `key` IN (?)", "invite", []string{
			"inviter_reward", "invitee_reward",
		}).Find(&configs)

		for _, cfg := range configs {
			if cfg.Key == "inviter_reward" && req.InviterReward == 0 {
				if val, err := strconv.ParseFloat(cfg.Value, 64); err == nil {
					req.InviterReward = val
				}
			}
			if cfg.Key == "invitee_reward" && req.InviteeReward == 0 {
				if val, err := strconv.ParseFloat(cfg.Value, 64); err == nil {
					req.InviteeReward = val
				}
			}
		}
	}

	// 如果提供了 expires_days，转换为 expires_at
	if req.ExpiresDays > 0 && req.ExpiresAt.IsZero() {
		req.ExpiresAt = utils.GetBeijingTime().AddDate(0, 0, req.ExpiresDays)
	}

	// 设置默认值
	if req.RewardType == "" {
		req.RewardType = "balance"
	}

	// 生成唯一邀请码（统一转换为大写存储）
	var code string
	maxAttempts := 10
	for i := 0; i < maxAttempts; i++ {
		code = strings.ToUpper(GenerateInviteCode())
		var existing models.InviteCode
		if err := db.Where("UPPER(code) = ?", code).First(&existing).Error; err == gorm.ErrRecordNotFound {
			break
		}
		if i == maxAttempts-1 {
			utils.ErrorResponse(c, http.StatusInternalServerError, "生成唯一邀请码失败，请重试", nil)
			return
		}
	}

	inviteCode := models.InviteCode{
		Code:           code,
		UserID:         user.ID,
		RewardType:     req.RewardType,
		InviterReward:  req.InviterReward,
		InviteeReward:  req.InviteeReward,
		MinOrderAmount: req.MinOrderAmount,
		NewUserOnly:    req.NewUserOnly,
		IsActive:       true,
	}

	if req.MaxUses > 0 {
		inviteCode.MaxUses = database.NullInt64(int64(req.MaxUses))
	}
	if !req.ExpiresAt.IsZero() {
		inviteCode.ExpiresAt = database.NullTime(req.ExpiresAt)
	}

	if err := db.Create(&inviteCode).Error; err != nil {
		utils.LogError("CreateInviteCode: create invite code failed", err, nil)
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建邀请码失败", err)
		return
	}

	// 生成邀请链接
	baseURL := utils.GetBuildBaseURL(c.Request, database.GetDB())
	inviteLink := baseURL + "/register?invite=" + code

	utils.SuccessResponse(c, http.StatusCreated, "邀请码生成成功", gin.H{
		"id":             inviteCode.ID,
		"code":           inviteCode.Code,
		"invite_link":    inviteLink,
		"max_uses":       inviteCode.MaxUses,
		"used_count":     inviteCode.UsedCount,
		"expires_at":     inviteCode.ExpiresAt,
		"reward_type":    inviteCode.RewardType,
		"inviter_reward": inviteCode.InviterReward,
		"invitee_reward": inviteCode.InviteeReward,
		"is_active":      inviteCode.IsActive,
		"created_at":     inviteCode.CreatedAt,
	})
}

// GetInviteCodes 获取邀请码列表
func GetInviteCodes(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	db := database.GetDB()
	var inviteCodes []models.InviteCode
	if err := db.Where("user_id = ?", user.ID).Preload("InviteRelations").Find(&inviteCodes).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取邀请码列表失败", err)
		return
	}

	// 生成邀请链接
	baseURL := utils.GetBuildBaseURL(c.Request, database.GetDB())
	now := utils.GetBeijingTime()
	var result []gin.H
	for _, code := range inviteCodes {
		inviteLink := baseURL + "/register?invite=" + code.Code

		// 处理 max_uses（sql.NullInt64 -> int 或 null）
		var maxUses interface{} = nil
		if code.MaxUses.Valid {
			maxUses = int(code.MaxUses.Int64)
		}

		// 处理 expires_at（sql.NullTime -> string 或 null）
		var expiresAt interface{} = nil
		if code.ExpiresAt.Valid {
			expiresAt = code.ExpiresAt.Time.Format("2006-01-02 15:04:05")
		}

		// 计算是否有效
		isValid := code.IsActive
		if isValid && code.ExpiresAt.Valid {
			isValid = code.ExpiresAt.Time.After(now)
		}
		if isValid && code.MaxUses.Valid {
			isValid = code.UsedCount < int(code.MaxUses.Int64)
		}

		result = append(result, gin.H{
			"id":             code.ID,
			"code":           code.Code,
			"invite_link":    inviteLink,
			"max_uses":       maxUses,
			"used_count":     code.UsedCount,
			"expires_at":     expiresAt,
			"reward_type":    code.RewardType,
			"inviter_reward": code.InviterReward,
			"invitee_reward": code.InviteeReward,
			"is_active":      code.IsActive,
			"is_valid":       isValid,
			"created_at":     code.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// GetInviteStats 获取邀请统计
func GetInviteStats(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	db := database.GetDB()

	var stats struct {
		TotalInviteCount     int     `json:"total_invite_count"`
		TotalInviteReward    float64 `json:"total_invite_reward"`
		ActiveInviteCodes    int64   `json:"active_invite_codes"`
		TotalInviteRelations int64   `json:"total_invite_relations"`
	}

	// 从用户表获取
	var u models.User
	db.First(&u, user.ID)
	stats.TotalInviteCount = u.TotalInviteCount
	stats.TotalInviteReward = u.TotalInviteReward

	// 统计活跃邀请码
	db.Model(&models.InviteCode{}).Where("user_id = ? AND is_active = ?", user.ID, true).Count(&stats.ActiveInviteCodes)

	// 统计邀请关系
	db.Model(&models.InviteRelation{}).Where("inviter_id = ?", user.ID).Count(&stats.TotalInviteRelations)

	utils.SuccessResponse(c, http.StatusOK, "", stats)
}

// GetRewardSettings 获取邀请奖励设置
func GetRewardSettings(c *gin.Context) {
	_, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	db := database.GetDB()

	// 从系统配置中获取邀请奖励设置
	var configs []models.SystemConfig
	db.Where("category = ? AND `key` IN (?)", "invite", []string{
		"inviter_reward", "invitee_reward", "min_order_amount", "new_user_only",
	}).Find(&configs)

	settings := make(map[string]interface{})
	for _, cfg := range configs {
		// 将数值类型的配置转换为数字
		if cfg.Key == "inviter_reward" || cfg.Key == "invitee_reward" || cfg.Key == "min_order_amount" {
			if val, err := strconv.ParseFloat(cfg.Value, 64); err == nil {
				settings[cfg.Key] = val
			} else {
				settings[cfg.Key] = 0.0
			}
		} else if cfg.Key == "new_user_only" {
			settings[cfg.Key] = cfg.Value == "true"
		} else {
			settings[cfg.Key] = cfg.Value
		}
	}

	// 如果没有配置，返回默认值（数字类型）
	if _, ok := settings["inviter_reward"]; !ok {
		settings["inviter_reward"] = 0.0
	}
	if _, ok := settings["invitee_reward"]; !ok {
		settings["invitee_reward"] = 0.0
	}
	if _, ok := settings["min_order_amount"]; !ok {
		settings["min_order_amount"] = 0.0
	}
	if _, ok := settings["new_user_only"]; !ok {
		settings["new_user_only"] = false
	}

	utils.SuccessResponse(c, http.StatusOK, "", settings)
}

// GetMyInviteCodes 获取我的邀请码列表（别名，实际使用 GetInviteCodes）
func GetMyInviteCodes(c *gin.Context) {
	// 直接调用 GetInviteCodes
	GetInviteCodes(c)
}

// ValidateInviteCode 验证邀请码（公开访问，用于注册）
func ValidateInviteCode(c *gin.Context) {
	code := strings.ToUpper(strings.TrimSpace(c.Param("code")))
	db := database.GetDB()

	var inviteCode models.InviteCode
	// 使用 UPPER() 函数进行大小写不敏感查询
	if err := db.Where("UPPER(code) = ?", code).First(&inviteCode).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "邀请码不存在", err)
		return
	}

	// 检查邀请码是否有效
	now := utils.GetBeijingTime()
	if !inviteCode.IsActive {
		utils.ErrorResponse(c, http.StatusBadRequest, "邀请码已停用", nil)
		return
	}

	if inviteCode.ExpiresAt.Valid && inviteCode.ExpiresAt.Time.Before(now) {
		utils.ErrorResponse(c, http.StatusBadRequest, "邀请码已过期", nil)
		return
	}

	if inviteCode.MaxUses.Valid && inviteCode.UsedCount >= int(inviteCode.MaxUses.Int64) {
		utils.ErrorResponse(c, http.StatusBadRequest, "邀请码使用次数已达上限", nil)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"code":             inviteCode.Code,
		"is_valid":         true,
		"expires_at":       inviteCode.ExpiresAt,
		"max_uses":         inviteCode.MaxUses,
		"used_count":       inviteCode.UsedCount,
		"invitee_reward":   inviteCode.InviteeReward,
		"inviter_reward":   inviteCode.InviterReward,
		"reward_type":      inviteCode.RewardType,
		"min_order_amount": inviteCode.MinOrderAmount,
		"new_user_only":    inviteCode.NewUserOnly,
	})
}

// UpdateInviteCode 更新邀请码
func UpdateInviteCode(c *gin.Context) {
	id := c.Param("id")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	db := database.GetDB()
	var inviteCode models.InviteCode
	if err := db.Where("id = ? AND created_by = ?", id, user.ID).First(&inviteCode).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "邀请码不存在或无权限", err)
		return
	}

	var req struct {
		IsActive  *bool      `json:"is_active"`
		ExpiresAt *time.Time `json:"expires_at"`
		MaxUses   *int       `json:"max_uses"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	if req.IsActive != nil {
		inviteCode.IsActive = *req.IsActive
	}
	if req.ExpiresAt != nil {
		inviteCode.ExpiresAt = database.NullTime(*req.ExpiresAt)
	}
	if req.MaxUses != nil {
		inviteCode.MaxUses = database.NullInt64(int64(*req.MaxUses))
	}

	if err := db.Save(&inviteCode).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新邀请码失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "更新成功", inviteCode)
}

// DeleteInviteCode 删除邀请码
func DeleteInviteCode(c *gin.Context) {
	id := c.Param("id")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	db := database.GetDB()
	var inviteCode models.InviteCode
	// 修复：使用 UserID 而不是 created_by
	if err := db.Where("id = ? AND user_id = ?", id, user.ID).First(&inviteCode).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "邀请码不存在或无权限", err)
		} else {
			utils.LogError("DeleteInviteCode: query invite code failed", err, nil)
			utils.ErrorResponse(c, http.StatusInternalServerError, "查询邀请码失败", err)
		}
		return
	}

	// 检查邀请码是否已被使用
	if inviteCode.UsedCount > 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "邀请码已被使用，无法删除", nil)
		return
	}

	// 删除邀请码
	if err := db.Delete(&inviteCode).Error; err != nil {
		utils.LogError("DeleteInviteCode: delete invite code failed", err, nil)
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除邀请码失败", err)
		return
	}

	// 记录审计日志
	utils.CreateAuditLogSimple(c, "delete_invite_code", "invite_code", inviteCode.ID, fmt.Sprintf("删除邀请码: %s", inviteCode.Code))

	utils.SuccessResponse(c, http.StatusOK, "删除成功", nil)
}
