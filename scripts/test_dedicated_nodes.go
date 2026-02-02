package main

import (
	"encoding/json"
	"fmt"
	"os"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"

	"gopkg.in/yaml.v3"
)

func main() {
	fmt.Println("=== 测试专线节点配置生成 ===")
	fmt.Println()

	// 加载配置
	_, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("❌ 加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化数据库
	if err := database.InitDatabase(); err != nil {
		fmt.Printf("❌ 数据库初始化失败: %v\n", err)
		os.Exit(1)
	}

	db := database.GetDB()

	// 查找用户 3219904322
	var user models.User
	err = db.Where("username = ?", "3219904322").First(&user).Error
	if err != nil {
		fmt.Printf("❌ 查找用户失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("找到用户: %s (ID: %d)\n", user.Username, user.ID)
	fmt.Println()

	// 查找用户的专线节点
	var customNodes []models.CustomNode
	err = db.Table("custom_nodes").
		Joins("JOIN user_custom_nodes ON user_custom_nodes.custom_node_id = custom_nodes.id").
		Where("user_custom_nodes.user_id = ? AND custom_nodes.is_active = ?", user.ID, true).
		Find(&customNodes).Error

	if err != nil {
		fmt.Printf("❌ 查找专线节点失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("找到 %d 个专线节点\n", len(customNodes))
	fmt.Println()

	// 显示每个节点的配置
	for i, node := range customNodes {
		fmt.Printf("=== 节点 %d: %s ===\n", i+1, node.DisplayName)
		fmt.Printf("协议: %s\n", node.Protocol)
		fmt.Printf("域名: %s\n", node.Domain)
		fmt.Printf("端口: %d\n", node.Port)
		fmt.Println()

		// 解析 Config
		var nodeConfig models.NodeConfig
		if err := json.Unmarshal([]byte(node.Config), &nodeConfig); err != nil {
			fmt.Printf("⚠️  解析配置失败: %v\n", err)
			fmt.Printf("配置内容: %s\n", node.Config)
		} else {
			fmt.Println("配置详情:")
			configJSON, _ := json.MarshalIndent(nodeConfig, "  ", "  ")
			fmt.Println(string(configJSON))
		}
		fmt.Println()
	}

	// 生成 Clash 配置
	fmt.Println("=== 生成 Clash 配置 ===")
	fmt.Println()

	var subscription models.Subscription
	err = db.Where("user_id = ? AND status = ?", user.ID, "active").First(&subscription).Error
	if err != nil {
		fmt.Printf("❌ 查找订阅失败: %v\n", err)
		os.Exit(1)
	}

	// 读取 Clash 订阅
	clashURL := fmt.Sprintf("https://dy.moneyfly.top/api/v1/subscriptions/clash/%s", subscription.SubscriptionURL)
	fmt.Printf("订阅地址: %s\n", clashURL)
	fmt.Println()

	// 从数据库生成配置（模拟）
	fmt.Println("检查生成的专线节点配置...")
	fmt.Println()

	testOutputPath := "uploads/config/test_dedicated_nodes_analysis.yaml"
	analysisFile, _ := os.Create(testOutputPath)
	defer analysisFile.Close()

	analysisFile.WriteString("# 专线节点配置分析\n\n")

	for i, node := range customNodes {
		analysisFile.WriteString(fmt.Sprintf("## 节点 %d: %s\n\n", i+1, node.DisplayName))
		analysisFile.WriteString(fmt.Sprintf("协议: %s\n", node.Protocol))
		analysisFile.WriteString(fmt.Sprintf("配置: %s\n\n", node.Config))

		var nodeConfig models.NodeConfig
		if err := json.Unmarshal([]byte(node.Config), &nodeConfig); err == nil {
			analysisFile.WriteString("解析后的配置:\n")
			yamlData, _ := yaml.Marshal(nodeConfig)
			analysisFile.WriteString(string(yamlData))
			analysisFile.WriteString("\n")
		}
	}

	fmt.Printf("📝 分析结果已保存到: %s\n", testOutputPath)
	fmt.Println()
	fmt.Println("✅ 测试完成")
}
