package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	ProjectName                string
	Version                    string
	APIv1Str                   string
	CorsOrigins                []string
	DatabaseURL                string
	MySQLHost                  string
	MySQLPort                  int
	MySQLUser                  string
	MySQLPassword              string
	MySQLDatabase              string
	PostgresServer             string
	PostgresUser               string
	PostgresPass               string
	PostgresDB                 string
	SecretKey                  string
	Algorithm                  string
	AccessTokenExpireMinutes   int
	RefreshTokenExpireDays     int
	SMTPTLS                    bool
	SMTPPort                   int
	SMTPHost                   string
	SMTPUser                   string
	SMTPPassword               string
	EmailsFromEmail            string
	EmailsFromName             string
	AlipayAppID                string
	AlipayPrivateKey           string
	AlipayPublicKey            string
	AlipayNotifyURL            string
	AlipayReturnURL            string
	UploadDir                  string
	MaxFileSize                int64
	SubscriptionURLPrefix      string
	DeviceLimitDefault         int
	Debug                      bool
	Host                       string
	Port                       int
	Workers                    int
	BaseURL                    string
	DisableScheduleTasks       bool
	OptimizeForLowEnd          bool
	DeviceUpgradePricePerMonth float64 // 设备升级价格（每月，兼容旧逻辑）
	DeviceUpgradePricePerYear  float64 // 设备升级参考年度价格（如 200 元/年）
	DeviceUpgradeBaseDevices   int     // 年度价格对应的设备数（如 200 元对应 5 台）
}

var AppConfig *Config

func LoadConfig() (*Config, error) {
	viper.SetConfigType("env")
	viper.SetConfigName(".env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("..")
	viper.AddConfigPath("../..")

	viper.AutomaticEnv()

	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	config := &Config{
		ProjectName: getString("PROJECT_NAME", "CBoard Modern"),
		Version:     getString("VERSION", "1.0.0"),
		APIv1Str:    getString("API_V1_STR", "/api/v1"),
		CorsOrigins: getStringSlice("BACKEND_CORS_ORIGINS", []string{
			"http://localhost:3000",
			"http://localhost:5173",
			"http://localhost:8080",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:5173",
			"http://127.0.0.1:8080",
		}),
		DatabaseURL:                getString("DATABASE_URL", "sqlite:///./cboard.db"),
		MySQLHost:                  getString("MYSQL_HOST", "localhost"),
		MySQLPort:                  getInt("MYSQL_PORT", 3306),
		MySQLUser:                  getString("MYSQL_USER", "cboard_user"),
		MySQLPassword:              getString("MYSQL_PASSWORD", ""),
		MySQLDatabase:              getString("MYSQL_DATABASE", "cboard_db"),
		PostgresServer:             getString("POSTGRES_SERVER", "localhost"),
		PostgresUser:               getString("POSTGRES_USER", "postgres"),
		PostgresPass:               getString("POSTGRES_PASSWORD", ""),
		PostgresDB:                 getString("POSTGRES_DB", "cboard"),
		SecretKey:                  getSecretKey(),
		Algorithm:                  getString("JWT_ALGORITHM", "HS256"),
		AccessTokenExpireMinutes:   getInt("JWT_EXPIRE_HOURS", 1) * 60,
		RefreshTokenExpireDays:     getInt("REFRESH_TOKEN_EXPIRE_DAYS", 30),
		SMTPTLS:                    getString("SMTP_ENCRYPTION", "tls") == "tls" || getString("SMTP_ENCRYPTION", "tls") == "ssl",
		SMTPPort:                   getInt("SMTP_PORT", 587),
		SMTPHost:                   getString("SMTP_HOST", ""),
		SMTPUser:                   getString("SMTP_USERNAME", ""),
		SMTPPassword:               getString("SMTP_PASSWORD", ""),
		EmailsFromEmail:            getString("SMTP_FROM_EMAIL", ""),
		EmailsFromName:             getString("SMTP_FROM_NAME", "CBoard Modern"),
		AlipayAppID:                getString("ALIPAY_APP_ID", "your-alipay-app-id"),
		AlipayPrivateKey:           getString("ALIPAY_PRIVATE_KEY", "your-private-key"),
		AlipayPublicKey:            getString("ALIPAY_PUBLIC_KEY", "alipay-public-key"),
		AlipayNotifyURL:            getString("ALIPAY_NOTIFY_URL", ""),
		AlipayReturnURL:            getString("ALIPAY_RETURN_URL", ""),
		UploadDir:                  getString("UPLOAD_DIR", "uploads"),
		MaxFileSize:                int64(getInt("MAX_FILE_SIZE", 10485760)),
		SubscriptionURLPrefix:      getString("SUBSCRIPTION_URL_PREFIX", ""),
		DeviceLimitDefault:         getInt("DEVICE_LIMIT_DEFAULT", 3),
		Debug:                      getBool("DEBUG", false),
		Host:                       getString("HOST", "0.0.0.0"),
		Port:                       getInt("PORT", 8000),
		Workers:                    getInt("WORKERS", 4),
		BaseURL:                    getString("BASE_URL", ""),
		DisableScheduleTasks:       getBool("DISABLE_SCHEDULE_TASKS", false),
		OptimizeForLowEnd:          getBool("OPTIMIZE_FOR_LOW_END", true),
		DeviceUpgradePricePerMonth: getFloat64("DEVICE_UPGRADE_PRICE_PER_MONTH", 10.0),
		DeviceUpgradePricePerYear:  getFloat64("DEVICE_UPGRADE_PRICE_PER_YEAR", 200.0),
		DeviceUpgradeBaseDevices:   getInt("DEVICE_UPGRADE_BASE_DEVICES", 5),
	}

	if err := validateConfig(config); err != nil {
		return nil, err
	}

	AppConfig = config
	return config, nil
}

func setDefaults() {
	viper.SetDefault("PROJECT_NAME", "CBoard Modern")
	viper.SetDefault("VERSION", "1.0.0")
	viper.SetDefault("API_V1_STR", "/api/v1")
	viper.SetDefault("DATABASE_URL", "sqlite:///./cboard.db")
	viper.SetDefault("HOST", "0.0.0.0")
	viper.SetDefault("PORT", 8000)
	viper.SetDefault("DEBUG", false)
}

func getString(key, defaultValue string) string {
	value := viper.GetString(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getInt(key string, defaultValue int) int {
	value := viper.GetInt(key)
	if value == 0 {
		return defaultValue
	}
	return value
}

func getBool(key string, defaultValue bool) bool {
	value := viper.GetBool(key)
	if !viper.IsSet(key) {
		return defaultValue
	}
	return value
}

func getFloat64(key string, defaultValue float64) float64 {
	value := viper.GetFloat64(key)
	if value == 0 {
		return defaultValue
	}
	return value
}

func getStringSlice(key string, defaultValue []string) []string {
	value := viper.GetString(key)
	if value == "" {
		return defaultValue
	}

	origins := strings.Split(value, ",")
	result := make([]string, 0, len(origins))
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin != "" && origin != "*" && origin != "null" {
			result = append(result, origin)
		}
	}

	if len(result) == 0 {
		return defaultValue
	}
	return result
}

func getSecretKey() string {
	key := strings.TrimSpace(viper.GetString("SECRET_KEY"))
	
	// 首先检查环境变量（优先级最高）
	if envKey := os.Getenv("JWT_SECRET_KEY"); envKey != "" {
		key = envKey
	}
	
	// 验证密钥强度
	if isWeakSecretKey(key) {
		env := strings.ToLower(strings.TrimSpace(os.Getenv("ENV")))
		if env == "production" || env == "prod" {
			// 生产环境必须使用强密钥
			log.Fatal("生产环境必须设置强密钥 (JWT_SECRET_KEY 或 SECRET_KEY，长度至少32字节且不是弱口令)")
		}
		
		// 开发/测试环境可以自动生成，但会警告
		log.Println("警告: SECRET_KEY 未设置或使用弱口令，建议设置强密钥")
		if key == "" {
			// 仅在完全未设置时生成临时密钥
			b := make([]byte, 32)
			if _, err := rand.Read(b); err != nil {
				panic(fmt.Errorf("生成 SECRET_KEY 失败: %w", err))
			}
			key = base64.URLEncoding.EncodeToString(b)
			log.Println("已生成临时开发密钥（注意：重启后所有JWT令牌将失效）")
		}
	}
	
	return key
}

func isWeakSecretKey(key string) bool {
	if key == "" || len(key) < 32 {
		return true
	}

	lower := strings.ToLower(strings.TrimSpace(key))
	weakPatterns := []string{
		"your-secret-key",
		"change-me",
		"changeme",
		"example",
		"default",
		"password",
		"test",
	}
	for _, pattern := range weakPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	// 同一字符重复的低熵 key（如 aaaaaa...）
	allSame := true
	for i := 1; i < len(lower); i++ {
		if lower[i] != lower[0] {
			allSame = false
			break
		}
	}
	return allSame
}

func validateConfig(config *Config) error {
	for _, origin := range config.CorsOrigins {
		if origin == "*" || origin == "null" {
			return fmt.Errorf("CORS源不能使用通配符 '*' 或 'null'，必须明确指定域名")
		}
	}

	env := strings.ToLower(strings.TrimSpace(os.Getenv("ENV")))
	if env == "production" || env == "prod" {
		rawSecret := strings.TrimSpace(viper.GetString("SECRET_KEY"))
		if isWeakSecretKey(rawSecret) {
			return fmt.Errorf("生产环境必须设置强 SECRET_KEY（至少32位随机字符串，且不能使用占位符）")
		}
		if config.MySQLPassword == "" || config.MySQLPassword == "cboard_password_2024" {
			return fmt.Errorf("生产环境必须设置强密码！(MYSQL_PASSWORD)")
		}
		if config.PostgresPass == "" || config.PostgresPass == "password" {
			return fmt.Errorf("生产环境必须设置强密码！(POSTGRES_PASSWORD)")
		}
	}

	return nil
}
