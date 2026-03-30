package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/email"
	"cboard-go/internal/services/notification"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ------------------------------------------------------------------------
// 公共辅助函数 (Helpers)
// ------------------------------------------------------------------------

// getIsAdmin 统一获取当前请求上下文中的管理员身份
func getIsAdmin(c *gin.Context) bool {
	if isAdminVal, exists := c.Get("is_admin"); exists {
		if isAdminBool, ok := isAdminVal.(bool); ok {
			return isAdminBool
		}
	}
	return false
}

// checkDBError 统一处理数据库查询错误，如果是未找到则返回 true 并写入 404 响应
func checkDBError(c *gin.Context, err error, notFoundMsg string) bool {
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.ErrorResponse(c, http.StatusNotFound, notFoundMsg, err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取工单失败", err)
		}
		return true
	}
	return false
}

// truncateString 限制字符串最大 rune 长度
func truncateString(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes])
	}
	return s
}

// processTicketReadStatus 提取的副作用函数：处理查看工单详情时的"已读状态"更新
func processTicketReadStatus(db *gorm.DB, ticket *models.Ticket, userID uint, isAdmin bool) {
	nowTime := utils.GetBeijingTime()
	var toMarkReadIDs []uint

	// 找出需要标记为已读的回复
	for i := range ticket.Replies {
		reply := &ticket.Replies[i]
		isFromOtherSide := reply.IsAdmin != isAdmin
		shouldMarkAsRead := isFromOtherSide && (!reply.IsRead || (reply.ReadBy != nil && *reply.ReadBy != userID))

		if shouldMarkAsRead {
			toMarkReadIDs = append(toMarkReadIDs, reply.ID)
		}
	}

	// 批量更新回复表
	if len(toMarkReadIDs) > 0 {
		db.Model(&models.TicketReply{}).Where("id IN ?", toMarkReadIDs).Updates(map[string]interface{}{
			"is_read": true,
			"read_by": userID,
			"read_at": nowTime,
		})
	}

	// 维护工单主体的已读记录表
	var ticketRead models.TicketRead
	err := db.Where("ticket_id = ? AND user_id = ?", ticket.ID, userID).First(&ticketRead).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		db.Create(&models.TicketRead{
			TicketID: ticket.ID,
			UserID:   userID,
			ReadAt:   nowTime,
		})
	} else if err == nil {
		ticketRead.ReadAt = nowTime
		db.Save(&ticketRead)
	}
}

// asyncNotifyTicketReply 提取的异步通知机制：处理工单回复后的站内信/邮件通知
func asyncNotifyTicketReply(db *gorm.DB, ticket *models.Ticket, user *models.User, replyContent string, isAdmin bool) {
	if !isAdmin {
		// 用户回复 → 通知管理员
		go notification.NewNotificationService().SendAdminNotification("ticket_replied", map[string]interface{}{
			"username":   user.Username,
			"email":      user.Email,
			"ticket_no":  ticket.TicketNo,
			"title":      ticket.Title,
			"reply_time": utils.FormatBeijingTime(utils.GetBeijingTime()),
		})
	} else {
		// 管理员回复 → 尝试邮件通知工单所有者
		var ticketOwner models.User
		if err := db.First(&ticketOwner, ticket.UserID).Error; err == nil && ticketOwner.Email != "" {
			go func() {
				emailService := email.NewEmailService()
				templateBuilder := email.NewEmailTemplateBuilder()
				content := templateBuilder.GetAdminReplyNotificationTemplate(ticket.TicketNo, ticket.Title, replyContent)
				if err := emailService.QueueEmail(ticketOwner.Email, "您的工单有新回复", content, "ticket_reply"); err != nil {
					utils.LogErrorMsg("发送工单回复邮件失败: ticket=%s, email=%s, error=%v", ticket.TicketNo, ticketOwner.Email, err)
				}
			}()
		}
	}
}

// ------------------------------------------------------------------------
// 控制器方法 (Handlers)
// ------------------------------------------------------------------------

func GetUnreadTicketRepliesCount(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	isAdmin := getIsAdmin(c)
	db := database.GetDB()
	var totalUnread int64 = 0

	if !isAdmin {
		db.Model(&models.TicketReply{}).
			Joins("JOIN tickets ON ticket_replies.ticket_id = tickets.id").
			Where("tickets.user_id = ? AND ticket_replies.is_admin = ? AND (ticket_replies.is_read = ? OR ticket_replies.read_by != ? OR ticket_replies.read_by IS NULL)",
				user.ID, true, false, user.ID).
			Count(&totalUnread)
	} else {
		var unreadReplies int64
		db.Model(&models.TicketReply{}).
			Where("is_admin = ? AND (is_read = ? OR read_by != ? OR read_by IS NULL)", false, false, user.ID).
			Count(&unreadReplies)

		var newTickets int64
		db.Model(&models.Ticket{}).
			Where("id NOT IN (SELECT ticket_id FROM ticket_reads WHERE user_id = ?)", user.ID).
			Count(&newTickets)

		totalUnread = unreadReplies + newTickets
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{"count": totalUnread})
}

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

	// 设置默认值
	if req.Type == "" {
		req.Type = "other"
	}
	if req.Priority == "" {
		req.Priority = "normal"
	}

	title := truncateString(utils.SanitizeInput(req.Title), 200)
	content := truncateString(utils.SanitizeInput(req.Content), 5000)

	ticket := models.Ticket{
		TicketNo: utils.GenerateTicketNo(user.ID),
		UserID:   user.ID,
		Title:    title,
		Content:  content,
		Type:     req.Type,
		Status:   "pending",
		Priority: req.Priority,
	}

	if err := database.GetDB().Create(&ticket).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建工单失败", err)
		return
	}

	utils.SetResponseStatus(c, http.StatusCreated)
	utils.CreateAuditLogSimple(c, "create_ticket", "ticket", ticket.ID, fmt.Sprintf("创建工单: %s", ticket.Title))

	go notification.NewNotificationService().SendAdminNotification("ticket_created", map[string]interface{}{
		"username":    user.Username,
		"email":       user.Email,
		"ticket_no":   ticket.TicketNo,
		"title":       ticket.Title,
		"type":        ticket.Type,
		"priority":    ticket.Priority,
		"create_time": utils.FormatBeijingTime(utils.GetBeijingTime()),
	})

	utils.SuccessResponse(c, http.StatusCreated, "", ticket)
}

func GetTickets(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	isAdmin := getIsAdmin(c)
	db := database.GetDB()
	query := db.Model(&models.Ticket{}).Preload("User").Preload("Assignee")

	if !isAdmin {
		query = query.Where("user_id = ?", user.ID)
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

	pagination := utils.ParsePagination(c)
	var total int64
	query.Count(&total)

	var tickets []models.Ticket
	offset := (pagination.Page - 1) * pagination.Size
	if err := query.Offset(offset).Limit(pagination.Size).Order("created_at DESC").Find(&tickets).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取工单列表失败", err)
		return
	}

	// 提取 ID 用于批量查询
	ticketIDs := make([]uint, len(tickets))
	for i, t := range tickets {
		ticketIDs[i] = t.ID
	}

	type ReplyStat struct {
		TicketID uint
		Count    int64
	}
	var totalRepliesStats, unreadRepliesStats []ReplyStat

	if len(ticketIDs) > 0 {
		// 统计总回复数
		db.Model(&models.TicketReply{}).
			Select("ticket_id, COUNT(*) as count").
			Where("ticket_id IN ?", ticketIDs).
			Group("ticket_id").
			Scan(&totalRepliesStats)

		// 统一处理未读回复统计逻辑：查询来自"对方"的未读回复 (若为Admin，找普通用户的；若为普通用户，找Admin的)
		db.Model(&models.TicketReply{}).
			Select("ticket_id, COUNT(*) as count").
			Where("ticket_id IN ? AND is_admin = ? AND (is_read = ? OR read_by != ? OR read_by IS NULL)",
				ticketIDs, !isAdmin, false, user.ID).
			Group("ticket_id").
			Scan(&unreadRepliesStats)
	}

	// 获取管理员的工单已读记录表
	ticketReadMap := make(map[uint]bool)
	if isAdmin && len(ticketIDs) > 0 {
		var ticketReads []models.TicketRead
		db.Where("ticket_id IN ? AND user_id = ?", ticketIDs, user.ID).Find(&ticketReads)
		for _, tr := range ticketReads {
			ticketReadMap[tr.TicketID] = true
		}
	}

	// 转换为快速映射表
	totalRepliesMap := make(map[uint]int64)
	unreadRepliesMap := make(map[uint]int64)
	for _, stat := range totalRepliesStats {
		totalRepliesMap[stat.TicketID] = stat.Count
	}
	for _, stat := range unreadRepliesStats {
		unreadRepliesMap[stat.TicketID] = stat.Count
	}

	// 构建最终返回列表
	ticketList := make([]gin.H, 0, len(tickets))
	for _, ticket := range tickets {
		unreadRepliesCount := unreadRepliesMap[ticket.ID]
		hasUnread := unreadRepliesCount > 0
		if isAdmin {
			hasUnread = !ticketReadMap[ticket.ID] || hasUnread
		}

		ticketList = append(ticketList, gin.H{
			"id":             ticket.ID,
			"ticket_no":      ticket.TicketNo,
			"title":          ticket.Title,
			"content":        ticket.Content,
			"type":           ticket.Type,
			"status":         ticket.Status,
			"priority":       ticket.Priority,
			"created_at":     utils.FormatBeijingTime(ticket.CreatedAt),
			"updated_at":     utils.FormatBeijingTime(ticket.UpdatedAt),
			"replies_count":  totalRepliesMap[ticket.ID],
			"unread_replies": unreadRepliesCount,
			"has_unread":     hasUnread,
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"tickets": ticketList,
		"total":   total,
		"page":    pagination.Page,
		"size":    pagination.Size,
	})
}

func GetTicket(c *gin.Context) {
	id := c.Param("id")
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	isAdmin := getIsAdmin(c)
	db := database.GetDB()
	var ticket models.Ticket

	query := db.Preload("User").Preload("Assignee").
		Preload("Replies", func(db *gorm.DB) *gorm.DB { return db.Order("created_at ASC") }).
		Preload("Attachments").
		Where("id = ?", id)

	if !isAdmin {
		query = query.Where("user_id = ?", user.ID)
	}

	if checkDBError(c, query.First(&ticket).Error, "工单不存在或无权限") {
		return
	}

	// 1. 构建基础信息
	responseData := gin.H{
		"id":         ticket.ID,
		"ticket_no":  ticket.TicketNo,
		"user_id":    ticket.UserID,
		"title":      ticket.Title,
		"content":    ticket.Content,
		"type":       ticket.Type,
		"status":     ticket.Status,
		"priority":   ticket.Priority,
		"created_at": utils.FormatBeijingTime(ticket.CreatedAt),
		"updated_at": utils.FormatBeijingTime(ticket.UpdatedAt),
	}

	// 补充可选字段
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
		responseData["resolved_at"] = utils.FormatBeijingTime(*ticket.ResolvedAt)
	}
	if ticket.ClosedAt != nil {
		responseData["closed_at"] = utils.FormatBeijingTime(*ticket.ClosedAt)
	}

	if ticket.User.ID > 0 {
		responseData["user"] = gin.H{"id": ticket.User.ID, "username": ticket.User.Username, "email": ticket.User.Email}
	}
	if ticket.Assignee.ID > 0 {
		responseData["assignee"] = gin.H{"id": ticket.Assignee.ID, "username": ticket.Assignee.Username, "email": ticket.Assignee.Email}
	}

	// 2. 梳理回复列表 (同时判断未读状态)
	replies := make([]gin.H, 0, len(ticket.Replies))
	for _, reply := range ticket.Replies {
		isFromOtherSide := reply.IsAdmin != isAdmin
		isUnread := isFromOtherSide && (!reply.IsRead || (reply.ReadBy != nil && *reply.ReadBy != user.ID))

		replies = append(replies, gin.H{
			"id":             reply.ID,
			"ticket_id":      reply.TicketID,
			"user_id":        reply.UserID,
			"content":        reply.Content,
			"is_admin":       reply.IsAdmin,
			"is_admin_reply": reply.IsAdmin, // 兼容原代码字段
			"created_at":     utils.FormatBeijingTime(reply.CreatedAt),
			"is_unread":      isUnread,
		})
	}
	responseData["replies"] = replies

	// 3. 梳理附件列表
	attachments := make([]gin.H, 0, len(ticket.Attachments))
	for _, attachment := range ticket.Attachments {
		att := gin.H{
			"id":          attachment.ID,
			"ticket_id":   attachment.TicketID,
			"file_name":   attachment.FileName,
			"file_path":   attachment.FilePath,
			"uploaded_by": attachment.UploadedBy,
			"created_at":  utils.FormatBeijingTime(attachment.CreatedAt),
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

	// 4. 执行更新已读状态的副作用动作
	go processTicketReadStatus(db, &ticket, user.ID, isAdmin)

	utils.SuccessResponse(c, http.StatusOK, "", responseData)
}

func ReplyTicket(c *gin.Context) {
	id := c.Param("id")
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	isAdmin := getIsAdmin(c)
	db := database.GetDB()
	var ticket models.Ticket

	query := db.Where("id = ?", id)
	if !isAdmin {
		query = query.Where("user_id = ?", user.ID)
	}

	if checkDBError(c, query.First(&ticket).Error, "工单不存在或无权限") {
		return
	}

	reply := models.TicketReply{
		TicketID: ticket.ID,
		UserID:   user.ID,
		Content:  req.Content,
		IsAdmin:  isAdmin,
		IsRead:   false,
	}

	if err := db.Create(&reply).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "回复工单失败", err)
		return
	}

	if ticket.Status == "pending" {
		ticket.Status = "processing"
		db.Save(&ticket)
	}

	// 异步执行通知机制
	asyncNotifyTicketReply(db, &ticket, user, req.Content, isAdmin)

	utils.SuccessResponse(c, http.StatusCreated, "", reply)
}

func UpdateTicketStatus(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status     string `json:"status" binding:"required"`
		AssignedTo uint   `json:"assigned_to"`
		AdminNotes string `json:"admin_notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	var ticket models.Ticket
	if checkDBError(c, db.First(&ticket, id).Error, "工单不存在") {
		return
	}

	ticket.Status = req.Status
	if req.AssignedTo > 0 {
		assignedTo := utils.MustSafeUintToInt64(req.AssignedTo)
		ticket.AssignedTo = &assignedTo
	}
	if req.AdminNotes != "" {
		ticket.AdminNotes = &req.AdminNotes
	}

	// 统一获取时间设置完成/关闭时间
	now := utils.GetBeijingTime()
	if req.Status == "resolved" {
		ticket.ResolvedAt = &now
	} else if req.Status == "closed" {
		ticket.ClosedAt = &now
	}

	if err := db.Save(&ticket).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新工单失败", err)
		return
	}

	utils.CreateAuditLogSimple(c, "update_ticket_status", "ticket", ticket.ID, fmt.Sprintf("管理员操作: 更新工单状态 %s -> %s", ticket.Title, req.Status))
	utils.SuccessResponse(c, http.StatusOK, "更新成功", ticket)
}

func CloseTicket(c *gin.Context) {
	id := c.Param("id")
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	db := database.GetDB()
	var ticket models.Ticket

	if checkDBError(c, db.Where("id = ? AND user_id = ?", id, user.ID).First(&ticket).Error, "工单不存在或无权限") {
		return
	}

	if ticket.Status == "closed" {
		utils.ErrorResponse(c, http.StatusBadRequest, "工单已关闭", nil)
		return
	}

	ticket.Status = "closed"
	now := utils.GetBeijingTime()
	ticket.ClosedAt = &now

	if err := db.Save(&ticket).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "关闭工单失败", err)
		return
	}

	utils.CreateAuditLogSimple(c, "close_ticket", "ticket", ticket.ID, fmt.Sprintf("关闭工单: %s", ticket.Title))
	utils.SuccessResponse(c, http.StatusOK, "工单已关闭", ticket)
}
