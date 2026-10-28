// Command backfill-user-levels 按累计消费回填用户等级（并可选重算累计消费）。
//
// 背景（2026-10-28）：等级此前只在「订单支付成功」时更新一次
// （order.processPaidOrderTx → updateUserLevelTx），因此
//   - 配置等级前就已经消费的老用户永远不会被升级；
//   - 历史累计消费按订单原价累加（未扣优惠），比实际支付金额偏高几十元，
//     配置门槛后可能让用户虚高升一档。
//
// 本工具两部分，默认全部为演练（不写库）：
//  1. 重算累计消费（-recalc-consumption）：按 status='paid' 的订单重算
//     total_consumption = Σ(amount - discount_amount)，退款单（status='refunded'）不计入。
//  2. 回填等级：用 services/userlevel 的同一套匹配规则（只升不降）挂等级。
//
// 用法（在站点目录下执行，会读取同目录 .env）：
//
//	go run ./cmd/backfill-user-levels                      # 演练：只报告差异
//	go run ./cmd/backfill-user-levels -apply                # 执行回填等级
//	go run ./cmd/backfill-user-levels -recalc-consumption -apply  # 先重算消费再回填
//
// 幂等：可重复执行；已经是目标等级的用户不会产生写入。
package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/core/dbtool"
	"cboard-go/internal/models"
	"cboard-go/internal/services/userlevel"
	"cboard-go/internal/utils"

	"gorm.io/gorm"
)

const batchSize = 500

// recalcConsumption 按已支付订单重算累计消费（退款单 status='refunded' 天然被排除）。
func recalcConsumption(db *gorm.DB, userID uint) (float64, error) {
	var orders []struct {
		Amount   float64
		Discount float64
	}
	err := db.Model(&models.Order{}).
		Select("COALESCE(amount,0) AS amount, COALESCE(discount_amount,0) AS discount").
		Where("user_id = ? AND status = ?", userID, "paid").
		Scan(&orders).Error
	if err != nil {
		return 0, err
	}
	sum := 0.0
	for _, o := range orders {
		value := o.Amount - o.Discount
		if value > 0 {
			sum += value
		}
	}
	return utils.RoundFloat(sum, 2), nil
}

func levelName(db *gorm.DB, id int64) string {
	var lvl models.UserLevel
	if err := db.First(&lvl, id).Error; err != nil {
		return "(已删除)"
	}
	return lvl.LevelName
}

func main() {
	apply := flag.Bool("apply", false, "实际写库（默认演练）")
	recalc := flag.Bool("recalc-consumption", false, "同时按已支付订单重算 users.total_consumption")
	show := flag.Int("show", 30, "每类明细最多打印条数")
	flag.Parse()

	if _, err := config.LoadConfig(); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 防呆：相对路径的 sqlite 由 database 层锚定到「可执行文件所在目录」，
	// 编译到别处执行会解析到另一侧的库（甚至新建空库）。先打印实际目标。
	if cfg := config.AppConfig; cfg != nil && strings.Contains(strings.ToLower(cfg.DatabaseURL), "sqlite") {
		fmt.Printf("目标数据库: %s\n", database.ResolveSQLitePath(cfg.DatabaseURL))
	}

	db, err := dbtool.Open()
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer database.CloseDatabase()

	levels, err := userlevel.ActiveLevels(db)
	if err != nil {
		log.Fatalf("读取等级配置失败: %v", err)
	}
	fmt.Printf("等级配置（%d 个启用）:\n", len(levels))
	for _, l := range levels {
		fmt.Printf("  #%d %-6s 门槛 %8.2f 元  折扣 %.2f%%  颜色 %s\n",
			l.LevelOrder, l.LevelName, l.MinConsumption, (1-l.DiscountRate)*100, l.Color)
	}
	if len(levels) == 0 {
		fmt.Println("⚠️  没有任何启用的等级，回填不会产生变化。请先在「用户等级管理」里添加等级。")
	}

	mode := "演练（不写库）"
	if *apply {
		mode = "执行（写库）"
	}
	fmt.Printf("模式: %s | 重算累计消费: %v\n\n", mode, *recalc)

	var (
		total            int
		consChanged      []string
		levelChanged     []string
		levelHigher      []string // 当前等级高于消费档（只升不降导致，仅提示）
		noMatch          int
		consTotalDelta   float64
		updatedConsumers int
		updatedLevels    int
	)

	var users []models.User
	if err := db.Select("id", "email", "username", "total_consumption", "user_level_id").
		Order("id ASC").FindInBatches(&users, batchSize, func(tx *gorm.DB, _ int) error {
		for i := range users {
			u := &users[i]
			total++

			if *recalc {
				newCons, err := recalcConsumption(db, u.ID)
				if err != nil {
					return err
				}
				if math.Abs(newCons-u.TotalConsumption) > 0.005 {
					consTotalDelta += newCons - u.TotalConsumption
					if len(consChanged) < *show {
						consChanged = append(consChanged, fmt.Sprintf(
							"  用户 #%-6d %-24s 累计消费 %.2f → %.2f（差 %+.2f）",
							u.ID, u.Email, u.TotalConsumption, newCons, newCons-u.TotalConsumption))
					}
					if *apply {
						if err := db.Model(&models.User{}).Where("id = ?", u.ID).
							Update("total_consumption", newCons).Error; err != nil {
							return err
						}
					}
					u.TotalConsumption = newCons
					updatedConsumers++
				}
			}

			target, changed, err := userlevel.Plan(db, u)
			if err != nil {
				return err
			}
			if target == nil {
				noMatch++
				continue
			}
			if !changed {
				// 当前等级 order 高于目标：只升不降，单独提示
				if u.UserLevelID.Valid && target.LevelOrder < levelOrder(db, u.UserLevelID.Int64) {
					if len(levelHigher) < *show {
						levelHigher = append(levelHigher, fmt.Sprintf(
							"  用户 #%-6d 当前 %s 高于消费档 %s（消费 %.2f）",
							u.ID, levelName(db, u.UserLevelID.Int64), target.LevelName, u.TotalConsumption))
					}
				}
				continue
			}
			oldName := "(无)"
			if u.UserLevelID.Valid {
				oldName = levelName(db, u.UserLevelID.Int64)
			}
			if len(levelChanged) < *show {
				levelChanged = append(levelChanged, fmt.Sprintf(
					"  用户 #%-6d %-24s %-6s → %-6s（累计消费 %.2f）",
					u.ID, u.Email, oldName, target.LevelName, u.TotalConsumption))
			}
			if *apply {
				if _, _, err := userlevel.Sync(db, u); err != nil {
					return err
				}
			}
			updatedLevels++
		}
		return nil
	}).Error; err != nil {
		log.Fatalf("遍历用户失败: %v", err)
	}

	sort.Slice(consChanged, func(i, j int) bool { return consChanged[i] < consChanged[j] })

	fmt.Printf("用户总数: %d\n", total)
	if *recalc {
		fmt.Printf("累计消费需重算: %d 人，合计差 %+.2f 元\n", updatedConsumers, consTotalDelta)
		for _, line := range consChanged {
			fmt.Println(line)
		}
		if updatedConsumers > len(consChanged) {
			fmt.Printf("  …其余 %d 人省略\n", updatedConsumers-len(consChanged))
		}
		fmt.Println()
	}
	levelVerb := "将写入"
	if *apply {
		levelVerb = "写入"
	}
	fmt.Printf("等级需变更: %d 人（%s）\n", updatedLevels, levelVerb)
	for _, line := range levelChanged {
		fmt.Println(line)
	}
	if updatedLevels > len(levelChanged) {
		fmt.Printf("  …其余 %d 人省略\n", updatedLevels-len(levelChanged))
	}
	if noMatch > 0 {
		fmt.Printf("未达任何等级门槛: %d 人（保持无等级）\n", noMatch)
	}
	if len(levelHigher) > 0 {
		fmt.Printf("当前等级高于消费档（只升不降，保持不动）: %d 人\n", len(levelHigher))
		for _, line := range levelHigher {
			fmt.Println(line)
		}
	}
	if !*apply && (updatedLevels > 0 || updatedConsumers > 0) {
		fmt.Println("\n以上为演练结果，加 -apply 才会真正写库。")
	}
}

// levelOrder 读取等级排序值（等级可能已被删除，返回 0）。
func levelOrder(db *gorm.DB, id int64) int {
	var lvl models.UserLevel
	if err := db.First(&lvl, id).Error; err != nil {
		return 0
	}
	return lvl.LevelOrder
}
