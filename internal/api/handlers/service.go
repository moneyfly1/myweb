package handlers

import (
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"cboard-go/internal/middleware"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/smservice"
	"cboard-go/internal/utils"
)

// ServiceCatalogItem 验证码服务商品（官方价已换算成人民币售价）。
type ServiceCatalogItem struct {
	ServiceCode      string  `json:"service_code"`
	ServiceName      string  `json:"service_name"`
	CountryID        int     `json:"country_id"`
	CountryName      string  `json:"country_name"`
	OfficialPriceUSD float64 `json:"official_price_usd"`
	PriceCNY         float64 `json:"price_cny"`
	Count            int     `json:"count"`
}

// serviceCatalogCache 缓存平台原始商品，避免每次请求都拉 1MB+ 的 getPrices（约 3 秒）。
var serviceCatalogCache struct {
	mu         sync.Mutex
	at         time.Time
	items      []ServiceCatalogItem
	multiplier float64
	rate       float64
}

const serviceCatalogCacheTTL = 5 * time.Minute

// GetServiceCatalog GET /api/v1/services/catalog
// 返回验证码服务商品列表（实时价格换算）。
func GetServiceCatalog(c *gin.Context) {
	db := database.GetDB()
	cfg := smservice.LoadConfig(db)
	if !cfg.Enabled {
		utils.SuccessResponse(c, 200, "服务未开启", gin.H{"enabled": false, "items": []ServiceCatalogItem{}})
		return
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		utils.SuccessResponse(c, 200, "未配置 API 密钥", gin.H{"enabled": true, "items": []ServiceCatalogItem{}, "notice": "管理员尚未配置 API 密钥"})
		return
	}

	serviceCatalogCache.mu.Lock()
	if time.Since(serviceCatalogCache.at) < serviceCatalogCacheTTL && len(serviceCatalogCache.items) > 0 &&
		serviceCatalogCache.multiplier == cfg.Multiplier && serviceCatalogCache.rate == cfg.ExchangeRate {
		cached := serviceCatalogCache.items
		serviceCatalogCache.mu.Unlock()
		respondServiceCatalog(c, cfg, cached)
		return
	}
	serviceCatalogCache.mu.Unlock()

	client := smservice.NewClient(cfg)
	prices, err := client.GetPrices()
	if err != nil {
		utils.ErrorResponse(c, 400, "获取价格失败："+err.Error(), nil)
		return
	}
	countries, err := client.GetCountries()
	if err != nil {
		utils.ErrorResponse(c, 400, "获取国家列表失败："+err.Error(), nil)
		return
	}
	services, err := client.GetServicesList()
	if err != nil {
		utils.ErrorResponse(c, 400, "获取服务列表失败："+err.Error(), nil)
		return
	}

	svcNames := make(map[string]string, len(services))
	for _, s := range services {
		code, _ := s["code"].(string)
		name, _ := s["name"].(string)
		if code == "" {
			continue
		}
		if name == "" {
			name = code
		}
		svcNames[code] = name
	}

	countryName := func(id string) string {
		row, ok := countries[id]
		if !ok {
			return "国家 " + id
		}
		for _, key := range []string{"chn", "eng", "name"} {
			if v, ok := row[key].(string); ok && strings.TrimSpace(v) != "" {
				return v
			}
		}
		return "国家 " + id
	}

	items := make([]ServiceCatalogItem, 0, 256)
	for cid, svcMap := range prices {
		countryID, _ := strconv.Atoi(cid)
		for serviceCode, p := range svcMap {
			if p.Cost <= 0 || p.Count <= 0 {
				continue
			}
			name := svcNames[serviceCode]
			if name == "" {
				name = serviceCode
			}
			items = append(items, ServiceCatalogItem{
				ServiceCode:      serviceCode,
				ServiceName:      name,
				CountryID:        countryID,
				CountryName:      countryName(cid),
				OfficialPriceUSD: p.Cost,
				PriceCNY:         cfg.PriceCNY(p.Cost),
				Count:            p.Count,
			})
		}
	}
	serviceCatalogCache.mu.Lock()
	serviceCatalogCache.at = time.Now()
	serviceCatalogCache.items = items
	serviceCatalogCache.multiplier = cfg.Multiplier
	serviceCatalogCache.rate = cfg.ExchangeRate
	serviceCatalogCache.mu.Unlock()

	respondServiceCatalog(c, cfg, items)
}

// respondServiceCatalog 对商品做筛选/排序/截断后返回（缓存命中与未命中都走这里）。
func respondServiceCatalog(c *gin.Context, cfg *smservice.Config, items []ServiceCatalogItem) {
	q := strings.ToLower(strings.TrimSpace(c.Query("q")))
	serviceFilter := strings.TrimSpace(c.Query("service"))
	countryFilter := strings.TrimSpace(c.Query("country"))
	if q != "" || serviceFilter != "" || countryFilter != "" {
		filtered := make([]ServiceCatalogItem, 0, len(items))
		for _, it := range items {
			if serviceFilter != "" && it.ServiceCode != serviceFilter {
				continue
			}
			if countryFilter != "" && strconv.Itoa(it.CountryID) != countryFilter {
				continue
			}
			if q != "" && !strings.Contains(strings.ToLower(it.ServiceName), q) && !strings.Contains(strings.ToLower(it.ServiceCode), q) {
				continue
			}
			filtered = append(filtered, it)
		}
		items = filtered
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		if items[i].PriceCNY == items[j].PriceCNY {
			return items[i].ServiceName < items[j].ServiceName
		}
		return items[i].PriceCNY < items[j].PriceCNY
	})
	total := len(items)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "300"))
	if limit <= 0 || limit > 1000 {
		limit = 300
	}
	if len(items) > limit {
		items = items[:limit]
	}
	utils.SuccessResponse(c, 200, "", gin.H{
		"enabled":       true,
		"multiplier":    cfg.Multiplier,
		"exchange_rate": cfg.ExchangeRate,
		"total":         total,
		"items":         items,
	})
}

type createServiceOrderRequest struct {
	ServiceCode string `json:"service_code" binding:"required"`
	CountryID   int    `json:"country_id" binding:"required"`
	PaymentMethod string `json:"payment_method" binding:"required"`
}

// CreateServiceOrder POST /api/v1/services/orders
func CreateServiceOrder(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, 401, "未登录", nil)
		return
	}
	var req createServiceOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, "参数错误", err)
		return
	}
	db := database.GetDB()
	cfg := smservice.LoadConfig(db)
	if !cfg.Enabled {
		utils.ErrorResponse(c, 403, "验证码服务未开启", nil)
		return
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		utils.ErrorResponse(c, 503, "验证码服务未配置 API 密钥", nil)
		return
	}

	// 实时取价（防止前端价格被篡改）
	client := smservice.NewClient(cfg)
	prices, err := client.GetPrices()
	if err != nil {
		utils.ErrorResponse(c, 400, "获取价格失败："+err.Error(), nil)
		return
	}
	countryPrices, ok := prices[strconv.Itoa(req.CountryID)]
	if !ok {
		utils.ErrorResponse(c, 400, "该国家暂无可用号码", nil)
		return
	}
	p, ok := countryPrices[req.ServiceCode]
	if !ok || p.Cost <= 0 || p.Count <= 0 {
		utils.ErrorResponse(c, 400, "该服务暂无可用号码", nil)
		return
	}

	// 支付方式校验
	methodID, methodName, ok := resolveServicePaymentMethod(db, req.PaymentMethod)
	if !ok {
		utils.ErrorResponse(c, 400, "支付方式不可用", nil)
		return
	}

	priceCNY := cfg.PriceCNY(p.Cost)
	orderNo := fmt.Sprintf("SVC%d%04d%04d", time.Now().Unix(), time.Now().Nanosecond()/1e4, user.ID%10000)
	if len(orderNo) > 50 {
		orderNo = orderNo[:50]
	}

	order := models.ServiceOrder{
		OrderNo:          orderNo,
		UserID:           user.ID,
		ServiceCode:      req.ServiceCode,
		ServiceName:      req.ServiceCode,
		CountryID:        req.CountryID,
		CountryName:      strconv.Itoa(req.CountryID),
		OfficialPriceUSD: p.Cost,
		Multiplier:       cfg.Multiplier,
		ExchangeRate:     cfg.ExchangeRate,
		PriceCNY:         priceCNY,
		PaymentMethod:    sql.NullString{String: methodName, Valid: methodName != ""},
		PaymentStatus:    "pending",
		Status:           "pending",
	}
	if err := db.Create(&order).Error; err != nil {
		utils.ErrorResponse(c, 500, "创建订单失败", err)
		return
	}

	// 用套餐订单结构生成支付链接（不落库，仅作为支付参数载体）
	paymentCfg, err := utils.FindEnabledPaymentConfig(db, req.PaymentMethod)
	if err != nil {
		_ = db.Model(&order).Update("status", "failed").Error
		utils.ErrorResponse(c, 400, "支付方式未启用", nil)
		return
	}
	tempOrder := &models.Order{
		OrderNo:       orderNo,
		UserID:        user.ID,
		PackageID:     0,
		Amount:        priceCNY,
		AmountDueOnline: sql.NullFloat64{Float64: priceCNY, Valid: true},
	}
	tempOrder.PaymentMethodName = sql.NullString{String: methodName, Valid: methodName != ""}
	paymentURL, err := generatePaymentURL(db, tempOrder, &paymentCfg, req.PaymentMethod, priceCNY)
	if err != nil {
		_ = db.Model(&order).Update("status", "failed").Error
		utils.ErrorResponse(c, 500, "生成支付链接失败："+err.Error(), nil)
		return
	}
	_ = methodID

	utils.SuccessResponse(c, 200, "订单创建成功", gin.H{
		"order_no":      orderNo,
		"price_cny":     priceCNY,
		"official_usd":  p.Cost,
		"payment_method": methodName,
		"payment_url":   paymentURL,
	})
}

// resolveServicePaymentMethod 校验前端传入的支付方式（key 或 name）是否启用。
func resolveServicePaymentMethod(db *gorm.DB, raw string) (uint, string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, "", false
	}
	var cfg models.PaymentConfig
	if err := db.Where("pay_type = ? AND status = 1", raw).First(&cfg).Error; err == nil {
		return cfg.ID, cfg.PayType, true
	}
	// 允许 codepay_alipay 这类带后缀的写法
	if strings.HasPrefix(raw, "codepay_") || strings.HasPrefix(raw, "yipay_") {
		base := strings.SplitN(raw, "_", 2)[0]
		if err := db.Where("pay_type = ? AND status = 1", base).First(&cfg).Error; err == nil {
			return cfg.ID, raw, true
		}
	}
	return 0, "", false
}

// GetMyServiceOrders GET /api/v1/services/orders
func GetMyServiceOrders(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, 401, "未登录", nil)
		return
	}
	db := database.GetDB()
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	var total int64
	var orders []models.ServiceOrder
	db.Model(&models.ServiceOrder{}).Where("user_id = ?", user.ID).Count(&total)
	if err := db.Where("user_id = ?", user.ID).Order("id DESC").Limit(size).Offset((page - 1) * size).Find(&orders).Error; err != nil {
		utils.ErrorResponse(c, 500, "查询订单失败", err)
		return
	}
	items := make([]gin.H, 0, len(orders))
	for _, o := range orders {
		items = append(items, serviceOrderView(&o))
	}
	utils.SuccessResponse(c, 200, "", gin.H{"total": total, "page": page, "size": size, "items": items})
}

func serviceOrderView(o *models.ServiceOrder) gin.H {
	return gin.H{
		"id":               o.ID,
		"order_no":         o.OrderNo,
		"service_code":     o.ServiceCode,
		"service_name":     o.ServiceName,
		"country_id":       o.CountryID,
		"country_name":     o.CountryName,
		"official_price_usd": o.OfficialPriceUSD,
		"price_cny":        o.PriceCNY,
		"payment_method":   o.PaymentMethod.String,
		"payment_status":   o.PaymentStatus,
		"status":           o.Status,
		"phone_number":     o.PhoneNumber.String,
		"otp":              o.OTP.String,
		"activation_id":    o.ActivationID.String,
		"refunded":         o.Refunded,
		"created_at":       utils.FormatBeijingTime(o.CreatedAt),
	}
}

// GetServiceOrderDetail GET /api/v1/services/orders/:orderNo
// 订单详情；若已激活且未拿到验证码，实时向平台查询一次。
func GetServiceOrderDetail(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, 401, "未登录", nil)
		return
	}
	orderNo := c.Param("orderNo")
	db := database.GetDB()
	var order models.ServiceOrder
	if err := db.Where("order_no = ? AND user_id = ?", orderNo, user.ID).First(&order).Error; err != nil {
		utils.ErrorResponse(c, 404, "订单不存在", nil)
		return
	}
	if order.Status == "active" && order.ActivationID.Valid && !order.OTP.Valid {
		cfg := smservice.LoadConfig(db)
		if cfg.Enabled && cfg.APIKey != "" {
			client := smservice.NewClient(cfg)
			if st, code, err := client.GetStatus(order.ActivationID.String); err == nil {
				if st == "ok" && code != "" {
					order.OTP = sql.NullString{String: code, Valid: true}
					order.Status = "completed"
					now := time.Now()
					order.CompletedAt.Valid = true
					order.CompletedAt.Time = now
					_ = db.Save(&order).Error
				}
			}
		}
	}
	utils.SuccessResponse(c, 200, "", serviceOrderView(&order))
}

// FinishServiceOrder POST /api/v1/services/orders/:orderNo/finish
func FinishServiceOrder(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, 401, "未登录", nil)
		return
	}
	orderNo := c.Param("orderNo")
	db := database.GetDB()
	var order models.ServiceOrder
	if err := db.Where("order_no = ? AND user_id = ?", orderNo, user.ID).First(&order).Error; err != nil {
		utils.ErrorResponse(c, 404, "订单不存在", nil)
		return
	}
	if !order.ActivationID.Valid {
		utils.ErrorResponse(c, 400, "订单尚未激活", nil)
		return
	}
	cfg := smservice.LoadConfig(db)
	client := smservice.NewClient(cfg)
	if err := client.Finish(order.ActivationID.String); err != nil {
		utils.ErrorResponse(c, 400, "完成激活失败："+err.Error(), nil)
		return
	}
	order.Status = "completed"
	now := time.Now()
	order.CompletedAt.Valid = true
	order.CompletedAt.Time = now
	_ = db.Save(&order).Error
	utils.SuccessResponse(c, 200, "已标记完成", nil)
}

// CancelServiceOrder POST /api/v1/services/orders/:orderNo/cancel
func CancelServiceOrder(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, 401, "未登录", nil)
		return
	}
	orderNo := c.Param("orderNo")
	db := database.GetDB()
	var order models.ServiceOrder
	if err := db.Where("order_no = ? AND user_id = ?", orderNo, user.ID).First(&order).Error; err != nil {
		utils.ErrorResponse(c, 404, "订单不存在", nil)
		return
	}
	if order.Status == "cancelled" {
		utils.SuccessResponse(c, 200, "订单已取消", nil)
		return
	}
	if order.ActivationID.Valid {
		cfg := smservice.LoadConfig(db)
		client := smservice.NewClient(cfg)
		if err := client.Cancel(order.ActivationID.String); err != nil {
			utils.ErrorResponse(c, 400, "取消激活失败："+err.Error(), nil)
			return
		}
	}
	order.Status = "cancelled"
	_ = db.Save(&order).Error
	utils.SuccessResponse(c, 200, "已取消", nil)
}


// applyServiceOrderPayment 支付回调命中服务订单时的履约：
// 校验金额 → 调用平台购买号码 → 成功激活；失败则自动退款到用户余额。
func applyServiceOrderPayment(c *gin.Context, db *gorm.DB, order *models.ServiceOrder, paymentType, orderNo, externalTransactionID string, paymentConfigID uint, params map[string]string) {
	if order.PaymentStatus == "paid" || order.PaymentStatus == "refunded" {
		paymentCallbackAck(c, paymentType, true, "OK")
		return
	}
	if amt, ok := parseCallbackAmount(paymentType, params); ok {
		if !amountMatches(order.PriceCNY, amt) {
			utils.LogError("ServiceOrderPay: amount mismatch", nil, map[string]interface{}{
				"order_no": orderNo, "expected": order.PriceCNY, "callback": amt,
			})
			c.String(400, "订单金额不匹配")
			return
		}
	}

	cfg := smservice.LoadConfig(db)
	client := smservice.NewClient(cfg)
	activationID, phone, err := client.GetNumber(order.ServiceCode, strconv.Itoa(order.CountryID))
	if err != nil {
		utils.LogError("ServiceOrderPay: 购买号码失败", err, map[string]interface{}{
			"order_no": orderNo, "service": order.ServiceCode, "country": order.CountryID,
		})
		refundServiceOrder(db, order, "购买号码失败："+err.Error())
		paymentCallbackAck(c, paymentType, true, "OK")
		return
	}

	now := time.Now()
	order.ActivationID = sql.NullString{String: activationID, Valid: true}
	order.PhoneNumber = sql.NullString{String: phone, Valid: true}
	order.PaymentStatus = "paid"
	order.Status = "active"
	order.PaidAt = sql.NullTime{Time: now, Valid: true}
	order.PaymentMethod = sql.NullString{String: paymentType, Valid: true}
	order.APIResponse = sql.NullString{String: fmt.Sprintf(`{"activation_id":%q,"phone":%q}`, activationID, phone), Valid: true}
	if err := db.Save(order).Error; err != nil {
		utils.LogError("ServiceOrderPay: 保存激活信息失败", err, map[string]interface{}{"order_no": orderNo})
	}
	utils.CreateBusinessLog(c, "service_order_paid", "验证码服务订单支付成功并已购买号码", "info", map[string]interface{}{
		"order_no": orderNo, "phone": phone, "service": order.ServiceCode, "country": order.CountryID,
	})
	paymentCallbackAck(c, paymentType, true, "OK")
}

// refundServiceOrder 购买失败时把订单金额退回用户余额（记录流水）。
func refundServiceOrder(db *gorm.DB, order *models.ServiceOrder, reason string) {
	_ = db.Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.First(&user, order.UserID).Error; err != nil {
			return err
		}
		newBalance := user.Balance + order.PriceCNY
		if err := tx.Model(&models.User{}).Where("id = ?", user.ID).Update("balance", gorm.Expr("balance + ?", order.PriceCNY)).Error; err != nil {
			return err
		}
		orderID := order.ID
		_ = utils.CreateBalanceLogWithDB(tx, user.ID, "refund", order.PriceCNY, user.Balance, newBalance, &orderID, nil, "验证码服务购买失败自动退款", "system", nil, "")
		return tx.Model(order).Updates(map[string]interface{}{
			"payment_status": "refunded",
			"status":         "failed",
			"refunded":       true,
			"refund_reason":  reason,
		}).Error
	})
}

// AdminGetServiceConfig GET /api/v1/admin/services/config
func AdminGetServiceConfig(c *gin.Context) {
	db := database.GetDB()
	cfg := smservice.LoadConfig(db)
	utils.SuccessResponse(c, 200, "", gin.H{
		"enabled":       cfg.Enabled,
		"base_url":      cfg.BaseURL,
		"api_key":       maskSecret(sql.NullString{String: cfg.APIKey, Valid: cfg.APIKey != ""}),
		"multiplier":    cfg.Multiplier,
		"exchange_rate": cfg.ExchangeRate,
		"min_price":     cfg.MinPrice,
	})
}

// AdminUpdateServiceConfig PUT /api/v1/admin/services/config
func AdminUpdateServiceConfig(c *gin.Context) {
	var req struct {
		Enabled      bool    `json:"enabled"`
		BaseURL      string  `json:"base_url"`
		APIKey       string  `json:"api_key"`
		Multiplier   float64 `json:"multiplier"`
		ExchangeRate float64 `json:"exchange_rate"`
		MinPrice     float64 `json:"min_price"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, "参数错误", err)
		return
	}
	db := database.GetDB()
	old := smservice.LoadConfig(db)
	apiKey := strings.TrimSpace(req.APIKey)
	// 前端回显的是掩码，原样提交时保留旧 key
	if apiKey == "" || strings.Contains(apiKey, "****") {
		apiKey = old.APIKey
	}
	if req.Multiplier <= 0 {
		req.Multiplier = 2.0
	}
	if req.ExchangeRate <= 0 {
		req.ExchangeRate = 7.2
	}
	if strings.TrimSpace(req.BaseURL) == "" {
		req.BaseURL = "https://hero-sms.com"
	}
	set := func(key, value string) {
		var cfg models.SystemConfig
		err := db.Where("key = ? AND category = ?", key, "sms_service").First(&cfg).Error
		if err == gorm.ErrRecordNotFound {
			cfg = models.SystemConfig{Key: key, Category: "sms_service", Type: "string", Value: value, DisplayName: key}
			_ = db.Create(&cfg).Error
			return
		}
		if err == nil {
			_ = db.Model(&cfg).Update("value", value).Error
		}
	}
	set("sms_service_enabled", strconv.FormatBool(req.Enabled))
	set("sms_service_base_url", strings.TrimRight(strings.TrimSpace(req.BaseURL), "/"))
	set("sms_service_api_key", apiKey)
	set("sms_service_multiplier", strconv.FormatFloat(req.Multiplier, 'f', 2, 64))
	set("sms_service_exchange_rate", strconv.FormatFloat(req.ExchangeRate, 'f', 4, 64))
	set("sms_service_min_price", strconv.FormatFloat(req.MinPrice, 'f', 2, 64))

	utils.SuccessResponse(c, 200, "保存成功", gin.H{"enabled": req.Enabled})
}
