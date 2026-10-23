// Command revoke-invalid-invite-rewards 撤回「未消费却已发放」的邀请奖励。
//
// 背景（业务规则）：邀请奖励只在**被邀请人真正消费**（首单支付成功）后结算。
// 线上曾用 backfill-invite-rewards 的 registration 规则补发过一次，
// 把「注册了但没花钱」的关系也算进去了 —— 典型表现：某用户邀请 4 人、只有 2 人付费，
// 却按 4 人发了 80 元（应为 40 元）。本工具把多发的部分退回来。
//
// 判定口径：invite_relations 中「已标记发放」但 `invitee_first_order_id` 为空/0 的关系
// （= 被邀请人没有已支付首单）。已消费的关系一律不动。
//
// 默认 dry-run，只打印将要执行的扣回明细；确认后加 -apply 真正写库。
// 每一条扣回都在同一事务内完成：扣余额 → 写余额日志（commission 反向）→
// 冲销佣金日志（status=cancelled）→ 重置关系上的发放标记（将来该用户消费时可正常结算）。
package main

import (
	"flag"
	"fmt"
	"log"

	"cboard-go/internal/core/database"
	"cboard-go/internal/core/dbtool"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"gorm.io/gorm"
)

type pendingRevoke struct {
	RelationID   uint
	UserID       uint
	Email        string
	Balance      float64
	Amount       float64
	Role         string // 邀请人 / 被邀请人
	Insufficient bool   // 余额不足以扣回
}

func main() {
	apply := flag.Bool("apply", false, "真正写库（缺省只预演，不修改任何数据）")
	onlyInviter := flag.Bool("only-inviter", false, "只撤回邀请人奖励（默认两侧都撤回）")
	flag.Parse()

	db, err := dbtool.Open()
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer database.CloseDatabase()

	// 找出「已发放但被邀请人未消费」的关系
	var relations []models.InviteRelation
	if err := db.Where("(inviter_reward_given = ? OR invitee_reward_given = ?) AND (invitee_first_order_id IS NULL OR invitee_first_order_id = 0)", true, true).
		Order("id ASC").Find(&relations).Error; err != nil {
		log.Fatalf("查询邀请关系失败: %v", err)
	}

	var pending []pendingRevoke
	totalInviter, totalInvitee := 0.0, 0.0
	for _, rel := range relations {
		if rel.InviterRewardGiven && rel.InviterRewardAmount > 0 {
			item := buildRevoke(db, rel.InviterID, rel.InviterRewardAmount, "邀请人", rel.ID)
			pending = append(pending, item)
			totalInviter += rel.InviterRewardAmount
		}
		if !*onlyInviter && rel.InviteeRewardGiven && rel.InviteeRewardAmount > 0 {
			item := buildRevoke(db, rel.InviteeID, rel.InviteeRewardAmount, "被邀请人", rel.ID)
			pending = append(pending, item)
			totalInvitee += rel.InviteeRewardAmount
		}
	}

	fmt.Printf("=== 待撤回明细（%d 条关系，%d 笔扣回）===\n", len(relations), len(pending))
	fmt.Printf("%-8s %-8s %-30s %-8s %-10s %-10s %s\n", "关系", "用户ID", "邮箱", "角色", "金额", "当前余额", "备注")
	insufficient := 0
	for _, p := range pending {
		note := ""
		if p.Insufficient {
			note = "余额不足，将扣至 0"
			insufficient++
		}
		fmt.Printf("%-8d %-8d %-30s %-8s %-10.2f %-10.2f %s\n", p.RelationID, p.UserID, p.Email, p.Role, p.Amount, p.Balance, note)
	}
	fmt.Printf("\n合计：邀请人侧 %.2f 元 / 被邀请人侧 %.2f 元 / 总计 %.2f 元（余额不足 %d 笔）\n",
		totalInviter, totalInvitee, totalInviter+totalInvitee, insufficient)

	if !*apply {
		fmt.Println("\n[预演] 未修改任何数据。确认无误后加 -apply 执行。")
		return
	}

	// 真正执行：整批一个事务，任一条失败全部回滚
	revoked := 0
	err = db.Transaction(func(tx *gorm.DB) error {
		for _, rel := range relations {
			if err := revokeRelation(tx, &rel, *onlyInviter); err != nil {
				return fmt.Errorf("关系 %d 撤回失败: %w", rel.ID, err)
			}
			revoked++
		}
		return nil
	})
	if err != nil {
		log.Fatalf("撤回失败（已回滚，数据未变）: %v", err)
	}
	fmt.Printf("\n✅ 已撤回 %d 条关系的错误奖励，合计 %.2f 元\n", revoked, totalInviter+totalInvitee)
	fmt.Println("说明：关系上的发放标记已重置，这些被邀请人将来真正消费时，会自动按当时配置正常结算奖励。")
}

func buildRevoke(db *gorm.DB, userID uint, amount float64, role string, relID uint) pendingRevoke {
	var user models.User
	_ = db.Select("id, email, balance").First(&user, userID).Error
	return pendingRevoke{
		RelationID:   relID,
		UserID:       userID,
		Email:        user.Email,
		Balance:      user.Balance,
		Amount:       amount,
		Role:         role,
		Insufficient: user.Balance < amount,
	}
}

// revokeRelation 撤回单条关系的错误奖励（邀请人侧 + 被邀请人侧）。
func revokeRelation(tx *gorm.DB, rel *models.InviteRelation, onlyInviter bool) error {
	if rel.InviterRewardGiven && rel.InviterRewardAmount > 0 {
		if err := revokeTx(tx, rel.InviterID, rel.InviterRewardAmount, rel, true,
			fmt.Sprintf("邀请奖励撤回：被邀请人（ID %d）未消费，追回错误发放的邀请人奖励", rel.InviteeID)); err != nil {
			return err
		}
	}
	if !onlyInviter && rel.InviteeRewardGiven && rel.InviteeRewardAmount > 0 {
		if err := revokeTx(tx, rel.InviteeID, rel.InviteeRewardAmount, rel, false,
			fmt.Sprintf("邀请奖励撤回：本人未消费，追回错误发放的被邀请人奖励（关系 %d）", rel.ID)); err != nil {
			return err
		}
	}
	return nil
}

// revokeTx 扣回单个用户的一笔奖励。
func revokeTx(tx *gorm.DB, userID uint, amount float64, rel *models.InviteRelation, isInviter bool, desc string) error {
	var user models.User
	if err := tx.First(&user, userID).Error; err != nil {
		return err
	}
	oldBalance := user.Balance
	// 余额不足时扣到 0（不产生负余额），并在日志里说明实际扣减额
	deduct := amount
	if oldBalance < amount {
		deduct = oldBalance
	}
	updates := map[string]interface{}{"balance": gorm.Expr("balance - ?", deduct)}
	if isInviter {
		updates["total_invite_reward"] = gorm.Expr("total_invite_reward - ?", deduct)
	}
	if err := tx.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return err
	}
	var fresh models.User
	if err := tx.First(&fresh, userID).Error; err != nil {
		return err
	}

	if deduct < amount {
		desc += fmt.Sprintf("（余额不足，实际扣减 %.2f，应扣 %.2f）", deduct, amount)
	}
	if err := utils.CreateBalanceLogWithDB(tx, userID, "commission", -deduct,
		oldBalance, fresh.Balance, nil, nil, desc, "system", nil, ""); err != nil {
		return err
	}

	// 关系侧重置发放标记（将来消费可重新结算）
	if isInviter {
		rel.InviterRewardGiven = false
		rel.InviterRewardAmount = 0
	} else {
		rel.InviteeRewardGiven = false
		rel.InviteeRewardAmount = 0
	}
	if err := tx.Save(rel).Error; err != nil {
		return err
	}

	// 冲销佣金日志：保留记录可审计，只把状态从 paid 改成 cancelled。
	// commission_logs 里「被邀请人奖励」那条也是 inviter_id=收款人（见 grantTx），
	// 所以按 (关系, 收款人) 定位即可；再限定 register_reward，避免误伤已消费的 order_reward。
	if err := tx.Model(&models.CommissionLog{}).
		Where("invite_relation_id = ? AND inviter_id = ? AND commission_type = ? AND status = ?",
			rel.ID, userID, "register_reward", "paid").
		Update("status", "cancelled").Error; err != nil {
		return err
	}

	fmt.Printf("  - 已撤回 user_id=%d 金额=%.2f\n", userID, deduct)
	return nil
}
