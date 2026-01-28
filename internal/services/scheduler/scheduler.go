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

type Scheduler struct {
	db           *gorm.DB
	emailService *email.EmailService
	running      bool
	stopChan     chan bool
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		db:           database.GetDB(),
		emailService: email.NewEmailService(),
		stopChan:     make(chan bool),
	}
}

func (s *Scheduler) Start() {
	if s.running {
		return
	}

	s.running = true
	log.Println("定时任务调度器已启动")

	go s.processEmailQueue()
	go s.checkExpiringSubscriptions()
	go s.cleanupExpiredData()
	go s.checkNodeHealth()
	go s.autoUpdateNodes()
}

func (s *Scheduler) Stop() {
	if !s.running {
		return
	}

	s.running = false
	close(s.stopChan)
	log.Println("定时任务调度器已停止")
}

func (s *Scheduler) processEmailQueue() {
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
			emailService := email.NewEmailService()
			if err := emailService.ProcessEmailQueue(); err != nil {
				utils.LogErrorMsg("处理邮件队列失败: %v", err)
			}
		}
	}
}

func (s *Scheduler) checkExpiringSubscriptions() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

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

func (s *Scheduler) checkExpiringSubscriptionsNow() {
	now := utils.GetBeijingTime()

	sevenDaysLater := now.Add(7 * 24 * time.Hour)
	s.sendExpirationReminders(now, sevenDaysLater, 7, false)

	threeDaysLater := now.Add(3 * 24 * time.Hour)
	s.sendExpirationReminders(now, threeDaysLater, 3, false)

	oneDayLater := now.Add(1 * 24 * time.Hour)
	s.sendExpirationReminders(now, oneDayLater, 1, false)

	s.sendExpirationReminders(now, now, 0, true)
}

func (s *Scheduler) sendExpirationReminders(now, targetTime time.Time, remainingDays int, isExpired bool) {
	var subscriptions []models.Subscription
	query := s.db.Where("is_active = ? AND status = ?", true, "active")

	if isExpired {
		yesterday := now.Add(-24 * time.Hour)
		query = query.Where("expire_time <= ? AND expire_time > ?", now, yesterday)
	} else {
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
		if sub.UserID == 0 || sub.User.ID == 0 {
			continue
		}

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

		if notification.ShouldSendCustomerNotification("subscription_expiry") {
			if err := emailService.QueueEmail(sub.User.Email, subject, content, "expiration_reminder"); err != nil {
				utils.LogErrorMsg("发送到期提醒邮件失败: 用户 %s, 错误: %v", sub.User.Email, err)
			} else {
				utils.LogInfo("订阅到期提醒邮件已加入队列: 用户 %s, 剩余天数: %d", sub.User.Email, remainingDays)
			}
		} else {
			utils.LogInfo("订阅到期提醒邮件未发送: 用户 %s, 客户通知已禁用", sub.User.Email)
		}

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

func (s *Scheduler) cleanupExpiredData() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

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

func (s *Scheduler) cleanupExpiredDataNow() {
	now := utils.GetBeijingTime()

	sevenDaysAgo := now.Add(-7 * 24 * time.Hour)
	s.db.Where("created_at < ?", sevenDaysAgo).Delete(&models.VerificationCode{})

	thirtyDaysAgo := now.Add(-30 * 24 * time.Hour)
	s.db.Where("created_at < ?", thirtyDaysAgo).Delete(&models.LoginAttempt{})

	s.db.Where("status = ? AND sent_at < ?", "sent", thirtyDaysAgo).Delete(&models.EmailQueue{})

	s.checkUsersForDeletionWarning(now)

	s.checkUsersForDeletion(now)

	log.Println("过期数据清理完成")
}

func (s *Scheduler) checkUsersForDeletionWarning(now time.Time) {
	thirtyDaysAgo := now.Add(-30 * 24 * time.Hour)
	sevenDaysAgo := now.Add(-7 * 24 * time.Hour)

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
		var currentUser models.User
		if err := s.db.First(&currentUser, user.ID).Error; err != nil {
			continue
		}

		var activeSubscriptionCount int64
		s.db.Model(&models.Subscription{}).
			Where("user_id = ? AND is_active = ? AND status = ? AND expire_time > ?",
				currentUser.ID, true, "active", now).
			Count(&activeSubscriptionCount)
		if activeSubscriptionCount > 0 {
			continue
		}

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

func (s *Scheduler) checkUsersForDeletion(now time.Time) {
	thirtyDaysAgo := now.Add(-30 * 24 * time.Hour)
	sevenDaysAgo := now.Add(-7 * 24 * time.Hour)

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
		var user models.User
		if err := s.db.Where("email = ?", warningEmail.ToEmail).First(&user).Error; err != nil {
			continue
		}

		var activeSubscriptionCount int64
		s.db.Model(&models.Subscription{}).
			Where("user_id = ? AND is_active = ? AND status = ? AND expire_time > ?",
				user.ID, true, "active", now).
			Count(&activeSubscriptionCount)
		if activeSubscriptionCount > 0 {
			utils.LogInfo("用户 %s (%s) 已有有效订阅，跳过删除", user.Username, user.Email)
			continue
		}

		shouldDelete := false
		if !user.LastLogin.Valid {
			shouldDelete = true
		} else if user.LastLogin.Time.Before(thirtyDaysAgo) {
			if warningEmail.CreatedAt.After(user.LastLogin.Time) {
				shouldDelete = true
			}
		}

		if !shouldDelete {
			utils.LogInfo("用户 %s (%s) 在警告后已登录，跳过删除", user.Username, user.Email)
			continue
		}

		deletionDate := now.Format("2006-01-02 15:04:05")
		reason := "30天未登录且无有效套餐，警告后7天内未登录"
		dataRetentionPeriod := "30天"
		content := templateBuilder.GetAccountDeletionTemplate(user.Username, deletionDate, reason, dataRetentionPeriod)
		subject := "账号删除确认"
		_ = emailService.QueueEmail(user.Email, subject, content, "account_deletion")

		utils.LogInfo("用户 %s (%s) 将被删除: 30天未登录且无有效套餐，警告后7天内未登录", user.Username, user.Email)
	}
}

func (s *Scheduler) checkNodeHealth() {
	interval := 30 * time.Minute

	var config models.SystemConfig
	if err := s.db.Where("key = ? AND category = ?", "node_health_check_interval", "general").First(&config).Error; err == nil {
		if minutes, err := strconv.Atoi(config.Value); err == nil {
			interval = time.Duration(minutes) * time.Minute
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

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

func (s *Scheduler) checkNodeHealthNow() {
	log.Println("开始执行节点健康检查...")

	healthService := node_health.NewNodeHealthService()

	var config models.SystemConfig
	if err := s.db.Where("key = ? AND category = ?", "node_max_latency", "general").First(&config).Error; err == nil {
		if maxLatency, err := strconv.Atoi(config.Value); err == nil {
			healthService.SetMaxLatency(maxLatency)
		}
	}

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

func (s *Scheduler) autoUpdateNodes() {
	checkInterval := 1 * time.Hour
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

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

func (s *Scheduler) checkAndRunNodeUpdate() {
	configService := config_update.NewConfigUpdateService()
	config, err := configService.GetConfig()
	if err != nil {
		utils.LogErrorMsg("获取节点更新配置失败: %v", err)
		return
	}

	enableSchedule := false
	if val, ok := config["enable_schedule"]; ok {
		if strVal, ok := val.(string); ok {
			enableSchedule = strVal == "true" || strVal == "1"
		} else if boolVal, ok := val.(bool); ok {
			enableSchedule = boolVal
		}
	}

	if !enableSchedule {
		return
	}

	if !enableSchedule {
		return
	}

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

	lastUpdateTime, shouldUpdate := s.shouldRunNodeUpdate(intervalSeconds)
	if !shouldUpdate {
		return
	}

	utils.LogInfo("开始执行自动节点更新任务（上次更新: %s）", lastUpdateTime)
	if err := configService.RunUpdateTask(); err != nil {
		utils.LogErrorMsg("自动节点更新失败: %v", err)
	} else {
		utils.LogInfo("自动节点更新任务执行成功")
	}
}

func (s *Scheduler) shouldRunNodeUpdate(intervalSeconds int) (string, bool) {
	var config models.SystemConfig
	err := s.db.Where("key = ?", "config_update_last_update").First(&config).Error

	if err != nil {
		return "从未更新", true
	}

	lastUpdateTime, err := time.Parse("2006-01-02T15:04:05", config.Value)
	if err != nil {
		return config.Value, true
	}

	lastUpdateTime = utils.ToBeijingTime(lastUpdateTime)
	now := utils.GetBeijingTime()

	elapsed := now.Sub(lastUpdateTime)
	interval := time.Duration(intervalSeconds) * time.Second

	if elapsed >= interval {
		return lastUpdateTime.Format("2006-01-02 15:04:05"), true
	}

	return lastUpdateTime.Format("2006-01-02 15:04:05"), false
}
