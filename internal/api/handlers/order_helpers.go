package handlers

import (
	"cboard-go/internal/models"
	"cboard-go/internal/utils"
	"gorm.io/gorm"
)

// buildOrderListData 构建订单列表数据（从 GetAdminOrders 提取）
func buildOrderListData(db *gorm.DB, orders []models.Order) []gin.H {
	orderList := make([]gin.H, 0, len(orders))
	for _, order := range orders {
		amount := order.Amount
		if order.FinalAmount.Valid {
			amount = order.FinalAmount.Float64
		}

		// 获取支付方式名称
		paymentMethod := ""
		if order.PaymentMethodName.Valid {
			paymentMethod = order.PaymentMethodName.String
		}

		// 获取支付时间
		paymentTime := ""
		if order.PaymentTime.Valid {
			paymentTime = order.PaymentTime.Time.Format("2006-01-02 15:04:05")
		}

		// 处理用户信息 - 如果 Preload 失败，单独查询
		var username, email string
		var userID uint
		if order.User.ID > 0 {
			userID = order.User.ID
			username = order.User.Username
			email = order.User.Email
		} else {
			// Preload 失败，单独查询用户
			var user models.User
			if err := db.First(&user, order.UserID).Error; err == nil {
				userID = user.ID
				username = user.Username
				email = user.Email
			} else {
				userID = order.UserID
				username = "已删除"
				email = "deleted"
			}
		}

		// 构建用户对象（前端期望嵌套的 user 对象）
		userInfo := gin.H{
			"id":       userID,
			"username": username,
			"email":    email,
		}

		// 处理 Package 信息（设备升级订单 PackageID 为 0）
		var packageName string
		if order.PackageID == 0 {
			packageName = "设备升级"
			// 如果有 ExtraData，尝试解析显示升级详情
			if order.ExtraData.Valid && order.ExtraData.String != "" {
				packageName = "设备升级订单"
			}
		} else {
			// 检查 Package 是否已加载
			if order.Package.ID > 0 && order.Package.ID == order.PackageID {
				packageName = order.Package.Name
			} else {
				// Preload 失败，单独查询 Package
				var pkg models.Package
				if err := db.First(&pkg, order.PackageID).Error; err == nil {
					packageName = pkg.Name
				} else {
					packageName = "未知套餐"
				}
			}
		}

		orderList = append(orderList, gin.H{
			"id":             order.ID,
			"order_no":       order.OrderNo,
			"user_id":        order.UserID,
			"user":           userInfo,
			"package_id":     order.PackageID,
			"package_name":   packageName,
			"amount":         amount,
			"payment_method": paymentMethod,
			"payment_time":   paymentTime,
			"status":         order.Status,
			"created_at":     order.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return orderList
}

