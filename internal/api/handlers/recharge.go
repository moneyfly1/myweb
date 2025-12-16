package handlers

import (
	"net/http"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/services/payment"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

// CreateRecharge 创建充值订单
func CreateRecharge(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	var req struct {
		Amount        float64 `json:"amount" binding:"required,gt=0"`
		PaymentMethod string  `json:"payment_method"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()

	// 生成订单号
	orderNo := utils.GenerateRechargeOrderNo(user.ID)

	recharge := models.RechargeRecord{
		UserID:        user.ID,
		OrderNo:       orderNo,
		Amount:        req.Amount,
		Status:        "pending",
		PaymentMethod: database.NullString(req.PaymentMethod),
	}

	if err := db.Create(&recharge).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建充值订单失败",
		})
		return
	}

	// 调用支付接口生成支付链接
	var paymentURL string
	paymentMethod := req.PaymentMethod
	if paymentMethod == "" {
		paymentMethod = "alipay"
	}

	var paymentConfig models.PaymentConfig
	if err := db.Where("pay_type = ? AND status = ?", paymentMethod, 1).First(&paymentConfig).Error; err == nil {
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
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"id":          recharge.ID,
			"order_no":    recharge.OrderNo,
			"amount":      recharge.Amount,
			"status":      recharge.Status,
			"payment_url": paymentURL,
		},
	})
}

// GetRechargeRecords 获取充值记录
func GetRechargeRecords(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	db := database.GetDB()
	var records []models.RechargeRecord
	if err := db.Where("user_id = ?", user.ID).Order("created_at DESC").Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取充值记录失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    records,
	})
}

// GetRechargeRecord 获取单个充值记录
func GetRechargeRecord(c *gin.Context) {
	id := c.Param("id")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	db := database.GetDB()
	var record models.RechargeRecord
	if err := db.Where("id = ? AND user_id = ?", id, user.ID).First(&record).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "充值记录不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    record,
	})
}

// CancelRecharge 取消充值订单
func CancelRecharge(c *gin.Context) {
	id := c.Param("id")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	db := database.GetDB()
	var record models.RechargeRecord
	if err := db.Where("id = ? AND user_id = ?", id, user.ID).First(&record).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "充值记录不存在",
		})
		return
	}

	if record.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "只能取消待支付的充值订单",
		})
		return
	}

	record.Status = "cancelled"
	if err := db.Save(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "取消充值订单失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "充值订单已取消",
		"data":    record,
	})
}
