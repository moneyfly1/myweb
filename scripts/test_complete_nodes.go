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
	fmt.Println("=== 测试完整节点链接解析 ===")
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

	// 测试节点链接
	testLinks := []string{
		"vless://00000000-0000-4000-8000-000000000028@node112.example.com:42115?encryption=none&flow=xtls-rprx-vision&security=tls&sni=node112.example.com&fp=chrome&insecure=0&allowInsecure=0&type=tcp&headerType=none&host=node112.example.com#%E7%BE%8E%E5%9B%BD%E4%B8%93%E7%BA%BF01",
		"vless://00000000-0000-4000-8000-000000000028@node112.example.com:26823?encryption=none&security=tls&sni=node112.example.com&fp=chrome&insecure=0&allowInsecure=0&type=ws&host=node112.example.com&path=%2Fzcxgws#%E7%BE%8E%E5%9B%BD%E4%B8%93%E7%BA%BF02",
		"vmess:ew0KICAidiI6ICIyIiwNCiAgInBzIjoi56S65L6L6IqC54K5IiwNCiAgImFkZCI6ICJub2RlMTEyLmV4YW1wbGUuY29tIiwNCiAgInBvcnQiOiAiMTIxMzAiLA0KICAiaWQiOiAiMDAwMDAwMDAtMDAwMC00MDAwLTgwMDAtMDAwMDAwMDAwMDI4IiwNCiAgImFpZCI6ICIwIiwNCiAgInNjeSI6ICJhdXRvIiwNCiAgIm5ldCI6ICJ3cyIsDQogICJ0eXBlIjogIm5vbmUiLA0KICAiaG9zdCI6ICJub2RlMTEyLmV4YW1wbGUuY29tIiwNCiAgInBhdGgiOiAiL2lpZWQiLA0KICAidGxzIjogInRscyIsDQogICJzbmkiOiAibm9kZTExMi5leGFtcGxlLmNvbSIsDQogICJhbHBuIjogIiIsDQogICJmcCI6ICIiLA0KICAiaW5zZWN1cmUiOiAiMCINCn0=",
		"trojan://00000000-0000-4000-8000-000000000028@node112.example.com:14367?security=tls&sni=node112.example.com&fp=chrome&alpn=http%2F1.1&insecure=0&allowInsecure=0&type=tcp&headerType=none#%E7%BE%8E%E5%9B%BD%E4%B8%93%E7%BA%BF04",
		"hysteria2://00000000-0000-4000-8000-000000000028@node112.example.com:26386?sni=node112.example.com&alpn=h3&insecure=0&allowInsecure=0#%E7%BE%8E%E5%9B%BD%E4%B8%93%E7%BA%BF05",
		"vless://00000000-0000-4000-8000-000000000028@node112.example.com:13060?encryption=none&flow=xtls-rprx-vision&security=reality&sni=cdn-dynmedia-1.microsoft.com&fp=chrome&pbk=lf2FVJzxSafTmEvbgJdGwc9-dAR_5OGP20JxDuimbgc&sid=6ba85179e30d4fc2&type=tcp&headerType=none#%E7%BE%8E%E5%9B%BD%E4%B8%93%E7%BA%BF06",
		"vless://00000000-0000-4000-8000-000000000028@node112.example.com:23435?encryption=none&security=reality&sni=node112.example.com&fp=chrome&pbk=lf2FVJzxSafTmEvbgJdGwc9-dAR_5OGP20JxDuimbgc&sid=6ba85179e30d4fc2&type=grpc&authority=&serviceName=grpc&mode=gun#%E7%BE%8E%E5%9B%BD%E4%B8%93%E7%BA%BF07",
		"tuic://00000000-0000-4000-8000-000000000028%3A00000000-0000-4000-8000-000000000028@node112.example.com:15074?sni=node112.example.com&alpn=h3&insecure=0&allowInsecure=0&congestion_control=bbr#%E7%BE%8E%E5%9B%BD%E4%B8%93%E7%BA%BF08",
		"anytls://00000000-0000-4000-8000-000000000028@node112.example.com:40361?security=tls&sni=node112.example.com&insecure=0&allowInsecure=0&type=tcp#%E7%BE%8E%E5%9B%BD%E4%B8%93%E7%BA%BF10",
	}

	var proxies []*config_update.ProxyNode

	for i, link := range testLinks {
		fmt.Printf("=== 节点 %d ===\n", i+1)

		// 解析节点
		node, err := config_update.ParseNodeLink(link)
		if err != nil {
			fmt.Printf("❌ 解析失败: %v\n", err)
			fmt.Println()
			continue
		}

		fmt.Printf("✅ 解析成功: %s (%s)\n", node.Name, node.Type)
		fmt.Printf("   Server: %s:%d\n", node.Server, node.Port)
		if node.UUID != "" {
			fmt.Printf("   UUID: %s\n", node.UUID)
		}
		if node.Password != "" {
			fmt.Printf("   Password: %s\n", node.Password)
		}
		if node.Network != "" {
			fmt.Printf("   Network: %s\n", node.Network)
		}
		if node.TLS {
			fmt.Printf("   TLS: true\n")
		}
		fmt.Printf("   Options: %d 个字段\n", len(node.Options))
		
		// 显示重要选项
		if flow, ok := node.Options["flow"].(string); ok && flow != "" {
			fmt.Printf("   ✓ flow: %s\n", flow)
		}
		if sni, ok := node.Options["servername"].(string); ok && sni != "" {
			fmt.Printf("   ✓ servername: %s\n", sni)
		}
		if fp, ok := node.Options["client-fingerprint"].(string); ok && fp != "" {
			fmt.Printf("   ✓ client-fingerprint: %s\n", fp)
		}
		if alpn, ok := node.Options["alpn"]; ok {
			fmt.Printf("   ✓ alpn: %v\n", alpn)
		}
		if wsOpts, ok := node.Options["ws-opts"]; ok {
			fmt.Printf("   ✓ ws-opts: %v\n", wsOpts)
		}
		if grpcOpts, ok := node.Options["grpc-opts"]; ok {
			fmt.Printf("   ✓ grpc-opts: %v\n", grpcOpts)
		}
		if realityOpts, ok := node.Options["reality-opts"]; ok {
			fmt.Printf("   ✓ reality-opts: %v\n", realityOpts)
		}
		if cc, ok := node.Options["congestion-control"]; ok {
			fmt.Printf("   ✓ congestion-control: %v\n", cc)
		}
		
		fmt.Println()
		proxies = append(proxies, node)
	}

	// 生成 Clash 配置
	fmt.Println("=== 生成 Clash 配置 ===")
	fmt.Println()

	// 创建一个简单的配置
	type ClashConfig struct {
		Proxies []map[string]interface{} `yaml:"proxies"`
	}

	config := ClashConfig{
		Proxies: make([]map[string]interface{}, 0),
	}

	for _, proxy := range proxies {
		proxyMap := map[string]interface{}{
			"name":   proxy.Name,
			"type":   proxy.Type,
			"server": proxy.Server,
			"port":   proxy.Port,
		}

		// 根据类型设置字段
		switch proxy.Type {
		case "vless":
			proxyMap["uuid"] = proxy.UUID
			if proxy.TLS {
				proxyMap["tls"] = true
			}
		case "vmess":
			proxyMap["uuid"] = proxy.UUID
			proxyMap["alterId"] = 0
			proxyMap["cipher"] = "auto"
			if proxy.TLS {
				proxyMap["tls"] = true
			}
		case "trojan":
			proxyMap["password"] = proxy.Password
		case "hysteria2":
			proxyMap["password"] = proxy.Password
			proxyMap["auth"] = proxy.Password
		case "tuic":
			proxyMap["uuid"] = proxy.UUID
			proxyMap["password"] = proxy.Password
		case "anytls":
			proxyMap["uuid"] = proxy.UUID
			proxyMap["password"] = proxy.Password
		}

		if proxy.Network != "" && proxy.Network != "tcp" {
			proxyMap["network"] = proxy.Network
		}
		if proxy.UDP {
			proxyMap["udp"] = true
		}

		// 添加所有选项
		for k, v := range proxy.Options {
			proxyMap[k] = v
		}

		config.Proxies = append(config.Proxies, proxyMap)
	}

	// 输出 YAML
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		fmt.Printf("❌ 生成 YAML 失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("生成的 Clash 配置:")
	fmt.Println(string(yamlData))

	// 保存到文件
	outputPath := "uploads/config/test_complete_nodes_output.yaml"
	if err := os.WriteFile(outputPath, yamlData, 0644); err == nil {
		fmt.Printf("\n📝 配置已保存到: %s\n", outputPath)
	}

	fmt.Println("\n✅ 测试完成")
}
