package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config 应用配置
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
	AliyunAccessKeyID          string
	AliyunAccessKeySecret      string
	AliyunSMSSignName          string
	AliyunSMSTemplateCode      string
	DeviceUpgradePricePerMonth float64 // 设备升级价格（每月）
}

var AppConfig *Config

// LoadConfig 加载配置
func LoadConfig() (*Config, error) {
	viper.SetConfigType("env")
	viper.SetConfigName(".env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("..")
	viper.AddConfigPath("../..")

	// 自动读取环境变量
	viper.AutomaticEnv()

	// 设置默认值
	setDefaults()

	// 读取配置文件（如果存在）
	if err := viper.ReadInConfig(); err != nil {
		// 配置文件不存在时使用环境变量和默认值
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	config := &Config{
		ProjectName: getString("PROJECT_NAME", "CBoard Modern"),
		Version:     getString("VERSION", "1.0.0"),
		APIv1Str:    getString("API_V1_STR", "/api/v1"),
		CorsOrigins: getStringSlice("BACKEND_CORS_ORIGINS", []string{}),
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
		AccessTokenExpireMinutes:   getInt("JWT_EXPIRE_HOURS", 24) * 60,
		RefreshTokenExpireDays:     getInt("REFRESH_TOKEN_EXPIRE_DAYS", 7),
		SMTPTLS:                    getString("SMTP_ENCRYPTION", "tls") == "tls" || getString("SMTP_ENCRYPTION", "tls") == "ssl",
		SMTPPort:                   getInt("SMTP_PORT", 587),
		SMTPHost:                   getString("SMTP_HOST", "smtp.qq.com"),
		SMTPUser:                   getString("SMTP_USERNAME", "your-email@qq.com"),
		SMTPPassword:               getString("SMTP_PASSWORD", "your-smtp-password"),
		EmailsFromEmail:            getString("SMTP_FROM_EMAIL", "your-email@qq.com"),
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
		AliyunAccessKeyID:          getString("ALIYUN_ACCESS_KEY_ID", ""),
		AliyunAccessKeySecret:      getString("ALIYUN_ACCESS_KEY_SECRET", ""),
		AliyunSMSSignName:          getString("ALIYUN_SMS_SIGN_NAME", ""),
		AliyunSMSTemplateCode:      getString("ALIYUN_SMS_TEMPLATE_CODE", ""),
		DeviceUpgradePricePerMonth: getFloat64("DEVICE_UPGRADE_PRICE_PER_MONTH", 10.0),
	}

	// 验证配置
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

	// 处理逗号分隔的字符串
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
	key := viper.GetString("SECRET_KEY")
	if key == "" || key == "your-secret-key-here" || len(key) < 32 {
		// 自动生成密钥
		b := make([]byte, 32)
		rand.Read(b)
		generatedKey := base64.URLEncoding.EncodeToString(b)
		fmt.Printf("警告: SECRET_KEY未设置或太弱，已自动生成: %s...\n", generatedKey[:20])
		return generatedKey
	}
	return key
}

func validateConfig(config *Config) error {
	// 验证 CORS 源
	for _, origin := range config.CorsOrigins {
		if origin == "*" || origin == "null" {
			return fmt.Errorf("CORS源不能使用通配符 '*' 或 'null'，必须明确指定域名")
		}
	}

	// 生产环境密码检查
	if os.Getenv("ENV") == "production" {
		if config.MySQLPassword == "" || config.MySQLPassword == "cboard_password_2024" {
			return fmt.Errorf("生产环境必须设置强密码！(MYSQL_PASSWORD)")
		}
		if config.PostgresPass == "" || config.PostgresPass == "password" {
			return fmt.Errorf("生产环境必须设置强密码！(POSTGRES_PASSWORD)")
		}
	}

	return nil
}
