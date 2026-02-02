package main

import (
	"fmt"
	"os"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/services/config_update"

	"gopkg.in/yaml.v3"
)

func main() {
	fmt.Println("=== 测试 SOCKS 代理节点 ===")
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

	// SOCKS 节点链接
	socksLink := "socks://cXN6cnJjcmRjZDpvcndtb3l1bW9weW1l@direct.miyaip.online:8001#socks%E4%BB%A3%E7%90%86"
	
	fmt.Printf("节点链接: %s\n\n", socksLink)
	
	// 解析节点
	node, err := config_update.ParseNodeLink(socksLink)
	if err != nil {
		fmt.Printf("❌ 解析失败: %v\n\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ 解析成功:\n")
	fmt.Printf("   名称: %s\n", node.Name)
	fmt.Printf("   类型: %s\n", node.Type)
	fmt.Printf("   服务器: %s\n", node.Server)
	fmt.Printf("   端口: %d\n", node.Port)
	if node.UUID != "" {
		fmt.Printf("   Username: %s\n", node.UUID)
	}
	if node.Password != "" {
		fmt.Printf("   Password: %s\n", node.Password)
	} else {
		fmt.Printf("   Password: (空)\n")
	}
	if node.UDP {
		fmt.Printf("   UDP: true\n")
	}
	fmt.Printf("   Options: %v\n", node.Options)
	fmt.Println()

	// 生成 Clash 配置
	proxies := []*config_update.ProxyNode{node}
	
	// 创建配置结构
	type ClashConfig struct {
		Proxies []map[string]interface{} `yaml:"proxies"`
	}
	
	config := ClashConfig{Proxies: make([]map[string]interface{}, 0)}
	
	for _, p := range proxies {
		proxyMap := map[string]interface{}{
			"name":   p.Name,
			"type":   p.Type,
			"server": p.Server,
			"port":   p.Port,
		}
		
		// SOCKS 使用 username 和 password
		if p.UUID != "" {
			proxyMap["username"] = p.UUID
		}
		if p.Password != "" {
			proxyMap["password"] = p.Password
		} else {
			proxyMap["password"] = "" // 确保包含字段
		}
		if p.UDP {
			proxyMap["udp"] = true
		}
		
		// 添加所有选项
		for k, v := range p.Options {
			proxyMap[k] = v
		}
		
		config.Proxies = append(config.Proxies, proxyMap)
	}
	
	yamlData, _ := yaml.Marshal(config)
	fmt.Println("✅ 生成的 Clash 配置:")
	fmt.Println(string(yamlData))
	fmt.Println()
	
	// 验证必需字段
	fmt.Println("🔍 字段验证:")
	proxy := config.Proxies[0]
	
	checkField := func(field string, expected interface{}) {
		if val, ok := proxy[field]; ok {
			if expected != nil && val != expected {
				fmt.Printf("   ⚠️  %s: %v (期望: %v)\n", field, val, expected)
			} else {
				fmt.Printf("   ✅ %s: %v\n", field, val)
			}
		} else {
			fmt.Printf("   ❌ %s: 缺失\n", field)
		}
	}
	
	checkField("username", nil)
	checkField("password", nil)
	checkField("udp", true)
	
	fmt.Println()
	fmt.Println("🎉 测试完成")
}
