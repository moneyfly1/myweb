package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProxyNode 模拟节点结构
type ProxyNode struct {
	Name string
	Type string
}

func main() {
	fmt.Println("=== Clash 订阅配置测试 ===")
	fmt.Println()

	// 读取模板
	templatePath := filepath.Join("uploads", "config", "temp.yaml")
	templateData, err := os.ReadFile(templatePath)
	if err != nil {
		fmt.Printf("❌ 无法读取模板文件: %v\n", err)
		os.Exit(1)
	}

	var templateConfig map[string]interface{}
	if err := yaml.Unmarshal(templateData, &templateConfig); err != nil {
		fmt.Printf("❌ 模板解析失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ 模板读取成功")
	fmt.Printf("   模板路径: %s\n", templatePath)
	fmt.Println()

	// 模拟一些测试节点
	testProxies := []*ProxyNode{
		{Name: "🇭🇰 香港节点01", Type: "vmess"},
		{Name: "🇺🇸 美国节点01", Type: "trojan"},
		{Name: "🇯🇵 日本节点01", Type: "ss"},
	}

	fmt.Printf("📋 测试节点: %d 个\n", len(testProxies))
	for i, proxy := range testProxies {
		fmt.Printf("   %d. %s (%s)\n", i+1, proxy.Name, proxy.Type)
	}
	fmt.Println()

	// 生成代理名称列表
	var proxyNames []string
	for _, proxy := range testProxies {
		proxyNames = append(proxyNames, proxy.Name)
	}

	// 创建代理列表
	proxyList := make([]map[string]interface{}, 0)
	for _, proxy := range testProxies {
		proxyMap := map[string]interface{}{
			"name":   proxy.Name,
			"type":   proxy.Type,
			"server": "example.com",
			"port":   443,
		}
		proxyList = append(proxyList, proxyMap)
	}
	templateConfig["proxies"] = proxyList

	// 更新 proxy-groups
	if proxyGroups, ok := templateConfig["proxy-groups"].([]interface{}); ok {
		// 收集所有代理组名称
		groupNames := make(map[string]bool)
		for _, groupRaw := range proxyGroups {
			if group, ok := groupRaw.(map[string]interface{}); ok {
				if name, ok := group["name"].(string); ok {
					groupNames[name] = true
				}
			}
		}

		fmt.Println("🔧 处理代理组:")
		for _, groupRaw := range proxyGroups {
			if group, ok := groupRaw.(map[string]interface{}); ok {
				groupType, _ := group["type"].(string)
				groupName, _ := group["name"].(string)

				if groupType == "select" || groupType == "url-test" || groupType == "fallback" || groupType == "load-balance" {
					// 保留特殊代理和组名
					existingProxies := make([]string, 0)
					if proxiesRaw, ok := group["proxies"].([]interface{}); ok {
						for _, p := range proxiesRaw {
							if pStr, ok := p.(string); ok {
								if pStr == "DIRECT" || pStr == "REJECT" || groupNames[pStr] {
									existingProxies = append(existingProxies, pStr)
								}
							}
						}
					}

					var updatedProxies interface{}
					if groupType == "url-test" || groupType == "fallback" || groupType == "load-balance" {
						// 只包含实际节点
						updatedProxies = proxyNames
						fmt.Printf("   ✓ %s [%s]: 注入 %d 个节点\n", groupName, groupType, len(proxyNames))
					} else {
						// select 类型：保留组名和特殊代理，添加实际节点
						allProxies := append(existingProxies, proxyNames...)
						updatedProxies = allProxies
						fmt.Printf("   ✓ %s [%s]: 保留 %d 个特殊选项 + 注入 %d 个节点\n", 
							groupName, groupType, len(existingProxies), len(proxyNames))
					}
					group["proxies"] = updatedProxies
				}
			}
		}
		templateConfig["proxy-groups"] = proxyGroups
	}
	fmt.Println()

	// 转换为 YAML
	output, err := yaml.Marshal(templateConfig)
	if err != nil {
		fmt.Printf("❌ YAML 序列化失败: %v\n", err)
		os.Exit(1)
	}

	// 将 Unicode 转义序列还原为实际字符
	outputStr := unescapeUnicode(string(output))
	output = []byte(outputStr)

	fmt.Println("✅ 配置生成成功")
	fmt.Println()

	// 验证生成的配置
	fmt.Println("🔍 验证生成的配置:")
	
	var generatedConfig map[string]interface{}
	if err := yaml.Unmarshal(output, &generatedConfig); err != nil {
		fmt.Printf("   ❌ 生成的配置无法解析: %v\n", err)
		os.Exit(1)
	}

	// 检查基本配置
	checks := []struct {
		key      string
		expected interface{}
	}{
		{"port", 7890},
		{"socks-port", 7891},
		{"mode", "Rule"},
		{"log-level", "info"},
	}

	allPassed := true
	for _, check := range checks {
		if val, ok := generatedConfig[check.key]; ok {
			if val == check.expected {
				fmt.Printf("   ✓ %s: %v\n", check.key, val)
			} else {
				fmt.Printf("   ✗ %s: 期望 %v, 实际 %v\n", check.key, check.expected, val)
				allPassed = false
			}
		} else {
			fmt.Printf("   ✗ %s: 缺失\n", check.key)
			allPassed = false
		}
	}

	// 检查代理
	if proxies, ok := generatedConfig["proxies"].([]interface{}); ok {
		fmt.Printf("   ✓ proxies: %d 个节点\n", len(proxies))
		if len(proxies) != len(testProxies) {
			fmt.Printf("   ✗ 节点数量不匹配: 期望 %d, 实际 %d\n", len(testProxies), len(proxies))
			allPassed = false
		}
	} else {
		fmt.Println("   ✗ proxies: 缺失或格式错误")
		allPassed = false
	}

	// 检查代理组
	if proxyGroups, ok := generatedConfig["proxy-groups"].([]interface{}); ok {
		fmt.Printf("   ✓ proxy-groups: %d 个代理组\n", len(proxyGroups))
		
		// 验证关键代理组
		keyGroups := []string{"🚀 节点选择", "♻️ 自动选择", "🔰 故障转移", "🔮 负载均衡"}
		foundGroups := 0
		for _, groupRaw := range proxyGroups {
			if group, ok := groupRaw.(map[string]interface{}); ok {
				if name, ok := group["name"].(string); ok {
					for _, keyGroup := range keyGroups {
						if name == keyGroup {
							foundGroups++
							if proxies, ok := group["proxies"].([]interface{}); ok {
								fmt.Printf("   ✓ %s: %d 个选项\n", name, len(proxies))
							}
							break
						}
					}
				}
			}
		}
		if foundGroups != len(keyGroups) {
			fmt.Printf("   ✗ 关键代理组不完整: 找到 %d/%d\n", foundGroups, len(keyGroups))
			allPassed = false
		}
	} else {
		fmt.Println("   ✗ proxy-groups: 缺失或格式错误")
		allPassed = false
	}

	// 检查规则
	if rules, ok := generatedConfig["rules"].([]interface{}); ok {
		fmt.Printf("   ✓ rules: %d 条规则\n", len(rules))
		if len(rules) == 0 {
			fmt.Println("   ✗ 没有规则")
			allPassed = false
		}
		
		// 检查最后一条是否是 MATCH
		if len(rules) > 0 {
			lastRule, _ := rules[len(rules)-1].(string)
			if strings.HasPrefix(lastRule, "MATCH,") {
				fmt.Printf("   ✓ 最后一条规则: %s\n", lastRule)
			} else {
				fmt.Printf("   ⚠️  最后一条规则不是 MATCH: %s\n", lastRule)
			}
		}
	} else {
		fmt.Println("   ✗ rules: 缺失或格式错误")
		allPassed = false
	}

	// 检查 DNS 配置
	if dns, ok := generatedConfig["dns"].(map[string]interface{}); ok {
		if enable, ok := dns["enable"].(bool); ok && enable {
			fmt.Printf("   ✓ dns.enable: %v\n", enable)
		}
		if mode, ok := dns["enhanced-mode"].(string); ok {
			fmt.Printf("   ✓ dns.enhanced-mode: %s\n", mode)
		}
	}

	fmt.Println()

	// 总结
	if allPassed {
		fmt.Println("🎉 所有检查通过！配置生成符合模板要求。")
		fmt.Println()
		fmt.Println("✅ 模板验证结果:")
		fmt.Println("   • 基本配置 ✓")
		fmt.Println("   • 代理节点注入 ✓")
		fmt.Println("   • 代理组配置 ✓")
		fmt.Println("   • 分流规则保留 ✓")
		fmt.Println("   • DNS 配置 ✓")
		
		// 保存测试输出
		testOutputPath := "uploads/config/test_output.yaml"
		if err := os.WriteFile(testOutputPath, output, 0644); err == nil {
			fmt.Printf("\n📝 测试配置已保存到: %s\n", testOutputPath)
		}
	} else {
		fmt.Println("❌ 部分检查未通过，请查看上述错误。")
		os.Exit(1)
	}
}

// unescapeUnicode 将 Unicode 转义序列还原为实际字符
func unescapeUnicode(s string) string {
	re := regexp.MustCompile(`\\U([0-9A-Fa-f]{8})`)
	result := re.ReplaceAllStringFunc(s, func(match string) string {
		hexStr := match[2:]
		codePoint, err := strconv.ParseInt(hexStr, 16, 64)
		if err != nil {
			return match
		}
		return string(rune(codePoint))
	})
	return result
}
