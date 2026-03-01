package handlers

import (
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetEmailTemplateByName 根据名称获取邮件模板
func GetEmailTemplateByName(c *gin.Context) {
	name := c.Param("name")
	db := database.GetDB()

	var template models.EmailTemplate
	if err := db.Where("name = ? AND is_active = ?", name, true).First(&template).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "模板不存在或未启用", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", template)
}

// GetEmailTemplates 获取所有邮件模板
func GetEmailTemplates(c *gin.Context) {
	db := database.GetDB()

	var templates []models.EmailTemplate
	if err := db.Where("is_active = ?", true).Order("name").Find(&templates).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取模板列表失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", templates)
}
