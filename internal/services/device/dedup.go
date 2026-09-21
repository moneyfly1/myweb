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
func stableFingerprintKey(d *models.Device) (string, bool) {
	get := func(p *string) string {
		if p == nil {
			return ""
		}
		return strings.TrimSpace(*p)
	}
	sw, os := get(d.SoftwareName), get(d.OSName)
	model, brand := get(d.DeviceModel), get(d.DeviceBrand)
	if sw == "" || os == "" || model == "" || brand == "" {
		return "", false
	}
	if strings.EqualFold(sw, "Unknown") || strings.EqualFold(os, "Unknown") {
		return "", false
	}
	if !IsSpecificDeviceModel(model) {
		return "", false
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
			key, ok := stableFingerprintKey(&d)
			if !ok {
				continue
			}
			d := d
			groups[key] = append(groups[key], d)
		}

		for _, group := range groups {
			if len(group) < 2 {
				continue
			}
			// 额外保险：版本号完全相同却有多行 → 不能证明是"同一台设备升级后残留"，
			// 可能是同型号的两台真机，宁可不合并（合并错误 = 剥夺用户设备名额）
			if !hasVersionSpread(group) {
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
				count, cErr := CountActiveDevices(tx, subID)
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
	seen := map[string]bool{}
	for _, d := range group {
		v := ""
		if d.SoftwareVersion != nil {
			v = strings.TrimSpace(*d.SoftwareVersion)
		}
		if seen[v] && v != "" {
			continue
		}
		seen[v] = true
	}
	// 至少两个不同的非空版本
	distinct := 0
	for v := range seen {
		if v != "" {
			distinct++
		}
	}
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
