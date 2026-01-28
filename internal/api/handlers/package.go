package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

func GetPackages(c *gin.Context) {
	db := database.GetDB()

	var packages []models.Package
	if err := db.Where("is_active = ?", true).Order("sort_order ASC").Find(&packages).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取套餐列表失败", err)
		return
	}

	result := make([]gin.H, 0)
	for _, pkg := range packages {
		result = append(result, gin.H{
			"id":             pkg.ID,
			"name":           pkg.Name,
			"description":    pkg.Description.String,
			"price":          pkg.Price,
			"duration_days":  pkg.DurationDays,
			"device_limit":   pkg.DeviceLimit,
			"sort_order":     pkg.SortOrder,
			"is_active":      pkg.IsActive,
			"is_recommended": pkg.IsRecommended,
			"created_at":     pkg.CreatedAt.Format("2006-01-02 15:04:05"),
			"updated_at":     pkg.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

func GetPackage(c *gin.Context) {
	id := c.Param("id")

	db := database.GetDB()
	var pkg models.Package
	if err := db.First(&pkg, id).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "套餐不存在", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", pkg)
}

func CreatePackage(c *gin.Context) {
	var req struct {
		Name          string  `json:"name" binding:"required"`
		Description   string  `json:"description"`
		Price         float64 `json:"price" binding:"required"`
		DurationDays  int     `json:"duration_days" binding:"required"`
		DeviceLimit   int     `json:"device_limit"`
		SortOrder     int     `json:"sort_order"`
		IsActive      bool    `json:"is_active"`
		IsRecommended bool    `json:"is_recommended"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	pkg := models.Package{
		Name:          req.Name,
		Price:         req.Price,
		DurationDays:  req.DurationDays,
		DeviceLimit:   req.DeviceLimit,
		SortOrder:     req.SortOrder,
		IsActive:      req.IsActive,
		IsRecommended: req.IsRecommended,
	}

	if req.Description != "" {
		pkg.Description = database.NullString(req.Description)
	}

	if err := db.Create(&pkg).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建套餐失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "", pkg)
}

func UpdatePackage(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name          *string  `json:"name"`           // 使用指针，允许检测是否提供
		Description   *string  `json:"description"`    // 使用指针，允许检测是否提供
		Price         *float64 `json:"price"`          // 使用指针，允许检测是否提供
		DurationDays  *int     `json:"duration_days"`  // 使用指针，允许检测是否提供
		DeviceLimit   *int     `json:"device_limit"`   // 使用指针，允许检测是否提供
		SortOrder     *int     `json:"sort_order"`     // 使用指针，允许检测是否提供
		IsActive      *bool    `json:"is_active"`      // 使用指针，允许检测是否提供
		IsRecommended *bool    `json:"is_recommended"` // 使用指针，允许检测是否提供
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.LogError("UpdatePackage: bind JSON failed", err, map[string]interface{}{
			"package_id": id,
		})
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	var pkg models.Package
	if err := db.First(&pkg, id).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "套餐不存在", err)
		return
	}

	if req.Name != nil {
		nameValue := strings.TrimSpace(*req.Name)
		if nameValue == "" {
			utils.ErrorResponse(c, http.StatusBadRequest, "套餐名称不能为空", nil)
			return
		}
		pkg.Name = nameValue
	}
	if req.Description != nil {
		descValue := strings.TrimSpace(*req.Description)
		utils.LogInfo("UpdatePackage: 更新描述字段 - package_id=%s, description_value=%q, trimmed_length=%d", id, *req.Description, len(descValue))
		if descValue == "" {
			pkg.Description = sql.NullString{Valid: false}
			utils.LogInfo("UpdatePackage: 描述为空，设置为无效")
		} else {
			pkg.Description = database.NullString(descValue)
			utils.LogInfo("UpdatePackage: 描述已更新为: %q", descValue)
		}
	} else {
		utils.LogInfo("UpdatePackage: 描述字段未提供，不更新 - package_id=%s", id)
	}
	if req.Price != nil {
		if *req.Price < 0 {
			utils.ErrorResponse(c, http.StatusBadRequest, "价格不能为负数", nil)
			return
		}
		pkg.Price = *req.Price
	}
	if req.DurationDays != nil {
		if *req.DurationDays < 1 {
			utils.ErrorResponse(c, http.StatusBadRequest, "时长必须大于0", nil)
			return
		}
		pkg.DurationDays = *req.DurationDays
	}
	if req.DeviceLimit != nil {
		if *req.DeviceLimit < 0 {
			utils.ErrorResponse(c, http.StatusBadRequest, "设备限制不能为负数", nil)
			return
		}
		pkg.DeviceLimit = *req.DeviceLimit
	}
	if req.SortOrder != nil {
		pkg.SortOrder = *req.SortOrder
	}
	if req.IsActive != nil {
		pkg.IsActive = *req.IsActive
	}
	if req.IsRecommended != nil {
		pkg.IsRecommended = *req.IsRecommended
	}

	if err := db.Save(&pkg).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新套餐失败", err)
		return
	}

	responseData := gin.H{
		"id":             pkg.ID,
		"name":           pkg.Name,
		"description":    pkg.Description.String, // 确保返回字符串
		"price":          pkg.Price,
		"duration_days":  pkg.DurationDays,
		"device_limit":   pkg.DeviceLimit,
		"sort_order":     pkg.SortOrder,
		"is_active":      pkg.IsActive,
		"is_recommended": pkg.IsRecommended,
		"created_at":     pkg.CreatedAt.Format("2006-01-02 15:04:05"),
		"updated_at":     pkg.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	utils.SuccessResponse(c, http.StatusOK, "更新成功", responseData)
}

func DeletePackage(c *gin.Context) {
	id := c.Param("id")

	db := database.GetDB()
	if err := db.Delete(&models.Package{}, id).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除套餐失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "删除成功", nil)
}

func GetAdminPackages(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.Package{})

	page := 1
	size := 20
	if pageStr := c.Query("page"); pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
	}
	if sizeStr := c.Query("size"); sizeStr != "" {
		fmt.Sscanf(sizeStr, "%d", &size)
	}
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}

	if name := c.Query("name"); name != "" {
		sanitizedName := utils.SanitizeSearchKeyword(name)
		if sanitizedName != "" {
			query = query.Where("name LIKE ?", "%"+sanitizedName+"%")
		}
	}

	if status := c.Query("status"); status != "" {
		switch status {
		case "active":
			query = query.Where("is_active = ?", true)
		case "inactive":
			query = query.Where("is_active = ?", false)
		}
	}

	var total int64
	query.Count(&total)

	var packages []models.Package
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("sort_order ASC, created_at DESC").Find(&packages).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取套餐列表失败", err)
		return
	}

	formattedPackages := make([]gin.H, 0, len(packages))
	for _, pkg := range packages {
		formattedPackages = append(formattedPackages, gin.H{
			"id":             pkg.ID,
			"name":           pkg.Name,
			"description":    pkg.Description.String, // 确保返回字符串
			"price":          pkg.Price,
			"duration_days":  pkg.DurationDays,
			"device_limit":   pkg.DeviceLimit,
			"sort_order":     pkg.SortOrder,
			"is_active":      pkg.IsActive,
			"is_recommended": pkg.IsRecommended,
			"created_at":     pkg.CreatedAt.Format("2006-01-02 15:04:05"),
			"updated_at":     pkg.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"packages": formattedPackages,
		"total":    total,
		"page":     page,
		"size":     size,
	})
}
