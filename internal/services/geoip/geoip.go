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
		geoipDB.Close()
		geoipDB = nil
	}

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

	if dbPath == "" {
		geoipEnabled = false
		return fmt.Errorf("未找到 GeoIP 数据库文件，地理位置解析功能已禁用")
	}

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		geoipEnabled = false
		return fmt.Errorf("GeoIP 数据库文件不存在: %s", dbPath)
	}

	db, err := geoip2.Open(dbPath)
	if err != nil {
		geoipEnabled = false
		return fmt.Errorf("打开 GeoIP 数据库失败: %w", err)
	}

	geoipDB = db
	geoipEnabled = true
	return nil
}

func GetLocation(ipAddress string) (*LocationInfo, error) {
	if !geoipEnabled || geoipDB == nil {
		return nil, fmt.Errorf("GeoIP 未启用")
	}

	if len(ipAddress) > 7 && ipAddress[:7] == "::ffff:" {
		ipAddress = ipAddress[7:]
	}

	if ipAddress == "127.0.0.1" || ipAddress == "::1" || ipAddress == "localhost" {
		return nil, fmt.Errorf("本地地址，跳过解析")
	}

	parsedIP := net.ParseIP(ipAddress)
	if parsedIP == nil {
		return nil, fmt.Errorf("无效的IP地址格式: %s", ipAddress)
	}

	geoipDBLock.RLock()
	defer geoipDBLock.RUnlock()

	record, err := geoipDB.City(parsedIP)
	if err != nil {
		return nil, fmt.Errorf("GeoIP解析失败: %w", err)
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

	if record.Location.Latitude != 0 || record.Location.Longitude != 0 {
		location.Latitude = record.Location.Latitude
		location.Longitude = record.Location.Longitude
	}

	if record.Location.TimeZone != "" {
		location.Timezone = record.Location.TimeZone
	}

	return location, nil
}

func GetLocationString(ipAddress string) sql.NullString {
	if ipAddress == "127.0.0.1" || ipAddress == "::1" || ipAddress == "localhost" {
		return sql.NullString{String: "本地", Valid: true}
	}

	ip := net.ParseIP(ipAddress)
	if ip != nil {
		if ip.To4() != nil {
			if ip.IsLoopback() || ip.IsPrivate() {
				return sql.NullString{String: "内网", Valid: true}
			}
		} else {
			if ip.IsLoopback() {
				return sql.NullString{String: "本地", Valid: true}
			}
		}
	}

	location, err := GetLocation(ipAddress)
	if err != nil || location == nil || location.Country == "" {
		ping0Location, err2 := GetLocationFromPing0(ipAddress)
		if err2 == nil && ping0Location != nil && ping0Location.Country != "" {
			location = ping0Location
		} else {
			return sql.NullString{Valid: false}
		}
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

func GetLocationSimple(ipAddress string) string {
	location, err := GetLocation(ipAddress)
	if err != nil || location == nil || location.Country == "" {
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

func IsEnabled() bool {
	geoipDBLock.RLock()
	defer geoipDBLock.RUnlock()
	return geoipEnabled
}

func Close() {
	geoipDBLock.Lock()
	defer geoipDBLock.Unlock()

	if geoipDB != nil {
		geoipDB.Close()
		geoipDB = nil
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
			regexp.MustCompile(`(中国|China)\s+([\u4e00-\u9fa5]+)\s+([\u4e00-\u9fa5]+)\s+([\u4e00-\u9fa5]+)`),
			regexp.MustCompile(`(中国|China)\s+([\u4e00-\u9fa5]+)\s+([\u4e00-\u9fa5]+)`),
			regexp.MustCompile(`(中国|China)[,，]\s*([\u4e00-\u9fa5]+)[,，]\s*([\u4e00-\u9fa5]+)[,，]\s*([\u4e00-\u9fa5]+)`),
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

	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("响应格式不正确")
	}

	locationStr := strings.TrimSpace(lines[1])
	if locationStr == "" {
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
		return location, nil
	}

	return nil, fmt.Errorf("未能从API解析到地理位置信息")
}

func GetLocationWithFallback(ipAddress string) (*LocationInfo, error) {
	location, err := GetLocation(ipAddress)
	if err == nil && location != nil && location.Country != "" {
		return location, nil
	}

	ping0Location, err := GetLocationFromPing0(ipAddress)
	if err == nil && ping0Location != nil && ping0Location.Country != "" {
		return ping0Location, nil
	}

	parsedIP := net.ParseIP(ipAddress)
	if parsedIP != nil && parsedIP.To4() == nil {
		ipwLocation, err := GetLocationFromIPW(ipAddress)
		if err == nil && ipwLocation != nil {
			return ipwLocation, nil
		}
	}

	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("无法解析地理位置信息")
}
