package scheduler

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/config_update"
	"cboard-go/internal/services/email"
	"cboard-go/internal/services/node_health"
	"cboard-go/internal/services/notification"
	"cboard-go/internal/utils"

	"gorm.io/gorm"
)

// Scheduler 定时任务调度器
type Scheduler struct {
	db           *gorm.DB
	emailService *email.EmailService
	running      bool
	stopChan     chan bool
}

// NewScheduler 创建调度器
func NewScheduler() *Scheduler {
	return &Scheduler{
		db:           database.GetDB(),
		emailService: email.NewEmailService(),
		stopChan:     make(chan bool),
	}
}

// Start 启动定时任务
func (s *Scheduler) Start() {
	if s.running {
		return
	}

	s.running = true
	log.Println("定时任务调度器已启动")

	// 启动各个定时任务
	go s.processEmailQueue()
	go s.checkExpiringSubscriptions()
	go s.cleanupExpiredData()
	go s.checkNodeHealth()
	go s.autoUpdateNodes()
}

// Stop 停止定时任务
func (s *Scheduler) Stop() {
	if !s.running {
		return
	}

	s.running = false
	close(s.stopChan)
	log.Println("定时任务调度器已停止")
}

// processEmailQueue 处理邮件队列（每1分钟执行一次，提高处理频率）
func (s *Scheduler) processEmailQueue() {
	// 启动时立即执行一次
	emailService := email.NewEmailService() // 每次重新创建，确保使用最新配置
	if err := emailService.ProcessEmailQueue(); err != nil {
		utils.LogErrorMsg("处理邮件队列失败: %v", err)
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			// 每次处理时重新创建 EmailService，确保使用最新的邮件配置
			emailService := email.NewEmailService()
			if err := emailService.ProcessEmailQueue(); err != nil {
				utils.LogErrorMsg("处理邮件队列失败: %v", err)
			}
		}
	}
}

// checkExpiringSubscriptions 检查即将过期的订阅（每天执行一次）
func (s *Scheduler) checkExpiringSubscriptions() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// 立即执行一次
	s.checkExpiringSubscriptionsNow()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.checkExpiringSubscriptionsNow()
		}
	}
}

// checkExpiringSubscriptionsNow 立即检查即将过期的订阅
func (s *Scheduler) checkExpiringSubscriptionsNow() {
	now := utils.GetBeijingTime()

	// 检查7天后到期的订阅
	sevenDaysLater := now.Add(7 * 24 * time.Hour)
	s.sendExpirationReminders(now, sevenDaysLater, 7, false)

	// 检查3天后到期的订阅
	threeDaysLater := now.Add(3 * 24 * time.Hour)
	s.sendExpirationReminders(now, threeDaysLater, 3, false)

	// 检查1天后到期的订阅
	oneDayLater := now.Add(1 * 24 * time.Hour)
	s.sendExpirationReminders(now, oneDayLater, 1, false)

	// 检查已过期的订阅
	s.sendExpirationReminders(now, now, 0, true)
}

// sendExpirationReminders 发送到期提醒邮件
func (s *Scheduler) sendExpirationReminders(now, targetTime time.Time, remainingDays int, isExpired bool) {
	var subscriptions []models.Subscription
	query := s.db.Where("is_active = ? AND status = ?", true, "active")

	if isExpired {
		// 已过期的订阅（过期时间在24小时内，避免重复发送）
		yesterday := now.Add(-24 * time.Hour)
		query = query.Where("expire_time <= ? AND expire_time > ?", now, yesterday)
	} else {
		// 即将到期的订阅（在目标时间前后1小时内）
		beforeTime := targetTime.Add(-1 * time.Hour)
		afterTime := targetTime.Add(1 * time.Hour)
		query = query.Where("expire_time >= ? AND expire_time <= ?", beforeTime, afterTime)
	}

	if err := query.Preload("User").Preload("Package").Find(&subscriptions).Error; err != nil {
		utils.LogErrorMsg("查询到期订阅失败: %v", err)
		return
	}

	utils.LogInfo("发现 %d 个%s的订阅", len(subscriptions), func() string {
		if isExpired {
			return "已过期"
		}
		return fmt.Sprintf("%d天后到期", remainingDays)
	}())

	emailService := email.NewEmailService()
	templateBuilder := email.NewEmailTemplateBuilder()

	for _, sub := range subscriptions {
		// 检查用户是否存在（通过Preload加载）
		if sub.UserID == 0 || sub.User.ID == 0 {
			continue
		}

		// 获取套餐信息
		var packageName string
		if sub.PackageID != nil && sub.Package.ID != 0 {
			packageName = sub.Package.Name
		}
		if packageName == "" {
			packageName = "默认套餐"
		}

		expireDate := "未设置"
		if !sub.ExpireTime.IsZero() {
			expireDate = sub.ExpireTime.Format("2006-01-02 15:04:05")
		}

		content := templateBuilder.GetExpirationReminderTemplate(
			sub.User.Username,
			packageName,
			expireDate,
			remainingDays,
			sub.DeviceLimit,
			sub.CurrentDevices,
			isExpired,
		)
		subject := "订阅已到期"
		if !isExpired {
			subject = fmt.Sprintf("订阅即将到期（剩余%d天）", remainingDays)
		}

		// 检查是否应该发送客户通知
		if notification.ShouldSendCustomerNotification("subscription_expiry") {
			if err := emailService.QueueEmail(sub.User.Email, subject, content, "expiration_reminder"); err != nil {
				utils.LogErrorMsg("发送到期提醒邮件失败: 用户 %s, 错误: %v", sub.User.Email, err)
			} else {
				utils.LogInfo("订阅到期提醒邮件已加入队列: 用户 %s, 剩余天数: %d", sub.User.Email, remainingDays)
			}
		} else {
			utils.LogInfo("订阅到期提醒邮件未发送: 用户 %s, 客户通知已禁用", sub.User.Email)
		}

		// 如果订阅已过期，发送管理员通知
		if isExpired {
			go func(sub models.Subscription) {
				notificationService := notification.NewNotificationService()
				expireTime := "未设置"
				if !sub.ExpireTime.IsZero() {
					expireTime = sub.ExpireTime.Format("2006-01-02 15:04:05")
				}
				_ = notificationService.SendAdminNotification("subscription_expired", map[string]interface{}{
					"username":     sub.User.Username,
					"email":        sub.User.Email,
					"package_name": packageName,
					"expire_time":  expireTime,
					"expired_time": utils.GetBeijingTime().Format("2006-01-02 15:04:05"),
				})
			}(sub)
		}
	}
}

// cleanupExpiredData 清理过期数据（每天执行一次）
func (s *Scheduler) cleanupExpiredData() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// 立即执行一次
	s.cleanupExpiredDataNow()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.cleanupExpiredDataNow()
		}
	}
}

// cleanupExpiredDataNow 立即清理过期数据
func (s *Scheduler) cleanupExpiredDataNow() {
	now := utils.GetBeijingTime()

	// 清理过期的验证码（7天前）
	sevenDaysAgo := now.Add(-7 * 24 * time.Hour)
	s.db.Where("created_at < ?", sevenDaysAgo).Delete(&models.VerificationCode{})

	// 清理过期的登录尝试记录（30天前）
	thirtyDaysAgo := now.Add(-30 * 24 * time.Hour)
	s.db.Where("created_at < ?", thirtyDaysAgo).Delete(&models.LoginAttempt{})

	// 清理已发送的邮件队列记录（30天前）
	s.db.Where("status = ? AND sent_at < ?", "sent", thirtyDaysAgo).Delete(&models.EmailQueue{})

	// 检查需要发送账户删除警告的用户（30天未登录且无有效套餐）
	s.checkUsersForDeletionWarning(now)

	// 检查需要删除的用户（30天未登录且无有效套餐，且已发送警告7天）
	s.checkUsersForDeletion(now)

	log.Println("过期数据清理完成")
}

// checkUsersForDeletionWarning 检查需要发送账户删除警告的用户
// 逻辑：如果客户不在有效期内且30天没有登录，通知客户一星期之后进行删除
// 如果在此期间客户进行了登录，那么就不要删除，而是重新计算时间
func (s *Scheduler) checkUsersForDeletionWarning(now time.Time) {
	thirtyDaysAgo := now.Add(-30 * 24 * time.Hour)
	sevenDaysAgo := now.Add(-7 * 24 * time.Hour)

	// 查找30天未登录、无有效套餐、且未在7天内发送过警告的用户
	// 注意：如果用户登录了，last_login会被更新，所以不会出现在这个查询中
	var users []models.User
	if err := s.db.Where("(last_login < ? OR last_login IS NULL)", thirtyDaysAgo).
		Where("id NOT IN (SELECT DISTINCT user_id FROM subscriptions WHERE is_active = ? AND status = ? AND expire_time > ?)", true, "active", now).
		Where("id NOT IN (SELECT DISTINCT user_id FROM email_queue WHERE email_type = ? AND created_at > ?)", "account_deletion_warning", sevenDaysAgo).
		Find(&users).Error; err != nil {
		utils.LogErrorMsg("查询需要警告的用户失败: %v", err)
		return
	}

	utils.LogInfo("发现 %d 个需要发送账户删除警告的用户", len(users))

	emailService := email.NewEmailService()
	templateBuilder := email.NewEmailTemplateBuilder()

	for _, user := range users {
		// 再次检查：如果用户在查询后登录了，跳过（虽然不太可能，但为了安全）
		var currentUser models.User
		if err := s.db.First(&currentUser, user.ID).Error; err != nil {
			continue
		}

		// 如果用户有有效订阅，跳过
		var activeSubscriptionCount int64
		s.db.Model(&models.Subscription{}).
			Where("user_id = ? AND is_active = ? AND status = ? AND expire_time > ?",
				currentUser.ID, true, "active", now).
			Count(&activeSubscriptionCount)
		if activeSubscriptionCount > 0 {
			continue
		}

		// 检查最后登录时间是否仍然超过30天
		shouldWarn := false
		if !currentUser.LastLogin.Valid {
			shouldWarn = true
		} else if currentUser.LastLogin.Time.Before(thirtyDaysAgo) {
			shouldWarn = true
		}

		if !shouldWarn {
			continue // 用户已登录，跳过
		}

		lastLogin := "从未登录"
		if currentUser.LastLogin.Valid {
			lastLogin = currentUser.LastLogin.Time.Format("2006-01-02 15:04:05")
		}

		content := templateBuilder.GetAccountDeletionWarningTemplate(
			currentUser.Username,
			currentUser.Email,
			lastLogin,
			7, // 7天后删除
		)
		subject := "账号删除提醒"

		if err := emailService.QueueEmail(currentUser.Email, subject, content, "account_deletion_warning"); err != nil {
			utils.LogErrorMsg("发送账户删除警告邮件失败: 用户 %s, 错误: %v", currentUser.Email, err)
		} else {
			utils.LogInfo("已发送账户删除警告邮件给用户: %s (%s)", currentUser.Username, currentUser.Email)
		}
	}
}

// checkUsersForDeletion 检查需要删除的用户
// 逻辑：如果客户在发送警告后7天内没有登录，则发送删除确认邮件
// 如果在此期间客户进行了登录，那么就不要删除，而是重新计算时间
func (s *Scheduler) checkUsersForDeletion(now time.Time) {
	thirtyDaysAgo := now.Add(-30 * 24 * time.Hour)
	sevenDaysAgo := now.Add(-7 * 24 * time.Hour)

	// 查找7天前已发送过警告的用户
	// 然后检查这些用户是否仍然满足删除条件（30天未登录且无有效套餐）
	var warningEmails []models.EmailQueue
	if err := s.db.Where("email_type = ? AND created_at < ? AND created_at > ?",
		"account_deletion_warning", sevenDaysAgo, now.Add(-14*24*time.Hour)). // 7-14天前发送的警告
		Find(&warningEmails).Error; err != nil {
		utils.LogErrorMsg("查询警告邮件失败: %v", err)
		return
	}

	utils.LogInfo("找到 %d 封7天前发送的账户删除警告邮件", len(warningEmails))

	emailService := email.NewEmailService()
	templateBuilder := email.NewEmailTemplateBuilder()

	for _, warningEmail := range warningEmails {
		// 查找对应的用户
		var user models.User
		if err := s.db.Where("email = ?", warningEmail.ToEmail).First(&user).Error; err != nil {
			continue
		}

		// 检查用户是否仍然满足删除条件
		// 1. 检查是否有有效订阅
		var activeSubscriptionCount int64
		s.db.Model(&models.Subscription{}).
			Where("user_id = ? AND is_active = ? AND status = ? AND expire_time > ?",
				user.ID, true, "active", now).
			Count(&activeSubscriptionCount)
		if activeSubscriptionCount > 0 {
			utils.LogInfo("用户 %s (%s) 已有有效订阅，跳过删除", user.Username, user.Email)
			continue
		}

		// 2. 检查最后登录时间是否仍然超过30天
		shouldDelete := false
		if !user.LastLogin.Valid {
			shouldDelete = true
		} else if user.LastLogin.Time.Before(thirtyDaysAgo) {
			// 检查警告发送时间，确保用户在警告发送后没有登录
			if warningEmail.CreatedAt.After(user.LastLogin.Time) {
				shouldDelete = true
			}
		}

		if !shouldDelete {
			utils.LogInfo("用户 %s (%s) 在警告后已登录，跳过删除", user.Username, user.Email)
			continue
		}

		// 发送删除确认邮件
		deletionDate := now.Format("2006-01-02 15:04:05")
		reason := "30天未登录且无有效套餐，警告后7天内未登录"
		dataRetentionPeriod := "30天"
		content := templateBuilder.GetAccountDeletionTemplate(user.Username, deletionDate, reason, dataRetentionPeriod)
		subject := "账号删除确认"
		_ = emailService.QueueEmail(user.Email, subject, content, "account_deletion")

		utils.LogInfo("用户 %s (%s) 将被删除: 30天未登录且无有效套餐，警告后7天内未登录", user.Username, user.Email)
		// 注意：实际删除操作应该在管理员确认后执行，这里只记录日志和发送确认邮件
	}
}

// checkNodeHealth 检查节点健康状态（每30分钟执行一次）
func (s *Scheduler) checkNodeHealth() {
	// 默认30分钟检查一次，可以通过配置调整
	interval := 30 * time.Minute

	// 从配置中获取检查间隔
	var config models.SystemConfig
	if err := s.db.Where("key = ? AND category = ?", "node_health_check_interval", "general").First(&config).Error; err == nil {
		if minutes, err := strconv.Atoi(config.Value); err == nil {
			interval = time.Duration(minutes) * time.Minute
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 立即执行一次
	s.checkNodeHealthNow()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.checkNodeHealthNow()
		}
	}
}

// checkNodeHealthNow 立即执行节点健康检查
func (s *Scheduler) checkNodeHealthNow() {
	log.Println("开始执行节点健康检查...")

	healthService := node_health.NewNodeHealthService()

	// 从配置中获取最大允许延迟
	var config models.SystemConfig
	if err := s.db.Where("key = ? AND category = ?", "node_max_latency", "general").First(&config).Error; err == nil {
		if maxLatency, err := strconv.Atoi(config.Value); err == nil {
			healthService.SetMaxLatency(maxLatency)
		}
	}

	// 从配置中获取测试超时时间
	if err := s.db.Where("key = ? AND category = ?", "node_test_timeout", "general").First(&config).Error; err == nil {
		if timeout, err := strconv.Atoi(config.Value); err == nil {
			healthService.SetTestTimeout(time.Duration(timeout) * time.Second)
		}
	}

	if err := healthService.CheckAllNodes(); err != nil {
		utils.LogErrorMsg("节点健康检查失败: %v", err)
	} else {
		utils.LogInfo("节点健康检查完成")
	}
}

// autoUpdateNodes 自动更新节点（根据配置的间隔执行）
func (s *Scheduler) autoUpdateNodes() {
	// 默认1小时检查一次配置
	checkInterval := 1 * time.Hour
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	// 启动时立即检查一次
	s.checkAndRunNodeUpdate()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.checkAndRunNodeUpdate()
		}
	}
}

// checkAndRunNodeUpdate 检查配置并执行节点更新
func (s *Scheduler) checkAndRunNodeUpdate() {
	// 获取配置更新服务的配置
	configService := config_update.NewConfigUpdateService()
	config, err := configService.GetConfig()
	if err != nil {
		utils.LogErrorMsg("获取节点更新配置失败: %v", err)
		return
	}

	// 检查是否启用自动更新
	enableSchedule := false
	if val, ok := config["enable_schedule"]; ok {
		if strVal, ok := val.(string); ok {
			enableSchedule = strVal == "true" || strVal == "1"
		} else if boolVal, ok := val.(bool); ok {
			enableSchedule = boolVal
		}
	}

	if !enableSchedule {
		// 未启用自动更新，不执行
		return
	}

	if !enableSchedule {
		// 未启用自动更新，不执行
		return
	}

	// 获取更新间隔（秒）
	// 优先使用 update_interval，如果没有则使用 schedule_interval
	intervalSeconds := 3600 // 默认1小时
	if val, ok := config["update_interval"]; ok {
		if strVal, ok := val.(string); ok {
			if seconds, err := strconv.Atoi(strVal); err == nil {
				intervalSeconds = seconds
			}
		} else if intVal, ok := val.(int); ok {
			intervalSeconds = intVal
		} else if floatVal, ok := val.(float64); ok {
			intervalSeconds = int(floatVal)
		}
	} else if val, ok := config["schedule_interval"]; ok {
		// 兼容旧的字段名
		if strVal, ok := val.(string); ok {
			if seconds, err := strconv.Atoi(strVal); err == nil {
				intervalSeconds = seconds
			}
		} else if intVal, ok := val.(int); ok {
			intervalSeconds = intVal
		} else if floatVal, ok := val.(float64); ok {
			intervalSeconds = int(floatVal)
		}
	}

	// 检查是否到了更新时间
	lastUpdateTime, shouldUpdate := s.shouldRunNodeUpdate(intervalSeconds)
	if !shouldUpdate {
		return
	}

	// 执行节点更新
	utils.LogInfo("开始执行自动节点更新任务（上次更新: %s）", lastUpdateTime)
	if err := configService.RunUpdateTask(); err != nil {
		utils.LogErrorMsg("自动节点更新失败: %v", err)
	} else {
		utils.LogInfo("自动节点更新任务执行成功")
	}
}

// shouldRunNodeUpdate 检查是否应该执行节点更新
func (s *Scheduler) shouldRunNodeUpdate(intervalSeconds int) (string, bool) {
	var config models.SystemConfig
	err := s.db.Where("key = ?", "config_update_last_update").First(&config).Error

	if err != nil {
		// 从未更新过，应该执行
		return "从未更新", true
	}

	// 解析最后更新时间
	lastUpdateTime, err := time.Parse("2006-01-02T15:04:05", config.Value)
	if err != nil {
		// 时间格式错误，应该执行
		return config.Value, true
	}

	// 转换为北京时间
	lastUpdateTime = utils.ToBeijingTime(lastUpdateTime)
	now := utils.GetBeijingTime()

	// 计算时间差
	elapsed := now.Sub(lastUpdateTime)
	interval := time.Duration(intervalSeconds) * time.Second

	if elapsed >= interval {
		return lastUpdateTime.Format("2006-01-02 15:04:05"), true
	}

	return lastUpdateTime.Format("2006-01-02 15:04:05"), false
}
