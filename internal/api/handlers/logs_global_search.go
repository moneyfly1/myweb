package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------------------
// 日志全局搜索：一个搜索框，把这个用户的**全部**日志一次性搜出来
//
// 需求（2026-10-19）：日志管理页原来按类型分 7 个 Tab，查一个问题要在注册/订阅/
// 余额/佣金/重置/邮件/操作 里分别搜一遍。现在提供一个统一入口：
//   输入用户名 / 邮箱 / 用户ID（纯数字）/ 订阅 token / IP → 并发查所有日志表 →
//   按时间倒序合并 → 分页返回统一结构（来源类型 + 时间 + 用户 + 动作 + 详情 + IP）。
//
// 实现要点：
//   - 各来源并行查询（goroutine + WaitGroup），单表最多取最近 N 条（默认 300），
//     再在内存里合并排序分页 —— 避免跨表 UNION 的方言差异（sqlite/mysql/pg 写法不同）；
//   - 关键词一律经 utils.SanitizeSearchKeyword + EscapeLikePattern 处理，防注入/通配；
//   - 纯数字关键词额外按 user_id 精确匹配（管理员常见用法：直接搜用户 ID）。
// ---------------------------------------------------------------------------

const globalSearchPerSource = 300

// GlobalLogItem 统一日志行
type GlobalLogItem struct {
	Source    string `json:"source"`       // 机器标识：registration/subscription/balance/…
	SourceLab string `json:"source_label"` // 中文名（前端直接显示）
	ID        uint   `json:"id"`
	Time      string `json:"time"` // 2006-01-02 15:04:05
	UserID    uint   `json:"user_id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Action    string `json:"action"` // 动作/类型
	Detail    string `json:"detail"` // 描述正文
	IP        string `json:"ip"`
	Extra     string `json:"extra"` // 附加信息（金额/状态/错误等）
	unix      int64
}

// GetGlobalLogSearch 日志全局搜索
// GET /api/v1/admin/logs/global-search?q=&page=1&limit=20&sources=
func GetGlobalLogSearch(c *gin.Context) {
	db := database.GetDB()
	raw := strings.TrimSpace(c.Query("q"))
	if raw == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "请输入用户名 / 邮箱 / 用户ID / 订阅 token 或 IP", nil)
		return
	}
	kw := utils.EscapeLikePattern(utils.SanitizeSearchKeyword(raw))
	like := "%" + kw + "%"
	uid, _ := strconv.ParseUint(raw, 10, 64)
	page, limit := parsePageLimit(c, 20, 200)

	// 可选：只看某些来源（逗号分隔）；空 = 全部
	wanted := map[string]bool{}
	for _, s := range strings.Split(c.Query("sources"), ",") {
		s = strings.TrimSpace(strings.ToLower(s))
		if s != "" {
			wanted[s] = true
		}
	}
	pick := func(src string) bool { return len(wanted) == 0 || wanted[src] }

	var mu sync.Mutex
	var wg sync.WaitGroup
	out := make([]GlobalLogItem, 0, 256)
	appendItems := func(items ...GlobalLogItem) {
		mu.Lock()
		out = append(out, items...)
		mu.Unlock()
	}
	fmtTime := func(t time.Time) string {
		return t.Format("2006-01-02 15:04:05")
	}

	// ---- 注册日志 ----
	if pick("registration") {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var rows []models.RegistrationLog
			q := db.Model(&models.RegistrationLog{}).
				Where("username LIKE ? OR email LIKE ? OR COALESCE(invite_code,'') LIKE ? OR COALESCE(ip_address,'') LIKE ?",
					like, like, like, like)
			if uid > 0 {
				q = q.Or("user_id = ?", uid)
			}
			q.Order("created_at DESC").Limit(globalSearchPerSource).Find(&rows)
			items := make([]GlobalLogItem, 0, len(rows))
			for _, r := range rows {
				items = append(items, GlobalLogItem{
					Source: "registration", SourceLab: "注册", ID: r.ID,
					Time: fmtTime(r.CreatedAt), UserID: r.UserID,
					Username: r.Username, Email: r.Email, Action: "注册",
					Detail: orDash(r.RegisterSource.String) + " " + orDash(r.InviteCode.String),
					IP:     r.IPAddress.String, Extra: r.Status, unix: r.CreatedAt.Unix(),
				})
			}
			appendItems(items...)
		}()
	}

	// ---- 订阅日志 ----
	if pick("subscription") {
		wg.Add(1)
		go func() {
			defer wg.Done()
			type row struct {
				ID          uint
				UserID      uint
				Username    string
				Email       string
				ActionType  string
				Description *string
				IPAddress   *string
				CreatedAt   time.Time
				Token       *string
			}
			var rows []row
			q := db.Table("subscription_logs").
				Select("subscription_logs.id, subscription_logs.user_id, users.username, users.email, "+
					"subscription_logs.action_type, subscription_logs.description, subscription_logs.ip_address, "+
					"subscription_logs.created_at, subscriptions.subscription_url AS token").
				Joins("LEFT JOIN users ON users.id = subscription_logs.user_id").
				Joins("LEFT JOIN subscriptions ON subscriptions.id = subscription_logs.subscription_id").
				Where("users.username LIKE ? OR users.email LIKE ? OR COALESCE(users.email,'') LIKE ? OR "+
					"COALESCE(subscriptions.subscription_url,'') LIKE ? OR COALESCE(subscription_logs.description,'') LIKE ?",
					like, like, like, like, like)
			if uid > 0 {
				q = q.Or("subscription_logs.user_id = ?", uid)
			}
			q.Order("subscription_logs.created_at DESC").Limit(globalSearchPerSource).Scan(&rows)
			items := make([]GlobalLogItem, 0, len(rows))
			for _, r := range rows {
				items = append(items, GlobalLogItem{
					Source: "subscription", SourceLab: "订阅", ID: r.ID,
					Time: fmtTime(r.CreatedAt), UserID: r.UserID,
					Username: r.Username, Email: r.Email, Action: r.ActionType,
					Detail: strOr(r.Description, ""), IP: strOr(r.IPAddress, ""),
					unix: r.CreatedAt.Unix(),
				})
			}
			appendItems(items...)
		}()
	}

	// ---- 余额日志 ----
	if pick("balance") {
		wg.Add(1)
		go func() {
			defer wg.Done()
			type row struct {
				ID          uint
				UserID      uint
				Username    string
				Email       string
				ChangeType  string
				Amount      float64
				Description *string
				IPAddress   *string
				CreatedAt   time.Time
			}
			var rows []row
			q := db.Table("balance_logs").
				Select("balance_logs.id, balance_logs.user_id, users.username, users.email, balance_logs.change_type, "+
					"balance_logs.amount, balance_logs.description, balance_logs.ip_address, balance_logs.created_at").
				Joins("LEFT JOIN users ON users.id = balance_logs.user_id").
				Where("users.username LIKE ? OR users.email LIKE ? OR COALESCE(balance_logs.description,'') LIKE ?",
					like, like, like)
			if uid > 0 {
				q = q.Or("balance_logs.user_id = ?", uid)
			}
			q.Order("balance_logs.created_at DESC").Limit(globalSearchPerSource).Scan(&rows)
			items := make([]GlobalLogItem, 0, len(rows))
			for _, r := range rows {
				items = append(items, GlobalLogItem{
					Source: "balance", SourceLab: "余额", ID: r.ID,
					Time: fmtTime(r.CreatedAt), UserID: r.UserID,
					Username: r.Username, Email: r.Email, Action: r.ChangeType,
					Detail: strOr(r.Description, ""), IP: strOr(r.IPAddress, ""),
					Extra: fmt.Sprintf("%.2f", r.Amount), unix: r.CreatedAt.Unix(),
				})
			}
			appendItems(items...)
		}()
	}

	// ---- 佣金日志 ----
	if pick("commission") {
		wg.Add(1)
		go func() {
			defer wg.Done()
			type row struct {
				ID             uint
				InviterID      uint
				InviteeID      uint
				InviterName    *string
				InviterEmail   *string
				InviteeName    *string
				InviteeEmail   *string
				CommissionType string
				Amount         float64
				Status         string
				Description    *string
				CreatedAt      time.Time
			}
			var rows []row
			q := db.Table("commission_logs").
				Select("commission_logs.id, commission_logs.inviter_id, commission_logs.invitee_id, "+
					"iu.username AS inviter_name, iu.email AS inviter_email, "+
					"eu.username AS invitee_name, eu.email AS invitee_email, "+
					"commission_logs.commission_type, commission_logs.amount, commission_logs.status, "+
					"commission_logs.description, commission_logs.created_at").
				Joins("LEFT JOIN users iu ON iu.id = commission_logs.inviter_id").
				Joins("LEFT JOIN users eu ON eu.id = commission_logs.invitee_id").
				Where("iu.username LIKE ? OR iu.email LIKE ? OR eu.username LIKE ? OR eu.email LIKE ?",
					like, like, like, like)
			if uid > 0 {
				q = q.Or("commission_logs.inviter_id = ? OR commission_logs.invitee_id = ?", uid, uid)
			}
			q.Order("commission_logs.created_at DESC").Limit(globalSearchPerSource).Scan(&rows)
			items := make([]GlobalLogItem, 0, len(rows))
			for _, r := range rows {
				items = append(items, GlobalLogItem{
					Source: "commission", SourceLab: "佣金", ID: r.ID,
					Time: fmtTime(r.CreatedAt), UserID: r.InviterID,
					Username: strOr(r.InviterName, ""), Email: strOr(r.InviterEmail, ""),
					Action: r.CommissionType,
					Detail: fmt.Sprintf("被邀请人 %s（%s）%s", strOr(r.InviteeName, ""), strOr(r.InviteeEmail, ""),
						strOr(r.Description, "")),
					Extra: fmt.Sprintf("%.2f %s", r.Amount, r.Status), unix: r.CreatedAt.Unix(),
				})
			}
			appendItems(items...)
		}()
	}

	// ---- 重置订阅日志 ----
	if pick("reset") {
		wg.Add(1)
		go func() {
			defer wg.Done()
			type row struct {
				ID        uint
				UserID    uint
				Username  *string
				Email     *string
				ResetType string
				Reason    string
				ResetBy   *string
				CreatedAt time.Time
			}
			var rows []row
			q := db.Table("subscription_resets").
				Select("subscription_resets.id, subscription_resets.user_id, users.username, users.email, "+
					"subscription_resets.reset_type, subscription_resets.reason, subscription_resets.reset_by, "+
					"subscription_resets.created_at").
				Joins("LEFT JOIN users ON users.id = subscription_resets.user_id").
				Where("users.username LIKE ? OR users.email LIKE ? OR subscription_resets.reason LIKE ?",
					like, like, like)
			if uid > 0 {
				q = q.Or("subscription_resets.user_id = ?", uid)
			}
			q.Order("subscription_resets.created_at DESC").Limit(globalSearchPerSource).Scan(&rows)
			items := make([]GlobalLogItem, 0, len(rows))
			for _, r := range rows {
				items = append(items, GlobalLogItem{
					Source: "reset", SourceLab: "重置订阅", ID: r.ID,
					Time: fmtTime(r.CreatedAt), UserID: r.UserID,
					Username: strOr(r.Username, ""), Email: strOr(r.Email, ""),
					Action: r.ResetType, Detail: r.Reason,
					Extra: "操作人: " + strOr(r.ResetBy, "系统"), unix: r.CreatedAt.Unix(),
				})
			}
			appendItems(items...)
		}()
	}

	// ---- 邮件日志 ----
	if pick("email") {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var rows []models.EmailQueue
			db.Where("to_email LIKE ? OR subject LIKE ?", like, like).
				Order("created_at DESC").Limit(globalSearchPerSource).Find(&rows)
			items := make([]GlobalLogItem, 0, len(rows))
			for _, r := range rows {
				items = append(items, GlobalLogItem{
					Source: "email", SourceLab: "邮件", ID: r.ID,
					Time: fmtTime(r.CreatedAt), Email: r.ToEmail,
					Action: r.EmailType, Detail: r.Subject,
					Extra: r.Status, unix: r.CreatedAt.Unix(),
				})
			}
			appendItems(items...)
		}()
	}

	// ---- 管理员操作日志 ----
	if pick("audit") {
		wg.Add(1)
		go func() {
			defer wg.Done()
			type row struct {
				ID                uint
				UserID            *int64
				Username          *string
				Email             *string
				ActionType        string
				ActionDescription *string
				IPAddress         *string
				RequestPath       *string
				CreatedAt         time.Time
			}
			var rows []row
			q := db.Table("audit_logs").
				Select("audit_logs.id, audit_logs.user_id, users.username, users.email, audit_logs.action_type, "+
					"audit_logs.action_description, audit_logs.ip_address, audit_logs.request_path, audit_logs.created_at").
				Joins("LEFT JOIN users ON users.id = audit_logs.user_id").
				Where("users.username LIKE ? OR users.email LIKE ? OR COALESCE(audit_logs.action_description,'') LIKE ? OR "+
					"COALESCE(audit_logs.request_path,'') LIKE ? OR COALESCE(audit_logs.ip_address,'') LIKE ?",
					like, like, like, like, like)
			if uid > 0 {
				q = q.Or("audit_logs.user_id = ?", uid)
			}
			q.Order("audit_logs.created_at DESC").Limit(globalSearchPerSource).Scan(&rows)
			items := make([]GlobalLogItem, 0, len(rows))
			for _, r := range rows {
				var userID uint
				if r.UserID != nil {
					userID = uint(*r.UserID)
				}
				items = append(items, GlobalLogItem{
					Source: "audit", SourceLab: "管理员操作", ID: r.ID,
					Time: fmtTime(r.CreatedAt), UserID: userID,
					Username: strOr(r.Username, ""), Email: strOr(r.Email, ""),
					Action: r.ActionType, Detail: strOr(r.ActionDescription, ""),
					IP: strOr(r.IPAddress, ""), Extra: strOr(r.RequestPath, ""),
					unix: r.CreatedAt.Unix(),
				})
			}
			appendItems(items...)
		}()
	}

	// ---- 登录记录（含失败）----
	if pick("login") {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var rows []models.LoginAttempt
			db.Where("username LIKE ? OR COALESCE(ip_address,'') LIKE ?", like, like).
				Order("created_at DESC").Limit(globalSearchPerSource).Find(&rows)
			items := make([]GlobalLogItem, 0, len(rows))
			for _, r := range rows {
				status := "登录失败"
				if r.Success {
					status = "登录成功"
				}
				items = append(items, GlobalLogItem{
					Source: "login", SourceLab: "登录", ID: r.ID,
					Time: fmtTime(r.CreatedAt), Username: r.Username,
					Action: status, Detail: strings.TrimSpace(r.UserAgent.String),
					IP: r.IPAddress.String, unix: r.CreatedAt.Unix(),
				})
			}
			appendItems(items...)
		}()
	}

	wg.Wait()

	// 合并排序（时间倒序；sort.SliceStable 保持同一时刻的来源顺序）
	items := out
	sort.SliceStable(items, func(i, j int) bool { return items[i].unix > items[j].unix })

	total := len(items)
	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	paged := make([]GlobalLogItem, 0, end-start)
	for _, it := range items[start:end] {
		// unix 只用于排序，不输出
		it.unix = 0
		paged = append(paged, it)
	}

	// 来源统计（前端展示"命中分布在哪些类型"）
	bySource := map[string]int{}
	for _, it := range items {
		bySource[it.Source]++
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"query":     raw,
		"total":     total,
		"page":      page,
		"limit":     limit,
		"by_source": bySource,
		"items":     paged,
	})
}

func strOr(p *string, def string) string {
	if p == nil {
		return def
	}
	if strings.TrimSpace(*p) == "" {
		return def
	}
	return *p
}

func orDash(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	return s
}
