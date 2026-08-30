package handlers

import (
	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 用户端 - 获取当前有效促销活动
func GetActivePromotions(c *gin.Context) {
	db := database.GetDB()
	now := utils.GetBeijingTime()
	var promotions []models.Promotion
	db.Where("is_active = ? AND start_time <= ? AND end_time >= ?", true, now, now).
		Order("created_at DESC").Limit(100).Find(&promotions)
	utils.SuccessResponse(c, http.StatusOK, "", promotions)
}

// 管理端 - 获取所有促销活动
func GetAdminPromotions(c *gin.Context) {
	db := database.GetDB()
	p := utils.ParsePaginationWithDefaultSize(c, 20)
	page, pageSize := p.Page, p.Size

	var total int64
	db.Model(&models.Promotion{}).Count(&total)

	var promotions []models.Promotion
	db.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&promotions)

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"list":  promotions,
		"total": total,
		"page":  page,
	})
}

// promotionRequest 管理端创建/更新营销活动请求。
// 前端 el-date-picker value-format="YYYY-MM-DD HH:mm:ss"，Go time.Time JSON 只接受 RFC3339，
// 因此用 string 接收后再按两种格式解析，避免保存必失败。
type promotionRequest struct {
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	DiscountType  string  `json:"discount_type"`
	DiscountValue float64 `json:"discount_value"`
	MinAmount     float64 `json:"min_amount"`
	MaxDiscount   float64 `json:"max_discount"`
	PackageIDs    string  `json:"package_ids"`
	StartTime     string  `json:"start_time"`
	EndTime       string  `json:"end_time"`
	IsActive      bool    `json:"is_active"`
	Description   string  `json:"description"`
}

// parsePromotionTime 解析前端时间字符串，兼容 "2006-01-02 15:04:05" 与 RFC3339
func parsePromotionTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z07:00",
		time.RFC3339,
	}
	var lastErr error
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, utils.BeijingTZ); err == nil {
			return t, nil
		} else {
			lastErr = err
		}
	}
	return time.Time{}, lastErr
}

func applyPromotionRequest(promo *models.Promotion, req *promotionRequest) error {
	promo.Name = req.Name
	promo.Type = req.Type
	promo.DiscountType = req.DiscountType
	promo.DiscountValue = req.DiscountValue
	promo.MinAmount = req.MinAmount
	promo.MaxDiscount = req.MaxDiscount
	if req.PackageIDs != "" {
		promo.PackageIDs = sql.NullString{String: req.PackageIDs, Valid: true}
	} else {
		promo.PackageIDs = sql.NullString{}
	}
	if req.StartTime != "" {
		t, err := parsePromotionTime(req.StartTime)
		if err != nil {
			return fmt.Errorf("开始时间格式错误")
		}
		promo.StartTime = t
	}
	if req.EndTime != "" {
		t, err := parsePromotionTime(req.EndTime)
		if err != nil {
			return fmt.Errorf("结束时间格式错误")
		}
		promo.EndTime = t
	}
	promo.IsActive = req.IsActive
	if req.Description != "" {
		promo.Description = sql.NullString{String: req.Description, Valid: true}
	} else {
		promo.Description = sql.NullString{}
	}
	return nil
}

// 管理端 - 创建促销活动
func CreatePromotion(c *gin.Context) {
	var req promotionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", nil)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "活动名称不能为空", nil)
		return
	}
	if strings.TrimSpace(req.Type) == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "活动类型不能为空", nil)
		return
	}
	if strings.TrimSpace(req.DiscountType) == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "优惠类型不能为空", nil)
		return
	}

	var promo models.Promotion
	if err := applyPromotionRequest(&promo, &req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	db := database.GetDB()
	if err := db.Create(&promo).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建失败", nil)
		return
	}
	utils.CreateAuditLogSimple(c, "create_promotion", "promotion", promo.ID, fmt.Sprintf("创建营销活动: %s", promo.Name))
	utils.SuccessResponse(c, http.StatusOK, "创建成功", promo)
}

// 管理端 - 更新促销活动
func UpdatePromotion(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()
	var promo models.Promotion
	if err := db.First(&promo, id).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "活动不存在", nil)
		return
	}
	var req promotionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", nil)
		return
	}
	if err := applyPromotionRequest(&promo, &req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	db.Save(&promo)
	utils.CreateAuditLogSimple(c, "update_promotion", "promotion", promo.ID, fmt.Sprintf("更新营销活动: %s", promo.Name))
	utils.SuccessResponse(c, http.StatusOK, "更新成功", promo)
}

// 管理端 - 删除促销活动
func DeletePromotion(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()
	var promo models.Promotion
	if err := db.First(&promo, id).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "活动不存在", nil)
		return
	}
	db.Delete(&models.Promotion{}, id)
	utils.CreateAuditLogSimple(c, "delete_promotion", "promotion", promo.ID, fmt.Sprintf("删除营销活动: %s", promo.Name))
	utils.SuccessResponse(c, http.StatusOK, "删除成功", nil)
}

// 用户端 - 参与营销活动
func ParticipatePromotion(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	promotionID := c.Param("id")
	db := database.GetDB()

	// 查询活动
	var promotion models.Promotion
	if err := db.First(&promotion, promotionID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "活动不存在", nil)
		return
	}

	// 验证活动是否有效
	now := utils.GetBeijingTime()
	if !promotion.IsActive {
		utils.ErrorResponse(c, http.StatusBadRequest, "活动已停用", nil)
		return
	}
	if now.Before(promotion.StartTime) {
		utils.ErrorResponse(c, http.StatusBadRequest, "活动尚未开始", nil)
		return
	}
	if now.After(promotion.EndTime) {
		utils.ErrorResponse(c, http.StatusBadRequest, "活动已结束", nil)
		return
	}

	// 检查用户是否已参与（快速失败；事务内还会复核，防止并发双领）
	var existingParticipation models.PromotionParticipation
	if err := db.Where("promotion_id = ? AND user_id = ?", promotion.ID, user.ID).First(&existingParticipation).Error; err == nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "您已参与过此活动", nil)
		return
	}

	// 根据活动类型应用奖励（重复参与检查在事务内进行，防止并发双领）
	err := utils.WithTransaction(db, func(tx *gorm.DB) error {
		// 事务内锁定用户行，串行化同一用户的并发参与请求
		var lockedUser models.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lockedUser, user.ID).Error; err != nil {
			return fmt.Errorf("锁定用户失败: %v", err)
		}

		// 在事务内复核是否已参与（唯一索引 (promotion_id, user_id) 作为数据库兜底）
		var existingParticipation models.PromotionParticipation
		if err := tx.Where("promotion_id = ? AND user_id = ?", promotion.ID, user.ID).First(&existingParticipation).Error; err == nil {
			return fmt.Errorf("您已参与过此活动")
		}

		participation := models.PromotionParticipation{
			PromotionID: promotion.ID,
			UserID:      user.ID,
			RewardType:  promotion.DiscountType,
			RewardValue: promotion.DiscountValue,
			Status:      "completed",
			AppliedAt:   database.NullTime(now),
		}

		switch promotion.DiscountType {
		case "balance":
			// 赠送余额
			if err := tx.Model(&models.User{}).Where("id = ?", user.ID).
				Update("balance", gorm.Expr("balance + ?", promotion.DiscountValue)).Error; err != nil {
				return fmt.Errorf("赠送余额失败: %v", err)
			}

			// 记录余额日志
			var updatedUser models.User
			if err := tx.First(&updatedUser, user.ID).Error; err != nil {
				return fmt.Errorf("查询用户失败: %v", err)
			}

			oldBalance := updatedUser.Balance - promotion.DiscountValue
			userID := user.ID
			ipAddress := utils.GetRealClientIP(c)
			if err := utils.CreateBalanceLog(
				user.ID,
				"promotion",
				promotion.DiscountValue,
				oldBalance,
				updatedUser.Balance,
				nil,
				nil,
				fmt.Sprintf("参与营销活动「%s」赠送余额", promotion.Name),
				"system",
				&userID,
				ipAddress,
			); err != nil {
				utils.LogError("ParticipatePromotion: failed to create balance log", err, nil)
			}

			participation.RewardType = "balance"

		case "free_days":
			// 赠送天数
			var subscription models.Subscription
			if err := tx.Where("user_id = ?", user.ID).First(&subscription).Error; err != nil {
				return fmt.Errorf("您还没有订阅，无法领取赠送天数")
			}

			days := int(promotion.DiscountValue)
			if subscription.ExpireTime.Before(now) {
				subscription.ExpireTime = now.AddDate(0, 0, days)
			} else {
				subscription.ExpireTime = subscription.ExpireTime.AddDate(0, 0, days)
			}

			if err := tx.Save(&subscription).Error; err != nil {
				return fmt.Errorf("赠送天数失败: %v", err)
			}

			participation.RewardType = "free_days"

		case "percentage", "fixed":
			// 折扣券类型，不立即应用，在下次购买时使用
			// 设置过期时间为活动结束时间
			participation.Status = "pending"
			participation.ExpireAt = database.NullTime(promotion.EndTime)
			participation.RewardType = "discount"

		default:
			return fmt.Errorf("不支持的奖励类型: %s", promotion.DiscountType)
		}

		if err := tx.Create(&participation).Error; err != nil {
			return fmt.Errorf("创建参与记录失败: %v", err)
		}

		return nil
	})

	if err != nil {
		if strings.Contains(err.Error(), "您已参与过此活动") {
			utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	// 返回成功信息
	var message string
	switch promotion.DiscountType {
	case "balance":
		message = fmt.Sprintf("恭喜！您已成功领取 ¥%.2f 余额", promotion.DiscountValue)
	case "free_days":
		message = fmt.Sprintf("恭喜！您已成功领取 %d 天订阅时长", int(promotion.DiscountValue))
	case "percentage":
		message = fmt.Sprintf("恭喜！您已成功领取 %.0f%% 折扣券，下次购买时自动使用", promotion.DiscountValue)
	case "fixed":
		message = fmt.Sprintf("恭喜！您已成功领取 ¥%.2f 优惠券，下次购买时自动使用", promotion.DiscountValue)
	default:
		message = "恭喜！您已成功参与活动"
	}

	utils.SuccessResponse(c, http.StatusOK, message, nil)
}

// 用户端 - 获取我的活动参与记录
func GetMyPromotionParticipations(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	db := database.GetDB()
	var participations []models.PromotionParticipation
	if err := db.Where("user_id = ?", user.ID).
		Preload("Promotion").
		Order("created_at DESC").
		Find(&participations).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "查询失败", nil)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", participations)
}

// 用户端 - 获取可用的活动折扣
func GetAvailablePromotionDiscounts(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	db := database.GetDB()
	now := utils.GetBeijingTime()

	// 查询用户未使用的折扣券
	var participations []models.PromotionParticipation
	if err := db.Where("user_id = ? AND status = ? AND reward_type = ? AND (order_id IS NULL OR order_id = 0) AND (expire_at IS NULL OR expire_at > ?)",
		user.ID, "pending", "discount", now).
		Preload("Promotion").
		Order("created_at DESC").
		Find(&participations).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "查询失败", nil)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", participations)
}
