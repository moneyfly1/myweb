package scheduler

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/backup_service"
	"cboard-go/internal/services/config_update"
	"cboard-go/internal/services/email"
	"cboard-go/internal/services/git"
	"cboard-go/internal/services/node_health"
	"cboard-go/internal/services/notification"
	"cboard-go/internal/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	if err := utils.CreateSchedulerLog("scheduler", "started", "定时任务调度器已启动", map[string]interface{}{
		"status": "started",
	}); err != nil {
		log.Printf("failed to create scheduler log: %v", err)
	}

	go s.processEmailQueue()
	go s.checkExpiringSubscriptions()
	go s.cleanupExpiredData()
	go s.checkNodeHealth()
	go s.autoUpdateNodes()
	go s.autoBackup()
}

func (s *Scheduler) Stop() {
	if !s.running {
		return
	}

	s.running = false
	close(s.stopChan)
	log.Println("定时任务调度器已停止")
	if err := utils.CreateSchedulerLog("scheduler", "stopped", "定时任务调度器已停止", map[string]interface{}{
		"status": "stopped",
	}); err != nil {
		log.Printf("failed to create scheduler log: %v", err)
	}
}

func (s *Scheduler) processEmailQueue() {
	emailService := email.NewEmailService() // 每次重新创建，确保使用最新配置
	if err := emailService.ProcessEmailQueue(); err != nil {
		utils.LogErrorMsg("处理邮件队列失败: %v", err)
		if logErr := utils.CreateSchedulerLog("email_queue", "error", fmt.Sprintf("处理邮件队列失败: %v", err), map[string]interface{}{
			"error": err.Error(),
		}); logErr != nil {
			log.Printf("failed to create scheduler log: %v", logErr)
		}
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
				if logErr := utils.CreateSchedulerLog("email_queue", "error", fmt.Sprintf("处理邮件队列失败: %v", err), map[string]interface{}{
					"error": err.Error(),
				}); logErr != nil {
					log.Printf("failed to create scheduler log: %v", logErr)
				}
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
		if logErr := utils.CreateSchedulerLog("expiring_subscriptions", "error", fmt.Sprintf("查询到期订阅失败: %v", err), map[string]interface{}{
			"error": err.Error(),
		}); logErr != nil {
			log.Printf("failed to create scheduler log: %v", logErr)
		}
		return
	}

	count := len(subscriptions)
	statusText := func() string {
		if isExpired {
			return "已过期"
		}
		return fmt.Sprintf("%d天后到期", remainingDays)
	}()
	utils.LogInfo("发现 %d 个%s的订阅", count, statusText)
	if count > 0 {
		if err := utils.CreateSchedulerLog("expiring_subscriptions", "info", fmt.Sprintf("发现 %d 个%s的订阅", count, statusText), map[string]interface{}{
			"count":          count,
			"remaining_days": remainingDays,
			"is_expired":     isExpired,
		}); err != nil {
			log.Printf("failed to create scheduler log: %v", err)
		}
	}

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
			expireDate = utils.FormatBeijingTime(sub.ExpireTime)
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
					expireTime = utils.FormatBeijingTime(sub.ExpireTime)
				}
				_ = notificationService.SendAdminNotification("subscription_expired", map[string]interface{}{
					"username":     sub.User.Username,
					"email":        sub.User.Email,
					"package_name": packageName,
					"expire_time":  expireTime,
					"expired_time": utils.FormatBeijingTime(utils.GetBeijingTime()),
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
		Where("LOWER(email) NOT IN (SELECT DISTINCT LOWER(to_email) FROM email_queue WHERE email_type = ? AND created_at > ?)", "account_deletion_warning", sevenDaysAgo).
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
			lastLogin = utils.FormatBeijingTime(currentUser.LastLogin.Time)
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
		if err := s.db.Where("LOWER(email) = ?", strings.ToLower(strings.TrimSpace(warningEmail.ToEmail))).First(&user).Error; err != nil {
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

		deletionDate := utils.FormatBeijingTime(now)
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
		if logErr := utils.CreateSchedulerLog("node_health_check", "error", fmt.Sprintf("节点健康检查失败: %v", err), map[string]interface{}{
			"error": err.Error(),
		}); logErr != nil {
			log.Printf("failed to create scheduler log: %v", logErr)
		}
	} else {
		utils.LogInfo("节点健康检查完成")
		if err := utils.CreateSchedulerLog("node_health_check", "success", "节点健康检查完成", nil); err != nil {
			log.Printf("failed to create scheduler log: %v", err)
		}
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
		return utils.FormatBeijingTime(lastUpdateTime), true
	}

	return utils.FormatBeijingTime(lastUpdateTime), false
}

func (s *Scheduler) autoBackup() {
	// 初始检查
	s.checkAndRunAutoBackup()

	// 每分钟检查一次配置变化
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.checkAndRunAutoBackup()
		}
	}
}

func (s *Scheduler) checkAndRunAutoBackup() {
	// 检查是否启用自动备份
	var config models.SystemConfig
	if err := s.db.Where("key = ? AND category = ?", "backup_auto_enabled", "backup").First(&config).Error; err != nil {
		return // 未配置或未启用
	}

	if config.Value != "true" {
		return // 未启用自动备份
	}

	// 获取备份间隔
	interval := 24 // 默认24小时
	if err := s.db.Where("key = ? AND category = ?", "backup_auto_interval", "backup").First(&config).Error; err == nil {
		if hours, parseErr := strconv.Atoi(config.Value); parseErr == nil && hours > 0 {
			interval = hours
		}
	}

	// 检查是否需要执行备份
	shouldBackup := s.shouldRunAutoBackup(interval)
	if !shouldBackup {
		return
	}

	// 执行备份
	utils.LogInfo("开始执行自动备份任务")
	if err := utils.CreateSchedulerLog("auto_backup", "started", "开始执行自动备份任务", nil); err != nil {
		log.Printf("failed to create scheduler log: %v", err)
	}
	if err := s.runAutoBackup(); err != nil {
		utils.LogErrorMsg("自动备份失败: %v", err)
		if logErr := utils.CreateSchedulerLog("auto_backup", "error", fmt.Sprintf("自动备份失败: %v", err), map[string]interface{}{
			"error": err.Error(),
		}); logErr != nil {
			log.Printf("failed to create scheduler log: %v", logErr)
		}
	} else {
		utils.LogInfo("自动备份任务执行成功")
		if err := utils.CreateSchedulerLog("auto_backup", "success", "自动备份任务执行成功", nil); err != nil {
			log.Printf("failed to create scheduler log: %v", err)
		}
		// 更新最后备份时间
		now := utils.GetBeijingTime()
		lastBackupTime := now.Format("2006-01-02T15:04:05")
		s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}, {Name: "category"}},
			DoUpdates: clause.Assignments(map[string]interface{}{"value": lastBackupTime}),
		}).Create(&models.SystemConfig{
			Key:      "backup_auto_last_time",
			Category: "backup",
			Value:    lastBackupTime,
		})
	}
}

func (s *Scheduler) shouldRunAutoBackup(intervalHours int) bool {
	var config models.SystemConfig
	err := s.db.Where("key = ? AND category = ?", "backup_auto_last_time", "backup").First(&config).Error

	if err != nil {
		return true // 从未备份过，需要备份
	}

	lastBackupTime, err := time.Parse("2006-01-02T15:04:05", config.Value)
	if err != nil {
		return true // 时间格式错误，需要备份
	}

	lastBackupTime = utils.ToBeijingTime(lastBackupTime)
	now := utils.GetBeijingTime()

	elapsed := now.Sub(lastBackupTime)
	interval := time.Duration(intervalHours) * time.Hour

	return elapsed >= interval
}

func (s *Scheduler) runAutoBackup() error {
	cfg := config.AppConfig

	// WAL checkpoint: 将 WAL 文件内容刷入主数据库文件
	if strings.Contains(cfg.DatabaseURL, "sqlite") {
		s.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	}

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取工作目录失败: %w", err)
	}

	backupDir := filepath.Join(wd, cfg.UploadDir, "backups")
	backupDir = filepath.Clean(backupDir)

	if !utils.IsWithinBaseDir(wd, backupDir) {
		return fmt.Errorf("无效的备份路径")
	}

	if err := os.MkdirAll(backupDir, 0750); err != nil {
		return fmt.Errorf("创建备份目录失败: %w", err)
	}

	// 使用固定文件名（覆盖模式）
	backupFileName := "backup_auto.zip"
	backupPath, ok := utils.JoinWithinBaseDir(backupDir, backupFileName)
	if !ok {
		return fmt.Errorf("无效的备份路径")
	}

	// 删除旧文件
	if _, err := os.Stat(backupPath); err == nil {
		if err := os.Remove(backupPath); err != nil {
			return fmt.Errorf("删除旧备份文件失败: %w", err)
		}
	}

	zipFile, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("创建备份文件失败: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 备份数据库文件
	dbPath, ok := utils.JoinWithinBaseDir(wd, "cboard.db")
	if ok {
		if _, err := os.Stat(dbPath); err == nil {
			dbFile, err := os.Open(dbPath)
			if err == nil {
				defer dbFile.Close()

				writer, err := zipWriter.Create("cboard.db")
				if err == nil {
					if _, copyErr := io.Copy(writer, dbFile); copyErr != nil {
						log.Printf("failed to copy database file: %v", copyErr)
					}
				}
			}
		}
	}

	// 备份配置文件
	configFiles := []string{".env", "config.yaml"}
	for _, configFile := range configFiles {
		if strings.Contains(configFile, "..") || strings.Contains(configFile, "/") ||
			strings.Contains(configFile, "\\") || strings.Contains(configFile, "~") {
			continue
		}

		configPath, inBase := utils.JoinWithinBaseDir(wd, configFile)
		if inBase {
			if _, err := os.Stat(configPath); err == nil {
				file, err := os.Open(configPath)
				if err == nil {
					defer file.Close()

					writer, err := zipWriter.Create(filepath.Base(configFile))
					if err == nil {
						if _, copyErr := io.Copy(writer, file); copyErr != nil {
							log.Printf("failed to copy config file %s: %v", configFile, copyErr)
						}
					}
				}
			}
		}
	}

	remoteCfg := backup_service.LoadRemoteBackupConfig(s.db)
	if remoteCfg.Enabled && remoteCfg.Token != "" {
		_, backupFilePath, _, err := backup_service.BuildDBOnlyBackupZip(wd, backupDir, utils.GetBeijingTime())
		if err != nil {
			utils.LogErrorMsg("创建数据库备份文件用于远程上传失败: %v", err)
		} else {
			client := git.NewClient(remoteCfg.PlatformType, remoteCfg.Token, remoteCfg.Owner, remoteCfg.Repo)
			if err := client.UploadBackup(backupFilePath); err != nil {
				utils.LogErrorMsg("上传备份到 %s 失败: %v", remoteCfg.PlatformName, err)
			} else {
				utils.LogInfo("数据库备份文件已成功上传到 %s（仅数据库文件）", remoteCfg.PlatformName)
			}
			if err := os.Remove(backupFilePath); err != nil {
				log.Printf("failed to remove backup file: %v", err)
			}
		}
	}

	return nil
}
