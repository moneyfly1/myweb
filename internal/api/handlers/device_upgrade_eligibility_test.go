package handlers

import (
	"strings"
	"testing"
	"time"

	"cboard-go/internal/models"
)

// TestCheckDeviceUpgradeEligibility 设备升级准入规则：
// 只有"已开通套餐（设备数 > 0）且未到期"的用户才能升级设备数量。
//
// 背景：新注册用户拿到的是默认订阅（设备数 0、当天 23:59:59 到期），
// 它并没有已开通的套餐；此前这类用户可以下单升级设备（线上真实案例：
// 用户 634584959 注册 5 分钟后花 0.06 元把设备数从 0 升到 1），
// 因此补上"已开通套餐 + 未到期"两道门。
func TestCheckDeviceUpgradeEligibility(t *testing.T) {
	now := time.Date(2026, 9, 18, 10, 30, 0, 0, time.UTC)

	cases := []struct {
		name         string
		subscription *models.Subscription
		wantEmpty    bool   // true = 允许
		wantContains string // 不允许时提示里应包含的关键词
	}{
		{
			name:         "已开通套餐且未到期 → 允许",
			subscription: &models.Subscription{DeviceLimit: 5, ExpireTime: now.AddDate(0, 0, 30)},
			wantEmpty:    true,
		},
		{
			name:         "默认试用：设备数 0（未开通套餐）→ 拒绝",
			subscription: &models.Subscription{DeviceLimit: 0, ExpireTime: now.Add(13 * time.Hour)},
			wantContains: "尚未开通套餐",
		},
		{
			name:         "设备数为 0 且已到期 → 仍按未开通套餐拒绝",
			subscription: &models.Subscription{DeviceLimit: 0, ExpireTime: now.Add(-time.Hour)},
			wantContains: "尚未开通套餐",
		},
		{
			name:         "已开通套餐但已到期 → 拒绝（先续费再升级）",
			subscription: &models.Subscription{DeviceLimit: 5, ExpireTime: now.Add(-time.Minute)},
			wantContains: "订阅已到期",
		},
		{
			name:         "到期时刻正好等于当前时间 → 视为已到期",
			subscription: &models.Subscription{DeviceLimit: 5, ExpireTime: now},
			wantContains: "订阅已到期",
		},
		{
			name:         "没有订阅 → 拒绝",
			subscription: nil,
			wantContains: "订阅不存在",
		},
	}

	for _, tc := range cases {
		msg := checkDeviceUpgradeEligibility(tc.subscription, now)
		if tc.wantEmpty {
			if msg != "" {
				t.Errorf("%s：应允许，实际被拒: %s", tc.name, msg)
			}
			continue
		}
		if msg == "" {
			t.Errorf("%s：应拒绝，实际放行", tc.name)
			continue
		}
		if !strings.Contains(msg, tc.wantContains) {
			t.Errorf("%s：提示应包含 %q，实际 %q", tc.name, tc.wantContains, msg)
		}
	}
}
