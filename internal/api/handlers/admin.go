package handlers

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetAdminInvites(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.InviteCode{}).Preload("User").Preload("InviteRelations")
	page, size, offset := getPagination(c)

	if userQuery := utils.SanitizeSearchKeyword(c.Query("user_query")); userQuery != "" {
		query = query.Where("user_id IN (SELECT id FROM users WHERE username LIKE ? OR email LIKE ?)", "%"+userQuery+"%", "%"+userQuery+"%")
	}
	if code := utils.SanitizeSearchKeyword(c.Query("code")); code != "" {
		query = query.Where("code LIKE ?", "%"+code+"%")
	}
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		if isActiveStr == "true" || isActiveStr == "1" {
			query = query.Where("is_active = ?", true)
		} else if isActiveStr == "false" || isActiveStr == "0" {
			query = query.Where("is_active = ?", false)
		}
	}

	var total int64
	query.Count(&total)
	var inviteCodes []models.InviteCode
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&inviteCodes).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取邀请码列表失败", err)
		return
	}

	var result []gin.H
	for _, code := range inviteCodes {
		var maxUses interface{}
		if code.MaxUses.Valid {
			maxUses = int(code.MaxUses.Int64)
		}
		var expiresAt interface{}
		if code.ExpiresAt.Valid {
			expiresAt = code.ExpiresAt.Time.Format("2006-01-02 15:04:05")
		}

		username, email := "", ""
		if code.User.ID != 0 {
			username, email = code.User.Username, code.User.Email
		} else if code.UserID != 0 {
			var user models.User
			if err := db.First(&user, code.UserID).Error; err == nil {
				username, email = user.Username, user.Email
			}
		}

		result = append(result, gin.H{
			"id":             code.ID,
			"code":           code.Code,
			"user_id":        code.UserID,
			"username":       username,
			"user_email":     email,
			"email":          email,
			"used_count":     code.UsedCount,
			"max_uses":       maxUses,
			"expires_at":     expiresAt,
			"reward_type":    code.RewardType,
			"inviter_reward": code.InviterReward,
			"invitee_reward": code.InviteeReward,
			"is_active":      code.IsActive,
			"created_at":     code.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{"invite_codes": result, "total": total, "page": page, "size": size})
}

func GetAdminInviteRelations(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.InviteRelation{}).Preload("Inviter").Preload("Invitee").Preload("InviteCode")
	page, size, offset := getPagination(c)

	if inviterQuery := c.Query("inviter_query"); inviterQuery != "" {
		query = query.Where("inviter_id IN (SELECT id FROM users WHERE username LIKE ? OR email LIKE ?)", "%"+inviterQuery+"%", "%"+inviterQuery+"%")
	}
	if inviteeQuery := c.Query("invitee_query"); inviteeQuery != "" {
		query = query.Where("invitee_id IN (SELECT id FROM users WHERE username LIKE ? OR email LIKE ?)", "%"+inviteeQuery+"%", "%"+inviteeQuery+"%")
	}

	var total int64
	query.Count(&total)
	var relations []models.InviteRelation
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&relations).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取邀请关系列表失败", err)
		return
	}

	var result []gin.H
	for _, relation := range relations {
		inviteCode := ""
		if relation.InviteCode.ID != 0 && relation.InviteCode.Code != "" {
			inviteCode = relation.InviteCode.Code
		} else if relation.InviteCodeID != 0 {
			var code models.InviteCode
			if err := db.First(&code, relation.InviteCodeID).Error; err == nil {
				inviteCode = code.Code
			}
		}

		inviterUsername, inviterEmail := "", ""
		if relation.Inviter.ID != 0 {
			inviterUsername, inviterEmail = relation.Inviter.Username, relation.Inviter.Email
		} else if relation.InviterID != 0 {
			var inviter models.User
			if err := db.First(&inviter, relation.InviterID).Error; err == nil {
				inviterUsername, inviterEmail = inviter.Username, inviter.Email
			}
		}

		inviteeUsername, inviteeEmail := "", ""
		if relation.Invitee.ID != 0 {
			inviteeUsername, inviteeEmail = relation.Invitee.Username, relation.Invitee.Email
		} else if relation.InviteeID != 0 {
			var invitee models.User
			if err := db.First(&invitee, relation.InviteeID).Error; err == nil {
				inviteeUsername, inviteeEmail = invitee.Username, invitee.Email
			}
		}

		result = append(result, gin.H{
			"id":                        relation.ID,
			"invite_code":               inviteCode,
			"inviter_id":                relation.InviterID,
			"inviter_username":          inviterUsername,
			"inviter_email":             inviterEmail,
			"invitee_id":                relation.InviteeID,
			"invitee_username":          inviteeUsername,
			"invitee_email":             inviteeEmail,
			"inviter_reward_amount":     relation.InviterRewardAmount,
			"inviter_reward_given":      relation.InviterRewardGiven,
			"invitee_reward_amount":     relation.InviteeRewardAmount,
			"invitee_reward_given":      relation.InviteeRewardGiven,
			"invitee_total_consumption": relation.InviteeTotalConsumption,
			"created_at":                relation.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{"relations": result, "total": total, "page": page, "size": size})
}

func GetAdminInviteStatistics(c *gin.Context) {
	db := database.GetDB()
	var stats struct {
		TotalInviteCodes     int64   `json:"total_invite_codes"`
		ActiveInviteCodes    int64   `json:"active_invite_codes"`
		TotalInviteRelations int64   `json:"total_invite_relations"`
		TotalInviteReward    float64 `json:"total_invite_reward"`
	}
	db.Model(&models.InviteCode{}).Count(&stats.TotalInviteCodes)
	db.Model(&models.InviteCode{}).Where("is_active = ?", true).Count(&stats.ActiveInviteCodes)
	db.Model(&models.InviteRelation{}).Count(&stats.TotalInviteRelations)
	var totalReward float64
	db.Model(&models.User{}).Select("COALESCE(SUM(total_invite_reward), 0)").Scan(&totalReward)
	stats.TotalInviteReward = totalReward
	utils.SuccessResponse(c, http.StatusOK, "", stats)
}

func GetAdminTickets(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.Ticket{}).Preload("User").Preload("Assignee")
	page, size, offset := getPagination(c)

	if keyword := utils.SanitizeSearchKeyword(c.Query("keyword")); keyword != "" {
		query = query.Where("ticket_no LIKE ? OR title LIKE ? OR content LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if ticketType := c.Query("type"); ticketType != "" {
		query = query.Where("type = ?", ticketType)
	}
	if priority := c.Query("priority"); priority != "" {
		query = query.Where("priority = ?", priority)
	}

	var total int64
	query.Count(&total)
	var tickets []models.Ticket
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&tickets).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取工单列表失败", err)
		return
	}

	adminUser, _ := middleware.GetCurrentUser(c)
	adminUserID := uint(0)
	if adminUser != nil {
		adminUserID = adminUser.ID
	}

	ticketList := make([]gin.H, 0)
	for _, ticket := range tickets {
		var repliesCount int64
		db.Model(&models.TicketReply{}).Where("ticket_id = ?", ticket.ID).Count(&repliesCount)
		var unreadRepliesCount int64 = 0
		if adminUserID > 0 {
			db.Model(&models.TicketReply{}).
				Where("ticket_id = ? AND is_admin != ? AND (is_read = ? OR read_by != ? OR read_by IS NULL)",
					ticket.ID, "true", false, adminUserID).
				Count(&unreadRepliesCount)
		}
		var ticketRead models.TicketRead
		err := db.Where("ticket_id = ? AND user_id = ?", ticket.ID, adminUserID).First(&ticketRead).Error
		hasNewTicket := errors.Is(err, gorm.ErrRecordNotFound)
		hasUnread := unreadRepliesCount > 0 || hasNewTicket

		ticketList = append(ticketList, gin.H{
			"id":             ticket.ID,
			"ticket_no":      ticket.TicketNo,
			"user_id":        ticket.UserID,
			"user":           ticket.User,
			"title":          ticket.Title,
			"content":        ticket.Content,
			"type":           ticket.Type,
			"status":         ticket.Status,
			"priority":       ticket.Priority,
			"assigned_to":    ticket.AssignedTo,
			"assignee":       ticket.Assignee,
			"admin_notes":    ticket.AdminNotes,
			"replies_count":  repliesCount,
			"unread_replies": unreadRepliesCount,
			"has_unread":     hasUnread,
			"has_new_ticket": hasNewTicket,
			"created_at":     ticket.CreatedAt.Format("2006-01-02 15:04:05"),
			"updated_at":     ticket.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{"tickets": ticketList, "total": total, "page": page, "size": size})
}

func GetAdminTicketStatistics(c *gin.Context) {
	db := database.GetDB()
	var stats struct {
		Total      int64 `json:"total"`
		Pending    int64 `json:"pending"`
		Processing int64 `json:"processing"`
		Resolved   int64 `json:"resolved"`
		Closed     int64 `json:"closed"`
	}
	db.Model(&models.Ticket{}).Count(&stats.Total)
	db.Model(&models.Ticket{}).Where("status = ?", "pending").Count(&stats.Pending)
	db.Model(&models.Ticket{}).Where("status = ?", "processing").Count(&stats.Processing)
	db.Model(&models.Ticket{}).Where("status = ?", "resolved").Count(&stats.Resolved)
	db.Model(&models.Ticket{}).Where("status = ?", "closed").Count(&stats.Closed)
	utils.SuccessResponse(c, http.StatusOK, "", stats)
}

func GetAdminTicket(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()
	var ticket models.Ticket
	if err := db.Preload("User").Preload("Assignee").
		Preload("Replies", func(db *gorm.DB) *gorm.DB { return db.Order("created_at ASC") }).
		Preload("Attachments").First(&ticket, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "工单不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取工单失败", err)
		}
		return
	}

	var repliesCount int64
	db.Model(&models.TicketReply{}).Where("ticket_id = ?", ticket.ID).Count(&repliesCount)
	adminUser, _ := middleware.GetCurrentUser(c)
	adminUserID := uint(0)
	if adminUser != nil {
		adminUserID = adminUser.ID
	}

	replies := make([]gin.H, 0)
	now := utils.GetBeijingTime()
	for _, reply := range ticket.Replies {
		replyData := gin.H{
			"id":         reply.ID,
			"ticket_id":  reply.TicketID,
			"user_id":    reply.UserID,
			"content":    reply.Content,
			"is_admin":   reply.IsAdmin,
			"created_at": reply.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if reply.IsAdmin != "true" {
			isUnread := !reply.IsRead || (reply.ReadBy != nil && *reply.ReadBy != adminUserID)
			replyData["is_unread"] = isUnread
			replyData["is_user_reply"] = true
			if isUnread && adminUserID > 0 {
				reply.IsRead = true
				reply.ReadBy = &adminUserID
				reply.ReadAt = &now
				db.Save(&reply)
			}
		} else {
			replyData["is_admin_reply"] = true
		}
		replies = append(replies, replyData)
	}

	if adminUserID > 0 {
		var ticketRead models.TicketRead
		err := db.Where("ticket_id = ? AND user_id = ?", ticket.ID, adminUserID).First(&ticketRead).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ticketRead = models.TicketRead{TicketID: ticket.ID, UserID: adminUserID, ReadAt: now}
			db.Create(&ticketRead)
		} else if err == nil {
			ticketRead.ReadAt = now
			db.Save(&ticketRead)
		}
	}

	attachments := make([]gin.H, 0)
	for _, attachment := range ticket.Attachments {
		att := gin.H{
			"id":          attachment.ID,
			"ticket_id":   attachment.TicketID,
			"file_name":   attachment.FileName,
			"file_path":   attachment.FilePath,
			"uploaded_by": attachment.UploadedBy,
			"created_at":  attachment.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if attachment.ReplyID != nil {
			att["reply_id"] = *attachment.ReplyID
		}
		if attachment.FileSize != nil {
			att["file_size"] = *attachment.FileSize
		}
		if attachment.FileType != nil {
			att["file_type"] = *attachment.FileType
		}
		attachments = append(attachments, att)
	}

	ticketData := gin.H{
		"id":            ticket.ID,
		"ticket_no":     ticket.TicketNo,
		"user_id":       ticket.UserID,
		"user":          ticket.User,
		"title":         ticket.Title,
		"content":       ticket.Content,
		"type":          ticket.Type,
		"status":        ticket.Status,
		"priority":      ticket.Priority,
		"assigned_to":   ticket.AssignedTo,
		"assignee":      ticket.Assignee,
		"admin_notes":   ticket.AdminNotes,
		"replies":       replies,
		"replies_count": repliesCount,
		"attachments":   attachments,
		"created_at":    ticket.CreatedAt.Format("2006-01-02 15:04:05"),
		"updated_at":    ticket.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if ticket.Rating != nil {
		ticketData["rating"] = *ticket.Rating
	}
	if ticket.RatingComment != nil {
		ticketData["rating_comment"] = *ticket.RatingComment
	}
	if ticket.ResolvedAt != nil {
		ticketData["resolved_at"] = ticket.ResolvedAt.Format("2006-01-02 15:04:05")
	}
	if ticket.ClosedAt != nil {
		ticketData["closed_at"] = ticket.ClosedAt.Format("2006-01-02 15:04:05")
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{"ticket": ticketData})
}

func GetAdminCoupons(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.Coupon{})
	page, size, offset := getPagination(c)

	if keyword := utils.SanitizeSearchKeyword(c.Query("keyword")); keyword != "" {
		query = query.Where("code LIKE ? OR name LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if status := c.Query("status"); status != "" {
		switch status {
		case "active":
			query = query.Where("status = ?", "active")
		case "inactive":
			query = query.Where("status = ?", "inactive")
		case "expired":
			query = query.Where("valid_until < ?", utils.GetBeijingTime())
		}
	}
	if couponType := c.Query("type"); couponType != "" {
		query = query.Where("type = ?", couponType)
	}

	var total int64
	query.Count(&total)
	var coupons []models.Coupon
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&coupons).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取优惠券列表失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{"coupons": coupons, "total": total, "page": page, "size": size})
}

func GetAdminUserLevels(c *gin.Context) {
	var userLevels []models.UserLevel
	if err := database.GetDB().Order("level_order ASC").Find(&userLevels).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取用户等级列表失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "", userLevels)
}

func CreateUserLevel(c *gin.Context) {
	var req struct {
		LevelName      string  `json:"level_name" binding:"required"`
		LevelOrder     int     `json:"level_order" binding:"required"`
		MinConsumption float64 `json:"min_consumption"`
		DiscountRate   float64 `json:"discount_rate"`
		Color          string  `json:"color"`
		IconURL        string  `json:"icon_url"`
		Benefits       string  `json:"benefits"`
		IsActive       bool    `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	var existing models.UserLevel
	if err := db.Where("level_name = ?", req.LevelName).First(&existing).Error; err == nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "等级名称已存在", nil)
		return
	}
	if err := db.Where("level_order = ?", req.LevelOrder).First(&existing).Error; err == nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "等级顺序已存在", nil)
		return
	}

	userLevel := models.UserLevel{
		LevelName:      req.LevelName,
		LevelOrder:     req.LevelOrder,
		MinConsumption: req.MinConsumption,
		DiscountRate:   req.DiscountRate,
		Color:          req.Color,
		IsActive:       req.IsActive,
	}
	if req.IconURL != "" {
		userLevel.IconURL = database.NullString(req.IconURL)
	}
	if req.Benefits != "" {
		userLevel.Benefits = database.NullString(req.Benefits)
	}

	if err := db.Create(&userLevel).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建用户等级失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusCreated, "", userLevel)
}

func UpdateUserLevel(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()
	var userLevel models.UserLevel
	if err := db.First(&userLevel, id).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "用户等级不存在", err)
		return
	}

	var req struct {
		LevelName      string  `json:"level_name"`
		LevelOrder     int     `json:"level_order"`
		MinConsumption float64 `json:"min_consumption"`
		DiscountRate   float64 `json:"discount_rate"`
		Color          string  `json:"color"`
		IconURL        *string `json:"icon_url"`
		Benefits       *string `json:"benefits"`
		IsActive       *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	if req.LevelName != "" && req.LevelName != userLevel.LevelName {
		var existing models.UserLevel
		if err := db.Where("level_name = ? AND id != ?", req.LevelName, id).First(&existing).Error; err == nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "等级名称已存在", nil)
			return
		}
		userLevel.LevelName = req.LevelName
	}
	if req.LevelOrder > 0 && req.LevelOrder != userLevel.LevelOrder {
		var existing models.UserLevel
		if err := db.Where("level_order = ? AND id != ?", req.LevelOrder, id).First(&existing).Error; err == nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "等级顺序已存在", nil)
			return
		}
		userLevel.LevelOrder = req.LevelOrder
	}
	if req.MinConsumption >= 0 {
		userLevel.MinConsumption = req.MinConsumption
	}
	if req.DiscountRate > 0 {
		userLevel.DiscountRate = req.DiscountRate
	}
	if req.Color != "" {
		userLevel.Color = req.Color
	}
	if req.IconURL != nil {
		userLevel.IconURL = database.NullString(*req.IconURL)
	}
	if req.Benefits != nil {
		userLevel.Benefits = database.NullString(*req.Benefits)
	}
	if req.IsActive != nil {
		userLevel.IsActive = *req.IsActive
	}

	if err := db.Save(&userLevel).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新用户等级失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "更新成功", userLevel)
}

func GetUserLevel(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}
	var userLevel models.UserLevel
	if user.UserLevelID.Valid {
		database.GetDB().First(&userLevel, user.UserLevelID.Int64)
	}
	utils.SuccessResponse(c, http.StatusOK, "", userLevel)
}

func GetUserSubscription(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}
	db := database.GetDB()
	var subscription models.Subscription
	if err := db.Where("user_id = ?", user.ID).First(&subscription).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.SuccessResponse(c, http.StatusOK, "", nil)
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅失败", err)
		return
	}

	baseURL := utils.GetBuildBaseURL(c.Request, database.GetDB())
	clashURL := fmt.Sprintf("%s/api/v1/subscriptions/clash/%s", baseURL, subscription.SubscriptionURL)
	universalURL := fmt.Sprintf("%s/api/v1/subscriptions/universal/%s", baseURL, subscription.SubscriptionURL)
	expiryDate := "未设置"
	if !subscription.ExpireTime.IsZero() {
		expiryDate = subscription.ExpireTime.Format("2006-01-02 15:04:05")
	}
	encodedURL := base64.StdEncoding.EncodeToString([]byte(universalURL))
	expiryDisplay := expiryDate
	if expiryDisplay == "未设置" {
		expiryDisplay = subscription.SubscriptionURL
	}
	qrcodeURL := fmt.Sprintf("sub://%s#%s", encodedURL, url.QueryEscape(expiryDisplay))

	remainingDays := 0
	isExpired := false
	if !subscription.ExpireTime.IsZero() {
		diff := utils.ToBeijingTime(subscription.ExpireTime).Sub(utils.GetBeijingTime())
		if diff > 0 {
			remainingDays = int(math.Ceil(diff.Hours() / 24.0))
		} else {
			isExpired = true
		}
	}
	var onlineDevices int64
	db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subscription.ID, true).Count(&onlineDevices)

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"id":               subscription.ID,
		"subscription_url": subscription.SubscriptionURL,
		"clash_url":        clashURL,
		"universal_url":    universalURL,
		"qrcode_url":       qrcodeURL,
		"device_limit":     subscription.DeviceLimit,
		"current_devices":  onlineDevices,
		"status":           subscription.Status,
		"is_active":        subscription.IsActive,
		"expire_time":      expiryDate,
		"expiryDate":       expiryDate,
		"remaining_days":   remainingDays,
		"is_expired":       isExpired,
		"created_at":       subscription.CreatedAt.Format("2006-01-02 15:04:05"),
	})
}

func GetUserTheme(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{"theme": user.Theme, "language": user.Language})
}

func UpdateUserTheme(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}
	var req struct {
		Theme    string `json:"theme"`
		Language string `json:"language"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}
	if req.Theme != "" {
		user.Theme = req.Theme
	}
	if req.Language != "" {
		user.Language = req.Language
	}
	if err := database.GetDB().Save(&user).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新主题失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "主题更新成功", gin.H{"theme": user.Theme, "language": user.Language})
}

func GetAdminEmailQueue(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.EmailQueue{})
	page, size, offset := getPagination(c)

	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if email := strings.TrimSpace(c.Query("email")); email != "" {
		query = query.Where("to_email LIKE ?", "%"+email+"%")
	}

	var total int64
	query.Count(&total)
	var emails []models.EmailQueue
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&emails).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取邮件队列失败", err)
		return
	}
	pages := (total + int64(size) - 1) / int64(size)
	if pages < 1 {
		pages = 1
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{"emails": emails, "total": total, "page": page, "size": size, "pages": pages})
}

func GetEmailQueueStatistics(c *gin.Context) {
	db := database.GetDB()
	var stats struct {
		Total         int64 `json:"total"`
		Pending       int64 `json:"pending"`
		Sent          int64 `json:"sent"`
		Failed        int64 `json:"failed"`
		TotalEmails   int64 `json:"total_emails"`
		PendingEmails int64 `json:"pending_emails"`
		SentEmails    int64 `json:"sent_emails"`
		FailedEmails  int64 `json:"failed_emails"`
	}
	db.Model(&models.EmailQueue{}).Count(&stats.TotalEmails)
	db.Model(&models.EmailQueue{}).Where("status = ?", "pending").Count(&stats.PendingEmails)
	db.Model(&models.EmailQueue{}).Where("status = ?", "sent").Count(&stats.SentEmails)
	db.Model(&models.EmailQueue{}).Where("status = ?", "failed").Count(&stats.FailedEmails)
	stats.Total = stats.TotalEmails
	stats.Pending = stats.PendingEmails
	stats.Sent = stats.SentEmails
	stats.Failed = stats.FailedEmails
	utils.SuccessResponse(c, http.StatusOK, "", stats)
}

func GetEmailQueueDetail(c *gin.Context) {
	id := c.Param("id")
	var email models.EmailQueue
	if err := database.GetDB().First(&email, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "邮件不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取邮件详情失败", err)
		}
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "", email)
}

func DeleteEmailFromQueue(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()
	var email models.EmailQueue
	if err := db.First(&email, id).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "邮件不存在", err)
		return
	}
	if err := db.Delete(&email).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除邮件失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "邮件删除成功", nil)
}

func RetryEmailFromQueue(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()
	var email models.EmailQueue
	if err := db.First(&email, id).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "邮件不存在", err)
		return
	}
	email.Status = "pending"
	email.RetryCount = 0
	email.ErrorMessage = sql.NullString{Valid: false}
	if err := db.Save(&email).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "重试邮件失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "邮件已加入重试队列", nil)
}

func ClearEmailQueue(c *gin.Context) {
	status := c.Query("status")
	db := database.GetDB()
	var result *gorm.DB
	if status != "" {
		result = db.Where("status = ?", status).Delete(&models.EmailQueue{})
	} else {
		result = db.Where("1 = 1").Delete(&models.EmailQueue{})
	}
	if result.Error != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "清空邮件队列失败", result.Error)
		return
	}
	message := "邮件队列已清空"
	if status != "" {
		message = fmt.Sprintf("已清空 %s 状态的邮件", status)
	}
	utils.SuccessResponse(c, http.StatusOK, message, gin.H{"deleted_count": result.RowsAffected})
}

func GetAdminSystemConfig(c *gin.Context) {
	var configs []models.SystemConfig
	database.GetDB().Where("category = ?", "system").Order("sort_order ASC").Find(&configs)
	configMap := make(map[string]interface{})
	for _, config := range configs {
		if config.Value == "true" || config.Value == "false" {
			configMap[config.Key] = config.Value == "true"
		} else {
			configMap[config.Key] = config.Value
		}
	}
	if len(configMap) == 0 {
		configMap = map[string]interface{}{"site_name": "", "site_description": "", "logo_url": "", "maintenance_mode": false, "maintenance_message": ""}
	}
	utils.SuccessResponse(c, http.StatusOK, "", configMap)
}

func GetAdminEmailConfig(c *gin.Context) {
	var configs []models.SystemConfig
	database.GetDB().Where("category = ?", "email").Find(&configs)
	configMap := make(map[string]interface{})
	for _, config := range configs {
		configMap[config.Key] = config.Value
	}
	utils.SuccessResponse(c, http.StatusOK, "", configMap)
}

func GetSoftwareConfig(c *gin.Context) {
	var configs []models.SystemConfig
	database.GetDB().Where("category = ?", "software").Find(&configs)
	configMap := make(map[string]interface{})
	for _, config := range configs {
		configMap[config.Key] = config.Value
	}
	utils.SuccessResponse(c, http.StatusOK, "", configMap)
}

func GetPaymentConfig(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.PaymentConfig{})

	page, size := 1, 100
	if pageStr := c.Query("page"); pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
	}
	if sizeStr := c.Query("size"); sizeStr != "" {
		fmt.Sscanf(sizeStr, "%d", &size)
	}
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 100
	}
	offset := (page - 1) * size

	var total int64
	query.Count(&total)
	var paymentConfigs []models.PaymentConfig
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&paymentConfigs).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取支付配置列表失败", err)
		return
	}

	type PaymentConfigResponse struct {
		ID                   uint                   `json:"id"`
		PayType              string                 `json:"pay_type"`
		AppID                string                 `json:"app_id,omitempty"`
		MerchantPrivateKey   string                 `json:"merchant_private_key,omitempty"`
		AlipayPublicKey      string                 `json:"alipay_public_key,omitempty"`
		WechatAppID          string                 `json:"wechat_app_id,omitempty"`
		WechatMchID          string                 `json:"wechat_mch_id,omitempty"`
		WechatAPIKey         string                 `json:"wechat_api_key,omitempty"`
		PaypalClientID       string                 `json:"paypal_client_id,omitempty"`
		PaypalSecret         string                 `json:"paypal_secret,omitempty"`
		StripePublishableKey string                 `json:"stripe_publishable_key,omitempty"`
		StripeSecretKey      string                 `json:"stripe_secret_key,omitempty"`
		BankName             string                 `json:"bank_name,omitempty"`
		AccountName          string                 `json:"account_name,omitempty"`
		AccountNumber        string                 `json:"account_number,omitempty"`
		WalletAddress        string                 `json:"wallet_address,omitempty"`
		Status               int                    `json:"status"`
		ReturnURL            string                 `json:"return_url,omitempty"`
		NotifyURL            string                 `json:"notify_url,omitempty"`
		SortOrder            int                    `json:"sort_order"`
		ConfigJSON           map[string]interface{} `json:"config_json,omitempty"`
		CreatedAt            string                 `json:"created_at"`
		UpdatedAt            string                 `json:"updated_at"`
	}

	configsResponse := make([]PaymentConfigResponse, len(paymentConfigs))
	for i, config := range paymentConfigs {
		configsResponse[i] = PaymentConfigResponse{
			ID:                   config.ID,
			PayType:              config.PayType,
			AppID:                safeNullString(config.AppID),
			MerchantPrivateKey:   safeNullString(config.MerchantPrivateKey),
			AlipayPublicKey:      safeNullString(config.AlipayPublicKey),
			WechatAppID:          safeNullString(config.WechatAppID),
			WechatMchID:          safeNullString(config.WechatMchID),
			WechatAPIKey:         safeNullString(config.WechatAPIKey),
			PaypalClientID:       safeNullString(config.PaypalClientID),
			PaypalSecret:         safeNullString(config.PaypalSecret),
			StripePublishableKey: safeNullString(config.StripePublishableKey),
			StripeSecretKey:      safeNullString(config.StripeSecretKey),
			BankName:             safeNullString(config.BankName),
			AccountName:          safeNullString(config.AccountName),
			AccountNumber:        safeNullString(config.AccountNumber),
			WalletAddress:        safeNullString(config.WalletAddress),
			Status:               config.Status,
			ReturnURL:            safeNullString(config.ReturnURL),
			NotifyURL:            safeNullString(config.NotifyURL),
			SortOrder:            config.SortOrder,
			CreatedAt:            config.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:            config.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
		if config.ConfigJSON.Valid {
			var jsonData map[string]interface{}
			if err := json.Unmarshal([]byte(config.ConfigJSON.String), &jsonData); err == nil {
				configsResponse[i].ConfigJSON = jsonData
			}
		}
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{"items": configsResponse, "total": total, "page": page, "size": size})
}

func GetUserTrend(c *gin.Context) {
	db := database.GetDB()
	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		fmt.Sscanf(daysStr, "%d", &days)
	}
	type UserTrend struct {
		Date      string `json:"date"`
		UserCount int64  `json:"user_count"`
	}
	var trends []UserTrend
	rows, err := db.Raw(`SELECT DATE(created_at) as date, COUNT(*) as user_count FROM users WHERE created_at >= DATE('now', '-' || ? || ' days') GROUP BY DATE(created_at) ORDER BY date ASC`, days).Rows()
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var trend UserTrend
			rows.Scan(&trend.Date, &trend.UserCount)
			trends = append(trends, trend)
		}
	}
	labels := make([]string, 0)
	data := make([]int64, 0)
	for _, trend := range trends {
		labels = append(labels, trend.Date)
		data = append(data, trend.UserCount)
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{"labels": labels, "data": data})
}

func GetRevenueTrend(c *gin.Context) {
	GetRevenueChart(c)
}

func UpdateClashConfig(c *gin.Context) {
	updateSingleConfig(c, "clash", "config", "Clash 配置")
}

func UpdateV2RayConfig(c *gin.Context) {
	updateSingleConfig(c, "v2ray", "config", "V2Ray 配置")
}

func UpdateEmailConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}
	db := database.GetDB()
	for key, value := range req {
		if err := upsertSystemConfig(db, "email", key, fmt.Sprintf("%v", value)); err != nil {
			utils.LogError("UpdateEmailConfig", err, map[string]interface{}{"key": key})
			utils.ErrorResponse(c, http.StatusInternalServerError, "更新配置失败，请稍后重试", err)
			return
		}
	}
	utils.SuccessResponse(c, http.StatusOK, "邮件配置已更新", nil)
}

func MarkClashConfigInvalid(c *gin.Context) {
	updateSingleConfig(c, "clash", "config_invalid", "Clash 失效配置")
}

func MarkV2RayConfigInvalid(c *gin.Context) {
	updateSingleConfig(c, "v2ray", "config_invalid", "V2Ray 失效配置")
}

func CreatePaymentConfig(c *gin.Context) {
	var req struct {
		PayType              string                 `json:"pay_type" binding:"required"`
		AppID                string                 `json:"app_id"`
		MerchantPrivateKey   string                 `json:"merchant_private_key"`
		AlipayPublicKey      string                 `json:"alipay_public_key"`
		WechatAppID          string                 `json:"wechat_app_id"`
		WechatMchID          string                 `json:"wechat_mch_id"`
		WechatAPIKey         string                 `json:"wechat_api_key"`
		PaypalClientID       string                 `json:"paypal_client_id"`
		PaypalSecret         string                 `json:"paypal_secret"`
		StripePublishableKey string                 `json:"stripe_publishable_key"`
		StripeSecretKey      string                 `json:"stripe_secret_key"`
		BankName             string                 `json:"bank_name"`
		AccountName          string                 `json:"account_name"`
		AccountNumber        string                 `json:"account_number"`
		WalletAddress        string                 `json:"wallet_address"`
		Status               int                    `json:"status"`
		ReturnURL            string                 `json:"return_url"`
		NotifyURL            string                 `json:"notify_url"`
		SortOrder            int                    `json:"sort_order"`
		ConfigJSON           map[string]interface{} `json:"config_json"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	baseURL := utils.GetBuildBaseURL(c.Request, database.GetDB())
	if req.NotifyURL == "" {
		notifySuffix := "alipay"
		if req.PayType == "wechat" {
			notifySuffix = "wechat"
		} else if req.PayType == "yipay" || strings.HasPrefix(req.PayType, "yipay_") {
			notifySuffix = "yipay"
		}
		req.NotifyURL = fmt.Sprintf("%s/api/v1/payment/notify/%s", baseURL, notifySuffix)
	}
	if req.ReturnURL == "" {
		req.ReturnURL = fmt.Sprintf("%s/payment/return", baseURL)
	}
	if req.Status == 0 {
		req.Status = 1
	}

	var configJSONStr sql.NullString
	if req.ConfigJSON != nil {
		bytes, _ := json.Marshal(req.ConfigJSON)
		configJSONStr = sql.NullString{String: string(bytes), Valid: true}
	}

	paymentConfig := models.PaymentConfig{
		PayType:              req.PayType,
		AppID:                database.NullString(req.AppID),
		MerchantPrivateKey:   database.NullString(req.MerchantPrivateKey),
		AlipayPublicKey:      database.NullString(req.AlipayPublicKey),
		WechatAppID:          database.NullString(req.WechatAppID),
		WechatMchID:          database.NullString(req.WechatMchID),
		WechatAPIKey:         database.NullString(req.WechatAPIKey),
		PaypalClientID:       database.NullString(req.PaypalClientID),
		PaypalSecret:         database.NullString(req.PaypalSecret),
		StripePublishableKey: database.NullString(req.StripePublishableKey),
		StripeSecretKey:      database.NullString(req.StripeSecretKey),
		BankName:             database.NullString(req.BankName),
		AccountName:          database.NullString(req.AccountName),
		AccountNumber:        database.NullString(req.AccountNumber),
		WalletAddress:        database.NullString(req.WalletAddress),
		Status:               req.Status,
		ReturnURL:            database.NullString(req.ReturnURL),
		NotifyURL:            database.NullString(req.NotifyURL),
		SortOrder:            req.SortOrder,
		ConfigJSON:           configJSONStr,
	}

	if err := database.GetDB().Create(&paymentConfig).Error; err != nil {
		utils.LogError("CreatePaymentConfig", err, nil)
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建支付配置失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusCreated, "支付配置创建成功", paymentConfig)
}

func UpdatePaymentConfig(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		PayType              string                 `json:"pay_type"`
		AppID                *string                `json:"app_id"`
		MerchantPrivateKey   *string                `json:"merchant_private_key"`
		AlipayPublicKey      *string                `json:"alipay_public_key"`
		WechatAppID          *string                `json:"wechat_app_id"`
		WechatMchID          *string                `json:"wechat_mch_id"`
		WechatAPIKey         *string                `json:"wechat_api_key"`
		PaypalClientID       *string                `json:"paypal_client_id"`
		PaypalSecret         *string                `json:"paypal_secret"`
		StripePublishableKey *string                `json:"stripe_publishable_key"`
		StripeSecretKey      *string                `json:"stripe_secret_key"`
		BankName             *string                `json:"bank_name"`
		AccountName          *string                `json:"account_name"`
		AccountNumber        *string                `json:"account_number"`
		WalletAddress        *string                `json:"wallet_address"`
		Status               int                    `json:"status"`
		ReturnURL            *string                `json:"return_url"`
		NotifyURL            *string                `json:"notify_url"`
		SortOrder            *int                   `json:"sort_order"`
		ConfigJSON           map[string]interface{} `json:"config_json"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	var paymentConfig models.PaymentConfig
	if err := db.First(&paymentConfig, id).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "支付配置不存在", err)
		return
	}

	baseURL := utils.GetBuildBaseURL(c.Request, database.GetDB())
	if (req.PayType == "yipay" || strings.HasPrefix(req.PayType, "yipay_")) && req.NotifyURL == nil {
		notifyURL := fmt.Sprintf("%s/api/v1/payment/notify/yipay", baseURL)
		req.NotifyURL = &notifyURL
		utils.LogInfo("易支付回调地址自动生成: %s", notifyURL)
	}

	if req.PayType != "" {
		paymentConfig.PayType = req.PayType
	}
	if req.AppID != nil {
		paymentConfig.AppID = ptrToNullString(req.AppID)
	}
	if req.MerchantPrivateKey != nil {
		paymentConfig.MerchantPrivateKey = ptrToNullString(req.MerchantPrivateKey)
	}
	if req.AlipayPublicKey != nil {
		paymentConfig.AlipayPublicKey = ptrToNullString(req.AlipayPublicKey)
	}
	if req.WechatAppID != nil {
		paymentConfig.WechatAppID = ptrToNullString(req.WechatAppID)
	}
	if req.WechatMchID != nil {
		paymentConfig.WechatMchID = ptrToNullString(req.WechatMchID)
	}
	if req.WechatAPIKey != nil {
		paymentConfig.WechatAPIKey = ptrToNullString(req.WechatAPIKey)
	}
	if req.PaypalClientID != nil {
		paymentConfig.PaypalClientID = ptrToNullString(req.PaypalClientID)
	}
	if req.PaypalSecret != nil {
		paymentConfig.PaypalSecret = ptrToNullString(req.PaypalSecret)
	}
	if req.StripePublishableKey != nil {
		paymentConfig.StripePublishableKey = ptrToNullString(req.StripePublishableKey)
	}
	if req.StripeSecretKey != nil {
		paymentConfig.StripeSecretKey = ptrToNullString(req.StripeSecretKey)
	}
	if req.BankName != nil {
		paymentConfig.BankName = ptrToNullString(req.BankName)
	}
	if req.AccountName != nil {
		paymentConfig.AccountName = ptrToNullString(req.AccountName)
	}
	if req.AccountNumber != nil {
		paymentConfig.AccountNumber = ptrToNullString(req.AccountNumber)
	}
	if req.WalletAddress != nil {
		paymentConfig.WalletAddress = ptrToNullString(req.WalletAddress)
	}
	if req.Status >= 0 {
		paymentConfig.Status = req.Status
	}
	if req.ReturnURL != nil {
		paymentConfig.ReturnURL = ptrToNullString(req.ReturnURL)
	}
	if req.NotifyURL != nil {
		paymentConfig.NotifyURL = ptrToNullString(req.NotifyURL)
	} else if req.PayType != "" && paymentConfig.NotifyURL.String == "" {
		notifySuffix := "alipay"
		if req.PayType == "wechat" {
			notifySuffix = "wechat"
		} else if req.PayType == "yipay" || strings.HasPrefix(req.PayType, "yipay_") {
			notifySuffix = "yipay"
		}
		paymentConfig.NotifyURL = database.NullString(fmt.Sprintf("%s/api/v1/payment/notify/%s", baseURL, notifySuffix))
	}
	if req.SortOrder != nil {
		paymentConfig.SortOrder = *req.SortOrder
	}
	if req.ConfigJSON != nil {
		bytes, err := json.Marshal(req.ConfigJSON)
		if err == nil {
			paymentConfig.ConfigJSON = sql.NullString{String: string(bytes), Valid: true}
		}
	}

	if err := db.Save(&paymentConfig).Error; err != nil {
		utils.LogError("UpdatePaymentConfig", err, map[string]interface{}{"id": id})
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新支付配置失败", err)
		return
	}

	responseData := gin.H{
		"id":                     paymentConfig.ID,
		"pay_type":               paymentConfig.PayType,
		"app_id":                 safeNullString(paymentConfig.AppID),
		"merchant_private_key":   safeNullString(paymentConfig.MerchantPrivateKey),
		"alipay_public_key":      safeNullString(paymentConfig.AlipayPublicKey),
		"wechat_app_id":          safeNullString(paymentConfig.WechatAppID),
		"wechat_mch_id":          safeNullString(paymentConfig.WechatMchID),
		"wechat_api_key":         safeNullString(paymentConfig.WechatAPIKey),
		"paypal_client_id":       safeNullString(paymentConfig.PaypalClientID),
		"paypal_secret":          safeNullString(paymentConfig.PaypalSecret),
		"stripe_publishable_key": safeNullString(paymentConfig.StripePublishableKey),
		"stripe_secret_key":      safeNullString(paymentConfig.StripeSecretKey),
		"bank_name":              safeNullString(paymentConfig.BankName),
		"account_name":           safeNullString(paymentConfig.AccountName),
		"account_number":         safeNullString(paymentConfig.AccountNumber),
		"wallet_address":         safeNullString(paymentConfig.WalletAddress),
		"status":                 paymentConfig.Status,
		"return_url":             safeNullString(paymentConfig.ReturnURL),
		"notify_url":             safeNullString(paymentConfig.NotifyURL),
		"sort_order":             paymentConfig.SortOrder,
		"created_at":             paymentConfig.CreatedAt.Format("2006-01-02 15:04:05"),
		"updated_at":             paymentConfig.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if paymentConfig.ConfigJSON.Valid {
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(paymentConfig.ConfigJSON.String), &jsonData); err == nil {
			responseData["config_json"] = jsonData
		}
	}
	utils.SuccessResponse(c, http.StatusOK, "支付配置更新成功", responseData)
}
func getPagination(c *gin.Context) (page int, size int, offset int) {
	page, size = 1, 20
	if pageStr := c.Query("page"); pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
	}
	if sizeStr := c.Query("size"); sizeStr != "" {
		fmt.Sscanf(sizeStr, "%d", &size)
	}
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	return page, size, (page - 1) * size
}

func upsertSystemConfig(db *gorm.DB, category, key, value string) error {
	var config models.SystemConfig
	err := db.Where("key = ? AND category = ?", key, category).First(&config).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			config = models.SystemConfig{Key: key, Category: category, Value: value}
			return db.Create(&config).Error
		}
		return err
	}
	config.Value = value
	return db.Save(&config).Error
}

func getSystemConfigValue(db *gorm.DB, category, key string) (string, error) {
	var config models.SystemConfig
	if err := db.Where("category = ? AND key = ?", category, key).First(&config).Error; err != nil {
		return "", err
	}
	return config.Value, nil
}

func updateSingleConfig(c *gin.Context, category, key, label string) {
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}
	if err := upsertSystemConfig(database.GetDB(), category, key, req.Content); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("%s更新失败", label), err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("%s已更新", label), nil)
}

func safeNullString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func ptrToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return database.NullString(*s)
}
