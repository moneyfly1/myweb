// Command backfill-invite-rewards 补发历史上漏发的邀请奖励。
//
// 背景：设置键名从 invite_inviter_reward / invite_invitee_reward 改成
// inviter_reward / invitee_reward 时没有迁移数据，旧键里的奖励金额成了孤儿、
// 新键为 0；而奖励发放的判断是「金额 > 0 才发」，于是全站邀请奖励一封未发，
// 邀请码的奖励金额又在注册时被冻结为 0，事后改配置也补不回来。
//
// 本命令按当前配置（含旧键回退）重新计算金额并补发，幂等、可重跑。
//
// 用法（在站点目录下执行，会读取同目录 .env）：
//
//	go run ./cmd/backfill-invite-rewards --dry-run              # 演练，只统计不写库
//	go run ./cmd/backfill-invite-rewards                        # 执行（默认 registration 规则）
//	go run ./cmd/backfill-invite-rewards --rule=paid            # 只补「被邀请人已付款」的关系
//	go run ./cmd/backfill-invite-rewards --fallback-amount=20   # 全局配置读不到时按 20 元补
//
// 规则说明：
//
//	registration（默认，忠实于代码原意）：最低消费为 0 的邀请码，注册即发；
//	                                      设置了最低消费的邀请码，被邀请人付款后发。
//	paid：只补被邀请人已付款的关系（更保守）。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/services/invite"
)

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "?"
	}
	return wd
}

func main() {
	dryRun := flag.Bool("dry-run", false, "只统计不写库（演练）")
	rule := flag.String("rule", invite.RuleRegistration, "补发规则：registration | paid")
	fallback := flag.Float64("fallback-amount", 0, "全局配置读不到时的兜底金额（默认 0 = 不补）")
	migrate := flag.Bool("migrate-settings", true, "补发前先把旧键值迁移到当前键")
	flag.Parse()

	if _, err := config.LoadConfig(); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 防呆：数据库路径默认是「工作目录下的 cboard.db」。用 `go run` 执行时，
	// 二进制会被放到 go-build 缓存目录里运行，工作目录不是站点目录 ——
	// 于是会静默新建一个空库、扫描到 0 条关系，看起来像「没有需要补发的」。
	// 这里先确认目标库真实存在，并要求用 DATABASE_URL 或编译后的二进制运行。
	dbFile := ""
	if du := strings.TrimSpace(os.Getenv("DATABASE_URL")); du != "" {
		if strings.Contains(strings.ToLower(du), "sqlite") {
			// sqlite:///abs/path 或 sqlite://rel/path 或 sqlite:///./rel
			trimmed := du[strings.Index(du, "://")+3:]
			trimmed = strings.TrimPrefix(trimmed, "/")
			if strings.HasPrefix(du, "sqlite:////") {
				dbFile = "/" + strings.TrimPrefix(trimmed, "/")
			} else {
				dbFile = trimmed
			}
			dbFile = strings.TrimPrefix(dbFile, "./")
		}
	} else {
		dbFile = "cboard.db"
	}
	if dbFile != "" {
		abs, _ := filepath.Abs(dbFile)
		st, err := os.Stat(dbFile)
		if err != nil {
			fmt.Printf("找不到数据库：%s（当前工作目录 %s）\n", abs, mustGetwd())
			fmt.Println("用法提示：二选一")
			fmt.Println("  1) 指定库路径：DATABASE_URL='sqlite:////绝对路径/cboard.db' go run ./cmd/backfill-invite-rewards --dry-run")
			fmt.Println("  2) 先编译再在站点目录执行：go build -o /tmp/bf-invite ./cmd/backfill-invite-rewards && cd /www/wwwroot/<站点> && /tmp/bf-invite --dry-run")
			fmt.Println("（用 go run 直接跑时工作目录是 go-build 缓存目录，会新建空库、扫到 0 条关系）")
			os.Exit(2)
		}
		fmt.Printf("目标数据库: %s（%.1f MB）\n", abs, float64(st.Size())/1024/1024)
	}

	if err := database.InitDatabase(); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer database.CloseDatabase()
	db := database.GetDB()
	if db == nil {
		log.Fatal("数据库未就绪")
	}

	if *migrate {
		migrated, err := invite.MigrateLegacySettings(db)
		if err != nil {
			log.Fatalf("邀请设置迁移失败: %v", err)
		}
		if len(migrated) > 0 {
			fmt.Printf("已迁移旧键值到当前键: %v\n", migrated)
		}
	}

	rewards := invite.LoadRewards(db)
	fmt.Printf("当前奖励设置: 邀请人 %.2f / 被邀请人 %.2f（来源 %s）\n", rewards.Inviter, rewards.Invitee, rewards.Source)
	if rewards.Inviter <= 0 && rewards.Invitee <= 0 {
		fmt.Println("⚠️  奖励金额为 0，补发不会有任何效果。请先在「系统设置 → 邀请设置」配置奖励金额，或使用 --fallback-amount。")
	}

	mode := "执行补发"
	if *dryRun {
		mode = "演练（不写库）"
	}
	fmt.Printf("规则: %s | 模式: %s\n\n", *rule, mode)

	res, err := invite.Backfill(db, invite.BackfillOptions{
		DryRun:         *dryRun,
		Rule:           *rule,
		FallbackAmount: *fallback,
	})
	if err != nil {
		log.Fatalf("补发失败: %v", err)
	}

	// 明细：只打印真正会补/已补的条目，避免刷屏
	shown := 0
	for _, it := range res.Items {
		if !it.GrantedInviter && !it.GrantedInvitee && it.InviterAmount == 0 && it.InviteeAmount == 0 {
			continue
		}
		fmt.Printf("  关系 #%d  邀请人=%d  被邀请人=%d  已付款=%v  邀请人奖励=%.2f  被邀请人奖励=%.2f\n",
			it.RelationID, it.InviterID, it.InviteeID, it.Paid, it.InviterAmount, it.InviteeAmount)
		shown++
	}
	if shown == 0 {
		fmt.Println("  （没有需要补发的关系）")
	}

	fmt.Printf("\n扫描 %d 条邀请关系 | 命中规则 %d 条 | 跳过 %d 条\n", res.Scanned, res.Eligible, res.Skipped)
	fmt.Printf("补发邀请人奖励 %d 笔，合计 ¥%.2f\n", res.GrantedInviter, res.InviterTotal)
	fmt.Printf("补发被邀请人奖励 %d 笔，合计 ¥%.2f\n", res.GrantedInvitee, res.InviteeTotal)
	fmt.Printf("总计 ¥%.2f\n", res.InviterTotal+res.InviteeTotal)

	if *dryRun {
		fmt.Println("\n（这是演练结果，未写入任何数据；去掉 --dry-run 即执行）")
	} else if os.Getenv("BACKFILL_JSON") != "" {
		out, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(out))
	}
}
