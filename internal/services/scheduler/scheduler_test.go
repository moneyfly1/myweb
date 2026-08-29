package scheduler

import (
	"testing"
	"time"

	"cboard-go/internal/models"
	"cboard-go/internal/utils"
)

func TestGroupExpiringSubscriptions(t *testing.T) {
	dayStart := time.Date(2026, 1, 10, 0, 0, 0, 0, utils.BeijingTZ)

	at := func(days int) time.Time {
		return dayStart.Add(time.Duration(days) * 24 * time.Hour)
	}

	subs := []models.Subscription{
		{ID: 1, UserID: 1, User: models.User{ID: 1}, ExpireTime: at(0)}, // 今天到期 → 0 天组
		{ID: 2, UserID: 1, User: models.User{ID: 1}, ExpireTime: at(1)}, // 明天到期 → 1 天组
		{ID: 3, UserID: 1, User: models.User{ID: 1}, ExpireTime: at(3)}, // 3 天后 → 3 天组
		{ID: 4, UserID: 1, User: models.User{ID: 1}, ExpireTime: at(7)}, // 7 天后 → 7 天组
		{ID: 5, UserID: 1, User: models.User{ID: 1}, ExpireTime: at(2)}, // 2 天后 → 不匹配任何组
		{ID: 6, UserID: 0, User: models.User{ID: 0}, ExpireTime: at(0)}, // 无用户 → 跳过
		{ID: 7, UserID: 1, User: models.User{ID: 1}, ExpireTime: at(8)}, // 8 天后 → 超出窗口
	}

	grouped := groupExpiringSubscriptions(subs, dayStart)

	want := map[int][]uint{
		0: {1},
		1: {2},
		3: {3},
		7: {4},
	}

	for group, ids := range want {
		got := grouped[group]
		if len(got) != len(ids) {
			t.Fatalf("group %d: 期望 %d 条, 实际 %d 条", group, len(ids), len(got))
		}
		for i, id := range ids {
			if got[i].ID != id {
				t.Errorf("group %d[%d]: 期望订阅 ID %d, 实际 %d", group, i, id, got[i].ID)
			}
		}
	}

	// 不匹配的订阅（2/8 天、无用户）不应出现在任何组
	for group, list := range grouped {
		for _, s := range list {
			switch s.ID {
			case 1, 2, 3, 4:
				// 合法
			default:
				t.Errorf("group %d 含不应出现的订阅 ID %d", group, s.ID)
			}
		}
	}

	// 空输入
	empty := groupExpiringSubscriptions(nil, dayStart)
	for group, list := range empty {
		if len(list) != 0 {
			t.Errorf("空输入 group %d 应为空, 实际 %d 条", group, len(list))
		}
	}
}
