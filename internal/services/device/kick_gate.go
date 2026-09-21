package device

import (
	"os"
	"strings"
)

// KickGateResult 是「被删除（踢下线）的设备再次拉取订阅」的处理结论。
type KickGateResult struct {
	// Blocked=true 表示明确拒绝本次请求（名额已满，无法自动恢复）。
	Blocked bool
	// Revived=true 表示本次已把该设备的软删行自动恢复为活跃设备，应放行。
	Revived bool
	// Reason 取值："not-kicked" / "already-active" / "auto-revived" / "device-limit" / "auto-revive-disabled"
	Reason string
	// DeviceID 命中的软删设备行 id（0 表示没有命中）。
	DeviceID uint
}

// KickAutoReviveDisabled 允许用环境变量关掉「自动恢复」，回到严格黑名单语义。
//   DEVICE_KICK_AUTO_REVIVE=false → 被删除的设备必须重新登录或找客服，不再自愈。
func KickAutoReviveDisabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("DEVICE_KICK_AUTO_REVIVE")))
	return v == "false" || v == "0" || v == "no" || v == "off"
}

// ResolveKickGate 决定「被删除的设备」再次请求订阅时应当如何处理。
//
// 事故背景（2026-09-21 生产排查）：
//
//	删除设备 = is_active=false + kicked_at=now；此后该设备每次拉订阅都会被
//	FindKickedDeviceWithHeaders 命中 → 403「此设备已被移除并踢下线,如需继续使用
//	请重新登录或联系客服」。但**全代码库没有任何地方清除 kicked_at**，
//	"重新登录"在旧版本里也并不会触发恢复 → 这条软删行等价于**永久黑名单**：
//	线上 148 个有效订阅 / 408 台设备被锁死，用户机器每 30 分钟重试一次、永久失败，
//	哪怕订阅名额根本没用满（例如 2/5）也照锁 —— 只能人工改库。
//
// 现行语义（既保留「删除设备=踢下线」，又保证可恢复）：
//  1. 该设备此刻已在活跃列表里 → 不拦截（历史软删行不应影响已登记的设备）；
//  2. 名额未满（或用户开启了不限制设备）→ 自动把这一行恢复为活跃设备并放行（自愈）；
//  3. 名额已满 → 仍然拒绝，但明确提示用户先删除闲置设备（客户端可自助重绑）。
//  4. 环境变量 DEVICE_KICK_AUTO_REVIVE=false 可关闭第 2 条，回到严格模式。
func (dm *DeviceManager) ResolveKickGate(
	subscriptionID uint,
	deviceLimit int,
	unlimited bool,
	userAgent, ipAddress string,
	headers map[string]string,
) (KickGateResult, error) {
	kicked, err := dm.FindKickedDeviceWithHeaders(subscriptionID, userAgent, ipAddress, headers)
	if err != nil {
		return KickGateResult{}, err
	}
	if kicked == nil {
		return KickGateResult{Reason: "not-kicked"}, nil
	}

	res := KickGateResult{DeviceID: kicked.ID}

	// 1) 这台设备此刻已经在活跃列表里（同一台机器既有历史软删行、又有活跃行，
	//    或刚刚被其它路径恢复）→ 历史软删行不该再拦住它。
	if active, exists, fErr := dm.FindExistingDeviceWithHeaders(subscriptionID, userAgent, ipAddress, headers); fErr != nil {
		return KickGateResult{}, fErr
	} else if exists && active != nil && active.ID != kicked.ID && active.IsActive && active.KickedAt == nil {
		res.Reason = "already-active"
		return res, nil
	}

	if KickAutoReviveDisabled() {
		res.Blocked = true
		res.Reason = "auto-revive-disabled"
		return res, nil
	}

	// 2) 名额未满 → 自动恢复这一行（保留备注/首次出现时间，只清 kicked_at）
	revived, reason, rErr := dm.ReviveKickedDeviceWithHeaders(subscriptionID, deviceLimit, unlimited, userAgent, ipAddress, headers)
	if rErr != nil {
		return KickGateResult{}, rErr
	}
	if revived {
		res.Revived = true
		res.Reason = "auto-revived"
		return res, nil
	}
	if reason == "device-limit" {
		res.Blocked = true
		res.Reason = "device-limit"
		return res, nil
	}
	// not-kicked：并发下已被其它请求恢复 → 放行。
	res.Reason = reason
	return res, nil
}
