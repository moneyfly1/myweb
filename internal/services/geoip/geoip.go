package geoip

import (
	"cboard-go/internal/core/netutil"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/ip2location/ip2location-go/v9"
	"github.com/oschwald/geoip2-golang"
)

var (
	geoipDB       *geoip2.Reader
	ip2locationDB *ip2location.DB
	geoipDBLock   sync.RWMutex
	geoipEnabled  bool
	dbType        string // "geoip2" or "ip2location"
)

type ping0CacheEntry struct {
	location  *LocationInfo
	expiresAt time.Time
	ok        bool
}

var ping0Cache = struct {
	mu      sync.RWMutex
	entries map[string]ping0CacheEntry
}{
	entries: make(map[string]ping0CacheEntry),
}

const (
	ping0CacheSuccessTTL = 24 * time.Hour
	ping0CacheFailureTTL = 10 * time.Minute
)

func getPing0Cache(ip string) (ping0CacheEntry, bool) {
	ping0Cache.mu.RLock()
	entry, ok := ping0Cache.entries[ip]
	ping0Cache.mu.RUnlock()
	if !ok {
		return ping0CacheEntry{}, false
	}
	if time.Now().After(entry.expiresAt) {
		ping0Cache.mu.Lock()
		delete(ping0Cache.entries, ip)
		ping0Cache.mu.Unlock()
		return ping0CacheEntry{}, false
	}
	return entry, true
}

// ClearPing0Cache 清空进程内 ping0 兜底缓存（换库后必须清，见 ClearLocationCaches）
func ClearPing0Cache() {
	ping0Cache.mu.Lock()
	ping0Cache.entries = make(map[string]ping0CacheEntry)
	ping0Cache.mu.Unlock()
}

func setPing0Cache(ip string, location *LocationInfo, ok bool) {
	ttl := ping0CacheFailureTTL
	if ok {
		ttl = ping0CacheSuccessTTL
	}
	ping0Cache.mu.Lock()
	ping0Cache.entries[ip] = ping0CacheEntry{
		location:  location,
		expiresAt: time.Now().Add(ttl),
		ok:        ok,
	}
	ping0Cache.mu.Unlock()
}

type LocationInfo struct {
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	City        string  `json:"city"`
	Region      string  `json:"region"`
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
	Timezone    string  `json:"timezone,omitempty"`
}

func InitGeoIP(dbPath string) error {
	geoipDBLock.Lock()
	defer geoipDBLock.Unlock()

	if geoipDB != nil {
		if err := geoipDB.Close(); err != nil {
			log.Printf("failed to close geoip database: %v", err)
		}
		geoipDB = nil
	}
	if ip2locationDB != nil {
		ip2locationDB.Close()
		ip2locationDB = nil
	}

	if dbPath == "" {
		possiblePaths := []string{
			// DB-IP database (MMDB format, compatible with GeoIP2)
			"./dbip-city-lite.mmdb",
			"./data/dbip-city-lite.mmdb",
			// IP2Location databases
			"./IP2LOCATION-LITE-DB11.BIN",
			"./IP2LOCATION-LITE-DB11.IPV6.BIN",
			"./data/IP2LOCATION-LITE-DB11.BIN",
			"./data/IP2LOCATION-LITE-DB11.IPV6.BIN",
			// GeoLite2 databases
			"./GeoLite2-City.mmdb",
			"./data/GeoLite2-City.mmdb",
			"/usr/share/GeoIP/GeoLite2-City.mmdb",
			"/var/lib/GeoIP/GeoLite2-City.mmdb",
		}

		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				dbPath = path
				break
			}
		}
	}

	if dbPath == "" {
		geoipEnabled = false
		return fmt.Errorf("未找到 GeoIP 数据库文件，地理位置解析功能已禁用")
	}

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		geoipEnabled = false
		return fmt.Errorf("GeoIP 数据库文件不存在: %s", dbPath)
	}

	// 根据文件扩展名判断数据库类型
	if strings.HasSuffix(strings.ToUpper(dbPath), ".BIN") {
		// IP2Location database
		db, err := ip2location.OpenDB(dbPath)
		if err != nil {
			geoipEnabled = false
			return fmt.Errorf("打开 IP2Location 数据库失败: %w", err)
		}
		ip2locationDB = db
		dbType = "ip2location"
		geoipEnabled = true
		return nil
	} else if strings.HasSuffix(strings.ToLower(dbPath), ".mmdb") {
		// GeoIP2 database
		db, err := geoip2.Open(dbPath)
		if err != nil {
			geoipEnabled = false
			return fmt.Errorf("打开 GeoIP2 数据库失败: %w", err)
		}
		geoipDB = db
		dbType = "geoip2"
		geoipEnabled = true
		return nil
	}

	geoipEnabled = false
	return fmt.Errorf("不支持的数据库格式: %s", dbPath)
}

func GetLocation(ipAddress string) (*LocationInfo, error) {
	if !geoipEnabled {
		return nil, fmt.Errorf("GeoIP 未启用")
	}

	// 处理 IPv4-mapped IPv6 地址 (::ffff:192.168.1.1)
	if len(ipAddress) > 7 && ipAddress[:7] == "::ffff:" {
		ipAddress = ipAddress[7:]
	}

	// 处理本地地址
	if ipAddress == "127.0.0.1" || ipAddress == "::1" || ipAddress == "localhost" {
		return nil, fmt.Errorf("本地地址，跳过解析")
	}

	// 解析 IP 地址（支持 IPv4 和 IPv6）
	parsedIP := net.ParseIP(ipAddress)
	if parsedIP == nil {
		return nil, fmt.Errorf("无效的IP地址格式: %s", ipAddress)
	}

	// 内网/保留地址判定统一走 netutil（此前这里是一份手写实现，
	// 与 utils.IsPrivateIP、cache.go、statistics.go 的判定互不等价，
	// 导致同一 IP 在不同模块结论不一致）
	if netutil.IsPrivateOrReserved(parsedIP) {
		return nil, fmt.Errorf("内网地址，跳过解析")
	}

	geoipDBLock.RLock()
	defer geoipDBLock.RUnlock()

	if dbType == "ip2location" && ip2locationDB != nil {
		return getLocationFromIP2Location(ipAddress)
	} else if dbType == "geoip2" && geoipDB != nil {
		return getLocationFromGeoIP2(parsedIP)
	}

	return nil, fmt.Errorf("没有可用的数据库")
}

func getLocationFromGeoIP2(parsedIP net.IP) (*LocationInfo, error) {
	record, err := geoipDB.City(parsedIP)
	if err != nil {
		return nil, fmt.Errorf("GeoIP2解析失败: %w", err)
	}

	if record.Country.IsoCode == "" {
		return nil, fmt.Errorf("数据库中没有该IP地址的地理位置记录")
	}

	location := &LocationInfo{
		Country:     record.Country.Names["zh-CN"],
		CountryCode: record.Country.IsoCode,
	}

	if location.Country == "" {
		location.Country = record.Country.Names["en"]
	}

	if len(record.Subdivisions) > 0 {
		location.Region = record.Subdivisions[0].Names["zh-CN"]
		if location.Region == "" {
			location.Region = record.Subdivisions[0].Names["en"]
		}
	}

	location.City = record.City.Names["zh-CN"]
	if location.City == "" {
		location.City = record.City.Names["en"]
	}

	finalizeLocationNames(location)

	if record.Location.Latitude != 0 || record.Location.Longitude != 0 {
		location.Latitude = record.Location.Latitude
		location.Longitude = record.Location.Longitude
	}

	if record.Location.TimeZone != "" {
		location.Timezone = record.Location.TimeZone
	}

	return location, nil
}

// finalizeLocationNames 位置字段的最终命名规则 —— 两条解析路径（GeoIP2 mmdb /
// IP2Location）共用同一份实现，避免"同一 IP 在不同库/不同路径下命名规则不同"。
//
// 规则（方案：位置列不出现中英混排）：
//  1. 城市名统一清理：去掉 "(Downtown)" 这类括号补充与 " City"/" Shi" 后缀
//     —— 此前只在中文翻译内部做，导致非中国城市显示成
//     "美国, Los Angeles (Downtown Los Angeles)"；
//  2. 中国 IP：省份翻成中文；城市能翻成中文就用中文，翻不出来就丢弃英文城市名，
//     交给展示层退回中文省份（"中国, 河南 Guancheng" → "中国, 河南"）；
//     省份也翻不出中文时一并丢弃，只留国家，避免位置列里蹦出英文；
//  3. 其它国家：保留清理后的当地名称（本来就是英文，属正常展示）。
func finalizeLocationNames(location *LocationInfo) {
	if location == nil {
		return
	}

	location.City = cleanCityName(location.City)

	if location.CountryCode != "CN" {
		return
	}

	if location.Region != "" {
		location.Region = translateRegionName(location.Region)
		if !isChineseName(location.Region) {
			location.Region = ""
		}
	}

	if location.City != "" {
		switch {
		case isChineseName(location.City):
			// 已经是中文（如 GeoLite2 的 zh-CN 城市名"北京"），原样保留
		default:
			if zhCity := translateCityName(location.City); zhCity != "" {
				location.City = zhCity
			} else {
				location.City = ""
			}
		}
	}
}

// cleanCityName 清理城市名：去掉括号补充与常见英文后缀。
// 各国通用（此前只在 translateCityName 内部做，导致非中国城市带着括号显示）。
func cleanCityName(city string) string {
	city = strings.TrimSpace(city)
	// 去掉括号补充，例如 "Jinrongjie (Xicheng District)"、"Singapore (Pioneer)"
	if idx := strings.Index(city, "("); idx > 0 {
		city = strings.TrimSpace(city[:idx])
	}
	// 去掉常见后缀
	city = strings.TrimSpace(strings.TrimSuffix(city, " City"))
	city = strings.TrimSpace(strings.TrimSuffix(city, " Shi"))
	return city
}

// containsCJK 判断是否含中文字符
func containsCJK(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// containsLatinLetter 判断是否含拉丁字母
func containsLatinLetter(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return true
		}
	}
	return false
}

// isChineseName 判断是否为"纯中文名"。
// 注意不能只用 containsCJK：历史值 "河南 Guancheng" 含中文，但它是
// "中文省 + 英文城市"的中英混排，必须算作"没翻出来"，否则清理不掉。
// 反过来"北京"这类真正的中文名（GeoLite2 的 zh-CN 名）必须保留。
func isChineseName(s string) bool {
	return containsCJK(s) && !containsLatinLetter(s)
}

// translateCityName 将英文城市名翻译为中文；表中没有时返回空串。
//
// 空串表示"翻不出来"：调用方据此丢弃英文城市名、退回中文省份，
// 而不是拼出"河南 Guancheng"这种中英混排（历史上就是这么拼的）。
func translateCityName(cityEN string) string {
	// 常见中国城市英文到中文的映射
	cityMap := map[string]string{
		"Beijing": "北京", "Shanghai": "上海", "Guangzhou": "广州", "Shenzhen": "深圳",
		"Chengdu": "成都", "Hangzhou": "杭州", "Wuhan": "武汉", "Xi'an": "西安",
		"Chongqing": "重庆", "Tianjin": "天津", "Nanjing": "南京", "Shenyang": "沈阳",
		"Harbin": "哈尔滨", "Changchun": "长春", "Dalian": "大连", "Jinan": "济南",
		"Qingdao": "青岛", "Zhengzhou": "郑州", "Shijiazhuang": "石家庄", "Taiyuan": "太原",
		"Hohhot": "呼和浩特", "Urumqi": "乌鲁木齐", "Lanzhou": "兰州", "Yinchuan": "银川",
		"Xining": "西宁", "Lhasa": "拉萨", "Kunming": "昆明", "Guiyang": "贵阳",
		"Nanning": "南宁", "Haikou": "海口", "Fuzhou": "福州", "Xiamen": "厦门",
		"Nanchang": "南昌", "Changsha": "长沙", "Hefei": "合肥", "Suzhou": "苏州",
		"Wuxi": "无锡", "Ningbo": "宁波", "Wenzhou": "温州", "Dongguan": "东莞",
		"Foshan": "佛山", "Zhuhai": "珠海", "Zhongshan": "中山", "Huizhou": "惠州",
		"Hong Kong": "香港", "Kowloon": "九龙", "Macau": "澳门",
		"Zhu Cheng": "诸城", "Jinrongjie": "金融街",
	}

	cityEN = cleanCityName(cityEN)
	if zhCity, ok := cityMap[cityEN]; ok {
		return zhCity
	}
	return ""
}

func getLocationFromIP2Location(ipAddress string) (*LocationInfo, error) {
	results, err := ip2locationDB.Get_all(ipAddress)
	if err != nil {
		return nil, fmt.Errorf("IP2Location解析失败: %w", err)
	}

	if results.Country_short == "-" || results.Country_short == "" {
		return nil, fmt.Errorf("数据库中没有该IP地址的地理位置记录")
	}

	location := &LocationInfo{
		CountryCode: results.Country_short,
		Country:     results.Country_long,
		Region:      results.Region,
		City:        results.City,
		Latitude:    float64(results.Latitude),
		Longitude:   float64(results.Longitude),
		Timezone:    results.Timezone,
	}

	// 清理无效数据
	if location.Country == "-" {
		location.Country = ""
	}
	if location.Region == "-" {
		location.Region = ""
	}
	if location.City == "-" {
		location.City = ""
	}
	if location.Timezone == "-" {
		location.Timezone = ""
	}

	// 如果国家名称为空，尝试使用国家代码
	if location.Country == "" && location.CountryCode != "" {
		location.Country = location.CountryCode
	}

	finalizeLocationNames(location)

	return location, nil
}

// translateRegionName 翻译省份/地区名称
func translateRegionName(regionEN string) string {
	regionMap := map[string]string{
		"Beijing": "北京", "Shanghai": "上海", "Tianjin": "天津", "Chongqing": "重庆",
		"Guangdong": "广东", "Shandong": "山东", "Henan": "河南", "Sichuan": "四川",
		"Jiangsu": "江苏", "Hebei": "河北", "Hunan": "湖南", "Anhui": "安徽",
		"Hubei": "湖北", "Zhejiang": "浙江", "Guangxi": "广西", "Yunnan": "云南",
		"Jiangxi": "江西", "Liaoning": "辽宁", "Fujian": "福建", "Shaanxi": "陕西",
		"Heilongjiang": "黑龙江", "Shanxi": "山西", "Guizhou": "贵州", "Jilin": "吉林",
		"Inner Mongolia": "内蒙古", "Xinjiang": "新疆", "Gansu": "甘肃", "Hainan": "海南",
		"Ningxia": "宁夏", "Qinghai": "青海", "Tibet": "西藏", "Hong Kong": "香港", "Macau": "澳门",
	}

	if zhRegion, ok := regionMap[regionEN]; ok {
		return zhRegion
	}
	return regionEN
}

func GetLocationString(ipAddress string) sql.NullString {
	// 统一规范化（去端口、::ffff: 前缀、localhost 归一）后再判定
	normalized := netutil.Normalize(ipAddress, false)
	if normalized == "" {
		return sql.NullString{}
	}
	if normalized == "127.0.0.1" {
		return sql.NullString{String: "本地", Valid: true}
	}
	if netutil.IsPrivateOrReservedString(normalized) {
		return sql.NullString{String: "内网", Valid: true}
	}

	location, err := GetLocation(ipAddress)
	if err != nil || location == nil || location.Country == "" {
		return sql.NullString{Valid: false}
	}

	locationJSON, err := json.Marshal(location)
	if err != nil {
		locationStr := location.Country
		if location.City != "" {
			locationStr = location.Country + ", " + location.City
		} else if location.Region != "" {
			locationStr = location.Country + ", " + location.Region
		}
		return sql.NullString{String: locationStr, Valid: true}
	}

	return sql.NullString{String: string(locationJSON), Valid: true}
}

func IsEnabled() bool {
	geoipDBLock.RLock()
	defer geoipDBLock.RUnlock()
	return geoipEnabled
}

func Close() {
	geoipDBLock.Lock()
	defer geoipDBLock.Unlock()

	if geoipDB != nil {
		if err := geoipDB.Close(); err != nil {
			log.Printf("failed to close geoip database: %v", err)
		}
		geoipDB = nil
	}
	if ip2locationDB != nil {
		ip2locationDB.Close()
		ip2locationDB = nil
	}
	geoipEnabled = false
}

func GetLocationFromIPW(ipAddress string) (*LocationInfo, error) {
	parsedIP := net.ParseIP(ipAddress)
	if parsedIP == nil {
		return nil, fmt.Errorf("无效的IP地址格式")
	}

	if parsedIP.To4() != nil {
		return nil, fmt.Errorf("仅支持 IPv6 地址")
	}

	if ipAddress == "::1" || ipAddress == "localhost" {
		return nil, fmt.Errorf("本地地址，跳过解析")
	}

	url := fmt.Sprintf("https://ipw.cn/ipv6/?ip=%s", ipAddress)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	htmlContent := string(body)

	location := &LocationInfo{}

	jsonPatterns := []*regexp.Regexp{
		regexp.MustCompile(`"location"\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`"city"\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`"region"\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`"country"\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`"province"\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`"归属地"\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`"位置"\s*:\s*"([^"]+)"`),
	}

	for _, pattern := range jsonPatterns {
		matches := pattern.FindAllStringSubmatch(htmlContent, -1)
		for _, match := range matches {
			if len(match) >= 2 && match[1] != "" {
				value := strings.TrimSpace(match[1])
				if strings.Contains(pattern.String(), "country") {
					location.Country = value
					location.CountryCode = "CN"
				} else if strings.Contains(pattern.String(), "city") {
					location.City = value
				} else if strings.Contains(pattern.String(), "region") || strings.Contains(pattern.String(), "province") {
					location.Region = value
				} else if strings.Contains(pattern.String(), "location") || strings.Contains(pattern.String(), "归属地") || strings.Contains(pattern.String(), "位置") {
					parts := strings.Fields(value)
					if len(parts) >= 1 {
						location.Country = parts[0]
						location.CountryCode = "CN"
					}
					if len(parts) >= 2 {
						location.Region = parts[1]
					}
					if len(parts) >= 3 {
						location.City = strings.Join(parts[2:], " ")
					}
				}
			}
		}
	}

	if location.Country == "" {
		locationPatterns := []*regexp.Regexp{
			regexp.MustCompile(`(中国|China)\s+([\x{4e00}-\x{9fa5}]+)\s+([\x{4e00}-\x{9fa5}]+)\s+([\x{4e00}-\x{9fa5}]+)`),
			regexp.MustCompile(`(中国|China)\s+([\x{4e00}-\x{9fa5}]+)\s+([\x{4e00}-\x{9fa5}]+)`),
			regexp.MustCompile(`(中国|China)[,，]\s*([\x{4e00}-\x{9fa5}]+)[,，]\s*([\x{4e00}-\x{9fa5}]+)[,，]\s*([\x{4e00}-\x{9fa5}]+)`),
		}

		for _, pattern := range locationPatterns {
			matches := pattern.FindAllStringSubmatch(htmlContent, -1)
			for _, match := range matches {
				if len(match) >= 3 {
					locationText := match[0]
					if strings.Contains(locationText, "中国") || strings.Contains(locationText, "省") ||
						strings.Contains(locationText, "市") || strings.Contains(locationText, "县") ||
						strings.Contains(locationText, "区") || strings.Contains(locationText, "乡") {
						if strings.Contains(locationText, ",") || strings.Contains(locationText, "，") {
							parts := regexp.MustCompile(`[,，]\s*`).Split(locationText, -1)
							if len(parts) >= 1 {
								location.Country = strings.TrimSpace(parts[0])
								location.CountryCode = "CN"
							}
							if len(parts) >= 2 {
								location.Region = strings.TrimSpace(parts[1])
							}
							if len(parts) >= 3 {
								location.City = strings.TrimSpace(strings.Join(parts[2:], " "))
							}
						} else {
							parts := strings.Fields(locationText)
							if len(parts) >= 1 {
								location.Country = parts[0]
								location.CountryCode = "CN"
							}
							if len(parts) >= 2 {
								location.Region = parts[1]
							}
							if len(parts) >= 3 {
								location.City = strings.Join(parts[2:], " ")
							}
						}
						if location.Country != "" {
							break
						}
					}
				}
			}
			if location.Country != "" {
				break
			}
		}
	}

	if location.Country != "" {
		if location.CountryCode == "" {
			if strings.Contains(location.Country, "中国") || location.Country == "China" {
				location.CountryCode = "CN"
				if location.Country == "China" {
					location.Country = "中国"
				}
			}
		}
		return location, nil
	}

	return nil, fmt.Errorf("未能从网站解析到地理位置信息")
}

func GetLocationFromPing0(ipAddress string) (*LocationInfo, error) {
	if ipAddress == "127.0.0.1" || ipAddress == "::1" || ipAddress == "localhost" {
		return nil, fmt.Errorf("本地地址，跳过解析")
	}

	parsedIP := net.ParseIP(ipAddress)
	if parsedIP == nil {
		return nil, fmt.Errorf("无效的IP地址格式: %s", ipAddress)
	}

	if entry, ok := getPing0Cache(ipAddress); ok {
		if entry.ok && entry.location != nil && entry.location.Country != "" {
			return entry.location, nil
		}
		return nil, fmt.Errorf("Ping0缓存未命中")
	}

	url := fmt.Sprintf("https://ping0.cc/geo?ip=%s", ipAddress)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		setPing0Cache(ipAddress, nil, false)
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		setPing0Cache(ipAddress, nil, false)
		return nil, fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		setPing0Cache(ipAddress, nil, false)
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	if len(lines) < 2 {
		setPing0Cache(ipAddress, nil, false)
		return nil, fmt.Errorf("响应格式不正确")
	}

	locationStr := strings.TrimSpace(lines[1])
	if locationStr == "" {
		setPing0Cache(ipAddress, nil, false)
		return nil, fmt.Errorf("位置信息为空")
	}

	location := &LocationInfo{}

	parts := strings.Split(locationStr, "—")
	locationParts := strings.TrimSpace(parts[0])

	locationFields := strings.Fields(locationParts)
	if len(locationFields) == 0 {
		return nil, fmt.Errorf("无法解析位置信息")
	}

	location.Country = locationFields[0]

	if strings.Contains(location.Country, "中国") || location.Country == "China" {
		location.CountryCode = "CN"
		if location.Country == "China" {
			location.Country = "中国"
		}
	} else if strings.Contains(location.Country, "美国") || location.Country == "United States" {
		location.CountryCode = "US"
		if location.Country == "United States" {
			location.Country = "美国"
		}
	}

	if len(locationFields) >= 2 {
		location.Region = locationFields[1]
	}

	if len(locationFields) >= 3 {
		location.City = strings.Join(locationFields[2:], " ")
	}

	if location.Country != "" {
		setPing0Cache(ipAddress, location, true)
		return location, nil
	}

	setPing0Cache(ipAddress, nil, false)
	return nil, fmt.Errorf("未能从API解析到地理位置信息")
}

// GetLocationWithFallback 使用多种方式尝试获取地理位置信息
// 优先级：1. GeoIP 数据库 2. Ping0 API 3. IPW API (仅 IPv6)
// 支持 IPv4 和 IPv6
func GetLocationWithFallback(ipAddress string) (*LocationInfo, error) {
	// 处理 IPv4-mapped IPv6 地址
	originalIP := ipAddress
	if len(ipAddress) > 7 && ipAddress[:7] == "::ffff:" {
		ipAddress = ipAddress[7:]
	}

	// 1. 优先使用本地 GeoIP 数据库（支持 IPv4 和 IPv6）
	location, err := GetLocation(originalIP)
	if err == nil && location != nil && location.Country != "" {
		return location, nil
	}

	// 2. 尝试使用 Ping0 API（支持 IPv4 和 IPv6）
	ping0Location, err := GetLocationFromPing0(originalIP)
	if err == nil && ping0Location != nil && ping0Location.Country != "" {
		return ping0Location, nil
	}

	// 3. 如果是 IPv6，尝试使用 IPW API
	parsedIP := net.ParseIP(originalIP)
	if parsedIP != nil && parsedIP.To4() == nil {
		ipwLocation, err := GetLocationFromIPW(originalIP)
		if err == nil && ipwLocation != nil && ipwLocation.Country != "" {
			return ipwLocation, nil
		}
	}

	// 所有方法都失败
	if err != nil {
		return nil, fmt.Errorf("无法解析地理位置信息: %w", err)
	}
	return nil, fmt.Errorf("无法解析地理位置信息: 所有方法都失败")
}
