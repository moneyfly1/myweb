package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/core/paths"
	"cboard-go/internal/services/domainpool"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

// 订阅域名池管理（系统设置 → 订阅域名池）
//
// 背景（2026-09-23）：官网域名在部分地区被屏蔽，需要经常更换订阅域名。
// 这里把「建 nginx 站点 → 签证书（含自动续期钩子）→ 重载 nginx → 写面板配置」
// 收敛成一个按钮，并逐步返回执行结果，避免管理员漏步骤导致「换了域名客户还是连不上」。
//
// 域名不生效时的排查顺序（界面里也按这个顺序展示）：
//  1. DNS 是否指向本服务器；
//  2. 站点配置（nginx vhost）是否存在、校验通过；
//  3. 证书是否签发、是否会自动续期；
//  4. HTTPS 自检是否返回面板接口数据；
//  5. 面板配置（主/备用域名）是否已写入。

func domainPoolManager() *domainpool.Manager {
	siteDomain := ""
	acmeEmail := ""
	if db := database.GetDB(); db != nil {
		siteDomain = utils.GetDomainFromDB(db)
		// 全新服务器上 certbot 尚未注册 ACME 账户，首次签发必须带邮箱，
		// 否则会 MissingCommandlineFlag 失败（表现为一键配置卡在签发证书）
		acmeEmail = utils.GetACMEEmailFromDB(db)
	}
	// 面板站点根目录 = 进程工作目录（与 backup/repo_sync/config 等模块一致）。
	// 之前这里传空字符串，导致 staticRootOf 只能命中写死的旧站点路径，
	// 换域名/迁移后新域名建出来的站点「只有 API、没有前端界面」。
	panelRoot := ""
	if wd, err := os.Getwd(); err == nil {
		panelRoot = wd
	}
	m := domainpool.New(panelRoot, siteDomain)
	m.Email = acmeEmail
	return m
}

// 域名池的读写与体检列表统一放在 services/domainpool 里（SSH 命令行工具也用同一套）。
func currentPool() (string, []string) {
	return domainpool.PoolFromConfig(database.GetDB())
}

func writePool(primary string, backups []string) error {
	return domainpool.WritePool(database.GetDB(), primary, backups)
}

func poolDomainsForInspect(primary string, backups []string, siteDomain string) []string {
	return domainpool.DomainsForInspect(primary, backups, siteDomain)
}

// GetDomainPool 订阅域名池：配置 + 每个域名的实时检测（DNS / 站点 / 证书 / 续期 / HTTPS 自检）
// GET /admin/domains/pool
func GetDomainPool(c *gin.Context) {
	primary, backups := currentPool()
	siteDomain := ""
	if db := database.GetDB(); db != nil {
		siteDomain = utils.GetDomainFromDB(db)
	}
	m := domainPoolManager()
	available, why := m.Available()

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()
	items := m.InspectMany(ctx, poolDomainsForInspect(primary, backups, siteDomain), primary, siteDomain)

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"available":   available,
		"unavailable": why,
		"primary":     primary,
		"backups":     backups,
		"site_domain": siteDomain,
		"items":       items,
		// 暴露本机路径解析结果（nginx/vhost 目录/ACME webroot/前端产物目录）：
		// 换机器后如果「一键配置」不可用或站点没界面，管理员在这里就能看到
		// 究竟探测到了哪个路径、来自环境变量还是自动探测，不必再翻服务器。
		"resolved_paths": paths.Describe(siteDomain),
	})
}

// ConfigureDomainPool 一键配置（新增或修复）一个订阅域名，可选设为订阅主域名
// POST /admin/domains/pool/configure {domain, make_primary}
func ConfigureDomainPool(c *gin.Context) {
	var req struct {
		Domain      string `json:"domain" binding:"required"`
		MakePrimary bool   `json:"make_primary"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}
	domain, err := domainpool.ValidateDomain(req.Domain)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	m := domainPoolManager()
	ctx, cancel := context.WithTimeout(c.Request.Context(), 300*time.Second)
	defer cancel()

	st, steps, cfgErr := m.Configure(ctx, domain)
	if cfgErr != nil {
		utils.CreateAuditLogSimple(c, "domain_pool_configure_failed", "system", 0,
			fmt.Sprintf("管理员操作: 配置订阅域名 %s 失败：%v", domain, cfgErr))
		utils.ErrorResponseWithData(c, http.StatusInternalServerError,
			"配置失败："+cfgErr.Error(), "", gin.H{"steps": steps, "item": st})
		return
	}

	primary, backups := currentPool()
	if req.MakePrimary || primary == "" {
		primary = domain
	}
	backups = append(backups, domain)
	if err := writePool(primary, backups); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "站点已配好，但写入面板配置失败："+err.Error(), err)
		return
	}
	utils.CreateAuditLogSimple(c, "domain_pool_configure", "system", 0,
		fmt.Sprintf("管理员操作: 配置订阅域名 %s（订阅主域名=%s）", domain, primary))

	utils.SuccessResponse(c, http.StatusOK, "配置完成", gin.H{
		"steps":   steps,
		"item":    st,
		"primary": primary,
		"backups": backups,
	})
}

// SetDomainPoolPrimary 把某个域名设为订阅主域名（同时保留在备用列表里，便于随时切回）
// POST /admin/domains/pool/primary {domain}
func SetDomainPoolPrimary(c *gin.Context) {
	var req struct {
		Domain string `json:"domain" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}
	domain, err := domainpool.ValidateDomain(req.Domain)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	_, backups := currentPool()
	backups = append(backups, domain)
	if err := writePool(domain, backups); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "写入失败："+err.Error(), err)
		return
	}
	utils.CreateAuditLogSimple(c, "domain_pool_set_primary", "system", 0,
		fmt.Sprintf("管理员操作: 订阅主域名改为 %s", domain))
	utils.SuccessResponse(c, http.StatusOK, "已切换订阅主域名（旧主域名仍保留为备用地址，客户无需换链接）",
		gin.H{"primary": domain})
}

// RemoveDomainFromPool 从池里移除域名并移除其 nginx 站点配置（证书保留，便于换回来）
// POST /admin/domains/pool/remove {domain}
func RemoveDomainFromPool(c *gin.Context) {
	var req struct {
		Domain string `json:"domain" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}
	domain, err := domainpool.ValidateDomain(req.Domain)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	siteDomain := ""
	if db := database.GetDB(); db != nil {
		siteDomain = utils.GetDomainFromDB(db)
	}
	if siteDomain != "" && strings.EqualFold(domain, siteDomain) {
		utils.ErrorResponse(c, http.StatusBadRequest, "这是网站域名，不能从订阅域名池移除", nil)
		return
	}

	primary, backups := currentPool()
	// 移除主域名时自动降级：优先换到池内其它域名，否则回退网站域名
	if strings.EqualFold(domain, primary) {
		next := siteDomain
		for _, b := range backups {
			if !strings.EqualFold(b, domain) {
				next = b
				break
			}
		}
		primary = next
	}
	rest := make([]string, 0, len(backups))
	for _, b := range backups {
		if !strings.EqualFold(b, domain) {
			rest = append(rest, b)
		}
	}
	if err := writePool(primary, rest); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "写入失败："+err.Error(), err)
		return
	}

	m := domainPoolManager()
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()
	steps, rmErr := m.RemoveVhost(ctx, domain)
	warn := ""
	if rmErr != nil {
		warn = "站点配置移除有告警：" + rmErr.Error()
	}
	utils.CreateAuditLogSimple(c, "domain_pool_remove", "system", 0,
		fmt.Sprintf("管理员操作: 从订阅域名池移除 %s（订阅主域名=%s）", domain, primary))

	utils.SuccessResponse(c, http.StatusOK, "已移除（证书保留，随时可以一键加回来）", gin.H{
		"steps":   steps,
		"primary": primary,
		"backups": rest,
		"warn":    warn,
	})
}

// RenewDomainPool 手动续期：对域名池涉及的证书跑 certbot renew 并重载 nginx
// POST /admin/domains/pool/renew {force?}
//
// 自动续期本来由 certbot 定时任务负责（面板已展示剩余天数与「自动续期 ✓」）；
// 这个接口用于两种情况：① 证书临近到期想立刻确认能续；② 自动续期失败需要手动补一次。
// force=true 表示未到期也重签 —— Let's Encrypt 对同一组域名重复签发有每周次数限制，
// 界面上只在「证书异常」时才引导使用。
func RenewDomainPool(c *gin.Context) {
	var req struct {
		Force bool `json:"force"`
	}
	// 允许空 body（默认不强制）
	_ = c.ShouldBindJSON(&req)

	m := domainPoolManager()
	if ok, why := m.Available(); !ok {
		utils.ErrorResponse(c, http.StatusServiceUnavailable, "本服务器未开启一键配置："+why, nil)
		return
	}
	primary, backups := currentPool()
	siteDomain := ""
	if db := database.GetDB(); db != nil {
		siteDomain = utils.GetDomainFromDB(db)
	}
	domains := domainpool.DomainsForInspect(primary, backups, siteDomain)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 300*time.Second)
	defer cancel()
	res, steps, rerr := m.Renew(ctx, domains, req.Force)
	if rerr != nil {
		// 失败也要把「已经走了哪几步、哪些证书成功/跳过/失败」带回前端，
		// 否则管理员只看到一句错误，不知道该点修复还是手工处理
		utils.ErrorResponseWithData(c, http.StatusInternalServerError,
			"partial_failure", "续期未全部成功："+rerr.Error(),
			gin.H{"steps": steps, "result": res})
		return
	}

	msg := "证书均未到期，无需续期（到期前 30 天起由 certbot 自动续期）"
	if len(res.Renewed) > 0 {
		msg = fmt.Sprintf("已续期 %d 张证书，nginx 已重载", len(res.Renewed))
		if req.Force {
			msg += "（强制续期）"
		}
	}
	utils.CreateAuditLogSimple(c, "domain_pool_renew", "system", 0,
		fmt.Sprintf("管理员操作: 订阅域名证书续期 renewed=%v not_due=%v force=%v",
			res.Renewed, res.NotDue, req.Force))
	utils.SuccessResponse(c, http.StatusOK, msg, gin.H{"steps": steps, "result": res})
}
