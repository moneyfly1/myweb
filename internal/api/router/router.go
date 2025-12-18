package router

import (
	"cboard-go/internal/api/handlers"
	"cboard-go/internal/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRouter 设置路由
func SetupRouter() *gin.Engine {
	r := gin.Default()

	// 中间件
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.SecurityHeadersMiddleware()) // 添加安全响应头
	r.Use(gin.Recovery())
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.RequestIDMiddleware())

	// 静态文件服务（前端构建后的文件）
	// 注意：这些路由需要在 API 路由之前注册
	r.Static("/static", "./frontend/dist/assets")
	r.StaticFile("/favicon.ico", "./frontend/dist/favicon.ico")
	r.StaticFile("/vite.svg", "./frontend/dist/vite.svg")

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"version": "1.0.0",
		})
	})

	// 维护模式中间件（在所有路由之前，但允许静态文件和健康检查）
	r.Use(middleware.MaintenanceMiddleware())

	// API 路由组
	api := r.Group("/api/v1")
	// 对API路由应用CSRF保护（除了公开的GET请求）
	api.Use(middleware.CSRFMiddleware())
	{
		// 认证相关
		auth := api.Group("/auth")
		{
			// 登录和注册使用速率限制
			auth.POST("/register", middleware.RegisterRateLimitMiddleware(), handlers.Register)
			auth.POST("/login", middleware.LoginRateLimitMiddleware(), handlers.Login)
			auth.POST("/login-json", middleware.LoginRateLimitMiddleware(), handlers.LoginJSON) // 兼容前端管理员/用户登录
			auth.POST("/refresh", handlers.RefreshToken)
			auth.POST("/logout", middleware.AuthMiddleware(), handlers.Logout)
			// 验证码发送使用速率限制
			auth.POST("/verification/send", middleware.VerifyCodeRateLimitMiddleware(), handlers.SendVerificationCode)
			auth.POST("/verification/verify", handlers.VerifyCode)
			auth.POST("/forgot-password", middleware.VerifyCodeRateLimitMiddleware(), handlers.ForgotPassword)     // 忘记密码
			auth.POST("/forgot-password-new", middleware.VerifyCodeRateLimitMiddleware(), handlers.ForgotPassword) // 别名，兼容前端
			auth.POST("/reset-password", handlers.ResetPasswordByCode)                                             // 重置密码
			auth.POST("/reset-password-new", handlers.ResetPasswordByCode)                                         // 别名，兼容前端
		}

		// 用户相关（需要认证）
		users := api.Group("/users")
		users.Use(middleware.AuthMiddleware())
		{
			users.GET("/me", handlers.GetCurrentUser)
			users.GET("/profile", handlers.GetCurrentUser) // 别名，兼容前端
			users.PUT("/me", handlers.UpdateCurrentUser)
			users.PUT("/profile", handlers.UpdateCurrentUser) // 别名，兼容前端
			users.GET("/dashboard-info", handlers.GetUserDashboard)
			users.POST("/change-password", handlers.ChangePassword)
			users.PUT("/preferences", handlers.UpdatePreferences)
			users.PUT("/preference-settings", handlers.UpdatePreferences) // 别名，兼容前端
			users.GET("/notification-settings", handlers.GetNotificationSettings)
			users.PUT("/notification-settings", handlers.UpdateUserNotificationSettings)
			users.GET("/privacy-settings", handlers.GetPrivacySettings)
			users.PUT("/privacy-settings", handlers.UpdatePrivacySettings)
			users.GET("/my-level", handlers.GetUserLevel)
			users.GET("/theme", handlers.GetUserTheme)
			users.PUT("/theme", handlers.UpdateUserTheme) // 更新用户主题
			users.GET("/login-history", handlers.GetLoginHistory)
			users.GET("/activities", handlers.GetUserActivities)
			users.GET("/subscription-resets", handlers.GetSubscriptionResets)
			users.GET("/devices", handlers.GetUserDevices)
		}

		// 订阅相关
		subscriptions := api.Group("/subscriptions")
		subscriptions.Use(middleware.AuthMiddleware())
		{
			subscriptions.GET("", handlers.GetSubscriptions)
			subscriptions.GET("/:id", handlers.GetSubscription)
			subscriptions.POST("", handlers.CreateSubscription)
			subscriptions.GET("/user-subscription", handlers.GetUserSubscription)
			subscriptions.GET("/devices", handlers.GetUserSubscriptionDevices)                 // 获取当前用户的订阅设备
			subscriptions.POST("/reset-subscription", handlers.ResetUserSubscriptionSelf)      // 用户重置自己的订阅
			subscriptions.POST("/send-subscription-email", handlers.SendSubscriptionEmailSelf) // 用户发送订阅邮件
			subscriptions.POST("/convert-to-balance", handlers.ConvertSubscriptionToBalance)   // 转换订阅为余额
			subscriptions.DELETE("/devices/:id", handlers.DeleteDevice)                        // 删除设备
		}

		// 订阅配置（公开访问，用于 Clash 等客户端，豁免CSRF）
		// 注意：虽然路径是公开的，但订阅URL本身就是密钥，只有知道URL的用户才能访问
		subscribePublic := api.Group("")
		subscribePublic.Use(middleware.CSRFExemptMiddleware())
		{
			subscribePublic.GET("/subscribe/:url", handlers.GetSubscriptionConfig)
			subscribePublic.GET("/subscriptions/clash/:url", handlers.GetSubscriptionConfig)        // 猫咪订阅（Clash YAML格式）
			subscribePublic.GET("/subscriptions/universal/:url", handlers.GetUniversalSubscription) // 通用订阅（Base64格式，适用于小火煎、v2ray等）
			// 兼容旧路径
			subscribePublic.GET("/subscriptions/ssr/:url", handlers.GetUniversalSubscription)   // 通用订阅（兼容旧路径）
			subscribePublic.GET("/subscriptions/v2ray/:url", handlers.GetUniversalSubscription) // 通用订阅（兼容旧路径）
		}

		// 订单相关
		orders := api.Group("/orders")
		orders.Use(middleware.AuthMiddleware())
		{
			orders.GET("", handlers.GetOrders)
			orders.POST("", handlers.CreateOrder)
			orders.POST("/create", handlers.CreateOrder)             // 别名，兼容前端
			orders.POST("/upgrade-devices", handlers.UpgradeDevices) // 升级设备数
			orders.GET("/stats", handlers.GetOrderStats)
			// 使用订单号的路由（放在前面，使用具体路径避免冲突）
			orders.POST("/:orderNo/pay", handlers.PayOrder)             // 支付订单
			orders.POST("/:orderNo/cancel", handlers.CancelOrderByNo)   // 通过订单号取消订单
			orders.GET("/:orderNo/status", handlers.GetOrderStatusByNo) // 通过订单号获取状态
			// 使用 ID 的路由（使用不同的路径前缀避免冲突）
			orders.GET("/id/:id", handlers.GetOrder) // 通过 ID 获取订单
		}

		// 套餐相关
		packages := api.Group("/packages")
		{
			packages.GET("", handlers.GetPackages)
			packages.GET("/:id", handlers.GetPackage)
		}

		// 支付相关
		payment := api.Group("/payment")
		payment.Use(middleware.AuthMiddleware())
		{
			payment.GET("/methods", handlers.GetPaymentMethods)
			payment.POST("", handlers.CreatePayment)
			payment.GET("/status/:id", handlers.GetPaymentStatus)
		}
		// 支付方式（公开访问）
		api.GET("/payment-methods/active", handlers.GetPaymentMethods)

		// 支付回调（不需要认证）
		api.POST("/payment/notify/:type", handlers.PaymentNotify)

		// 节点相关
		nodes := api.Group("/nodes")
		{
			nodes.GET("", handlers.GetNodes)
			nodes.GET("/stats", handlers.GetNodeStats)
			nodes.GET("/:id", handlers.GetNode)
		}
		// 节点操作（需要认证）
		nodesAuth := api.Group("/nodes")
		nodesAuth.Use(middleware.AuthMiddleware())
		{
			nodesAuth.POST("/:id/test", handlers.TestNode)
			nodesAuth.POST("/batch-test", handlers.BatchTestNodes)
			nodesAuth.POST("/import-from-clash", handlers.ImportFromClash)
		}

		// 优惠券相关
		coupons := api.Group("/coupons")
		{
			coupons.GET("", handlers.GetCoupons)
			coupons.GET("/:code", handlers.GetCoupon)
			coupons.POST("/verify", handlers.VerifyCoupon)
		}
		couponsAuth := coupons.Group("")
		couponsAuth.Use(middleware.AuthMiddleware())
		{
			couponsAuth.GET("/my", handlers.GetUserCoupons)
		}
		// 管理员优惠券
		couponsAdmin := coupons.Group("/admin")
		couponsAdmin.Use(middleware.AuthMiddleware())
		couponsAdmin.Use(middleware.AdminMiddleware())
		{
			couponsAdmin.GET("", handlers.GetAdminCoupons)
			couponsAdmin.GET("/:id", handlers.GetAdminCoupon)
			couponsAdmin.POST("", handlers.CreateCoupon) // 添加创建路由
			couponsAdmin.PUT("/:id", handlers.UpdateCoupon)
			couponsAdmin.DELETE("/:id", handlers.DeleteCoupon)
		}

		// 通知相关
		notifications := api.Group("/notifications")
		notifications.Use(middleware.AuthMiddleware())
		{
			notifications.GET("", handlers.GetNotifications)
			notifications.GET("/unread-count", handlers.GetUnreadCount)
			notifications.PUT("/:id/read", handlers.MarkAsRead)
			notifications.POST("/:id/read", handlers.MarkAsRead) // 兼容前端可能使用 POST
			notifications.PUT("/read-all", handlers.MarkAllAsRead)
			notifications.POST("/mark-all-read", handlers.MarkAllAsRead) // 兼容前端
			notifications.DELETE("/:id", handlers.DeleteNotification)
			notifications.GET("/user-notifications", handlers.GetUserNotifications)
		}
		// 管理员通知
		notificationsAdmin := api.Group("/notifications/admin")
		notificationsAdmin.Use(middleware.AuthMiddleware())
		notificationsAdmin.Use(middleware.AdminMiddleware())
		{
			notificationsAdmin.GET("/notifications", handlers.GetAdminNotifications)
			notificationsAdmin.POST("/notifications", handlers.CreateAdminNotification)
			notificationsAdmin.PUT("/notifications/:id", handlers.UpdateAdminNotification)
			notificationsAdmin.DELETE("/notifications/:id", handlers.DeleteAdminNotification)
		}

		// 工单相关
		tickets := api.Group("/tickets")
		tickets.Use(middleware.AuthMiddleware())
		{
			tickets.GET("", handlers.GetTickets)
			tickets.GET("/:id", handlers.GetTicket)
			tickets.POST("", handlers.CreateTicket)
			tickets.POST("/:id/reply", handlers.ReplyTicket)
			tickets.POST("/:id/replies", handlers.ReplyTicket) // 兼容前端调用
		}
		// 管理员工单
		ticketsAdmin := api.Group("/tickets/admin")
		ticketsAdmin.Use(middleware.AuthMiddleware())
		ticketsAdmin.Use(middleware.AdminMiddleware())
		{
			ticketsAdmin.GET("/all", handlers.GetAdminTickets)
			ticketsAdmin.GET("/statistics", handlers.GetAdminTicketStatistics)
			ticketsAdmin.GET("/:id", handlers.GetAdminTicket)
			ticketsAdmin.PUT("/:id", handlers.UpdateTicketStatus) // 通用更新接口
		}

		// 设备管理
		devices := api.Group("/devices")
		devices.Use(middleware.AuthMiddleware())
		{
			devices.GET("", handlers.GetDevices)
			devices.DELETE("/:id", handlers.DeleteDevice)
		}

		// 邀请码（公开访问，用于注册时验证）
		api.GET("/invites/validate/:code", handlers.ValidateInviteCode)

		// 邀请码（需要认证）
		invites := api.Group("/invites")
		invites.Use(middleware.AuthMiddleware())
		{
			invites.GET("", handlers.GetInviteCodes)
			invites.POST("", handlers.CreateInviteCode)
			invites.POST("/generate", handlers.CreateInviteCode) // 兼容旧路径
			invites.GET("/stats", handlers.GetInviteStats)
			invites.GET("/reward-settings", handlers.GetRewardSettings)
			invites.GET("/my-codes", handlers.GetMyInviteCodes)
			invites.PUT("/:id", handlers.UpdateInviteCode)
			invites.DELETE("/:id", handlers.DeleteInviteCode)
		}

		// 充值
		recharge := api.Group("/recharge")
		recharge.Use(middleware.AuthMiddleware())
		{
			recharge.GET("", handlers.GetRechargeRecords)
			recharge.GET("/:id", handlers.GetRechargeRecord)
			recharge.POST("", handlers.CreateRecharge)
			recharge.POST("/create", handlers.CreateRecharge) // 别名，兼容前端
			recharge.POST("/:id/cancel", handlers.CancelRecharge)
		}

		// 配置
		config := api.Group("/config")
		{
			config.GET("", handlers.GetSystemConfigs)
			config.GET("/:key", handlers.GetSystemConfig)
		}

		// 软件配置
		api.GET("/software-config", handlers.GetSoftwareConfig)
		softwareConfig := api.Group("/software-config")
		softwareConfig.Use(middleware.AuthMiddleware())
		softwareConfig.Use(middleware.AdminMiddleware())
		{
			softwareConfig.PUT("", handlers.UpdateSoftwareConfig)
		}

		// 支付配置
		paymentConfig := api.Group("/payment-config")
		paymentConfig.Use(middleware.AuthMiddleware())
		paymentConfig.Use(middleware.AdminMiddleware())
		{
			paymentConfig.GET("", handlers.GetPaymentConfig)
			paymentConfig.POST("", handlers.CreatePaymentConfig)
			paymentConfig.PUT("/:id", handlers.UpdatePaymentConfig)
		}

		// 公开设置
		settings := api.Group("/settings")
		{
			settings.GET("/public-settings", handlers.GetPublicSettings)
		}

		// 统计
		statistics := api.Group("/statistics")
		statistics.Use(middleware.AuthMiddleware())
		statistics.Use(middleware.AdminMiddleware())
		{
			statistics.GET("", handlers.GetStatistics)
			statistics.GET("/revenue", handlers.GetRevenueChart)
			statistics.GET("/users", handlers.GetUserStatistics)
			statistics.GET("/user-trend", handlers.GetUserTrend)
			statistics.GET("/revenue-trend", handlers.GetRevenueTrend)
		}

		// 管理员路由
		admin := api.Group("/admin")
		admin.Use(middleware.AuthMiddleware())
		admin.Use(middleware.AdminMiddleware())
		{
			// Dashboard 相关
			admin.GET("/dashboard", handlers.GetDashboard)
			admin.GET("/stats", handlers.GetAdminStats)
			admin.GET("/users/recent", handlers.GetRecentUsers)
			admin.GET("/orders/recent", handlers.GetRecentOrders)
			admin.GET("/users/abnormal", handlers.GetAbnormalUsers)
			admin.POST("/users/abnormal/:id/mark-normal", handlers.MarkUserNormal)

			// 用户管理
			admin.GET("/users", handlers.GetUsers)
			admin.POST("/users", handlers.CreateUser)
			admin.GET("/users/:id", handlers.GetUser)
			admin.GET("/users/:id/details", handlers.GetUserDetails)
			admin.PUT("/users/:id", handlers.UpdateUser)
			admin.PUT("/users/:id/status", handlers.UpdateUserStatus)
			admin.POST("/users/:id/unlock-login", handlers.UnlockUserLogin)
			admin.DELETE("/users/:id", handlers.DeleteUser)
			admin.POST("/users/:id/reset-password", handlers.ResetPassword)
			admin.POST("/users/:id/login-as", handlers.LoginAsUser)
			admin.POST("/users/batch-delete", handlers.BatchDeleteUsers)

			// 订单管理
			admin.GET("/orders", handlers.GetAdminOrders)
			admin.PUT("/orders/:id", handlers.UpdateAdminOrder)
			admin.DELETE("/orders/:id", handlers.DeleteAdminOrder)
			admin.GET("/orders/export", handlers.ExportOrders)
			admin.GET("/orders/statistics", handlers.GetOrderStatistics)
			admin.POST("/orders/bulk-mark-paid", handlers.BulkMarkOrdersPaid)
			admin.POST("/orders/bulk-cancel", handlers.BulkCancelOrders)
			admin.POST("/orders/batch-delete", handlers.BatchDeleteOrders)

			// 套餐管理
			admin.GET("/packages", handlers.GetAdminPackages)
			admin.POST("/packages", handlers.CreatePackage)
			admin.PUT("/packages/:id", handlers.UpdatePackage)
			admin.DELETE("/packages/:id", handlers.DeletePackage)

			// 节点管理
			admin.GET("/nodes", handlers.GetAdminNodes)
			admin.GET("/nodes/stats", handlers.GetNodeStats)
			admin.POST("/nodes", handlers.CreateNode)
			admin.POST("/nodes/import-links", handlers.ImportNodeLinks)
			admin.PUT("/nodes/:id", handlers.UpdateNode)
			admin.DELETE("/nodes/:id", handlers.DeleteNode)
			admin.POST("/nodes/:id/test", handlers.TestNode)
			admin.POST("/nodes/batch-test", handlers.BatchTestNodes)
			admin.POST("/nodes/import-from-file", handlers.ImportFromFile)

			// 优惠券管理（保留兼容性路由，但主要使用 /coupons/admin）

			// 工单管理
			admin.PUT("/tickets/:id/status", handlers.UpdateTicketStatus)

			// 设备统计
			admin.GET("/devices/stats", handlers.GetDeviceStats)

			// 配置管理

			// 统计（管理员专用）
			admin.GET("/statistics", handlers.GetStatistics)
			admin.GET("/statistics/user-trend", handlers.GetUserTrend)
			admin.GET("/statistics/revenue-trend", handlers.GetRevenueTrend)

			// 系统设置
			admin.GET("/settings", handlers.GetAdminSettings)
			admin.PUT("/settings/general", handlers.UpdateGeneralSettings)
			admin.PUT("/settings/registration", handlers.UpdateRegistrationSettings)
			admin.PUT("/settings/notification", handlers.UpdateNotificationSettings)
			admin.PUT("/settings/announcement", handlers.UpdateAnnouncementSettings)
			admin.PUT("/settings/security", handlers.UpdateSecuritySettings)
			admin.PUT("/settings/theme", handlers.UpdateThemeSettings)
			admin.PUT("/settings/invite", handlers.UpdateInviteSettings)
			admin.PUT("/settings/admin-notification", handlers.UpdateAdminNotificationSystemSettings)
			admin.POST("/settings/admin-notification/test/email", handlers.TestAdminEmailNotification)
			admin.POST("/settings/admin-notification/test/telegram", handlers.TestAdminTelegramNotification)
			admin.POST("/settings/admin-notification/test/bark", handlers.TestAdminBarkNotification)
			admin.PUT("/settings/node_health", handlers.UpdateNodeHealthSettings)

			// 管理员个人资料
			admin.GET("/profile", handlers.GetAdminProfile)
			admin.PUT("/profile", handlers.UpdateAdminProfile)
			admin.POST("/change-password", handlers.ChangePassword)
			admin.GET("/login-history", handlers.GetLoginHistory)
			admin.GET("/security-settings", handlers.GetSecuritySettings)
			admin.PUT("/security-settings", handlers.UpdateAdminSecuritySettings)
			admin.GET("/notification-settings", handlers.GetNotificationSettings)
			admin.PUT("/notification-settings", handlers.UpdateAdminNotificationSettings)

			// 订阅管理
			admin.GET("/subscriptions", handlers.GetAdminSubscriptions)
			admin.PUT("/subscriptions/:id", handlers.UpdateSubscription)
			admin.POST("/subscriptions/:id/reset", handlers.ResetSubscription)
			admin.POST("/subscriptions/:id/extend", handlers.ExtendSubscription)
			admin.GET("/subscriptions/:id/devices", handlers.GetSubscriptionDevices)
			admin.POST("/subscriptions/user/:id/reset-all", handlers.ResetUserSubscription)
			admin.POST("/subscriptions/user/:id/send-email", handlers.SendSubscriptionEmail)
			admin.DELETE("/subscriptions/user/:id/delete-all", handlers.ClearUserDevices)
			admin.DELETE("/devices/:id", handlers.RemoveDevice)
			admin.GET("/subscriptions/export", handlers.ExportSubscriptions)
			admin.POST("/subscriptions/batch-clear-devices", handlers.BatchClearDevices)

			// 配置更新相关
			admin.GET("/config-update/status", handlers.GetConfigUpdateStatus)
			admin.GET("/config-update/config", handlers.GetConfigUpdateConfig)
			admin.PUT("/config-update/config", handlers.UpdateConfigUpdateConfig)
			admin.POST("/config-update/start", handlers.StartConfigUpdate)
			admin.POST("/config-update/stop", handlers.StopConfigUpdate)
			admin.POST("/config-update/test", handlers.TestConfigUpdate)
			admin.GET("/config-update/files", handlers.GetConfigUpdateFiles)
			admin.GET("/config-update/logs", handlers.GetConfigUpdateLogs)
			admin.POST("/config-update/logs/clear", handlers.ClearConfigUpdateLogs)

			// 邀请管理
			admin.GET("/invites", handlers.GetAdminInvites)
			admin.GET("/invite-relations", handlers.GetAdminInviteRelations)
			admin.GET("/invite-statistics", handlers.GetAdminInviteStatistics)

			// 用户等级管理
			admin.GET("/user-levels", handlers.GetAdminUserLevels)
			admin.POST("/user-levels", handlers.CreateUserLevel)
			admin.PUT("/user-levels/:id", handlers.UpdateUserLevel)

			// 邮件队列管理
			admin.GET("/email-queue", handlers.GetAdminEmailQueue)
			admin.GET("/email-queue/statistics", handlers.GetEmailQueueStatistics)
			admin.GET("/email-queue/:id", handlers.GetEmailQueueDetail)
			admin.DELETE("/email-queue/:id", handlers.DeleteEmailFromQueue)
			admin.POST("/email-queue/:id/retry", handlers.RetryEmailFromQueue)
			admin.POST("/email-queue/clear", handlers.ClearEmailQueue)

			// 配置管理
			admin.GET("/email-config", handlers.GetAdminEmailConfig)
			admin.POST("/email-config", handlers.UpdateEmailConfig)

			// 文件上传
			admin.POST("/upload", handlers.UploadFile)

			// 配置更新
			admin.POST("/config-update", handlers.UpdateSubscriptionConfig)

			// 系统监控
			admin.GET("/monitoring/system", handlers.GetSystemInfo)
			admin.GET("/monitoring/database", handlers.GetDatabaseStats)

			// 备份管理
			admin.POST("/backup", handlers.CreateBackup)
			admin.GET("/backups", handlers.ListBackups)

			// 日志管理
			admin.GET("/logs/audit", handlers.GetAuditLogs)
			admin.GET("/logs/login-attempts", handlers.GetLoginAttempts)
			// 系统日志（兼容前端API）
			admin.GET("/system-logs", handlers.GetSystemLogs)
			admin.GET("/logs-stats", handlers.GetLogsStats)
			admin.GET("/export-logs", handlers.ExportLogs)
			admin.POST("/clear-logs", handlers.ClearLogs)
		}
	}

	// SPA 路由处理：所有未匹配的路由都返回前端 index.html
	// 这样前端路由可以正常工作
	r.NoRoute(func(c *gin.Context) {
		// 如果是 API 请求，返回 404
		path := c.Request.URL.Path
		if len(path) >= 4 && path[:4] == "/api" {
			c.JSON(404, gin.H{
				"success": false,
				"message": "API endpoint not found",
			})
			return
		}
		// 否则返回前端页面
		c.File("./frontend/dist/index.html")
	})

	return r
}
