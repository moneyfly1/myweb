package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/config_update"
	"cboard-go/internal/utils"
)

func main() {
	// 设置环境变量以指向正确的数据库
	os.Setenv("DATABASE_URL", "sqlite:///./cboard.db")

	// 初始化配置
	if _, err := config.LoadConfig(); err != nil {
		log.Fatalf("无法加载配置: %v", err)
	}

	// 初始化数据库
	if err := database.InitDatabase(); err != nil {
		log.Fatalf("无法初始化数据库: %v", err)
	}

	// 初始化日志
	utils.InitLogger("./logs")

	db := database.GetDB()
	var user models.User
	// 用户 ID 2 是 '3219904322'
	if err := db.First(&user, 2).Error; err != nil {
		log.Fatalf("找不到用户: %v", err)
	}

	var sub models.Subscription
	if err := db.Where("user_id = ?", user.ID).First(&sub).Error; err != nil {
		log.Fatalf("找不到订阅: %v", err)
	}

	fmt.Printf("用户: %s, 专线类型: %s, 专线到期: %v\n", user.Username, user.SpecialNodeSubscriptionType, user.SpecialNodeExpiresAt)

	svc := config_update.NewConfigUpdateService()

	// 生成 Base64 格式的通用订阅
	configBase64, err := svc.GenerateUniversalConfig(sub.SubscriptionURL, "127.0.0.1", "TestAgent", "base64")
	if err != nil {
		log.Fatalf("生成配置时出错: %v", err)
	}

	fmt.Printf("配置长度: %d\n", len(configBase64))

	decoded, _ := base64.StdEncoding.DecodeString(configBase64)
	fmt.Println("解密后的订阅内容:")
	fmt.Println(string(decoded))
}
