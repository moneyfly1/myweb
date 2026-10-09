// Package invite 统一处理邀请奖励：配置解析、奖励发放、邀请记录查询与历史补发。
//
// 背景（线上事故）：2026-04-14 的提交把设置键名从 invite_inviter_reward /
// invite_invitee_reward 改成了 inviter_reward / invitee_reward，但没有迁移库里已有的值。
// 旧键里的 20 元成了孤儿数据，新键被初始化为 0，而奖励发放的判断是
// 「金额 > 0 才发」——于是从那天起，全站邀请奖励一封都没发出去，且邀请码的
// 奖励金额在注册时就被冻结为 0，即使后来改配置也无法补救。
//
// 本包解决三件事：
//  1. 配置：读取当前键，缺失或为 0 时回退旧键；并提供一次性迁移（幂等）。
//  2. 发放：发放时按「邀请码显式设置 > 全局配置」实时解析金额，不再依赖注册时冻结的值。
//  3. 记录/补发：给用户端提供邀请记录（含奖励是否到账），并提供可重跑的历史补发。
package invite

import (
	"fmt"
	"log"
	"strings"

	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 设置分类与键名
const (
	CategoryInvite = "invite"

	KeyInviterReward = "inviter_reward"
	KeyInviteeReward = "invitee_reward"

	// 旧键名（历史遗留，仅用于兼容与迁移）
	LegacyKeyInviterReward = "invite_inviter_reward"
	LegacyKeyInviteeReward = "invite_invitee_reward"
)

// 补发规则
const (
	// RuleRegistration 忠实于代码原意：最低消费为 0 的邀请码，注册即发；
	// 设置了最低消费的邀请码，需要被邀请人付款后才发。
	RuleRegistration = "registration"
	// RulePaid 只补发「被邀请人已付款」的关系（更保守）。
	RulePaid = "paid"
)

// Rewards 全局奖励设置
type Rewards struct {
	Inviter float64 `json:"inviter_reward"`
	Invitee float64 `json:"invitee_reward"`
	// Source 说明取值来源：current / legacy / none（便于诊断）
	Source string `json:"source"`
}

// ---------------------------------------------------------------------------
// 配置
// ---------------------------------------------------------------------------

func readSetting(db *gorm.DB, key string) (float64, bool) {
	if db == nil {
		return 0, false
	}
	var cfg models.SystemConfig
	if err := db.Where("key = ? AND category = ?", key, CategoryInvite).First(&cfg).Error; err != nil {
		return 0, false
	}
	v := strings.TrimSpace(cfg.Value)
	if v == "" {
		return 0, false
	}
	var f float64
	if _, err := fmt.Sscanf(v, "%f", &f); err != nil {
		return 0, false
	}
	return f, true
}

// LoadRewards 读取全局奖励设置：当前键优先，缺失或为 0 时回退旧键。
// 回退时会打日志，避免再次出现「配置看起来设了、实际读的另一个键」这种静默失效。
func LoadRewards(db *gorm.DB) Rewards {
	inviter, okI := readSetting(db, KeyInviterReward)
	invitee, okE := readSetting(db, KeyInviteeReward)

	// 当前键缺失或为 0 时，考虑旧键
	legacyI, legacyIOK := readSetting(db, LegacyKeyInviterReward)
	legacyE, legacyEOK := readSetting(db, LegacyKeyInviteeReward)

	source := "current"
	if (!okI || inviter <= 0) && legacyIOK && legacyI > 0 {
		inviter = legacyI
		source = "legacy"
	}
	if (!okE || invitee <= 0) && legacyEOK && legacyE > 0 {
		invitee = legacyE
		source = "legacy"
	}
	if source == "current" && !okI && !okE {
		source = "none"
	}
	if source == "legacy" {
		log.Printf("邀请奖励：当前键 %s/%s 未配置有效值，已沿用旧键 %s/%s 的值（邀请人 %.2f / 被邀请人 %.2f）",
			KeyInviterReward, KeyInviteeReward, LegacyKeyInviterReward, LegacyKeyInviteeReward, inviter, invitee)
	}
	return Rewards{Inviter: inviter, Invitee: invitee, Source: source}
}

// MigrateLegacySettings 把旧键的值迁移到当前键（幂等）：
// 仅当当前键缺失或 <= 0、且旧键有正值时写入，返回被迁移的键名。
func MigrateLegacySettings(db *gorm.DB) ([]string, error) {
	if db == nil {
		return nil, nil
	}
	pairs := [][2]string{
		{KeyInviterReward, LegacyKeyInviterReward},
		{KeyInviteeReward, LegacyKeyInviteeReward},
	}
	var migrated []string
	for _, p := range pairs {
		cur, curOK := readSetting(db, p[0])
		old, oldOK := readSetting(db, p[1])
		if !oldOK || old <= 0 {
			continue
		}
		if curOK && cur > 0 {
			continue // 当前键已有有效值，不动
		}
		var cfg models.SystemConfig
		err := db.Where("key = ? AND category = ?", p[0], CategoryInvite).First(&cfg).Error
		if err == gorm.ErrRecordNotFound {
			cfg = models.SystemConfig{Key: p[0], Category: CategoryInvite, Value: fmt.Sprintf("%.2f", old)}
			if err := db.Create(&cfg).Error; err != nil {
				return migrated, err
			}
		} else if err != nil {
			return migrated, err
		} else {
			if err := db.Model(&cfg).Update("value", fmt.Sprintf("%.2f", old)).Error; err != nil {
				return migrated, err
			}
		}
		migrated = append(migrated, p[0])
		log.Printf("邀请奖励：旧键 %s=%.2f 已迁移到 %s", p[1], old, p[0])
	}
	return migrated, nil
}

// EffectiveRewards 计算某邀请码实际应发的奖励金额。
// 优先级：邀请码上显式设置的值（>0）> 全局配置。
// 这样即使邀请码是历史上「奖励为 0」时创建的，只要全局配置有值，仍能正常发放。
func EffectiveRewards(db *gorm.DB, code *models.InviteCode) Rewards {
	global := LoadRewards(db)
	out := global
	if code != nil {
		if code.InviterReward > 0 {
			out.Inviter = code.InviterReward
			out.Source = "code"
		}
		if code.InviteeReward > 0 {
			out.Invitee = code.InviteeReward
			out.Source = "code"
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// 发放
// ---------------------------------------------------------------------------

// grantTx 在事务内给用户加余额、维护邀请计数，并写余额日志与佣金日志。
// 已发放（*Given=true）时由调用方提前跳过，本函数只负责「发一次」。
func grantTx(tx *gorm.DB, userID uint, amount float64, inviteeID uint, relation *models.InviteRelation, isInviter bool, desc string) error {
	if amount <= 0 || tx == nil {
		return nil
	}
	var user models.User
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
		return err
	}
	oldBalance := user.Balance

	updates := map[string]interface{}{"balance": gorm.Expr("balance + ?", amount)}
	if isInviter {
		updates["total_invite_reward"] = gorm.Expr("total_invite_reward + ?", amount)
		updates["total_invite_count"] = gorm.Expr("total_invite_count + 1")
	}
	if err := tx.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return err
	}

	var fresh models.User
	if err := tx.First(&fresh, userID).Error; err != nil {
		return err
	}

	if relation != nil {
		if isInviter {
			relation.InviterRewardGiven = true
			relation.InviterRewardAmount = amount
		} else {
			relation.InviteeRewardGiven = true
			relation.InviteeRewardAmount = amount
		}
		if err := tx.Save(relation).Error; err != nil {
			return err
		}
	}

	if err := utils.CreateBalanceLogWithDB(tx, userID, "commission", amount,
		oldBalance, fresh.Balance, nil, nil, desc, "system", nil, ""); err != nil {
		log.Printf("邀请奖励：写余额日志失败（不影响发放）: %v", err)
	}

	if utils.AppLogger != nil {
		role := "被邀请人"
		if isInviter {
			role = "邀请人"
		}
		utils.AppLogger.Info("邀请奖励：✅ 已发放 %s奖励 - user_id=%d, amount=%.2f, relation_id=%d", role, userID, amount, relationIDOf(relation))
	}

	kind := "register_reward"
	if relation != nil && relation.InviteeFirstOrderID.Valid {
		kind = "order_reward"
	}
	var relID *uint
	if relation != nil {
		id := uint(relation.ID)
		relID = &id
	}
	if err := utils.CreateCommissionLogWithDB(tx, userID, inviteeID, kind, amount, relID, nil, desc); err != nil {
		log.Printf("邀请奖励：写佣金日志失败（不影响发放）: %v", err)
	}
	return nil
}

func relationIDOf(relation *models.InviteRelation) uint {
	if relation == nil {
		return 0
	}
	return uint(relation.ID)
}

// GrantForRelation 在给定事务内发放该关系尚未发放的奖励。
// 金额按「邀请码显式值 > 全局配置」实时解析，因此历史上的「奖励 0」关系在配置补上后也能正常发放。
// 调用方负责自己的业务判断（最低消费、仅新用户、首单记录等）。
func GrantForRelation(tx *gorm.DB, relation *models.InviteRelation, code *models.InviteCode, suffix string) (bool, error) {
	if tx == nil || relation == nil {
		return false, nil
	}
	amounts := EffectiveRewards(tx, code)
	granted := false
	if !relation.InviterRewardGiven && amounts.Inviter > 0 {
		desc := "邀请奖励：邀请人奖励"
		if suffix != "" {
			desc += "（" + suffix + "）"
		}
		if err := grantTx(tx, relation.InviterID, amounts.Inviter, relation.InviteeID, relation, true, desc); err != nil {
			return granted, err
		}
		granted = true
	}
	if !relation.InviteeRewardGiven && amounts.Invitee > 0 {
		desc := "邀请奖励：被邀请人奖励"
		if suffix != "" {
			desc += "（" + suffix + "）"
		}
		if err := grantTx(tx, relation.InviteeID, amounts.Invitee, relation.InviteeID, relation, false, desc); err != nil {
			return granted, err
		}
		granted = true
	}
	return granted, nil
}

// GrantForNewInvitee 注册流程使用：按被邀请人找到待发放关系并发放即时奖励。
// 设有最低消费要求的邀请码留给订单支付后发放（与原实现一致）。
func GrantForNewInvitee(db *gorm.DB, inviteeID uint) error {
	if db == nil {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var relation models.InviteRelation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("invitee_id = ?", inviteeID).First(&relation).Error; err != nil {
			return nil // 无邀请关系
		}
		if relation.InviterRewardGiven && relation.InviteeRewardGiven {
			return nil
		}
		var code models.InviteCode
		if err := tx.First(&code, relation.InviteCodeID).Error; err != nil {
			return err
		}
		if code.MinOrderAmount > 0 {
			return nil
		}
		_, err := GrantForRelation(tx, &relation, &code, "注册")
		return err
	})
}

// ---------------------------------------------------------------------------
// 记录查询（用户端「邀请记录」）
// ---------------------------------------------------------------------------

// Record 一条邀请记录（含奖励是否到账）。
//
// 字段命名同时提供两套：语义化的新字段（paid/consumption/...）与用户端历史表格
// 已在使用的字段（has_purchased/total_consumption/reward_given/created_at），
// 这样前端不需要改列定义就能直接拿到数据。
type Record struct {
	InviteeID           uint    `json:"invitee_id"`
	InviteeUsername     string  `json:"invitee_username"`
	InviteeEmail        string  `json:"invitee_email"`
	RegisteredAt        string  `json:"registered_at"`
	CreatedAt           string  `json:"created_at"` // 兼容历史表格字段
	Paid                bool    `json:"paid"`
	HasPurchased        bool    `json:"has_purchased"` // 兼容历史表格字段
	FirstOrderNo        string  `json:"first_order_no,omitempty"`
	Consumption         float64 `json:"consumption"`
	TotalConsumption    float64 `json:"total_consumption"` // 兼容历史表格字段
	InviterRewardAmount float64 `json:"inviter_reward_amount"`
	RewardAmount        float64 `json:"reward_amount"` // 兼容历史表格字段（邀请人视角金额）
	InviterRewardGiven  bool    `json:"inviter_reward_given"`
	RewardGiven         bool    `json:"reward_given"` // 兼容历史表格字段
	InviteeRewardGiven  bool    `json:"invitee_reward_given"`
	StatusText          string  `json:"status_text"`
}

// Summary 邀请汇总
type Summary struct {
	Registered       int64   `json:"registered"`
	Purchased        int64   `json:"purchased"`
	TotalReward      float64 `json:"total_reward"`
	PendingReward    float64 `json:"pending_reward"`
	TotalConsumption float64 `json:"total_consumption"`
}

// ListRecords 返回某邀请人名下的邀请记录与汇总（按注册时间倒序）。
func ListRecords(db *gorm.DB, inviterID uint) ([]Record, Summary, error) {
	var out []Record
	var summary Summary
	if db == nil {
		return out, summary, nil
	}

	type row struct {
		InviteeID           uint
		InviteeUsername     string
		InviteeEmail        string
		CreatedAt           string
		FirstOrderID        *int64
		OrderNo             *string
		Consumption         float64
		InviterRewardAmount float64
		InviterRewardGiven  bool
		InviteeRewardGiven  bool
		InviteeRewardAmount float64
	}
	var rows []row
	err := db.Table("invite_relations AS r").
		Select(`r.invitee_id, COALESCE(u.username,'') AS invitee_username, COALESCE(u.email,'') AS invitee_email,
		        r.created_at AS created_at, r.invitee_first_order_id AS first_order_id, o.order_no AS order_no,
		        r.invitee_total_consumption AS consumption, r.inviter_reward_amount AS inviter_reward_amount,
		        r.inviter_reward_given AS inviter_reward_given, r.invitee_reward_given AS invitee_reward_given,
		        r.invitee_reward_amount AS invitee_reward_amount`).
		Joins("LEFT JOIN users u ON u.id = r.invitee_id").
		Joins("LEFT JOIN orders o ON o.id = r.invitee_first_order_id").
		Where("r.inviter_id = ?", inviterID).
		Order("r.id DESC").
		Scan(&rows).Error
	if err != nil {
		return out, summary, err
	}

	for _, r := range rows {
		rec := Record{
			InviteeID:           r.InviteeID,
			InviteeUsername:     r.InviteeUsername,
			InviteeEmail:        r.InviteeEmail,
			RegisteredAt:        r.CreatedAt,
			CreatedAt:           r.CreatedAt,
			Paid:                r.FirstOrderID != nil,
			HasPurchased:        r.FirstOrderID != nil,
			Consumption:         r.Consumption,
			TotalConsumption:    r.Consumption,
			InviterRewardAmount: r.InviterRewardAmount,
			RewardAmount:        r.InviterRewardAmount,
			InviterRewardGiven:  r.InviterRewardGiven,
			RewardGiven:         r.InviterRewardGiven,
			InviteeRewardGiven:  r.InviteeRewardGiven,
		}
		if r.OrderNo != nil {
			rec.FirstOrderNo = *r.OrderNo
		}
		switch {
		case r.InviterRewardGiven:
			rec.StatusText = fmt.Sprintf("奖励 ¥%.2f 已到账", r.InviterRewardAmount)
		case r.InviterRewardAmount > 0:
			rec.StatusText = fmt.Sprintf("奖励 ¥%.2f 待发放", r.InviterRewardAmount)
		case r.FirstOrderID != nil:
			rec.StatusText = "已下单，奖励待结算"
		default:
			rec.StatusText = "已注册，尚未下单"
		}
		out = append(out, rec)

		summary.Registered++
		if rec.Paid {
			summary.Purchased++
		}
		summary.TotalConsumption += rec.Consumption
		if r.InviterRewardGiven {
			summary.TotalReward += r.InviterRewardAmount
		} else if r.InviterRewardAmount > 0 {
			summary.PendingReward += r.InviterRewardAmount
		}
	}
	return out, summary, nil
}

// ---------------------------------------------------------------------------
// 历史补发
// ---------------------------------------------------------------------------

// BackfillOptions 补发参数
type BackfillOptions struct {
	DryRun bool   // 只统计不写库
	Rule   string // RuleRegistration（默认，忠实原意）或 RulePaid
	// MinAmount 仅在全局配置读取不到时作为兜底金额（默认 0 = 不补）
	FallbackAmount float64
}

// BackfillItem 单个关系的补发明细
type BackfillItem struct {
	RelationID     uint    `json:"relation_id"`
	InviterID      uint    `json:"inviter_id"`
	InviteeID      uint    `json:"invitee_id"`
	Paid           bool    `json:"paid"`
	InviterAmount  float64 `json:"inviter_amount"`
	InviteeAmount  float64 `json:"invitee_amount"`
	GrantedInviter bool    `json:"granted_inviter"`
	GrantedInvitee bool    `json:"granted_invitee"`
	SkipReason     string  `json:"skip_reason,omitempty"`
}

// BackfillResult 补发汇总
type BackfillResult struct {
	Scanned        int            `json:"scanned"`
	Eligible       int            `json:"eligible"`
	GrantedInviter int            `json:"granted_inviter"`
	GrantedInvitee int            `json:"granted_invitee"`
	InviterTotal   float64        `json:"inviter_total"`
	InviteeTotal   float64        `json:"invitee_total"`
	Skipped        int            `json:"skipped"`
	Rewards        Rewards        `json:"rewards"`
	Items          []BackfillItem `json:"items"`
}

// Backfill 按规则补发历史上漏发的邀请奖励。
//
// 幂等：已发放（*Given=true）的关系会跳过；金额解析与线上一致
// （邀请码显式值 > 全局配置）；DryRun 时不写任何数据。
func Backfill(db *gorm.DB, opts BackfillOptions) (BackfillResult, error) {
	res := BackfillResult{}
	if db == nil {
		return res, fmt.Errorf("数据库未初始化")
	}
	if opts.Rule == "" {
		opts.Rule = RuleRegistration
	}
	rewards := LoadRewards(db)
	if rewards.Inviter <= 0 && opts.FallbackAmount > 0 {
		rewards.Inviter = opts.FallbackAmount
	}
	if rewards.Invitee <= 0 && opts.FallbackAmount > 0 {
		rewards.Invitee = opts.FallbackAmount
	}
	res.Rewards = rewards

	var relations []models.InviteRelation
	if err := db.Order("id").Find(&relations).Error; err != nil {
		return res, err
	}
	res.Scanned = len(relations)

	codes := map[uint]models.InviteCode{}
	var allCodes []models.InviteCode
	if err := db.Find(&allCodes).Error; err != nil {
		return res, err
	}
	for _, c := range allCodes {
		codes[c.ID] = c
	}

	for i := range relations {
		rel := relations[i]
		code := codes[rel.InviteCodeID]
		amounts := EffectiveRewards(db, &code)
		paid := rel.InviteeFirstOrderID.Valid

		// 规则判定
		eligible := false
		switch opts.Rule {
		case RulePaid:
			eligible = paid
		default: // RuleRegistration
			eligible = paid || code.MinOrderAmount <= 0
		}
		if !eligible {
			res.Skipped++
			res.Items = append(res.Items, BackfillItem{RelationID: rel.ID, InviterID: rel.InviterID,
				InviteeID: rel.InviteeID, Paid: paid, SkipReason: "不满足补发规则"})
			continue
		}

		needInviter := !rel.InviterRewardGiven && amounts.Inviter > 0
		needInvitee := !rel.InviteeRewardGiven && amounts.Invitee > 0
		if !needInviter && !needInvitee {
			res.Skipped++
			res.Items = append(res.Items, BackfillItem{RelationID: rel.ID, InviterID: rel.InviterID,
				InviteeID: rel.InviteeID, Paid: paid, SkipReason: "已发放或奖励金额为 0"})
			continue
		}

		item := BackfillItem{RelationID: rel.ID, InviterID: rel.InviterID, InviteeID: rel.InviteeID, Paid: paid}
		res.Eligible++

		if opts.DryRun {
			item.InviterAmount = amounts.Inviter
			item.InviteeAmount = amounts.Invitee
			res.Items = append(res.Items, item)
			if needInviter {
				res.GrantedInviter++
				res.InviterTotal += amounts.Inviter
			}
			if needInvitee {
				res.GrantedInvitee++
				res.InviteeTotal += amounts.Invitee
			}
			continue
		}

		err := db.Transaction(func(tx *gorm.DB) error {
			var cur models.InviteRelation
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&cur, rel.ID).Error; err != nil {
				return err
			}
			if needInviter && !cur.InviterRewardGiven {
				if err := grantTx(tx, cur.InviterID, amounts.Inviter, cur.InviteeID, &cur, true,
					"邀请奖励补发：邀请人奖励（历史漏发补发）"); err != nil {
					return err
				}
				item.GrantedInviter = true
				item.InviterAmount = amounts.Inviter
			}
			if needInvitee && !cur.InviteeRewardGiven {
				if err := grantTx(tx, cur.InviteeID, amounts.Invitee, cur.InviteeID, &cur, false,
					"邀请奖励补发：被邀请人奖励（历史漏发补发）"); err != nil {
					return err
				}
				item.GrantedInvitee = true
				item.InviteeAmount = amounts.Invitee
			}
			return nil
		})
		if err != nil {
			return res, fmt.Errorf("补发关系 %d 失败: %w", rel.ID, err)
		}
		if item.GrantedInviter {
			res.GrantedInviter++
			res.InviterTotal += item.InviterAmount
		}
		if item.GrantedInvitee {
			res.GrantedInvitee++
			res.InviteeTotal += item.InviteeAmount
		}
		res.Items = append(res.Items, item)
	}
	return res, nil
}
