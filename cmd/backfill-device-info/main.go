// 设备信息重新分类 / 回填（存量数据清理）
//
// 背景（2026-10-19 用户反馈）：「用户分析里有很多未知设备，看看到底是不是未知设备」。
// 排查结论：多数并不是"识别失败"，而是两类情况 ——
//   1. UA 里**根本没写操作系统**：如 `mihomo.party/v2.0.0 (clash.meta)`、
//      `clash.meta/alpha-…`、`sing-box/1.13.2`（跨平台桌面客户端/裸内核）。
//      旧解析器把它们落到 Clash / Unknown，于是软件名与系统双"未知"。
//   2. 自动化/抓取类 UA：`curl/8.9.1`、`Go-http-client/2.0`、`python-requests` 等，
//      它们**不是用户设备**，以前也混进"未知设备"里拉高了数字。
//
// 本工具用**增强后的解析器**重新解析 user_agent / device_ua，并把结果回填到：
//   software_name / software_version / os_name / os_version /
//   device_model / device_brand / device_type / device_name
//
// 安全策略（默认只预演，绝不写库）：
//   - 只覆盖「空 / Unknown」的字段，**不覆盖已有有效值**（人工标注、X-MF-* 头带来的
//     机型品牌都保留）；
//   - device_type 特殊：Unknown/unknown 才改；
//   - 每条变更都可用 --apply 前后对照统计（识别前 vs 识别后）。
//
// 用法（在项目目录下执行）：
//
//	# 1) 预演：只统计"能识别出什么"，不写库
//	go run ./cmd/backfill-device-info
//
//	# 2) 只看某类（例如自动化 UA 有多少）
//	go run ./cmd/backfill-device-info
//
//	# 3) 确认后写库
//	go run ./cmd/backfill-device-info -apply
package main

import (
	"flag"
	"fmt"
	"log"
	"sort"
	"strings"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	devicesvc "cboard-go/internal/services/device"
)

func main() {
	apply := flag.Bool("apply", false, "真正写库（缺省只预演，不修改任何数据）")
	limit := flag.Int("limit", 0, "只处理前 N 行（0 = 全部，调试用）")
	flag.Parse()

	if _, err := config.LoadConfig(); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	if err := database.InitDatabase(); err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	db := database.GetDB()

	mode := "预演（不写库）"
	if *apply {
		mode = "执行（写库）"
	}
	fmt.Printf("=== 设备信息重新分类 · %s ===\n\n", mode)

	var devices []models.Device
	q := db.Order("id ASC")
	if *limit > 0 {
		q = q.Limit(*limit)
	}
	if err := q.Find(&devices).Error; err != nil {
		log.Fatalf("读取设备失败: %v", err)
	}
	dm := devicesvc.NewDeviceManager()

	type stat struct {
		scanned  int
		changed  int
		software map[string]int // 回填后的软件名分布
		osDist   map[string]int
		samples  []string
	}
	st := stat{software: map[string]int{}, osDist: map[string]int{}}

	for i := range devices {
		d := &devices[i]
		ua := ""
		if d.UserAgent != nil && strings.TrimSpace(*d.UserAgent) != "" {
			ua = *d.UserAgent
		} else if d.DeviceUA != nil {
			ua = *d.DeviceUA
		}
		if strings.TrimSpace(ua) == "" {
			continue
		}
		st.scanned++

		info := dm.ParseUserAgent(ua)
		changedFields := []string{}
		updates := map[string]interface{}{}

		// 只补「空 / Unknown」，绝不覆盖有效值
		isBlank := func(p *string) bool {
			return p == nil || strings.TrimSpace(*p) == "" || strings.EqualFold(strings.TrimSpace(*p), "unknown")
		}
		setIfBlank := func(cur *string, col, val string) {
			if val == "" || strings.EqualFold(val, "unknown") || !isBlank(cur) {
				return
			}
			updates[col] = val
			changedFields = append(changedFields, col)
		}

		setIfBlank(d.SoftwareName, "software_name", info.SoftwareName)
		setIfBlank(d.SoftwareVersion, "software_version", info.SoftwareVersion)
		setIfBlank(d.OSName, "os_name", info.OSName)
		setIfBlank(d.OSVersion, "os_version", info.OSVersion)
		setIfBlank(d.DeviceModel, "device_model", info.DeviceModel)
		setIfBlank(d.DeviceBrand, "device_brand", info.DeviceBrand)

		// device_type：只有 unknown/空才改
		if info.DeviceType != "" && (d.DeviceType == nil ||
			strings.EqualFold(strings.TrimSpace(*d.DeviceType), "unknown") ||
			strings.TrimSpace(*d.DeviceType) == "") {
			updates["device_type"] = info.DeviceType
			changedFields = append(changedFields, "device_type")
		}

		if len(updates) == 0 {
			continue
		}
		st.changed++
		finalSoftware := info.SoftwareName
		if !isBlank(d.SoftwareName) {
			finalSoftware = strings.TrimSpace(*d.SoftwareName)
		}
		st.software[finalSoftware]++
		finalOS := info.OSName
		if !isBlank(d.OSName) {
			finalOS = strings.TrimSpace(*d.OSName)
		}
		st.osDist[finalOS]++
		if len(st.samples) < 12 {
			st.samples = append(st.samples, fmt.Sprintf("设备#%d %s → %s / %s（%s）",
				d.ID, truncate(ua, 46), finalSoftware, finalOS, strings.Join(changedFields, ",")))
		}

		if *apply {
			if err := db.Model(&models.Device{}).Where("id = ?", d.ID).Updates(updates).Error; err != nil {
				log.Printf("更新设备 %d 失败: %v", d.ID, err)
			}
		}
	}

	fmt.Printf("扫描有 UA 的设备: %d 台\n", st.scanned)
	fmt.Printf("可回填（信息更准确）: %d 台\n\n", st.changed)

	fmt.Println("-- 回填后的软件分布 --")
	printTop(st.software, 15)
	fmt.Println("\n-- 回填后的系统分布 --")
	printTop(st.osDist, 15)
	if len(st.samples) > 0 {
		fmt.Println("\n-- 变更样例 --")
		for _, s := range st.samples {
			fmt.Println("  " + s)
		}
	}
	if !*apply {
		fmt.Println("\n以上为预演结果。确认无误后加 -apply 执行：")
		fmt.Println("  go run ./cmd/backfill-device-info -apply")
	}
}

func printTop(m map[string]int, n int) {
	type kv struct {
		k string
		v int
	}
	list := make([]kv, 0, len(m))
	for k, v := range m {
		list = append(list, kv{k, v})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].v != list[j].v {
			return list[i].v > list[j].v
		}
		return list[i].k < list[j].k
	})
	for i, e := range list {
		if i >= n {
			break
		}
		fmt.Printf("  %-24s %d\n", e.k, e.v)
	}
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
