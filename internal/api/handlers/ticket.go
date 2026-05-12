package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

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

func isValidTicketType(ticketType string) bool {
	switch ticketType {
	case "technical", "billing", "account", "other":
		return true
	default:
		return false
	}
}

func isValidTicketPriority(priority string) bool {
	switch priority {
	case "low", "normal", "high", "urgent":
		return true
	default:
		return false
	}
}

func isValidTicketStatus(status string) bool {
	switch status {
	case "pending", "processing", "resolved", "closed", "cancelled":
		return true
	default:
		return false
	}
}

type ticketReadAtResult struct {
	TicketID uint
	ReadAt   time.Time
}

func getTicketReadAtMap(db *gorm.DB, ticketIDs []uint, userID uint) map[uint]time.Time {
	readMap := make(map[uint]time.Time)
	if len(ticketIDs) == 0 || userID == 0 {
		return readMap
	}

	var reads []ticketReadAtResult
	db.Model(&models.TicketRead{}).
		Select("ticket_id, MAX(read_at) AS read_at").
		Where("ticket_id IN ? AND user_id = ?", ticketIDs, userID).
		Group("ticket_id").
		Scan(&reads)
	for _, read := range reads {
		readMap[read.TicketID] = read.ReadAt
	}
	return readMap
}

func ticketReadJoin(userID uint) string {
	return fmt.Sprintf(
		"LEFT JOIN (SELECT ticket_id, MAX(read_at) AS read_at FROM ticket_reads WHERE user_id = %d GROUP BY ticket_id) ticket_read_state ON ticket_read_state.ticket_id = ticket_replies.ticket_id",
		userID,
	)
}

// processTicketReadStatus 处理查看工单详情时的已读状态更新。
// ticket_reads 是按用户维度的真实已读依据，ticket_replies 的 read 字段仅保留兼容旧逻辑。
func processTicketReadStatus(db *gorm.DB, ticket *models.Ticket, userID uint, isAdmin bool) {
	nowTime := utils.GetBeijingTime()

	db.Model(&models.TicketReply{}).
		Where("ticket_id = ? AND is_admin = ? AND (is_read = ? OR read_by != ? OR read_by IS NULL)", ticket.ID, !isAdmin, false, userID).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_by": userID,
			"read_at": nowTime,
		})

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
				notificationService := notification.NewNotificationService()
				if notification.ShouldSendCustomerNotificationToUser(&ticketOwner, "ticket_reply", notification.ChannelSystem) {
					content := fmt.Sprintf("您的工单「%s」有新的管理员回复。", ticket.Title)
					if err := notificationService.CreateUserSystemNotification(&ticketOwner, "ticket_reply", "工单新回复", content); err != nil {
						utils.LogErrorMsg("创建工单回复站内通知失败: ticket=%s, user_id=%d, error=%v", ticket.TicketNo, ticketOwner.ID, err)
					}
				}
				emailService := email.NewEmailService()
				templateBuilder := email.NewEmailTemplateBuilder()
				emailContent := templateBuilder.GetAdminReplyNotificationTemplate(ticket.TicketNo, ticket.Title, replyContent)
				if notification.ShouldSendCustomerNotificationToUser(&ticketOwner, "ticket_reply", notification.ChannelEmail) {
					if err := emailService.QueueEmail(ticketOwner.Email, "您的工单有新回复", emailContent, "ticket_reply"); err != nil {
						utils.LogErrorMsg("发送工单回复邮件失败: ticket=%s, email=%s, error=%v", ticket.TicketNo, ticketOwner.Email, err)
					}
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
			Joins(ticketReadJoin(user.ID)).
			Where("tickets.user_id = ? AND ticket_replies.is_admin = ? AND (ticket_read_state.read_at IS NULL OR ticket_replies.created_at > ticket_read_state.read_at)",
				user.ID, true).
			Count(&totalUnread)
	} else {
		db.Model(&models.Ticket{}).
			Joins("LEFT JOIN ticket_replies ON ticket_replies.ticket_id = tickets.id AND ticket_replies.is_admin = ?", false).
			Joins("LEFT JOIN (SELECT ticket_id, MAX(read_at) AS read_at FROM ticket_reads WHERE user_id = ? GROUP BY ticket_id) ticket_read_state ON ticket_read_state.ticket_id = tickets.id", user.ID).
			Where("ticket_read_state.read_at IS NULL OR ticket_replies.created_at > ticket_read_state.read_at").
			Distinct("tickets.id").
			Count(&totalUnread)
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
	if !isValidTicketType(req.Type) {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的工单类型", nil)
		return
	}
	if !isValidTicketPriority(req.Priority) {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的工单优先级", nil)
		return
	}

	title := truncateString(utils.SanitizeInput(req.Title), 200)
	content := truncateString(utils.SanitizeInput(req.Content), 5000)
	if title == "" || content == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "工单标题和内容不能为空", nil)
		return
	}

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
	query := db.Model(&models.Ticket{}).Preload("User")

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

		// 统一处理未读回复统计逻辑：查询来自对方且晚于当前用户最后阅读时间的回复。
		db.Model(&models.TicketReply{}).
			Select("ticket_id, COUNT(*) as count").
			Joins(ticketReadJoin(user.ID)).
			Where("ticket_replies.ticket_id IN ? AND ticket_replies.is_admin = ? AND (ticket_read_state.read_at IS NULL OR ticket_replies.created_at > ticket_read_state.read_at)",
				ticketIDs, !isAdmin).
			Group("ticket_id").
			Scan(&unreadRepliesStats)
	}

	ticketReadMap := getTicketReadAtMap(db, ticketIDs, user.ID)

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
			_, hasRead := ticketReadMap[ticket.ID]
			hasUnread = !hasRead || hasUnread
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

	query := db.Preload("User").
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
	if user.ID > 0 {
		processTicketReadStatus(db, &ticket, user.ID, isAdmin)
	}

	// 2. 梳理回复列表。打开详情即标记已读，响应中不再显示“新”。
	replies := make([]gin.H, 0, len(ticket.Replies))
	for _, reply := range ticket.Replies {
		replies = append(replies, gin.H{
			"id":             reply.ID,
			"ticket_id":      reply.TicketID,
			"user_id":        reply.UserID,
			"content":        reply.Content,
			"is_admin":       reply.IsAdmin,
			"is_admin_reply": reply.IsAdmin, // 兼容原代码字段
			"created_at":     utils.FormatBeijingTime(reply.CreatedAt),
			"is_unread":      false,
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

	content := truncateString(utils.SanitizeInput(req.Content), 5000)
	if content == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "回复内容不能为空", nil)
		return
	}
	reply := models.TicketReply{
		TicketID: ticket.ID,
		UserID:   user.ID,
		Content:  content,
		IsAdmin:  isAdmin,
		IsRead:   false,
	}

	if err := db.Create(&reply).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "回复工单失败", err)
		return
	}

	shouldSaveTicket := false
	if isAdmin && ticket.Status == "pending" {
		ticket.Status = "processing"
		shouldSaveTicket = true
	}
	if !isAdmin && (ticket.Status == "pending" || ticket.Status == "resolved" || ticket.Status == "closed") {
		ticket.Status = "processing"
		ticket.ResolvedAt = nil
		ticket.ClosedAt = nil
		shouldSaveTicket = true
	}
	if shouldSaveTicket {
		db.Save(&ticket)
	}

	// 异步执行通知机制
	asyncNotifyTicketReply(db, &ticket, user, content, isAdmin)

	utils.SuccessResponse(c, http.StatusCreated, "", reply)
}

func UpdateTicketStatus(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status     string `json:"status"`
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

	if req.Status == "" && req.AdminNotes == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "没有需要更新的内容", nil)
		return
	}

	oldStatus := ticket.Status
	if req.Status != "" {
		if !isValidTicketStatus(req.Status) {
			utils.ErrorResponse(c, http.StatusBadRequest, "无效的工单状态", nil)
			return
		}
		ticket.Status = req.Status
	}
	if req.AdminNotes != "" {
		notes := truncateString(utils.SanitizeInput(req.AdminNotes), 5000)
		ticket.AdminNotes = &notes
	}

	// 统一获取时间设置完成/关闭时间
	now := utils.GetBeijingTime()
	if req.Status == "resolved" {
		ticket.ResolvedAt = &now
		ticket.ClosedAt = nil
	} else if req.Status == "closed" {
		ticket.ClosedAt = &now
	} else if req.Status == "pending" || req.Status == "processing" {
		ticket.ResolvedAt = nil
		ticket.ClosedAt = nil
	}

	if err := db.Save(&ticket).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新工单失败", err)
		return
	}

	utils.CreateAuditLogSimple(c, "update_ticket_status", "ticket", ticket.ID, fmt.Sprintf("管理员操作: 更新工单 %s 状态 %s -> %s", ticket.Title, oldStatus, ticket.Status))
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
