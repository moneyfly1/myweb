package main

import (
	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"encoding/json"
	"fmt"
	"log"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	if cfg == nil {
		log.Fatal("配置未正确加载")
	}

	if err := database.InitDatabase(); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	db := database.GetDB()

	var configs []models.PaymentConfig
	if err := db.Where("pay_type = ? OR pay_type LIKE ?", "yipay", "yipay_%").Find(&configs).Error; err != nil {
		log.Fatalf("查询支付配置失败: %v", err)
	}

	fmt.Println("=== 易支付配置检查 ===\n")

	if len(configs) == 0 {
		fmt.Println("❌ 未找到易支付配置")
		return
	}

	for _, config := range configs {
		fmt.Printf("配置ID: %d\n", config.ID)
		fmt.Printf("支付类型: %s\n", config.PayType)
		fmt.Printf("状态: %d (1=启用, 0=禁用)\n", config.Status)
		fmt.Printf("商户ID: %s\n", config.AppID.String)

		var configData map[string]interface{}
		if config.ConfigJSON.Valid {
			if err := json.Unmarshal([]byte(config.ConfigJSON.String), &configData); err == nil {
				fmt.Printf("API地址 (api_url): %v\n", configData["api_url"])
				fmt.Printf("网关地址 (gateway_url): %v\n", configData["gateway_url"])
				fmt.Printf("签名方式 (sign_type): %v\n", configData["sign_type"])

				apiURL, _ := configData["api_url"].(string)
				if apiURL == "" {
					gatewayURL, _ := configData["gateway_url"].(string)
					if gatewayURL != "" {
						expectedURL := gatewayURL
						if gatewayURL[len(gatewayURL)-1] == '/' {
							expectedURL = gatewayURL[:len(gatewayURL)-1]
						}
						expectedURL += "/mapi.php"
						fmt.Printf("⚠️  警告: api_url为空，系统将使用: %s\n", expectedURL)
					} else {
						fmt.Printf("❌ 错误: api_url和gateway_url都为空！\n")
					}
				} else {
					if apiURL != "https://fhymw.com/mapi.php" && !contains(apiURL, "mapi.php") {
						fmt.Printf("❌ 错误: API地址不正确！应该是: https://fhymw.com/mapi.php\n")
						fmt.Printf("   当前值: %s\n", apiURL)
					} else {
						fmt.Printf("✅ API地址正确\n")
					}
				}
			}
		} else {
			fmt.Printf("❌ 错误: ConfigJSON为空\n")
		}

		fmt.Println()
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
