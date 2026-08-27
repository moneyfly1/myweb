package handlers

import (
	"encoding/json"
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
		utils.ErrorResponse(c, http.StatusBadRequest, "连接失败: "+err.Error(), nil)
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
		if current := strings.TrimSpace(softwareConfig[key]); current != "" && !strings.HasPrefix(current, "pan://") {
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
			// 配置了自定义外部链接的入口：不使用云盘，明确标注
			if current := strings.TrimSpace(softwareConfig[t.ConfigKey]); current != "" && !strings.HasPrefix(current, "pan://") {
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
