package discount

import (
	"cboard-go/internal/models"
	"cboard-go/internal/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CouponQuoteOptions struct {
	Code               string
	UserID             uint
	PackageID          uint
	BaseAmount         float64
	ApplyLevelDiscount bool
}

type CouponQuote struct {
	Coupon               *models.Coupon
	BaseAmount           float64
	AmountBeforeCoupon   float64
	LevelDiscountAmount  float64
	CouponDiscountAmount float64
	TotalDiscountAmount  float64
	FinalAmount          float64
	FreeDays             int
}

func CalculateLevelDiscount(db *gorm.DB, userID uint, baseAmount float64) (float64, float64) {
	baseAmount = utils.RoundFloat(baseAmount, 2)
	if userID == 0 || baseAmount <= 0 {
		return 0, baseAmount
	}

	var user models.User
	if err := db.First(&user, userID).Error; err != nil || !user.UserLevelID.Valid {
		return 0, baseAmount
	}

	var level models.UserLevel
	if err := db.First(&level, user.UserLevelID.Int64).Error; err != nil {
		return 0, baseAmount
	}
	if level.DiscountRate <= 0 || level.DiscountRate >= 1.0 {
		return 0, baseAmount
	}

	discountAmount := utils.RoundFloat(baseAmount*(1.0-level.DiscountRate), 2)
	amountAfterDiscount := utils.RoundFloat(baseAmount-discountAmount, 2)
	if amountAfterDiscount < 0 {
		amountAfterDiscount = 0
	}
	return discountAmount, amountAfterDiscount
}

func QuoteCoupon(db *gorm.DB, opts CouponQuoteOptions) (*CouponQuote, error) {
	code := strings.TrimSpace(opts.Code)
	if code == "" {
		return nil, fmt.Errorf("请输入优惠券码")
	}

	baseAmount := utils.RoundFloat(opts.BaseAmount, 2)
	if baseAmount < 0 {
		return nil, fmt.Errorf("订单金额无效")
	}

	levelDiscountAmount := 0.0
	amountBeforeCoupon := baseAmount
	if opts.ApplyLevelDiscount && opts.UserID > 0 {
		levelDiscountAmount, amountBeforeCoupon = CalculateLevelDiscount(db, opts.UserID, baseAmount)
	}
	return quoteCouponWithPreparedAmount(db, code, opts.UserID, opts.PackageID, baseAmount, amountBeforeCoupon, levelDiscountAmount)
}

func QuoteCouponForPreparedAmount(db *gorm.DB, code string, userID uint, packageID uint, amountBeforeCoupon float64) (*CouponQuote, error) {
	preparedAmount := utils.RoundFloat(amountBeforeCoupon, 2)
	if preparedAmount < 0 {
		return nil, fmt.Errorf("订单金额无效")
	}
	return quoteCouponWithPreparedAmount(db, code, userID, packageID, preparedAmount, preparedAmount, 0)
}

func quoteCouponWithPreparedAmount(db *gorm.DB, code string, userID uint, packageID uint, baseAmount float64, amountBeforeCoupon float64, levelDiscountAmount float64) (*CouponQuote, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, fmt.Errorf("请输入优惠券码")
	}
	var coupon models.Coupon
	if err := db.Where("code = ? AND status = ?", code, "active").First(&coupon).Error; err != nil {
		return nil, fmt.Errorf("优惠券不存在或已失效")
	}
	if err := ValidateCoupon(db, &coupon, userID, packageID, amountBeforeCoupon); err != nil {
		return nil, err
	}

	couponDiscountAmount, freeDays, err := CalculateCouponBenefit(coupon, amountBeforeCoupon)
	if err != nil {
		return nil, err
	}

	finalAmount := utils.RoundFloat(amountBeforeCoupon-couponDiscountAmount, 2)
	if finalAmount < 0 {
		finalAmount = 0
	}

	return &CouponQuote{
		Coupon:               &coupon,
		BaseAmount:           baseAmount,
		AmountBeforeCoupon:   amountBeforeCoupon,
		LevelDiscountAmount:  levelDiscountAmount,
		CouponDiscountAmount: couponDiscountAmount,
		TotalDiscountAmount:  utils.RoundFloat(levelDiscountAmount+couponDiscountAmount, 2),
		FinalAmount:          finalAmount,
		FreeDays:             freeDays,
	}, nil
}

func GetCouponQuoteByCode(db *gorm.DB, code string, userID uint, packageID uint, amountBeforeCoupon float64) (*CouponQuote, error) {
	return QuoteCouponForPreparedAmount(db, code, userID, packageID, amountBeforeCoupon)
}

func ValidateCoupon(db *gorm.DB, coupon *models.Coupon, userID uint, packageID uint, amountBeforeCoupon float64) error {
	if coupon == nil || coupon.ID == 0 {
		return fmt.Errorf("优惠券不存在或已失效")
	}
	if coupon.Status != "active" {
		return fmt.Errorf("优惠券已失效")
	}

	now := utils.GetBeijingTime()
	if now.Before(coupon.ValidFrom) || now.After(coupon.ValidUntil) {
		return fmt.Errorf("优惠券不在有效期内")
	}
	if coupon.TotalQuantity.Valid && coupon.UsedQuantity >= int(coupon.TotalQuantity.Int64) {
		return fmt.Errorf("优惠券已被领完")
	}
	if !CouponAppliesToPackage(*coupon, packageID) {
		return fmt.Errorf("该优惠券不适用于当前套餐")
	}
	if coupon.MinAmount.Valid && amountBeforeCoupon < coupon.MinAmount.Float64 {
		return fmt.Errorf("订单金额不满足优惠券使用条件（最低%.2f元）", coupon.MinAmount.Float64)
	}
	if userID > 0 && coupon.MaxUsesPerUser > 0 {
		var usageCount int64
		db.Model(&models.CouponUsage{}).Where("coupon_id = ? AND user_id = ?", coupon.ID, userID).Count(&usageCount)
		if int(usageCount) >= coupon.MaxUsesPerUser {
			return fmt.Errorf("您已达到该优惠券的使用上限")
		}
	}
	return nil
}

func CalculateCouponBenefit(coupon models.Coupon, amountBeforeCoupon float64) (float64, int, error) {
	amountBeforeCoupon = utils.RoundFloat(amountBeforeCoupon, 2)
	switch coupon.Type {
	case string(models.CouponTypeDiscount):
		if coupon.DiscountValue <= 0 {
			return 0, 0, fmt.Errorf("优惠券折扣配置无效")
		}
		discountRate := math.Min(coupon.DiscountValue, 100)
		if discountRate < 0 {
			discountRate = 0
		}
		discountAmount := utils.RoundFloat(amountBeforeCoupon*(discountRate/100), 2)
		if coupon.MaxDiscount.Valid && discountAmount > coupon.MaxDiscount.Float64 {
			discountAmount = utils.RoundFloat(coupon.MaxDiscount.Float64, 2)
		}
		if discountAmount > amountBeforeCoupon {
			discountAmount = amountBeforeCoupon
		}
		return discountAmount, 0, nil
	case string(models.CouponTypeFixed):
		if coupon.DiscountValue <= 0 {
			return 0, 0, fmt.Errorf("优惠券减免金额配置无效")
		}
		discountAmount := utils.RoundFloat(coupon.DiscountValue, 2)
		if discountAmount > amountBeforeCoupon {
			discountAmount = amountBeforeCoupon
		}
		return discountAmount, 0, nil
	case string(models.CouponTypeFreeDays):
		freeDays := int(math.Round(coupon.DiscountValue))
		if freeDays <= 0 {
			return 0, 0, fmt.Errorf("优惠券赠送天数配置无效")
		}
		return 0, freeDays, nil
	default:
		return 0, 0, fmt.Errorf("不支持的优惠券类型: %s", coupon.Type)
	}
}

func ReserveCouponUsageTx(tx *gorm.DB, couponID uint, userID uint, orderID uint, discountAmount float64) error {
	if couponID == 0 || userID == 0 || orderID == 0 {
		return fmt.Errorf("优惠券使用记录参数无效")
	}
	var existingUsage models.CouponUsage
	if err := tx.Where("coupon_id = ? AND order_id = ?", couponID, orderID).First(&existingUsage).Error; err == nil {
		return nil
	} else if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("检查优惠券使用记录失败: %v", err)
	}

	var coupon models.Coupon
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&coupon, couponID).Error; err != nil {
		return fmt.Errorf("锁定优惠券失败: %v", err)
	}
	if coupon.Status != "active" {
		return fmt.Errorf("优惠券已失效")
	}
	now := utils.GetBeijingTime()
	if now.Before(coupon.ValidFrom) || now.After(coupon.ValidUntil) {
		return fmt.Errorf("优惠券不在有效期内")
	}
	if coupon.TotalQuantity.Valid && coupon.UsedQuantity >= int(coupon.TotalQuantity.Int64) {
		return fmt.Errorf("优惠券已被领完")
	}
	if coupon.MaxUsesPerUser > 0 {
		var usageCount int64
		if err := tx.Model(&models.CouponUsage{}).Where("coupon_id = ? AND user_id = ?", couponID, userID).Count(&usageCount).Error; err != nil {
			return fmt.Errorf("检查优惠券使用次数失败: %v", err)
		}
		if int(usageCount) >= coupon.MaxUsesPerUser {
			return fmt.Errorf("您已达到该优惠券的使用上限")
		}
	}

	usage := models.CouponUsage{
		CouponID:       couponID,
		UserID:         userID,
		OrderID:        sql.NullInt64{Int64: utils.MustSafeUintToInt64(orderID), Valid: true},
		DiscountAmount: utils.RoundFloat(discountAmount, 2),
	}
	if err := tx.Create(&usage).Error; err != nil {
		return fmt.Errorf("记录优惠券使用失败: %v", err)
	}
	if err := tx.Model(&models.Coupon{}).Where("id = ?", couponID).
		UpdateColumn("used_quantity", gorm.Expr("used_quantity + 1")).Error; err != nil {
		return fmt.Errorf("更新优惠券使用次数失败: %v", err)
	}
	return nil
}

func ReleaseCouponUsageForOrderTx(tx *gorm.DB, orderID uint) error {
	var usages []models.CouponUsage
	if err := tx.Where("order_id = ?", orderID).Find(&usages).Error; err != nil {
		return err
	}
	for _, usage := range usages {
		if err := tx.Model(&models.Coupon{}).Where("id = ?", usage.CouponID).
			UpdateColumn("used_quantity", gorm.Expr("CASE WHEN used_quantity > 0 THEN used_quantity - 1 ELSE 0 END")).Error; err != nil {
			return err
		}
	}
	if len(usages) > 0 {
		if err := tx.Where("order_id = ?", orderID).Delete(&models.CouponUsage{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func CouponAppliesToPackage(coupon models.Coupon, packageID uint) bool {
	values := parseApplicablePackages(coupon.ApplicablePackages)
	if len(values) == 0 {
		return true
	}

	if packageID == 0 {
		return values["0"] || values["custom"] || values["custom_package"] || values["all"]
	}
	return values[strconv.FormatUint(uint64(packageID), 10)] || values["all"]
}

func parseApplicablePackages(raw string) map[string]bool {
	result := make(map[string]bool)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return result
	}

	var ids []uint
	if err := json.Unmarshal([]byte(raw), &ids); err == nil {
		for _, id := range ids {
			result[strconv.FormatUint(uint64(id), 10)] = true
		}
		return result
	}

	var mixed []interface{}
	if err := json.Unmarshal([]byte(raw), &mixed); err == nil {
		for _, item := range mixed {
			switch v := item.(type) {
			case float64:
				result[strconv.FormatUint(uint64(v), 10)] = true
			case string:
				addApplicablePackageToken(result, v)
			}
		}
		return result
	}

	for _, part := range strings.Split(raw, ",") {
		addApplicablePackageToken(result, part)
	}
	return result
}

func addApplicablePackageToken(result map[string]bool, token string) {
	token = strings.TrimSpace(strings.Trim(token, `"'`))
	if token == "" {
		return
	}
	lower := strings.ToLower(token)
	if lower == "custom" || lower == "custom_package" || lower == "all" {
		result[lower] = true
		return
	}
	if id, err := strconv.ParseUint(token, 10, 64); err == nil {
		result[strconv.FormatUint(id, 10)] = true
	}
}
