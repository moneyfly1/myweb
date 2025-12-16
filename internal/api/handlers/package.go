package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

// GetPackages 获取套餐列表
func GetPackages(c *gin.Context) {
	db := database.GetDB()

	var packages []models.Package
	if err := db.Where("is_active = ?", true).Order("sort_order ASC").Find(&packages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取套餐列表失败",
		})
		return
	}

	// 确保返回格式正确
	result := make([]gin.H, 0)
	for _, pkg := range packages {
		result = append(result, gin.H{
			"id":              pkg.ID,
			"name":            pkg.Name,
			"description":     pkg.Description.String,
			"price":           pkg.Price,
			"duration_days":   pkg.DurationDays,
			"device_limit":    pkg.DeviceLimit,
			"bandwidth_limit": pkg.BandwidthLimit.Int64,
			"sort_order":      pkg.SortOrder,
			"is_active":       pkg.IsActive,
			"is_recommended":  pkg.IsRecommended,
			"created_at":      pkg.CreatedAt.Format("2006-01-02 15:04:05"),
			"updated_at":      pkg.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetPackage 获取单个套餐
func GetPackage(c *gin.Context) {
	id := c.Param("id")

	db := database.GetDB()
	var pkg models.Package
	if err := db.First(&pkg, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "套餐不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    pkg,
	})
}

// CreatePackage 创建套餐（管理员）
func CreatePackage(c *gin.Context) {
	var req struct {
		Name           string  `json:"name" binding:"required"`
		Description    string  `json:"description"`
		Price          float64 `json:"price" binding:"required"`
		DurationDays   int     `json:"duration_days" binding:"required"`
		DeviceLimit    int     `json:"device_limit"`
		BandwidthLimit int64   `json:"bandwidth_limit"`
		SortOrder      int     `json:"sort_order"`
		IsActive       bool    `json:"is_active"`
		IsRecommended  bool    `json:"is_recommended"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
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
	if req.BandwidthLimit > 0 {
		pkg.BandwidthLimit = database.NullInt64(req.BandwidthLimit)
	}

	if err := db.Create(&pkg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建套餐失败",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    pkg,
	})
}

// UpdatePackage 更新套餐（管理员）
func UpdatePackage(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name           string  `json:"name"`
		Description    string  `json:"description"`
		Price          float64 `json:"price"`
		DurationDays   int     `json:"duration_days"`
		DeviceLimit    int     `json:"device_limit"`
		BandwidthLimit int64   `json:"bandwidth_limit"`
		SortOrder      int     `json:"sort_order"`
		IsActive       bool    `json:"is_active"`
		IsRecommended  bool    `json:"is_recommended"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	var pkg models.Package
	if err := db.First(&pkg, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "套餐不存在",
		})
		return
	}

	// 更新字段（允许空值更新）
	if req.Name != "" {
		pkg.Name = req.Name
	}
	if req.Description != "" || req.Description == "" {
		// 允许清空描述
		if req.Description == "" {
			pkg.Description = sql.NullString{Valid: false}
		} else {
			pkg.Description = database.NullString(req.Description)
		}
	}
	if req.Price >= 0 {
		pkg.Price = req.Price
	}
	if req.DurationDays > 0 {
		pkg.DurationDays = req.DurationDays
	}
	if req.DeviceLimit > 0 {
		pkg.DeviceLimit = req.DeviceLimit
	}
	if req.BandwidthLimit >= 0 {
		if req.BandwidthLimit == 0 {
			pkg.BandwidthLimit = sql.NullInt64{Valid: false}
		} else {
			pkg.BandwidthLimit = database.NullInt64(req.BandwidthLimit)
		}
	}
	// SortOrder、IsActive 和 IsRecommended 总是更新
	pkg.SortOrder = req.SortOrder
	pkg.IsActive = req.IsActive
	pkg.IsRecommended = req.IsRecommended

	if err := db.Save(&pkg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新套餐失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "更新成功",
		"data":    pkg,
	})
}

// DeletePackage 删除套餐（管理员）
func DeletePackage(c *gin.Context) {
	id := c.Param("id")

	db := database.GetDB()
	if err := db.Delete(&models.Package{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除套餐失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "删除成功",
	})
}

// GetAdminPackages 管理员获取套餐列表（包含所有套餐，包括未激活的）
func GetAdminPackages(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.Package{})

	// 分页参数
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

	// 搜索参数
	if name := c.Query("name"); name != "" {
		// 清理和验证搜索关键词，防止SQL注入
		sanitizedName := utils.SanitizeSearchKeyword(name)
		if sanitizedName != "" {
			query = query.Where("name LIKE ?", "%"+sanitizedName+"%")
		}
	}

	// 状态筛选
	if status := c.Query("status"); status != "" {
		switch status {
		case "active":
			query = query.Where("is_active = ?", true)
		case "inactive":
			query = query.Where("is_active = ?", false)
		}
	}

	// 计算总数
	var total int64
	query.Count(&total)

	// 分页查询
	var packages []models.Package
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("sort_order ASC, created_at DESC").Find(&packages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取套餐列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"packages": packages,
			"items":    packages, // 兼容前端可能使用的 items 字段
			"total":    total,
			"page":     page,
			"size":     size,
		},
	})
}
