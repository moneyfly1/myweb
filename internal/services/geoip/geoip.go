package geoip

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"

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

	geoipDBLock.RLock()
	defer geoipDBLock.RUnlock()

	record, err := geoipDB.City(net.ParseIP(ipAddress))
	if err != nil {
		return nil, fmt.Errorf("解析IP地址失败: %w", err)
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
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
			return sql.NullString{String: "内网", Valid: true}
		}
	}

	location, err := GetLocation(ipAddress)
	if err != nil {
		return sql.NullString{Valid: false}
	}

	// 格式化为 JSON 字符串
	locationJSON, err := json.Marshal(location)
	if err != nil {
		// 如果 JSON 序列化失败，使用简单格式
		locationStr := location.Country
		if location.City != "" {
			locationStr = location.Country + ", " + location.City
		}
		return sql.NullString{String: locationStr, Valid: true}
	}

	return sql.NullString{String: string(locationJSON), Valid: true}
}

// GetLocationSimple 获取简单的位置字符串（国家, 城市）
func GetLocationSimple(ipAddress string) string {
	location, err := GetLocation(ipAddress)
	if err != nil {
		return ""
	}

	if location.City != "" {
		return fmt.Sprintf("%s, %s", location.Country, location.City)
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
