package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"cboard-go/internal/api/router"
	"cboard-go/internal/core/auth"
	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/geoip"
	"cboard-go/internal/services/scheduler"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	if cfg == nil {
		log.Fatal("配置未正确加载")
	}

	if cfg.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	if err := database.InitDatabase(); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	ensureDefaultAdmin()

	ensureDefaultEmailTemplates()

	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		log.Printf("创建上传目录失败: %v", err)
	}

	logDir := filepath.Join(cfg.UploadDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("创建日志目录失败: %v", err)
	}

	if err := utils.InitLogger(logDir); err != nil {
		log.Printf("初始化日志失败: %v", err)
	}

	geoipPath := os.Getenv("GEOIP_DB_PATH")
	if geoipPath == "" {
		geoipPath = "./GeoLite2-City.mmdb"
	}

	if _, err := os.Stat(geoipPath); os.IsNotExist(err) {
		log.Println("GeoIP 数据库文件不存在，尝试自动下载...")
		if err := downloadGeoIPDatabase(geoipPath); err != nil {
			log.Printf("自动下载 GeoIP 数据库失败: %v", err)
			log.Println("提示: 如需启用地理位置解析，请手动下载 GeoLite2-City.mmdb 文件")
			log.Println("下载地址: https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb")
		} else {
			log.Println("GeoIP 数据库自动下载成功")
		}
	}

	if err := geoip.InitGeoIP(geoipPath); err != nil {
		log.Printf("GeoIP 初始化失败（地理位置解析功能已禁用）: %v", err)
		log.Println("提示: 如需启用地理位置解析，请下载 GeoLite2-City.mmdb 文件")
		log.Println("下载地址: https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb")
	} else {
		log.Println("GeoIP 数据库已加载，地理位置解析功能已启用")
	}
	defer geoip.Close()

	if !cfg.DisableScheduleTasks {
		sched := scheduler.NewScheduler()
		sched.Start()
		log.Println("定时任务已启动")
	}

	r := router.SetupRouter()

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("服务器启动在 %s", addr)

	if err := r.Run(addr); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

func downloadGeoIPDatabase(filePath string) error {
	url := "https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb"

	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("保存文件失败: %w", err)
	}

	return nil
}

func ensureDefaultAdmin() {
	db := database.GetDB()
	if db == nil {
		log.Println("数据库未初始化，跳过管理员检查")
		return
	}

	username := "admin"
	email := "admin@example.com"

	var user models.User
	err := db.Where("username = ? OR email = ?", username, email).First(&user).Error
	if err == nil {
		log.Printf("管理员账号已存在: %s (%s)", username, email)
		return
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("查询管理员失败: %v", err)
		return
	}

	password := generateRandomPassword()
	hashed, err := auth.HashPassword(password)
	if err != nil {
		log.Printf("生成管理员密码哈希失败: %v", err)
		return
	}

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

	log.Println("========================================")
	log.Printf("管理员账号已自动创建")
	log.Printf("用户名: %s", username)
	log.Printf("邮箱: %s", email)
	log.Printf("初始密码: %s", password)
	log.Println("========================================")
	log.Println("⚠️  请立即登录并修改密码！")
	log.Println("⚠️  此密码仅显示一次，请妥善保存！")
	log.Println("========================================")
}

func generateRandomPassword() string {
	const (
		lowercase = "abcdefghijklmnopqrstuvwxyz"
		uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digits    = "0123456789"
		special   = "!@#$%^&*()_+-=[]{}|;:,.<>?"
		allChars  = lowercase + uppercase + digits + special
	)

	password := make([]byte, 16)

	password[0] = lowercase[randomInt(len(lowercase))]
	password[1] = uppercase[randomInt(len(uppercase))]
	password[2] = digits[randomInt(len(digits))]
	password[3] = special[randomInt(len(special))]

	for i := 4; i < 16; i++ {
		password[i] = allChars[randomInt(len(allChars))]
	}

	for i := len(password) - 1; i > 0; i-- {
		j := randomInt(i + 1)
		password[i], password[j] = password[j], password[i]
	}

	return string(password)
}

func randomInt(max int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return int(time.Now().UnixNano()) % max
	}
	return int(n.Int64())
}

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
