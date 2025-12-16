package handlers

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

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
	query := db.Model(&models.InviteCode{}).Preload("User")

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
		query = query.Where("user_id IN (SELECT id FROM users WHERE username LIKE ? OR email LIKE ?)", "%"+userQuery+"%", "%"+userQuery+"%")
	}
	if code := c.Query("code"); code != "" {
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
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&inviteCodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取邀请码列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"invite_codes": inviteCodes,
			"total":        total,
			"page":         page,
			"size":         size,
		},
	})
}

// GetAdminInviteRelations 管理员获取邀请关系列表
func GetAdminInviteRelations(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.InviteRelation{}).Preload("Inviter").Preload("Invitee")

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
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取邀请关系列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"relations": relations,
			"total":     total,
			"page":      page,
			"size":      size,
		},
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
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
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&tickets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取工单列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"tickets": tickets,
			"total":   total,
			"page":    page,
			"size":    size,
		},
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetAdminTicket 管理员获取单个工单详情
func GetAdminTicket(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()

	var ticket models.Ticket
	if err := db.Preload("User").Preload("Assignee").Preload("Replies").First(&ticket, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "工单不存在",
		})
		return
	}

	// 计算回复数量
	var repliesCount int64
	db.Model(&models.TicketReply{}).Where("ticket_id = ?", ticket.ID).Count(&repliesCount)

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
		"replies":       ticket.Replies,
		"replies_count": repliesCount,
		"created_at":    ticket.CreatedAt.Format("2006-01-02 15:04:05"),
		"updated_at":    ticket.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"ticket": ticketData, // 前端期望嵌套在 ticket 字段中
		},
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
		query = query.Where("code LIKE ? OR name LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取优惠券列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"coupons": coupons,
			"total":   total,
			"page":    page,
			"size":    size,
		},
	})
}

// GetAdminUserLevels 管理员获取用户等级列表
func GetAdminUserLevels(c *gin.Context) {
	db := database.GetDB()
	var userLevels []models.UserLevel
	if err := db.Order("level_order ASC").Find(&userLevels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取用户等级列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    userLevels,
	})
}

// GetUserLevel 获取用户等级
func GetUserLevel(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	db := database.GetDB()
	var userLevel models.UserLevel
	if user.UserLevelID.Valid {
		if err := db.First(&userLevel, user.UserLevelID.Int64).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    nil,
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    userLevel,
	})
}

// GetUserSubscription 获取用户订阅
func GetUserSubscription(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	db := database.GetDB()
	var subscription models.Subscription
	if err := db.Where("user_id = ?", user.ID).First(&subscription).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    nil,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取订阅失败",
		})
		return
	}

	// buildBaseURL 根据请求构造带协议的基础 URL
	buildBaseURL := func(c *gin.Context) string {
		scheme := "http"
		if proto := c.Request.Header.Get("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		} else if c.Request.TLS != nil {
			scheme = "https"
		}
		host := c.Request.Host
		return fmt.Sprintf("%s://%s", scheme, host)
	}

	// 生成订阅地址（与原始 Python 代码保持一致）
	baseURL := buildBaseURL(c)
	timestamp := fmt.Sprintf("%d", utils.GetBeijingTime().Unix())
	clashURL := fmt.Sprintf("%s/api/v1/subscriptions/clash/%s?t=%s", baseURL, subscription.SubscriptionURL, timestamp)
	ssrURL := fmt.Sprintf("%s/api/v1/subscriptions/ssr/%s?t=%s", baseURL, subscription.SubscriptionURL, timestamp)
	v2rayURL := ssrURL

	// 计算到期时间
	expiryDate := "未设置"
	if !subscription.ExpireTime.IsZero() {
		expiryDate = subscription.ExpireTime.Format("2006-01-02 15:04:05")
	}

	// 生成二维码 URL（sub://格式，包含到期时间）
	encodedURL := base64.StdEncoding.EncodeToString([]byte(ssrURL))
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
		diff := subscription.ExpireTime.Sub(now)
		if diff > 0 {
			remainingDays = int(diff.Hours() / 24)
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
		"ssr_url":          ssrURL,
		"v2ray_url":        v2rayURL,
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    subscriptionData,
	})
}

// GetUserTheme 获取用户主题
func GetUserTheme(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"theme":    user.Theme,
			"language": user.Language,
		},
	})
}

// UpdateUserTheme 更新用户主题
func UpdateUserTheme(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	var req struct {
		Theme    string `json:"theme"`
		Language string `json:"language"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新主题失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "主题更新成功",
		"data": gin.H{
			"theme":    user.Theme,
			"language": user.Language,
		},
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
		query = query.Where("status = ?", status)
	}
	if email := c.Query("email"); email != "" {
		query = query.Where("to_email LIKE ?", "%"+email+"%")
	}

	var total int64
	query.Count(&total)

	var emails []models.EmailQueue
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&emails).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取邮件队列失败",
		})
		return
	}

	// 计算总页数
	pages := (total + int64(size) - 1) / int64(size)
	if pages < 1 {
		pages = 1
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"emails": emails,
			"total":  total,
			"page":   page,
			"size":   size,
			"pages":  pages,
		},
	})
}

// GetEmailQueueStatistics 获取邮件队列统计
func GetEmailQueueStatistics(c *gin.Context) {
	db := database.GetDB()

	var stats struct {
		TotalEmails   int64 `json:"total_emails"`
		PendingEmails int64 `json:"pending_emails"`
		SentEmails    int64 `json:"sent_emails"`
		FailedEmails  int64 `json:"failed_emails"`
	}

	db.Model(&models.EmailQueue{}).Count(&stats.TotalEmails)
	db.Model(&models.EmailQueue{}).Where("status = ?", "pending").Count(&stats.PendingEmails)
	db.Model(&models.EmailQueue{}).Where("status = ?", "sent").Count(&stats.SentEmails)
	db.Model(&models.EmailQueue{}).Where("status = ?", "failed").Count(&stats.FailedEmails)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetAdminSystemConfig 获取系统配置
func GetAdminSystemConfig(c *gin.Context) {
	db := database.GetDB()
	var configs []models.SystemConfig
	db.Order("category ASC, sort_order ASC").Find(&configs)

	configMap := make(map[string]interface{})
	for _, config := range configs {
		if configMap[config.Category] == nil {
			configMap[config.Category] = make(map[string]interface{})
		}
		configMap[config.Category].(map[string]interface{})[config.Key] = config.Value
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    configMap,
	})
}

// GetAdminClashConfig 获取 Clash 配置
func GetAdminClashConfig(c *gin.Context) {
	db := database.GetDB()
	var config models.SystemConfig
	if err := db.Where("category = ? AND key = ?", "clash", "config").First(&config).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    config.Value,
	})
}

// GetAdminV2RayConfig 获取 V2Ray 配置
func GetAdminV2RayConfig(c *gin.Context) {
	db := database.GetDB()
	var config models.SystemConfig
	if err := db.Where("category = ? AND key = ?", "v2ray", "config").First(&config).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    config.Value,
	})
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    configMap,
	})
}

// GetAdminClashConfigInvalid 获取无效的 Clash 配置
func GetAdminClashConfigInvalid(c *gin.Context) {
	// 返回空数组，表示没有无效配置
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    []interface{}{},
	})
}

// GetAdminV2RayConfigInvalid 获取无效的 V2Ray 配置
func GetAdminV2RayConfigInvalid(c *gin.Context) {
	// 返回空数组，表示没有无效配置
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    []interface{}{},
	})
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    configMap,
	})
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取支付配置列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":           paymentConfigs, // 前端期望 items 字段
			"payment_configs": paymentConfigs, // 兼容字段
			"total":           total,
			"page":            page,
			"size":            size,
		},
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
		ORDER BY date DESC
	`, days).Rows()

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var trend UserTrend
			rows.Scan(&trend.Date, &trend.UserCount)
			trends = append(trends, trend)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    trends,
	})
}

// GetRevenueTrend 获取收入趋势
func GetRevenueTrend(c *gin.Context) {
	GetRevenueChart(c)
}

// UpdateClashConfig 更新 Clash 配置
func UpdateClashConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	for key, value := range req {
		var config models.SystemConfig
		if err := db.Where("key = ? AND category = ?", key, "clash").First(&config).Error; err != nil {
			config = models.SystemConfig{
				Key:      key,
				Category: "clash",
				Value:    fmt.Sprintf("%v", value),
			}
			db.Create(&config)
		} else {
			config.Value = fmt.Sprintf("%v", value)
			db.Save(&config)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Clash 配置已更新",
	})
}

// UpdateV2RayConfig 更新 V2Ray 配置
func UpdateV2RayConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	for key, value := range req {
		var config models.SystemConfig
		if err := db.Where("key = ? AND category = ?", key, "v2ray").First(&config).Error; err != nil {
			config = models.SystemConfig{
				Key:      key,
				Category: "v2ray",
				Value:    fmt.Sprintf("%v", value),
			}
			db.Create(&config)
		} else {
			config.Value = fmt.Sprintf("%v", value)
			db.Save(&config)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "V2Ray 配置已更新",
	})
}

// UpdateEmailConfig 更新邮件配置
func UpdateEmailConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	for key, value := range req {
		var config models.SystemConfig
		if err := db.Where("key = ? AND category = ?", key, "email").First(&config).Error; err != nil {
			config = models.SystemConfig{
				Key:      key,
				Category: "email",
				Value:    fmt.Sprintf("%v", value),
			}
			db.Create(&config)
		} else {
			config.Value = fmt.Sprintf("%v", value)
			db.Save(&config)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "邮件配置已更新",
	})
}

// MarkClashConfigInvalid 标记 Clash 配置无效
func MarkClashConfigInvalid(c *gin.Context) {
	var req struct {
		ConfigID uint   `json:"config_id" binding:"required"`
		Reason   string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	var config models.SystemConfig
	if err := db.First(&config, req.ConfigID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "配置不存在",
		})
		return
	}

	// 更新配置状态为无效
	config.Value = "invalid"
	if req.Reason != "" {
		config.Description = req.Reason
	}

	if err := db.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "标记配置失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "配置已标记为无效",
	})
}

// MarkV2RayConfigInvalid 标记 V2Ray 配置无效
func MarkV2RayConfigInvalid(c *gin.Context) {
	var req struct {
		ConfigID uint   `json:"config_id" binding:"required"`
		Reason   string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	var config models.SystemConfig
	if err := db.First(&config, req.ConfigID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "配置不存在",
		})
		return
	}

	// 更新配置状态为无效
	config.Value = "invalid"
	if req.Reason != "" {
		config.Description = req.Reason
	}

	if err := db.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "标记配置失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "配置已标记为无效",
	})
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
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": fmt.Sprintf("请求参数错误: %v", err),
		})
		return
	}

	// 构建基础 URL
	buildBaseURL := func(c *gin.Context) string {
		scheme := "http"
		if proto := c.Request.Header.Get("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		} else if c.Request.TLS != nil {
			scheme = "https"
		}
		host := c.Request.Host
		return fmt.Sprintf("%s://%s", scheme, host)
	}
	baseURL := buildBaseURL(c)

	// 如果没有提供回调地址，自动生成
	if req.NotifyURL == "" {
		notifySuffix := "alipay"
		if req.PayType == "wechat" {
			notifySuffix = "wechat"
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": fmt.Sprintf("创建支付配置失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "支付配置创建成功",
		"data":    paymentConfig,
	})
}
