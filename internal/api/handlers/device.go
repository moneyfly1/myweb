package handlers

import (
	"net/http"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"

	"github.com/gin-gonic/gin"
)

// GetDevices 获取设备列表
func GetDevices(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	db := database.GetDB()
	var devices []models.Device
	if err := db.Where("user_id = ?", user.ID).Preload("Subscription").Find(&devices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取设备列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    devices,
	})
}

// DeleteDevice 删除设备
func DeleteDevice(c *gin.Context) {
	id := c.Param("id")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	db := database.GetDB()
	var device models.Device
	if err := db.Where("id = ? AND user_id = ?", id, user.ID).First(&device).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "设备不存在",
		})
		return
	}

	// 更新订阅的设备数量
	var subscription models.Subscription
	if err := db.First(&subscription, device.SubscriptionID).Error; err == nil {
		if subscription.CurrentDevices > 0 {
			subscription.CurrentDevices--
			db.Save(&subscription)
		}
	}

	if err := db.Delete(&device).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除设备失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "删除成功",
	})
}

// GetDeviceStats 获取设备统计（管理员）
func GetDeviceStats(c *gin.Context) {
	db := database.GetDB()

	var stats struct {
		TotalDevices       int64 `json:"total_devices"`
		ActiveDevices      int64 `json:"active_devices"`
		TotalSubscriptions int64 `json:"total_subscriptions"`
	}

	db.Model(&models.Device{}).Count(&stats.TotalDevices)
	db.Model(&models.Device{}).Where("is_active = ?", true).Count(&stats.ActiveDevices)
	db.Model(&models.Subscription{}).Count(&stats.TotalSubscriptions)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}
