package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetUnreadTicketRepliesCount(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

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
		db.Model(&models.TicketReply{}).
			Joins("JOIN tickets ON ticket_replies.ticket_id = tickets.id").
			Where("tickets.user_id = ? AND ticket_replies.is_admin = ? AND (ticket_replies.is_read = ? OR ticket_replies.read_by != ? OR ticket_replies.read_by IS NULL)",
				user.ID, "true", false, user.ID).
			Count(&totalUnread)
	} else {
		var unreadReplies int64
		db.Model(&models.TicketReply{}).
			Where("is_admin != ? AND (is_read = ? OR read_by != ? OR read_by IS NULL)",
				"true", false, user.ID).
			Count(&unreadReplies)

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

	ticketNo := utils.GenerateTicketNo(user.ID)

	title := utils.SanitizeInput(req.Title)
	content := utils.SanitizeInput(req.Content)

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

	utils.SetResponseStatus(c, http.StatusCreated)
	utils.CreateAuditLogSimple(c, "create_ticket", "ticket", ticket.ID, fmt.Sprintf("创建工单: %s", ticket.Title))

	utils.SuccessResponse(c, http.StatusCreated, "", ticket)
}

func GetTickets(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

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
	page := pagination.Page
	size := pagination.Size

	var total int64
	query.Count(&total)

	var tickets []models.Ticket
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&tickets).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取工单列表失败", err)
		return
	}

	ticketIDs := make([]uint, len(tickets))
	for i, t := range tickets {
		ticketIDs[i] = t.ID
	}

	type ReplyStat struct {
		TicketID uint
		Count    int64
	}
	var totalRepliesStats []ReplyStat
	var unreadRepliesStats []ReplyStat

	if len(ticketIDs) > 0 {
		db.Model(&models.TicketReply{}).
			Select("ticket_id, COUNT(*) as count").
			Where("ticket_id IN ?", ticketIDs).
			Group("ticket_id").
			Scan(&totalRepliesStats)

		if !isAdmin {
			db.Model(&models.TicketReply{}).
				Select("ticket_id, COUNT(*) as count").
				Where("ticket_id IN ? AND is_admin = ? AND (is_read = ? OR read_by != ? OR read_by IS NULL)",
					ticketIDs, "true", false, user.ID).
				Group("ticket_id").
				Scan(&unreadRepliesStats)
		} else {
			db.Model(&models.TicketReply{}).
				Select("ticket_id, COUNT(*) as count").
				Where("ticket_id IN ? AND is_admin != ? AND (is_read = ? OR read_by != ? OR read_by IS NULL)",
					ticketIDs, "true", false, user.ID).
				Group("ticket_id").
				Scan(&unreadRepliesStats)
		}
	}

	var ticketReads []models.TicketRead
	ticketReadMap := make(map[uint]bool)
	if isAdmin && len(ticketIDs) > 0 {
		db.Where("ticket_id IN ? AND user_id = ?", ticketIDs, user.ID).Find(&ticketReads)
		for _, tr := range ticketReads {
			ticketReadMap[tr.TicketID] = true
		}
	}

	totalRepliesMap := make(map[uint]int64)
	unreadRepliesMap := make(map[uint]int64)
	for _, stat := range totalRepliesStats {
		totalRepliesMap[stat.TicketID] = stat.Count
	}
	for _, stat := range unreadRepliesStats {
		unreadRepliesMap[stat.TicketID] = stat.Count
	}

	ticketList := make([]gin.H, 0)
	for _, ticket := range tickets {
		unreadRepliesCount := unreadRepliesMap[ticket.ID]
		totalRepliesCount := totalRepliesMap[ticket.ID]

		var hasUnread bool
		if !isAdmin {
			hasUnread = unreadRepliesCount > 0
		} else {
			hasUnread = !ticketReadMap[ticket.ID] || unreadRepliesCount > 0
		}

		ticketList = append(ticketList, gin.H{
			"id":             ticket.ID,
			"ticket_no":      ticket.TicketNo,
			"title":          ticket.Title,
			"content":        ticket.Content,
			"type":           ticket.Type,
			"status":         ticket.Status,
			"priority":       ticket.Priority,
			"created_at":     ticket.CreatedAt.Format("2006-01-02 15:04:05"),
			"updated_at":     ticket.UpdatedAt.Format("2006-01-02 15:04:05"),
			"replies_count":  totalRepliesCount,
			"unread_replies": unreadRepliesCount, // 未读回复数量
			"has_unread":     hasUnread,          // 是否有未读回复或新工单
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"tickets": ticketList,
		"total":   total,
		"page":    page,
		"size":    size,
	})
}

func GetTicket(c *gin.Context) {
	id := c.Param("id")
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

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
			utils.ErrorResponse(c, http.StatusNotFound, "工单不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取工单失败", err)
		}
		return
	}

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
		if reply.IsAdmin == "true" {
			replyData["is_admin_reply"] = true
		}

		isUnread := false
		if !isAdmin && reply.IsAdmin == "true" {
			isUnread = !reply.IsRead || (reply.ReadBy != nil && *reply.ReadBy != user.ID)
		} else if isAdmin && reply.IsAdmin != "true" {
			isUnread = !reply.IsRead || (reply.ReadBy != nil && *reply.ReadBy != user.ID)
		}
		replyData["is_unread"] = isUnread

		replies = append(replies, replyData)
	}
	responseData["replies"] = replies

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

	if ticket.User.ID > 0 {
		responseData["user"] = gin.H{
			"id":       ticket.User.ID,
			"username": ticket.User.Username,
			"email":    ticket.User.Email,
		}
	}

	if ticket.Assignee.ID > 0 {
		responseData["assignee"] = gin.H{
			"id":       ticket.Assignee.ID,
			"username": ticket.Assignee.Username,
			"email":    ticket.Assignee.Email,
		}
	}

	nowTime := utils.GetBeijingTime()
	userID := user.ID
	for i := range ticket.Replies {
		reply := &ticket.Replies[i]
		shouldMarkAsRead := false
		if !isAdmin && reply.IsAdmin == "true" {
			shouldMarkAsRead = !reply.IsRead || (reply.ReadBy != nil && *reply.ReadBy != userID)
		} else if isAdmin && reply.IsAdmin != "true" {
			shouldMarkAsRead = !reply.IsRead || (reply.ReadBy != nil && *reply.ReadBy != userID)
		}

		if shouldMarkAsRead {
			reply.IsRead = true
			reply.ReadBy = &userID
			reply.ReadAt = &nowTime
			db.Save(reply)
		}
	}

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

	db := database.GetDB()

	var ticket models.Ticket
	query := db.Where("id = ?", id)

	isAdmin := false
	if isAdminVal, exists := c.Get("is_admin"); exists {
		if isAdminBool, ok := isAdminVal.(bool); ok {
			isAdmin = isAdminBool
		} else if isAdminStr, ok := isAdminVal.(string); ok {
			isAdmin = isAdminStr == "true" || isAdminStr == "1"
		}
	}

	if !isAdmin {
		query = query.Where("user_id = ?", user.ID)
	}

	if err := query.First(&ticket).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "工单不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取工单失败", err)
		}
		return
	}

	reply := models.TicketReply{
		TicketID: ticket.ID,
		UserID:   user.ID,
		Content:  req.Content,
		IsAdmin:  fmt.Sprintf("%v", isAdmin),
		IsRead:   false, // 新回复默认未读
	}

	if err := db.Create(&reply).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "回复工单失败", err)
		return
	}

	if ticket.Status == "pending" {
		ticket.Status = "processing"
		db.Save(&ticket)
	}

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
	if err := db.First(&ticket, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "工单不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取工单失败", err)
		}
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
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新工单失败", err)
		return
	}

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

	if err := db.Where("id = ? AND user_id = ?", id, user.ID).First(&ticket).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "工单不存在或无权限", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取工单失败", err)
		}
		return
	}

	if ticket.Status == "closed" {
		utils.ErrorResponse(c, http.StatusBadRequest, "工单已关闭", nil)
		return
	}

	ticket.Status = "closed"
	now := time.Now()
	ticket.ClosedAt = &now

	if err := db.Save(&ticket).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "关闭工单失败", err)
		return
	}

	utils.CreateAuditLogSimple(c, "close_ticket", "ticket", ticket.ID, fmt.Sprintf("关闭工单: %s", ticket.Title))

	utils.SuccessResponse(c, http.StatusOK, "工单已关闭", ticket)
}
