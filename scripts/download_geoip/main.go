package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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

	if err := downloadFile(geoipURL, geoipFile); err != nil {
		fmt.Printf("❌ 下载失败: %v\n", err)
		os.Exit(1)
	}

	if info, err := os.Stat(geoipFile); err == nil {
		size := float64(info.Size())
		var sizeStr string
		if size < 1024 {
			sizeStr = fmt.Sprintf("%.0f B", size)
		} else if size < 1024*1024 {
			sizeStr = fmt.Sprintf("%.2f KB", size/1024)
		} else {
			sizeStr = fmt.Sprintf("%.2f MB", size/(1024*1024))
		}
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

func downloadFile(url, filePath string) error {
	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	fmt.Print("下载中... ")

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("保存文件失败: %w", err)
	}

	fmt.Printf("✅ 已下载 %d 字节\n", written)
	return nil
}
