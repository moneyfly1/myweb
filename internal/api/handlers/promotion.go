package handlers

import (
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

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

// 管理端 - 创建促销活动
func CreatePromotion(c *gin.Context) {
	var promo models.Promotion
	if err := c.ShouldBindJSON(&promo); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", nil)
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
	if err := c.ShouldBindJSON(&promo); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", nil)
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
