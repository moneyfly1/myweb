package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetUnreadTicketRepliesCount 获取未读工单回复总数（用户和管理员通用）
func GetUnreadTicketRepliesCount(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	// 检查是否为管理员
	isAdmin := false
	if isAdminVal, exists := c.Get("is_admin"); exists {
		if isAdminBool, ok := isAdminVal.(bool); ok {
			isAdmin = isAdminBool
		} else if isAdminStr, ok := isAdminVal.(string); ok {
			isAdmin = isAdminStr == "true" || isAdminStr == "1"
		}
	}

	db := database.GetDB()
	var totalUnread int64 = 0
	
	if !isAdmin {
		// 用户端：统计未读的管理员回复
		db.Model(&models.TicketReply{}).
			Joins("JOIN tickets ON ticket_replies.ticket_id = tickets.id").
			Where("tickets.user_id = ? AND ticket_replies.is_admin = ? AND (ticket_replies.is_read = ? OR ticket_replies.read_by != ? OR ticket_replies.read_by IS NULL)", 
				user.ID, "true", false, user.ID).
			Count(&totalUnread)
	} else {
		// 管理员端：统计未读的用户回复 + 新工单
		// 1. 统计未读的用户回复
		var unreadReplies int64
		db.Model(&models.TicketReply{}).
			Where("is_admin != ? AND (is_read = ? OR read_by != ? OR read_by IS NULL)", 
				"true", false, user.ID).
			Count(&unreadReplies)
		
		// 2. 统计管理员未查看的新工单
		var newTickets int64
		db.Model(&models.Ticket{}).
			Where("id NOT IN (SELECT ticket_id FROM ticket_reads WHERE user_id = ?)", user.ID).
			Count(&newTickets)
		
		totalUnread = unreadReplies + newTickets
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"count": totalUnread,
	})
}

// CreateTicket 创建工单
func CreateTicket(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	var req struct {
		Title    string `json:"title" binding:"required"`
		Content  string `json:"content" binding:"required"`
		Type     string `json:"type"`
		Priority string `json:"priority"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	if req.Type == "" {
		req.Type = "other"
	}
	if req.Priority == "" {
		req.Priority = "normal"
	}

	db := database.GetDB()

	// 生成工单号
	ticketNo := utils.GenerateTicketNo(user.ID)

	// 清理输入，防止XSS
	title := utils.SanitizeInput(req.Title)
	content := utils.SanitizeInput(req.Content)

	// 限制长度
	if len(title) > 200 {
		title = title[:200]
	}
	if len(content) > 5000 {
		content = content[:5000]
	}

	ticket := models.Ticket{
		TicketNo: ticketNo,
		UserID:   user.ID,
		Title:    title,
		Content:  content,
		Type:     req.Type,
		Status:   "pending",
		Priority: req.Priority,
	}

	if err := db.Create(&ticket).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建工单失败", err)
		return
	}

	// 记录创建工单审计日志
	utils.SetResponseStatus(c, http.StatusCreated)
	utils.CreateAuditLogSimple(c, "create_ticket", "ticket", ticket.ID, fmt.Sprintf("创建工单: %s", ticket.Title))

	utils.SuccessResponse(c, http.StatusCreated, "", ticket)
}

// GetTickets 获取工单列表
func GetTickets(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	// 检查是否为管理员
	isAdmin := false
	if isAdminVal, exists := c.Get("is_admin"); exists {
		if isAdminBool, ok := isAdminVal.(bool); ok {
			isAdmin = isAdminBool
		} else if isAdminStr, ok := isAdminVal.(string); ok {
			isAdmin = isAdminStr == "true" || isAdminStr == "1"
		}
	}

	db := database.GetDB()
	query := db.Model(&models.Ticket{}).Preload("User").Preload("Assignee")

	// 如果不是管理员，只查询当前用户的工单
	if !isAdmin {
		query = query.Where("user_id = ?", user.ID)
	}

	// 筛选条件
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if ticketType := c.Query("type"); ticketType != "" {
		query = query.Where("type = ?", ticketType)
	}
	if priority := c.Query("priority"); priority != "" {
		query = query.Where("priority = ?", priority)
	}

	// 分页参数
	pagination := utils.ParsePagination(c)
	page := pagination.Page
	size := pagination.Size

	// 计算总数
	var total int64
	query.Count(&total)

	// 查询工单列表
	var tickets []models.Ticket
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&tickets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取工单列表失败",
		})
		return
	}

	// 批量查询优化：避免 N+1 查询问题
	ticketIDs := make([]uint, len(tickets))
	for i, t := range tickets {
		ticketIDs[i] = t.ID
	}

	// 批量查询所有工单的回复统计
	type ReplyStat struct {
		TicketID uint
		Count    int64
	}
	var totalRepliesStats []ReplyStat
	var unreadRepliesStats []ReplyStat
	
	if len(ticketIDs) > 0 {
		// 批量查询总回复数量
		db.Model(&models.TicketReply{}).
			Select("ticket_id, COUNT(*) as count").
			Where("ticket_id IN ?", ticketIDs).
			Group("ticket_id").
			Scan(&totalRepliesStats)

		// 批量查询未读回复数量
		if !isAdmin {
			// 用户端：统计未读的管理员回复
			db.Model(&models.TicketReply{}).
				Select("ticket_id, COUNT(*) as count").
				Where("ticket_id IN ? AND is_admin = ? AND (is_read = ? OR read_by != ? OR read_by IS NULL)",
					ticketIDs, "true", false, user.ID).
				Group("ticket_id").
				Scan(&unreadRepliesStats)
		} else {
			// 管理员端：统计未读的用户回复
			db.Model(&models.TicketReply{}).
				Select("ticket_id, COUNT(*) as count").
				Where("ticket_id IN ? AND is_admin != ? AND (is_read = ? OR read_by != ? OR read_by IS NULL)",
					ticketIDs, "true", false, user.ID).
				Group("ticket_id").
				Scan(&unreadRepliesStats)
		}
	}

	// 批量查询管理员查看记录（仅管理员端需要）
	var ticketReads []models.TicketRead
	ticketReadMap := make(map[uint]bool)
	if isAdmin && len(ticketIDs) > 0 {
		db.Where("ticket_id IN ? AND user_id = ?", ticketIDs, user.ID).Find(&ticketReads)
		for _, tr := range ticketReads {
			ticketReadMap[tr.TicketID] = true
		}
	}

	// 构建统计映射
	totalRepliesMap := make(map[uint]int64)
	unreadRepliesMap := make(map[uint]int64)
	for _, stat := range totalRepliesStats {
		totalRepliesMap[stat.TicketID] = stat.Count
	}
	for _, stat := range unreadRepliesStats {
		unreadRepliesMap[stat.TicketID] = stat.Count
	}

	// 格式化工单数据，包含未读回复数量
	ticketList := make([]gin.H, 0)
	for _, ticket := range tickets {
		unreadRepliesCount := unreadRepliesMap[ticket.ID]
		totalRepliesCount := totalRepliesMap[ticket.ID]
		
		var hasUnread bool
		if !isAdmin {
			hasUnread = unreadRepliesCount > 0
		} else {
			// 管理员端：检查是否有新工单（未查看过）或未读回复
			hasUnread = !ticketReadMap[ticket.ID] || unreadRepliesCount > 0
		}

		ticketList = append(ticketList, gin.H{
			"id":                 ticket.ID,
			"ticket_no":          ticket.TicketNo,
			"title":              ticket.Title,
			"content":            ticket.Content,
			"type":               ticket.Type,
			"status":             ticket.Status,
			"priority":           ticket.Priority,
			"created_at":         ticket.CreatedAt.Format("2006-01-02 15:04:05"),
			"updated_at":         ticket.UpdatedAt.Format("2006-01-02 15:04:05"),
			"replies_count":      totalRepliesCount,
			"unread_replies":     unreadRepliesCount, // 未读回复数量
			"has_unread":         hasUnread, // 是否有未读回复或新工单
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"tickets": ticketList,
			"total":   total,
			"page":    page,
			"size":    size,
		},
	})
}

// GetTicket 获取单个工单
func GetTicket(c *gin.Context) {
	id := c.Param("id")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	// 检查是否为管理员
	isAdmin := false
	if isAdminVal, exists := c.Get("is_admin"); exists {
		if isAdminBool, ok := isAdminVal.(bool); ok {
			isAdmin = isAdminBool
		} else if isAdminStr, ok := isAdminVal.(string); ok {
			isAdmin = isAdminStr == "true" || isAdminStr == "1"
		}
	}

	db := database.GetDB()
	var ticket models.Ticket
	query := db.Preload("User").Preload("Assignee").
		Preload("Replies", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Preload("Attachments").
		Where("id = ?", id)

	if !isAdmin {
		query = query.Where("user_id = ?", user.ID)
	}

	if err := query.First(&ticket).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "工单不存在",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "获取工单失败",
			})
		}
		return
	}

	// 构建响应数据，确保包含 replies
	responseData := gin.H{
		"id":         ticket.ID,
		"ticket_no":  ticket.TicketNo,
		"user_id":    ticket.UserID,
		"title":      ticket.Title,
		"content":    ticket.Content,
		"type":       ticket.Type,
		"status":     ticket.Status,
		"priority":   ticket.Priority,
		"created_at": ticket.CreatedAt.Format("2006-01-02 15:04:05"),
		"updated_at": ticket.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	// 添加可选字段
	if ticket.AssignedTo != nil {
		responseData["assigned_to"] = *ticket.AssignedTo
	}
	if ticket.AdminNotes != nil {
		responseData["admin_notes"] = *ticket.AdminNotes
	}
	if ticket.Rating != nil {
		responseData["rating"] = *ticket.Rating
	}
	if ticket.RatingComment != nil {
		responseData["rating_comment"] = *ticket.RatingComment
	}
	if ticket.ResolvedAt != nil {
		responseData["resolved_at"] = ticket.ResolvedAt.Format("2006-01-02 15:04:05")
	}
	if ticket.ClosedAt != nil {
		responseData["closed_at"] = ticket.ClosedAt.Format("2006-01-02 15:04:05")
	}

	// 格式化回复数据，突出显示管理员回复，并标记未读状态
	replies := make([]gin.H, 0)
	for _, reply := range ticket.Replies {
		replyData := gin.H{
			"id":         reply.ID,
			"ticket_id":  reply.TicketID,
			"user_id":    reply.UserID,
			"content":    reply.Content,
			"is_admin":   reply.IsAdmin,
			"created_at": reply.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		// 标记是否为管理员回复，用于前端样式区分
		if reply.IsAdmin == "true" {
			replyData["is_admin_reply"] = true
		}
		
		// 判断是否为未读（对于用户：管理员回复未读；对于管理员：用户回复未读）
		isUnread := false
		if !isAdmin && reply.IsAdmin == "true" {
			// 用户查看：管理员回复未读
			isUnread = !reply.IsRead || (reply.ReadBy != nil && *reply.ReadBy != user.ID)
		} else if isAdmin && reply.IsAdmin != "true" {
			// 管理员查看：用户回复未读
			isUnread = !reply.IsRead || (reply.ReadBy != nil && *reply.ReadBy != user.ID)
		}
		replyData["is_unread"] = isUnread
		
		replies = append(replies, replyData)
	}
	responseData["replies"] = replies

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
	responseData["attachments"] = attachments

	// 添加用户信息（如果已预加载）
	if ticket.User.ID > 0 {
		responseData["user"] = gin.H{
			"id":       ticket.User.ID,
			"username": ticket.User.Username,
			"email":    ticket.User.Email,
		}
	}

	// 添加分配人信息（如果已预加载）
	if ticket.Assignee.ID > 0 {
		responseData["assignee"] = gin.H{
			"id":       ticket.Assignee.ID,
			"username": ticket.Assignee.Username,
			"email":    ticket.Assignee.Email,
		}
	}

	// 标记回复为已读：用户查看时标记管理员回复为已读，管理员查看时标记用户回复为已读
	nowTime := utils.GetBeijingTime()
	userID := user.ID
	for i := range ticket.Replies {
		reply := &ticket.Replies[i]
		shouldMarkAsRead := false
		if !isAdmin && reply.IsAdmin == "true" {
			// 用户查看：标记管理员回复为已读
			shouldMarkAsRead = !reply.IsRead || (reply.ReadBy != nil && *reply.ReadBy != userID)
		} else if isAdmin && reply.IsAdmin != "true" {
			// 管理员查看：标记用户回复为已读
			shouldMarkAsRead = !reply.IsRead || (reply.ReadBy != nil && *reply.ReadBy != userID)
		}
		
		if shouldMarkAsRead {
			reply.IsRead = true
			reply.ReadBy = &userID
			reply.ReadAt = &nowTime
			db.Save(reply)
		}
	}
	
	// 记录工单查看时间（用于统计）
	var ticketRead models.TicketRead
	err := db.Where("ticket_id = ? AND user_id = ?", ticket.ID, user.ID).First(&ticketRead).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		ticketRead = models.TicketRead{
			TicketID: ticket.ID,
			UserID:   user.ID,
			ReadAt:   nowTime,
		}
		db.Create(&ticketRead)
	} else if err == nil {
		ticketRead.ReadAt = nowTime
		db.Save(&ticketRead)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    responseData,
	})
}

// ReplyTicket 回复工单
func ReplyTicket(c *gin.Context) {
	id := c.Param("id")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()

	// 验证工单
	var ticket models.Ticket
	query := db.Where("id = ?", id)

	// 检查是否为管理员
	isAdmin := false
	if isAdminVal, exists := c.Get("is_admin"); exists {
		if isAdminBool, ok := isAdminVal.(bool); ok {
			isAdmin = isAdminBool
		} else if isAdminStr, ok := isAdminVal.(string); ok {
			isAdmin = isAdminStr == "true" || isAdminStr == "1"
		}
	}

	// 如果不是管理员，只能回复自己的工单
	if !isAdmin {
		query = query.Where("user_id = ?", user.ID)
	}

	if err := query.First(&ticket).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "工单不存在",
		})
		return
	}

	// 创建回复（默认未读）
	reply := models.TicketReply{
		TicketID: ticket.ID,
		UserID:   user.ID,
		Content:  req.Content,
		IsAdmin:  fmt.Sprintf("%v", isAdmin),
		IsRead:   false, // 新回复默认未读
	}

	if err := db.Create(&reply).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "回复工单失败",
		})
		return
	}

	// 更新工单状态
	if ticket.Status == "pending" {
		ticket.Status = "processing"
		db.Save(&ticket)
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    reply,
	})
}

// UpdateTicketStatus 更新工单状态（管理员）
func UpdateTicketStatus(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Status     string `json:"status" binding:"required"`
		AssignedTo uint   `json:"assigned_to"`
		AdminNotes string `json:"admin_notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	var ticket models.Ticket
	if err := db.First(&ticket, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "工单不存在",
		})
		return
	}

	ticket.Status = req.Status
	if req.AssignedTo > 0 {
		assignedTo := int64(req.AssignedTo)
		ticket.AssignedTo = &assignedTo
	}
	if req.AdminNotes != "" {
		ticket.AdminNotes = &req.AdminNotes
	}

	if req.Status == "resolved" {
		now := time.Now()
		ticket.ResolvedAt = &now
	} else if req.Status == "closed" {
		now := time.Now()
		ticket.ClosedAt = &now
	}

	if err := db.Save(&ticket).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新工单失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "更新成功",
		"data":    ticket,
	})
}
