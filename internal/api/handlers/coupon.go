package handlers

import (
	"net/http"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

func GetCoupons(c *gin.Context) {
	db := database.GetDB()

	var coupons []models.Coupon
	now := utils.GetBeijingTime()
	if err := db.Where("status = ? AND valid_from <= ? AND valid_until >= ?", "active", now, now).Find(&coupons).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取优惠券列表失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", coupons)
}

func GetCoupon(c *gin.Context) {
	code := c.Param("code")

	db := database.GetDB()
	var coupon models.Coupon
	if err := db.Where("code = ?", code).First(&coupon).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "优惠券不存在", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", coupon)
}

func VerifyCoupon(c *gin.Context) {
	var req struct {
		Code      string  `json:"code" binding:"required"`
		Amount    float64 `json:"amount" binding:"required"`
		PackageID uint    `json:"package_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	var coupon models.Coupon
	if err := db.Where("code = ?", req.Code).First(&coupon).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "优惠券不存在", err)
		return
	}

	now := utils.GetBeijingTime()
	if coupon.Status != "active" {
		utils.ErrorResponse(c, http.StatusBadRequest, "优惠券已失效", nil)
		return
	}

	if now.Before(coupon.ValidFrom) || now.After(coupon.ValidUntil) {
		utils.ErrorResponse(c, http.StatusBadRequest, "优惠券不在有效期内", nil)
		return
	}

	if coupon.MinAmount.Valid && req.Amount < coupon.MinAmount.Float64 {
		utils.ErrorResponse(c, http.StatusBadRequest, "订单金额不满足优惠券使用条件", nil)
		return
	}

	discountAmount := 0.0
	if coupon.Type == "discount" {
		discountAmount = req.Amount * (coupon.DiscountValue / 100)
		if coupon.MaxDiscount.Valid && discountAmount > coupon.MaxDiscount.Float64 {
			discountAmount = coupon.MaxDiscount.Float64
		}
	} else if coupon.Type == "fixed" {
		discountAmount = coupon.DiscountValue
		if discountAmount > req.Amount {
			discountAmount = req.Amount
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"coupon":          coupon,
		"discount_amount": discountAmount,
		"final_amount":    req.Amount - discountAmount,
	})
}

func CreateCoupon(c *gin.Context) {
	var req struct {
		Code               string  `json:"code"`
		Name               string  `json:"name" binding:"required"`
		Description        string  `json:"description"`
		Type               string  `json:"type" binding:"required"`
		DiscountValue      float64 `json:"discount_value" binding:"required"`
		MinAmount          float64 `json:"min_amount"`
		MaxDiscount        float64 `json:"max_discount"`
		ValidFrom          string  `json:"valid_from" binding:"required"`
		ValidUntil         string  `json:"valid_until" binding:"required"`
		TotalQuantity      int     `json:"total_quantity"`
		MaxUsesPerUser     int     `json:"max_uses_per_user"`
		ApplicablePackages string  `json:"applicable_packages"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.LogError("CreateCoupon: bind JSON failed", err, nil)
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()

	validFrom, err := time.Parse("2006-01-02T15:04:05", req.ValidFrom)
	if err != nil {
		validFrom, err = time.Parse("2006-01-02 15:04:05", req.ValidFrom)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "生效时间格式错误", err)
			return
		}
	}
	validUntil, err := time.Parse("2006-01-02T15:04:05", req.ValidUntil)
	if err != nil {
		validUntil, err = time.Parse("2006-01-02 15:04:05", req.ValidUntil)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "失效时间格式错误", err)
			return
		}
	}

	code := req.Code
	if code == "" {
		code = utils.GenerateCouponCode()
		var existing models.Coupon
		for db.Where("code = ?", code).First(&existing).Error == nil {
			code = utils.GenerateCouponCode()
		}
	} else {
		var existing models.Coupon
		if err := db.Where("code = ?", code).First(&existing).Error; err == nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "优惠券码已存在", nil)
			return
		}
	}

	coupon := models.Coupon{
		Code:          code,
		Name:          req.Name,
		Type:          req.Type,
		DiscountValue: req.DiscountValue,
		ValidFrom:     validFrom,
		ValidUntil:    validUntil,
		Status:        "active",
	}

	if req.ApplicablePackages != "" {
		coupon.ApplicablePackages = req.ApplicablePackages
	}

	if req.Description != "" {
		coupon.Description = req.Description
	}
	if req.MinAmount > 0 {
		coupon.MinAmount = database.NullFloat64(req.MinAmount)
	}
	if req.MaxDiscount > 0 {
		coupon.MaxDiscount = database.NullFloat64(req.MaxDiscount)
	}
	if req.TotalQuantity > 0 {
		coupon.TotalQuantity = database.NullInt64(int64(req.TotalQuantity))
	}
	if req.MaxUsesPerUser > 0 {
		coupon.MaxUsesPerUser = req.MaxUsesPerUser
	} else {
		coupon.MaxUsesPerUser = 1 // 默认值
	}

	if err := db.Create(&coupon).Error; err != nil {
		utils.LogError("CreateCoupon: create coupon failed", err, nil)
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建优惠券失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "", coupon)
}

func GetUserCoupons(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	db := database.GetDB()
	var usages []models.CouponUsage
	if err := db.Where("user_id = ?", user.ID).Preload("Coupon").Find(&usages).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取优惠券使用记录失败", err)
		return
	}

	var result []map[string]interface{}
	for _, usage := range usages {
		result = append(result, map[string]interface{}{
			"id":              usage.ID,
			"coupon":          usage.Coupon,
			"discount_amount": usage.DiscountAmount,
			"used_at":         usage.UsedAt,
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

func GetAdminCoupon(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()

	var coupon models.Coupon
	if err := db.First(&coupon, id).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "优惠券不存在", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", coupon)
}

func UpdateCoupon(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()

	var coupon models.Coupon
	if err := db.First(&coupon, id).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "优惠券不存在", err)
		return
	}

	var req struct {
		Name               string  `json:"name"`
		Description        string  `json:"description"`
		Type               string  `json:"type"`
		DiscountValue      float64 `json:"discount_value"`
		MinAmount          float64 `json:"min_amount"`
		MaxDiscount        float64 `json:"max_discount"`
		ValidFrom          string  `json:"valid_from"`
		ValidUntil         string  `json:"valid_until"`
		TotalQuantity      int     `json:"total_quantity"`
		MaxUsesPerUser     int     `json:"max_uses_per_user"`
		Status             string  `json:"status"`
		ApplicablePackages string  `json:"applicable_packages"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.LogError("CreateCoupon: bind JSON failed", err, nil)
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	if req.Name != "" {
		coupon.Name = req.Name
	}
	if req.Description != "" {
		coupon.Description = req.Description
	}
	if req.Type != "" {
		coupon.Type = req.Type
	}
	if req.DiscountValue > 0 {
		coupon.DiscountValue = req.DiscountValue
	}
	if req.MinAmount > 0 {
		coupon.MinAmount = database.NullFloat64(req.MinAmount)
	}
	if req.MaxDiscount > 0 {
		coupon.MaxDiscount = database.NullFloat64(req.MaxDiscount)
	}
	if req.ValidFrom != "" {
		validFrom, err := time.Parse("2006-01-02T15:04:05", req.ValidFrom)
		if err != nil {
			validFrom, err = time.Parse("2006-01-02 15:04:05", req.ValidFrom)
			if err != nil {
				utils.ErrorResponse(c, http.StatusBadRequest, "生效时间格式错误", err)
				return
			}
		}
		coupon.ValidFrom = validFrom
	}
	if req.ValidUntil != "" {
		validUntil, err := time.Parse("2006-01-02T15:04:05", req.ValidUntil)
		if err != nil {
			validUntil, err = time.Parse("2006-01-02 15:04:05", req.ValidUntil)
			if err != nil {
				utils.ErrorResponse(c, http.StatusBadRequest, "失效时间格式错误", err)
				return
			}
		}
		coupon.ValidUntil = validUntil
	}
	if req.TotalQuantity > 0 {
		coupon.TotalQuantity = database.NullInt64(int64(req.TotalQuantity))
	}
	if req.MaxUsesPerUser > 0 {
		coupon.MaxUsesPerUser = req.MaxUsesPerUser
	}
	if req.Status != "" {
		coupon.Status = req.Status
	}
	if req.ApplicablePackages != "" {
		coupon.ApplicablePackages = req.ApplicablePackages
	} else if req.ApplicablePackages == "" {
		coupon.ApplicablePackages = ""
	}

	if err := db.Save(&coupon).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新优惠券失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "更新成功", coupon)
}

func DeleteCoupon(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()

	if err := db.Delete(&models.Coupon{}, id).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除优惠券失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "删除成功", nil)
}
