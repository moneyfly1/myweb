package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/ghrelease"
	"cboard-go/internal/services/pan123"
	"cboard-go/internal/services/software_sync"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

const (
	pan123Category = "pan123"
	panLinkTTL     = 10 * time.Minute
)

func init() {
	// 同步完成后清理直链缓存，避免旧链接（指向已删除文件）在缓存期内继续生效
	software_sync.SetOnSyncComplete(clearPanLinkCache)
}

// 公共解析接口全局限流：防止滥用（每次请求会消耗管理员的云盘接口额度）
var resolveGate = struct {
	mu   sync.Mutex
	last time.Time
}{}

// resolveThrottled 全局限流（约 5 次/秒）；超限返回 false
func resolveThrottled() bool {
	resolveGate.mu.Lock()
	defer resolveGate.mu.Unlock()
	if time.Since(resolveGate.last) < 200*time.Millisecond {
		return false
	}
	resolveGate.last = time.Now()
	return true
}

// isPanManagedLink 判断是否为工具写入的 123 云盘直链（静态模式产物）。
// 用于区分"管理员手工自定义链接"（应保护）与"工具写入的静态直链"（可被重新接管）。
func isPanManagedLink(u string) bool {
	return strings.Contains(u, "123295.com") || strings.Contains(u, "cjjd19.com")
}

// ---------------------------------------------------------------------------
// 配置读写
// ---------------------------------------------------------------------------

type pan123Config struct {
	Username string            `json:"username"`
	Password string            `json:"password"`
	SharePwd string            `json:"share_pwd"`
	Mode     string            `json:"mode"`
	Cookies  string            `json:"cookies"`
	Token    string            `json:"token"`
	FileMap  map[string]string `json:"file_map"`
	// FileExts 每个软件配置键 → 扩展名过滤（如 exe/dmg/apk），空表示不过滤
	FileExts map[string]string `json:"file_exts"`
}

func loadPan123Config() (pan123Config, error) {
	var cfg pan123Config
	cfg.Mode = "direct" // 直链为主要取链方式（已实测验证）；分享链接为实验性
	cfg.FileMap = map[string]string{}
	cfg.FileExts = map[string]string{}
	db := database.GetDB()
	var configs []models.SystemConfig
	if err := db.Where("category = ?", pan123Category).Find(&configs).Error; err != nil {
		return cfg, err
	}
	for _, c := range configs {
		switch c.Key {
		case "username":
			cfg.Username = c.Value
		case "password":
			cfg.Password = c.Value
		case "share_pwd":
			cfg.SharePwd = c.Value
		case "mode":
			if c.Value != "" {
				cfg.Mode = c.Value
			}
		case "cookies":
			cfg.Cookies = c.Value
		case "token":
			cfg.Token = c.Value
		case "file_map":
			if strings.TrimSpace(c.Value) != "" {
				_ = json.Unmarshal([]byte(c.Value), &cfg.FileMap)
			}
		case "file_exts":
			if strings.TrimSpace(c.Value) != "" {
				_ = json.Unmarshal([]byte(c.Value), &cfg.FileExts)
			}
		}
	}
	return cfg, nil
}

func savePan123ConfigValue(key, value string) error {
	db := database.GetDB()
	var conf models.SystemConfig
	if err := db.Where("key = ? AND category = ?", key, pan123Category).FirstOrInit(&conf).Error; err != nil {
		return err
	}
	conf.Key = key
	conf.Category = pan123Category
	conf.Value = value
	conf.Type = "text"
	return db.Save(&conf).Error
}

func maskIfNonEmpty(v string) string {
	if strings.TrimSpace(v) == "" {
		return ""
	}
	return maskedSecretValue
}

// GetPan123Config 获取 123 云盘配置（密码字段脱敏）
func GetPan123Config(c *gin.Context) {
	cfg, err := loadPan123Config()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "读取配置失败", err)
		return
	}
	syncEnabled := true
	syncInterval := 12
	db := database.GetDB()
	var sc []models.SystemConfig
	if err := db.Where("category = ?", pan123Category).Find(&sc).Error; err == nil {
		for _, c := range sc {
			switch c.Key {
			case "sync_enabled":
				syncEnabled = c.Value == "" || c.Value == "true" || c.Value == "1"
			case "sync_interval_hours":
				if v, err2 := strconv.Atoi(c.Value); err2 == nil && v >= 1 {
					syncInterval = v
				}
			}
		}
	}
	// 已同步文件映射（仅文件名，用于前端回填关键词输入框）
	fileIDMap, _ := software_sync.LoadFileIDMap()
	syncedFiles := map[string]string{}
	for key, entry := range fileIDMap {
		if entry.FileId > 0 {
			syncedFiles[key] = entry.FileName
		}
	}
	tokenExpiry := pan123.ParseTokenExpiry(cfg.Token)
	expiryText := ""
	daysLeft := 0
	if !tokenExpiry.IsZero() {
		expiryText = tokenExpiry.Format(time.RFC3339)
		daysLeft = int(time.Until(tokenExpiry).Hours() / 24)
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"username":   cfg.Username,
		"password":   maskIfNonEmpty(cfg.Password),
		"share_pwd":  maskIfNonEmpty(cfg.SharePwd),
		"mode":       cfg.Mode,
		"cookies":    maskIfNonEmpty(cfg.Cookies),
		"token":      maskIfNonEmpty(cfg.Token),
		"file_map":   cfg.FileMap,
		"file_exts":  cfg.FileExts,
		"sync_enabled":        syncEnabled,
		"sync_interval_hours": syncInterval,
		"synced_files":        syncedFiles,
		"token_expiry":        expiryText,
		"token_days_left":     daysLeft,
		"configured": pan123AuthConfigured(cfg),
	})
}

// pan123AuthConfigured 判断是否已配置任意一种登录方式
func pan123AuthConfigured(cfg pan123Config) bool {
	return cfg.Token != "" || cfg.Cookies != "" ||
		(strings.TrimSpace(cfg.Username) != "" && cfg.Password != "")
}

// SavePan123Config 保存 123 云盘配置
func SavePan123Config(c *gin.Context) {
	var req struct {
		Username          string            `json:"username"`
		Password          string            `json:"password"`
		SharePwd          string            `json:"share_pwd"`
		Mode              string            `json:"mode"`
		Cookies           string            `json:"cookies"`
		Token             string            `json:"token"`
		FileMap           map[string]string `json:"file_map"`
		FileExts          map[string]string `json:"file_exts"`
		SyncEnabled       *bool             `json:"sync_enabled"`
		SyncIntervalHours *int              `json:"sync_interval_hours"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	cfg, _ := loadPan123Config()

	// 敏感字段收到脱敏占位符时保留旧值
	password := strings.TrimSpace(req.Password)
	if password == "" || password == maskedSecretValue {
		password = cfg.Password
	}
	sharePwd := strings.TrimSpace(req.SharePwd)
	if sharePwd == maskedSecretValue {
		sharePwd = cfg.SharePwd
	}
	cookies := strings.TrimSpace(req.Cookies)
	if cookies == maskedSecretValue {
		cookies = cfg.Cookies
	}
	token := strings.TrimSpace(req.Token)
	if token == maskedSecretValue {
		token = cfg.Token
	}

	mode := strings.TrimSpace(req.Mode)
	if mode != "direct" && mode != "share" {
		mode = "share"
	}

	values := map[string]string{
		"username": strings.TrimSpace(req.Username),
		"password": password,
		"share_pwd": sharePwd,
		"mode":     mode,
		"cookies":  cookies,
		"token":    token,
	}
	fileMapJSON := ""
	if len(req.FileMap) > 0 {
		b, _ := json.Marshal(req.FileMap)
		fileMapJSON = string(b)
	}
	fileExtsJSON := ""
	if len(req.FileExts) > 0 {
		b, _ := json.Marshal(req.FileExts)
		fileExtsJSON = string(b)
	}
	values["file_map"] = fileMapJSON
	values["file_exts"] = fileExtsJSON

	if req.SyncEnabled != nil {
		values["sync_enabled"] = strconv.FormatBool(*req.SyncEnabled)
	}
	if req.SyncIntervalHours != nil && *req.SyncIntervalHours >= 1 && *req.SyncIntervalHours <= 168 {
		values["sync_interval_hours"] = strconv.Itoa(*req.SyncIntervalHours)
	}

	for key, value := range values {
		if err := savePan123ConfigValue(key, value); err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "保存失败", err)
			return
		}
	}

	utils.InvalidateAllSettingCache()
	utils.CreateAuditLogSimple(c, "pan123_config_save", "settings", 0, "管理员操作: 保存 123 云盘自动填充配置")
	utils.SuccessResponse(c, http.StatusOK, "保存成功", nil)
}

// ---------------------------------------------------------------------------
// 连接测试 / 搜索
// ---------------------------------------------------------------------------

func newPan123ClientFromConfig(cfg pan123Config) (*pan123.Client, error) {
	switch {
	case cfg.Token != "":
		return pan123.NewWithToken(cfg.Token, cfg.SharePwd), nil
	case cfg.Cookies != "":
		return pan123.NewWithCookies(cfg.Cookies, cfg.SharePwd), nil
	case strings.TrimSpace(cfg.Username) != "" && cfg.Password != "":
		return pan123.New(cfg.Username, cfg.Password, cfg.SharePwd), nil
	case strings.TrimSpace(cfg.Username) != "":
		return nil, fmt.Errorf("已配置账号但缺少密码")
	default:
		return nil, fmt.Errorf("未配置 123 云盘登录信息（账号密码 / Cookie / Token 任选其一）")
	}
}

// Pan123Test 测试 123 云盘连接
func Pan123Test(c *gin.Context) {
	cfg, err := loadPan123Config()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "读取配置失败", err)
		return
	}
	client, err := newPan123ClientFromConfig(cfg)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	info, err := client.Test()
	if err != nil {
		claims := pan123.SummaryClaims(cfg.Token)
		utils.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("连接失败: %s（token claims: %s）", err.Error(), claims), nil)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "连接成功", gin.H{
		"nickname": info["nickname"],
		"vip_type": info["vipType"],
		"mode":     cfg.Mode,
	})
}

// Pan123Search 按关键词搜索云盘文件（可选扩展名过滤）
func Pan123Search(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("keyword"))
	if keyword == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "缺少 keyword 参数", nil)
		return
	}
	ext := strings.TrimSpace(c.Query("ext"))
	limit := 20
	if v := c.Query("limit"); v != "" {
		fmt.Sscanf(v, "%d", &limit)
	}

	cfg, err := loadPan123Config()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "读取配置失败", err)
		return
	}
	client, err := newPan123ClientFromConfig(cfg)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	files, err := client.SearchFilesByExt(keyword, ext, limit)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "搜索失败: "+err.Error(), nil)
		return
	}

	list := make([]gin.H, 0, len(files))
	for _, f := range files {
		list = append(list, gin.H{
			"file_id":   f.FileID,
			"file_name": f.FileName,
			"size":      f.Size,
			"size_text": f.DisplaySize(),
			"type":      f.Type,
			"update_at": f.UpdateAt,
		})
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{"list": list, "total": len(list)})
}

// ---------------------------------------------------------------------------
// 一键生成并填充
// ---------------------------------------------------------------------------

func pickDownloadFile(files []pan123.File) (*pan123.File, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("未找到匹配的文件")
	}
	for i := range files {
		if files[i].Type == 0 {
			return &files[i], nil
		}
	}
	return nil, fmt.Errorf("搜索到的都是文件夹，请改用具体文件名关键词")
}

type panRefreshItem struct {
	Key      string `json:"key"`
	Keyword  string `json:"keyword"`
	Ext      string `json:"ext,omitempty"`
	OK       bool   `json:"ok"`
	FileName string `json:"file_name,omitempty"`
	SizeText string `json:"size_text,omitempty"`
	URL      string `json:"url,omitempty"`
	Error    string `json:"error,omitempty"`
}

// Pan123Refresh 对每个软件关键词搜索云盘并生成链接，写入软件下载配置。
// dynamic=true 时写入 pan://<关键词>（前端点击时实时解析）；否则写入静态链接。
func Pan123Refresh(c *gin.Context) {
	var req struct {
		Mode     string            `json:"mode"`
		Dynamic  *bool             `json:"dynamic"`
		FileMap  map[string]string `json:"file_map"`
		FileExts map[string]string `json:"file_exts"`
	}
	_ = c.ShouldBindJSON(&req)

	if software_sync.IsRunning() {
		utils.ErrorResponse(c, http.StatusConflict, "软件库同步正在进行中，请稍后再试", nil)
		return
	}

	cfg, err := loadPan123Config()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "读取配置失败", err)
		return
	}
	// 请求体携带的映射优先（前端一键生成前会把当前表格状态一起发来，避免读到旧配置）
	for key, value := range req.FileMap {
		cfg.FileMap[key] = strings.TrimSpace(value)
	}
	for key, value := range req.FileExts {
		cfg.FileExts[key] = strings.TrimSpace(value)
	}
	client, err := newPan123ClientFromConfig(cfg)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	mode := strings.TrimSpace(req.Mode)
	if mode != "direct" && mode != "share" {
		mode = cfg.Mode
	}
	dynamic := true
	if req.Dynamic != nil {
		dynamic = *req.Dynamic
	}

	keys := make([]string, 0, len(cfg.FileMap))
	for k := range cfg.FileMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	report := make([]panRefreshItem, 0, len(keys))
	updates := map[string]string{}
	// 该入口已配置自定义外部链接（非 pan://）→ 不接管，避免误覆盖（如 Clash for Windows 用自定义链接）
	softwareConfig := getConfigMap("software")
	for _, key := range keys {
		item := panRefreshItem{Key: key, Keyword: strings.TrimSpace(cfg.FileMap[key])}
		item.Ext = strings.ToLower(strings.TrimSpace(cfg.FileExts[key]))
		if current := strings.TrimSpace(softwareConfig[key]); current != "" && !strings.HasPrefix(current, "pan://") && !isPanManagedLink(current) {
			item.Error = "该入口使用自定义链接，跳过"
			report = append(report, item)
			continue
		}
		if item.Keyword == "" {
			item.Error = "未填写关键词，已跳过"
			report = append(report, item)
			continue
		}
		files, serr := client.SearchFilesByExt(item.Keyword, item.Ext, 10)
		if serr != nil {
			item.Error = "搜索失败: " + serr.Error()
			report = append(report, item)
			continue
		}
		if item.Ext != "" && len(files) == 0 {
			item.Error = fmt.Sprintf("未找到以 .%s 结尾的文件，请检查关键词或扩展名", item.Ext)
			report = append(report, item)
			continue
		}
		file, ferr := pickDownloadFile(files)
		if ferr != nil {
			item.Error = ferr.Error()
			report = append(report, item)
			continue
		}
		link, lerr := client.GetDownloadLink(*file, mode)
		if lerr != nil {
			item.Error = lerr.Error()
			report = append(report, item)
			continue
		}
		item.OK = true
		item.FileName = file.FileName
		item.SizeText = file.DisplaySize()
		item.URL = link

		if dynamic {
			// 动态链接统一写 pan://<配置键>：解析时优先走 fileId（云盘同步），
			// 无需依赖关键词输入框；未同步时后端会自动按"键即关键词"回退搜索。
			updates[key] = "pan://" + key
		} else {
			updates[key] = link
		}
		report = append(report, item)
	}

	// 写入软件配置（category=software）
	db := database.GetDB()
	for key, value := range updates {
		var conf models.SystemConfig
		if err := db.Where("key = ? AND category = ?", key, "software").FirstOrInit(&conf).Error; err != nil {
			continue
		}
		conf.Key = key
		conf.Category = "software"
		conf.Value = value
		conf.Type = "text"
		_ = db.Save(&conf).Error
	}

	clearPanLinkCache()
	utils.InvalidateAllSettingCache()
	utils.CreateAuditLogSimple(c, "pan123_refresh", "settings", 0, fmt.Sprintf("管理员操作: 123云盘自动填充 %d 个软件（模式=%s）", len(updates), mode))

	successCount := 0
	for _, r := range report {
		if r.OK {
			successCount++
		}
	}
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("完成：成功 %d 个，失败 %d 个", successCount, len(report)-successCount), gin.H{
		"mode":    mode,
		"dynamic": dynamic,
		"report":  report,
	})
}

// ---------------------------------------------------------------------------
// 用户端动态解析：/api/v1/download/pan
// ---------------------------------------------------------------------------

type panLinkCacheEntry struct {
	URL    string
	Expire time.Time
}

var panLinkCache sync.Map // key: mode|keyword → entry

func getCachedPanLink(key string) (string, bool) {
	if v, ok := panLinkCache.Load(key); ok {
		e := v.(panLinkCacheEntry)
		if time.Now().Before(e.Expire) {
			return e.URL, true
		}
		panLinkCache.Delete(key)
	}
	return "", false
}

func setCachedPanLink(key, rawURL string) {
	panLinkCache.Store(key, panLinkCacheEntry{URL: rawURL, Expire: time.Now().Add(panLinkTTL)})
}

func clearPanLinkCache() {
	panLinkCache.Range(func(k, _ interface{}) bool {
		panLinkCache.Delete(k)
		return true
	})
}

// fileIDMapOf 读取同步维护的文件映射
func fileIDMapOf(key string) (software_sync.FileEntry, bool) {
	m, err := software_sync.LoadFileIDMap()
	if err != nil {
		return software_sync.FileEntry{}, false
	}
	e, ok := m[key]
	return e, ok
}

func i64s(v int64) string {
	return strconv.FormatInt(v, 10)
}

// Pan123Resolve 根据软件配置 key 或关键词，实时生成最新的下载链接并 302 跳转。
// 支持 ?key=<softwareConfigKey> 或 ?q=<关键词>
func Pan123Resolve(c *gin.Context) {
	if !resolveThrottled() {
		utils.ErrorResponse(c, http.StatusTooManyRequests, "请求过于频繁，请稍后再试", nil)
		return
	}
	cfg, err := loadPan123Config()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "读取配置失败", err)
		return
	}
	if !pan123AuthConfigured(cfg) {
		utils.ErrorResponse(c, http.StatusBadRequest, "管理员尚未配置 123 云盘登录信息", nil)
		return
	}

	keyword := ""
	ext := ""
	rawKey := ""
	if key := strings.TrimSpace(c.Query("key")); key != "" {
		rawKey = key
		keyword = strings.TrimSpace(cfg.FileMap[key])
		ext = strings.TrimSpace(cfg.FileExts[key])
		// 优先走同步任务维护的文件映射（fileId 直取，无需搜索）
		if entry, ok := fileIDMapOf(key); ok {
			if entry.HasMeta() {
				client, cerr := newPan123ClientFromConfig(cfg)
				if cerr == nil {
					file := pan123.File{
						FileID:    entry.FileId,
						FileName:  entry.FileName,
						Size:      entry.Size,
						Etag:      entry.Etag,
						S3KeyFlag: entry.S3KeyFlag,
						Type:      0,
					}
					cacheKey := "fid:" + cfg.Mode + "|" + i64s(entry.FileId)
					if cached, ok := getCachedPanLink(cacheKey); ok {
						if validateDownloadURL(cached) == nil {
							c.Redirect(http.StatusFound, cached)
							return
						}
						panLinkCache.Delete(cacheKey)
					}
					link, lerr := client.GetDirectLink(file)
					if lerr == nil {
						if verr := validateDownloadURL(link); verr == nil {
							setCachedPanLink(cacheKey, link)
							c.Redirect(http.StatusFound, link)
							return
						}
					}
				}
			}
			if keyword == "" {
				// 有文件映射但没有可用元数据时，退回按文件名关键词搜索
				keyword = entry.FileName
			}
		}
	}
	if keyword == "" {
		keyword = strings.TrimSpace(c.Query("q"))
	}
	if keyword == "" {
		// 键本身即关键词（兼容 pan://<文件名> 等旧格式，不依赖关键词输入框）
		keyword = rawKey
	}
	if keyword == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "缺少 key 或 q 参数", nil)
		return
	}

	mode := cfg.Mode
	cacheKey := mode + "|" + strings.ToLower(ext) + "|" + keyword
	if cached, ok := getCachedPanLink(cacheKey); ok {
		if validateDownloadURL(cached) == nil {
			c.Redirect(http.StatusFound, cached)
			return
		}
		panLinkCache.Delete(cacheKey)
	}

	client, cerr := newPan123ClientFromConfig(cfg)
	if cerr != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, cerr.Error(), nil)
		return
	}
	files, serr := client.SearchFilesByExt(keyword, ext, 10)
	if serr != nil {
		utils.ErrorResponse(c, http.StatusBadGateway, "云盘搜索失败: "+serr.Error(), nil)
		return
	}
	file, ferr := pickDownloadFile(files)
	if ferr != nil {
		utils.ErrorResponse(c, http.StatusNotFound, ferr.Error(), nil)
		return
	}
	link, lerr := client.GetDownloadLink(*file, mode)
	if lerr != nil {
		utils.ErrorResponse(c, http.StatusBadGateway, "生成下载链接失败: "+lerr.Error(), nil)
		return
	}
	if verr := validateDownloadURL(link); verr != nil {
		utils.ErrorResponse(c, http.StatusBadGateway, "生成的下载链接不合法: "+verr.Error(), nil)
		return
	}
	setCachedPanLink(cacheKey, link)
	c.Redirect(http.StatusFound, link)
}

// ---------------------------------------------------------------------------
// GitHub → 123 云盘 软件库同步
// ---------------------------------------------------------------------------

// Pan123Sync 触发一次软件库同步（异步执行，前端轮询状态）
func Pan123Sync(c *gin.Context) {
	if !software_sync.TriggerAsync() {
		utils.SuccessResponse(c, http.StatusOK, "同步任务正在进行中", gin.H{"started": false, "running": true})
		return
	}
	utils.CreateAuditLogSimple(c, "pan123_sync", "settings", 0, "管理员操作: 手动触发 GitHub→123云盘 软件库同步")
	utils.SuccessResponse(c, http.StatusOK, "同步已开始，请稍候查看结果", gin.H{"started": true, "running": true})
}

// Pan123SyncStatus 同步状态
func Pan123SyncStatus(c *gin.Context) {
	utils.SuccessResponse(c, http.StatusOK, "", software_sync.GetStatus())
}

// GitHub 版本对照缓存（30 分钟，避免每次打开页面都打 GitHub API）
type ghVersionCacheEntry struct {
	Version string
	Expire  time.Time
}

var ghVersionCache sync.Map // repo → entry

func cachedGitHubVersion(repo, token string) (string, bool) {
	if v, ok := ghVersionCache.Load(repo); ok {
		e := v.(ghVersionCacheEntry)
		if time.Now().Before(e.Expire) {
			return e.Version, true
		}
		ghVersionCache.Delete(repo)
	}
	release, err := ghrelease.Latest(repo, loadDownloadProxyPrefixes(), token)
	if err != nil {
		return "", false
	}
	ver := release.Version()
	ghVersionCache.Store(repo, ghVersionCacheEntry{Version: ver, Expire: time.Now().Add(30 * time.Minute)})
	return ver, true
}

// Pan123Versions 版本对照：GitHub 最新版本 vs 云盘已同步版本
func Pan123Versions(c *gin.Context) {
	cfg, err := loadPan123Config()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "读取配置失败", err)
		return
	}
	fileIDMap, _ := software_sync.LoadFileIDMap()
	token := software_sync.LoadGitHubToken()
	softwareConfig := getConfigMap("software")

	rows := make([]gin.H, 0)
	for _, sw := range software_sync.Catalog {
		ghVersion := ""
		ghErr := ""
		if v, ok := cachedGitHubVersion(sw.Repo, token); ok {
			ghVersion = v
		} else {
			ghErr = "GitHub 版本获取失败（可能限流，稍后自动重试）"
		}
		for _, t := range sw.Targets {
			row := gin.H{
				"key":            t.ConfigKey,
				"name":           sw.Name,
				"label":          t.Label,
				"os":             t.OS,
				"arch":           t.Arch,
				"github_version": ghVersion,
				"cloud_version":  "",
				"file_name":      "",
				"synced":         false,
				"custom":         false,
				"gh_error":       ghErr,
			}
			// 配置了手工自定义外部链接（非 pan://、非工具写入的云盘直链）的入口：不使用云盘，明确标注
			if current := strings.TrimSpace(softwareConfig[t.ConfigKey]); current != "" && !strings.HasPrefix(current, "pan://") && !isPanManagedLink(current) {
				row["custom"] = true
				row["file_name"] = current
				rows = append(rows, row)
				continue
			}
			if entry, ok := fileIDMap[t.ConfigKey]; ok && entry.FileId > 0 {
				row["cloud_version"] = entry.Version
				row["file_name"] = entry.FileName
				row["synced"] = entry.Version == ghVersion
			}
			rows = append(rows, row)
		}
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"configured": pan123AuthConfigured(cfg),
		"list":       rows,
	})
}

// GetSoftwareVersions 用户端：已同步软件版本信息（公共接口，供下载页展示）
func GetSoftwareVersions(c *gin.Context) {
	fileIDMap, err := software_sync.LoadFileIDMap()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "读取版本信息失败", err)
		return
	}
	out := make([]gin.H, 0)
	for key, entry := range fileIDMap {
		if entry.FileId == 0 {
			continue
		}
		out = append(out, gin.H{
			"key":       key,
			"version":   entry.Version,
			"file_name": entry.FileName,
			"size":      entry.Size,
			"size_text": formatFileSize(entry.Size),
			"updated_at": entry.UpdatedAt,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i]["key"].(string) < out[j]["key"].(string)
	})
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{"list": out})
}

// ---------------------------------------------------------------------------
// 短信验证码登录（海外/风控场景）
// ---------------------------------------------------------------------------

// Pan123SmsSend 发送登录短信验证码：先探测是否触发境外风控（7012），是则发送验证码
func Pan123SmsSend(c *gin.Context) {
	var req struct {
		Username  string `json:"username"`
		Password  string `json:"password"`
		Traceless string `json:"traceless"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		cfg, _ := loadPan123Config()
		username = strings.TrimSpace(cfg.Username)
	}
	password := req.Password
	if password == "" {
		cfg, _ := loadPan123Config()
		password = cfg.Password
	}
	if username == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "缺少账号，请先填写 123 云盘账号", nil)
		return
	}

	client := pan123.New(username, password, "")
	hashCode, err := client.GetLoginHashCode(username, password)
	if err == nil && hashCode == "" {
		utils.SuccessResponse(c, http.StatusOK, "账号密码登录成功，无需短信验证", gin.H{"need_sms": false})
		return
	}
	var riskErr *pan123.RiskError
	if err != nil && !errors.As(err, &riskErr) {
		utils.ErrorResponse(c, http.StatusBadRequest, "登录探测失败: "+err.Error(), nil)
		return
	}
	if riskErr == nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "未知错误", nil)
		return
	}
	timeStamp, needsCaptcha, err := client.SendSmsCode(username, riskErr.HashCode, req.Traceless)
	if err != nil {
		if needsCaptcha {
			// 被滑块验证拦截：前端展示阿里云滑块，用户滑动后带 traceless 重发
			msg := "需要滑块验证"
			if req.Traceless != "" {
				msg = "滑块验证未通过，请重新滑动"
			}
			utils.SuccessResponse(c, http.StatusOK, msg, gin.H{
				"need_sms":      true,
				"needs_captcha": true,
				"hash_code":     riskErr.HashCode,
			})
			return
		}
		utils.ErrorResponse(c, http.StatusBadGateway, err.Error(), nil)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "验证码已发送，请注意查收短信", gin.H{
		"need_sms": true, "needs_captcha": false, "hash_code": riskErr.HashCode, "time_stamp": timeStamp,
	})
}

// Pan123SmsLogin 用短信验证码完成登录，成功后自动把 token 保存到配置
func Pan123SmsLogin(c *gin.Context) {
	var req struct {
		Username  string `json:"username"`
		Password  string `json:"password"`
		HashCode  string `json:"hash_code"`
		SmsCode   string `json:"sms_code"`
		TimeStamp string `json:"time_stamp"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		cfg, _ := loadPan123Config()
		username = strings.TrimSpace(cfg.Username)
	}
	smsCode := strings.TrimSpace(req.SmsCode)
	if smsCode == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "请输入短信验证码", nil)
		return
	}
	client := pan123.New(username, "", "")
	hashCode := strings.TrimSpace(req.HashCode)
	if hashCode == "" {
		// 自动探测 hashCode（账号密码登录被风控时返回 7012 + hashCode）
		password := ""
		cfg, _ := loadPan123Config()
		password = cfg.Password
		if hc, err := client.GetLoginHashCode(username, password); err == nil {
			hashCode = hc
		} else {
			var riskErr *pan123.RiskError
			if errors.As(err, &riskErr) {
				hashCode = riskErr.HashCode
			}
		}
	}
	if hashCode == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "缺少 hashCode，请先点击「短信验证码登录」获取", nil)
		return
	}

	token, err := client.LoginWithSmsCode(username, smsCode, req.TimeStamp, hashCode)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadGateway, err.Error(), nil)
		return
	}

	// 自动保存 token（绕过风控的登录凭证）
	if err := savePan123ConfigValue("token", token); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "登录成功但保存 token 失败", err)
		return
	}
	utils.InvalidateAllSettingCache()
	utils.CreateAuditLogSimple(c, "pan123_sms_login", "settings", 0, "管理员操作: 通过短信验证码完成 123 云盘登录")
	utils.SuccessResponse(c, http.StatusOK, "登录成功，token 已自动保存", gin.H{"token_masked": maskIfNonEmpty(token)})
}

// ---------------------------------------------------------------------------
// 二维码扫码登录（123pan 手机 App / 微信）
// ---------------------------------------------------------------------------

// Pan123QrGenerate 生成登录二维码
func Pan123QrGenerate(c *gin.Context) {
	cfg, err := loadPan123Config()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "读取配置失败", err)
		return
	}
	client, cerr := newPan123ClientFromConfig(cfg)
	if cerr != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, cerr.Error(), nil)
		return
	}
	qrURL, uniID, gerr := client.GenerateQrCode()
	if gerr != nil {
		utils.ErrorResponse(c, http.StatusBadGateway, gerr.Error(), nil)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"qr_value": pan123.BuildQrValue(qrURL, uniID), // 与登录中心一致的完整参数（App/微信识别）
		"uni_id":   uniID,
	})
}

// Pan123QrStatus 轮询扫码状态；已确认且带 token 时自动保存（token 绑定服务器 IP）
func Pan123QrStatus(c *gin.Context) {
	uniID := strings.TrimSpace(c.Query("uni_id"))
	if uniID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "缺少 uni_id 参数", nil)
		return
	}
	cfg, err := loadPan123Config()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "读取配置失败", err)
		return
	}
	client, cerr := newPan123ClientFromConfig(cfg)
	if cerr != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, cerr.Error(), nil)
		return
	}
	status, scanPlatform, token, serr := client.GetQrCodeStatusV2(uniID)
	if serr != nil {
		utils.ErrorResponse(c, http.StatusBadGateway, serr.Error(), nil)
		return
	}
	switch status {
	case 0:
		utils.SuccessResponse(c, http.StatusOK, "", gin.H{"status": "waiting"})
	case 1:
		utils.SuccessResponse(c, http.StatusOK, "", gin.H{"status": "scanned", "message": "已扫码，请在手机上确认"})
	case 2:
		utils.SuccessResponse(c, http.StatusOK, "", gin.H{"status": "rejected", "message": "已拒绝登录"})
	case 4:
		utils.SuccessResponse(c, http.StatusOK, "", gin.H{"status": "expired", "message": "二维码已过期，请重新生成"})
	default:
		// status 3 = 已确认
		if token == "" && scanPlatform == 4 {
			// 微信扫码：需用 wx_code 换取登录 token
			token, serr = client.WechatLoginByQr(uniID)
			if serr != nil {
				utils.SuccessResponse(c, http.StatusOK, "", gin.H{"status": "scanned", "message": "微信已确认，换取凭证失败: " + serr.Error()})
				return
			}
		}
		if token == "" {
			utils.SuccessResponse(c, http.StatusOK, "", gin.H{"status": "waiting"})
			return
		}
		if err := savePan123ConfigValue("token", token); err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "登录成功但保存 token 失败", err)
			return
		}
		utils.InvalidateAllSettingCache()
		utils.CreateAuditLogSimple(c, "pan123_qr_login", "settings", 0, "管理员操作: 通过二维码扫码完成 123 云盘登录")
		utils.SuccessResponse(c, http.StatusOK, "登录成功，token 已自动保存", gin.H{
			"status": "success", "token_masked": maskIfNonEmpty(token),
			"claims": pan123.SummaryClaims(token),
		})
	}
}
