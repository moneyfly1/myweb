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
	orderNo := utils.GenerateRechargeOrderNo(user.ID)
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

	if err := db.Create(&recharge).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建充值订单失败", err)
		return
	}

	var paymentURL string
	paymentMethod := req.PaymentMethod
	if paymentMethod == "" {
		paymentMethod = "alipay"
	}

	var paymentConfig models.PaymentConfig
	queryPayType := paymentMethod
	if strings.HasPrefix(paymentMethod, "yipay_") {
		if err := db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", "yipay", 1).First(&paymentConfig).Error; err == nil {
			queryPayType = "yipay"
		} else {
			if err := db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", paymentMethod, 1).First(&paymentConfig).Error; err != nil {
				utils.ErrorResponse(c, http.StatusBadRequest, "未找到启用的支付配置", nil)
				return
			}
		}
	} else {
		if err := db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", paymentMethod, 1).First(&paymentConfig).Error; err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "未找到启用的支付配置", nil)
			return
		}
	}

	if paymentConfig.Status == 1 {
		if paymentMethod == "alipay" {
			alipayService, err := payment.NewAlipayService(&paymentConfig)
			if err == nil {
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
		} else if queryPayType == "yipay" || strings.HasPrefix(paymentMethod, "yipay_") {
			yipayService, err := payment.NewYipayService(&paymentConfig)
			if err == nil {
				tempOrder := &models.Order{
					OrderNo: recharge.OrderNo,
					UserID:  user.ID,
					Amount:  recharge.Amount,
				}
				paymentType := "alipay"
				if strings.HasPrefix(paymentMethod, "yipay_") {
					paymentType = strings.TrimPrefix(paymentMethod, "yipay_")
				}
				paymentURL, _ = yipayService.CreatePayment(tempOrder, recharge.Amount, paymentType)
			}
		}

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

	utils.SuccessResponse(c, http.StatusOK, "", formattedRecords)
}

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
	}

	utils.SuccessResponse(c, http.StatusOK, "", formattedRecord)
}

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

	if record.Status == "pending" {
		timeSinceCreated := time.Since(record.CreatedAt)
		var shouldQuery bool
		if timeSinceCreated >= 3*time.Second && timeSinceCreated < 10*time.Second {
			shouldQuery = true
		} else if timeSinceCreated >= 10*time.Second && timeSinceCreated < 60*time.Second {
			shouldQuery = int(timeSinceCreated.Seconds())%5 < 2
		} else if timeSinceCreated >= 60*time.Second {
			shouldQuery = int(timeSinceCreated.Seconds())%30 < 2
		}

		if shouldQuery {
			paymentMethod := "alipay"
			if record.PaymentMethod.Valid {
				paymentMethod = record.PaymentMethod.String
			}

			var paymentConfig models.PaymentConfig
			if err := db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", paymentMethod, 1).First(&paymentConfig).Error; err == nil {
				if paymentConfig.PayType == "alipay" {
					alipayService, err := payment.NewAlipayService(&paymentConfig)
					if err == nil {
						queryResult, err := alipayService.QueryOrder(orderNo)
						if err == nil && queryResult != nil && queryResult.IsPaid() {
							err := utils.WithTransaction(db, func(tx *gorm.DB) error {
								var latestRecord models.RechargeRecord
								if err := tx.Where("order_no = ? AND status = ?", orderNo, "pending").First(&latestRecord).Error; err == nil {
									latestRecord.Status = "paid"
									latestRecord.PaidAt = database.NullTime(utils.GetBeijingTime())
									if queryResult.TradeNo != "" {
										latestRecord.PaymentTransactionID = database.NullString(queryResult.TradeNo)
									}
									if !latestRecord.PaymentMethod.Valid || latestRecord.PaymentMethod.String == "" {
										latestRecord.PaymentMethod = database.NullString(paymentMethod)
									}
									if err := tx.Save(&latestRecord).Error; err != nil {
										return err
									}
									var user models.User
									if err := tx.First(&user, latestRecord.UserID).Error; err == nil {
										user.Balance += latestRecord.Amount
										if err := tx.Save(&user).Error; err != nil {
											return err
										}
									}
								}
								return nil
							})

							if err == nil {
								db.Where("order_no = ?", orderNo).First(&record)
							}
						}
					}
				}
			}
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

func GetAdminRechargeRecords(c *gin.Context) {
	db := database.GetDB()
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

	keyword := c.Query("keyword")
	if keyword == "" {
		keyword = c.Query("search")
	}

	status := c.Query("status")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	countQuery := db.Model(&models.RechargeRecord{}).Joins("LEFT JOIN users ON recharge_records.user_id = users.id")
	findQuery := db.Model(&models.RechargeRecord{}).Preload("User")

	if keyword != "" {
		sanitizedKeyword := utils.SanitizeSearchKeyword(keyword)
		if sanitizedKeyword != "" {
			searchCondition := "recharge_records.order_no LIKE ? OR recharge_records.order_no LIKE ? OR users.username LIKE ? OR users.email LIKE ?"
			searchParams := []interface{}{
				"%" + sanitizedKeyword + "%",
				"%RCH%" + sanitizedKeyword + "%",
				"%" + sanitizedKeyword + "%",
				"%" + sanitizedKeyword + "%",
			}
			countQuery = countQuery.Where(searchCondition, searchParams...)
			findQuery = findQuery.Joins("LEFT JOIN users ON recharge_records.user_id = users.id").Where(searchCondition, searchParams...)
		}
	}

	if status != "" && status != "all" {
		statusCondition := "recharge_records.status = ?"
		countQuery = countQuery.Where(statusCondition, status)
		findQuery = findQuery.Where(statusCondition, status)
	}

	if startDate != "" {
		dateCondition := "DATE(recharge_records.created_at) >= ?"
		countQuery = countQuery.Where(dateCondition, startDate)
		findQuery = findQuery.Where(dateCondition, startDate)
	}
	if endDate != "" {
		dateCondition := "DATE(recharge_records.created_at) <= ?"
		countQuery = countQuery.Where(dateCondition, endDate)
		findQuery = findQuery.Where(dateCondition, endDate)
	}

	var total int64
	countQuery.Count(&total)

	offset := (page - 1) * size
	var records []models.RechargeRecord
	if err := findQuery.Order("recharge_records.created_at DESC").Offset(offset).Limit(size).Find(&records).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取充值记录失败", err)
		return
	}

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
