package scheduler

import (
	"log"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/email"

	"gorm.io/gorm"
)

// Scheduler 定时任务调度器
type Scheduler struct {
	db          *gorm.DB
	emailService *email.EmailService
	running     bool
	stopChan    chan bool
}

// NewScheduler 创建调度器
func NewScheduler() *Scheduler {
	return &Scheduler{
		db:          database.GetDB(),
		emailService: email.NewEmailService(),
		stopChan:    make(chan bool),
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

// processEmailQueue 处理邮件队列（每5分钟执行一次）
func (s *Scheduler) processEmailQueue() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			if err := s.emailService.ProcessEmailQueue(); err != nil {
				log.Printf("处理邮件队列失败: %v", err)
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
	// 查找3天内即将过期的订阅
	threeDaysLater := time.Now().Add(3 * 24 * time.Hour)

	var subscriptions []models.Subscription
	if err := s.db.Where("expire_time <= ? AND expire_time > ? AND is_active = ? AND status = ?",
		threeDaysLater, time.Now(), true, "active").Find(&subscriptions).Error; err != nil {
		log.Printf("查询即将过期的订阅失败: %v", err)
		return
	}

	log.Printf("发现 %d 个即将过期的订阅", len(subscriptions))

	// 发送提醒邮件（这里可以扩展为发送通知）
	for _, sub := range subscriptions {
		var user models.User
		if err := s.db.First(&user, sub.UserID).Error; err != nil {
			continue
		}

		// 可以在这里发送到期提醒邮件
		log.Printf("订阅即将过期: 用户 %s, 订阅ID %d, 过期时间 %s", user.Email, sub.ID, sub.ExpireTime.Format("2006-01-02 15:04:05"))
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
	// 清理过期的验证码（7天前）
	sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour)
	s.db.Where("created_at < ?", sevenDaysAgo).Delete(&models.VerificationCode{})

	// 清理过期的登录尝试记录（30天前）
	thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour)
	s.db.Where("created_at < ?", thirtyDaysAgo).Delete(&models.LoginAttempt{})

	// 清理已发送的邮件队列记录（30天前）
	s.db.Where("status = ? AND sent_at < ?", "sent", thirtyDaysAgo).Delete(&models.EmailQueue{})

	log.Println("过期数据清理完成")
}

