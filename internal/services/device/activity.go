package device

import (
	"fmt"
	"math"
	"time"

	"cboard-go/internal/models"
)

// 设备级「是否在用」判定。
//
// 与用户级（internal/services/activity）同一套思路，但设备没有独立心跳：
// 每次该设备拉订阅时，device_manager 会把 LastSeen/LastAccess 更新为当前时间，
// 所以设备活跃度 = 「这个设备最近一次拉订阅的时间」。
//
// ⚠️ 关键实测数据（它决定了不能用固定窗口）：生产库 6562 台设备里，单台设备的
// **平均回拉间隔普遍是 4~15 小时**，远大于面板下发的 Profile-Update-Interval（60 分钟）——
// 因为多数客户端只在打开 App / 手动刷新时才回拉，并不严格按 interval 轮询。
// 实测某用户设备（access_count 均 >100）的平均间隔：235 / 283 / 450 / 597 / 689 / 742 / 860 / 903 分钟。
//
// 结论：固定 1 小时窗口会把一个天天在用的账号判成「活跃设备 0 台」（用户真实反馈）。
// 因此改为**按设备自身节奏自适应**：
//
//	阈值 = clamp(2 × 该设备平均回拉间隔, 1 小时, 24 小时)
//	活跃     —— 距上次回拉 ≤ 阈值（处在它自己的正常节奏内）
//	近期用过 —— ≤ 24 小时（超出自身节奏，但一天内还出现过）
//	离线     —— > 24 小时
//
// 乘 2 是允许偶发漏掉一个周期（关机、切网络、App 没打开）仍算在用；
// 上限 24 小时 / 下限 1 小时是护栏：新设备样本少不会被算出极小阈值，
// 一年只拉过一次的设备也不会被一直当成「活跃」。
var (
	// minActiveWindow 活跃阈值下限（新设备/样本不足时的兜底）
	minActiveWindow = time.Hour
	// maxActiveWindow 活跃阈值上限（防止历史均值过大导致长期误判为活跃）
	maxActiveWindow = 24 * time.Hour
	// RecentWindow 超过它就算离线
	RecentWindow = 24 * time.Hour
	// intervalMultiplier 允许漏掉的周期倍数
	intervalMultiplier = 2.0
)

// 状态常量（供 handler 判断，避免散落魔法字符串）
const (
	StateActiveDevice  = "active"
	StateRecentDevice  = "recent"
	StateOfflineDevice = "offline"
)

// DeviceActivity 单个设备的活跃状态
type DeviceActivity struct {
	State          string    `json:"state"` // active / recent / offline
	StateLabel     string    `json:"state_label"`
	AgoSeconds     int64     `json:"ago_seconds"`      // 距最近一次拉订阅的秒数
	LastActiveAt   time.Time `json:"last_active_at"`   //
	AvgIntervalSec int64     `json:"avg_interval_sec"` // 该设备观察到的平均回拉间隔
	ThresholdSec   int64     `json:"threshold_sec"`    // 本次判定用的活跃阈值
	Basis          string    `json:"basis"`            // 给人看的判定依据
}

// LastActiveAt 取设备的最近活跃时间：LastSeen 优先（历史上只有该字段被单独更新过），
// 缺省用 LastAccess。
func LastActiveAt(d models.Device) time.Time {
	if d.LastSeen != nil && !d.LastSeen.IsZero() {
		return *d.LastSeen
	}
	return d.LastAccess
}

// observedInterval 该设备观察到的平均回拉间隔：(now - 首次出现) / 累计拉取次数。
// 样本不足（<2 次）或时间信息缺失时返回 0，由护栏兜底。
func observedInterval(d models.Device, now time.Time) time.Duration {
	if d.AccessCount < 2 {
		return 0
	}
	first := d.CreatedAt
	if d.FirstSeen != nil && !d.FirstSeen.IsZero() {
		first = *d.FirstSeen
	}
	if first.IsZero() || !now.After(first) {
		return 0
	}
	avg := now.Sub(first) / time.Duration(d.AccessCount)
	if avg <= 0 {
		return 0
	}
	return avg
}

// activeThreshold 活跃阈值 = clamp(2 × 平均间隔, 1h, 24h)；无样本时取下限
func activeThreshold(avg time.Duration) time.Duration {
	if avg <= 0 {
		return minActiveWindow
	}
	th := time.Duration(float64(avg) * intervalMultiplier)
	if th < minActiveWindow {
		th = minActiveWindow
	}
	if th > maxActiveWindow {
		th = maxActiveWindow
	}
	return th
}

// ActivityOf 计算设备活跃状态。
//
// 语义提醒：active 表示「这台设备还在按自己的节奏回拉订阅」，不等于「此刻正在上网」。
// 设备级要做到秒级在线，只能靠客户端主动心跳（自研客户端可加），见文档。
func ActivityOf(d models.Device, now time.Time) DeviceActivity {
	last := LastActiveAt(d)
	avg := observedInterval(d, now)
	th := activeThreshold(avg)

	act := DeviceActivity{
		AvgIntervalSec: int64(avg.Seconds()),
		ThresholdSec:   int64(th.Seconds()),
	}

	if last.IsZero() {
		act.State = StateOfflineDevice
		act.StateLabel = "离线"
		act.Basis = "从未拉取过订阅"
		return act
	}

	ago := now.Sub(last)
	if ago < 0 {
		ago = 0 // 时钟回拨保护
	}
	act.AgoSeconds = int64(ago.Seconds())
	act.LastActiveAt = last

	avgText := "回拉样本不足"
	if avg > 0 {
		avgText = fmt.Sprintf("平均约每 %s 回拉一次", humanDuration(avg))
	}

	switch {
	case ago <= th:
		act.State = StateActiveDevice
		act.StateLabel = "活跃"
		act.Basis = fmt.Sprintf("最近一次回拉在 %s前，%s，处在正常节奏内（阈值 %s）",
			humanDuration(ago), avgText, humanDuration(th))
	case ago <= RecentWindow:
		act.State = StateRecentDevice
		act.StateLabel = "近期用过"
		act.Basis = fmt.Sprintf("最近一次回拉在 %s前，已超出它自己的节奏（%s，阈值 %s）",
			humanDuration(ago), avgText, humanDuration(th))
	default:
		act.State = StateOfflineDevice
		act.StateLabel = "离线"
		act.Basis = fmt.Sprintf("超过 %s没有回拉订阅（%s）", humanDuration(RecentWindow), avgText)
	}
	return act
}

// humanDuration 把时长写成「3 分钟 / 4.7 小时 / 2 天」这种给人看的文本
func humanDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%d 秒", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%d 分钟", int(math.Round(d.Minutes())))
	case d < 24*time.Hour:
		return fmt.Sprintf("%.1f 小时", d.Hours())
	case d < 48*time.Hour:
		return "1 天"
	default:
		return fmt.Sprintf("%.0f 天", d.Hours()/24)
	}
}

// ActivityCounts 批量统计（设备管理页的概览数字）
type ActivityCounts struct {
	Active  int64 `json:"active"`
	Recent  int64 `json:"recent"`
	Offline int64 `json:"offline"`
}

func CountByActivity(devices []models.Device, now time.Time) ActivityCounts {
	var c ActivityCounts
	for _, d := range devices {
		switch ActivityOf(d, now).State {
		case StateActiveDevice:
			c.Active++
		case StateRecentDevice:
			c.Recent++
		default:
			c.Offline++
		}
	}
	return c
}

// Windows 返回当前生效的窗口参数，供接口下发（前端据此渲染文案，避免写死后与后端漂移）
func Windows() map[string]int64 {
	return map[string]int64{
		"min_active_seconds": int64(minActiveWindow.Seconds()),
		"max_active_seconds": int64(maxActiveWindow.Seconds()),
		"recent_seconds":     int64(RecentWindow.Seconds()),
		"multiplier":         int64(intervalMultiplier),
	}
}
