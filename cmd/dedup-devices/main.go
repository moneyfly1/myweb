// 设备重复行合并工具（存量数据清理）
//
// 背景（2026-09-21）：旧设备身份算法把 App 版本号/系统版本号也算进身份，用户
// 每次升级 App 或系统就会以「新设备」重新登记一行 —— 旧行永远留着占名额，
// 于是出现「设备数虚高」「明明只有一台手机却提示设备超限」「删掉设备后本机
// 再也回不来」等现象。新代码已改用稳定身份（客户端 X-MF-Device-Id 或
// 软件名+系统名+机型+品牌的稳定特征），但历史数据需要清理一次。
//
// 用法（在生产服务器项目目录下执行）：
//
//	# 1) 先预演，只打印将要合并的内容，绝不写库（默认行为）
//	go run ./cmd/dedup-devices
//
//	# 2) 确认无误后执行
//	go run ./cmd/dedup-devices -apply
//
// 合并规则（保守，详见 internal/services/device.MergeDuplicateDevices）：
//   - 只合并同一订阅内**稳定指纹完全相同**的行（四要素齐备且非 Unknown）；
//   - 保留最近使用的那行；整组都被踢下线时保持被踢状态（不偷偷复活设备）；
//   - 合并访问次数/备注/首见时间，随后重算 current_devices。
package main

import (
	"flag"
	"fmt"
	"log"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	devicesvc "cboard-go/internal/services/device"
)

func main() {
	apply := flag.Bool("apply", false, "真正写库（缺省只预演，不修改任何数据）")
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
	fmt.Printf("=== 设备重复行合并 · %s ===\n", mode)

	report, err := devicesvc.MergeDuplicateDevices(db, *apply)
	if err != nil {
		log.Fatalf("合并失败: %v", err)
	}

	for _, line := range report.Details {
		fmt.Println(" -", line)
	}
	fmt.Printf("扫描订阅 %d 个，合并 %d 组，%s %d 行\n",
		report.SubscriptionsScanned, report.GroupsMerged,
		map[bool]string{true: "已删除", false: "将删除"}[*apply], report.RowsRemoved)

	if !*apply && report.GroupsMerged > 0 {
		fmt.Println("\n以上为预演结果。确认无误后加 -apply 执行：")
		fmt.Println("  go run ./cmd/dedup-devices -apply")
	}
}
