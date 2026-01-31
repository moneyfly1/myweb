package router

import (
	"cboard-go/internal/api/handlers"
	"cboard-go/internal/middleware"
	"cboard-go/internal/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.SetTrustedProxies(nil)

	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.SecurityHeadersMiddleware())
	r.Use(middleware.ErrorRecoveryMiddleware())
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.RequestIDMiddleware())

	r.Static("/static", "./frontend/dist/assets")
	r.StaticFile("/favicon.ico", "./frontend/dist/favicon.ico")
	r.StaticFile("/vite.svg", "./frontend/dist/vite.svg")

	r.GET("/health", func(c *gin.Context) {
		utils.SuccessResponse(c, http.StatusOK, "", gin.H{
			"status":  "healthy",
			"version": "1.0.0",
		})
	})

	r.Use(middleware.MaintenanceMiddleware())

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", middleware.RegisterRateLimitMiddleware(), handlers.Register)
			auth.POST("/login", middleware.LoginRateLimitMiddleware(), handlers.Login)
			auth.POST("/login-json", middleware.LoginRateLimitMiddleware(), handlers.LoginJSON)
			auth.POST("/refresh", handlers.RefreshToken)
			auth.POST("/logout", middleware.AuthMiddleware(), handlers.Logout)
			auth.POST("/verification/send", middleware.VerifyCodeRateLimitMiddleware(), handlers.SendVerificationCode)
			auth.POST("/verification/verify", handlers.VerifyCode)
			auth.POST("/forgot-password", middleware.VerifyCodeRateLimitMiddleware(), handlers.ForgotPassword)
			auth.POST("/reset-password", handlers.ResetPasswordByCode)
		}

		api.POST("/payment/notify/:type", handlers.PaymentNotify)
		api.GET("/payment/notify/:type", handlers.PaymentNotify)

		api.Use(middleware.CSRFMiddleware())

		users := api.Group("/users")
		users.Use(middleware.AuthMiddleware())
		{
			users.GET("/me", handlers.GetCurrentUser)
			users.PUT("/me", handlers.UpdateCurrentUser)
			users.GET("/dashboard-info", handlers.GetUserDashboard)
			users.POST("/change-password", handlers.ChangePassword)
			users.PUT("/preferences", handlers.UpdatePreferences)
			users.GET("/notification-settings", handlers.GetNotificationSettings)
			users.PUT("/notification-settings", handlers.UpdateUserNotificationSettings)
			users.GET("/privacy-settings", handlers.GetPrivacySettings)
			users.PUT("/privacy-settings", handlers.UpdatePrivacySettings)
			users.GET("/my-level", handlers.GetUserLevel)
			users.GET("/theme", handlers.GetUserTheme)
			users.PUT("/theme", handlers.UpdateUserTheme)
			users.GET("/login-history", handlers.GetLoginHistory)
			users.GET("/activities", handlers.GetUserActivities)
			users.GET("/subscription-resets", handlers.GetSubscriptionResets)
			users.GET("/devices", handlers.GetUserDevices)
		}

		xboardCompat := api.Group("")
		xboardCompat.Use(middleware.AuthMiddleware())
		{
			xboardCompat.GET("/user/info", handlers.GetCurrentUserXBoardCompat)

			xboardCompat.GET("/user/subscribe", handlers.GetUserSubscriptionXBoardCompat)
		}

		subscriptions := api.Group("/subscriptions")
		subscriptions.Use(middleware.AuthMiddleware())
		{
			subscriptions.GET("", handlers.GetSubscriptions)
			subscriptions.GET("/:id", handlers.GetSubscription)
			subscriptions.POST("", handlers.CreateSubscription)
			subscriptions.GET("/user-subscription", handlers.GetUserSubscription)
			subscriptions.GET("/devices", handlers.GetUserSubscriptionDevices)
			subscriptions.POST("/reset-subscription", handlers.ResetUserSubscriptionSelf)
			subscriptions.POST("/send-subscription-email", handlers.SendSubscriptionEmailSelf)
			subscriptions.POST("/convert-to-balance", handlers.ConvertSubscriptionToBalance)
			subscriptions.DELETE("/devices/:id", handlers.DeleteDevice)
		}

		subscribePublic := api.Group("")
		subscribePublic.Use(middleware.CSRFExemptMiddleware())
		{
			subscribePublic.GET("/subscribe/:url", handlers.GetSubscriptionConfig)
			subscribePublic.GET("/subscriptions/clash/:url", handlers.GetSubscriptionConfig)
			subscribePublic.GET("/subscriptions/universal/:url", handlers.GetUniversalSubscription)

			subscribePublic.GET("/client/subscribe", handlers.GetClientSubscribeXBoardCompat)
		}

		orders := api.Group("/orders")
		orders.Use(middleware.AuthMiddleware())
		{
			orders.GET("", handlers.GetOrders)
			orders.POST("", handlers.CreateOrder)
			orders.POST("/upgrade-devices", handlers.UpgradeDevices)
			orders.GET("/stats", handlers.GetOrderStats)
			orders.POST("/:orderNo/pay", handlers.PayOrder)
			orders.POST("/:orderNo/cancel", handlers.CancelOrderByNo)
			orders.GET("/:orderNo/status", handlers.GetOrderStatusByNo)
			orders.GET("/id/:id", handlers.GetOrder)
		}

		packages := api.Group("/packages")
		{
			packages.GET("", handlers.GetPackages)
			packages.GET("/:id", handlers.GetPackage)
		}

		payment := api.Group("/payment")
		payment.Use(middleware.AuthMiddleware())
		{
			payment.GET("/methods", handlers.GetPaymentMethods)
			payment.POST("", handlers.CreatePayment)
			payment.GET("/status/:id", handlers.GetPaymentStatus)
		}
		api.GET("/payment-methods/active", handlers.GetPaymentMethods)

		nodes := api.Group("/nodes")
		{
			nodes.GET("", middleware.TryAuthMiddleware(), handlers.GetNodes)
			nodes.GET("/stats", middleware.TryAuthMiddleware(), handlers.GetNodeStats)
			nodes.GET("/:id", handlers.GetNode)
		}
		nodesAuth := api.Group("/nodes")
		nodesAuth.Use(middleware.AuthMiddleware())
		{
			nodesAuth.POST("/:id/test", handlers.TestNode)
			nodesAuth.POST("/batch-test", handlers.BatchTestNodes)
			nodesAuth.POST("/import-from-clash", handlers.ImportFromClash)
		}

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
		couponsAdmin := coupons.Group("/admin")
		couponsAdmin.Use(middleware.AuthMiddleware())
		couponsAdmin.Use(middleware.AdminMiddleware())
		{
			couponsAdmin.GET("", handlers.GetAdminCoupons)
			couponsAdmin.GET("/:id", handlers.GetAdminCoupon)
			couponsAdmin.POST("", handlers.CreateCoupon)
			couponsAdmin.PUT("/:id", handlers.UpdateCoupon)
			couponsAdmin.DELETE("/:id", handlers.DeleteCoupon)
		}

		notifications := api.Group("/notifications")
		notifications.Use(middleware.AuthMiddleware())
		{
			notifications.GET("", handlers.GetNotifications)
			notifications.GET("/unread-count", handlers.GetUnreadCount)
			notifications.PUT("/:id/read", handlers.MarkAsRead)
			notifications.PUT("/read-all", handlers.MarkAllAsRead)
			notifications.DELETE("/:id", handlers.DeleteNotification)
			notifications.GET("/user-notifications", handlers.GetUserNotifications)
		}
		notificationsAdmin := api.Group("/notifications/admin")
		notificationsAdmin.Use(middleware.AuthMiddleware())
		notificationsAdmin.Use(middleware.AdminMiddleware())
		{
			notificationsAdmin.GET("/notifications", handlers.GetAdminNotifications)
			notificationsAdmin.POST("/notifications", handlers.CreateAdminNotification)
			notificationsAdmin.PUT("/notifications/:id", handlers.UpdateAdminNotification)
			notificationsAdmin.DELETE("/notifications/:id", handlers.DeleteAdminNotification)
		}

		tickets := api.Group("/tickets")
		tickets.Use(middleware.AuthMiddleware())
		{
			tickets.GET("", handlers.GetTickets)
			tickets.GET("/unread-count", handlers.GetUnreadTicketRepliesCount)
			tickets.GET("/:id", handlers.GetTicket)
			tickets.POST("", handlers.CreateTicket)
			tickets.POST("/:id/reply", handlers.ReplyTicket)
			tickets.POST("/:id/replies", handlers.ReplyTicket)
			tickets.PUT("/:id", handlers.CloseTicket)
		}
		ticketsAdmin := api.Group("/tickets/admin")
		ticketsAdmin.Use(middleware.AuthMiddleware())
		ticketsAdmin.Use(middleware.AdminMiddleware())
		{
			ticketsAdmin.GET("/all", handlers.GetAdminTickets)
			ticketsAdmin.GET("/statistics", handlers.GetAdminTicketStatistics)
			ticketsAdmin.GET("/:id", handlers.GetAdminTicket)
			ticketsAdmin.PUT("/:id", handlers.UpdateTicketStatus)
		}

		devices := api.Group("/devices")
		devices.Use(middleware.AuthMiddleware())
		{
			devices.GET("", handlers.GetDevices)
			devices.DELETE("/:id", handlers.DeleteDevice)
		}

		api.GET("/invites/validate/:code", handlers.ValidateInviteCode)

		invites := api.Group("/invites")
		invites.Use(middleware.AuthMiddleware())
		{
			invites.GET("", handlers.GetInviteCodes)
			invites.POST("", handlers.CreateInviteCode)
			invites.GET("/stats", handlers.GetInviteStats)
			invites.GET("/reward-settings", handlers.GetRewardSettings)
			invites.GET("/my-codes", handlers.GetMyInviteCodes)
			invites.PUT("/:id", handlers.UpdateInviteCode)
			invites.DELETE("/:id", handlers.DeleteInviteCode)
		}

		recharge := api.Group("/recharge")
		recharge.Use(middleware.AuthMiddleware())
		{
			recharge.GET("", handlers.GetRechargeRecords)
			recharge.GET("/status/:orderNo", handlers.GetRechargeStatusByNo)
			rechargeAdmin := recharge.Group("/admin")
			rechargeAdmin.Use(middleware.AdminMiddleware())
			{
				rechargeAdmin.GET("", handlers.GetAdminRechargeRecords)
			}
			recharge.GET("/:id", handlers.GetRechargeRecord)
			recharge.POST("", handlers.CreateRecharge)
			recharge.POST("/:id/cancel", handlers.CancelRecharge)
		}

		config := api.Group("/config")
		{
			config.GET("", handlers.GetSystemConfigs)
			config.GET("/:key", handlers.GetSystemConfig)
		}

		api.GET("/software-config", handlers.GetSoftwareConfig)

		api.GET("/mobile-config", handlers.GetMobileConfig)
		softwareConfig := api.Group("/software-config")
		softwareConfig.Use(middleware.AuthMiddleware())
		softwareConfig.Use(middleware.AdminMiddleware())
		{
			softwareConfig.PUT("", handlers.UpdateSoftwareConfig)
		}

		paymentConfig := api.Group("/payment-config")
		paymentConfig.Use(middleware.AuthMiddleware())
		paymentConfig.Use(middleware.AdminMiddleware())
		{
			paymentConfig.GET("", handlers.GetPaymentConfig)
			paymentConfig.POST("", handlers.CreatePaymentConfig)
			paymentConfig.PUT("/:id", handlers.UpdatePaymentConfig)
		}

		settings := api.Group("/settings")
		{
			settings.GET("/public-settings", handlers.GetPublicSettings)
		}

		statistics := api.Group("/statistics")
		statistics.Use(middleware.AuthMiddleware())
		statistics.Use(middleware.AdminMiddleware())
		{
			statistics.GET("", handlers.GetStatistics)
			statistics.GET("/revenue", handlers.GetRevenueChart)
			statistics.GET("/users", handlers.GetUserStatistics)
			statistics.GET("/user-trend", handlers.GetUserTrend)
			statistics.GET("/revenue-trend", handlers.GetRevenueTrend)
			statistics.GET("/regions", handlers.GetRegionStats)
		}

		admin := api.Group("/admin")
		admin.Use(middleware.AuthMiddleware())
		admin.Use(middleware.AdminMiddleware())
		{
			admin.GET("/dashboard", handlers.GetDashboard)
			admin.GET("/stats", handlers.GetDashboard)
			admin.GET("/users/recent", handlers.GetRecentUsers)
			admin.GET("/orders/recent", handlers.GetRecentOrders)
			admin.GET("/users/abnormal", handlers.GetAbnormalUsers)
			admin.POST("/users/abnormal/:id/mark-normal", handlers.MarkUserNormal)

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
			admin.POST("/users/batch-enable", handlers.BatchEnableUsers)
			admin.POST("/users/batch-disable", handlers.BatchDisableUsers)
			admin.POST("/users/batch-send-subscription-email", handlers.BatchSendSubEmail)
			admin.POST("/users/batch-expire-reminder", handlers.BatchSendExpireReminder)

			admin.GET("/orders", handlers.GetAdminOrders)
			admin.PUT("/orders/:id", handlers.UpdateAdminOrder)
			admin.DELETE("/orders/:id", handlers.DeleteAdminOrder)
			admin.GET("/orders/export", handlers.ExportOrders)
			admin.GET("/orders/statistics", handlers.GetOrderStatistics)
			admin.POST("/orders/bulk-mark-paid", handlers.BulkMarkOrdersPaid)
			admin.POST("/orders/bulk-cancel", handlers.BulkCancelOrders)
			admin.POST("/orders/batch-delete", handlers.BatchDeleteOrders)

			admin.GET("/packages", handlers.GetAdminPackages)
			admin.POST("/packages", handlers.CreatePackage)
			admin.PUT("/packages/:id", handlers.UpdatePackage)
			admin.DELETE("/packages/:id", handlers.DeletePackage)

			admin.GET("/nodes", handlers.GetAdminNodes)
			admin.GET("/nodes/stats", handlers.GetNodeStats)
			admin.POST("/nodes", handlers.CreateNode)
			admin.POST("/nodes/import-links", handlers.ImportNodeLinks)
			admin.PUT("/nodes/:id", handlers.UpdateNode)
			admin.DELETE("/nodes/:id", handlers.DeleteNode)
			admin.POST("/nodes/:id/test", handlers.TestNode)
			admin.POST("/nodes/batch-test", handlers.BatchTestNodes)
			admin.POST("/nodes/batch-delete", handlers.BatchDeleteNodes)
			admin.POST("/nodes/import-from-file", handlers.ImportFromFile)

			admin.GET("/custom-nodes", handlers.GetCustomNodes)
			admin.GET("/custom-nodes/:id/users", handlers.GetCustomNodeUsers)
			admin.POST("/custom-nodes", handlers.CreateCustomNode)
			admin.POST("/custom-nodes/import-links", handlers.ImportCustomNodeLinks)
			admin.POST("/custom-nodes/batch-delete", handlers.BatchDeleteCustomNodes)
			admin.POST("/custom-nodes/batch-assign", handlers.BatchAssignCustomNodes)
			admin.POST("/custom-nodes/batch-test", handlers.BatchTestCustomNodes)
			admin.POST("/custom-nodes/:id/test", handlers.TestCustomNode)
			admin.GET("/custom-nodes/:id/link", handlers.GetCustomNodeLink)
			admin.PUT("/custom-nodes/:id", handlers.UpdateCustomNode)
			admin.DELETE("/custom-nodes/:id", handlers.DeleteCustomNode)

			admin.GET("/users/:id/custom-nodes", handlers.GetUserCustomNodes)
			admin.POST("/users/:id/custom-nodes", handlers.AssignCustomNodeToUser)
			admin.DELETE("/users/:id/custom-nodes/:node_id", handlers.UnassignCustomNodeFromUser)

			admin.PUT("/tickets/:id/status", handlers.UpdateTicketStatus)

			admin.GET("/devices/stats", handlers.GetDeviceStats)

			admin.GET("/statistics", handlers.GetStatistics)
			admin.GET("/statistics/user-trend", handlers.GetUserTrend)
			admin.GET("/statistics/revenue-trend", handlers.GetRevenueTrend)
			admin.GET("/statistics/regions", handlers.GetRegionStats)

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
			admin.GET("/settings/geoip/status", handlers.GetGeoIPStatus)
			admin.POST("/settings/geoip/update", handlers.UpdateGeoIPDatabase)

			admin.GET("/profile", handlers.GetAdminProfile)
			admin.PUT("/profile", handlers.UpdateAdminProfile)
			admin.POST("/change-password", handlers.ChangePassword)
			admin.GET("/login-history", handlers.GetLoginHistory)
			admin.GET("/security-settings", handlers.GetSecuritySettings)
			admin.PUT("/security-settings", handlers.UpdateAdminSecuritySettings)
			admin.GET("/notification-settings", handlers.GetNotificationSettings)
			admin.PUT("/notification-settings", handlers.UpdateAdminNotificationSettings)

			admin.GET("/subscriptions", handlers.GetAdminSubscriptions)
			admin.PUT("/subscriptions/:id", handlers.UpdateSubscription)
			admin.POST("/subscriptions/:id/reset", handlers.ResetSubscription)
			admin.POST("/subscriptions/:id/extend", handlers.ExtendSubscription)
			admin.GET("/subscriptions/:id/devices", handlers.GetSubscriptionDevices)
			admin.POST("/subscriptions/user/:id/reset-all", handlers.ResetUserSubscription)
			admin.POST("/subscriptions/user/:id/send-email", handlers.SendSubscriptionEmail)
			admin.DELETE("/subscriptions/user/:id/delete-all", handlers.ClearUserDevices)
			admin.DELETE("/devices/:id", handlers.RemoveDevice)
			admin.POST("/devices/batch-delete", handlers.BatchDeleteDevices)
			admin.GET("/subscriptions/export", handlers.ExportSubscriptions)
			admin.POST("/subscriptions/batch-clear-devices", handlers.BatchClearDevices)
			admin.POST("/subscriptions/batch-delete", handlers.BatchDeleteSubscriptions)
			admin.POST("/subscriptions/batch-enable", handlers.BatchEnableSubscriptions)
			admin.POST("/subscriptions/batch-disable", handlers.BatchDisableSubscriptions)
			admin.POST("/subscriptions/batch-reset", handlers.BatchResetSubscriptions)
			admin.POST("/subscriptions/batch-send-email", handlers.BatchSendAdminSubEmail)
			admin.GET("/subscriptions/expiring", handlers.GetExpiringSubscriptions)

			admin.GET("/config-update/status", handlers.GetConfigUpdateStatus)
			admin.GET("/config-update/config", handlers.GetConfigUpdateConfig)
			admin.PUT("/config-update/config", handlers.UpdateConfigUpdateConfig)
			admin.POST("/config-update/start", handlers.StartConfigUpdate)
			admin.POST("/config-update/stop", handlers.StopConfigUpdate)
			admin.POST("/config-update/test", handlers.TestConfigUpdate)
			admin.GET("/config-update/files", handlers.GetConfigUpdateFiles)
			admin.GET("/config-update/logs", handlers.GetConfigUpdateLogs)
			admin.POST("/config-update/logs/clear", handlers.ClearConfigUpdateLogs)

			admin.GET("/invites", handlers.GetAdminInvites)
			admin.GET("/invite-relations", handlers.GetAdminInviteRelations)
			admin.GET("/invite-statistics", handlers.GetAdminInviteStatistics)

			admin.GET("/user-levels", handlers.GetAdminUserLevels)
			admin.POST("/user-levels", handlers.CreateUserLevel)
			admin.PUT("/user-levels/:id", handlers.UpdateUserLevel)

			admin.GET("/email-queue", handlers.GetAdminEmailQueue)
			admin.GET("/email-queue/statistics", handlers.GetEmailQueueStatistics)
			admin.GET("/email-queue/:id", handlers.GetEmailQueueDetail)
			admin.DELETE("/email-queue/:id", handlers.DeleteEmailFromQueue)
			admin.POST("/email-queue/:id/retry", handlers.RetryEmailFromQueue)
			admin.POST("/email-queue/clear", handlers.ClearEmailQueue)

			admin.GET("/email-config", handlers.GetAdminEmailConfig)
			admin.POST("/email-config", handlers.UpdateEmailConfig)
			admin.GET("/configs", handlers.GetSystemConfigs)
			admin.POST("/configs", handlers.CreateSystemConfig)
			admin.PUT("/configs/:key", handlers.UpdateSystemConfig)

			admin.POST("/upload", handlers.UploadFile)

			admin.POST("/config-update", handlers.UpdateSubscriptionConfig)

			admin.GET("/monitoring/system", handlers.GetSystemInfo)
			admin.GET("/monitoring/database", handlers.GetDatabaseStats)

			admin.POST("/backup", handlers.CreateBackup)
			admin.GET("/backups", handlers.ListBackups)

			admin.GET("/logs/audit", handlers.GetAuditLogs)
			admin.GET("/logs/login-attempts", handlers.GetLoginAttempts)
			admin.GET("/system-logs", handlers.GetSystemLogs)
			admin.GET("/logs-stats", handlers.GetLogsStats)
			admin.GET("/export-logs", handlers.ExportLogs)
			admin.POST("/clear-logs", handlers.ClearLogs)
		}
	}

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if len(path) >= 4 && path[:4] == "/api" {
			utils.ErrorResponse(c, http.StatusNotFound, "API endpoint not found", nil)
			return
		}
		c.File("./frontend/dist/index.html")
	})

	return r
}
