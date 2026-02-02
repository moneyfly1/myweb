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

	// 测试节点
	testNodes := []struct {
		name string
		link string
	}{
		{
			name: "TUIC 节点",
			link: "tuic://33c41229-3e5a-456f-bf62-e050d2b84d81%3A33c41229-3e5a-456f-bf62-e050d2b84d81@usbwg.icandoit.eu.org:15074?sni=usbwg.icandoit.eu.org&alpn=h3&insecure=0&allowInsecure=0&congestion_control=bbr#%E7%BE%8E%E5%9B%BD%E4%B8%93%E7%BA%BF08",
		},
		{
			name: "Anytls 节点",
			link: "anytls://33c41229-3e5a-456f-bf62-e050d2b84d81@usbwg.icandoit.eu.org:40361?security=tls&sni=usbwg.icandoit.eu.org&insecure=0&allowInsecure=0&type=tcp#%E7%BE%8E%E5%9B%BD%E4%B8%93%E7%BA%BF10",
		},
	}

	for i, test := range testNodes {
		fmt.Printf("=== 测试 %d: %s ===\n", i+1, test.name)
		
		// 解析节点
		node, err := config_update.ParseNodeLink(test.link)
		if err != nil {
			fmt.Printf("❌ 解析失败: %v\n\n", err)
			continue
		}

		fmt.Printf("✅ 解析成功: %s\n", node.Name)
		fmt.Printf("   类型: %s\n", node.Type)
		fmt.Printf("   服务器: %s:%d\n", node.Server, node.Port)
		if node.UUID != "" {
			fmt.Printf("   UUID: %s\n", node.UUID)
		}
		if node.Password != "" {
			fmt.Printf("   Password: %s\n", node.Password)
		}
		if node.TLS {
			fmt.Printf("   TLS: true\n")
		}
		fmt.Printf("   Options: %v\n", node.Options)
		fmt.Println()

		// 生成 Clash 配置
		proxies := []*config_update.ProxyNode{node}
		
		// 创建简单的配置结构
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
			
			// 添加基本字段
			if p.UUID != "" {
				proxyMap["uuid"] = p.UUID
			}
			if p.Password != "" {
				proxyMap["password"] = p.Password
			}
			if p.TLS {
				proxyMap["tls"] = true
			}
			if p.UDP {
				proxyMap["udp"] = true
			}
			
			// 添加所有选项
			for k, v := range p.Options {
				proxyMap[k] = v
			}
			
			// TUIC 特殊处理
			if p.Type == "tuic" {
				// 字段名映射
				if cc, ok := proxyMap["congestion_control"]; ok {
					proxyMap["congestion-controller"] = cc
					delete(proxyMap, "congestion_control")
				}
				if sn, ok := proxyMap["servername"]; ok {
					proxyMap["sni"] = sn
					delete(proxyMap, "servername")
				}
				// 设置默认值
				if _, ok := proxyMap["disable-sni"]; !ok {
					proxyMap["disable-sni"] = false
				}
				if _, ok := proxyMap["reduce-rtt"]; !ok {
					proxyMap["reduce-rtt"] = false
				}
				if _, ok := proxyMap["request-timeout"]; !ok {
					proxyMap["request-timeout"] = 15000
				}
				if _, ok := proxyMap["udp-relay-mode"]; !ok {
					proxyMap["udp-relay-mode"] = "native"
				}
			}
			
			// Anytls 特殊处理
			if p.Type == "anytls" {
				if sn, ok := proxyMap["servername"]; ok {
					proxyMap["sni"] = sn
					delete(proxyMap, "servername")
				}
				proxyMap["udp"] = false // Anytls 默认 udp 为 false
			}
			
			config.Proxies = append(config.Proxies, proxyMap)
		}
		
		yamlData, _ := yaml.Marshal(config)
		fmt.Println("生成的 Clash 配置:")
		fmt.Println(string(yamlData))
		fmt.Println()
	}
}
