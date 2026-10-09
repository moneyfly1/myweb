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

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/services/invite"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "只统计不写库（演练）")
	rule := flag.String("rule", invite.RuleRegistration, "补发规则：registration | paid")
	fallback := flag.Float64("fallback-amount", 0, "全局配置读不到时的兜底金额（默认 0 = 不补）")
	migrate := flag.Bool("migrate-settings", true, "补发前先把旧键值迁移到当前键")
	flag.Parse()

	if _, err := config.LoadConfig(); err != nil {
		log.Fatalf("加载配置失败: %v", err)
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
