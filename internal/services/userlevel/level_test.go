package userlevel

import (
	"testing"

	"cboard-go/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.UserLevel{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	return db
}

func seedVIP(t *testing.T, db *gorm.DB) []models.UserLevel {
	t.Helper()
	levels := []models.UserLevel{
		{LevelName: "VIP1", LevelOrder: 1, MinConsumption: 200, DiscountRate: 0.99, Color: "#909399", IsActive: true},
		{LevelName: "VIP2", LevelOrder: 2, MinConsumption: 400, DiscountRate: 0.98, Color: "#67C23A", IsActive: true},
		{LevelName: "VIP3", LevelOrder: 3, MinConsumption: 600, DiscountRate: 0.97, Color: "#409EFF", IsActive: true},
	}
	for i := range levels {
		if err := db.Create(&levels[i]).Error; err != nil {
			t.Fatalf("写入等级失败: %v", err)
		}
	}
	return levels
}

func newUser(t *testing.T, db *gorm.DB, consumption float64) *models.User {
	t.Helper()
	u := models.User{Username: "u" + t.Name()[:min(3, len(t.Name()))], Email: "u@test.local", TotalConsumption: consumption}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	return &u
}

// 按累计消费命中最高达标等级；未达标返回 nil
func TestMatch(t *testing.T) {
	db := newTestDB(t)
	seedVIP(t, db)

	cases := []struct {
		consumption float64
		want        string
	}{
		{0, ""},
		{199.99, ""},
		{200, "VIP1"},
		{399.99, "VIP1"},
		{400, "VIP2"},
		{1000, "VIP3"},
	}
	for _, c := range cases {
		lvl, err := Match(db, c.consumption)
		if err != nil {
			t.Fatalf("Match(%v) 报错: %v", c.consumption, err)
		}
		got := ""
		if lvl != nil {
			got = lvl.LevelName
		}
		if got != c.want {
			t.Fatalf("消费 %.2f 命中 %q，期望 %q", c.consumption, got, c.want)
		}
	}
}

// 停用的等级不参与匹配
func TestMatch_skipsInactive(t *testing.T) {
	db := newTestDB(t)
	levels := seedVIP(t, db)
	db.Model(&models.UserLevel{}).Where("id IN ?", []uint{levels[1].ID, levels[2].ID}).Update("is_active", false)

	lvl, _ := Match(db, 1000)
	if lvl == nil || lvl.LevelName != "VIP1" {
		t.Fatalf("停用等级仍被匹配: %+v", lvl)
	}
}

// Sync 只升不降 + Plan 与 Sync 判定一致（演练/执行不会出现两种答案）
func TestSync_onlyUpNeverDown(t *testing.T) {
	db := newTestDB(t)
	levels := seedVIP(t, db)
	u := newUser(t, db, 700)

	lvl, changed, err := Sync(db, u)
	if err != nil || !changed || lvl == nil || lvl.LevelName != "VIP3" {
		t.Fatalf("首次同步应升到 VIP3: lvl=%v changed=%v err=%v", lvl, changed, err)
	}

	// 消费被下调（退款）：等级不回退
	u.TotalConsumption = 250
	if err := db.Save(u).Error; err != nil {
		t.Fatal(err)
	}
	planLevel, planChanged, err := Plan(db, u)
	if err != nil {
		t.Fatal(err)
	}
	if planChanged {
		t.Fatalf("只升不降被破坏：应保持 VIP3，实际计划改为 %v", planLevel.LevelName)
	}
	if _, changed, _ = Sync(db, u); changed {
		t.Fatal("Sync 不应降级")
	}
	var reloaded models.User
	db.First(&reloaded, u.ID)
	if !reloaded.UserLevelID.Valid || reloaded.UserLevelID.Int64 != int64(levels[2].ID) {
		t.Fatalf("等级被降级: %+v", reloaded.UserLevelID)
	}
}

// 生效等级：停用后 ForUser 返回 nil、折扣不再生效
func TestForUserDisabledLevelNotEffective(t *testing.T) {
	db := newTestDB(t)
	levels := seedVIP(t, db)
	u := newUser(t, db, 500)
	if _, _, err := Sync(db, u); err != nil {
		t.Fatal(err)
	}

	lvl, err := ForUser(db, u.ID)
	if err != nil || lvl == nil || lvl.LevelName != "VIP2" {
		t.Fatalf("生效等级应为 VIP2: %+v %v", lvl, err)
	}
	discount, payable := ApplyDiscount(db, u.ID, 100)
	if discount != 2 || payable != 98 {
		t.Fatalf("VIP2 2%% 折扣计算错误: 优惠=%v 应付=%v", discount, payable)
	}

	// 停用该等级：立即失效
	db.Model(&models.UserLevel{}).Where("id = ?", levels[1].ID).Update("is_active", false)
	if lvl, _ := ForUser(db, u.ID); lvl != nil {
		t.Fatalf("停用后不应再有生效等级: %+v", lvl)
	}
	if discount, payable := ApplyDiscount(db, u.ID, 100); discount != 0 || payable != 100 {
		t.Fatalf("停用后不应再打折: 优惠=%v 应付=%v", discount, payable)
	}
}

// 折扣边界：0 / 1 / >1 一律按无折扣
func TestApplyDiscount_boundaries(t *testing.T) {
	db := newTestDB(t)
	u := newUser(t, db, 0)

	for _, rate := range []float64{0, 1.0, 1.2} {
		lvl := models.UserLevel{LevelName: "X", LevelOrder: int(rate*100) + 1, MinConsumption: 0, DiscountRate: rate, IsActive: true}
		db.Create(&lvl)
		db.Model(u).Update("user_level_id", lvl.ID)
		if discount, payable := ApplyDiscount(db, u.ID, 100); discount != 0 || payable != 100 {
			t.Fatalf("折扣率 %.2f 应视为无折扣: 优惠=%v 应付=%v", rate, discount, payable)
		}
	}
}

// NextLevel：用于前台"再消费 X 元升级"的提示
func TestNextLevel(t *testing.T) {
	db := newTestDB(t)
	levels := seedVIP(t, db)

	if next := NextLevel(db, nil); next == nil || next.LevelName != "VIP1" {
		t.Fatalf("无等级时应指向 VIP1: %+v", next)
	}
	if next := NextLevel(db, &levels[1]); next == nil || next.LevelName != "VIP3" {
		t.Fatalf("VIP2 的下一档应为 VIP3: %+v", next)
	}
	if next := NextLevel(db, &levels[2]); next != nil {
		t.Fatalf("最高档不应再有下一档: %+v", next)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
