// Package userlevel 用户等级的公共逻辑：匹配、同步、折扣与"下一等级"。
//
// 为什么单独成包（2026-10-28）：等级逻辑此前散在三处 ——
//   - services/order.updateUserLevelTx 自己写一遍匹配 SQL；
//   - services/discount.CalculateLevelDiscount 直接按 user_level_id 取等级；
//   - handlers 里升级设备又抄了一遍折扣算法。
//
// 三处都漏了 is_active 判断，导致"停用等级仍能打折"；改一处漏两处。
// 现在全部收敛到这里，调用方只拿结果。
package userlevel

import (
	"database/sql"

	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"gorm.io/gorm"
)

// MinDiscountRate / MaxDiscountRate 折扣率合法区间（开区间）：0 < rate < 1
// 0 / 1 / >1 都视为"无折扣"，避免脏配置变成加价或白送。
func discountRateValid(rate float64) bool { return rate > 0 && rate < 1 }

// ActiveLevels 启用中的等级，按 level_order 升序。
func ActiveLevels(db *gorm.DB) ([]models.UserLevel, error) {
	var levels []models.UserLevel
	err := db.Where("is_active = ?", true).Order("level_order ASC").Find(&levels).Error
	return levels, err
}

// Match 返回累计消费命中的最高达标等级；没有达标等级时返回 nil。
func Match(db *gorm.DB, totalConsumption float64) (*models.UserLevel, error) {
	levels, err := ActiveLevels(db)
	if err != nil {
		return nil, err
	}
	var target *models.UserLevel
	for i := range levels {
		if totalConsumption >= levels[i].MinConsumption &&
			(target == nil || levels[i].LevelOrder > target.LevelOrder) {
			target = &levels[i]
		}
	}
	return target, nil
}

// Plan 只计算同步结果、不写库：返回目标等级、是否需要更新。
// 只升不降：当前等级的 level_order 高于目标时保持原样。
// 回填工具的干跑模式直接用它，保证"演练"与"执行"判定完全一致。
func Plan(db *gorm.DB, user *models.User) (*models.UserLevel, bool, error) {
	target, err := Match(db, user.TotalConsumption)
	if err != nil || target == nil {
		return nil, false, err
	}
	if !user.UserLevelID.Valid {
		return target, true, nil
	}
	var current models.UserLevel
	if err := db.First(&current, user.UserLevelID.Int64).Error; err == nil {
		if current.LevelOrder > target.LevelOrder {
			return &current, false, nil // 只升不降
		}
		return target, current.ID != target.ID, nil
	}
	// 当前等级已被删除：重新挂上目标等级
	return target, true, nil
}

// Sync 按累计消费同步用户等级（只升不降），返回同步后的等级与是否发生变更。
// 用户没有命中任何等级时保持原样（不降级）。
func Sync(tx *gorm.DB, user *models.User) (*models.UserLevel, bool, error) {
	target, changed, err := Plan(tx, user)
	if err != nil || target == nil || !changed {
		return target, false, err
	}
	user.UserLevelID = sql.NullInt64{Int64: utils.MustSafeUintToInt64(target.ID), Valid: true}
	if err := tx.Save(user).Error; err != nil {
		return nil, false, err
	}
	return target, true, nil
}

// ForUser 取用户当前**生效**的等级：等级被删除或停用时返回 nil。
// 折扣计算与前台展示都必须走这里，保证"停用即不生效"。
func ForUser(db *gorm.DB, userID uint) (*models.UserLevel, error) {
	if userID == 0 {
		return nil, nil
	}
	var user models.User
	if err := db.Select("id", "user_level_id").First(&user, userID).Error; err != nil {
		return nil, err
	}
	if !user.UserLevelID.Valid {
		return nil, nil
	}
	var level models.UserLevel
	if err := db.First(&level, user.UserLevelID.Int64).Error; err != nil {
		return nil, nil // 等级已被删除
	}
	if !level.IsActive {
		return nil, nil // 停用 = 不生效
	}
	return &level, nil
}

// DiscountRate 返回用户当前生效的折扣率；无等级或折扣非法时返回 1.0（原价）。
func DiscountRate(db *gorm.DB, userID uint) float64 {
	level, err := ForUser(db, userID)
	if err != nil || level == nil || !discountRateValid(level.DiscountRate) {
		return 1.0
	}
	return level.DiscountRate
}

// ApplyDiscount 按等级折扣计算金额，返回（优惠金额，折后金额）。
// 无等级/无折扣时原样返回 0 与 baseAmount。
func ApplyDiscount(db *gorm.DB, userID uint, baseAmount float64) (float64, float64) {
	baseAmount = utils.RoundFloat(baseAmount, 2)
	rate := DiscountRate(db, userID)
	if baseAmount <= 0 || rate >= 1.0 {
		return 0, baseAmount
	}
	discountAmount := utils.RoundFloat(baseAmount*(1.0-rate), 2)
	payable := utils.RoundFloat(baseAmount-discountAmount, 2)
	if payable < 0 {
		payable = 0
	}
	return discountAmount, payable
}

// NextLevel 返回比 current 更高的最低档等级；current 为 nil 时返回最低档。
func NextLevel(db *gorm.DB, current *models.UserLevel) *models.UserLevel {
	levels, err := ActiveLevels(db)
	if err != nil || len(levels) == 0 {
		return nil
	}
	baseOrder := 0
	if current != nil {
		baseOrder = current.LevelOrder
	}
	for i := range levels {
		if levels[i].LevelOrder > baseOrder {
			return &levels[i]
		}
	}
	return nil
}
