package handlers

import (
	"testing"
	"time"

	"cboard-go/internal/models"
)

// 刷新令牌「轮换重放」判定（用户反馈：勾了自动登录还是被退出）
//
// 生产现象（2026-10-20，用户 454487210 / macOS 2.2.31）：每 60 秒一次
//
//	POST /api/v1/auth/refresh 401 → POST /api/v1/auth/logout 401
//
// 直到用户手动重新登录。根因是「刷新响应丢了」：服务端已经把旧 refresh 拉黑，
// 而内存宽限期只有 60 秒且服务重启（部署）就清空，客户端过了窗口再来必然 401，
// 客户端按 rejected 语义登出 —— 用户看到的就是「无故退出」。
//
// 修复：按黑名单记录的「原因 + 时间」做持久化豁免（rotate 且 30 分钟内 → 重放），
// 登出/吊销不豁免。本测试锁住判定口径。
func TestCanReplayRotatedToken(t *testing.T) {
	now := time.Now()

	cases := []struct {
		name  string
		entry models.TokenBlacklist
		want  bool
	}{
		{
			name:  "轮换后 10 秒重放（客户端响应丢失）→ 放行",
			entry: models.TokenBlacklist{Reason: models.BlacklistReasonRotate, CreatedAt: now.Add(-10 * time.Second)},
			want:  true,
		},
		{
			name:  "轮换后 25 分钟（后台定时任务下一轮才重试）→ 仍放行",
			entry: models.TokenBlacklist{Reason: models.BlacklistReasonRotate, CreatedAt: now.Add(-25 * time.Minute)},
			want:  true,
		},
		{
			name:  "轮换后 31 分钟（超出豁免窗口）→ 拒绝",
			entry: models.TokenBlacklist{Reason: models.BlacklistReasonRotate, CreatedAt: now.Add(-31 * time.Minute)},
			want:  false,
		},
		{
			name:  "用户显式登出的令牌 → 一律拒绝（登出必须立刻生效）",
			entry: models.TokenBlacklist{Reason: models.BlacklistReasonLogout, CreatedAt: now.Add(-2 * time.Second)},
			want:  false,
		},
		{
			name:  "管理员吊销的令牌 → 一律拒绝",
			entry: models.TokenBlacklist{Reason: models.BlacklistReasonRevoke, CreatedAt: now.Add(-2 * time.Second)},
			want:  false,
		},
		{
			name:  "历史数据没有 reason 字段（空串）→ 按非轮换处理，拒绝",
			entry: models.TokenBlacklist{CreatedAt: now.Add(-5 * time.Second)},
			want:  false,
		},
	}

	for _, c := range cases {
		if got := canReplayRotatedToken(c.entry, now); got != c.want {
			t.Errorf("%s: 期望 %v，实际 %v", c.name, c.want, got)
		}
	}
}

// 豁免窗口必须显著大于内存宽限期（60 秒），否则部署重启后仍会掉线
func TestReplayWindowCoversDeployRestart(t *testing.T) {
	if refreshRotateReplayWindow <= refreshRotateGracePeriod {
		t.Fatalf("持久化豁免窗口（%s）应远大于内存宽限期（%s）",
			refreshRotateReplayWindow, refreshRotateGracePeriod)
	}
	if refreshRotateReplayWindow < 10*time.Minute {
		t.Fatalf("豁免窗口（%s）至少 10 分钟，才能覆盖客户端分钟级重试与部署重启",
			refreshRotateReplayWindow)
	}
}
