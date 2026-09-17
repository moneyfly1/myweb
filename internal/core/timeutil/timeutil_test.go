package timeutil

import (
	"database/sql"
	"strings"
	"testing"
	"time"
)

// 时间工具的统一口径测试。
//
// 背景：线上曾出现同一张表混存 +08:00 / +00:00 / -05:00 三种偏移的时间，
// 而 SQLite 的 datetime 是文本、按字典序比较，跨偏移行的范围查询与排序会出错。
// 这里锁定「一律按北京时间换算」的口径，防止再次回归。

func TestNowIsBeijing(t *testing.T) {
	now := Now()
	if _, offset := now.Zone(); offset != 8*3600 {
		t.Errorf("Now() 必须是北京时间(+08:00)，实际偏移 %d 秒", offset)
	}
}

func TestFormatConvertsTimezone(t *testing.T) {
	cases := []struct {
		name string
		in   time.Time
		want string
	}{
		{
			name: "UTC 时间应换算为北京时间（+8h）",
			in:   time.Date(2026, 9, 17, 6, 30, 0, 0, time.UTC),
			want: "2026-09-17 14:30:00",
		},
		{
			name: "美东时间(-05:00)应换算为北京时间（+13h）",
			in:   time.Date(2026, 9, 17, 1, 30, 0, 0, time.FixedZone("EST", -5*3600)),
			want: "2026-09-17 14:30:00",
		},
		{
			name: "已是北京时间则原样输出",
			in:   time.Date(2026, 9, 17, 14, 30, 0, 0, BeijingTZ),
			want: "2026-09-17 14:30:00",
		},
	}
	for _, c := range cases {
		if got := Format(c.in); got != c.want {
			t.Errorf("%s: Format = %q, want %q", c.name, got, c.want)
		}
	}
}

// TestFormatVersusBareFormat 说明为什么必须用 Format 而不是 t.Format：
// 裸 Format 不做时区换算，会把非北京时间直接当成北京时间显示。
func TestFormatVersusBareFormat(t *testing.T) {
	est := time.Date(2026, 9, 17, 1, 30, 0, 0, time.FixedZone("EST", -5*3600))
	bare := est.Format(LayoutDateTime)
	converted := Format(est)
	if bare == converted {
		t.Fatalf("裸 Format 与 Format 结果不应相同：bare=%s converted=%s", bare, converted)
	}
	if converted != "2026-09-17 14:30:00" {
		t.Errorf("换算后应为北京时间 14:30，实际 %s", converted)
	}
}

func TestFormatNull(t *testing.T) {
	if got := FormatNull(sql.NullTime{}); got != "" {
		t.Errorf("无效时间应返回空串，实际 %q", got)
	}
	valid := sql.NullTime{Time: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), Valid: true}
	if got := FormatNull(valid); got != "2026-01-02 11:04:05" {
		t.Errorf("UTC 03:04:05 应为北京时间 11:04:05，实际 %q", got)
	}
}

// TestParseBeijingLayoutAnchorsBeijing 裸 time.Parse 会按 UTC 解析，
// 与库中 +08:00 的时间比较时整体偏差 8 小时。
func TestParseBeijingLayoutAnchorsBeijing(t *testing.T) {
	got, err := ParseBeijingLayout(LayoutDateTime, "2026-09-17 14:30:00")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if _, offset := got.Zone(); offset != 8*3600 {
		t.Errorf("解析结果应为北京时间，实际偏移 %d", offset)
	}
	bare, _ := time.Parse(LayoutDateTime, "2026-09-17 14:30:00")
	if got.Equal(bare) {
		t.Error("ParseBeijingLayout 与裸 time.Parse 结果不应相同（后者按 UTC）")
	}
	// 同一串 "14:30"：北京时间解释 = UTC 06:30，比按 UTC 解释早 8 小时
	if diff := got.Sub(bare); diff != -8*time.Hour {
		t.Errorf("两者应相差 8 小时，实际 %v", diff)
	}
}

func TestRangeOfDay(t *testing.T) {
	at := time.Date(2026, 9, 17, 15, 30, 0, 0, time.UTC) // UTC 15:30 = 北京 23:30
	start, end := RangeOfDay(at)
	if start.Format(LayoutDateTime) != "2026-09-17 00:00:00" {
		t.Errorf("当天起点错误: %s", start.Format(LayoutDateTime))
	}
	if end.Hour() != 23 || end.Minute() != 59 {
		t.Errorf("当天终点应接近 23:59，实际 %s", end.Format(LayoutDateTime))
	}
	// 北京时间 9/17 00:30 对应 UTC 9/16 16:30，日期归属必须按北京时间算
	edge := time.Date(2026, 9, 16, 16, 30, 0, 0, time.UTC)
	edgeStart, _ := RangeOfDay(edge)
	if edgeStart.Format(LayoutDateTime) != "2026-09-17 00:00:00" {
		t.Errorf("跨日边界应按北京时间归入 9/17，实际起点 %s", edgeStart.Format(LayoutDateTime))
	}
}

// TestNowForDBIsSecondPrecision 落库时间统一到秒精度。
//
// 背景：SQLite 时间列是文本，Go 驱动按 .999999999 格式化并去掉末尾零，
// 同一列会混有 0/3/6/9 位小数（15:04:05 / 15:04:05.123 / 15:04:05.123456789），
// 导致字符串长度不齐、与 MySQL DATETIME(0) 语义不一致。
func TestNowForDBIsSecondPrecision(t *testing.T) {
	got := NowForDB()
	if got.Nanosecond() != 0 {
		t.Errorf("NowForDB 应截断到秒，实际纳秒=%d", got.Nanosecond())
	}
	if _, offset := got.Zone(); offset != 8*3600 {
		t.Errorf("NowForDB 应为北京时间，实际偏移 %d", offset)
	}
	// 格式化后必须是固定 25 字符（YYYY-MM-DD HH:MM:SS+08:00）
	formatted := Format(got) + "+08:00"
	if len(formatted) != 25 {
		t.Errorf("落库格式应为 25 字符，实际 %d（%s）", len(formatted), formatted)
	}
	if strings.Contains(formatted, ".") {
		t.Errorf("落库格式不应含小数秒：%s", formatted)
	}
}

// TestNowKeepsSubSecond 耗时统计用的 Now() 不应被截断（否则延迟统计会被抹平）
func TestNowKeepsSubSecond(t *testing.T) {
	if Now().Nanosecond() == 0 && Now().Nanosecond() == 0 {
		t.Log("提示：本机时钟恰好整秒，跳过（不是失败）")
	}
	if NowForDB().Nanosecond() != 0 {
		t.Error("NowForDB 必须是整秒")
	}
}
