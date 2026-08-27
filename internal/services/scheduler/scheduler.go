package scheduler

import (
	"archive/zip"
	"fmt"
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
	"cboard-go/internal/services/repo_sync"
	"cboard-go/internal/services/selfhost"
	"cboard-go/internal/services/software_sync"
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
	go s.checkSelfHostNodes()
	go s.autoUpdateNodes()
	go s.autoBackup()
	go s.syncRepoFiles()
	go s.syncSoftwareLibrary()
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

	// 一次查询未来 7 天内（含当天）到期的订阅，按剩余天数分组提醒。
	// 注意：查询窗口必须覆盖 +1/+3/+7 天，否则对应分组永远为空。
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, utils.BeijingTZ)
	windowEnd := dayStart.Add(8 * 24 * time.Hour)

	var subscriptions []models.Subscription
	if err := s.db.Where("is_active = ? AND status = ? AND expire_time >= ? AND expire_time < ?",
		true, "active", dayStart, windowEnd).
		Preload("User").Preload("Package").Find(&subscriptions).Error; err != nil {
		utils.LogErrorMsg("查询未来24小时到期订阅失败: %v", err)
		return
	}

	// 按到期时间精确分组：0天（今日到期）、1天、3天、7天
	grouped := groupExpiringSubscriptions(subscriptions, dayStart)

	// 已过期处理：昨天到期但未标记的订阅
	var expiredSubs []models.Subscription
	if err := s.db.Where("is_active = ? AND status = ? AND expire_time < ? AND expire_time >= ?",
		true, "active", dayStart, dayStart.Add(-24*time.Hour)).
		Preload("User").Preload("Package").Find(&expiredSubs).Error; err != nil {
		utils.LogErrorMsg("查询已过期订阅失败: %v", err)
	}

	if len(grouped[7]) > 0 {
		s.sendExpirationReminders(grouped[7], 7, false)
	}
	if len(grouped[3]) > 0 {
		s.sendExpirationReminders(grouped[3], 3, false)
	}
	if len(grouped[1]) > 0 {
		s.sendExpirationReminders(grouped[1], 1, false)
	}
	if len(grouped[0]) > 0 {
		s.sendExpirationReminders(grouped[0], 0, false)
	}
	if len(expiredSubs) > 0 {
		s.sendExpirationReminders(expiredSubs, 0, true)
	}
}

// groupExpiringSubscriptions 按到期日把订阅分组为：0（当天到期）、1、3、7 天后到期。
// 不匹配任何档位的订阅（如剩余 2 天）不进入任何分组。
// 纯函数，便于单元测试。
func groupExpiringSubscriptions(subscriptions []models.Subscription, dayStart time.Time) map[int][]models.Subscription {
	grouped := map[int][]models.Subscription{
		0: {}, 1: {}, 3: {}, 7: {},
	}
	for _, sub := range subscriptions {
		if sub.UserID == 0 || sub.User.ID == 0 {
			continue
		}
		expireDay := time.Date(sub.ExpireTime.Year(), sub.ExpireTime.Month(), sub.ExpireTime.Day(), 0, 0, 0, 0, utils.BeijingTZ)
		switch {
		case expireDay.Equal(dayStart):
			grouped[0] = append(grouped[0], sub)
		case expireDay.Equal(dayStart.Add(24 * time.Hour)):
			grouped[1] = append(grouped[1], sub)
		case expireDay.Equal(dayStart.Add(3 * 24 * time.Hour)):
			grouped[3] = append(grouped[3], sub)
		case expireDay.Equal(dayStart.Add(7 * 24 * time.Hour)):
			grouped[7] = append(grouped[7], sub)
		}
	}
	return grouped
}

func (s *Scheduler) sendExpirationReminders(subscriptions []models.Subscription, remainingDays int, isExpired bool) {
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

	// 读取自动清理保留天数（system_configs, category=cleanup，可在设置页配置）。
	// 各类型默认保留天数：历史短数据少留，审计/资金日志多留。
	cleanupRetention := s.loadCleanupRetention()

	// 验证码：默认保留 7 天
	s.cleanupByRetention(&models.VerificationCode{}, "verification_codes", cleanupRetention, 7, now)

	// 登录失败记录（防爆破依赖近期数据）：默认保留 30 天
	s.cleanupByRetention(&models.LoginAttempt{}, "login_attempts", cleanupRetention, 30, now)

	// 邮件队列：默认保留 30 天
	s.cleanupByRetention(&models.EmailQueue{}, "email_queue", cleanupRetention, 30, now)

	// 登录历史 / 用户活动 / 站内通知：默认保留 90 天
	s.cleanupByRetentionCol(&models.LoginHistory{}, "login_history", cleanupRetention, 90, now, "login_time")
	s.cleanupByRetention(&models.UserActivity{}, "user_activities", cleanupRetention, 90, now)
	s.cleanupByRetention(&models.Notification{}, "notifications", cleanupRetention, 90, now)

	// 各类业务日志：默认保留 180 天
	s.cleanupByRetention(&models.RegistrationLog{}, "registration_logs", cleanupRetention, 180, now)
	s.cleanupByRetention(&models.SubscriptionLog{}, "subscription_logs", cleanupRetention, 180, now)
	s.cleanupByRetention(&models.BalanceLog{}, "balance_logs", cleanupRetention, 180, now)
	s.cleanupByRetention(&models.CommissionLog{}, "commission_logs", cleanupRetention, 180, now)
	s.cleanupByRetention(&models.SubscriptionReset{}, "subscription_reset_logs", cleanupRetention, 180, now)
	s.cleanupByRetention(&models.CheckinRecord{}, "checkin_logs", cleanupRetention, 180, now)

	// 支付回调记录：默认保留 365 天（保留近期回调便于对账）
	s.cleanupByRetention(&models.PaymentCallback{}, "payment_callbacks", cleanupRetention, 365, now)

	// 审计日志：默认保留 90 天（兼容旧配置键 log_retention_days/security）
	auditDays := cleanupRetention["audit_logs"]
	if auditDays <= 0 {
		auditDays = 90
		var legacy models.SystemConfig
		if err := s.db.Where("key = ? AND category = ?", "log_retention_days", "security").First(&legacy).Error; err == nil {
			if v, err2 := strconv.Atoi(legacy.Value); err2 == nil && v > 0 {
				auditDays = v
			}
		}
	}
	auditRetention := now.Add(-time.Duration(auditDays) * 24 * time.Hour)
	// 审计日志清理保护登录/注册/签到等安全关键记录
	if result := s.db.Where("created_at < ? AND action_type NOT IN ?", auditRetention, []string{"login", "register", "checkin"}).Delete(&models.AuditLog{}); result.RowsAffected > 0 {
		log.Printf("过期审计日志清理完成，删除 %d 条（保留 %d 天）", result.RowsAffected, auditDays)
	}

	// 过期邀请码（按有效期过期，非创建时间）
	if err := s.db.Where("expires_at IS NOT NULL AND expires_at < ?", now).Delete(&models.InviteCode{}).Error; err != nil {
		log.Printf("过期邀请码清理失败: %v", err)
	}

	// 过期黑名单令牌（防重放必须保留未过期项）
	if result := s.db.Where("expires_at < ?", now).Delete(&models.TokenBlacklist{}); result.RowsAffected > 0 {
		log.Printf("过期黑名单令牌清理完成，删除 %d 条", result.RowsAffected)
	}

	log.Println("过期数据清理完成")
}

// loadCleanupRetention 读取 category=cleanup 的自动清理保留天数配置。
func (s *Scheduler) loadCleanupRetention() map[string]int {
	retention := map[string]int{}
	var configs []models.SystemConfig
	if err := s.db.Where("category = ?", "cleanup").Find(&configs).Error; err != nil {
		return retention
	}
	for _, cfg := range configs {
		key := strings.TrimSuffix(cfg.Key, "_retention_days")
		if v, err := strconv.Atoi(cfg.Value); err == nil && v > 0 {
			retention[key] = v
		}
	}
	return retention
}

// cleanupByRetention 按保留天数清理某张表（created_at < now-天数），days<=0 时用默认值。
func (s *Scheduler) cleanupByRetention(model interface{}, key string, retention map[string]int, defaultDays int, now time.Time) {
	s.cleanupByRetentionCol(model, key, retention, defaultDays, now, "created_at")
}

// cleanupByRetentionCol 同 cleanupByRetention，但可指定时间列（如 login_history 用 login_time）。
func (s *Scheduler) cleanupByRetentionCol(model interface{}, key string, retention map[string]int, defaultDays int, now time.Time, column string) {
	days := retention[key]
	if days <= 0 {
		days = defaultDays
	}
	cutoff := now.Add(-time.Duration(days) * 24 * time.Hour)
	if result := s.db.Where(column+" < ?", cutoff).Delete(model); result.Error != nil {
		log.Printf("清理 %s 失败: %v", key, result.Error)
	} else if result.RowsAffected > 0 {
		log.Printf("过期 %s 清理完成，删除 %d 条（保留 %d 天）", key, result.RowsAffected, days)
	}
}

func (s *Scheduler) checkNodeHealth() {
	interval := 30 * time.Minute

	var config models.SystemConfig
	if err := s.db.Where("key = ? AND category = ?", "node_health_check_interval", "node_health").First(&config).Error; err == nil {
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
	if err := s.db.Where("key = ? AND category = ?", "node_max_latency", "node_health").First(&config).Error; err == nil {
		if maxLatency, err := strconv.Atoi(config.Value); err == nil {
			healthService.SetMaxLatency(maxLatency)
		}
	}

	if err := s.db.Where("key = ? AND category = ?", "node_test_timeout", "node_health").First(&config).Error; err == nil {
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

// checkSelfHostNodes 自建节点守护：心跳超时标记离线，安装令牌过期标记过期。
func (s *Scheduler) checkSelfHostNodes() {
	interval := 30 * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.checkSelfHostNodesNow()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.checkSelfHostNodesNow()
		}
	}
}

func (s *Scheduler) checkSelfHostNodesNow() {
	// 心跳超时 → 离线（阈值 3 分钟，须大于脚本心跳间隔 30s）
	offline, err := selfhost.MarkOffline(s.db, 3*time.Minute)
	if err != nil {
		utils.LogErrorMsg("自建节点离线检测失败: %v", err)
	} else if offline > 0 {
		utils.LogInfo("自建节点离线检测: %d 个节点心跳超时已标记离线", offline)
		// 按"自动屏蔽超时节点"开关（system_configs category=node_health），
		// 心跳超时离线的自建节点一并屏蔽（is_active=false），
		// 保证用户订阅中不再出现失效的自建节点；心跳恢复后 Heartbeat 会自动重新启用。
		var cfg models.SystemConfig
		disableEnabled := true
		if err := s.db.Where("key = ? AND category = ?", "auto_disable_timeout", "node_health").First(&cfg).Error; err == nil {
			disableEnabled = cfg.Value != "false" && cfg.Value != "0"
		}
		if disableEnabled {
			disabled := s.db.Model(&models.CustomNode{}).
				Where("self_hosted = ? AND status = ? AND is_active = ?", true, selfhost.StatusOffline, true).
				Update("is_active", false).RowsAffected
			if disabled > 0 {
				utils.LogInfo("自建节点自动屏蔽: %d 个离线节点已禁用", disabled)
			}
		}
	}

	// 安装令牌过期且未回传 → 过期
	expired, err := selfhost.ExpirePending(s.db)
	if err != nil {
		utils.LogErrorMsg("自建节点安装令牌过期检测失败: %v", err)
	} else if expired > 0 {
		utils.LogInfo("自建节点安装令牌过期检测: %d 个节点已标记过期", expired)
	}

	// 流量配额检查：启用配额且已用流量 >= 配额的节点自动屏蔽（is_active=false）
	s.checkTrafficLimitNow()
}

// checkTrafficLimitNow 检查自建节点的流量配额，超限自动禁用并记录日志。
// 支持两种配额：节点级（traffic_limit_bytes）与分配级（user_custom_nodes.traffic_limit_bytes，客户独享场景）。
func (s *Scheduler) checkTrafficLimitNow() {
	now := utils.GetBeijingTime()

	// 1. 节点级配额
	var nodes []models.CustomNode
	if err := s.db.Where("self_hosted = ? AND traffic_limit_enabled = ? AND traffic_limit_bytes > ? AND is_active = ?",
		true, true, 0, true).Find(&nodes).Error; err != nil {
		utils.LogErrorMsg("流量配额检查查询失败: %v", err)
		return
	}
	for _, n := range nodes {
		used := n.TrafficUp + n.TrafficDown
		if n.TrafficLimitResetAt != nil && now.After(*n.TrafficLimitResetAt) {
			s.db.Model(&n).Updates(map[string]interface{}{
				"traffic_up":             0,
				"traffic_down":           0,
				"traffic_limit_reset_at": nil,
			})
			continue
		}
		if used >= n.TrafficLimitBytes {
			s.db.Model(&n).Update("is_active", false)
			utils.LogInfo("自建节点流量配额超限: %s 已用 %d 字节 / 配额 %d 字节，已自动屏蔽", n.Name, used, n.TrafficLimitBytes)
		}
	}

	// 2. 分配级配额（客户独享节点）：任一分配超限 → 屏蔽该节点（独享场景下节点仅一个客户）
	var userNodes []models.UserCustomNode
	if err := s.db.Where("traffic_limit_enabled = ? AND traffic_limit_bytes > ?", true, 0).Find(&userNodes).Error; err == nil && len(userNodes) > 0 {
		var assignedNodeIDs []uint
		for _, un := range userNodes {
			assignedNodeIDs = append(assignedNodeIDs, un.CustomNodeID)
		}
		var assignedNodes []models.CustomNode
		s.db.Where("id IN ? AND self_hosted = ? AND is_active = ?", assignedNodeIDs, true, true).Find(&assignedNodes)
		nodeMap := make(map[uint]*models.CustomNode, len(assignedNodes))
		for i := range assignedNodes {
			nodeMap[assignedNodes[i].ID] = &assignedNodes[i]
		}
		for _, un := range userNodes {
			n, ok := nodeMap[un.CustomNodeID]
			if !ok {
				continue
			}
			used := n.TrafficUp + n.TrafficDown
			if used >= un.TrafficLimitBytes {
				s.db.Model(n).Update("is_active", false)
				utils.LogInfo("自建节点分配配额超限: 节点 %s 客户配额 %d 字节 / 已用 %d 字节，已自动屏蔽", n.Name, un.TrafficLimitBytes, used)
			}
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

	// 修复时区问题：存储的时间字符串是北京时间，需要在北京时区中解析
	beijingLoc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return config.Value, true
	}

	lastUpdateTime, err := time.ParseInLocation("2006-01-02T15:04:05", config.Value, beijingLoc)
	if err != nil {
		return config.Value, true
	}

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

// syncRepoFiles GitHub 仓库文件定时同步：每分钟检查配置，按间隔执行
func (s *Scheduler) syncRepoFiles() {
	// 初始检查
	repo_sync.NewService().Tick()

	// 每分钟检查一次配置变化，按配置的间隔（默认10分钟）执行同步
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			repo_sync.NewService().Tick()
		}
	}
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

	// 备份前在临时副本上清理日志类数据并压缩（可选；失败回退为原始文件，不阻断备份）
	var dbSourcePath string
	var rawDBSize, cleanedSize int64
	if info, statErr := os.Stat(filepath.Join(wd, "cboard.db")); statErr == nil {
		rawDBSize = info.Size()
	}
	cleanCfg := backup_service.LoadBackupCleanConfig(s.db)
	if strings.Contains(cfg.DatabaseURL, "sqlite") && cleanCfg.Enabled {
		tmpPath, tmpSize, prepErr := backup_service.PrepareBackupDB(wd, cleanCfg.RetentionDays)
		if prepErr != nil {
			utils.LogWarn("自动备份: 数据库清理失败，回退为原始文件: %v", prepErr)
		} else {
			dbSourcePath = tmpPath
			cleanedSize = tmpSize
			defer func() {
				if rmErr := os.Remove(tmpPath); rmErr != nil {
					log.Printf("failed to remove temp db: %v", rmErr)
				}
			}()
		}
	}

	// 备份数据库文件
	sourcePath := dbSourcePath
	if sourcePath == "" {
		sourcePath, ok = utils.JoinWithinBaseDir(wd, "cboard.db")
		if !ok {
			return fmt.Errorf("无效的数据库路径")
		}
	}
	if _, err := os.Stat(sourcePath); err == nil {
		if copyErr := backup_service.WriteDBEntryToZip(zipWriter, sourcePath); copyErr != nil {
			log.Printf("failed to copy database file: %v", copyErr)
		}
	}

	// 配置文件以脱敏形式写入备份（密钥值替换为掩码，防止凭据随备份外泄）
	configFiles := []string{".env", "config.yaml"}
	for _, configFile := range configFiles {
		backup_service.AddSanitizedConfigFile(zipWriter, wd, configFile)
	}

	remoteCfg := backup_service.LoadRemoteBackupConfig(s.db)
	if remoteCfg.CanPush() {
		_, backupFilePath, _, err := backup_service.BuildDBOnlyBackupZip(wd, backupDir, utils.GetBeijingTime(), dbSourcePath)
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

	if cleanedSize > 0 {
		utils.LogInfo("自动备份完成: 数据库清理后 %d 字节（原始 %d 字节）", cleanedSize, rawDBSize)
	}

	return nil
}

func (s *Scheduler) syncSoftwareLibrary() {
	// 启动时检查一次（若已到期）
	s.maybeRunSoftwareSync()

	// 每 10 分钟检查一次是否到期（间隔默认 12 小时，可在云盘配置中调整）
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.maybeRunSoftwareSync()
		}
	}
}

func (s *Scheduler) maybeRunSoftwareSync() {
	if software_sync.Due() {
		if !software_sync.IsRunning() {
			log.Println("定时任务: 开始 GitHub→阿里云盘 软件库同步")
			software_sync.TriggerAsync()
		}
	}
}
