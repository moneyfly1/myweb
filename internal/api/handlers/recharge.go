package handlers

import (
	"fmt"
	"math"
	"net/http"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/services/geoip"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

func formatRechargeIP(rawIP string) string {
	ip := utils.ParseIP(rawIP)
	if ip == "" {
		return "-"
	}
	return ip
}

func rechargeIPAddress(record models.RechargeRecord) string {
	if !record.IPAddress.Valid {
		return "-"
	}
	return formatRechargeIP(record.IPAddress.String)
}

func rechargePaidAt(record models.RechargeRecord) interface{} {
	if record.PaidAt.Valid {
		return utils.FormatBeijingTime(record.PaidAt.Time)
	}
	return nil
}

func formatRechargeRecord(record models.RechargeRecord, includeUser bool, includeLocation bool) gin.H {
	ipAddress := rechargeIPAddress(record)
	location := ""
	if includeLocation && ipAddress != "-" && geoip.IsEnabled() {
		locationStr := geoip.GetLocationWithCache(ipAddress)
		if locationStr.Valid {
			location = locationStr.String
		}
	}

	data := gin.H{
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
		"paid_at":                rechargePaidAt(record),
		"created_at":             utils.FormatBeijingTime(record.CreatedAt),
		"updated_at":             utils.FormatBeijingTime(record.UpdatedAt),
	}

	if includeUser {
		data["user"] = resolveUserInfo(record.User)
	}

	return data
}

func CreateRecharge(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	var req struct {
		Amount        float64 `json:"amount" binding:"required,gt=0,lte=50000"`
		PaymentMethod string  `json:"payment_method"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	orderNo, err := utils.GenerateRechargeOrderNo(user.ID, db)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成充值订单号失败", err)
		return
	}
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

	paymentMethod := req.PaymentMethod
	if paymentMethod == "" {
		paymentMethod = "alipay"
	}

	paymentConfig, err := utils.FindEnabledPaymentConfig(db, paymentMethod)
	if err != nil {
		db.Model(&models.RechargeRecord{}).Where("id = ? AND status = ?", recharge.ID, "pending").Update("status", "failed")
		utils.ErrorResponse(c, http.StatusBadRequest, "未找到启用的支付配置", nil)
		return
	}

	var paymentURL string
	if paymentConfig.Status == 1 {
		// 创建支付交易记录
		amt := int(math.Round(recharge.Amount * 100)) // 转换为分
		paymentTx := models.PaymentTransaction{
			OrderID:         0, // 充值订单没有 order_id，使用 0
			UserID:          user.ID,
			PaymentMethodID: paymentConfig.ID,
			Amount:          amt,
			Currency:        "CNY",
			TransactionID:   database.NullString(recharge.OrderNo),
			Status:          "pending",
			CreatedAt:       beijingTime,
			UpdatedAt:       beijingTime,
		}
		if err := db.Create(&paymentTx).Error; err != nil {
			utils.LogError("CreateRecharge: failed to create payment transaction", err, map[string]interface{}{
				"order_no": recharge.OrderNo,
			})
			db.Model(&models.RechargeRecord{}).Where("id = ? AND status = ?", recharge.ID, "pending").Update("status", "failed")
			utils.ErrorResponse(c, http.StatusInternalServerError, "创建支付交易记录失败", err)
			return
		}

		tempOrder := &models.Order{
			OrderNo: recharge.OrderNo,
			UserID:  user.ID,
			Amount:  recharge.Amount,
		}

		paymentURL, err = generatePaymentURL(db, tempOrder, &paymentConfig, paymentMethod, recharge.Amount)
		if err != nil {
			utils.LogError("CreateRecharge: create payment failed", err, map[string]interface{}{
				"payment_method": paymentMethod,
				"order_no":       recharge.OrderNo,
			})
			utils.CreateBusinessLog(c, "recharge_payment_url_failed", "充值生成支付链接失败", "error", map[string]interface{}{
				"user_id": user.ID, "order_no": recharge.OrderNo, "payment_method": paymentMethod, "reason": err.Error(),
			})
			db.Model(&models.RechargeRecord{}).Where("id = ? AND status = ?", recharge.ID, "pending").Update("status", "failed")
			utils.ErrorResponse(c, http.StatusInternalServerError, "创建支付链接失败: "+err.Error(), nil)
			return
		}

		if paymentURL != "" {
			recharge.PaymentURL = database.NullString(paymentURL)
			if err := db.Save(&recharge).Error; err != nil {
				utils.LogError("CreateRecharge: save payment URL failed", err, nil)
			}
		} else {
			db.Model(&models.RechargeRecord{}).Where("id = ? AND status = ?", recharge.ID, "pending").Update("status", "failed")
			utils.ErrorResponse(c, http.StatusInternalServerError, "支付链接生成失败", nil)
			return
		}
	} else {
		db.Model(&models.RechargeRecord{}).Where("id = ? AND status = ?", recharge.ID, "pending").Update("status", "failed")
		utils.ErrorResponse(c, http.StatusBadRequest, "支付配置未启用", nil)
		return
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

	page, size, offset := getPagination(c)
	status := c.Query("status")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	db := database.GetDB()
	var records []models.RechargeRecord
	var total int64
	query := db.Model(&models.RechargeRecord{}).Where("user_id = ?", user.ID)

	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	var startParsed time.Time
	if startDate != "" {
		parsed, err := time.ParseInLocation("2006-01-02", startDate, utils.BeijingTZ)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "start_date格式错误，应为YYYY-MM-DD", err)
			return
		}
		startParsed = parsed
		query = query.Where("created_at >= ?", startParsed)
	}
	if endDate != "" {
		parsed, err := time.ParseInLocation("2006-01-02", endDate, utils.BeijingTZ)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "end_date格式错误，应为YYYY-MM-DD", err)
			return
		}
		if !startParsed.IsZero() && parsed.Before(startParsed) {
			utils.ErrorResponse(c, http.StatusBadRequest, "end_date不能早于start_date", nil)
			return
		}
		query = query.Where("created_at < ?", parsed.AddDate(0, 0, 1))
	}

	if err := query.Count(&total).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取充值记录失败", err)
		return
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(size).Find(&records).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取充值记录失败", err)
		return
	}

	formattedRecords := make([]gin.H, 0, len(records))
	for _, record := range records {
		formattedRecords = append(formattedRecords, formatRechargeRecord(record, false, false))
	}

	c.Header("X-Total-Count", fmt.Sprintf("%d", total))
	c.Header("X-Page", fmt.Sprintf("%d", page))
	c.Header("X-Page-Size", fmt.Sprintf("%d", size))
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

	utils.SuccessResponse(c, http.StatusOK, "", formatRechargeRecord(record, false, true))
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

	if record.Status == "pending" && shouldQueryPaymentStatus(record.CreatedAt) {
		if success, err := performPaymentStatusQuery(db, orderNo, true); success {
			db.Where("order_no = ?", orderNo).First(&record)
		} else if err != nil {
			utils.LogWarn("GetRechargeStatusByNo: active payment query failed: %+v", map[string]interface{}{
				"order_no": orderNo,
				"error":    err.Error(),
			})
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
		"paid_at":                rechargePaidAt(record),
		"created_at":             utils.FormatBeijingTime(record.CreatedAt),
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
	if err := db.Model(&models.PaymentTransaction{}).
		Where("order_id = ? AND transaction_id = ? AND status = ?", 0, record.OrderNo, "pending").
		Update("status", "cancelled").Error; err != nil {
		utils.LogError("CancelRecharge: mark payment transaction cancelled", err, map[string]interface{}{
			"order_no": record.OrderNo,
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "充值订单已取消", record)
}

func GetAdminRechargeRecords(c *gin.Context) {
	db := database.GetDB()
	page := 1
	size := 20
	if pageStr := c.Query("page"); pageStr != "" {
		_, _ = fmt.Sscanf(pageStr, "%d", &page) // Ignore error, use default value
	}
	if sizeStr := c.Query("size"); sizeStr != "" {
		_, _ = fmt.Sscanf(sizeStr, "%d", &size) // Ignore error, use default value
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

	var startParsed time.Time
	if startDate != "" {
		parsed, err := time.ParseInLocation("2006-01-02", startDate, utils.BeijingTZ)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "start_date格式错误，应为YYYY-MM-DD", err)
			return
		}
		startParsed = parsed
		dateCondition := "recharge_records.created_at >= ?"
		countQuery = countQuery.Where(dateCondition, startParsed)
		findQuery = findQuery.Where(dateCondition, startParsed)
	}
	if endDate != "" {
		parsed, err := time.ParseInLocation("2006-01-02", endDate, utils.BeijingTZ)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "end_date格式错误，应为YYYY-MM-DD", err)
			return
		}
		if !startParsed.IsZero() && parsed.Before(startParsed) {
			utils.ErrorResponse(c, http.StatusBadRequest, "end_date不能早于start_date", nil)
			return
		}
		endTime := parsed.AddDate(0, 0, 1)
		dateCondition := "recharge_records.created_at < ?"
		countQuery = countQuery.Where(dateCondition, endTime)
		findQuery = findQuery.Where(dateCondition, endTime)
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
	for _, record := range records {
		formattedRecords = append(formattedRecords, formatRechargeRecord(record, true, false))
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"recharges": formattedRecords,
		"total":     total,
		"page":      page,
		"size":      size,
	})
}
