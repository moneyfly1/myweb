package handlers

import (
	"net/http"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"

	"github.com/gin-gonic/gin"
)

func GetDevices(c *gin.Context) {
	u, _ := middleware.GetCurrentUser(c)
	db := database.GetDB()
	var devices []models.Device
	db.Where("user_id = ?", u.ID).Preload("Subscription").Find(&devices)
	c.JSON(200, gin.H{"success": true, "data": devices})
}

func DeleteDevice(c *gin.Context) {
	u, _ := middleware.GetCurrentUser(c)
	db := database.GetDB()
	var device models.Device
	if err := db.Where("id = ? AND user_id = ?", c.Param("id"), u.ID).First(&device).Error; err != nil {
		c.JSON(404, gin.H{"success": false, "message": "设备不存在"})
		return
	}
	db.Delete(&device)
	var count int64
	db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", device.SubscriptionID, true).Count(&count)
	db.Model(&models.Subscription{}).Where("id = ?", device.SubscriptionID).Update("current_devices", count)
	c.JSON(200, gin.H{"success": true, "message": "删除成功"})
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
