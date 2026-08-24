package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// cleanupTargets 可清理的数据类型注册表。
// 安全设计：仅清理"日志/历史/临时"类数据，绝不触碰用户/订单/订阅/支付等业务核心表。
type cleanupTarget struct {
	Label      string // 展示名
	Clear      func(db *gorm.DB, before time.Time) *gorm.DB
	IsTimeType bool   // 是否按 created_at 时间过滤（部分目标如邀请码按过期时间）
}

var cleanupTargets = map[string]cleanupTarget{
	"audit_logs": {
		Label: "操作审计日志",
		Clear: func(db *gorm.DB, before time.Time) *gorm.DB {
			// 保护登录/注册/签到等安全关键日志：全量清空也不删这三类
			q := db.Where("action_type NOT IN ?", []string{"login", "register", "checkin"})
			if !before.IsZero() {
				q = q.Where("created_at < ?", before)
			}
			return q.Delete(&models.AuditLog{})
		},
	},
	"registration_logs": {
		Label: "注册日志",
		Clear: func(db *gorm.DB, before time.Time) *gorm.DB {
			return deleteBefore(db, &models.RegistrationLog{}, before)
		},
	},
	"subscription_logs": {
		Label: "订阅日志",
		Clear: func(db *gorm.DB, before time.Time) *gorm.DB {
			return deleteBefore(db, &models.SubscriptionLog{}, before)
		},
	},
	"balance_logs": {
		Label: "余额变动日志",
		Clear: func(db *gorm.DB, before time.Time) *gorm.DB {
			return deleteBefore(db, &models.BalanceLog{}, before)
		},
	},
	"commission_logs": {
		Label: "佣金日志",
		Clear: func(db *gorm.DB, before time.Time) *gorm.DB {
			return deleteBefore(db, &models.CommissionLog{}, before)
		},
	},
	"subscription_reset_logs": {
		Label: "订阅重置日志",
		Clear: func(db *gorm.DB, before time.Time) *gorm.DB {
			return deleteBefore(db, &models.SubscriptionReset{}, before)
		},
	},
	"email_queue": {
		Label: "邮件队列/邮件日志",
		Clear: func(db *gorm.DB, before time.Time) *gorm.DB {
			return deleteBefore(db, &models.EmailQueue{}, before)
		},
	},
	"login_history": {
		Label: "登录历史",
		Clear: func(db *gorm.DB, before time.Time) *gorm.DB {
			// login_history 无 created_at 列，按 login_time 过滤
			q := db
			if !before.IsZero() {
				q = q.Where("login_time < ?", before)
			} else {
				q = q.Where("1 = 1")
			}
			return q.Delete(&models.LoginHistory{})
		},
	},
	"user_activities": {
		Label: "用户活动记录",
		Clear: func(db *gorm.DB, before time.Time) *gorm.DB {
			return deleteBefore(db, &models.UserActivity{}, before)
		},
	},
	"notifications": {
		Label: "站内通知",
		Clear: func(db *gorm.DB, before time.Time) *gorm.DB {
			return deleteBefore(db, &models.Notification{}, before)
		},
	},
	"login_attempts": {
		Label: "登录失败记录",
		Clear: func(db *gorm.DB, before time.Time) *gorm.DB {
			return deleteBefore(db, &models.LoginAttempt{}, before)
		},
	},
	"verification_codes": {
		Label: "验证码记录",
		Clear: func(db *gorm.DB, before time.Time) *gorm.DB {
			return deleteBefore(db, &models.VerificationCode{}, before)
		},
	},
	"checkin_logs": {
		Label: "签到记录",
		Clear: func(db *gorm.DB, before time.Time) *gorm.DB {
			return deleteBefore(db, &models.CheckinRecord{}, before)
		},
	},
	"payment_callbacks": {
		Label: "支付回调记录",
		Clear: func(db *gorm.DB, before time.Time) *gorm.DB {
			return deleteBefore(db, &models.PaymentCallback{}, before)
		},
	},
	"invite_codes": {
		Label: "过期邀请码",
		Clear: func(db *gorm.DB, before time.Time) *gorm.DB {
			// 只清理已过期的邀请码，避免误删仍可用的邀请链接
			q := db.Where("expires_at IS NOT NULL AND expires_at < ?", utils.GetBeijingTime())
			if !before.IsZero() {
				q = q.Where("created_at < ?", before)
			}
			return q.Delete(&models.InviteCode{})
		},
	},
	"token_blacklist": {
		Label: "已过期黑名单令牌",
		Clear: func(db *gorm.DB, before time.Time) *gorm.DB {
			// 只清理已过期的黑名单令牌（仍然有效的黑名单必须保留以防重放）
			return db.Where("expires_at < ?", utils.GetBeijingTime()).Delete(&models.TokenBlacklist{})
		},
	},
}

func deleteBefore(db *gorm.DB, model interface{}, before time.Time) *gorm.DB {
	q := db
	if !before.IsZero() {
		q = q.Where("created_at < ?", before)
	} else {
		// 显式条件触发全量删除：GORM 对无 WHERE 的 Delete 默认拒绝
		q = q.Where("1 = 1")
	}
	return q.Delete(model)
}

// CleanupData 统一数据清理端点：POST /admin/cleanup/:type?before=YYYY-MM-DD
// before 缺省 = 清空该类型全部（受类型自身保护规则约束）。
func CleanupData(c *gin.Context) {
	cleanupType := strings.ToLower(strings.TrimSpace(c.Param("type")))
	target, ok := cleanupTargets[cleanupType]
	if !ok {
		utils.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("不支持的清理类型: %s", cleanupType), nil)
		return
	}

	var before time.Time
	if beforeStr := strings.TrimSpace(c.Query("before")); beforeStr != "" {
		parsed, err := time.ParseInLocation("2006-01-02", beforeStr, utils.BeijingTZ)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "before 参数格式应为 YYYY-MM-DD", nil)
			return
		}
		before = parsed
	}

	db := database.GetDB()
	result := target.Clear(db, before)
	if result.Error != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "清理失败", result.Error)
		return
	}

	scopeDesc := "全部"
	if !before.IsZero() {
		scopeDesc = before.Format("2006-01-02") + " 之前"
	}
	utils.CreateAuditLogSimple(c, "data_cleanup", "cleanup", 0,
		fmt.Sprintf("管理员操作: 清理%s（%s）共 %d 条", target.Label, scopeDesc, result.RowsAffected))
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("已清理 %d 条%s", result.RowsAffected, target.Label), gin.H{
		"type":           cleanupType,
		"deleted_count":  result.RowsAffected,
		"before":         before.Format("2006-01-02"),
		"cleared_all":    before.IsZero(),
	})
}

// GetCleanupRetention 读取自动清理保留天数配置（category=cleanup）。
func GetCleanupRetention(c *gin.Context) {
	db := database.GetDB()
	settings := map[string]string{}
	var configs []models.SystemConfig
	if err := db.Where("category = ?", "cleanup").Find(&configs).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "读取清理配置失败", err)
		return
	}
	for _, cfg := range configs {
		settings[cfg.Key] = cfg.Value
	}
	utils.SuccessResponse(c, http.StatusOK, "", settings)
}

// UpdateCleanupRetention 保存自动清理保留天数配置（复用通用设置保存逻辑）。
func UpdateCleanupRetention(c *gin.Context) {
	updateSettingsCommon(c, "cleanup")
}
