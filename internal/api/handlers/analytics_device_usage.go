package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// 设备 / 客户端使用分析（用户分析页「设备与客户端使用」区块）
//
// 需求（2026-10-16）：后台「用户分析」里要能看到「iOS 设备型号 / MoneyFly 各版本 /
// 其他客户端软件」的使用分布；点开某一行能看到对应的用户明细；对**没在用官方客户端**
// 的用户可以勾选（或全选）批量发邮件，引导他升级到官方客户端并给出（加速镜像）
// 下载地址。
//
// 设计要点：
//   - 分组维度收敛成一个 dim 参数，前端只切下拉（见 deviceUsageDims）；
//   - 「官方客户端」= software_name 去空格后等于 MoneyFly（大小写不敏感）；
//     第三方客户端没有安装标识，只能按软件名归类 —— 这正是要发升级提醒的那批人；
//   - 统计口径：devices 表按维度分组；用户数按 devices.user_id 去重
//     （同一用户多台设备算一个人，也决定群发时"发几封"）；
//   - 只统计**活跃行**（is_active=1 且未被踢下线），已删除的历史设备不计入；
//   - q 搜索 / days 活跃窗口 / official 筛选全部下推到 SQL，避免把大表拉进内存；
//   - **不用字符串拼接表达式**（`||` 在 MySQL 里是逻辑或、在 SQLite 里是连接），
//     所有维度都是单列，跨 SQLite/MySQL/PostgreSQL 都能跑。
// ---------------------------------------------------------------------------

// 官方（自研）客户端名单：MoneyFly 与它的后续版本 MBox 并存 ——
// 品牌改名不代表老客户端不再是我们自己的客户端，两套 UA 都要算「官方」，
// 否则后台「官方 / 第三方」口径会把 MBox 用户错算成「待升级的第三方用户」。
var officialClientNames = []string{"MoneyFly", "MBox"}

// officialClientName 保留单值形式，仅用于展示/日志等只想要一个代表名的场景。
const officialClientName = "MoneyFly"

func lowerOfficialClientNames() []string {
	out := make([]string, 0, len(officialClientNames))
	for _, n := range officialClientNames {
		out = append(out, strings.ToLower(n))
	}
	return out
}

// isOfficialClientName 判断软件名是否属于官方客户端（大小写/空白不敏感）
func isOfficialClientName(name string) bool {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return false
	}
	for _, n := range officialClientNames {
		if key == strings.ToLower(n) {
			return true
		}
	}
	return false
}

// deviceUsageDim 分组维度（Column 只允许是白名单里写死的列名，杜绝注入）
type deviceUsageDim struct {
	Key     string
	Label   string
	Column  string
	Hint    string
	iOSOnly bool
}

var deviceUsageDims = []deviceUsageDim{
	{Key: "client", Label: "客户端软件", Column: "software_name"},
	{Key: "client_version", Label: "客户端版本", Column: "software_version"},
	{Key: "platform", Label: "平台（系统）", Column: "os_name"},
	{Key: "ios_model", Label: "iOS 设备型号", Column: "device_model", Hint: "仅 iOS 设备", iOSOnly: true},
	{Key: "ios_version", Label: "iOS 系统版本", Column: "os_version", Hint: "仅 iOS 设备", iOSOnly: true},
	{Key: "os_version", Label: "系统版本", Column: "os_version"},
	{Key: "device_type", Label: "设备类型", Column: "device_type"},
	{Key: "brand", Label: "设备品牌", Column: "device_brand"},
}

func resolveDeviceUsageDim(key string) deviceUsageDim {
	k := strings.TrimSpace(strings.ToLower(key))
	for _, d := range deviceUsageDims {
		if d.Key == k {
			return d
		}
	}
	return deviceUsageDims[0]
}

// deviceUsageBaseQuery 活跃设备 + 通用筛选（官方/时间窗/iOS/搜索）
func deviceUsageBaseQuery(db *gorm.DB, q, official, days string, dim deviceUsageDim) *gorm.DB {
	tx := db.Model(&models.Device{}).
		Where("is_active = ? AND kicked_at IS NULL", true)

	scope := strings.ToLower(strings.TrimSpace(official))

	// 自动化/抓取类 UA（curl / Go-http-client / python-requests / okhttp /
	// 链接预览爬虫 …）**不是用户设备**，所有维度默认排除：
	//   - 混进「待升级的第三方用户」会让数字虚高（用户反馈"很多未知设备"）；
	//   - 混进分组列表会挤掉真实客户端：线上「客户端软件」维度里爬虫占了 10 组，
	//     官方客户端 MBox 只有 2 台，被 20 条/页的分页挤到第 2 页，后台完全看不到
	//     （用户反馈"为什么没有 MBox 的显示"）。
	// 排除后 summary 仍用 automation_devices 单独给出计数，供解释总设备数.
	if scope != "automation" {
		tx = tx.Where("LOWER(TRIM(COALESCE(software_name, ''))) NOT IN ?", automationSoftwareNames()).
			Where("LOWER(TRIM(COALESCE(os_name, ''))) <> ?", "自动化/非客户端")
	}

	switch scope {
	case "official":
		tx = tx.Where("LOWER(TRIM(COALESCE(software_name, ''))) IN ?", lowerOfficialClientNames())
	case "third_party", "unofficial":
		tx = tx.Where("LOWER(TRIM(COALESCE(software_name, ''))) NOT IN ?", lowerOfficialClientNames())
	case "automation":
		// 只统计自动化/抓取请求（summary 用；列表维度不展示）
		tx = tx.Where("LOWER(TRIM(COALESCE(software_name, ''))) IN ? OR LOWER(TRIM(COALESCE(os_name, ''))) = ?",
			automationSoftwareNames(), "自动化/非客户端")
	}

	if n, err := strconv.Atoi(strings.TrimSpace(days)); err == nil && n > 0 {
		tx = tx.Where("last_access >= ?", utils.GetBeijingTime().AddDate(0, 0, -n))
	}

	if dim.iOSOnly {
		tx = tx.Where("LOWER(COALESCE(os_name, '')) LIKE ?", "%ios%")
	}

	if s := strings.TrimSpace(q); s != "" {
		like := "%" + strings.ToLower(s) + "%"
		tx = tx.Where(`LOWER(COALESCE(device_model, '')) LIKE ?
			OR LOWER(COALESCE(device_brand, '')) LIKE ?
			OR LOWER(COALESCE(os_name, '')) LIKE ?
			OR LOWER(COALESCE(os_version, '')) LIKE ?
			OR LOWER(COALESCE(software_name, '')) LIKE ?
			OR LOWER(COALESCE(software_version, '')) LIKE ?`,
			like, like, like, like, like, like)
	}
	return tx
}

// automationSoftwareNames 自动化/抓取类客户端（不是用户设备），统计第三方用户时排除。
// 与 device 包 UA 归类保持同一份名单：软件名命中即视为抓取请求。
func automationSoftwareNames() []string {
	return []string{"curl", "wget", "go-http-client", "python-requests", "python-urllib",
		"okhttp", "postman", "axios", "node-fetch", "java", "apache-httpclient", "libwww-perl",
		"got", "undici", "java-http-client",
		// 链接预览爬虫 / 订阅链接被直接请求（UA 本身就是链接）
		"链接预览", "直接抓取订阅", "协议链接请求", "反垃圾检测"}
}

// DeviceUsageGroup 一条分组统计
type DeviceUsageGroup struct {
	Key        string  `json:"key"`
	Label      string  `json:"label"`
	Devices    int64   `json:"devices"`
	Users      int64   `json:"users"`
	Share      float64 `json:"share"`
	LastAccess string  `json:"last_access"`
	Official   bool    `json:"official"`
}

// GetDeviceUsageAnalytics 设备/客户端使用分布
// GET /api/v1/admin/analytics/device-usage?dim=client&q=&official=all|official|third_party&days=0&page=1&limit=20
func GetDeviceUsageAnalytics(c *gin.Context) {
	db := database.GetDB()
	dim := resolveDeviceUsageDim(c.DefaultQuery("dim", "client"))
	q := c.Query("q")
	official := c.DefaultQuery("official", "all")
	days := c.DefaultQuery("days", "0")
	page, limit := parsePageLimit(c, 20, 200)

	base := deviceUsageBaseQuery(db, q, official, days, dim)

	var totalDevices int64
	if err := base.Session(&gorm.Session{}).Count(&totalDevices).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "统计设备失败", err)
		return
	}

	var totalUsers int64
	base.Session(&gorm.Session{}).Where("user_id IS NOT NULL").Distinct("user_id").Count(&totalUsers)

	var officialDevices int64
	deviceUsageBaseQuery(db, q, "official", days, dim).Count(&officialDevices)

	// 自动化请求数（curl/脚本/链接预览爬虫）：单列出来，让"设备总数"能被解释，
	// 也便于前端提示「其中 N 台不是用户设备」。注意这里必须用 scope=automation
	// 的独立查询 —— base 已经把这些行排除掉了。
	var automationDevices int64
	deviceUsageBaseQuery(db, q, "automation", days, dim).Count(&automationDevices)

	// 第三方设备数：base 已排除自动化请求，totalDevices 里本来就没有它们。
	thirdPartyDevices := totalDevices - officialDevices
	if thirdPartyDevices < 0 {
		thirdPartyDevices = 0
	}

	var groupTotal int64
	base.Session(&gorm.Session{}).Distinct(dim.Column).Count(&groupTotal)

	type row struct {
		Key     string
		Devices int64
		Users   int64
		// 聚合结果用字符串接：SQLite 把 datetime 存成字符串，直接扫进 time.Time 会
		// 报 "unsupported Scan, storing driver.Value type string into type *time.Time"
		// （MySQL 返回 time.Time、SQLite 返回 string，跨库统一按字符串处理最省事）
		LastAccess *string
	}
	var rows []row
	if err := base.Session(&gorm.Session{}).
		Select(dim.Column + " AS key, COUNT(*) AS devices, COUNT(DISTINCT user_id) AS users, MAX(last_access) AS last_access").
		Group(dim.Column).
		Order("devices DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Scan(&rows).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "分组统计失败", err)
		return
	}

	groups := make([]DeviceUsageGroup, 0, len(rows))
	for _, r := range rows {
		key := strings.TrimSpace(r.Key)
		label := key
		if label == "" {
			label = "未知"
		}
		share := 0.0
		if totalDevices > 0 {
			share = float64(r.Devices) * 100 / float64(totalDevices)
		}
		last := formatAggTime(r.LastAccess)
		groups = append(groups, DeviceUsageGroup{
			Key: key, Label: label, Devices: r.Devices, Users: r.Users,
			Share:      round2(share),
			LastAccess: last,
			Official:   isOfficialClientName(key),
		})
	}

	dims := make([]gin.H, 0, len(deviceUsageDims))
	for _, d := range deviceUsageDims {
		dims = append(dims, gin.H{"key": d.Key, "label": d.Label, "hint": d.Hint, "ios_only": d.iOSOnly})
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"dim":         dim.Key,
		"dim_label":   dim.Label,
		"dims":        dims,
		"page":        page,
		"limit":       limit,
		"group_total": groupTotal,
		"summary": gin.H{
			"devices":             totalDevices,
			"users":               totalUsers,
			"official_devices":    officialDevices,
			"third_party_devices": thirdPartyDevices,
			"automation_devices":  automationDevices,
		},
		"groups": groups,
	})
}

// DeviceUsageDetailRow 明细行（设备 + 对应用户 + 订阅状态）
type DeviceUsageDetailRow struct {
	DeviceID        uint   `json:"device_id"`
	UserID          uint   `json:"user_id"`
	Username        string `json:"username"`
	Email           string `json:"email"`
	DeviceModel     string `json:"device_model"`
	DeviceBrand     string `json:"device_brand"`
	OSName          string `json:"os_name"`
	OSVersion       string `json:"os_version"`
	SoftwareName    string `json:"software_name"`
	SoftwareVersion string `json:"software_version"`
	DeviceType      string `json:"device_type"`
	IPAddress       string `json:"ip_address"`
	Location        string `json:"location"`
	Official        bool   `json:"official"`
	LastAccess      string `json:"last_access"`
	FirstSeen       string `json:"first_seen"`
	Remark          string `json:"remark"`
	SubscriptionID  uint   `json:"subscription_id"`
	SubStatus       string `json:"sub_status"`
	SubExpire       string `json:"sub_expire"`
	UserActive      bool   `json:"user_active"`
	// DeviceLabel / DeviceSub：抽屉里「设备」列的两行文本。
	// 直接拼后端，前端不用自己判断该显示机型还是系统还是客户端
	// （历史问题：抽屉里曾经整列都是「未知设备」，因为只看了 device_model）。
	DeviceLabel string `json:"device_label"`
	DeviceSub   string `json:"device_sub"`
}

// GetDeviceUsageDetail 分组下的用户/设备明细（抽屉）
// GET /api/v1/admin/analytics/device-usage/detail?dim=client&key=Shadowrocket&page=1&limit=20
func GetDeviceUsageDetail(c *gin.Context) {
	db := database.GetDB()
	dim := resolveDeviceUsageDim(c.DefaultQuery("dim", "client"))
	key := c.Query("key")
	q := c.Query("q")
	official := c.DefaultQuery("official", "all")
	days := c.DefaultQuery("days", "0")
	page, limit := parsePageLimit(c, 20, 200)

	base := deviceUsageBaseQuery(db, q, official, days, dim)
	if strings.TrimSpace(key) != "" {
		base = base.Where(dim.Column+" = ?", key)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "统计明细失败", err)
		return
	}
	var distinctUsers int64
	base.Session(&gorm.Session{}).Where("user_id IS NOT NULL").Distinct("user_id").Count(&distinctUsers)

	var devices []models.Device
	if err := base.Session(&gorm.Session{}).
		Order("last_access DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&devices).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "查询明细失败", err)
		return
	}

	// 一次性把用户与订阅查出来（避免 N+1）
	userMap := map[uint]models.User{}
	subMap := map[uint]models.Subscription{}
	if len(devices) > 0 {
		userIDs := make([]uint, 0, len(devices))
		subIDs := make([]uint, 0, len(devices))
		for _, d := range devices {
			if d.UserID != nil && *d.UserID > 0 {
				userIDs = append(userIDs, uint(*d.UserID))
			}
			subIDs = append(subIDs, d.SubscriptionID)
		}
		if len(userIDs) > 0 {
			var users []models.User
			db.Where("id IN ?", userIDs).Find(&users)
			for _, u := range users {
				userMap[u.ID] = u
			}
		}
		if len(subIDs) > 0 {
			var subs []models.Subscription
			db.Where("id IN ?", subIDs).Find(&subs)
			for _, s := range subs {
				subMap[s.ID] = s
			}
		}
	}

	rows := make([]DeviceUsageDetailRow, 0, len(devices))
	for _, d := range devices {
		r := DeviceUsageDetailRow{
			DeviceID:        d.ID,
			DeviceModel:     strVal(d.DeviceModel),
			DeviceBrand:     strVal(d.DeviceBrand),
			OSName:          strVal(d.OSName),
			OSVersion:       strVal(d.OSVersion),
			SoftwareName:    strVal(d.SoftwareName),
			SoftwareVersion: strVal(d.SoftwareVersion),
			DeviceType:      strVal(d.DeviceType),
			IPAddress:       strVal(d.IPAddress),
			Location:        strVal(d.Location),
			Remark:          strVal(d.Remark),
			Official:        isOfficialClientName(strVal(d.SoftwareName)),
			LastAccess:      d.LastAccess.Format("2006-01-02 15:04"),
			SubscriptionID:  d.SubscriptionID,
		}
		r.DeviceLabel, r.DeviceSub = deviceLabelOf(&d)
		if d.FirstSeen != nil {
			r.FirstSeen = d.FirstSeen.Format("2006-01-02 15:04")
		}
		if d.UserID != nil {
			r.UserID = uint(*d.UserID)
			if u, ok := userMap[r.UserID]; ok {
				r.Username = u.Username
				r.Email = u.Email
				r.UserActive = u.IsActive
			}
		}
		if s, ok := subMap[d.SubscriptionID]; ok {
			r.SubStatus = s.Status
			if !s.ExpireTime.IsZero() {
				r.SubExpire = s.ExpireTime.Format("2006-01-02")
			}
		}
		rows = append(rows, r)
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"total":          total,
		"distinct_users": distinctUsers,
		"page":           page,
		"limit":          limit,
		"items":          rows,
	})
}

// formatAggTime 把聚合出来的时间字符串规整成 "YYYY-MM-DD HH:MM"。
// 兼容 SQLite 的 "2026-10-16 16:40:05.733+08:00" 与 MySQL 的 "2026-10-16 16:40:05"。
func formatAggTime(v *string) string {
	if v == nil {
		return ""
	}
	s := strings.TrimSpace(*v)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "T", " ")
	if len(s) >= 16 {
		return s[:16]
	}
	return s
}

func strVal(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

// deviceLabelOf 生成「设备」列的两行文本（主标题 + 副标题）。
//
// 历史问题：抽屉里只取 device_model，而 device_model 只有 iOS/Android 的 UA
// 才解析得出来，于是桌面客户端、系统 UA 全都显示「未知设备」——实际信息
// 明明都在 os_name / software_name 里。现在按信息量从高到低兜底：
//
//	机型（iPhone 15 Pro）> 品牌（HUAWEI）> 系统+版本（iOS 18.6 / Windows 11）
//	> 客户端+版本（Mihomo Party 2.0.0）> 未知设备
//
// 副标题与主标题互补（主标题是设备就给系统，主标题是系统就给客户端），
// 避免两行重复同一句话。
func deviceLabelOf(d *models.Device) (label, sub string) {
	blank := func(s string) bool {
		v := strings.TrimSpace(strings.ToLower(s))
		return v == "" || v == "unknown" || s == "未知" || s == "未知设备"
	}

	model := strVal(d.DeviceModel)
	brand := strVal(d.DeviceBrand)
	osName := strVal(d.OSName)
	osVersion := strVal(d.OSVersion)
	sw := strVal(d.SoftwareName)
	swVersion := strVal(d.SoftwareVersion)

	osText := strings.TrimSpace(osName)
	if !blank(osVersion) {
		osText = strings.TrimSpace(osText + " " + osVersion)
	}
	swText := strings.TrimSpace(sw)
	if !blank(swVersion) {
		swText = strings.TrimSpace(swText + " " + swVersion)
	}

	switch {
	case !blank(model):
		return model, osText
	case !blank(brand):
		return brand, osText
	case !blank(osName):
		// 系统能确定时，主标题给系统，副标题给客户端（客户端列本身也在，这里是补充）
		if blank(sw) {
			return osText, osVersion
		}
		return osText, swText
	case !blank(sw):
		return swText, ""
	}
	return "未知设备", ""
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

func parsePageLimit(c *gin.Context, defLimit, maxLimit int) (int, int) {
	page := 1
	if v, err := strconv.Atoi(strings.TrimSpace(c.DefaultQuery("page", "1"))); err == nil && v > 0 {
		page = v
	}
	limit := defLimit
	if v, err := strconv.Atoi(strings.TrimSpace(c.DefaultQuery("limit", ""))); err == nil && v > 0 {
		limit = v
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return page, limit
}

// ---------------------------------------------------------------------------
// 「按用户」视角：一个用户一行，直接看出他在用什么软件/版本，并能一键发升级通知
//
// 需求（2026-10-19）：管理员要看到「哪些用户用的是哪些软件版本」，并且点这个用户
// 就能给他发「安装官方客户端」的通知邮件。原来的分组统计只能"从分组下钻到设备"，
// 反过来查人要多点两步 —— 这里补一个用户维度的列表。
//
// 实现：两步查询（先按用户分页，再取这些用户的设备聚合成标签），避免大表全量扫。
// ---------------------------------------------------------------------------

// DeviceUsageUserRow 一个用户的使用情况
type DeviceUsageUserRow struct {
	UserID      uint     `json:"user_id"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	UserActive  bool     `json:"user_active"`
	Devices     int64    `json:"devices"`
	Clients     []string `json:"clients"`      // 形如 ["Shadowrocket 2.0.3", "MoneyFly 2.2.30"]
	OfficialUse bool     `json:"official_use"` // 该用户是否在用官方客户端
	OnlyThird   bool     `json:"only_third"`   // 全部都是第三方（= 待劝升级）
	Brands      []string `json:"brands"`       // 机型/系统摘要（最多 3 个）
	LastAccess  string   `json:"last_access"`
	SubStatus   string   `json:"sub_status"`
	SubExpire   string   `json:"sub_expire"`
}

// GetDeviceUsageUsers 按用户列出「谁在用哪个软件/版本」
// GET /api/v1/admin/analytics/device-usage/users?q=&official=all|official|third_party&days=0&page=1&limit=20
func GetDeviceUsageUsers(c *gin.Context) {
	db := database.GetDB()
	q := c.Query("q")
	official := c.DefaultQuery("official", "all")
	days := c.DefaultQuery("days", "0")
	page, limit := parsePageLimit(c, 20, 200)

	// dim 固定用 platform（不影响筛选；只借 base 的通用条件）
	dim := deviceUsageDim{Key: "platform", Column: "os_name"}
	base := deviceUsageBaseQuery(db, q, official, days, dim).
		Where("user_id IS NOT NULL")

	var totalUsers int64
	base.Session(&gorm.Session{}).Distinct("user_id").Count(&totalUsers)

	type userAgg struct {
		UserID     int64
		Devices    int64
		LastAccess *string
	}
	var aggs []userAgg
	if err := base.Session(&gorm.Session{}).
		Select("user_id, COUNT(*) AS devices, MAX(last_access) AS last_access").
		Group("user_id").
		Order("last_access DESC").
		Offset((page - 1) * limit).Limit(limit).
		Scan(&aggs).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "按用户聚合失败", err)
		return
	}

	userIDs := make([]uint, 0, len(aggs))
	for _, a := range aggs {
		userIDs = append(userIDs, uint(a.UserID))
	}
	if len(userIDs) == 0 {
		utils.SuccessResponse(c, http.StatusOK, "", gin.H{
			"total": 0, "page": page, "limit": limit, "items": []DeviceUsageUserRow{},
		})
		return
	}

	// 这些用户的设备行（同一套筛选条件）→ 在内存里聚合成软件/版本标签
	var devices []models.Device
	base.Session(&gorm.Session{}).Where("user_id IN ?", userIDs).
		Order("last_access DESC").Find(&devices)

	type acc struct {
		clients     []string
		brands      []string
		officialUse bool
		third       bool
	}
	accs := map[uint]*acc{}
	for _, d := range devices {
		uid := uint(0)
		if d.UserID != nil {
			uid = uint(*d.UserID)
		}
		a := accs[uid]
		if a == nil {
			a = &acc{}
			accs[uid] = a
		}
		sw := strVal(d.SoftwareName)
		ver := strVal(d.SoftwareVersion)
		if sw != "" {
			label := sw
			if ver != "" {
				label = sw + " " + ver
			}
			dup := false
			for _, c := range a.clients {
				if c == label {
					dup = true
					break
				}
			}
			if !dup {
				a.clients = append(a.clients, label)
			}
			if isOfficialClientName(sw) {
				a.officialUse = true
			} else {
				a.third = true
			}
		}
		brand := strVal(d.DeviceModel)
		if brand == "" {
			brand = strVal(d.OSName)
		}
		if brand != "" && len(a.brands) < 3 {
			dup := false
			for _, b := range a.brands {
				if b == brand {
					dup = true
					break
				}
			}
			if !dup {
				a.brands = append(a.brands, brand)
			}
		}
	}

	var users []models.User
	db.Where("id IN ?", userIDs).Find(&users)
	userMap := map[uint]models.User{}
	for _, u := range users {
		userMap[u.ID] = u
	}

	subMap := map[uint]models.Subscription{}
	var subs []models.Subscription
	db.Where("user_id IN ?", userIDs).Order("created_at DESC").Find(&subs)
	for _, s := range subs {
		if _, ok := subMap[s.UserID]; !ok {
			subMap[s.UserID] = s
		}
	}

	rows := make([]DeviceUsageUserRow, 0, len(aggs))
	for _, a := range aggs {
		uid := uint(a.UserID)
		u := userMap[uid]
		ac := accs[uid]
		row := DeviceUsageUserRow{
			UserID:     uid,
			Username:   u.Username,
			Email:      u.Email,
			UserActive: u.IsActive,
			Devices:    a.Devices,
			LastAccess: formatAggTime(a.LastAccess),
		}
		if ac != nil {
			row.Clients = ac.clients
			row.Brands = ac.brands
			row.OfficialUse = ac.officialUse
			// 「只用第三方」= 没有任何官方客户端记录，且确实有第三方记录
			row.OnlyThird = ac.third && !ac.officialUse
		}
		if s, ok := subMap[uid]; ok {
			row.SubStatus = s.Status
			if !s.ExpireTime.IsZero() {
				row.SubExpire = s.ExpireTime.Format("2006-01-02")
			}
		}
		rows = append(rows, row)
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"total": totalUsers,
		"page":  page,
		"limit": limit,
		"items": rows,
	})
}
