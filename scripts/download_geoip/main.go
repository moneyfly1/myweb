package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"cboard-go/internal/core/dl"
)

func main() {
	geoipDir := "."
	if len(os.Args) > 1 {
		geoipDir = os.Args[1]
	}

	geoipFile := filepath.Join(geoipDir, "GeoLite2-City.mmdb")
	geoipURL := "https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb"

	fmt.Println("==========================================")
	fmt.Println("  下载 GeoIP 数据库")
	fmt.Println("==========================================")
	fmt.Println()

	if err := os.MkdirAll(geoipDir, 0755); err != nil {
		fmt.Printf("❌ 创建目录失败: %v\n", err)
		os.Exit(1)
	}

	if _, err := os.Stat(geoipFile); err == nil {
		fmt.Printf("⚠️  GeoIP 数据库文件已存在: %s\n", geoipFile)
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
		os.Remove(geoipFile)
	}

	fmt.Println("正在从 GitHub 下载 GeoIP 数据库...")
	fmt.Printf("URL: %s\n", geoipURL)
	fmt.Printf("保存路径: %s\n", geoipFile)
	fmt.Println()

	if _, err := dl.Download(context.Background(), geoipURL, geoipFile, dl.Options{Label: "下载中"}); err != nil {
		fmt.Printf("❌ 下载失败: %v\n", err)
		os.Exit(1)
	}

	if info, err := os.Stat(geoipFile); err == nil {
		sizeStr := dl.HumanSize(info.Size())
		fmt.Println()
		fmt.Println("文件信息:")
		fmt.Printf("  路径: %s\n", geoipFile)
		fmt.Printf("  大小: %s\n", sizeStr)
		fmt.Println()
		fmt.Println("✅ GeoIP 数据库下载完成！")
	} else {
		fmt.Println("❌ 文件下载失败")
		os.Exit(1)
	}
}
