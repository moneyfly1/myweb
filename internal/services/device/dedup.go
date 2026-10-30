package device

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"cboard-go/internal/models"

	"gorm.io/gorm"
)

// DedupReport 合并结果（供 CLI 打印）
type DedupReport struct {
	SubscriptionsScanned int
	GroupsMerged         int
	RowsRemoved          int
	Details              []string
}

// 稳定指纹：软件名 + 系统名 + 机型 + 品牌。
//
// 为什么是这四个：它们是「这台设备是什么」的稳定描述；而版本号（App 版本 /
// 系统版本）会随升级变化，**不属于身份**。旧哈希算法把它们算进去了，于是用户
// 每次升级 App 或系统就会以「新设备」重新登记一行，旧行变成永久占用名额的
// 幽灵设备（也与「删除设备 = 踢下线」的判定互相漂移）。
//
// relaxed=true 时用于「新算法对齐」：2026-10-30 起设备身份 = 软件+系统+机型+品牌
// （不含网络段），而 Clash Meta / Mihomo Party 等客户端的 UA 解析不出具体机型
// （device_model 为空或只是 "Android"/"Desktop" 这类品类名），它们的身份实际上
// 就是「软件+系统」两项。宽松模式按同样的口径归并这些行，否则存量里
// 「同一台设备 82 行」的组永远清不掉。
func stableFingerprintKey(d *models.Device, relaxed bool) (string, bool) {
	get := func(p *string) string {
		if p == nil {
			return ""
		}
		return strings.TrimSpace(*p)
	}
	sw, os := get(d.SoftwareName), get(d.OSName)
	model, brand := get(d.DeviceModel), get(d.DeviceBrand)
	if sw == "" || os == "" {
		return "", false
	}
	if strings.EqualFold(sw, "Unknown") || strings.EqualFold(os, "Unknown") {
		return "", false
	}
	if !relaxed {
		if model == "" || brand == "" {
			return "", false
		}
		if !IsSpecificDeviceModel(model) {
			return "", false
		}
	}
	return strings.ToLower(sw) + "|" + strings.ToLower(os) + "|" +
		strings.ToLower(model) + "|" + strings.ToLower(brand), true
}

// MergeDuplicateDevices 合并「同一台设备被重复登记」的幽灵设备（存量数据清理）。
//
// 适用场景：旧算法（身份含版本号）时期积累的重复行 —— 同一台手机升级一次
// App/系统就多一行，设备数虚高、甚至被判定超限；被踢下线的判定也会错位。
//
// 规则（保守）：
//   - 只合并**同一订阅内稳定指纹完全相同**的行；指纹四要素缺失/Unknown 的一律不动；
//   - 保留「活跃且未踢」中最新的那行作为存活设备；若整组都被踢，则保留最近被踢的
//     那行并**保持被踢状态**（不偷偷复活用户主动删除的设备）；
//   - 合并时累计访问次数、取最早的首次出现时间、保留备注、取最近访问时间；
//   - 合并后重算该订阅的 current_devices。
//
// apply=false（默认）：只报告将要做什么，不写库。
func MergeDuplicateDevices(db *gorm.DB, apply bool) (DedupReport, error) {
	return MergeDuplicateDevicesFiltered(db, apply, DedupFilter{})
}

// DedupFilter 限定处理范围（留空 = 全部）。用于客服针对单个用户执行，
// 避免全量清理的风险（历史数据里"同型号多台真机"和"升级残留"难以区分）。
type DedupFilter struct {
	UserEmail      string
	SubscriptionID uint
	// SameVersion 允许合并「稳定指纹相同、软件版本也相同」的重复行。
	//
	// 缺省 false 时只合并"版本不同"的组（升级残留，证据明确）。设为 true 后，
	// 同一台设备因**换网络段**产生的重复行（版本完全相同、只有 device_hash
	// 不同）也会被合并 —— 2026-10-30 修复前，网络段参与设备身份，这类重复
	// 是幽灵设备的主要来源（全站 967 组）。代价：同型号的两台真机同版本时
	// 会被合并成一台，所以命令默认要求先 dry-run 人工复核。
	SameVersion bool
	// Relaxed 用「新身份算法对齐」的口径归并：不要求机型/品牌齐全，只要
	// 软件+系统一致就视为同一身份（机型为空或只是 "Android"/"Desktop" 这类
	// 品类名时也一样）。
	//
	// 这是清理「Clash Meta / Mihomo Party 等 UA 解析不出机型」残留的最后一档：
	// 2026-10-30 后设备身份不再含网络段，这类客户端在新算法下本来就只有一行，
	// 存量里 80 多行的组（订阅 623 实测 82 行）必须按同一口径收拢。
	// 风险与 SameVersion 相同：同型号两台真机会被合并成一台，务必先 dry-run。
	Relaxed bool
}

// MergeDuplicateDevicesFiltered 同 MergeDuplicateDevices，但可按用户/订阅过滤。
func MergeDuplicateDevicesFiltered(db *gorm.DB, apply bool, filter DedupFilter) (DedupReport, error) {
	report := DedupReport{}

	q := db.Model(&models.Device{}).
		Select("subscription_id").
		Group("subscription_id").
		Having("COUNT(*) > 1")
	if filter.SubscriptionID > 0 {
		q = q.Where("subscription_id = ?", filter.SubscriptionID)
	}
	if filter.UserEmail != "" {
		var uid uint
		if err := db.Model(&models.User{}).
			Where("LOWER(email) = ? OR username = ?",
				strings.ToLower(strings.TrimSpace(filter.UserEmail)),
				strings.TrimSpace(filter.UserEmail)).
			Pluck("id", &uid).Error; err != nil {
			return report, err
		}
		if uid == 0 {
			return report, fmt.Errorf("未找到用户: %s", filter.UserEmail)
		}
		var sids []uint
		if err := db.Model(&models.Subscription{}).Where("user_id = ?", uid).
			Pluck("id", &sids).Error; err != nil {
			return report, err
		}
		if len(sids) == 0 {
			return report, nil
		}
		q = q.Where("subscription_id IN ?", sids)
	}

	var subIDs []uint
	if err := q.Pluck("subscription_id", &subIDs).Error; err != nil {
		return report, err
	}
	report.SubscriptionsScanned = len(subIDs)

	for _, subID := range subIDs {
		var devices []models.Device
		if err := db.Where("subscription_id = ?", subID).Find(&devices).Error; err != nil {
			return report, err
		}

		groups := map[string][]models.Device{}
		for _, d := range devices {
			key, ok := stableFingerprintKey(&d, filter.Relaxed)
			if !ok {
				continue
			}
			groups[key] = append(groups[key], d)
		}

		for _, group := range groups {
			if len(group) < 2 {
				continue
			}
			// 默认只合并「版本不同」的组（升级残留，证据明确）。
			// filter.SameVersion=true 时额外合并「版本相同」的组：这类行是
			// 2026-10-30 之前"网络段并入设备身份"造成的换网重复，
			// 只有 device_hash 不同、软件版本完全一致。
			// filter.Relaxed=true 时按新身份口径直接合并（软件+系统即身份）。
			if !filter.Relaxed && !hasVersionSpread(group) && !filter.SameVersion {
				continue
			}
			sort.SliceStable(group, func(i, j int) bool {
				ai, aj := group[i].IsActive && group[i].KickedAt == nil, group[j].IsActive && group[j].KickedAt == nil
				if ai != aj {
					return ai // 优先保留活跃且未踢的
				}
				return group[i].LastAccess.After(group[j].LastAccess)
			})
			keeper := group[0]
			dups := group[1:]

			merged := keeper
			for _, d := range dups {
				merged.AccessCount += d.AccessCount
				if d.FirstSeen != nil && (merged.FirstSeen == nil || d.FirstSeen.Before(*merged.FirstSeen)) {
					merged.FirstSeen = d.FirstSeen
				}
				if merged.Remark == nil || strings.TrimSpace(*merged.Remark) == "" {
					if d.Remark != nil && strings.TrimSpace(*d.Remark) != "" {
						merged.Remark = d.Remark
					}
				}
				if d.LastAccess.After(merged.LastAccess) {
					merged.LastAccess = d.LastAccess
				}
				if d.LastSeen != nil && (merged.LastSeen == nil || d.LastSeen.After(*merged.LastSeen)) {
					merged.LastSeen = d.LastSeen
				}
			}
			// 整组都被踢 → 保持被踢（用户主动删过这台设备，不因合并而复活）
			if merged.KickedAt == nil && !merged.IsActive {
				var latestKick *time.Time
				for _, d := range append([]models.Device{keeper}, dups...) {
					if d.KickedAt != nil && (latestKick == nil || d.KickedAt.After(*latestKick)) {
						latestKick = d.KickedAt
					}
				}
				merged.KickedAt = latestKick
			}

			report.GroupsMerged++
			report.RowsRemoved += len(dups)
			report.Details = append(report.Details, fmt.Sprintf(
				"订阅 %d：保留设备 %d（%s，active=%v kicked=%v），删除重复行 %d 个",
				subID, keeper.ID, deviceLabel(&keeper), merged.IsActive, merged.KickedAt != nil, len(dups)))

			if !apply {
				continue
			}
			if err := db.Transaction(func(tx *gorm.DB) error {
				if err := tx.Model(&models.Device{}).Where("id = ?", keeper.ID).
					Updates(map[string]interface{}{
						"access_count": merged.AccessCount,
						"first_seen":   merged.FirstSeen,
						"remark":       merged.Remark,
						"last_access":  merged.LastAccess,
						"last_seen":    merged.LastSeen,
						"kicked_at":    merged.KickedAt,
					}).Error; err != nil {
					return err
				}
				ids := make([]uint, 0, len(dups))
				for _, d := range dups {
					ids = append(ids, d.ID)
				}
				if err := tx.Where("id IN ?", ids).Delete(&models.Device{}).Error; err != nil {
					return err
				}
				count, cErr := CountRecentDevices(tx, subID, DisplayWindow)
				if cErr != nil {
					return cErr
				}
				return tx.Model(&models.Subscription{}).Where("id = ?", subID).
					Update("current_devices", count).Error
			}); err != nil {
				return report, err
			}
		}
	}
	return report, nil
}

// hasVersionSpread 同组内是否存在「软件名相同但版本不同」的行。
// 这是"同一台设备因升级被重复登记"的判定依据；版本全同说明证据不足。
func hasVersionSpread(group []models.Device) bool {
	seen := map[string]struct{}{}
	for _, d := range group {
		if d.SoftwareVersion == nil {
			continue
		}
		if v := strings.TrimSpace(*d.SoftwareVersion); v != "" {
			seen[v] = struct{}{}
		}
	}
	// 至少两个不同的非空版本
	distinct := len(seen)
	return distinct >= 2
}

func deviceLabel(d *models.Device) string {
	parts := []string{}
	for _, p := range []*string{d.DeviceBrand, d.DeviceModel, d.OSName, d.SoftwareName} {
		if p != nil && strings.TrimSpace(*p) != "" {
			parts = append(parts, strings.TrimSpace(*p))
		}
	}
	if len(parts) == 0 {
		return fmt.Sprintf("设备#%d", d.ID)
	}
	return strings.Join(parts, " ")
}
