package invite

import (
	"database/sql"
	"fmt"
	"testing"

	"cboard-go/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupInviteDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.SystemConfig{}, &models.User{}, &models.InviteCode{},
		&models.InviteRelation{}, &models.Order{}, &models.BalanceLog{}, &models.CommissionLog{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}

func setCfg(t *testing.T, db *gorm.DB, key, value string) {
	t.Helper()
	if err := db.Create(&models.SystemConfig{Key: key, Value: value, Category: CategoryInvite}).Error; err != nil {
		t.Fatalf("写入配置 %s 失败: %v", key, err)
	}
}

func mkUser(t *testing.T, db *gorm.DB, id uint, balance float64) {
	t.Helper()
	u := models.User{}
	u.ID = id
	u.Username = fmt.Sprintf("user%d", id)
	u.Email = fmt.Sprintf("user%d@example.com", id)
	u.Balance = balance
	u.IsActive = true
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
}

func mkCode(t *testing.T, db *gorm.DB, inviter uint, inviterReward, inviteeReward float64, minOrder float64) models.InviteCode {
	t.Helper()
	c := models.InviteCode{Code: "TESTCODE", UserID: inviter, InviterReward: inviterReward,
		InviteeReward: inviteeReward, MinOrderAmount: minOrder, IsActive: true}
	if err := db.Create(&c).Error; err != nil {
		t.Fatalf("创建邀请码失败: %v", err)
	}
	return c
}

// 核心回归：历史上创建的邀请码奖励为 0，但全局配置有值时必须按全局配置发放，
// 不能再出现「配置改了也发不出奖励」。
func TestEffectiveRewardsFallsBackToGlobalWhenCodeIsZero(t *testing.T) {
	db := setupInviteDB(t)
	setCfg(t, db, KeyInviterReward, "20")
	setCfg(t, db, KeyInviteeReward, "20")

	code := mkCode(t, db, 1, 0, 0, 0) // 历史遗留：码上奖励为 0
	got := EffectiveRewards(db, &code)
	if got.Inviter != 20 || got.Invitee != 20 {
		t.Fatalf("应回退到全局配置 20/20，实际 %+v", got)
	}
}

// 邀请码显式设置优先于全局配置
func TestEffectiveRewardsCodeOverridesGlobal(t *testing.T) {
	db := setupInviteDB(t)
	setCfg(t, db, KeyInviterReward, "20")
	setCfg(t, db, KeyInviteeReward, "20")

	code := mkCode(t, db, 1, 50, 0, 0) // 码上邀请人奖励 50，被邀请人未设置
	got := EffectiveRewards(db, &code)
	if got.Inviter != 50 {
		t.Fatalf("邀请人奖励应取码上的 50，实际 %.2f", got.Inviter)
	}
	if got.Invitee != 20 {
		t.Fatalf("被邀请人奖励应回退全局 20，实际 %.2f", got.Invitee)
	}
}

// 旧键迁移：当前键为 0 时沿用旧键值，并且是幂等的
func TestMigrateLegacySettings(t *testing.T) {
	db := setupInviteDB(t)
	setCfg(t, db, KeyInviterReward, "0")
	setCfg(t, db, LegacyKeyInviterReward, "20")
	setCfg(t, db, LegacyKeyInviteeReward, "30") // 当前键缺失

	migrated, err := MigrateLegacySettings(db)
	if err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	if len(migrated) != 2 {
		t.Fatalf("期望迁移 2 个键，实际 %v", migrated)
	}
	r := LoadRewards(db)
	if r.Inviter != 20 || r.Invitee != 30 {
		t.Fatalf("迁移后取值不符: %+v", r)
	}

	// 幂等：再跑一次不应有变化
	if again, _ := MigrateLegacySettings(db); len(again) != 0 {
		t.Fatalf("重复迁移应无操作，实际 %v", again)
	}
}

// 当前键已有有效值时，不被旧键覆盖（避免把管理员后来设的值改回去）
func TestMigrateLegacyDoesNotOverrideValidCurrent(t *testing.T) {
	db := setupInviteDB(t)
	setCfg(t, db, KeyInviterReward, "88")
	setCfg(t, db, LegacyKeyInviterReward, "20")

	if migrated, _ := MigrateLegacySettings(db); len(migrated) != 0 {
		t.Fatalf("当前键有效时不该迁移，实际 %v", migrated)
	}
	if got := LoadRewards(db).Inviter; got != 88 {
		t.Fatalf("当前值被改坏: %.2f", got)
	}
}

// 发放：注册即时奖励（最低消费为 0 的码）应给邀请人与被邀请人都加余额并置位
func TestGrantForNewInviteeGrantsBothSides(t *testing.T) {
	db := setupInviteDB(t)
	setCfg(t, db, KeyInviterReward, "20")
	setCfg(t, db, KeyInviteeReward, "20")
	mkUser(t, db, 1, 0) // 邀请人
	mkUser(t, db, 2, 5) // 被邀请人
	code := mkCode(t, db, 1, 0, 0, 0)
	rel := models.InviteRelation{InviteCodeID: code.ID, InviterID: 1, InviteeID: 2}
	if err := db.Create(&rel).Error; err != nil {
		t.Fatal(err)
	}

	if err := GrantForNewInvitee(db, 2); err != nil {
		t.Fatalf("发放失败: %v", err)
	}

	var inviter, invitee models.User
	db.First(&inviter, 1)
	db.First(&invitee, 2)
	if inviter.Balance != 20 || invitee.Balance != 25 {
		t.Fatalf("余额不符: 邀请人 %.2f(期望20) 被邀请人 %.2f(期望25)", inviter.Balance, invitee.Balance)
	}
	if inviter.TotalInviteReward != 20 || inviter.TotalInviteCount != 1 {
		t.Fatalf("邀请计数不符: reward=%.2f count=%d", inviter.TotalInviteReward, inviter.TotalInviteCount)
	}
	var got models.InviteRelation
	db.First(&got, rel.ID)
	if !got.InviterRewardGiven || !got.InviteeRewardGiven {
		t.Fatalf("发放标记未置位: %+v", got)
	}
	if got.InviterRewardAmount != 20 {
		t.Fatalf("关系上的金额应为 20，实际 %.2f", got.InviterRewardAmount)
	}

	// 幂等：再调用一次不应重复发放
	if err := GrantForNewInvitee(db, 2); err != nil {
		t.Fatal(err)
	}
	db.First(&inviter, 1)
	if inviter.Balance != 20 {
		t.Fatalf("重复发放导致余额翻倍: %.2f", inviter.Balance)
	}
}

// 有最低消费要求的码，注册时不发，付款后由订单路径发
func TestGrantForNewInviteeSkipsWhenMinOrderRequired(t *testing.T) {
	db := setupInviteDB(t)
	setCfg(t, db, KeyInviterReward, "20")
	setCfg(t, db, KeyInviteeReward, "20")
	mkUser(t, db, 1, 0)
	mkUser(t, db, 2, 0)
	code := mkCode(t, db, 1, 0, 0, 100)
	rel := models.InviteRelation{InviteCodeID: code.ID, InviterID: 1, InviteeID: 2}
	db.Create(&rel)

	if err := GrantForNewInvitee(db, 2); err != nil {
		t.Fatal(err)
	}
	var inviter models.User
	db.First(&inviter, 1)
	if inviter.Balance != 0 {
		t.Fatalf("有最低消费要求时注册阶段不应发放，实际余额 %.2f", inviter.Balance)
	}
}

// 补发：历史漏发关系应被补上，且金额按当前配置计算；已发放的跳过（幂等）
func TestBackfillGrantsMissedRewards(t *testing.T) {
	db := setupInviteDB(t)
	setCfg(t, db, KeyInviterReward, "20")
	setCfg(t, db, KeyInviteeReward, "20")
	mkUser(t, db, 1, 0)
	mkUser(t, db, 2, 0)
	mkUser(t, db, 3, 0)
	code := mkCode(t, db, 1, 0, 0, 0)
	// 关系 A：已付款但奖励为 0（历史漏发）
	db.Create(&models.InviteRelation{InviteCodeID: code.ID, InviterID: 1, InviteeID: 2,
		InviterRewardAmount: 0, InviteeRewardAmount: 0})
	// 关系 B：未付款（registration 规则下也应补）
	db.Create(&models.InviteRelation{InviteCodeID: code.ID, InviterID: 1, InviteeID: 3,
		InviterRewardAmount: 0, InviteeRewardAmount: 0})

	// 演练不应写库
	if res, err := Backfill(db, BackfillOptions{DryRun: true, Rule: RuleRegistration}); err != nil {
		t.Fatal(err)
	} else {
		if res.GrantedInviter != 2 || res.GrantedInvitee != 2 {
			t.Fatalf("演练统计不符: %+v", res)
		}
		var u models.User
		db.First(&u, 1)
		if u.Balance != 0 {
			t.Fatalf("演练模式不应写库，实际余额 %.2f", u.Balance)
		}
	}

	res, err := Backfill(db, BackfillOptions{Rule: RuleRegistration})
	if err != nil {
		t.Fatalf("补发失败: %v", err)
	}
	if res.InviterTotal != 40 || res.InviteeTotal != 40 {
		t.Fatalf("补发金额不符: 邀请人 %.2f 被邀请人 %.2f", res.InviterTotal, res.InviteeTotal)
	}
	var inviter models.User
	db.First(&inviter, 1)
	if inviter.Balance != 40 || inviter.TotalInviteCount != 2 {
		t.Fatalf("邀请人余额/计数不符: %.2f / %d", inviter.Balance, inviter.TotalInviteCount)
	}

	// 再跑一次：全部已发放，不应重复
	res2, err := Backfill(db, BackfillOptions{Rule: RuleRegistration})
	if err != nil {
		t.Fatal(err)
	}
	if res2.GrantedInviter != 0 || res2.GrantedInvitee != 0 {
		t.Fatalf("重复补发应无操作: %+v", res2)
	}
}

// paid 规则只补已付款的关系
func TestBackfillPaidRuleOnlyTouchesPaidRelations(t *testing.T) {
	db := setupInviteDB(t)
	setCfg(t, db, KeyInviterReward, "20")
	setCfg(t, db, KeyInviteeReward, "20")
	mkUser(t, db, 1, 0)
	mkUser(t, db, 2, 0)
	mkUser(t, db, 3, 0)
	code := mkCode(t, db, 1, 0, 0, 0)
	db.Create(&models.InviteRelation{InviteCodeID: code.ID, InviterID: 1, InviteeID: 2,
		InviteeFirstOrderID: sqlNullInt(750)})
	db.Create(&models.InviteRelation{InviteCodeID: code.ID, InviterID: 1, InviteeID: 3})

	res, err := Backfill(db, BackfillOptions{Rule: RulePaid})
	if err != nil {
		t.Fatal(err)
	}
	if res.GrantedInviter != 1 || res.InviterTotal != 20 {
		t.Fatalf("paid 规则应只补 1 条，实际 %+v", res)
	}
}

// 记录查询：应返回明细与汇总，且字段名兼容用户端历史表格
func TestListRecordsShape(t *testing.T) {
	db := setupInviteDB(t)
	mkUser(t, db, 1, 0)
	mkUser(t, db, 2, 0)
	code := mkCode(t, db, 1, 0, 0, 0)
	db.Create(&models.InviteRelation{InviteCodeID: code.ID, InviterID: 1, InviteeID: 2,
		InviterRewardAmount: 20, InviterRewardGiven: true, InviteeTotalConsumption: 198})

	records, summary, err := ListRecords(db, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("期望 1 条记录，实际 %d", len(records))
	}
	r := records[0]
	if r.InviteeID != 2 || r.InviterRewardAmount != 20 || !r.RewardGiven || r.TotalConsumption != 198 {
		t.Fatalf("记录内容不符: %+v", r)
	}
	if r.StatusText == "" {
		t.Fatal("应给出可读的状态文案")
	}
	if summary.Registered != 1 || summary.TotalReward != 20 || summary.TotalConsumption != 198 {
		t.Fatalf("汇总不符: %+v", summary)
	}
}

func sqlNullInt(v int64) sql.NullInt64 { return sql.NullInt64{Int64: v, Valid: true} }
