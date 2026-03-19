package main

import (
	"cboard-go/internal/services/geoip"
	"fmt"
)

func main() {
	if err := geoip.InitGeoIP(""); err != nil {
		fmt.Printf("❌ 初始化失败: %v\n", err)
		return
	}
	defer geoip.Close()

	testIPs := []string{
		"61.242.235.58",                          // 中国 IPv4
		"182.37.160.182",                         // 中国 IPv4
		"8.8.8.8",                                // 美国 Google DNS
		"240e:47c:6a0e:e8a0:c5e9:e0e5:e0e5:e0e5", // 中国 IPv6
	}

	fmt.Println("测试 IP 地理位置解析...")
	fmt.Println("")

	for _, ip := range testIPs {
		simple := geoip.GetLocationSimple(ip)
		if simple != "" {
			fmt.Printf("✅ %s -> %s\n", ip, simple)
		} else {
			fmt.Printf("❌ %s -> 解析失败\n", ip)
		}
	}

	fmt.Println("")
	fmt.Println("✅ 测试完成！")
}
