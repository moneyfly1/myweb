package handlers

import (
	"net/http"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/services/payment"

	"github.com/gin-gonic/gin"
)

// GetPaymentMethods 获取支付方式列表
func GetPaymentMethods(c *gin.Context) {
	db := database.GetDB()

	var methods []models.PaymentConfig
	if err := db.Where("status = ?", 1).Order("sort_order ASC").Find(&methods).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取支付方式失败",
		})
		return
	}

	// 只返回必要信息，不返回敏感配置
	result := make([]map[string]interface{}, 0)
	for _, method := range methods {
		result = append(result, map[string]interface{}{
			"id":       method.ID,
			"pay_type": method.PayType,
			"status":   method.Status,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// CreatePayment 创建支付
func CreatePayment(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	var req struct {
		OrderID         uint `json:"order_id" binding:"required"`
		PaymentMethodID uint `json:"payment_method_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()

	// 验证订单
	var order models.Order
	if err := db.Where("id = ? AND user_id = ?", req.OrderID, user.ID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "订单不存在",
		})
		return
	}

	if order.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "订单状态不允许支付",
		})
		return
	}

	// 获取支付配置
	var paymentConfig models.PaymentConfig
	if err := db.First(&paymentConfig, req.PaymentMethodID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "支付方式不存在",
		})
		return
	}

	if paymentConfig.Status != 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "支付方式已停用",
		})
		return
	}

	// 计算支付金额（分）
	amount := int(order.FinalAmount.Float64 * 100)
	if !order.FinalAmount.Valid {
		amount = int(order.Amount * 100)
	}

	// 创建支付交易
	transaction := models.PaymentTransaction{
		OrderID:         order.ID,
		UserID:          user.ID,
		PaymentMethodID: uint(req.PaymentMethodID),
		Amount:          amount,
		Currency:        "CNY",
		Status:          "pending",
	}

	if err := db.Create(&transaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建支付交易失败",
		})
		return
	}

	// 根据支付方式调用相应的支付接口
	var paymentURL string
	var err error

	paymentMethod := paymentConfig.PayType
	if paymentMethod == "alipay" {
		alipayService, err := payment.NewAlipayService(&paymentConfig)
		if err == nil {
			paymentURL, err = alipayService.CreatePayment(&order, float64(amount)/100)
		}
	} else if paymentMethod == "wechat" {
		wechatService, err := payment.NewWechatService(&paymentConfig)
		if err == nil {
			paymentURL, err = wechatService.CreatePayment(&order, float64(amount)/100)
		}
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "生成支付链接失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"transaction_id": transaction.ID,
			"amount":         float64(amount) / 100,
			"payment_url":    paymentURL,
			"qr_code":        "", // 二维码可以通过前端生成
		},
	})
}

// PaymentNotify 支付回调
func PaymentNotify(c *gin.Context) {
	paymentType := c.Param("type") // alipay, wechat, etc.
	db := database.GetDB()

	// 获取回调参数
	params := make(map[string]string)
	if paymentType == "alipay" {
		// 支付宝回调参数在 query 中
		for k, v := range c.Request.URL.Query() {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}
	} else if paymentType == "wechat" {
		// 微信回调参数在 body 中（XML格式）
		// 这里简化处理，实际需要解析 XML
	}

	// 获取支付配置
	var paymentConfig models.PaymentConfig
	if err := db.Where("type = ? AND is_active = ?", paymentType, true).First(&paymentConfig).Error; err != nil {
		c.String(http.StatusBadRequest, "支付配置不存在")
		return
	}

	// 验证签名
	var verified bool
	if paymentType == "alipay" {
		alipayService, err := payment.NewAlipayService(&paymentConfig)
		if err == nil {
			verified = alipayService.VerifyNotify(params)
		}
	} else if paymentType == "wechat" {
		wechatService, err := payment.NewWechatService(&paymentConfig)
		if err == nil {
			verified = wechatService.VerifyNotify(params)
		}
	}

	if !verified {
		c.String(http.StatusBadRequest, "签名验证失败")
		return
	}

	// 获取订单号
	orderNo := params["out_trade_no"]
	if orderNo == "" {
		c.String(http.StatusBadRequest, "订单号不存在")
		return
	}

	// 更新订单状态
	var order models.Order
	if err := db.Where("order_no = ?", orderNo).First(&order).Error; err != nil {
		c.String(http.StatusBadRequest, "订单不存在")
		return
	}

	// 更新订单和支付交易状态
	order.Status = "paid"
	db.Save(&order)

	var transaction models.PaymentTransaction
	if err := db.Where("order_id = ?", order.ID).First(&transaction).Error; err == nil {
		transaction.Status = "success"
		db.Save(&transaction)
	}

	// 处理订阅（如果订单包含订阅）
	if order.PackageID > 0 {
		// 这里可以触发订阅激活逻辑
	}

	c.String(http.StatusOK, "success")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "回调处理成功",
	})
}

// GetPaymentStatus 查询支付状态
func GetPaymentStatus(c *gin.Context) {
	transactionID := c.Param("id")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	db := database.GetDB()
	var transaction models.PaymentTransaction
	if err := db.Where("id = ? AND user_id = ?", transactionID, user.ID).First(&transaction).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "支付交易不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"status":   transaction.Status,
			"amount":   float64(transaction.Amount) / 100,
			"order_id": transaction.OrderID,
		},
	})
}
