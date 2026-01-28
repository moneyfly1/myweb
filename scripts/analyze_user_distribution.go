package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/geoip"

	"gorm.io/gorm"
)

type RegionStats struct {
	Region     string `json:"region"`
	Country    string `json:"country"`
	City       string `json:"city"`
	UserCount  int    `json:"user_count"`
	LoginCount int    `json:"login_count"`
}

type ReferrerStats struct {
	Domain     string `json:"domain"`
	Count      int    `json:"count"`
	UserCount  int    `json:"user_count"`
	Percentage string `json:"percentage"`
}

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	if cfg == nil {
		log.Fatal("配置未正确加载")
	}

	if err := database.InitDatabase(); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	geoipPath := os.Getenv("GEOIP_DB_PATH")
	if geoipPath == "" {
		geoipPath = "./GeoLite2-City.mmdb"
	}
	if err := geoip.InitGeoIP(geoipPath); err != nil {
		fmt.Printf("⚠️  GeoIP 未启用: %v\n", err)
		fmt.Println("提示: 如需启用地理位置解析，请下载 GeoLite2-City.mmdb 文件")
		fmt.Println("下载地址: https://dev.maxmind.com/geoip/geoip2/geolite2/")
		fmt.Println()
	} else {
		fmt.Println("✅ GeoIP 数据库已加载")
		fmt.Println()
	}
	defer geoip.Close()

	db := database.GetDB()

	fmt.Println("==========================================")
	fmt.Println("   用户地区分布和访问来源分析报告")
	fmt.Println("==========================================")
	fmt.Println()

	analyzeUserRegions(db)

	fmt.Println()
	fmt.Println("==========================================")
	fmt.Println()

	analyzeAccessSources(db)

	fmt.Println()
	fmt.Println("==========================================")
	fmt.Println()

	analyzeUserActivity(db)
}

func analyzeUserRegions(db *gorm.DB) {
	fmt.Println("📊 用户地区分布分析")
	fmt.Println("----------------------------------------")

	var auditLogs []models.AuditLog
	db.Select("DISTINCT user_id, location, ip_address").
		Where("user_id IS NOT NULL AND (location IS NOT NULL AND location != '' OR ip_address IS NOT NULL AND ip_address != '' AND ip_address != '127.0.0.1' AND ip_address != '::1')").
		Find(&auditLogs)

	var activities []models.UserActivity
	db.Select("DISTINCT user_id, location, ip_address").
		Where("location IS NOT NULL AND location != ''").
		Find(&activities)

	var devices []models.Device
	db.Select("DISTINCT subscription_id, ip_address").
		Where("ip_address IS NOT NULL AND ip_address != '' AND ip_address != '127.0.0.1' AND ip_address != '::1'").
		Find(&devices)

	regionMap := make(map[string]*RegionStats)
	userRegionMap := make(map[uint]string) // 用户ID -> 地区

	for _, log := range auditLogs {
		if !log.UserID.Valid {
			continue
		}
		userID := uint(log.UserID.Int64)

		var country, city string

		if log.Location.Valid && log.Location.String != "" {
			country, city = parseLocation(log.Location.String)
		} else if log.IPAddress.Valid && log.IPAddress.String != "" {
			ip := log.IPAddress.String
			if geoip.IsEnabled() {
				location, err := geoip.GetLocation(ip)
				if err == nil && location != nil {
					country = location.Country
					city = location.City
				} else {
					country = guessRegionFromIP(ip)
				}
			} else {
				country = guessRegionFromIP(ip)
			}
		} else {
			continue
		}

		if country == "" || country == "内网" || country == "未知" {
			continue
		}

		regionKey := country
		if city != "" {
			regionKey = country + " - " + city
		}

		if _, exists := regionMap[regionKey]; !exists {
			regionMap[regionKey] = &RegionStats{
				Region:     regionKey,
				Country:    country,
				City:       city,
				UserCount:  0,
				LoginCount: 0,
			}
		}

		regionMap[regionKey].LoginCount++

		if _, exists := userRegionMap[userID]; !exists {
			userRegionMap[userID] = regionKey
			regionMap[regionKey].UserCount++
		}
	}

	for _, device := range devices {
		if device.IPAddress == nil || *device.IPAddress == "" {
			continue
		}

		ip := *device.IPAddress
		region := guessRegionFromIP(ip)

		if region != "" && region != "内网" && region != "未知" {
			regionKey := region
			if _, exists := regionMap[regionKey]; !exists {
				regionMap[regionKey] = &RegionStats{
					Region:     regionKey,
					Country:    region,
					City:       "",
					UserCount:  0,
					LoginCount: 0,
				}
			}

			regionMap[regionKey].LoginCount++
		}
	}

	for _, activity := range activities {
		if !activity.Location.Valid || activity.Location.String == "" {
			continue
		}

		country, city := parseLocation(activity.Location.String)
		if country == "" {
			continue
		}

		regionKey := country
		if city != "" {
			regionKey = country + " - " + city
		}

		if _, exists := userRegionMap[activity.UserID]; !exists {
			userRegionMap[activity.UserID] = regionKey
			if _, exists := regionMap[regionKey]; !exists {
				regionMap[regionKey] = &RegionStats{
					Region:     regionKey,
					Country:    country,
					City:       city,
					UserCount:  0,
					LoginCount: 0,
				}
			}
			regionMap[regionKey].UserCount++
		}
	}

	if len(regionMap) == 0 {
		fmt.Println("❌ 未找到任何地区数据")
		return
	}

	fmt.Printf("✅ 共发现 %d 个地区，%d 个用户\n\n", len(regionMap), len(userRegionMap))

	regions := make([]*RegionStats, 0, len(regionMap))
	for _, stats := range regionMap {
		regions = append(regions, stats)
	}

	for i := 0; i < len(regions)-1; i++ {
		for j := i + 1; j < len(regions); j++ {
			if regions[i].UserCount < regions[j].UserCount {
				regions[i], regions[j] = regions[j], regions[i]
			}
		}
	}

	fmt.Println("地区分布（按用户数量排序）：")
	fmt.Printf("%-30s %10s %10s\n", "地区", "用户数", "登录次数")
	fmt.Println(strings.Repeat("-", 52))
	for _, stats := range regions {
		if stats.UserCount > 0 {
			fmt.Printf("%-30s %10d %10d\n", stats.Region, stats.UserCount, stats.LoginCount)
		}
	}
}

func guessRegionFromIP(ip string) string {
	if strings.HasPrefix(ip, "::ffff:") {
		ip = strings.TrimPrefix(ip, "::ffff:")
	}

	if strings.HasPrefix(ip, "1.") || strings.HasPrefix(ip, "14.") || strings.HasPrefix(ip, "27.") || strings.HasPrefix(ip, "36.") || strings.HasPrefix(ip, "39.") || strings.HasPrefix(ip, "42.") || strings.HasPrefix(ip, "49.") || strings.HasPrefix(ip, "58.") || strings.HasPrefix(ip, "59.") || strings.HasPrefix(ip, "60.") || strings.HasPrefix(ip, "61.") || strings.HasPrefix(ip, "103.") || strings.HasPrefix(ip, "106.") || strings.HasPrefix(ip, "110.") || strings.HasPrefix(ip, "111.") || strings.HasPrefix(ip, "112.") || strings.HasPrefix(ip, "113.") || strings.HasPrefix(ip, "114.") || strings.HasPrefix(ip, "115.") || strings.HasPrefix(ip, "116.") || strings.HasPrefix(ip, "117.") || strings.HasPrefix(ip, "118.") || strings.HasPrefix(ip, "119.") || strings.HasPrefix(ip, "120.") || strings.HasPrefix(ip, "121.") || strings.HasPrefix(ip, "122.") || strings.HasPrefix(ip, "123.") || strings.HasPrefix(ip, "124.") || strings.HasPrefix(ip, "125.") || strings.HasPrefix(ip, "171.") || strings.HasPrefix(ip, "175.") || strings.HasPrefix(ip, "180.") || strings.HasPrefix(ip, "182.") || strings.HasPrefix(ip, "183.") || strings.HasPrefix(ip, "202.") || strings.HasPrefix(ip, "203.") || strings.HasPrefix(ip, "210.") || strings.HasPrefix(ip, "211.") || strings.HasPrefix(ip, "218.") || strings.HasPrefix(ip, "219.") || strings.HasPrefix(ip, "220.") || strings.HasPrefix(ip, "221.") || strings.HasPrefix(ip, "222.") {
		return "中国"
	}
	if strings.HasPrefix(ip, "192.168.") || strings.HasPrefix(ip, "10.") || strings.HasPrefix(ip, "172.16.") || strings.HasPrefix(ip, "172.17.") || strings.HasPrefix(ip, "172.18.") || strings.HasPrefix(ip, "172.19.") || strings.HasPrefix(ip, "172.20.") || strings.HasPrefix(ip, "172.21.") || strings.HasPrefix(ip, "172.22.") || strings.HasPrefix(ip, "172.23.") || strings.HasPrefix(ip, "172.24.") || strings.HasPrefix(ip, "172.25.") || strings.HasPrefix(ip, "172.26.") || strings.HasPrefix(ip, "172.27.") || strings.HasPrefix(ip, "172.28.") || strings.HasPrefix(ip, "172.29.") || strings.HasPrefix(ip, "172.30.") || strings.HasPrefix(ip, "172.31.") {
		return "内网"
	}

	return "未知"
}

func analyzeAccessSources(db *gorm.DB) {
	fmt.Println("🌐 用户访问来源分析")
	fmt.Println("----------------------------------------")

	var auditLogs []models.AuditLog
	db.Select("DISTINCT user_id, user_agent").
		Where("user_id IS NOT NULL AND user_agent IS NOT NULL AND user_agent != ''").
		Find(&auditLogs)

	var activities []models.UserActivity
	db.Select("DISTINCT user_id, user_agent").
		Where("user_agent IS NOT NULL AND user_agent != ''").
		Find(&activities)

	var devices []models.Device
	db.Select("DISTINCT subscription_id, device_ua").
		Where("device_ua IS NOT NULL AND device_ua != ''").
		Find(&devices)

	browserMap := make(map[string]int)
	osMap := make(map[string]int)
	deviceMap := make(map[string]int)

	for _, log := range auditLogs {
		if !log.UserAgent.Valid || log.UserAgent.String == "" {
			continue
		}

		ua := log.UserAgent.String
		browser := extractBrowser(ua)
		os := extractOS(ua)
		device := extractDevice(ua)

		if browser != "" {
			browserMap[browser]++
		}
		if os != "" {
			osMap[os]++
		}
		if device != "" {
			deviceMap[device]++
		}
	}

	for _, activity := range activities {
		if !activity.UserAgent.Valid || activity.UserAgent.String == "" {
			continue
		}

		ua := activity.UserAgent.String
		browser := extractBrowser(ua)
		os := extractOS(ua)
		device := extractDevice(ua)

		if browser != "" {
			browserMap[browser]++
		}
		if os != "" {
			osMap[os]++
		}
		if device != "" {
			deviceMap[device]++
		}
	}

	for _, device := range devices {
		if device.DeviceUA == nil || *device.DeviceUA == "" {
			continue
		}

		ua := *device.DeviceUA
		browser := extractBrowser(ua)
		os := extractOS(ua)
		deviceType := extractDevice(ua)

		if browser != "" {
			browserMap[browser]++
		}
		if os != "" {
			osMap[os]++
		}
		if deviceType != "" {
			deviceMap[deviceType]++
		}
	}

	if len(browserMap) > 0 {
		fmt.Println("\n📱 浏览器分布：")
		for browser, count := range browserMap {
			fmt.Printf("  %-20s: %d 次\n", browser, count)
		}
	}

	if len(osMap) > 0 {
		fmt.Println("\n💻 操作系统分布：")
		for os, count := range osMap {
			fmt.Printf("  %-20s: %d 次\n", os, count)
		}
	}

	if len(deviceMap) > 0 {
		fmt.Println("\n📱 设备类型分布：")
		for device, count := range deviceMap {
			fmt.Printf("  %-20s: %d 次\n", device, count)
		}
	}

	if len(browserMap) == 0 && len(osMap) == 0 && len(deviceMap) == 0 {
		fmt.Println("❌ 未找到访问来源数据")
	}
}

func analyzeUserActivity(db *gorm.DB) {
	fmt.Println("📈 用户活跃度分析")
	fmt.Println("----------------------------------------")

	var totalUsers int64
	db.Model(&models.User{}).Where("is_admin = ?", false).Count(&totalUsers)

	var activeUsers int64
	oneWeekAgo := time.Now().AddDate(0, 0, -7)
	db.Model(&models.User{}).
		Where("is_admin = ? AND (last_login IS NOT NULL AND last_login > ? OR created_at > ?)", false, oneWeekAgo, oneWeekAgo).
		Count(&activeUsers)

	var activityCount int64
	db.Model(&models.UserActivity{}).
		Where("created_at > ?", oneWeekAgo).
		Count(&activityCount)

	var auditCount int64
	db.Model(&models.AuditLog{}).
		Where("created_at > ? AND user_id IS NOT NULL", oneWeekAgo).
		Count(&auditCount)

	loginCount := activityCount + auditCount

	fmt.Printf("总用户数: %d\n", totalUsers)
	fmt.Printf("活跃用户（7天内）: %d (%.1f%%)\n", activeUsers, float64(activeUsers)/float64(totalUsers)*100)
	fmt.Printf("7天内登录次数: %d\n", loginCount)
	if activeUsers > 0 {
		fmt.Printf("平均每用户登录次数: %.1f\n", float64(loginCount)/float64(activeUsers))
	}
}

func parseLocation(locationStr string) (country, city string) {
	if locationStr == "" {
		return "", ""
	}

	var locationData map[string]interface{}
	if err := json.Unmarshal([]byte(locationStr), &locationData); err == nil {
		if c, ok := locationData["country"].(string); ok {
			country = c
		}
		if c, ok := locationData["city"].(string); ok {
			city = c
		}
		return
	}

	if strings.Contains(locationStr, ",") {
		parts := strings.Split(locationStr, ",")
		if len(parts) >= 1 {
			country = strings.TrimSpace(parts[0])
		}
		if len(parts) >= 2 {
			city = strings.TrimSpace(parts[1])
		}
		return
	}

	country = strings.TrimSpace(locationStr)
	return
}

func extractBrowser(ua string) string {
	ua = strings.ToLower(ua)
	if strings.Contains(ua, "chrome") && !strings.Contains(ua, "edg") {
		return "Chrome"
	}
	if strings.Contains(ua, "firefox") {
		return "Firefox"
	}
	if strings.Contains(ua, "safari") && !strings.Contains(ua, "chrome") {
		return "Safari"
	}
	if strings.Contains(ua, "edg") {
		return "Edge"
	}
	if strings.Contains(ua, "opera") {
		return "Opera"
	}
	if strings.Contains(ua, "msie") || strings.Contains(ua, "trident") {
		return "IE"
	}
	return "其他"
}

func extractOS(ua string) string {
	ua = strings.ToLower(ua)
	if strings.Contains(ua, "windows") {
		return "Windows"
	}
	if strings.Contains(ua, "mac") || strings.Contains(ua, "darwin") {
		return "macOS"
	}
	if strings.Contains(ua, "linux") {
		return "Linux"
	}
	if strings.Contains(ua, "android") {
		return "Android"
	}
	if strings.Contains(ua, "ios") || strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") {
		return "iOS"
	}
	return "其他"
}

func extractDevice(ua string) string {
	ua = strings.ToLower(ua)
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone") {
		return "移动设备"
	}
	if strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad") {
		return "平板"
	}
	return "桌面设备"
}
