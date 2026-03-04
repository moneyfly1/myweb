package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	geoipDir := "."
	if len(os.Args) > 1 {
		geoipDir = os.Args[1]
	}

	fmt.Println("==========================================")
	fmt.Println("  下载 IP2Location LITE 数据库")
	fmt.Println("==========================================")
	fmt.Println()

	if err := os.MkdirAll(geoipDir, 0755); err != nil {
		fmt.Printf("❌ 创建目录失败: %v\n", err)
		os.Exit(1)
	}

	// 下载 IPv4 数据库
	ipv4File := filepath.Join(geoipDir, "IP2LOCATION-LITE-DB11.BIN")

	// 下载 IPv6 数据库
	ipv6File := filepath.Join(geoipDir, "IP2LOCATION-LITE-DB11.IPV6.BIN")

	fmt.Println("⚠️  注意：IP2Location LITE 数据库需要免费注册后下载")
	fmt.Println("请访问: https://lite.ip2location.com/database/ip-country-region-city-latitude-longitude-zipcode-timezone")
	fmt.Println()
	fmt.Println("下载步骤：")
	fmt.Println("1. 注册免费账号")
	fmt.Println("2. 下载 DB11.LITE (包含国家、地区、城市、经纬度、邮编、时区)")
	fmt.Println("3. 解压 ZIP 文件")
	fmt.Println("4. 将 .BIN 文件放到项目根目录或 data/ 目录")
	fmt.Println()
	fmt.Println("支持的文件名：")
	fmt.Println("  - IP2LOCATION-LITE-DB11.BIN (IPv4)")
	fmt.Println("  - IP2LOCATION-LITE-DB11.IPV6.BIN (IPv6)")
	fmt.Println()
	fmt.Println("或者使用其他免费数据库：")
	fmt.Println("  - GeoLite2-City.mmdb (MaxMind)")
	fmt.Println("  - db-ip.com 的免费数据库")
	fmt.Println()

	// 检查是否已存在
	if _, err := os.Stat(ipv4File); err == nil {
		fmt.Printf("✅ IPv4 数据库已存在: %s\n", ipv4File)
	} else {
		fmt.Printf("❌ IPv4 数据库不存在: %s\n", ipv4File)
	}

	if _, err := os.Stat(ipv6File); err == nil {
		fmt.Printf("✅ IPv6 数据库已存在: %s\n", ipv6File)
	} else {
		fmt.Printf("❌ IPv6 数据库不存在: %s\n", ipv6File)
	}

	fmt.Println()
	fmt.Println("提示：由于 IP2Location 需要注册下载，建议使用以下替代方案：")
	fmt.Println()
	fmt.Println("方案1: 使用 GeoLite2 (已支持)")
	fmt.Println("  go run scripts/download_geoip.go")
	fmt.Println()
	fmt.Println("方案2: 手动下载 IP2Location LITE")
	fmt.Println("  访问: https://lite.ip2location.com")
	fmt.Println()
}
