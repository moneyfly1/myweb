package handlers

import (
	"testing"
	"time"

	"cboard-go/internal/utils"
)

// 注册时间筛选：标准 YYYY-MM-DD 与历史 ISO 字符串都要解析成「北京时间的当天 00:00」，
// 否则区间筛选会差一天或被忽略（前端曾把 9/1 发成 2026-08-31T16:00:00.000Z）。
func TestParseUserDateFilter(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string // 期望的北京时间日期，空表示零值
	}{
		{"标准日期", "2026-09-01", "2026-09-01"},
		{"标准日期带空白", "  2026-09-30  ", "2026-09-30"},
		{"ISO(UTC)：北京时间 9/1 00:00 → 8/31T16:00Z", "2026-08-31T16:00:00.000Z", "2026-09-01"},
		{"ISO(UTC) 带毫秒 3 位", "2026-09-29T16:00:00.000Z", "2026-09-30"},
		{"RFC3339 带 +08:00", "2026-09-01T00:00:00+08:00", "2026-09-01"},
		{"RFC3339 带 Z（无毫秒）", "2026-08-31T16:00:00Z", "2026-09-01"},
		{"空字符串", "", ""},
		{"只有空白", "   ", ""},
		{"垃圾输入不报错", "not-a-date", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseUserDateFilter(tc.raw)
			if tc.want == "" {
				if !got.IsZero() {
					t.Fatalf("期望零值，实际 %v", got)
				}
				return
			}
			if got.IsZero() {
				t.Fatalf("期望解析出 %s，实际零值", tc.want)
			}
			if got.Location() != utils.BeijingTZ {
				t.Fatalf("期望北京时间时区，实际 %v", got.Location())
			}
			if got.Format("2006-01-02") != tc.want {
				t.Fatalf("期望日期 %s，实际 %s", tc.want, got.Format("2006-01-02"))
			}
			if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 {
				t.Fatalf("期望当天 00:00:00，实际 %v", got)
			}
		})
	}
}

// 区间含首尾两天：end_date 当天 23:59 的记录必须命中，
// 实现方式是把上界做成 end+1 天且用 <（半开区间）。
func TestUserDateFilterEndIsInclusive(t *testing.T) {
	end := parseUserDateFilter("2026-09-30")
	if end.IsZero() {
		t.Fatal("解析失败")
	}
	upper := end.AddDate(0, 0, 1)

	lastMoment := time.Date(2026, 9, 30, 23, 59, 59, 0, utils.BeijingTZ)
	if !lastMoment.Before(upper) {
		t.Fatalf("end_date 当天 23:59:59 应小于上界 %v，否则最后一天被漏掉", upper)
	}
	nextDay := time.Date(2026, 10, 1, 0, 0, 0, 0, utils.BeijingTZ)
	if nextDay.Before(upper) {
		t.Fatalf("次日 00:00 不应命中区间（上界 %v）", upper)
	}
}
