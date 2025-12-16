package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"cboard-go/internal/api/router"
	"cboard-go/internal/core/auth"
	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/scheduler"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 确保配置已设置
	if cfg == nil {
		log.Fatal("配置未正确加载")
	}

	// 设置 Gin 模式
	if cfg.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 初始化数据库
	if err := database.InitDatabase(); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	// 自动迁移
	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// 确保默认管理员存在
	ensureDefaultAdmin()

	// 初始化默认邮件模板
	ensureDefaultEmailTemplates()

	// 创建上传目录
	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		log.Printf("创建上传目录失败: %v", err)
	}

	// 创建日志目录
	logDir := filepath.Join(cfg.UploadDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("创建日志目录失败: %v", err)
	}

	// 初始化日志
	if err := utils.InitLogger(logDir); err != nil {
		log.Printf("初始化日志失败: %v", err)
	}

	// 启动定时任务（如果未禁用）
	if !cfg.DisableScheduleTasks {
		sched := scheduler.NewScheduler()
		sched.Start()
		log.Println("定时任务已启动")
	}

	// 创建路由
	r := router.SetupRouter()

	// 启动服务器
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("服务器启动在 %s", addr)

	if err := r.Run(addr); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

// ensureDefaultAdmin 创建或更新默认管理员账号
func ensureDefaultAdmin() {
	db := database.GetDB()
	if db == nil {
		log.Println("数据库未初始化，跳过管理员检查")
		return
	}

	const (
		username = "admin"
		email    = "admin@example.com"
		password = "admin123"
	)

	hashed, err := auth.HashPassword(password)
	if err != nil {
		log.Printf("生成管理员密码哈希失败: %v", err)
		return
	}

	var user models.User
	err = db.Where("username = ? OR email = ?", username, email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			user = models.User{
				Username:   username,
				Email:      email,
				Password:   hashed,
				IsAdmin:    true,
				IsVerified: true,
				IsActive:   true,
			}
			if err := db.Create(&user).Error; err != nil {
				log.Printf("创建默认管理员失败: %v", err)
				return
			}
			log.Println("默认管理员已创建: admin / admin123")
		} else {
			log.Printf("查询管理员失败: %v", err)
		}
		return
	}

	updates := map[string]interface{}{
		"password":    hashed,
		"is_admin":    true,
		"is_verified": true,
		"is_active":   true,
	}
	if err := db.Model(&user).Updates(updates).Error; err != nil {
		log.Printf("更新默认管理员失败: %v", err)
		return
	}
	log.Println("默认管理员已更新: admin / admin123")
}

// ensureDefaultEmailTemplates 确保默认邮件模板存在
func ensureDefaultEmailTemplates() {
	db := database.GetDB()
	if db == nil {
		log.Println("数据库未初始化，跳过邮件模板检查")
		return
	}

	templates := []models.EmailTemplate{
		{
			Name:      "verification",
			Subject:   "邮箱验证 - {{code}}",
			Content:   `<html><body><h2>邮箱验证</h2><p>您的验证码是：<strong>{{code}}</strong></p><p>验证码有效期为 {{validity}} 分钟，请勿泄露给他人。</p></body></html>`,
			Variables: `{"code": "验证码", "email": "邮箱地址", "validity": "有效期（分钟）"}`,
			IsActive:  true,
		},
		{
			Name:      "password_reset",
			Subject:   "密码重置",
			Content:   `<html><body><h2>密码重置</h2><p>您请求重置密码，请点击以下链接：</p><p><a href="{{reset_link}}">{{reset_link}}</a></p><p>如果这不是您的操作，请忽略此邮件。</p></body></html>`,
			Variables: `{"reset_link": "重置链接", "email": "邮箱地址"}`,
			IsActive:  true,
		},
		{
			Name:      "subscription",
			Subject:   "订阅信息",
			Content:   `<html><body><h2>您的订阅信息</h2><p>订阅地址：<strong>{{subscription_url}}</strong></p><p>请妥善保管您的订阅地址，不要泄露给他人。</p></body></html>`,
			Variables: `{"subscription_url": "订阅地址", "email": "邮箱地址"}`,
			IsActive:  true,
		},
		{
			Name:      "welcome",
			Subject:   "欢迎注册",
			Content:   `<html><body><h2>欢迎注册</h2><p>感谢您注册我们的服务！</p><p>您的账户已创建成功，请尽快验证邮箱以激活账户。</p></body></html>`,
			Variables: `{"username": "用户名", "email": "邮箱地址"}`,
			IsActive:  true,
		},
	}

	for _, template := range templates {
		var existing models.EmailTemplate
		err := db.Where("name = ?", template.Name).First(&existing).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := db.Create(&template).Error; err != nil {
					log.Printf("创建邮件模板失败 %s: %v", template.Name, err)
				} else {
					log.Printf("邮件模板已创建: %s", template.Name)
				}
			}
		}
	}
}
