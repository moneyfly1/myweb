package device

import (
	"time"

	"cboard-go/internal/models"
)

// 设备级「是否在用」判定。
//
// 与用户级（internal/services/activity）同一套思路，但有个关键差别：
// **设备没有独立心跳**。每次该设备拉订阅时，device_manager 会把 LastSeen/LastAccess
// 一起更新为当前时间，所以设备活跃度 = 「这个设备最近一次拉订阅的时间」，
// 粒度取决于客户端更新周期（面板下发 Profile-Update-Interval，默认 60 分钟）。
//
// 因此设备级只有三档（而不是用户级的四档，用户级多一个「在线」——那个来自节点侧按用户上报）：
//
//	active   1 小时内拉取过订阅 —— 客户端仍在按周期更新，等价于「活跃」
//	recent   24 小时内拉取过 —— 近期用过，现在没动静
//	offline  超过 24 小时没拉取 —— 离线（卸载 / 换客户端 / 已不用）
//
// 想要设备级「秒级在线」，只有两条路（当前都没做）：
//  1. 自研客户端增加独立心跳接口（带上设备唯一 ID，如 X-MF-Device-Id），面板实时打点；
//  2. 节点侧按用户上报连接数（需要节点多用户化）+ 设备维度绑定（一个用户多设备时无法区分）。
var (
	// ActiveWindow 多久内拉取过订阅算「活跃」（与用户级一致，≈ 客户端更新周期）
	ActiveWindow = 60 * time.Minute
	// RecentWindow 多久内拉取过算「近期用过」，超过则离线
	RecentWindow = 24 * time.Hour
)

// 状态常量（供 handler 判断，避免散落魔法字符串）
const (
	StateActiveDevice  = "active"
	StateRecentDevice  = "recent"
	StateOfflineDevice = "offline"
)

// DeviceActivity 单个设备的活跃状态
type DeviceActivity struct {
	State        string    `json:"state"`       // active / recent / offline
	StateLabel   string    `json:"state_label"` // 活跃 / 近期用过 / 离线
	AgoSeconds   int64     `json:"ago_seconds"` // 距最近一次拉订阅的秒数（前端可渲染「3 分钟前」）
	LastActiveAt time.Time `json:"last_active_at"`
	Basis        string    `json:"basis"`
}

// LastActiveAt 取设备的最近活跃时间：LastSeen 优先（历史上只有该字段被单独更新过），
// 缺省用 LastAccess。
func LastActiveAt(d models.Device) time.Time {
	if d.LastSeen != nil && !d.LastSeen.IsZero() {
		return *d.LastSeen
	}
	return d.LastAccess
}

// ActivityOf 计算设备活跃状态。
// 判据是「最近一次拉订阅的时间」，注意这**不等于**用户此刻正在上网——
// 第三方客户端只会在更新周期到点时才回拉一次，所以 active 的真实含义是
// 「客户端还在按周期更新，说明这台设备还在用这个订阅」。
func ActivityOf(d models.Device, now time.Time) DeviceActivity {
	last := LastActiveAt(d)
	if last.IsZero() {
		return DeviceActivity{
			State:      StateOfflineDevice,
			StateLabel: "离线",
			Basis:      "从未拉取过订阅",
		}
	}
	ago := now.Sub(last)
	if ago < 0 {
		ago = 0 // 时钟回拨保护
	}
	act := DeviceActivity{
		AgoSeconds:   int64(ago.Seconds()),
		LastActiveAt: last,
	}
	switch {
	case ago <= ActiveWindow:
		act.State = StateActiveDevice
		act.StateLabel = "活跃"
		act.Basis = "1 小时内拉取过订阅（客户端仍在按更新周期回拉）"
	case ago <= RecentWindow:
		act.State = StateRecentDevice
		act.StateLabel = "近期用过"
		act.Basis = "24 小时内拉取过订阅，当前没有动静"
	default:
		act.State = StateOfflineDevice
		act.StateLabel = "离线"
		act.Basis = "超过 24 小时没有拉取订阅"
	}
	return act
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
