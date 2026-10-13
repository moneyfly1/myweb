package device

import (
	"testing"
	"time"

	"cboard-go/internal/models"
)

// mkActDevice 造一台设备：距离上次回拉 ago，首次出现 first，累计拉取 count 次
func mkActDevice(now time.Time, ago, first time.Duration, count int) models.Device {
	last := now.Add(-ago)
	fs := now.Add(-first)
	return models.Device{LastAccess: last, LastSeen: &last, FirstSeen: &fs, AccessCount: count, CreatedAt: fs}
}

// 固定「现在」，避免测试对真实时钟敏感
func fixedNow() time.Time { return time.Date(2026, 10, 13, 15, 0, 0, 0, time.UTC) }

func TestActivityAdaptsToDeviceRhythm(t *testing.T) {
	now := fixedNow()

	// 生产实测场景（老站真实数据）：设备 8611 首次出现 55 天前、累计回拉 281 次
	// → 平均间隔约 4.7 小时，最近一次在 2.4 小时前 → 应为「活跃」
	// （旧实现用固定 1 小时窗口，会把这类设备判成"近期用过"，用户看到"活跃设备 0 台"）
	d := mkActDevice(now, 144*time.Minute, 55*24*time.Hour, 281)
	got := ActivityOf(d, now)
	if got.State != StateActiveDevice {
		t.Fatalf("按设备自身节奏应判为活跃，实际 %s（阈值 %ds 间隔 %ds）", got.State, got.ThresholdSec, got.AvgIntervalSec)
	}
	if got.AvgIntervalSec < 4*3600 || got.AvgIntervalSec > 6*3600 {
		t.Fatalf("平均间隔应约 4.7 小时，实际 %d 秒", got.AvgIntervalSec)
	}

	// 同一台设备：上次回拉在 30 小时前 → 离线
	if st := ActivityOf(mkActDevice(now, 30*time.Hour, 55*24*time.Hour, 281), now); st.State != StateOfflineDevice {
		t.Fatalf("30 小时前应离线，实际 %s", st.State)
	}

	// 同一台设备：上次回拉 20 小时前（阈值约 9.4 小时）→ 近期用过
	if st := ActivityOf(mkActDevice(now, 20*time.Hour, 55*24*time.Hour, 281), now); st.State != StateRecentDevice {
		t.Fatalf("20 小时前应近期用过，实际 %s", st.State)
	}
}

func TestActivityThresholdGuards(t *testing.T) {
	now := fixedNow()

	// 新设备（样本少）→ 阈值取下限 1 小时：10 分钟前活跃
	d := mkActDevice(now, 10*time.Minute, 20*time.Minute, 2)
	if got := ActivityOf(d, now); got.State != StateActiveDevice || got.ThresholdSec != 3600 {
		t.Fatalf("新设备应取 1 小时下限并判为活跃，实际 state=%s threshold=%d", got.State, got.ThresholdSec)
	}

	// 一年只拉过两次（平均间隔极大）→ 阈值被上限 24 小时钳制；一年前的回拉仍应为离线
	d2 := mkActDevice(now, 300*24*time.Hour, 360*24*time.Hour, 2)
	if got := ActivityOf(d2, now); got.State != StateOfflineDevice || got.ThresholdSec != int64((24*time.Hour).Seconds()) {
		t.Fatalf("阈值应被钳到 24 小时且判离线，实际 state=%s threshold=%d", got.State, got.ThresholdSec)
	}
}

func TestActivityNeverFetchedAndClockSkew(t *testing.T) {
	now := fixedNow()
	if got := ActivityOf(models.Device{}, now); got.State != StateOfflineDevice || got.AgoSeconds != 0 {
		t.Fatalf("从未拉取应离线且秒数为 0，实际 %+v", got)
	}
	// 时钟偏差（未来时间）不能出现负数秒
	future := now.Add(10 * time.Minute)
	d := models.Device{LastAccess: future, LastSeen: &future}
	if got := ActivityOf(d, now); got.AgoSeconds < 0 {
		t.Fatalf("时钟回拨保护失效: %d", got.AgoSeconds)
	}
}

func TestCountByActivity(t *testing.T) {
	now := fixedNow()
	list := []models.Device{
		mkActDevice(now, 5*time.Minute, 30*24*time.Hour, 200),   // 活跃（间隔约 3.6h → 阈值 7.2h）
		mkActDevice(now, 20*time.Hour, 30*24*time.Hour, 200),    // 近期用过
		mkActDevice(now, 30*24*time.Hour, 60*24*time.Hour, 100), // 离线
		{}, // 从未拉取 → 离线
	}
	c := CountByActivity(list, now)
	if c.Active != 1 || c.Recent != 1 || c.Offline != 2 {
		t.Fatalf("统计异常: active=%d recent=%d offline=%d（期望 1/1/2）", c.Active, c.Recent, c.Offline)
	}
}

func TestLastSeenPreferredOverLastAccess(t *testing.T) {
	now := fixedNow()
	old := now.Add(-30 * time.Hour)
	recent := now.Add(-2 * time.Minute)
	fs := now.Add(-10 * 24 * time.Hour)
	d := models.Device{LastAccess: old, LastSeen: &recent, FirstSeen: &fs, AccessCount: 50, CreatedAt: fs}
	if got := ActivityOf(d, now); got.State != StateActiveDevice {
		t.Fatalf("应以 LastSeen 为准判为活跃，实际 %s", got.State)
	}
}

func TestBasisIsHumanReadable(t *testing.T) {
	now := fixedNow()
	got := ActivityOf(mkActDevice(now, 144*time.Minute, 55*24*time.Hour, 281), now)
	if got.Basis == "" || got.StateLabel == "" {
		t.Fatalf("判定依据/文案不应为空: %+v", got)
	}
	t.Logf("活跃设备判定依据: %s", got.Basis)
}
