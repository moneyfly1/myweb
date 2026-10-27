package device

import (
	"time"

	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"gorm.io/gorm"
)

// CountActiveDevices 统计指定订阅下处于活跃状态的设备数。
// 统一各处的 "subscription_id = ? AND is_active = true" 计数查询。
func CountActiveDevices(db *gorm.DB, subscriptionID uint) (int64, error) {
	var count int64
	err := db.Model(&models.Device{}).
		Where("subscription_id = ? AND is_active = ?", subscriptionID, true).
		Count(&count).Error
	return count, err
}

// OnlineWindow 「在线」判定窗口：多久之内拉过订阅的设备算作"当前在线"。
//
// 为什么设备数限制要用「在线窗口」而不是历史行数：
//   - 第三方客户端（Shadowrocket 等）不上报设备 ID，服务端只能靠 UA + 网络段区分；
//     同一台手机从 WiFi 切到 4G 会被记成另一行（这是识别两台同型号设备必须付出的代价）。
//   - 若按历史行数计名额，用户只是换个网络就会"设备超额"被断连 —— 明显不合理。
//     按在线窗口计，旧行很快不再活跃、不占名额；而两台设备同时使用仍是两个在线，
//     名额该占还是占（限制语义不变）。
//   - 这与业界通行做法一致：机场面板的设备/并发限制普遍按「同时在线的 IP/连接数」
//     来算，而不是"历史累计登记过的设备"（参见 Xboard 的在线设备数限制设计）。
//
// 取 15 分钟：客户端订阅更新间隔默认 60 分钟，但客户端重连与节点侧上报都会刷新
// last_access；15 分钟既能容忍短暂离线，也不会让"昨天用过"的旧行长期占名额。
const OnlineWindow = 15 * time.Minute

// CountOnlineDevices 统计指定订阅下「最近 OnlineWindow 内活跃」的设备数（= 当前在线设备数）。
// 设备数限制判定与 current_devices 展示统一使用这个口径。
func CountOnlineDevices(db *gorm.DB, subscriptionID uint) (int64, error) {
	cutoff := utils.GetBeijingTime().Add(-OnlineWindow)
	var count int64
	err := db.Model(&models.Device{}).
		Where("subscription_id = ? AND is_active = ? AND (last_access >= ? OR last_seen >= ?)",
			subscriptionID, true, cutoff, cutoff).
		Count(&count).Error
	return count, err
}
