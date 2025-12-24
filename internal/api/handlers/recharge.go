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

	recharge := models.RechargeRecord{
		UserID:        user.ID,
		OrderNo:       orderNo,
		Amount:        req.Amount,
		Status:        "pending",
		PaymentMethod: database.NullString(req.PaymentMethod),
	}

	if err := db.Create(&recharge).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建充值订单失败", err)
		return
	}

	// 调用支付接口生成支付链接
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

	utils.SuccessResponse(c, http.StatusOK, "", records)
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

	utils.SuccessResponse(c, http.StatusOK, "", record)
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
