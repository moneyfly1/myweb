package geoip

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/oschwald/geoip2-golang"
)

var (
	geoipDB      *geoip2.Reader
	geoipDBLock  sync.RWMutex
	geoipEnabled bool
)

// LocationInfo 地理位置信息
type LocationInfo struct {
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	City        string  `json:"city"`
	Region      string  `json:"region"`
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
	Timezone    string  `json:"timezone,omitempty"`
}

// InitGeoIP 初始化 GeoIP 数据库
func InitGeoIP(dbPath string) error {
	geoipDBLock.Lock()
	defer geoipDBLock.Unlock()

	// 如果已经初始化，先关闭
	if geoipDB != nil {
		geoipDB.Close()
		geoipDB = nil
	}

	// 如果未指定路径，尝试默认路径
	if dbPath == "" {
		possiblePaths := []string{
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

	// 如果仍然没有找到，返回错误但不阻止程序运行
	if dbPath == "" {
		geoipEnabled = false
		return fmt.Errorf("未找到 GeoIP 数据库文件，地理位置解析功能已禁用")
	}

	// 检查文件是否存在
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		geoipEnabled = false
		return fmt.Errorf("GeoIP 数据库文件不存在: %s", dbPath)
	}

	// 打开数据库
	db, err := geoip2.Open(dbPath)
	if err != nil {
		geoipEnabled = false
		return fmt.Errorf("打开 GeoIP 数据库失败: %w", err)
	}

	geoipDB = db
	geoipEnabled = true
	return nil
}

// GetLocation 根据IP地址获取地理位置信息
func GetLocation(ipAddress string) (*LocationInfo, error) {
	if !geoipEnabled || geoipDB == nil {
		return nil, fmt.Errorf("GeoIP 未启用")
	}

	// 处理IPv6映射的IPv4地址（注意：ip.go 的 ParseIP 已经处理过，这里作为二次检查）
	if len(ipAddress) > 7 && ipAddress[:7] == "::ffff:" {
		ipAddress = ipAddress[7:]
	}

	// 跳过本地地址（注意：ip.go 的 ParseIP 已经将 ::1 转换为 127.0.0.1，这里作为二次检查）
	if ipAddress == "127.0.0.1" || ipAddress == "::1" || ipAddress == "localhost" {
		return nil, fmt.Errorf("本地地址，跳过解析")
	}

	// 解析IP地址
	parsedIP := net.ParseIP(ipAddress)
	if parsedIP == nil {
		return nil, fmt.Errorf("无效的IP地址格式: %s", ipAddress)
	}

	geoipDBLock.RLock()
	defer geoipDBLock.RUnlock()

	// GeoIP2 数据库支持 IPv4 和 IPv6 地址
	// 直接使用 net.ParseIP 解析的地址，GeoIP2 库会自动处理 IPv6
	record, err := geoipDB.City(parsedIP)
	if err != nil {
		// 如果解析失败，可能是数据库中没有该IPv6地址的记录
		// 对于IPv6地址，即使解析失败也返回错误，让调用者知道解析失败
		// 但不会阻止其他IPv6地址的解析尝试
		return nil, fmt.Errorf("GeoIP解析失败: %w", err)
	}

	// 检查是否返回了有效的地理位置信息
	if record.Country.IsoCode == "" {
		// 如果国家代码为空，说明数据库中没有该地址的记录
		return nil, fmt.Errorf("数据库中没有该IP地址的地理位置记录")
	}

	location := &LocationInfo{
		Country:     record.Country.Names["zh-CN"],
		CountryCode: record.Country.IsoCode,
	}

	// 如果中文名称不存在，使用英文名称
	if location.Country == "" {
		location.Country = record.Country.Names["en"]
	}

	// 获取城市信息
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

	// 获取坐标
	if record.Location.Latitude != 0 || record.Location.Longitude != 0 {
		location.Latitude = record.Location.Latitude
		location.Longitude = record.Location.Longitude
	}

	// 获取时区
	if record.Location.TimeZone != "" {
		location.Timezone = record.Location.TimeZone
	}

	return location, nil
}

// GetLocationString 获取格式化的位置字符串
func GetLocationString(ipAddress string) sql.NullString {
	// 处理本地IP和内网IP（注意：ip.go 的 ParseIP 已经将 ::1 转换为 127.0.0.1，这里作为二次检查）
	if ipAddress == "127.0.0.1" || ipAddress == "::1" || ipAddress == "localhost" {
		return sql.NullString{String: "本地", Valid: true}
	}

	// 检查是否为内网IP
	ip := net.ParseIP(ipAddress)
	if ip != nil {
		// 对于IPv4，检查是否为内网
		if ip.To4() != nil {
			// IPv4地址：检查是否为内网
			if ip.IsLoopback() || ip.IsPrivate() {
				return sql.NullString{String: "内网", Valid: true}
			}
		} else {
			// IPv6地址：只检查是否为回环地址，不检查私有地址（因为IPv6的私有地址范围很广，可能误判）
			// 只跳过明确的本地回环地址
			if ip.IsLoopback() {
				return sql.NullString{String: "本地", Valid: true}
			}
			// 对于IPv6，不跳过LinkLocal地址，因为某些移动网络的IPv6可能是LinkLocal格式但仍然是公网地址
			// 让GeoIP数据库来判断是否为可解析的公网地址
		}
	}

	// 首先尝试使用本地数据库
	location, err := GetLocation(ipAddress)
	if err != nil || location == nil || location.Country == "" {
		// 本地数据库失败，尝试使用 ping0.cc API
		ping0Location, err2 := GetLocationFromPing0(ipAddress)
		if err2 == nil && ping0Location != nil && ping0Location.Country != "" {
			location = ping0Location
		} else {
			// 如果都失败了，返回无效
			return sql.NullString{Valid: false}
		}
	}

	// 格式化为 JSON 字符串
	locationJSON, err := json.Marshal(location)
	if err != nil {
		// 如果 JSON 序列化失败，使用简单格式
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

// GetLocationSimple 获取简单的位置字符串（国家, 城市）
// 优先使用本地数据库，失败时尝试使用 ping0.cc API
func GetLocationSimple(ipAddress string) string {
	// 首先尝试使用本地数据库
	location, err := GetLocation(ipAddress)
	if err != nil || location == nil || location.Country == "" {
		// 本地数据库失败，尝试使用 ping0.cc API
		ping0Location, err2 := GetLocationFromPing0(ipAddress)
		if err2 == nil && ping0Location != nil && ping0Location.Country != "" {
			location = ping0Location
		} else {
			return ""
		}
	}

	if location.City != "" {
		return fmt.Sprintf("%s, %s", location.Country, location.City)
	} else if location.Region != "" {
		return fmt.Sprintf("%s, %s", location.Country, location.Region)
	}
	return location.Country
}

// IsEnabled 检查 GeoIP 是否已启用
func IsEnabled() bool {
	geoipDBLock.RLock()
	defer geoipDBLock.RUnlock()
	return geoipEnabled
}

// Close 关闭 GeoIP 数据库
func Close() {
	geoipDBLock.Lock()
	defer geoipDBLock.Unlock()

	if geoipDB != nil {
		geoipDB.Close()
		geoipDB = nil
	}
	geoipEnabled = false
}

// GetLocationFromIPW 从 ipw.cn 网站获取 IPv6 地址的地理位置信息
// 作为本地数据库的补充，提供更精确的 IPv6 地址解析
func GetLocationFromIPW(ipAddress string) (*LocationInfo, error) {
	// 只处理 IPv6 地址
	parsedIP := net.ParseIP(ipAddress)
	if parsedIP == nil {
		return nil, fmt.Errorf("无效的IP地址格式")
	}

	// 如果是 IPv4，跳过（IPv4 使用本地数据库即可）
	if parsedIP.To4() != nil {
		return nil, fmt.Errorf("仅支持 IPv6 地址")
	}

	// 跳过本地地址
	if ipAddress == "::1" || ipAddress == "localhost" {
		return nil, fmt.Errorf("本地地址，跳过解析")
	}

	// 构建查询 URL
	url := fmt.Sprintf("https://ipw.cn/ipv6/?ip=%s", ipAddress)

	// 创建 HTTP 客户端，设置超时
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// 发送请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置 User-Agent，模拟浏览器访问
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

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	htmlContent := string(body)

	// 解析 HTML 内容，提取地理位置信息
	location := &LocationInfo{}

	// 方式1: 查找 JSON 数据（网站可能使用 API 返回 JSON）
	// 尝试查找 window.__INITIAL_STATE__ 或其他 JSON 数据
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
					// 解析完整的位置字符串（如：中国 湖南 长沙 宁乡）
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

	// 方式2: 查找 HTML 文本中的地理位置信息
	// 匹配格式：中国 湖南 长沙 宁乡 或 中国,湖南,长沙,宁乡
	if location.Country == "" {
		locationPatterns := []*regexp.Regexp{
			// 匹配：中国 湖南 长沙 宁乡
			regexp.MustCompile(`(中国|China)\s+([\u4e00-\u9fa5]+)\s+([\u4e00-\u9fa5]+)\s+([\u4e00-\u9fa5]+)`),
			// 匹配：中国 湖南 长沙
			regexp.MustCompile(`(中国|China)\s+([\u4e00-\u9fa5]+)\s+([\u4e00-\u9fa5]+)`),
			// 匹配：中国,湖南,长沙,宁乡
			regexp.MustCompile(`(中国|China)[,，]\s*([\u4e00-\u9fa5]+)[,，]\s*([\u4e00-\u9fa5]+)[,，]\s*([\u4e00-\u9fa5]+)`),
		}

		for _, pattern := range locationPatterns {
			matches := pattern.FindAllStringSubmatch(htmlContent, -1)
			for _, match := range matches {
				if len(match) >= 3 {
					// 检查是否包含地理位置关键词
					locationText := match[0]
					if strings.Contains(locationText, "中国") || strings.Contains(locationText, "省") ||
						strings.Contains(locationText, "市") || strings.Contains(locationText, "县") ||
						strings.Contains(locationText, "区") || strings.Contains(locationText, "乡") {
						// 解析位置信息
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

	// 如果找到了位置信息，返回结果
	if location.Country != "" {
		// 设置国家代码
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

// GetLocationFromPing0 从 ping0.cc API 获取 IP 地址的地理位置信息
// 使用免费API：curl ping0.cc/geo?ip=IP地址
// 返回格式：4行文本，第一行IP，第二行位置，第三行ASN，第四行商家
func GetLocationFromPing0(ipAddress string) (*LocationInfo, error) {
	// 跳过本地地址
	if ipAddress == "127.0.0.1" || ipAddress == "::1" || ipAddress == "localhost" {
		return nil, fmt.Errorf("本地地址，跳过解析")
	}

	// 解析IP地址
	parsedIP := net.ParseIP(ipAddress)
	if parsedIP == nil {
		return nil, fmt.Errorf("无效的IP地址格式: %s", ipAddress)
	}

	// 构建查询 URL（使用免费API）
	url := fmt.Sprintf("https://ping0.cc/geo?ip=%s", ipAddress)

	// 创建 HTTP 客户端，设置超时
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// 发送请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置 User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
	}

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析响应（4行文本格式）
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("响应格式不正确")
	}

	// 第二行是位置信息，格式如：美国 华盛顿州 西雅圖 — 斯巴达
	locationStr := strings.TrimSpace(lines[1])
	if locationStr == "" {
		return nil, fmt.Errorf("位置信息为空")
	}

	// 解析位置字符串
	location := &LocationInfo{}

	// 移除可能的商家名称（— 后面的内容）
	parts := strings.Split(locationStr, "—")
	locationParts := strings.TrimSpace(parts[0])

	// 按空格分割位置信息
	locationFields := strings.Fields(locationParts)
	if len(locationFields) == 0 {
		return nil, fmt.Errorf("无法解析位置信息")
	}

	// 第一段通常是国家
	location.Country = locationFields[0]

	// 设置国家代码（简单映射）
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

	// 第二段可能是省份/州
	if len(locationFields) >= 2 {
		location.Region = locationFields[1]
	}

	// 第三段及之后是城市
	if len(locationFields) >= 3 {
		location.City = strings.Join(locationFields[2:], " ")
	}

	// 如果解析成功，返回结果
	if location.Country != "" {
		return location, nil
	}

	return nil, fmt.Errorf("未能从API解析到地理位置信息")
}

// GetLocationWithFallback 获取地理位置信息，优先使用本地数据库，失败时尝试从 ping0.cc 或 ipw.cn 获取
func GetLocationWithFallback(ipAddress string) (*LocationInfo, error) {
	// 首先尝试使用本地数据库
	location, err := GetLocation(ipAddress)
	if err == nil && location != nil && location.Country != "" {
		// 本地数据库解析成功，直接返回
		return location, nil
	}

	// 本地数据库解析失败，尝试使用 ping0.cc API（支持IPv4和IPv6）
	ping0Location, err := GetLocationFromPing0(ipAddress)
	if err == nil && ping0Location != nil && ping0Location.Country != "" {
		return ping0Location, nil
	}

	// 如果是 IPv6 地址且 ping0.cc 也失败，尝试从 ipw.cn 获取（作为最后的fallback）
	parsedIP := net.ParseIP(ipAddress)
	if parsedIP != nil && parsedIP.To4() == nil {
		// 是 IPv6 地址
		ipwLocation, err := GetLocationFromIPW(ipAddress)
		if err == nil && ipwLocation != nil {
			return ipwLocation, nil
		}
	}

	// 如果都失败了，返回原始错误
	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("无法解析地理位置信息")
}
