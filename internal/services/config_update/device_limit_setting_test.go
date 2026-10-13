package config_update

import (
	"strings"
	"testing"

	"cboard-go/internal/core/database"
	"cboard-go/internal/services/device"
)

// 设备上限的取值语义（用户澄清两件事）：
//
//	① 上限 0 = **拒绝**（不是"不限制"）：只下发错误节点，任何设备都拿不到可用节点；
//	② 上限非 0 时，允许数量**完全取后台设置**（订阅上的 device_limit），
//	   代码里不许出现固定值 —— 把上限从 3 改成 6，放行台数必须随之从 3 变到 6。
func TestDeviceLimitZeroMeansRefuse(t *testing.T) {
	db, sub, user := setupLimitTestDB(t, "")
	prevDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prevDB })

	// 后台把上限设成 0（= 拒绝）
	if err := db.Model(&sub).Update("device_limit", 0).Error; err != nil {
		t.Fatalf("更新上限失败: %v", err)
	}

	mfUA := "MoneyFly/2.2.25 (Windows NT 10.0; Win64; x64)"
	dm := device.NewDeviceManager()
	if _, err := dm.RecordDeviceAccessWithHeaders(sub.ID, user.ID, mfUA, "10.0.0.1", "clash",
		map[string]string{"X-MF-Device-Id": "dev-1"}); err != nil {
		t.Fatalf("记录设备失败: %v", err)
	}

	svc := &ConfigUpdateService{db: db}
	svc.refreshSystemConfig()

	// 上限 0：即便这台设备早就登记过，也只给错误节点（没有任何可用节点）
	ctx := svc.GetSubscriptionContext(sub.SubscriptionURL, "10.0.0.1", mfUA,
		map[string]string{"X-MF-Device-Id": "dev-1"})
	if ctx.Status != StatusDeviceOverLimit {
		t.Fatalf("上限 0 应判为拒绝（超限），实际状态 %v", ctx.Status)
	}
	names := namesOf(svc.buildSubscriptionNodes(ctx))
	for _, n := range names {
		if n == "可用节点A" {
			t.Fatalf("上限 0 时不得下发真实节点，实际: %v", names)
		}
	}
	// 必须带可见的错误原因（客户端据此提示「设备数量超限」）
	hasReason := false
	for _, n := range names {
		if strings.Contains(n, "设备数量超限") {
			hasReason = true
		}
	}
	if !hasReason {
		t.Fatalf("拒绝时必须带「设备数量超限」原因节点，实际: %v", names)
	}
	if ctx.DeviceLimit != 0 {
		t.Fatalf("上下文里的上限应回传订阅上的 0，实际 %d", ctx.DeviceLimit)
	}
}

// 上限完全跟随订阅值：3 → 第 4 台被拒；改成 6 → 第 5、6 台放行、第 7 台被拒。
// 这条测试就是用「同一套代码换数字」来证明没有写死的 5。
func TestDeviceLimitFollowsSubscriptionValue(t *testing.T) {
	db, sub, user := setupLimitTestDB(t, "")
	prevDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prevDB })

	mfUA := "MoneyFly/2.2.25 (Windows NT 10.0; Win64; x64)"
	dm := device.NewDeviceManager()
	svc := &ConfigUpdateService{db: db}
	svc.refreshSystemConfig()

	use := func(id string) SubscriptionStatus {
		ctx := svc.GetSubscriptionContext(sub.SubscriptionURL, "10.0.0.9", mfUA,
			map[string]string{"X-MF-Device-Id": id})
		if ctx.Status == StatusNormal {
			// 放行的设备会被登记（真实路径里由 handler 负责登记，这里显式补上）
			if _, err := dm.RecordDeviceAccessWithHeaders(sub.ID, user.ID, mfUA, "10.0.0.9", "clash",
				map[string]string{"X-MF-Device-Id": id}); err != nil {
				t.Fatalf("记录设备 %s 失败: %v", id, err)
			}
		}
		return ctx.Status
	}

	// 上限 3：第 1~3 台放行，第 4 台拒绝
	if err := db.Model(&sub).Update("device_limit", 3).Error; err != nil {
		t.Fatalf("更新上限失败: %v", err)
	}
	for _, id := range []string{"a1", "a2", "a3"} {
		if st := use(id); st != StatusNormal {
			t.Fatalf("上限 3 时 %s 应放行，实际 %v", id, st)
		}
	}
	if st := use("a4"); st != StatusDeviceOverLimit {
		t.Fatalf("上限 3 时第 4 台应被拒，实际 %v", st)
	}

	// 后台把同一个订阅改成 6：放行台数必须跟着变
	if err := db.Model(&sub).Update("device_limit", 6).Error; err != nil {
		t.Fatalf("更新上限失败: %v", err)
	}
	for _, id := range []string{"a4", "a5", "a6"} {
		if st := use(id); st != StatusNormal {
			t.Fatalf("上限改成 6 后 %s 应放行，实际 %v", id, st)
		}
	}
	if st := use("a7"); st != StatusDeviceOverLimit {
		t.Fatalf("上限 6 时第 7 台应被拒，实际 %v", st)
	}
}
