// 临时诊断工具：遍历所有活跃订阅，拉取 Clash 配置并用 yaml.v3 验证可解析性，
// 找出客户端 "Parse config error: yaml: line N" 对应的用户配置。
//
// 用法（生产项目目录）：go run ./cmd/checkyaml
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	yaml "gopkg.in/yaml.v3"
	"gorm.io/gorm"

	"cboard-go/internal/core/dbtool"
	"cboard-go/internal/models"
)

func main() {
	db, err := dbtool.Open()
	if err != nil {
		fmt.Println("打开数据库失败:", err)
		os.Exit(1)
	}

	var subs []models.Subscription
	if err := db.Where("is_active = ?", true).Find(&subs).Error; err != nil {
		fmt.Println("查询订阅失败:", err)
		os.Exit(1)
	}
	fmt.Printf("活跃订阅 %d 个 (UA=MoneyFly)\n", len(subs))

	outDir := "/tmp/badsub"
	_ = os.MkdirAll(outDir, 0o755)

	client := &http.Client{Timeout: 60 * time.Second}
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		bad     int
		okCount int
		sem     = make(chan struct{}, 8)
	)

	for i := range subs {
		sub := subs[i]
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			url := fmt.Sprintf("http://127.0.0.1:8000/api/v1/client/subscribe?token=%s&type=clash", sub.SubscriptionURL)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("User-Agent", "MoneyFly/2.2.37 (Windows NT 10.0)")
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil || resp.StatusCode != 200 {
				return
			}
			var out map[string]any
			if err := yaml.Unmarshal(body, &out); err != nil {
				mu.Lock()
				bad++
				mu.Unlock()
				path := filepath.Join(outDir, fmt.Sprintf("sub_%d.yaml", sub.ID))
				_ = os.WriteFile(path, body, 0o644)
				fmt.Printf("FAIL sub=%d user=%d size=%d err=%v -> %s\n", sub.ID, sub.UserID, len(body), err, path)
				return
			}
			mu.Lock()
			okCount++
			mu.Unlock()
		}()
	}
	wg.Wait()

	fmt.Printf("\n完成：通过 %d，失败 %d\n", okCount, bad)
	_ = gorm.ErrRecordNotFound
}
