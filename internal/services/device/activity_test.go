package device

import (
	"testing"
	"time"

	"cboard-go/internal/models"
)

func mkActDevice(lastActive time.Time) models.Device {
	d := models.Device{LastAccess: lastActive}
	t := lastActive
	d.LastSeen = &t
	return d
}

func TestActivityOfThresholds(t *testing.T) {
	now := time.Now()

	cases := []struct {
		name   string
		ago    time.Duration
		expect string
	}{
		{"刚拉过（1 分钟前）", time.Minute, StateActiveDevice},
		{"59 分钟前", 59 * time.Minute, StateActiveDevice},
		{"61 分钟前 → 近期用过", 61 * time.Minute, StateRecentDevice},
		{"20 小时前（旧逻辑会误判为在线）", 20 * time.Hour, StateRecentDevice},
		{"25 小时前 → 离线", 25 * time.Hour, StateOfflineDevice},
		{"30 天前", 30 * 24 * time.Hour, StateOfflineDevice},
	}
	for _, c := range cases {
		got := ActivityOf(mkActDevice(now.Add(-c.ago)), now)
		if got.State != c.expect {
			t.Errorf("%s: 期望 %s，实际 %s", c.name, c.expect, got.State)
		}
		if got.StateLabel == "" || got.Basis == "" {
			t.Errorf("%s: 状态文案/判定依据不应为空: %+v", c.name, got)
		}
		if got.AgoSeconds <= 0 {
			t.Errorf("%s: 距上次活跃秒数应 > 0，实际 %d", c.name, got.AgoSeconds)
		}
	}
}

func TestActivityOfNeverFetched(t *testing.T) {
	// 从未拉取（零值时间）必须判为离线，且不能出现负数秒
	got := ActivityOf(models.Device{}, time.Now())
	if got.State != StateOfflineDevice {
		t.Fatalf("从未拉取应为离线，实际 %s", got.State)
	}
	if got.AgoSeconds != 0 {
		t.Fatalf("从未拉取时秒数应为 0，实际 %d", got.AgoSeconds)
	}
}

func TestActivityOfClockSkew(t *testing.T) {
	// 客户端/服务器时钟偏差导致「未来时间」时，秒数不能是负数
	now := time.Now()
	got := ActivityOf(mkActDevice(now.Add(10*time.Minute)), now)
	if got.AgoSeconds < 0 {
		t.Fatalf("时钟回拨保护失效，秒数 = %d", got.AgoSeconds)
	}
	if got.State != StateActiveDevice {
		t.Fatalf("未来时间应视为刚活跃，实际 %s", got.State)
	}
}

func TestCountByActivity(t *testing.T) {
	now := time.Now()
	list := []models.Device{
		mkActDevice(now.Add(-5 * time.Minute)), // active
		mkActDevice(now.Add(-3 * time.Hour)),   // recent
		mkActDevice(now.Add(-23 * time.Hour)),  // recent
		mkActDevice(now.Add(-48 * time.Hour)),  // offline
		{},                                     // offline（从未）
	}
	c := CountByActivity(list, now)
	if c.Active != 1 || c.Recent != 2 || c.Offline != 2 {
		t.Fatalf("统计异常: active=%d recent=%d offline=%d（期望 1/2/2）", c.Active, c.Recent, c.Offline)
	}
}

func TestLastSeenPreferredOverLastAccess(t *testing.T) {
	// 两个字段并存时以 LastSeen 为准（历史上只有它被单独更新过）
	old := time.Now().Add(-30 * time.Hour)
	recent := time.Now().Add(-2 * time.Minute)
	d := models.Device{LastAccess: old, LastSeen: &recent}
	if got := ActivityOf(d, time.Now()); got.State != StateActiveDevice {
		t.Fatalf("应以 LastSeen 为准判为活跃，实际 %s", got.State)
	}
}
