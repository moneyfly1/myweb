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
	"strings"
	"time"

	"cboard-go/internal/api/router"
	"cboard-go/internal/core/auth"
	"cboard-go/internal/core/cache"
	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/queue"
	"cboard-go/internal/services/cache_service"
	"cboard-go/internal/services/geoip"
	"cboard-go/internal/services/scheduler"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
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

	// 初始化可信代理列表（TRUSTED_PROXIES 环境变量），
	// 必须在路由/限流使用 GetRealClientIP 之前调用
	utils.InitTrustedProxies(os.Getenv("TRUSTED_PROXIES"))

	if err := database.InitDatabase(); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	// 数据库保护：如果 SQLite 数据库文件不存在（即将创建全新库），大声告警，
	// 避免"重建后数据消失"（真实原因是启动目录/路径变化导致连到了新库）
	if database.IsSQLiteFreshDB() {
		log.Println("======================================================")
		log.Println("⚠️  未找到现有数据库文件，即将创建【全新】数据库！")
		log.Println("    如您已有数据，请检查 .env 中 DATABASE_URL 指向的路径是否正确，")
		log.Println("    否则本次启动将生成一个全新的空库（旧数据不会被删除，只是不在这个路径）。")
		log.Println("======================================================")
	}

	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	middleware.ReloadLoginRateLimiter()

	ensureDefaultAdmin()

	ensureDefaultEmailTemplates()

	if err := os.MkdirAll(cfg.UploadDir, 0750); err != nil {
		log.Printf("创建上传目录失败: %v", err)
	}

	logDir := filepath.Join(cfg.UploadDir, "logs")
	if err := os.MkdirAll(logDir, 0750); err != nil {
		log.Printf("创建日志目录失败: %v", err)
	}

	if err := utils.InitLogger(logDir); err != nil {
		log.Printf("初始化日志失败: %v", err)
	}

	geoipPath := os.Getenv("GEOIP_DB_PATH")
	if geoipPath == "" {
		// 从数据库配置中读取
		db := database.GetDB()
		var conf models.SystemConfig
		if err := db.Where("key = ? AND category = ?", "geoip_database_path", "system").First(&conf).Error; err == nil && conf.Value != "" {
			geoipPath = conf.Value
		} else {
			geoipPath = "./GeoLite2-City.mmdb"
		}
	}

	// 验证 geoipPath 安全性（防止路径遍历攻击）
	cleanGeoipPath, err := safePathJoin(".", geoipPath)
	if err != nil {
		log.Printf("GeoIP 路径不安全 (%v)，使用默认路径", err)
		cleanGeoipPath = "./GeoLite2-City.mmdb"
	}

	if _, err := os.Stat(cleanGeoipPath); os.IsNotExist(err) {
		// 数据库文件缺失：异步下载，绝不阻塞服务启动（部署/升级/重启都不应被 GeoIP 下载拖慢）
		log.Println("GeoIP 数据库文件不存在，将在后台异步下载（不阻塞启动）...")
		go func(path string) {
			if err := downloadGeoIPDatabase(path); err != nil {
				log.Printf("自动下载 GeoIP 数据库失败: %v", err)
				log.Println("提示: 如需启用地理位置解析，可稍后在系统设置 → GeoIP 手动更新，或下载 GeoLite2-City.mmdb 到项目目录")
			} else {
				log.Println("GeoIP 数据库自动下载成功，正在加载...")
				geoip.InitGeoIP(path)
			}
		}(cleanGeoipPath)
	}

	if err := geoip.InitGeoIP(cleanGeoipPath); err != nil {
		log.Printf("GeoIP 初始化失败（地理位置解析功能已禁用）: %v", err)
		log.Println("提示: 如需启用地理位置解析，请在系统设置 → GeoIP 中更新数据库，或下载 GeoLite2-City.mmdb 文件")
		log.Println("下载地址: https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb")
	} else {
		log.Println("GeoIP 数据库已加载，地理位置解析功能已启用")
	}
	defer geoip.Close()

	// 初始化 Redis（可选，如果连接失败会自动禁用缓存）
	if err := cache.InitRedis(); err != nil {
		log.Printf("Redis 初始化失败（缓存功能已禁用）: %v", err)
		log.Println("提示: 如需启用缓存功能，请配置 REDIS_ADDR 环境变量")
	} else {
		log.Println("Redis 缓存已启用，GeoIP 查询将使用缓存加速")
		// 预热缓存
		cache_service.WarmupCache()

		// 初始化任务队列（基于 Redis），并启动 worker 消费任务
		redisAddr := os.Getenv("REDIS_ADDR")
		if redisAddr == "" {
			redisAddr = "localhost:6379"
		}
		if err := queue.InitQueue(redisAddr); err != nil {
			log.Printf("任务队列初始化失败: %v", err)
		} else {
			go func() {
				if err := queue.StartWorker(queue.RegisterHandlers()); err != nil {
					utils.LogErrorMsg("任务队列 worker 异常退出: %v", err)
				}
			}()
			log.Println("任务队列 worker 已启动")
		}
	}
	defer cache.Close()
	defer queue.Close()

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
	// 验证文件路径安全性
	cleanPath := filepath.Clean(filePath)
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("不安全的文件路径: %s", filePath)
	}

	url := "https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb"

	out, err := os.Create(cleanPath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer out.Close()

	// 带超时的客户端（30s），避免下载挂起
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	// 限制大小 200MB，防止异常下载撑满磁盘
	written, err := io.Copy(out, io.LimitReader(resp.Body, 200<<20))
	if err != nil {
		return fmt.Errorf("保存文件失败: %w", err)
	}
	if written >= 200<<20 {
		_ = os.Remove(cleanPath)
		return fmt.Errorf("下载内容超过 200MB 限制，已中止")
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
	// 支持环境变量/.env 覆盖默认管理员用户名/邮箱/密码（与 scripts/admin_tool 一致；
	// viper 优先取真实环境变量，其次取 .env 文件中的值）
	if v := viper.GetString("ADMIN_USERNAME"); v != "" {
		username = v
	}
	if v := viper.GetString("ADMIN_EMAIL"); v != "" {
		email = v
	}
	adminPassword := viper.GetString("ADMIN_PASSWORD")

	var user models.User
	// 先按用户名精确匹配；未命中再按邮箱匹配。
	// 不要用 username = ? OR email = ? 的单查询——用户名命中 A、邮箱命中 B 时会误改错误用户的密码。
	err := db.Where("username = ?", username).First(&user).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("查询管理员失败: %v", err)
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = db.Where("email = ?", email).First(&user).Error
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("查询管理员失败: %v", err)
		return
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 首次创建管理员：优先使用 ADMIN_PASSWORD 环境变量（部署可预测、重建不随机），
		// 未设置则生成随机密码并打印到日志（仅显示一次）。
		password := adminPassword
		if password == "" {
			password = generateRandomPassword()
		} else if len(password) < 6 {
			log.Println("警告: ADMIN_PASSWORD 长度不足 6 位，忽略并使用随机密码")
			password = generateRandomPassword()
		}
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
		if adminPassword != "" {
			log.Println("密码: [使用 ADMIN_PASSWORD 配置的密码]")
		} else {
			log.Printf("初始密码: %s", password)
			log.Println("⚠️  此密码仅显示一次，请妥善保存！")
		}
		log.Println("========================================")
		log.Println("⚠️  请立即登录并修改密码！")
		log.Println("========================================")
		return
	}

	// 管理员已存在
	if adminPassword == "" {
		log.Printf("管理员账号已存在: %s (%s)", username, email)
		return
	}
	if len(adminPassword) < 6 {
		log.Println("警告: ADMIN_PASSWORD 长度不足 6 位，已忽略本次密码重置")
		return
	}

	// 每次启动按 ADMIN_PASSWORD 重置管理员密码，保证固定密码始终可用，
	// 并确保管理员账户始终处于激活/已验证状态（即使被锁定，重启后也能登录）。
	updates := map[string]interface{}{}
	if !auth.VerifyPassword(adminPassword, user.Password) {
		hashed, err := auth.HashPassword(adminPassword)
		if err != nil {
			log.Printf("生成管理员密码哈希失败: %v", err)
			return
		}
		updates["password"] = hashed
		log.Printf("管理员 %s 的密码与 ADMIN_PASSWORD 不一致，已重置为固定密码", username)
	}
	if !user.IsAdmin {
		updates["is_admin"] = true
	}
	if !user.IsVerified {
		updates["is_verified"] = true
	}
	if !user.IsActive {
		updates["is_active"] = true
		log.Printf("管理员 %s 账户处于锁定状态，已自动解锁", username)
	}
	if len(updates) > 0 {
		if err := db.Model(&user).Updates(updates).Error; err != nil {
			log.Printf("更新管理员失败: %v", err)
			return
		}
	}
	log.Printf("✅ 管理员账号已就绪: %s (%s)，密码固定为 ADMIN_PASSWORD 配置值", username, email)
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

// 安全的路径验证，防止路径遍历攻击
func safePathJoin(baseDir, userPath string) (string, error) {
	// 清理路径
	cleaned := filepath.Clean(userPath)

	// 检查是否包含 .. 或绝对路径
	if strings.Contains(cleaned, "..") || filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("非法路径: 包含禁止的组件")
	}

	// 转换为绝对路径
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("无法解析基础目录: %w", err)
	}

	absPath := filepath.Join(absBase, cleaned)

	// 确保结果在基础目录内
	rel, err := filepath.Rel(absBase, absPath)
	if err != nil {
		return "", fmt.Errorf("路径计算失败: %w", err)
	}

	// 如果相对路径以 .. 开头，说明试图访问基础目录之外
	if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("路径超出允许范围")
	}

	return absPath, nil
}
