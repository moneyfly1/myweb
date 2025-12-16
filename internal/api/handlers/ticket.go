package handlers

import (
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

// CreateTicket 创建工单
func CreateTicket(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	var req struct {
		Title    string `json:"title" binding:"required"`
		Content  string `json:"content" binding:"required"`
		Type     string `json:"type"`
		Priority string `json:"priority"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建工单失败",
		})
		return
	}

	// 记录创建工单审计日志
	utils.SetResponseStatus(c, http.StatusCreated)
	utils.CreateAuditLogSimple(c, "create_ticket", "ticket", ticket.ID, fmt.Sprintf("创建工单: %s", ticket.Title))

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    ticket,
	})
}

// GetTickets 获取工单列表
func GetTickets(c *gin.Context) {
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
	var tickets []models.Ticket
	query := db.Preload("User").Preload("Assignee")

	if !isAdmin {
		query = query.Where("user_id = ?", user.ID)
	}

	if err := query.Order("created_at DESC").Find(&tickets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取工单列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tickets,
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    ticket,
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

	// 创建回复
	reply := models.TicketReply{
		TicketID: ticket.ID,
		UserID:   user.ID,
		Content:  req.Content,
		IsAdmin:  fmt.Sprintf("%v", isAdmin),
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
