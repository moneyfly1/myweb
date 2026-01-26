package handlers

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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

// GetAdminInvites 管理员获取邀请码列表
func GetAdminInvites(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.InviteCode{}).Preload("User").Preload("InviteRelations")

	// 分页参数
	page := 1
	size := 20
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

	// 搜索和筛选
	if userQuery := c.Query("user_query"); userQuery != "" {
		// 清理和验证搜索关键词，防止SQL注入
		sanitizedQuery := utils.SanitizeSearchKeyword(userQuery)
		if sanitizedQuery != "" {
			query = query.Where("user_id IN (SELECT id FROM users WHERE username LIKE ? OR email LIKE ?)", "%"+sanitizedQuery+"%", "%"+sanitizedQuery+"%")
		}
	}
	if code := c.Query("code"); code != "" {
		// 清理邀请码，只允许字母数字
		sanitizedCode := utils.SanitizeSearchKeyword(code)
		if sanitizedCode != "" {
			query = query.Where("code LIKE ?", "%"+sanitizedCode+"%")
		}
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
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&inviteCodes).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取邀请码列表失败", err)
		return
	}

	// 处理数据格式，将sql.NullInt64转换为普通数字或null
	var result []gin.H
	for _, code := range inviteCodes {
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

		// 安全获取用户信息
		username := ""
		email := ""
		if code.User.ID != 0 {
			username = code.User.Username
			email = code.User.Email
		} else if code.UserID != 0 {
			// 如果预加载失败，手动查询用户
			var user models.User
			if err := db.First(&user, code.UserID).Error; err == nil {
				username = user.Username
				email = user.Email
			}
		}

		result = append(result, gin.H{
			"id":             code.ID,
			"code":           code.Code,
			"user_id":        code.UserID,
			"username":       username,
			"user_email":     email,
			"email":          email, // 保持向后兼容
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

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"invite_codes": result,
		"total":        total,
		"page":         page,
		"size":         size,
	})
}

// GetAdminInviteRelations 管理员获取邀请关系列表
func GetAdminInviteRelations(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.InviteRelation{}).Preload("Inviter").Preload("Invitee").Preload("InviteCode")

	// 分页参数
	page := 1
	size := 20
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

	// 搜索和筛选
	if inviterQuery := c.Query("inviter_query"); inviterQuery != "" {
		query = query.Where("inviter_id IN (SELECT id FROM users WHERE username LIKE ? OR email LIKE ?)", "%"+inviterQuery+"%", "%"+inviterQuery+"%")
	}
	if inviteeQuery := c.Query("invitee_query"); inviteeQuery != "" {
		query = query.Where("invitee_id IN (SELECT id FROM users WHERE username LIKE ? OR email LIKE ?)", "%"+inviteeQuery+"%", "%"+inviteeQuery+"%")
	}

	var total int64
	query.Count(&total)

	var relations []models.InviteRelation
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&relations).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取邀请关系列表失败", err)
		return
	}

	// 处理数据格式，扁平化用户信息
	var result []gin.H
	for _, relation := range relations {
		// 安全获取邀请码，防止空指针或预加载失败
		inviteCode := ""
		if relation.InviteCode.ID != 0 && relation.InviteCode.Code != "" {
			inviteCode = relation.InviteCode.Code
		} else if relation.InviteCodeID != 0 {
			// 如果预加载失败，手动查询邀请码
			var code models.InviteCode
			if err := db.First(&code, relation.InviteCodeID).Error; err == nil {
				inviteCode = code.Code
			}
		}

		// 安全获取邀请人信息
		inviterUsername := ""
		inviterEmail := ""
		if relation.Inviter.ID != 0 && relation.Inviter.Username != "" {
			inviterUsername = relation.Inviter.Username
			inviterEmail = relation.Inviter.Email
		} else if relation.InviterID != 0 {
			// 如果预加载失败，手动查询邀请人
			var inviter models.User
			if err := db.First(&inviter, relation.InviterID).Error; err == nil {
				inviterUsername = inviter.Username
				inviterEmail = inviter.Email
			}
		}

		// 安全获取被邀请人信息
		inviteeUsername := ""
		inviteeEmail := ""
		if relation.Invitee.ID != 0 && relation.Invitee.Username != "" {
			inviteeUsername = relation.Invitee.Username
			inviteeEmail = relation.Invitee.Email
		} else if relation.InviteeID != 0 {
			// 如果预加载失败，手动查询被邀请人
			var invitee models.User
			if err := db.First(&invitee, relation.InviteeID).Error; err == nil {
				inviteeUsername = invitee.Username
				inviteeEmail = invitee.Email
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

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"relations": result,
		"total":     total,
		"page":      page,
		"size":      size,
	})

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"relations": relations,
		"total":     total,
		"page":      page,
		"size":      size,
	})
}

// GetAdminInviteStatistics 管理员获取邀请统计
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

// GetAdminTickets 管理员工单列表
func GetAdminTickets(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.Ticket{}).Preload("User").Preload("Assignee")

	// 分页参数
	page := 1
	size := 20
	if pageStr := c.Query("page"); pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
	}
	if sizeStr := c.Query("size"); sizeStr != "" {
		fmt.Sscanf(sizeStr, "%d", &size)
	}

	// 搜索和筛选
	if keyword := c.Query("keyword"); keyword != "" {
		// 清理和验证搜索关键词，防止SQL注入
		sanitizedKeyword := utils.SanitizeSearchKeyword(keyword)
		if sanitizedKeyword != "" {
			query = query.Where("ticket_no LIKE ? OR title LIKE ? OR content LIKE ?", "%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%")
		}
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
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&tickets).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取工单列表失败", err)
		return
	}

	// 获取当前管理员用户ID（用于计算未读）
	adminUser, _ := middleware.GetCurrentUser(c)
	adminUserID := uint(0)
	if adminUser != nil {
		adminUserID = adminUser.ID
	}

	// 为每个工单添加回复数量和未读提醒
	ticketList := make([]gin.H, 0)
	for _, ticket := range tickets {
		var repliesCount int64
		db.Model(&models.TicketReply{}).Where("ticket_id = ?", ticket.ID).Count(&repliesCount)

		// 计算未读回复数量（用户回复）
		var unreadRepliesCount int64 = 0
		if adminUserID > 0 {
			db.Model(&models.TicketReply{}).
				Where("ticket_id = ? AND is_admin != ? AND (is_read = ? OR read_by != ? OR read_by IS NULL)",
					ticket.ID, "true", false, adminUserID).
				Count(&unreadRepliesCount)
		}

		// 检查是否有新工单（管理员未查看过）
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

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"tickets": ticketList,
		"total":   total,
		"page":    page,
		"size":    size,
	})
}

// GetAdminTicketStatistics 管理员工单统计
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

// GetAdminTicket 管理员获取单个工单详情
func GetAdminTicket(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()

	var ticket models.Ticket
	// Preload所有相关数据，包括按创建时间排序的回复
	if err := db.Preload("User").Preload("Assignee").
		Preload("Replies", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Preload("Attachments").
		First(&ticket, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "工单不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取工单失败", err)
		}
		return
	}

	// 计算回复数量
	var repliesCount int64
	db.Model(&models.TicketReply{}).Where("ticket_id = ?", ticket.ID).Count(&repliesCount)

	// 获取当前管理员用户ID
	adminUser, _ := middleware.GetCurrentUser(c)
	adminUserID := uint(0)
	if adminUser != nil {
		adminUserID = adminUser.ID
	}

	// 格式化回复数据，标记未读状态
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

		// 标记是否为未读（管理员查看：用户回复未读）
		if reply.IsAdmin != "true" {
			isUnread := !reply.IsRead || (reply.ReadBy != nil && *reply.ReadBy != adminUserID)
			replyData["is_unread"] = isUnread
			replyData["is_user_reply"] = true

			// 标记为已读
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

	// 记录管理员查看该工单的时间
	if adminUserID > 0 {
		var ticketRead models.TicketRead
		err := db.Where("ticket_id = ? AND user_id = ?", ticket.ID, adminUserID).First(&ticketRead).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ticketRead = models.TicketRead{
				TicketID: ticket.ID,
				UserID:   adminUserID,
				ReadAt:   now,
			}
			db.Create(&ticketRead)
		} else if err == nil {
			ticketRead.ReadAt = now
			db.Save(&ticketRead)
		}
	}

	// 格式化附件数据
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

	// 构建返回数据，包含回复数量
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

	// 添加可选字段
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

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"ticket": ticketData, // 前端期望嵌套在 ticket 字段中
	})
}

// GetAdminCoupons 管理员获取优惠券列表
func GetAdminCoupons(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.Coupon{})

	// 分页参数
	page := 1
	size := 20
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

	// 搜索参数（支持 keyword 搜索优惠券码或名称）
	keyword := c.Query("keyword")
	if keyword != "" {
		// 清理和验证搜索关键词，防止SQL注入
		sanitizedKeyword := utils.SanitizeSearchKeyword(keyword)
		if sanitizedKeyword != "" {
			query = query.Where("code LIKE ? OR name LIKE ?", "%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%")
		}
	}

	// 状态筛选
	if status := c.Query("status"); status != "" {
		switch status {
		case "active":
			query = query.Where("status = ?", "active")
		case "inactive":
			query = query.Where("status = ?", "inactive")
		case "expired":
			now := utils.GetBeijingTime()
			query = query.Where("valid_until < ?", now)
		}
	}

	// 类型筛选
	if couponType := c.Query("type"); couponType != "" {
		query = query.Where("type = ?", couponType)
	}

	// 计算总数
	var total int64
	query.Count(&total)

	// 分页查询
	var coupons []models.Coupon
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&coupons).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取优惠券列表失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"coupons": coupons,
		"total":   total,
		"page":    page,
		"size":    size,
	})
}

// GetAdminUserLevels 管理员获取用户等级列表
func GetAdminUserLevels(c *gin.Context) {
	db := database.GetDB()
	var userLevels []models.UserLevel
	if err := db.Order("level_order ASC").Find(&userLevels).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取用户等级列表失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", userLevels)
}

// CreateUserLevel 创建用户等级（管理员）
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
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误，请检查输入格式", err)
		return
	}

	db := database.GetDB()

	// 检查等级名称是否已存在
	var existing models.UserLevel
	if err := db.Where("level_name = ?", req.LevelName).First(&existing).Error; err == nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "等级名称已存在", nil)
		return
	}

	// 检查等级顺序是否已存在
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

// UpdateUserLevel 更新用户等级（管理员）
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
		IconURL        *string `json:"icon_url"` // 使用指针以区分空字符串和未传递
		Benefits       *string `json:"benefits"` // 使用指针以区分空字符串和未传递
		IsActive       *bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误，请检查输入格式", err)
		return
	}

	// 更新字段
	if req.LevelName != "" && req.LevelName != userLevel.LevelName {
		// 检查新名称是否已存在
		var existing models.UserLevel
		if err := db.Where("level_name = ? AND id != ?", req.LevelName, id).First(&existing).Error; err == nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "等级名称已存在", nil)
			return
		}
		userLevel.LevelName = req.LevelName
	}

	if req.LevelOrder > 0 && req.LevelOrder != userLevel.LevelOrder {
		// 检查新顺序是否已存在
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
	// 如果传递了IconURL字段（包括空字符串），则更新
	if req.IconURL != nil {
		userLevel.IconURL = database.NullString(*req.IconURL)
	}
	// 如果传递了Benefits字段（包括空字符串），则更新
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

// GetUserLevel 获取用户等级
func GetUserLevel(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	db := database.GetDB()
	var userLevel models.UserLevel
	if user.UserLevelID.Valid {
		if err := db.First(&userLevel, user.UserLevelID.Int64).Error; err != nil {
			utils.SuccessResponse(c, http.StatusOK, "", nil)
			return
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "", userLevel)
}

// GetUserSubscription 获取用户订阅
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

	// 生成订阅地址（使用统一的 GetBuildBaseURL 逻辑，优先从数据库配置获取域名）
	baseURL := utils.GetBuildBaseURL(c.Request, database.GetDB())
	clashURL := fmt.Sprintf("%s/api/v1/subscriptions/clash/%s", baseURL, subscription.SubscriptionURL)         // 猫咪订阅（Clash YAML格式）
	universalURL := fmt.Sprintf("%s/api/v1/subscriptions/universal/%s", baseURL, subscription.SubscriptionURL) // 通用订阅（Base64格式，适用于小火煎、v2ray等）

	// 计算到期时间
	expiryDate := "未设置"
	if !subscription.ExpireTime.IsZero() {
		expiryDate = subscription.ExpireTime.Format("2006-01-02 15:04:05")
	}

	// 生成二维码 URL（sub://格式，包含到期时间）
	encodedURL := base64.StdEncoding.EncodeToString([]byte(universalURL))
	expiryDisplay := expiryDate
	if expiryDisplay == "未设置" {
		expiryDisplay = subscription.SubscriptionURL
	}
	qrcodeURL := fmt.Sprintf("sub://%s#%s", encodedURL, url.QueryEscape(expiryDisplay))

	// 计算剩余天数
	remainingDays := 0
	isExpired := false
	if !subscription.ExpireTime.IsZero() {
		now := utils.GetBeijingTime()
		beijingTime := utils.ToBeijingTime(subscription.ExpireTime)
		diff := beijingTime.Sub(now)
		if diff > 0 {
			// 使用更精确的天数计算：将时间差转换为天数（向上取整）
			days := diff.Hours() / 24.0
			remainingDays = int(days)
			// 如果有小数部分（即使只有1小时），也算作1天
			if days > float64(remainingDays) {
				remainingDays++
			}
		} else {
			isExpired = true
		}
	}

	// 在线设备数
	var onlineDevices int64
	db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subscription.ID, true).Count(&onlineDevices)

	subscriptionData := gin.H{
		"id":               subscription.ID,
		"subscription_url": subscription.SubscriptionURL,
		"clash_url":        clashURL,
		"universal_url":    universalURL, // 通用订阅（Base64格式）
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
	}

	utils.SuccessResponse(c, http.StatusOK, "", subscriptionData)
}

// GetUserTheme 获取用户主题
func GetUserTheme(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"theme":    user.Theme,
		"language": user.Language,
	})
}

// UpdateUserTheme 更新用户主题
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

	db := database.GetDB()

	// 更新主题
	if req.Theme != "" {
		user.Theme = req.Theme
	}

	// 更新语言
	if req.Language != "" {
		user.Language = req.Language
	}

	if err := db.Save(&user).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新主题失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "主题更新成功", gin.H{
		"theme":    user.Theme,
		"language": user.Language,
	})
}

// GetAdminEmailQueue 管理员获取邮件队列
func GetAdminEmailQueue(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.EmailQueue{})

	// 分页参数
	page := 1
	size := 20
	if pageStr := c.Query("page"); pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
	}
	if sizeStr := c.Query("size"); sizeStr != "" {
		fmt.Sscanf(sizeStr, "%d", &size)
	}

	// 筛选
	if status := c.Query("status"); status != "" {
		status = strings.TrimSpace(status)
		if status != "" {
			query = query.Where("status = ?", status)
		}
	}
	if email := c.Query("email"); email != "" {
		email = strings.TrimSpace(email)
		if email != "" {
			query = query.Where("to_email LIKE ?", "%"+email+"%")
		}
	}

	var total int64
	query.Count(&total)

	var emails []models.EmailQueue
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&emails).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取邮件队列失败", err)
		return
	}

	// 计算总页数
	pages := (total + int64(size) - 1) / int64(size)
	if pages < 1 {
		pages = 1
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"emails": emails,
		"total":  total,
		"page":   page,
		"size":   size,
		"pages":  pages,
	})
}

// GetEmailQueueStatistics 获取邮件队列统计
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

	// 同步到新字段名
	stats.Total = stats.TotalEmails
	stats.Pending = stats.PendingEmails
	stats.Sent = stats.SentEmails
	stats.Failed = stats.FailedEmails

	utils.SuccessResponse(c, http.StatusOK, "", stats)
}

// GetEmailQueueDetail 获取邮件详情
func GetEmailQueueDetail(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()

	var email models.EmailQueue
	if err := db.First(&email, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "邮件不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取邮件详情失败", err)
		}
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", email)
}

// DeleteEmailFromQueue 删除邮件
func DeleteEmailFromQueue(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()

	var email models.EmailQueue
	if err := db.First(&email, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "邮件不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取邮件失败", err)
		}
		return
	}

	if err := db.Delete(&email).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除邮件失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "邮件删除成功", nil)
}

// RetryEmailFromQueue 重试发送邮件
func RetryEmailFromQueue(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()

	var email models.EmailQueue
	if err := db.First(&email, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "邮件不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取邮件失败", err)
		}
		return
	}

	// 重置邮件状态为待发送
	email.Status = "pending"
	email.RetryCount = 0
	email.ErrorMessage = sql.NullString{Valid: false}

	if err := db.Save(&email).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "重试邮件失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "邮件已加入重试队列", nil)
}

// ClearEmailQueue 清空邮件队列
func ClearEmailQueue(c *gin.Context) {
	status := c.Query("status")
	db := database.GetDB()

	var result *gorm.DB
	if status != "" {
		// 清空指定状态的邮件
		result = db.Where("status = ?", status).Delete(&models.EmailQueue{})
	} else {
		// 清空所有邮件
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

	utils.SuccessResponse(c, http.StatusOK, message, gin.H{
		"deleted_count": result.RowsAffected,
	})
}

// UpdateAdminSystemConfig 批量更新系统配置（管理员）
func UpdateAdminSystemConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误，请检查输入格式", err)
		return
	}

	db := database.GetDB()
	for key, value := range req {
		var config models.SystemConfig
		// 查找 system 类别的配置
		if err := db.Where("key = ? AND category = ?", key, "system").First(&config).Error; err != nil {
			// 如果不存在，创建新配置
			config = models.SystemConfig{
				Key:      key,
				Category: "system",
				Value:    fmt.Sprintf("%v", value),
			}
			if err := db.Create(&config).Error; err != nil {
				utils.ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("保存配置 %s 失败", key), err)
				return
			}
		} else {
			// 更新现有配置
			config.Value = fmt.Sprintf("%v", value)
			if err := db.Save(&config).Error; err != nil {
				utils.ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("更新配置 %s 失败", key), err)
				return
			}
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "系统配置保存成功", nil)
}

// GetAdminSystemConfig 获取系统配置
func GetAdminSystemConfig(c *gin.Context) {
	db := database.GetDB()
	var configs []models.SystemConfig
	// 获取 system 类别的配置
	db.Where("category = ?", "system").Order("sort_order ASC").Find(&configs)

	// 返回扁平化的配置对象，方便前端直接使用
	configMap := make(map[string]interface{})
	for _, config := range configs {
		// 处理布尔值
		if config.Value == "true" || config.Value == "false" {
			configMap[config.Key] = config.Value == "true"
		} else {
			configMap[config.Key] = config.Value
		}
	}

	// 如果没有配置，返回默认值
	if len(configMap) == 0 {
		configMap = map[string]interface{}{
			"site_name":           "",
			"site_description":    "",
			"logo_url":            "",
			"maintenance_mode":    false,
			"maintenance_message": "",
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "", configMap)
}

// GetAdminClashConfig 获取 Clash 配置
func GetAdminClashConfig(c *gin.Context) {
	db := database.GetDB()
	var config models.SystemConfig
	if err := db.Where("category = ? AND key = ?", "clash", "config").First(&config).Error; err != nil {
		utils.SuccessResponse(c, http.StatusOK, "", nil)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", config.Value)
}

// GetAdminV2RayConfig 获取 V2Ray 配置
func GetAdminV2RayConfig(c *gin.Context) {
	db := database.GetDB()
	var config models.SystemConfig
	if err := db.Where("category = ? AND key = ?", "v2ray", "config").First(&config).Error; err != nil {
		utils.SuccessResponse(c, http.StatusOK, "", nil)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", config.Value)
}

// GetAdminEmailConfig 获取邮件配置
func GetAdminEmailConfig(c *gin.Context) {
	db := database.GetDB()
	var configs []models.SystemConfig
	db.Where("category = ?", "email").Find(&configs)

	configMap := make(map[string]interface{})
	for _, config := range configs {
		configMap[config.Key] = config.Value
	}

	utils.SuccessResponse(c, http.StatusOK, "", configMap)
}

// GetAdminClashConfigInvalid 获取无效的 Clash 配置
func GetAdminClashConfigInvalid(c *gin.Context) {
	db := database.GetDB()
	var config models.SystemConfig
	if err := db.Where("category = ? AND key = ?", "clash", "config_invalid").First(&config).Error; err != nil {
		utils.SuccessResponse(c, http.StatusOK, "", "")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", config.Value)
}

// GetAdminV2RayConfigInvalid 获取无效的 V2Ray 配置
func GetAdminV2RayConfigInvalid(c *gin.Context) {
	db := database.GetDB()
	var config models.SystemConfig
	if err := db.Where("category = ? AND key = ?", "v2ray", "config_invalid").First(&config).Error; err != nil {
		utils.SuccessResponse(c, http.StatusOK, "", "")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", config.Value)
}

// GetSoftwareConfig 获取软件配置
func GetSoftwareConfig(c *gin.Context) {
	db := database.GetDB()
	var configs []models.SystemConfig
	db.Where("category = ?", "software").Find(&configs)

	configMap := make(map[string]interface{})
	for _, config := range configs {
		configMap[config.Key] = config.Value
	}

	utils.SuccessResponse(c, http.StatusOK, "", configMap)
}

// GetPaymentConfig 获取支付配置列表
func GetPaymentConfig(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.PaymentConfig{})

	// 分页参数
	page := 1
	size := 100
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

	var total int64
	query.Count(&total)

	var paymentConfigs []models.PaymentConfig
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&paymentConfigs).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取支付配置列表失败", err)
		return
	}

	// 转换数据格式，将 sql.NullString 转换为字符串
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
			AppID:                getPaymentConfigStringValue(config.AppID),
			MerchantPrivateKey:   getPaymentConfigStringValue(config.MerchantPrivateKey),
			AlipayPublicKey:      getPaymentConfigStringValue(config.AlipayPublicKey),
			WechatAppID:          getPaymentConfigStringValue(config.WechatAppID),
			WechatMchID:          getPaymentConfigStringValue(config.WechatMchID),
			WechatAPIKey:         getPaymentConfigStringValue(config.WechatAPIKey),
			PaypalClientID:       getPaymentConfigStringValue(config.PaypalClientID),
			PaypalSecret:         getPaymentConfigStringValue(config.PaypalSecret),
			StripePublishableKey: getPaymentConfigStringValue(config.StripePublishableKey),
			StripeSecretKey:      getPaymentConfigStringValue(config.StripeSecretKey),
			BankName:             getPaymentConfigStringValue(config.BankName),
			AccountName:          getPaymentConfigStringValue(config.AccountName),
			AccountNumber:        getPaymentConfigStringValue(config.AccountNumber),
			WalletAddress:        getPaymentConfigStringValue(config.WalletAddress),
			Status:               config.Status,
			ReturnURL:            getPaymentConfigStringValue(config.ReturnURL),
			NotifyURL:            getPaymentConfigStringValue(config.NotifyURL),
			SortOrder:            config.SortOrder,
			CreatedAt:            config.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:            config.UpdatedAt.Format("2006-01-02 15:04:05"),
		}

		// 解析 ConfigJSON
		if config.ConfigJSON.Valid {
			var jsonData map[string]interface{}
			if err := json.Unmarshal([]byte(config.ConfigJSON.String), &jsonData); err == nil {
				configsResponse[i].ConfigJSON = jsonData
			}
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"items": configsResponse,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// GetUserTrend 获取用户趋势
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
	rows, err := db.Raw(`
		SELECT DATE(created_at) as date, COUNT(*) as user_count
		FROM users 
		WHERE created_at >= DATE('now', '-' || ? || ' days')
		GROUP BY DATE(created_at)
		ORDER BY date ASC
	`, days).Rows()

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var trend UserTrend
			rows.Scan(&trend.Date, &trend.UserCount)
			trends = append(trends, trend)
		}
	}

	// 转换为前端期望的格式
	labels := make([]string, 0)
	data := make([]int64, 0)
	for _, trend := range trends {
		labels = append(labels, trend.Date)
		data = append(data, trend.UserCount)
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"labels": labels,
		"data":   data,
	})
}

// GetRevenueTrend 获取收入趋势
func GetRevenueTrend(c *gin.Context) {
	GetRevenueChart(c)
}

// UpdateClashConfig 更新 Clash 配置
func UpdateClashConfig(c *gin.Context) {
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	var config models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "config", "clash").First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 如果不存在，创建新配置
			config = models.SystemConfig{
				Key:      "config",
				Category: "clash",
				Value:    req.Content,
			}
			if err := db.Create(&config).Error; err != nil {
				utils.ErrorResponse(c, http.StatusInternalServerError, "创建配置失败", err)
				return
			}
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "查询配置失败", err)
			return
		}
	} else {
		// 更新现有配置
		config.Value = req.Content
		if err := db.Save(&config).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "更新配置失败", err)
			return
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "Clash 配置已更新", nil)
}

// UpdateV2RayConfig 更新 V2Ray 配置
func UpdateV2RayConfig(c *gin.Context) {
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	var config models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "config", "v2ray").First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 如果不存在，创建新配置
			config = models.SystemConfig{
				Key:      "config",
				Category: "v2ray",
				Value:    req.Content,
			}
			if err := db.Create(&config).Error; err != nil {
				utils.ErrorResponse(c, http.StatusInternalServerError, "创建配置失败", err)
				return
			}
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "查询配置失败", err)
			return
		}
	} else {
		// 更新现有配置
		config.Value = req.Content
		if err := db.Save(&config).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "更新配置失败", err)
			return
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "V2Ray 配置已更新", nil)
}

// UpdateEmailConfig 更新邮件配置
func UpdateEmailConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	for key, value := range req {
		var config models.SystemConfig
		if err := db.Where("key = ? AND category = ?", key, "email").First(&config).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// 如果不存在，创建新配置
				config = models.SystemConfig{
					Key:      key,
					Category: "email",
					Value:    fmt.Sprintf("%v", value),
				}
				if err := db.Create(&config).Error; err != nil {
					utils.ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("保存配置 %s 失败", key), err)
					return
				}
			} else {
				// 不向客户端返回详细错误信息，防止信息泄露
				utils.LogError("UpdateSystemConfig: query config", err, map[string]interface{}{
					"key": key,
				})
				utils.ErrorResponse(c, http.StatusInternalServerError, "更新配置失败，请稍后重试", err)
				return
			}
		} else {
			// 更新现有配置
			config.Value = fmt.Sprintf("%v", value)
			if err := db.Save(&config).Error; err != nil {
				utils.ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("更新配置 %s 失败", key), err)
				return
			}
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "邮件配置已更新", nil)
}

// MarkClashConfigInvalid 保存 Clash 失效配置
func MarkClashConfigInvalid(c *gin.Context) {
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	var config models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "config_invalid", "clash").First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 如果不存在，创建新配置
			config = models.SystemConfig{
				Key:      "config_invalid",
				Category: "clash",
				Value:    req.Content,
			}
			if err := db.Create(&config).Error; err != nil {
				utils.ErrorResponse(c, http.StatusInternalServerError, "创建配置失败", err)
				return
			}
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "查询配置失败", err)
			return
		}
	} else {
		// 更新现有配置
		config.Value = req.Content
		if err := db.Save(&config).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "更新配置失败", err)
			return
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "Clash 失效配置已保存", nil)
}

// MarkV2RayConfigInvalid 保存 V2Ray 失效配置
func MarkV2RayConfigInvalid(c *gin.Context) {
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	var config models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "config_invalid", "v2ray").First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 如果不存在，创建新配置
			config = models.SystemConfig{
				Key:      "config_invalid",
				Category: "v2ray",
				Value:    req.Content,
			}
			if err := db.Create(&config).Error; err != nil {
				utils.ErrorResponse(c, http.StatusInternalServerError, "创建配置失败", err)
				return
			}
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "查询配置失败", err)
			return
		}
	} else {
		// 更新现有配置
		config.Value = req.Content
		if err := db.Save(&config).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "更新配置失败", err)
			return
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "V2Ray 失效配置已保存", nil)
}

// CreatePaymentConfig 创建支付配置
func CreatePaymentConfig(c *gin.Context) {
	var req struct {
		PayType              string                 `json:"pay_type" binding:"required"`
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
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误，请检查输入格式", err)
		return
	}

	// 构建基础 URL
	baseURL := utils.GetBuildBaseURL(c.Request, database.GetDB())

	// 如果没有提供回调地址，自动生成
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

	// 默认状态为启用
	if req.Status == 0 {
		req.Status = 1
	}

	// 将 ConfigJSON 转换为 JSON 字符串
	var configJSONStr sql.NullString
	if req.ConfigJSON != nil {
		configJSONBytes, _ := json.Marshal(req.ConfigJSON)
		configJSONStr = sql.NullString{String: string(configJSONBytes), Valid: true}
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

	db := database.GetDB()
	if err := db.Create(&paymentConfig).Error; err != nil {
		// 不向客户端返回详细错误信息，防止信息泄露
		utils.LogError("CreatePaymentConfig: create payment config", err, nil)
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建支付配置失败，请稍后重试", err)
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "支付配置创建成功", paymentConfig)
}

// UpdatePaymentConfig 更新支付配置
func UpdatePaymentConfig(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		PayType              string                 `json:"pay_type"`
		AppID                *string                `json:"app_id,omitempty"`
		MerchantPrivateKey   *string                `json:"merchant_private_key,omitempty"`
		AlipayPublicKey      *string                `json:"alipay_public_key,omitempty"`
		WechatAppID          *string                `json:"wechat_app_id,omitempty"`
		WechatMchID          *string                `json:"wechat_mch_id,omitempty"`
		WechatAPIKey         *string                `json:"wechat_api_key,omitempty"`
		PaypalClientID       *string                `json:"paypal_client_id,omitempty"`
		PaypalSecret         *string                `json:"paypal_secret,omitempty"`
		StripePublishableKey *string                `json:"stripe_publishable_key,omitempty"`
		StripeSecretKey      *string                `json:"stripe_secret_key,omitempty"`
		BankName             *string                `json:"bank_name,omitempty"`
		AccountName          *string                `json:"account_name,omitempty"`
		AccountNumber        *string                `json:"account_number,omitempty"`
		WalletAddress        *string                `json:"wallet_address,omitempty"`
		Status               int                    `json:"status"`
		ReturnURL            *string                `json:"return_url,omitempty"`
		NotifyURL            *string                `json:"notify_url,omitempty"`
		SortOrder            *int                   `json:"sort_order,omitempty"`
		ConfigJSON           map[string]interface{} `json:"config_json,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误，请检查输入格式", err)
		return
	}

	db := database.GetDB()
	var paymentConfig models.PaymentConfig
	if err := db.First(&paymentConfig, id).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "支付配置不存在", err)
		return
	}

	// 构建基础 URL
	baseURL := utils.GetBuildBaseURL(c.Request, database.GetDB())

	// 如果易支付没有提供回调地址，自动生成
	if (req.PayType == "yipay" || strings.HasPrefix(req.PayType, "yipay_")) && req.NotifyURL == nil {
		notifyURL := fmt.Sprintf("%s/api/v1/payment/notify/yipay", baseURL)
		req.NotifyURL = &notifyURL
		utils.LogInfo("易支付回调地址自动生成: %s", notifyURL)
	}

	// 更新字段
	if req.PayType != "" {
		paymentConfig.PayType = req.PayType
	}
	// 更新可选字段（如果提供了值，即使为空字符串也更新）
	if req.AppID != nil {
		paymentConfig.AppID = database.NullString(*req.AppID)
	}
	if req.MerchantPrivateKey != nil {
		paymentConfig.MerchantPrivateKey = database.NullString(*req.MerchantPrivateKey)
	}
	if req.AlipayPublicKey != nil {
		paymentConfig.AlipayPublicKey = database.NullString(*req.AlipayPublicKey)
	}
	if req.WechatAppID != nil {
		paymentConfig.WechatAppID = database.NullString(*req.WechatAppID)
	}
	if req.WechatMchID != nil {
		paymentConfig.WechatMchID = database.NullString(*req.WechatMchID)
	}
	if req.WechatAPIKey != nil {
		paymentConfig.WechatAPIKey = database.NullString(*req.WechatAPIKey)
	}
	if req.PaypalClientID != nil {
		paymentConfig.PaypalClientID = database.NullString(*req.PaypalClientID)
	}
	if req.PaypalSecret != nil {
		paymentConfig.PaypalSecret = database.NullString(*req.PaypalSecret)
	}
	if req.StripePublishableKey != nil {
		paymentConfig.StripePublishableKey = database.NullString(*req.StripePublishableKey)
	}
	if req.StripeSecretKey != nil {
		paymentConfig.StripeSecretKey = database.NullString(*req.StripeSecretKey)
	}
	if req.BankName != nil {
		paymentConfig.BankName = database.NullString(*req.BankName)
	}
	if req.AccountName != nil {
		paymentConfig.AccountName = database.NullString(*req.AccountName)
	}
	if req.AccountNumber != nil {
		paymentConfig.AccountNumber = database.NullString(*req.AccountNumber)
	}
	if req.WalletAddress != nil {
		paymentConfig.WalletAddress = database.NullString(*req.WalletAddress)
	}
	// Status 字段总是更新（允许设置为0）
	if req.Status >= 0 {
		paymentConfig.Status = req.Status
	}
	if req.ReturnURL != nil {
		paymentConfig.ReturnURL = database.NullString(*req.ReturnURL)
	}
	if req.NotifyURL != nil {
		paymentConfig.NotifyURL = database.NullString(*req.NotifyURL)
	} else if req.PayType != "" && paymentConfig.NotifyURL.String == "" {
		// 如果更新了支付类型但没有提供回调地址，自动生成
		notifySuffix := "alipay"
		if req.PayType == "wechat" {
			notifySuffix = "wechat"
		} else if req.PayType == "yipay" || strings.HasPrefix(req.PayType, "yipay_") {
			notifySuffix = "yipay"
		}
		paymentConfig.NotifyURL = database.NullString(fmt.Sprintf("%s/api/v1/payment/notify/%s", baseURL, notifySuffix))
	}
	// SortOrder 总是更新（如果提供了值）
	if req.SortOrder != nil {
		paymentConfig.SortOrder = *req.SortOrder
	}
	if req.ConfigJSON != nil {
		configJSONBytes, err := json.Marshal(req.ConfigJSON)
		if err == nil {
			paymentConfig.ConfigJSON = sql.NullString{String: string(configJSONBytes), Valid: true}
		}
	}

	if err := db.Save(&paymentConfig).Error; err != nil {
		// 不向客户端返回详细错误信息，防止信息泄露
		utils.LogError("UpdatePaymentConfig: update payment config", err, map[string]interface{}{
			"payment_config_id": id,
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新支付配置失败，请稍后重试", err)
		return
	}

	// 构建响应数据（转换 sql.NullString 为字符串）
	responseData := gin.H{
		"id":                     paymentConfig.ID,
		"pay_type":               paymentConfig.PayType,
		"app_id":                 getPaymentConfigStringValue(paymentConfig.AppID),
		"merchant_private_key":   getPaymentConfigStringValue(paymentConfig.MerchantPrivateKey),
		"alipay_public_key":      getPaymentConfigStringValue(paymentConfig.AlipayPublicKey),
		"wechat_app_id":          getPaymentConfigStringValue(paymentConfig.WechatAppID),
		"wechat_mch_id":          getPaymentConfigStringValue(paymentConfig.WechatMchID),
		"wechat_api_key":         getPaymentConfigStringValue(paymentConfig.WechatAPIKey),
		"paypal_client_id":       getPaymentConfigStringValue(paymentConfig.PaypalClientID),
		"paypal_secret":          getPaymentConfigStringValue(paymentConfig.PaypalSecret),
		"stripe_publishable_key": getPaymentConfigStringValue(paymentConfig.StripePublishableKey),
		"stripe_secret_key":      getPaymentConfigStringValue(paymentConfig.StripeSecretKey),
		"bank_name":              getPaymentConfigStringValue(paymentConfig.BankName),
		"account_name":           getPaymentConfigStringValue(paymentConfig.AccountName),
		"account_number":         getPaymentConfigStringValue(paymentConfig.AccountNumber),
		"wallet_address":         getPaymentConfigStringValue(paymentConfig.WalletAddress),
		"status":                 paymentConfig.Status,
		"return_url":             getPaymentConfigStringValue(paymentConfig.ReturnURL),
		"notify_url":             getPaymentConfigStringValue(paymentConfig.NotifyURL),
		"sort_order":             paymentConfig.SortOrder,
		"created_at":             paymentConfig.CreatedAt.Format("2006-01-02 15:04:05"),
		"updated_at":             paymentConfig.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	// 解析 ConfigJSON
	if paymentConfig.ConfigJSON.Valid {
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(paymentConfig.ConfigJSON.String), &jsonData); err == nil {
			responseData["config_json"] = jsonData
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "支付配置更新成功", responseData)
}

// getPaymentConfigStringValue 辅助函数：从 sql.NullString 获取字符串值（用于支付配置）
func getPaymentConfigStringValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
