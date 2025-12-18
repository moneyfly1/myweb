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

// GetCoupons 获取优惠券列表
func GetCoupons(c *gin.Context) {
	db := database.GetDB()

	var coupons []models.Coupon
	now := utils.GetBeijingTime()
	if err := db.Where("status = ? AND valid_from <= ? AND valid_until >= ?", "active", now, now).Find(&coupons).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取优惠券列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    coupons,
	})
}

// GetCoupon 获取单个优惠券
func GetCoupon(c *gin.Context) {
	code := c.Param("code")

	db := database.GetDB()
	var coupon models.Coupon
	if err := db.Where("code = ?", code).First(&coupon).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "优惠券不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    coupon,
	})
}

// VerifyCoupon 验证优惠券
func VerifyCoupon(c *gin.Context) {
	var req struct {
		Code      string  `json:"code" binding:"required"`
		Amount    float64 `json:"amount" binding:"required"`
		PackageID uint    `json:"package_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	var coupon models.Coupon
	if err := db.Where("code = ?", req.Code).First(&coupon).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "优惠券不存在",
		})
		return
	}

	now := utils.GetBeijingTime()
	if coupon.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "优惠券已失效",
		})
		return
	}

	if now.Before(coupon.ValidFrom) || now.After(coupon.ValidUntil) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "优惠券不在有效期内",
		})
		return
	}

	if coupon.MinAmount.Valid && req.Amount < coupon.MinAmount.Float64 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "订单金额不满足优惠券使用条件",
		})
		return
	}

	// 计算折扣金额
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"coupon":          coupon,
			"discount_amount": discountAmount,
			"final_amount":    req.Amount - discountAmount,
		},
	})
}

// CreateCoupon 创建优惠券（管理员）
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
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()

	// 解析日期时间
	validFrom, err := time.Parse("2006-01-02T15:04:05", req.ValidFrom)
	if err != nil {
		// 尝试其他格式
		validFrom, err = time.Parse("2006-01-02 15:04:05", req.ValidFrom)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "生效时间格式错误",
			})
			return
		}
	}
	validUntil, err := time.Parse("2006-01-02T15:04:05", req.ValidUntil)
	if err != nil {
		// 尝试其他格式
		validUntil, err = time.Parse("2006-01-02 15:04:05", req.ValidUntil)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "失效时间格式错误",
			})
			return
		}
	}

	// 如果code为空，自动生成
	code := req.Code
	if code == "" {
		code = utils.GenerateCouponCode()
		// 确保生成的code不重复
		var existing models.Coupon
		for db.Where("code = ?", code).First(&existing).Error == nil {
			code = utils.GenerateCouponCode()
		}
	} else {
		// 检查优惠券码是否已存在
		var existing models.Coupon
		if err := db.Where("code = ?", code).First(&existing).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "优惠券码已存在",
			})
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建优惠券失败",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    coupon,
	})
}

// GetUserCoupons 获取用户优惠券
func GetUserCoupons(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	db := database.GetDB()
	var usages []models.CouponUsage
	if err := db.Where("user_id = ?", user.ID).Preload("Coupon").Find(&usages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取优惠券使用记录失败",
		})
		return
	}

	// 转换为前端需要的格式
	var result []map[string]interface{}
	for _, usage := range usages {
		result = append(result, map[string]interface{}{
			"id":              usage.ID,
			"coupon":          usage.Coupon,
			"discount_amount": usage.DiscountAmount,
			"used_at":         usage.UsedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetAdminCoupon 管理员获取单个优惠券详情
func GetAdminCoupon(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()

	var coupon models.Coupon
	if err := db.First(&coupon, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "优惠券不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    coupon,
	})
}

// UpdateCoupon 更新优惠券（管理员）
func UpdateCoupon(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()

	var coupon models.Coupon
	if err := db.First(&coupon, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "优惠券不存在",
		})
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
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	// 更新字段
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
	// 解析日期时间
	if req.ValidFrom != "" {
		validFrom, err := time.Parse("2006-01-02T15:04:05", req.ValidFrom)
		if err != nil {
			// 尝试其他格式
			validFrom, err = time.Parse("2006-01-02 15:04:05", req.ValidFrom)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"message": "生效时间格式错误",
				})
				return
			}
		}
		coupon.ValidFrom = validFrom
	}
	if req.ValidUntil != "" {
		validUntil, err := time.Parse("2006-01-02T15:04:05", req.ValidUntil)
		if err != nil {
			// 尝试其他格式
			validUntil, err = time.Parse("2006-01-02 15:04:05", req.ValidUntil)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"message": "失效时间格式错误",
				})
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
	// ApplicablePackages 字段允许更新（包括空字符串）
	if req.ApplicablePackages != "" {
		coupon.ApplicablePackages = req.ApplicablePackages
	} else if req.ApplicablePackages == "" {
		// 允许清空
		coupon.ApplicablePackages = ""
	}

	if err := db.Save(&coupon).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新优惠券失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "更新成功",
		"data":    coupon,
	})
}

// DeleteCoupon 删除优惠券（管理员）
func DeleteCoupon(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()

	if err := db.Delete(&models.Coupon{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除优惠券失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "删除成功",
	})
}
