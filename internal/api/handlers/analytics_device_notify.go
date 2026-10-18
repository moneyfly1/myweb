package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/email"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// 「引导第三方客户端用户升级到官方客户端」批量发信
//
// 需求（2026-10-16）：用户分析页里筛出没在用官方客户端的用户，勾选（或全选）
// 后批量发邮件，邮件里给出**带国内加速镜像**的官方客户端下载地址。
//
// 安全与克制：
//   - dry_run 默认 true：先返回渲染好的标题/正文（含真实镜像链接）与目标人数，
//     管理员确认后才真正入队 —— 群发邮件不可撤销，必须能先看内容；
//   - 单次最多 1000 人（maxNotifyBatch），超出需分批，避免一次点爆邮件队列；
//   - 只发**活跃账号 + 有邮箱 + 未关闭邮件通知**的用户，跳过的原因逐类计数返回；
//   - 同一用户多台第三方设备只发一封（按用户去重）。
// ---------------------------------------------------------------------------

const maxNotifyBatch = 1000

var clientUpgradeEmailType = "client_upgrade"

// defaultClientUpgradeSubject 内置主题（可在弹窗里改成自己的文案）
const defaultClientUpgradeSubject = "建议升级到官方客户端 MoneyFly（含国内加速下载地址）"

// ClientUpgradeLinks 各平台下载链接（全部走本站的加速镜像解析入口）
//
//   - /api/v1/download/gh?key=…   → 服务器代查 GitHub 最新 Release 资产，302 到
//     当前可用的国内加速镜像（ghfast.top 等），且永远是最新版本；
//   - /download/resolve?target=…  → 通用镜像解析（备用入口）。
//
// key 与 internal/services/software_sync 目录里的 MoneyFly 目标一一对应。
func ClientUpgradeLinks(siteBase string) map[string]string {
	base := strings.TrimRight(siteBase, "/")
	gh := func(key string) string {
		return fmt.Sprintf("%s/api/v1/download/gh?key=%s", base, key)
	}
	return map[string]string{
		"windows":     gh("moneyfly_windows_url"),
		"macos_arm":   gh("moneyfly_macos_arm_url"),
		"macos_intel": gh("moneyfly_macos_url"),
		"android":     gh("moneyfly_android_url"),
		// iOS 走站点下载页（sideload 说明），没有直链安装包
		"ios":  base + "/download",
		"page": base + "/download",
	}
}

// deviceUsageTargetUserIDs 按当前筛选条件取出「需要提醒的用户」（去重）
//
// 传了 user_ids 就用传的；否则按 dim/key/official/days/q 从 devices 表反查用户。
// 说明：只取**活跃设备行**对应的人 —— 已经不用第三方客户端的老记录不该再打扰。
func deviceUsageTargetUserIDs(c *gin.Context, req notifyUpgradeRequest) ([]uint, error) {
	if len(req.UserIDs) > 0 {
		out := make([]uint, 0, len(req.UserIDs))
		seen := map[uint]bool{}
		for _, id := range req.UserIDs {
			if id > 0 && !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
		return out, nil
	}

	db := database.GetDB()
	dim := resolveDeviceUsageDim(req.Dim)
	official := strings.TrimSpace(req.Official)
	if official == "" {
		official = "third_party" // 默认只看非官方客户端：这正是要劝升级的人
	}
	base := deviceUsageBaseQuery(db, req.Q, official, req.Days, dim)
	if strings.TrimSpace(req.Key) != "" {
		base = base.Where(dim.Column+" = ?", req.Key)
	} else if len(req.Keys) > 0 {
		// 前端勾选多个分组：一次把这几个分组下的用户都取出来（按用户去重）
		keys := make([]string, 0, len(req.Keys))
		for _, k := range req.Keys {
			if strings.TrimSpace(k) != "" {
				keys = append(keys, strings.TrimSpace(k))
			}
		}
		if len(keys) > 0 {
			base = base.Where(dim.Column+" IN ?", keys)
		}
	}

	var ids []uint
	err := base.Session(&gorm.Session{}).
		Where("user_id IS NOT NULL").
		Distinct("user_id").
		Pluck("user_id", &ids).Error
	return ids, err
}

type notifyUpgradeRequest struct {
	Dim      string   `json:"dim"`
	Key      string   `json:"key"`
	Keys     []string `json:"keys"`
	Official string   `json:"official"`
	Days     string   `json:"days"`
	Q        string   `json:"q"`
	UserIDs  []uint   `json:"user_ids"`
	Subject  string   `json:"subject"`
	Content  string   `json:"content"`
	DryRun   *bool    `json:"dry_run"`
	Note     string   `json:"note"`
}

// NotifyClientUpgrade 批量发「升级到官方客户端」邮件
// POST /api/v1/admin/analytics/device-usage/notify-upgrade
func NotifyClientUpgrade(c *gin.Context) {
	var req notifyUpgradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}
	dryRun := true
	if req.DryRun != nil {
		dryRun = *req.DryRun
	}

	db := database.GetDB()
	ids, err := deviceUsageTargetUserIDs(c, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "筛选目标用户失败", err)
		return
	}
	if len(ids) == 0 {
		utils.SuccessResponse(c, http.StatusOK, "", gin.H{
			"targets": 0, "dry_run": dryRun,
			"message": "当前筛选条件下没有可提醒的用户",
		})
		return
	}

	// 截断保护：单次最多 maxNotifyBatch 人
	truncated := 0
	if len(ids) > maxNotifyBatch {
		truncated = len(ids) - maxNotifyBatch
		ids = ids[:maxNotifyBatch]
	}

	var users []models.User
	if err := db.Where("id IN ?", ids).Find(&users).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "读取用户失败", err)
		return
	}

	builder := email.NewEmailTemplateBuilder()
	siteBase := builder.GetBaseURL()
	links := ClientUpgradeLinks(siteBase)

	// 每个用户当前的第三方客户端（邮件里点名，比泛泛而谈更有说服力）
	clientOf := currentThirdPartyClientOf(db, ids)

	skippedNoEmail, skippedInactive, skippedUnsubscribed := 0, 0, 0
	type preview struct {
		UserID  uint   `json:"user_id"`
		Email   string `json:"email"`
		Subject string `json:"subject"`
		Content string `json:"content"`
	}
	var first *preview
	targets := make([]models.User, 0, len(users))
	for _, u := range users {
		if strings.TrimSpace(u.Email) == "" {
			skippedNoEmail++
			continue
		}
		if !u.IsActive {
			skippedInactive++
			continue
		}
		if !u.EmailNotifications {
			skippedUnsubscribed++
			continue
		}
		targets = append(targets, u)
	}

	subject := strings.TrimSpace(req.Subject)
	contentTpl := strings.TrimSpace(req.Content)

	queued, failed := 0, 0
	failReasons := []string{}
	svc := email.NewEmailService()
	for i := range targets {
		u := targets[i]
		subj, body := subject, contentTpl
		if subj == "" {
			subj = defaultClientUpgradeSubject
		}
		if body == "" {
			// 未自定义正文 → 用内置模板（正文里含官方下载页 + 各平台加速镜像直链）
			body = builder.GetClientUpgradeNoticeTemplate(
				displayName(u), clientOf[u.ID], links, req.Note)
			body = renderUpgradeVars(body, u, clientOf[u.ID], links)
		}
		subj = renderUpgradeVars(subj, u, clientOf[u.ID], links)
		if first == nil {
			first = &preview{UserID: u.ID, Email: u.Email, Subject: subj, Content: body}
		}
		if dryRun {
			continue
		}
		if err := svc.QueueEmail(u.Email, subj, body, clientUpgradeEmailType); err != nil {
			failed++
			if len(failReasons) < 5 {
				failReasons = append(failReasons, fmt.Sprintf("%s: %v", u.Email, err))
			}
			continue
		}
		queued++
	}

	// 邮件配置是否可用（SMTP 未配置时提示管理员，避免"发了但发不出去"）。
	// 同时回显主机与脱敏账号 —— 管理员能一眼看出"是不是配置丢了"。
	smtpConfigured, smtpHost, smtpUser := smtpStatus(db)

	out := gin.H{
		"dry_run":              dryRun,
		"targets":              len(targets),
		"truncated":            truncated,
		"queued":               queued,
		"failed":               failed,
		"skipped_no_email":     skippedNoEmail,
		"skipped_inactive":     skippedInactive,
		"skipped_unsubscribed": skippedUnsubscribed,
		"smtp_configured":      smtpConfigured,
		"smtp_host":            smtpHost,
		"smtp_user":            smtpUser,
		"download_links":       links,
		"max_batch":            maxNotifyBatch,
	}
	if first != nil {
		out["preview"] = gin.H{
			"user_id": first.UserID,
			"email":   first.Email,
			"subject": first.Subject,
			"content": first.Content,
		}
	}
	if len(failReasons) > 0 {
		out["fail_reasons"] = failReasons
	}
	utils.SuccessResponse(c, http.StatusOK, "", out)
}

// currentThirdPartyClientOf 每个用户最近一次使用的客户端名（用于邮件正文点名）
func currentThirdPartyClientOf(db *gorm.DB, userIDs []uint) map[uint]string {
	out := map[uint]string{}
	if len(userIDs) == 0 {
		return out
	}
	type row struct {
		UserID       int64
		SoftwareName string
	}
	var rows []row
	// 每个用户取最后活跃的一行（SQLite/MySQL/PG 都支持的写法：先排序再在内存里取首个）
	db.Model(&models.Device{}).
		Select("user_id, COALESCE(software_name, '') AS software_name").
		Where("user_id IN ? AND is_active = ? AND kicked_at IS NULL", userIDs, true).
		Order("last_access DESC").
		Scan(&rows)
	for _, r := range rows {
		uid := uint(r.UserID)
		if _, ok := out[uid]; ok {
			continue
		}
		name := strings.TrimSpace(r.SoftwareName)
		if name == "" || strings.EqualFold(name, officialClientName) {
			continue
		}
		out[uid] = name
	}
	return out
}

func displayName(u models.User) string {
	if strings.TrimSpace(u.Nickname.String) != "" {
		return strings.TrimSpace(u.Nickname.String)
	}
	if strings.TrimSpace(u.Username) != "" {
		return u.Username
	}
	return u.Email
}

// renderUpgradeVars 支持在自定义主题/正文里使用占位符
func renderUpgradeVars(tpl string, u models.User, client string, links map[string]string) string {
	repl := map[string]string{
		"{username}":       displayName(u),
		"{email}":          u.Email,
		"{client}":         client,
		"{download_page}":  links["page"],
		"{dl_windows}":     links["windows"],
		"{dl_macos}":       links["macos_arm"],
		"{dl_macos_intel}": links["macos_intel"],
		"{dl_android}":     links["android"],
		"{dl_ios}":         links["ios"],
	}
	out := tpl
	for k, v := range repl {
		out = strings.ReplaceAll(out, k, v)
	}
	return out
}

// isSMTPConfigured SMTP 是否真的可用（后台「系统设置 → 邮件」或 .env）
//
// ⚠️ 键名有两套，必须都认（EmailService 自己就是双写兼容的，见
// internal/services/email/email.go 的 NewEmailService）：
//
//	主机：smtp_host
//	账号：smtp_username  → 缺失时回退 email_username
//	密码：smtp_password  → 缺失时回退 email_password
//
// 只认其中一套会误报"未配置"（2026-10-18 实际踩到：线上后台用的是
// email_username / email_password，接口却回报 smtp_configured=false，
// 管理员以为配置丢了）。
//
// 另外排除 .env 里的占位值（your-email@qq.com / your-smtp-password），
// 那种"看起来配了其实发不出去"的情况也要如实报出来。
func smtpStatus(db *gorm.DB) (configured bool, host, user string) {
	var configs []models.SystemConfig
	db.Where("category = ?", "email").Find(&configs)
	m := map[string]string{}
	for _, c := range configs {
		m[c.Key] = strings.TrimSpace(c.Value)
	}
	pick := func(keys ...string) string {
		for _, k := range keys {
			if v := m[k]; v != "" {
				return v
			}
		}
		return ""
	}
	host = pick("smtp_host", "email_host", "mail_host")
	user = pick("smtp_username", "email_username", "mail_username")
	pass := pick("smtp_password", "email_password", "mail_password")
	if host != "" && user != "" && pass != "" && !looksPlaceholder(user, pass) {
		return true, host, maskEmail(user)
	}
	// 回退到 .env（SMTP_HOST / SMTP_USERNAME / SMTP_PASSWORD）
	if cfg := config.AppConfig; cfg != nil {
		h := strings.TrimSpace(cfg.SMTPHost)
		u := strings.TrimSpace(cfg.SMTPUser)
		pw := strings.TrimSpace(cfg.SMTPPassword)
		if h != "" && u != "" && pw != "" && !looksPlaceholder(u, pw) {
			return true, h, maskEmail(u)
		}
		if h != "" {
			return false, h, ""
		}
	}
	return false, host, user
}

// looksPlaceholder 占位值探测（.env 模板里的 your-email@qq.com 等）
func looksPlaceholder(user, pass string) bool {
	u, p := strings.ToLower(user), strings.ToLower(pass)
	return strings.Contains(u, "your-email") ||
		strings.Contains(u, "your_email") ||
		strings.Contains(p, "your-smtp") ||
		strings.Contains(p, "your_password") ||
		p == "password"
}

// maskEmail 账号脱敏（admin 接口回显用）：moneyflysubssr@gmail.com → mon***
func maskEmail(s string) string {
	if len(s) <= 3 {
		return "***"
	}
	at := strings.Index(s, "@")
	if at > 3 {
		return s[:3] + "***" + s[at:]
	}
	return s[:3] + "***"
}

// isSMTPConfigured 兼容旧调用
func isSMTPConfigured(db *gorm.DB) bool {
	ok, _, _ := smtpStatus(db)
	return ok
}
