package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	discountService "cboard-go/internal/services/discount"
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
		Code               string  `json:"code" binding:"required"`
		Amount             float64 `json:"amount" binding:"required"`
		PackageID          uint    `json:"package_id"`
		ApplyLevelDiscount *bool   `json:"apply_level_discount"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	// 获取当前用户（如果已登录）
	user, userExists := middleware.GetCurrentUser(c)
	userID := uint(0)
	if userExists {
		userID = user.ID
	}
	applyLevelDiscount := userExists
	if req.ApplyLevelDiscount != nil {
		applyLevelDiscount = userExists && *req.ApplyLevelDiscount
	}

	db := database.GetDB()
	quote, err := discountService.QuoteCoupon(db, discountService.CouponQuoteOptions{
		Code:               req.Code,
		UserID:             userID,
		PackageID:          req.PackageID,
		BaseAmount:         req.Amount,
		ApplyLevelDiscount: applyLevelDiscount,
	})
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "不存在") {
			status = http.StatusNotFound
		}
		utils.ErrorResponse(c, status, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"coupon":                 quote.Coupon,
		"base_amount":            quote.BaseAmount,
		"amount_before_coupon":   quote.AmountBeforeCoupon,
		"level_discount_amount":  quote.LevelDiscountAmount,
		"coupon_discount_amount": quote.CouponDiscountAmount,
		"discount_amount":        quote.CouponDiscountAmount,
		"total_discount_amount":  quote.TotalDiscountAmount,
		"final_amount":           quote.FinalAmount,
		"free_days":              quote.FreeDays,
		"valid":                  true,
		"message":                "优惠券验证成功",
	})
}

func normalizeApplicablePackagesInput(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	var arr []interface{}
	if err := json.Unmarshal([]byte(raw), &arr); err == nil {
		ids := make([]string, 0, len(arr))
		for _, item := range arr {
			switch v := item.(type) {
			case float64:
				ids = append(ids, fmt.Sprintf("%.0f", v))
			case string:
				token := strings.TrimSpace(v)
				if token != "" {
					ids = append(ids, token)
				}
			}
		}
		return strings.Join(ids, ",")
	}

	parts := strings.Split(raw, ",")
	ids := make([]string, 0, len(parts))
	for _, part := range parts {
		token := strings.TrimSpace(strings.Trim(part, `"'`))
		if token != "" {
			ids = append(ids, token)
		}
	}
	return strings.Join(ids, ",")
}

func validateCouponConfig(coupon models.Coupon) error {
	switch coupon.Type {
	case string(models.CouponTypeDiscount):
		if coupon.DiscountValue <= 0 || coupon.DiscountValue > 100 {
			return fmt.Errorf("折扣优惠券的优惠值必须在 0 到 100 之间")
		}
	case string(models.CouponTypeFixed):
		if coupon.DiscountValue <= 0 {
			return fmt.Errorf("固定金额优惠券的优惠值必须大于 0")
		}
	case string(models.CouponTypeFreeDays):
		if coupon.DiscountValue <= 0 {
			return fmt.Errorf("赠送天数优惠券的天数必须大于 0")
		}
	default:
		return fmt.Errorf("不支持的优惠券类型: %s", coupon.Type)
	}
	if !coupon.ValidUntil.After(coupon.ValidFrom) {
		return fmt.Errorf("失效时间必须晚于生效时间")
	}
	if coupon.MaxUsesPerUser < 1 {
		return fmt.Errorf("每用户限用次数必须大于 0")
	}
	if coupon.TotalQuantity.Valid && coupon.TotalQuantity.Int64 > 0 && int64(coupon.UsedQuantity) > coupon.TotalQuantity.Int64 {
		return fmt.Errorf("总数量不能小于已使用数量")
	}
	return nil
}

func parseCouponTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("时间不能为空")
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z07:00"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.In(utils.BeijingTZ), nil
		}
	}
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02 15:04:05"} {
		if parsed, err := time.ParseInLocation(layout, value, utils.BeijingTZ); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("时间格式错误")
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

	validFrom, err := parseCouponTime(req.ValidFrom)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "生效时间格式错误", err)
		return
	}
	validUntil, err := parseCouponTime(req.ValidUntil)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "失效时间格式错误", err)
		return
	}

	code := req.Code
	if code == "" {
		code = utils.GenerateCouponCode()
		var existing models.Coupon
		maxRetries := 20
		for db.Where("code = ?", code).First(&existing).Error == nil {
			maxRetries--
			if maxRetries <= 0 {
				utils.ErrorResponse(c, http.StatusInternalServerError, "生成唯一优惠券码失败，请重试", nil)
				return
			}
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
		coupon.ApplicablePackages = normalizeApplicablePackagesInput(req.ApplicablePackages)
	}

	if req.Description != "" {
		coupon.Description = req.Description
	}
	if req.MinAmount > 0 {
		coupon.MinAmount = database.NullFloat64(req.MinAmount)
	}
	if req.MaxDiscount > 0 {
		coupon.MaxDiscount = database.NullFloat64(req.MaxDiscount)
	} else if req.MaxDiscount == 0 {
		coupon.MaxDiscount = sql.NullFloat64{}
	}
	if req.TotalQuantity > 0 {
		coupon.TotalQuantity = database.NullInt64(int64(req.TotalQuantity))
	} else if req.TotalQuantity == 0 {
		coupon.TotalQuantity = sql.NullInt64{}
	}
	if req.MaxUsesPerUser > 0 {
		coupon.MaxUsesPerUser = req.MaxUsesPerUser
	} else {
		coupon.MaxUsesPerUser = 1 // 默认值
	}
	if coupon.Type == string(models.CouponTypeFreeDays) {
		coupon.MinAmount = sql.NullFloat64{}
		coupon.MaxDiscount = sql.NullFloat64{}
	} else if coupon.Type == string(models.CouponTypeFixed) {
		coupon.MaxDiscount = sql.NullFloat64{}
	}
	if err := validateCouponConfig(coupon); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	if err := db.Create(&coupon).Error; err != nil {
		utils.LogError("CreateCoupon: create coupon failed", err, nil)
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建优惠券失败", err)
		return
	}
	utils.CreateAuditLogSimple(c, "create_coupon", "coupon", coupon.ID, fmt.Sprintf("管理员操作: 创建优惠券 %s", coupon.Name))
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
		Name               string   `json:"name"`
		Description        string   `json:"description"`
		Type               string   `json:"type"`
		DiscountValue      float64  `json:"discount_value"`
		MinAmount          *float64 `json:"min_amount"`
		MaxDiscount        *float64 `json:"max_discount"`
		ValidFrom          string   `json:"valid_from"`
		ValidUntil         string   `json:"valid_until"`
		TotalQuantity      *int     `json:"total_quantity"`
		MaxUsesPerUser     *int     `json:"max_uses_per_user"`
		Status             string   `json:"status"`
		ApplicablePackages string   `json:"applicable_packages"`
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
	if req.MinAmount != nil {
		if *req.MinAmount > 0 {
			coupon.MinAmount = database.NullFloat64(*req.MinAmount)
		} else {
			coupon.MinAmount = sql.NullFloat64{}
		}
	}
	if req.MaxDiscount != nil {
		if *req.MaxDiscount > 0 {
			coupon.MaxDiscount = database.NullFloat64(*req.MaxDiscount)
		} else {
			coupon.MaxDiscount = sql.NullFloat64{}
		}
	}
	if req.ValidFrom != "" {
		validFrom, err := parseCouponTime(req.ValidFrom)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "生效时间格式错误", err)
			return
		}
		coupon.ValidFrom = validFrom
	}
	if req.ValidUntil != "" {
		validUntil, err := parseCouponTime(req.ValidUntil)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "失效时间格式错误", err)
			return
		}
		coupon.ValidUntil = validUntil
	}
	if req.TotalQuantity != nil {
		if *req.TotalQuantity > 0 {
			coupon.TotalQuantity = database.NullInt64(int64(*req.TotalQuantity))
		} else {
			coupon.TotalQuantity = sql.NullInt64{}
		}
	}
	if req.MaxUsesPerUser != nil {
		if *req.MaxUsesPerUser > 0 {
			coupon.MaxUsesPerUser = *req.MaxUsesPerUser
		} else {
			coupon.MaxUsesPerUser = 1
		}
	}
	if req.Status != "" {
		coupon.Status = req.Status
	}
	if req.ApplicablePackages != "" {
		coupon.ApplicablePackages = normalizeApplicablePackagesInput(req.ApplicablePackages)
	} else if req.ApplicablePackages == "" {
		coupon.ApplicablePackages = ""
	}
	if coupon.Type == string(models.CouponTypeFreeDays) {
		coupon.MinAmount = sql.NullFloat64{}
		coupon.MaxDiscount = sql.NullFloat64{}
	} else if coupon.Type == string(models.CouponTypeFixed) {
		coupon.MaxDiscount = sql.NullFloat64{}
	}
	if err := validateCouponConfig(coupon); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	if err := db.Save(&coupon).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新优惠券失败", err)
		return
	}
	utils.CreateAuditLogSimple(c, "update_coupon", "coupon", coupon.ID, fmt.Sprintf("管理员操作: 更新优惠券 %s", coupon.Name))
	utils.SuccessResponse(c, http.StatusOK, "更新成功", coupon)
}

func DeleteCoupon(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()
	var coupon models.Coupon
	if err := db.First(&coupon, id).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "优惠券不存在", err)
		return
	}
	if err := db.Delete(&coupon).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除优惠券失败", err)
		return
	}
	utils.CreateAuditLogSimple(c, "delete_coupon", "coupon", coupon.ID, fmt.Sprintf("管理员操作: 删除优惠券 %s", coupon.Name))
	utils.SuccessResponse(c, http.StatusOK, "删除成功", nil)
}
