package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"cboard-go/internal/core/dl"
)

func main() {
	geoipDir := "."
	if len(os.Args) > 1 {
		geoipDir = os.Args[1]
	}

	fmt.Println("==========================================")
	fmt.Println("  下载 DB-IP 免费数据库")
	fmt.Println("==========================================")
	fmt.Println()

	if err := os.MkdirAll(geoipDir, 0755); err != nil {
		fmt.Printf("❌ 创建目录失败: %v\n", err)
		os.Exit(1)
	}

	// 获取当前年月
	now := time.Now()
	yearMonth := now.Format("2006-01")

	// DB-IP 免费数据库 URL (MMDB 格式，兼容 GeoIP2)
	dbipFile := filepath.Join(geoipDir, "dbip-city-lite.mmdb")
	dbipURL := fmt.Sprintf("https://download.db-ip.com/free/dbip-city-lite-%s.mmdb.gz", yearMonth)

	fmt.Println("正在下载 DB-IP City Lite 数据库...")
	fmt.Printf("URL: %s\n", dbipURL)
	fmt.Printf("保存路径: %s\n", dbipFile)
	fmt.Println()

	if _, err := os.Stat(dbipFile); err == nil {
		fmt.Printf("⚠️  数据库文件已存在: %s\n", dbipFile)
		if os.Getenv("CI") != "" || os.Getenv("BUILD_MODE") != "" {
			fmt.Println("构建模式：跳过下载（文件已存在）")
			os.Exit(0)
		}
		fmt.Print("是否覆盖? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("跳过下载")
			os.Exit(0)
		}
		os.Remove(dbipFile)
	}

	if _, err := dl.Download(context.Background(), dbipURL, dbipFile, dl.Options{Gzip: true, Label: "下载并解压中"}); err != nil {
		fmt.Printf("❌ 下载失败: %v\n", err)
		fmt.Println()
		fmt.Println("提示：DB-IP 免费数据库每月更新，URL 格式为:")
		fmt.Println("  https://download.db-ip.com/free/dbip-city-lite-YYYY-MM.mmdb.gz")
		fmt.Println()
		fmt.Println("如果当前月份的数据库尚未发布，请尝试上个月的:")
		lastMonth := now.AddDate(0, -1, 0).Format("2006-01")
		fmt.Printf("  https://download.db-ip.com/free/dbip-city-lite-%s.mmdb.gz\n", lastMonth)
		os.Exit(1)
	}

	if info, err := os.Stat(dbipFile); err == nil {
		sizeStr := dl.HumanSize(info.Size())
		fmt.Println()
		fmt.Println("文件信息:")
		fmt.Printf("  路径: %s\n", dbipFile)
		fmt.Printf("  大小: %s\n", sizeStr)
		fmt.Println()
		fmt.Println("✅ DB-IP 数据库下载完成！")
		fmt.Println()
		fmt.Println("注意：此数据库使用 MMDB 格式，与 GeoLite2 兼容")
		fmt.Println("系统会自动使用此数据库进行 IP 地理位置解析")
	} else {
		fmt.Println("❌ 文件下载失败")
		os.Exit(1)
	}
}
