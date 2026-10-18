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

const officialClientName = "MoneyFly"

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

	switch strings.ToLower(strings.TrimSpace(official)) {
	case "official":
		tx = tx.Where("LOWER(TRIM(COALESCE(software_name, ''))) = ?", strings.ToLower(officialClientName))
	case "third_party", "unofficial":
		tx = tx.Where("LOWER(TRIM(COALESCE(software_name, ''))) <> ?", strings.ToLower(officialClientName))
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
			Official:   strings.EqualFold(key, officialClientName),
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
			"third_party_devices": totalDevices - officialDevices,
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
			Official:        strings.EqualFold(strVal(d.SoftwareName), officialClientName),
			LastAccess:      d.LastAccess.Format("2006-01-02 15:04"),
			SubscriptionID:  d.SubscriptionID,
		}
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
