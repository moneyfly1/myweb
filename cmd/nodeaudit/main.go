// nodeaudit 全库节点体检工具（只读）
//
// 用途：上线前/日常排查"当前库里是否存在会让整份订阅失效的坏节点"。
// 做法：逐节点跑第一层静态校验 → 用存活节点生成 Clash 配置 → 真内核 mihomo -t →
// 若失败则折半二分定位到具体坏节点。
//
// 只读：仅 SELECT，不写库、不删数据、不改任何节点。
//
// 用法：
//
//	go run ./cmd/nodeaudit -db ./cboard.db -kernel ./bin/mihomo
//	go run ./cmd/nodeaudit -db ./cboard.db -kernel ./bin/mihomo -json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"cboard-go/internal/services/config_update"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	dbPath := flag.String("db", "cboard.db", "SQLite 数据库路径")
	kernel := flag.String("kernel", "bin/mihomo", "mihomo 内核二进制路径（传空串则只做静态校验）")
	asJSON := flag.Bool("json", false, "以 JSON 输出")
	flag.Parse()

	// 只读打开，物理上杜绝误写
	dsn := fmt.Sprintf("file:%s?mode=ro", *dbPath)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "打开数据库失败(%s): %v\n", dsn, err)
		os.Exit(1)
	}

	report, err := config_update.AuditActiveNodes(db, *kernel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "体检失败: %v\n", err)
		os.Exit(1)
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(report)
		return
	}

	fmt.Printf("=== MoneyFly 节点体检报告 ===\n")
	fmt.Printf("活跃节点总数        : %d\n", report.Total)
	fmt.Printf("静态校验保留        : %d\n", report.Kept)
	fmt.Printf("静态校验丢弃        : %d\n", report.Dropped)
	fmt.Printf("自动修正（密钥编码）: %d\n", report.Corrected)
	fmt.Printf("生成配置大小        : %d 字节\n", report.ConfigBytes)
	if !report.KernelRan {
		fmt.Printf("内核自检            : 未执行（未提供内核）\n")
	} else {
		fmt.Printf("内核自检            : %s（%.2fs，内核调用 %d 次）\n",
			map[bool]string{true: "通过", false: "失败"}[report.KernelOK],
			float64(report.KernelDurationMS)/1000, report.KernelCalls)
		if report.KernelError != "" {
			fmt.Printf("内核自检详情        : %s\n", report.KernelError)
		}
	}

	printEntries := func(title string, list []config_update.NodeAuditEntry) {
		if len(list) == 0 {
			return
		}
		fmt.Printf("\n--- %s（%d 条）---\n", title, len(list))
		for _, e := range list {
			fmt.Printf("  [%s] %s | type=%s cipher=%s %s:%d\n     原因: %s\n",
				e.Action, e.Name, e.Type, e.Cipher, e.Server, e.Port, e.Reason)
		}
	}
	printEntries("静态校验丢弃 / 修正", report.Entries)
	printEntries("内核自检剔除", report.KernelPruned)

	// 聚合结论：回答"这类坏节点现网还有多少个、集中在哪"
	if len(report.DroppedByReason) > 0 {
		fmt.Printf("\n--- 按原因聚合（丢弃 %d 个）---\n", report.Dropped)
		for _, r := range report.DroppedByReason {
			fmt.Printf("  %-34s %4d 个   样例: %s\n", r.ReasonCode, r.Count, r.Sample)
		}
	}
	if len(report.DroppedByTypeCipher) > 0 {
		fmt.Printf("\n--- 按 协议类型 + cipher 聚合 ---\n")
		for _, t := range report.DroppedByTypeCipher {
			fmt.Printf("  type=%-10s cipher=%-32s %4d 个\n", t.Type, t.Cipher, t.Count)
		}
	}
	if len(report.DroppedBySource) > 0 {
		fmt.Printf("\n--- 按来源订阅编号聚合 ---\n")
		for _, sc := range report.DroppedBySource {
			src := fmt.Sprintf("%d", sc.SourceIndex)
			if sc.SourceIndex == 0 {
				src = "0(手动/未知)"
			}
			fmt.Printf("  source_index=%-12s %4d 个\n", src, sc.Count)
		}
	}
	if fr, err := config_update.AuditOutputFormats(mustDB(*dbPath)); err == nil {
		fmt.Printf("\n--- 全格式结构校验（不依赖客户端内核；共 %d 个活跃节点）---\n", fr.Total)
		for _, r := range fr.Formats {
			mark := "✓"
			if !r.Validated {
				mark = "✗"
			}
			fmt.Printf("  %-14s %-16s 节点=%-4d %6d bytes  校验=%s %s\n",
				r.Format, r.ContentType, r.Nodes, r.Bytes, mark, r.Error)
			for i, d := range r.NodeDropped {
				if i >= 4 {
					fmt.Printf("        剔除: ... 其余 %d 条省略\n", len(r.NodeDropped)-4)
					break
				}
				fmt.Printf("        剔除: %s\n", d)
			}
		}
		fmt.Printf("  → 全部格式结构校验: %s\n", map[bool]string{true: "通过", false: "存在失败"}[fr.AllOK])
	} else if err != nil {
		fmt.Printf("\n--- 全格式结构校验: 执行失败 %v\n", err)
	}

	if report.LinkRoundTripTotal > 0 {
		n := len(report.LinkRoundTripMismatches)
		fmt.Printf("\n--- 通用订阅链接往返体检（%d 个节点）---\n", report.LinkRoundTripTotal)
		if n == 0 {
			fmt.Printf("  全部往返一致 ✓（nodeToLink → ParseNodeLink 逐字段相等）\n")
		} else {
			fmt.Printf("  !! %d 个节点往返不一致（通用/Shadowrocket/v2rayN 客户会拿到坏节点）\n", n)
			for i, m := range report.LinkRoundTripMismatches {
				if i >= 10 {
					fmt.Printf("  ... 其余 %d 条省略\n", n-10)
					break
				}
				fmt.Printf("  [%s] %s | type=%s 字段=%s\n     %s\n     链接(脱敏): %s\n",
					"mismatch", m.NodeName, m.Type, m.Field, m.Detail, m.LinkSafe)
			}
		}
	}

	c := report.Credentials
	if c.Scanned > 0 {
		fmt.Printf("\n--- 凭据形态体检（ss/ssr 共 %d 个）---\n", c.Scanned)
		row := func(label, disposition string, n int) {
			fmt.Printf("  %-42s %4d 个   %s\n", label, n, disposition)
		}
		row("单段合法密钥 (base64 16/32B)", "可用", c.SingleSegmentOK)
		row("两段式 serverKey:userKey 合法", "可用（内核接受）", c.TwoSegmentOK)
		row("URL 编码（%XX）但解码后合法", "**自动修正并保留**", c.URLCorrectable)
		row("非 2022 cipher（口令为任意串）", "不适用校验", c.NonKeyPassword)
		row("URL 编码但解码后仍不合法", "必须丢弃 cipher-key-invalid", c.URLNotCorrectable)
		row("密钥长度不符", "必须丢弃 cipher-key-invalid", c.LengthMismatch)
		row("非合法 base64", "必须丢弃 cipher-key-invalid", c.NotBase64)
		fmt.Printf("  → 可自动修正: %d 个 | 必须丢弃: %d 个\n", c.Correctable(), c.NeedDrop())
	}

	if len(report.CorrectedByReason) > 0 {
		fmt.Printf("\n--- 自动修正（已修复并保留，未丢弃）---\n")
		for _, r := range report.CorrectedByReason {
			fmt.Printf("  %-34s %4d 个   样例: %s\n", r.ReasonCode, r.Count, r.Sample)
		}
	}
}

// mustDB 只读打开数据库（供全格式结构校验复用）
func mustDB(path string) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=ro", path)), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil
	}
	return db
}
