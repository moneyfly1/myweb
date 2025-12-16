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
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
		Type    string `json:"type"`
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

	ticket := models.Ticket{
		TicketNo: ticketNo,
		UserID:   user.ID,
		Title:    req.Title,
		Content:  req.Content,
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

	isAdmin, _ := c.Get("is_admin")
	admin := isAdmin.(bool)

	db := database.GetDB()
	var tickets []models.Ticket
	query := db.Preload("User").Preload("Assignee")

	if !admin {
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

	isAdmin, _ := c.Get("is_admin")
	admin := isAdmin.(bool)

	db := database.GetDB()
	var ticket models.Ticket
	query := db.Preload("User").Preload("Assignee").Preload("Replies").Preload("Attachments").Where("id = ?", id)

	if !admin {
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
	isAdmin, _ := c.Get("is_admin")
	if !isAdmin.(bool) {
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
		IsAdmin:  fmt.Sprintf("%v", isAdmin.(bool)),
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
		ticket.AssignedTo = database.NullInt64(int64(req.AssignedTo))
	}
	if req.AdminNotes != "" {
		ticket.AdminNotes = database.NullString(req.AdminNotes)
	}

	if req.Status == "resolved" {
		now := time.Now()
		ticket.ResolvedAt = database.NullTime(now)
	} else if req.Status == "closed" {
		now := time.Now()
		ticket.ClosedAt = database.NullTime(now)
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

