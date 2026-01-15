package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/services/geoip"
	"cboard-go/internal/services/payment"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateRecharge 创建充值订单
func CreateRecharge(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	var req struct {
		Amount        float64 `json:"amount" binding:"required,gt=0"`
		PaymentMethod string  `json:"payment_method"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()

	// 生成订单号
	orderNo := utils.GenerateRechargeOrderNo(user.ID)

	// 使用北京时间创建充值记录
	// 注意：无论支付URL是否生成成功，充值记录都应该被创建
	beijingTime := utils.GetBeijingTime()
	recharge := models.RechargeRecord{
		UserID:        user.ID,
		OrderNo:       orderNo,
		Amount:        req.Amount,
		Status:        "pending",
		PaymentMethod: database.NullString(req.PaymentMethod),
		CreatedAt:     beijingTime,
		UpdatedAt:     beijingTime,
	}

	// 先创建充值记录，确保记录一定存在
	if err := db.Create(&recharge).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建充值订单失败", err)
		return
	}

	// 调用支付接口生成支付链接
	// 即使支付URL生成失败，充值记录也已经创建，可以后续重试
	var paymentURL string
	paymentMethod := req.PaymentMethod
	if paymentMethod == "" {
		paymentMethod = "alipay"
	}

	var paymentConfig models.PaymentConfig
	// 查找对应类型的支付配置（不区分大小写）
	if err := db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", paymentMethod, 1).First(&paymentConfig).Error; err == nil {
		if paymentMethod == "alipay" {
			alipayService, err := payment.NewAlipayService(&paymentConfig)
			if err == nil {
				// 创建临时订单用于充值
				tempOrder := &models.Order{
					OrderNo: recharge.OrderNo,
					UserID:  user.ID,
					Amount:  recharge.Amount,
				}
				paymentURL, _ = alipayService.CreatePayment(tempOrder, recharge.Amount)
			}
		} else if paymentMethod == "wechat" {
			wechatService, err := payment.NewWechatService(&paymentConfig)
			if err == nil {
				tempOrder := &models.Order{
					OrderNo: recharge.OrderNo,
					UserID:  user.ID,
					Amount:  recharge.Amount,
				}
				paymentURL, _ = wechatService.CreatePayment(tempOrder, recharge.Amount)
			}
		}

		// 如果支付URL生成成功，更新充值记录
		if paymentURL != "" {
			recharge.PaymentURL = database.NullString(paymentURL)
			db.Save(&recharge)
		}
	}

	utils.SuccessResponse(c, http.StatusCreated, "", gin.H{
		"id":          recharge.ID,
		"order_no":    recharge.OrderNo,
		"amount":      recharge.Amount,
		"status":      recharge.Status,
		"payment_url": paymentURL,
	})
}

// GetRechargeRecords 获取充值记录
func GetRechargeRecords(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	db := database.GetDB()
	var records []models.RechargeRecord
	if err := db.Where("user_id = ?", user.ID).Order("created_at DESC").Find(&records).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取充值记录失败", err)
		return
	}

	// 格式化充值记录，确保支付方式等字段正确序列化
	formattedRecords := make([]gin.H, 0, len(records))
	formatIP := func(ip string) string {
		if ip == "" {
			return "-"
		}
		if ip == "::1" {
			return "127.0.0.1"
		}
		if strings.HasPrefix(ip, "::ffff:") {
			return strings.TrimPrefix(ip, "::ffff:")
		}
		return ip
	}
	for _, record := range records {
		ipValue := utils.GetNullStringValue(record.IPAddress)
		var ipStr string
		if ipValue != nil {
			ipStr = ipValue.(string)
		}
		ipAddress := formatIP(ipStr)
		// 使用GeoIP解析地理位置
		location := ""
		if ipAddress != "" && ipAddress != "-" && geoip.IsEnabled() {
			locationStr := geoip.GetLocationString(ipAddress)
			if locationStr.Valid {
				location = locationStr.String
			}
		}

		formattedRecords = append(formattedRecords, gin.H{
			"id":                     record.ID,
			"user_id":                record.UserID,
			"order_no":               record.OrderNo,
			"amount":                 record.Amount,
			"status":                 record.Status,
			"payment_method":         utils.GetNullStringValue(record.PaymentMethod),
			"payment_transaction_id": utils.GetNullStringValue(record.PaymentTransactionID),
			"payment_qr_code":        utils.GetNullStringValue(record.PaymentQRCode),
			"payment_url":            utils.GetNullStringValue(record.PaymentURL),
			"ip_address":             ipAddress,
			"location":               location, // 添加归属地信息
			"user_agent":             utils.GetNullStringValue(record.UserAgent),
			"paid_at": func() interface{} {
				if record.PaidAt.Valid {
					return record.PaidAt.Time.Format("2006-01-02 15:04:05")
				}
				return nil
			}(),
			"created_at": record.CreatedAt.Format("2006-01-02 15:04:05"),
			"updated_at": record.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", formattedRecords)
}

// GetRechargeRecord 获取单个充值记录
func GetRechargeRecord(c *gin.Context) {
	id := c.Param("id")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	db := database.GetDB()
	var record models.RechargeRecord
	if err := db.Where("id = ? AND user_id = ?", id, user.ID).First(&record).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "充值记录不存在", err)
		return
	}

	// 格式化充值记录，确保支付方式等字段正确序列化
	formatIP := func(ip string) string {
		if ip == "" {
			return "-"
		}
		if ip == "::1" {
			return "127.0.0.1"
		}
		if strings.HasPrefix(ip, "::ffff:") {
			return strings.TrimPrefix(ip, "::ffff:")
		}
		return ip
	}
	ipValue := utils.GetNullStringValue(record.IPAddress)
	var ipStr string
	if ipValue != nil {
		ipStr = ipValue.(string)
	}
	ipAddress := formatIP(ipStr)
	// 使用GeoIP解析地理位置
	location := ""
	if ipAddress != "" && ipAddress != "-" && geoip.IsEnabled() {
		locationStr := geoip.GetLocationString(ipAddress)
		if locationStr.Valid {
			location = locationStr.String
		}
	}

	formattedRecord := gin.H{
		"id":                     record.ID,
		"user_id":                record.UserID,
		"order_no":               record.OrderNo,
		"amount":                 record.Amount,
		"status":                 record.Status,
		"payment_method":         utils.GetNullStringValue(record.PaymentMethod),
		"payment_transaction_id": utils.GetNullStringValue(record.PaymentTransactionID),
		"payment_qr_code":        utils.GetNullStringValue(record.PaymentQRCode),
		"payment_url":            utils.GetNullStringValue(record.PaymentURL),
		"ip_address":             ipAddress,
		"location":               location, // 添加归属地信息
		"user_agent":             utils.GetNullStringValue(record.UserAgent),
		"paid_at": func() interface{} {
			if record.PaidAt.Valid {
				return record.PaidAt.Time.Format("2006-01-02 15:04:05")
			}
			return nil
		}(),
		"created_at": record.CreatedAt.Format("2006-01-02 15:04:05"),
		"updated_at": record.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	utils.SuccessResponse(c, http.StatusOK, "", formattedRecord)
}

// GetRechargeStatusByNo 通过订单号获取充值状态（支持主动查询支付状态）
func GetRechargeStatusByNo(c *gin.Context) {
	orderNo := c.Param("orderNo")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	db := database.GetDB()
	var record models.RechargeRecord
	if err := db.Where("order_no = ? AND user_id = ?", orderNo, user.ID).First(&record).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "充值记录不存在", err)
		return
	}

	// 如果充值记录是 pending 状态，且创建时间超过3秒，主动查询支付状态
	// 优化查询间隔：每5秒查询一次支付宝状态，确保及时回调
	if record.Status == "pending" {
		// 检查充值记录创建时间，避免频繁查询
		timeSinceCreated := time.Since(record.CreatedAt)
		// 使用秒数取模，只在5秒的倍数时查询（例如：5秒、10秒、15秒...）
		// 这样可以更快地检测到支付成功，同时避免过于频繁的查询
		shouldQuery := timeSinceCreated > 3*time.Second && int(timeSinceCreated.Seconds())%5 < 2
		if shouldQuery {
			// 获取支付方式
			paymentMethod := "alipay"
			if record.PaymentMethod.Valid {
				paymentMethod = record.PaymentMethod.String
			}

			// 获取支付配置
			var paymentConfig models.PaymentConfig
			if err := db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", paymentMethod, 1).First(&paymentConfig).Error; err == nil {
				// 根据支付方式查询状态
				if paymentConfig.PayType == "alipay" {
					alipayService, err := payment.NewAlipayService(&paymentConfig)
					if err == nil {
						queryResult, err := alipayService.QueryOrder(orderNo)
						if err == nil && queryResult != nil && queryResult.IsPaid() {
							// 支付成功，更新充值记录状态（模拟回调处理）
							utils.LogInfo("GetRechargeStatusByNo: payment query success, updating recharge - order_no=%s, trade_no=%s",
								orderNo, queryResult.TradeNo)

							// 使用事务更新充值记录状态
							err := utils.WithTransaction(db, func(tx *gorm.DB) error {
								// 重新加载充值记录（获取最新状态，避免重复处理）
								var latestRecord models.RechargeRecord
								if err := tx.Where("order_no = ? AND status = ?", orderNo, "pending").First(&latestRecord).Error; err == nil {
									latestRecord.Status = "paid"
									latestRecord.PaidAt = database.NullTime(utils.GetBeijingTime())
									if queryResult.TradeNo != "" {
										latestRecord.PaymentTransactionID = database.NullString(queryResult.TradeNo)
									}
									// 确保支付方式被正确设置
									if !latestRecord.PaymentMethod.Valid || latestRecord.PaymentMethod.String == "" {
										latestRecord.PaymentMethod = database.NullString(paymentMethod)
									}
									if err := tx.Save(&latestRecord).Error; err != nil {
										return err
									}

									// 更新用户余额
									var user models.User
									if err := tx.First(&user, latestRecord.UserID).Error; err == nil {
										oldBalance := user.Balance
										user.Balance += latestRecord.Amount
										if err := tx.Save(&user).Error; err != nil {
											utils.LogError("GetRechargeStatusByNo: failed to update user balance", err, map[string]interface{}{
												"order_no": orderNo,
												"user_id":  user.ID,
											})
											return err
										}
										// 记录充值成功日志
										utils.LogInfo("GetRechargeStatusByNo: 充值成功 - order_no=%s, user_id=%d, amount=%.2f, old_balance=%.2f, new_balance=%.2f",
											orderNo, user.ID, latestRecord.Amount, oldBalance, user.Balance)
									}
								}
								return nil
							})

							if err != nil {
								utils.LogError("GetRechargeStatusByNo: failed to update recharge transaction", err, map[string]interface{}{
									"order_no": orderNo,
								})
							} else {
								// 重新加载充值记录以返回最新状态
								db.Where("order_no = ?", orderNo).First(&record)
							}
						}
					}
				}
			}
		}
	}

	// 格式化充值记录，确保支付方式等字段正确序列化
	formattedRecord := gin.H{
		"id":                     record.ID,
		"user_id":                record.UserID,
		"order_no":               record.OrderNo,
		"amount":                 record.Amount,
		"status":                 record.Status,
		"payment_method":         utils.GetNullStringValue(record.PaymentMethod),
		"payment_transaction_id": utils.GetNullStringValue(record.PaymentTransactionID),
		"paid_at": func() interface{} {
			if record.PaidAt.Valid {
				return record.PaidAt.Time.Format("2006-01-02 15:04:05")
			}
			return nil
		}(),
		"created_at": record.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	utils.SuccessResponse(c, http.StatusOK, "", formattedRecord)
}

// CancelRecharge 取消充值订单
func CancelRecharge(c *gin.Context) {
	id := c.Param("id")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	db := database.GetDB()
	var record models.RechargeRecord
	if err := db.Where("id = ? AND user_id = ?", id, user.ID).First(&record).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "充值记录不存在", err)
		return
	}

	if record.Status != "pending" {
		utils.ErrorResponse(c, http.StatusBadRequest, "只能取消待支付的充值订单", nil)
		return
	}

	record.Status = "cancelled"
	if err := db.Save(&record).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "取消充值订单失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "充值订单已取消", record)
}

// GetAdminRechargeRecords 管理员获取充值记录列表
func GetAdminRechargeRecords(c *gin.Context) {
	db := database.GetDB()

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
	if size < 1 || size > 100 {
		size = 20
	}

	// 搜索参数
	keyword := c.Query("keyword")
	if keyword == "" {
		keyword = c.Query("search")
	}

	// 状态筛选
	status := c.Query("status")

	// 日期范围筛选
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// 构建查询（使用JOIN提高性能）
	query := db.Model(&models.RechargeRecord{}).Joins("LEFT JOIN users ON recharge_records.user_id = users.id").Preload("User")

	// 搜索关键词
	if keyword != "" {
		sanitizedKeyword := utils.SanitizeSearchKeyword(keyword)
		if sanitizedKeyword != "" {
			// 支持通过订单号、用户名、邮箱查询
			// 如果关键词是纯数字，可能是时间戳，也尝试匹配订单号中的时间戳部分
			query = query.Where(
				"recharge_records.order_no LIKE ? OR recharge_records.order_no LIKE ? OR users.username LIKE ? OR users.email LIKE ?",
				"%"+sanitizedKeyword+"%", // 完整订单号匹配
				"%RCH%"+sanitizedKeyword+"%", // 时间戳匹配（充值订单号格式：RCH + timestamp + userID + 随机数）
				"%"+sanitizedKeyword+"%", // 用户名匹配
				"%"+sanitizedKeyword+"%", // 邮箱匹配
			)
		}
	}

	// 状态筛选
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	// 日期范围筛选
	if startDate != "" {
		query = query.Where("DATE(created_at) >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("DATE(created_at) <= ?", endDate)
	}

	// 计算总数
	var total int64
	query.Count(&total)

	// 分页查询
	offset := (page - 1) * size
	var records []models.RechargeRecord
	if err := query.Order("created_at DESC").Offset(offset).Limit(size).Find(&records).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取充值记录失败", err)
		return
	}

	// 格式化充值记录
	formattedRecords := make([]gin.H, 0, len(records))
	formatIP := func(ip string) string {
		if ip == "" {
			return "-"
		}
		if ip == "::1" {
			return "127.0.0.1"
		}
		if strings.HasPrefix(ip, "::ffff:") {
			return strings.TrimPrefix(ip, "::ffff:")
		}
		return ip
	}
	for _, record := range records {
		ipValue := utils.GetNullStringValue(record.IPAddress)
		var ipStr string
		if ipValue != nil {
			ipStr = ipValue.(string)
		}
		ipAddress := formatIP(ipStr)
		// 使用GeoIP解析地理位置
		location := ""
		if ipAddress != "" && ipAddress != "-" && geoip.IsEnabled() {
			locationStr := geoip.GetLocationString(ipAddress)
			if locationStr.Valid {
				location = locationStr.String
			}
		}

		userInfo := gin.H{}
		if record.User.ID > 0 {
			userInfo = gin.H{
				"id":       record.User.ID,
				"username": record.User.Username,
				"email":    record.User.Email,
			}
		}

		formattedRecords = append(formattedRecords, gin.H{
			"id":                     record.ID,
			"user_id":                record.UserID,
			"user":                   userInfo,
			"order_no":               record.OrderNo,
			"amount":                 record.Amount,
			"status":                 record.Status,
			"payment_method":         utils.GetNullStringValue(record.PaymentMethod),
			"payment_transaction_id": utils.GetNullStringValue(record.PaymentTransactionID),
			"payment_qr_code":        utils.GetNullStringValue(record.PaymentQRCode),
			"payment_url":            utils.GetNullStringValue(record.PaymentURL),
			"ip_address":             ipAddress,
			"location":               location,
			"user_agent":             utils.GetNullStringValue(record.UserAgent),
			"paid_at": func() interface{} {
				if record.PaidAt.Valid {
					return record.PaidAt.Time.Format("2006-01-02 15:04:05")
				}
				return nil
			}(),
			"created_at": record.CreatedAt.Format("2006-01-02 15:04:05"),
			"updated_at": record.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"recharges": formattedRecords,
		"total":     total,
		"page":      page,
		"size":      size,
	})
}
