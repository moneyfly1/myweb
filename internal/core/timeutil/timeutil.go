// Package timeutil 是全站唯一的「时间/时区」公共实现（叶子包，不 import 任何 internal 包）。
//
// 为什么单独建包：internal/utils 已经 import internal/core/database 与
// internal/services/geoip，因此 database / geoip 反向 import utils 会形成循环依赖。
// 这正是历史上时区对象被就地内联、出现第 2/3 份 time.LoadLocation 的原因
// （见 internal/core/database/database.go、internal/services/scheduler/scheduler.go）。
//
// 使用约定（务必遵守，否则会再次产生跨时区混写）：
//   - 需要「当前时间」写入数据库或参与比较 → 一律用 timeutil.Now()，不要用 time.Now()
//   - 需要把时间转成北京时间展示/落库 → 一律用 timeutil.Format / timeutil.ToBeijing
//   - 需要解析前端传来的时间字符串 → 一律用 timeutil.ParseBeijing / ParseBeijingLayout，
//     不要用裸 time.Parse（它会按 UTC 解析，与库里的 +08:00 相差 8 小时）
//   - 「今天/本月」区间 → 用 timeutil.DayRange / MonthRange，不要手写 0 点计算
package timeutil

import (
	"database/sql"
	"time"
)

// BeijingTZ 全站唯一的北京时区对象，带固定偏移兜底（容器内缺 tzdata 时仍可用）
var BeijingTZ = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Shanghai"); err == nil && loc != nil {
		return loc
	}
	return time.FixedZone("CST", 8*3600)
}()

// 时间格式常量（全站唯一来源，避免各处硬编码 layout 字符串）
const (
	LayoutDateTime = "2006-01-02 15:04:05"
	LayoutDate     = "2006-01-02"
	LayoutCompact  = "20060102"
	LayoutMonth    = "200601"
	LayoutClock    = "15:04:05"
)

// Now 返回当前北京时间。写库/比较一律用这个函数。
func Now() time.Time {
	return time.Now().In(BeijingTZ)
}

// NowForDB 返回"落库用"的当前北京时间：截断到秒。
//
// 为什么要截断：SQLite 时间列是文本，Go 驱动按 `.999999999` 格式化并去掉末尾零，
// 于是同一列里会混有 0/3/6/9 位小数（如 15:04:05、15:04:05.123、15:04:05.123456789），
// 造成：① 字符串长度不齐、外部工具按固定位置解析会错；② 与 MySQL DATETIME(0) 语义不一致。
// 统一截断到秒后，全库时间形如 "2006-01-02 15:04:05+08:00"（固定 25 字符），
// 与展示格式 utils.FormatBeijingTime 一致，也便于跨数据库保持一致。
//
// 注意：耗时/延迟统计请用 time.Now()（带单调时钟），不要用本函数。
func NowForDB() time.Time {
	return Now().Truncate(time.Second)
}

// ToBeijing 把任意时间转换到北京时区
func ToBeijing(t time.Time) time.Time {
	return t.In(BeijingTZ)
}

// Format 输出 "2006-01-02 15:04:05"（北京时间）。
// 注意：与裸 t.Format(LayoutDateTime) 的区别是这里会先做时区换算，
// 因此 +00:00 / -05:00 的历史时间也能正确显示为北京时间。
func Format(t time.Time) string {
	return t.In(BeijingTZ).Format(LayoutDateTime)
}

// FormatDate 输出 "2006-01-02"（北京时间）
func FormatDate(t time.Time) string {
	return t.In(BeijingTZ).Format(LayoutDate)
}

// FormatLayout 按指定 layout 输出（先换算到北京时间）
func FormatLayout(t time.Time, layout string) string {
	return t.In(BeijingTZ).Format(layout)
}

// FormatNull 格式化可空时间，无效值返回空串
func FormatNull(nt sql.NullTime) string {
	if !nt.Valid {
		return ""
	}
	return Format(nt.Time)
}

// FormatNullOrDash 格式化可空时间，无效值返回 "-"
func FormatNullOrDash(nt sql.NullTime) string {
	if !nt.Valid {
		return "-"
	}
	return Format(nt.Time)
}

// RFC3339 输出带时区的 RFC3339（北京时间）
func RFC3339(t time.Time) string {
	return t.In(BeijingTZ).Format(time.RFC3339)
}

// ParseBeijingLayout 按指定 layout 在北京时区解析（替代裸 time.Parse）
func ParseBeijingLayout(layout, value string) (time.Time, error) {
	return time.ParseInLocation(layout, value, BeijingTZ)
}

// ParseBeijing 按 "2006-01-02 15:04:05" 解析（北京时区）
func ParseBeijing(value string) (time.Time, error) {
	return time.ParseInLocation(LayoutDateTime, value, BeijingTZ)
}

// ParseAny 依次尝试常见格式解析（兼容前端可能传来的多种写法），统一锚定北京时间。
// 覆盖：带纳秒/带时区偏移、纯日期、RFC3339、日期时间空格分隔。
func ParseAny(value string) (time.Time, error) {
	var lastErr error
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		LayoutDateTime,
		LayoutDate,
		"2006-01-02 15:04",
	} {
		if t, err := time.ParseInLocation(layout, value, BeijingTZ); err == nil {
			return t, nil
		} else {
			lastErr = err
		}
	}
	return time.Time{}, lastErr
}

// ParseDateRange 解析 [开始, 结束] 两个日期字符串（可为 "2006-01-02" 或日期时间），
// 结束时间若只给到日则补足到当天 23:59:59，避免"少算最后一天"。
func ParseDateRange(start, end string) (time.Time, time.Time, error) {
	var startAt, endAt time.Time
	var err error

	if start != "" {
		if startAt, err = ParseAny(start); err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	if end != "" {
		if endAt, err = ParseAny(end); err != nil {
			return time.Time{}, time.Time{}, err
		}
		// 只精确到日时，扩展到当天最后一秒
		if len(end) <= len(LayoutDate) {
			endAt = endAt.Add(24*time.Hour - time.Second)
		}
	}
	return startAt, endAt, nil
}

// RangeOfDay 返回 t 所在自然日的 [00:00:00, 23:59:59]（北京时间）
func RangeOfDay(t time.Time) (time.Time, time.Time) {
	local := t.In(BeijingTZ)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, BeijingTZ)
	return start, start.Add(24*time.Hour - time.Nanosecond)
}

// RangeOfMonth 返回 t 所在自然月的 [1日00:00:00, 月末23:59:59]（北京时间）
func RangeOfMonth(t time.Time) (time.Time, time.Time) {
	local := t.In(BeijingTZ)
	start := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, BeijingTZ)
	return start, start.AddDate(0, 1, 0).Add(-time.Nanosecond)
}

// RangeOfDays 返回最近 n 天（含今天）的起始时刻与当前时刻
func RangeOfDays(n int) (time.Time, time.Time) {
	now := Now()
	start, _ := RangeOfDay(now)
	if n > 1 {
		start = start.AddDate(0, 0, -(n - 1))
	}
	return start, now
}

// Since 返回距 t 过去的时间长度（用于耗时统计，与 time.Since 等价但基于北京时间同样正确）
func Since(t time.Time) time.Duration {
	return time.Since(t)
}
