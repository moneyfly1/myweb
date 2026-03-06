package main

import (
	"log"
	"os"
	"path/filepath"

	"cboard-go/internal/models"
	"cboard-go/internal/services/geoip"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// 获取项目根目录
	rootDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("获取当前目录失败: %v", err)
	}

	// 初始化数据库
	dbPath := filepath.Join(rootDir, "cboard.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 初始化 GeoIP
	geoipPath := filepath.Join(rootDir, "dbip-city-lite.mmdb")
	if _, err := os.Stat(geoipPath); os.IsNotExist(err) {
		geoipPath = filepath.Join(rootDir, "GeoLite2-City.mmdb")
	}

	if err := geoip.InitGeoIP(geoipPath); err != nil {
		log.Printf("警告: GeoIP 初始化失败: %v", err)
		log.Println("将跳过位置信息更新")
		return
	}

	log.Println("GeoIP 初始化成功")

	// 查询所有没有位置信息的设备
	var devices []models.Device
	if err := db.Where("location IS NULL OR location = ''").Find(&devices).Error; err != nil {
		log.Fatalf("查询设备失败: %v", err)
	}

	log.Printf("找到 %d 个需要更新位置信息的设备", len(devices))

	updated := 0
	failed := 0

	for i, device := range devices {
		if device.IPAddress == nil || *device.IPAddress == "" {
			continue
		}

		ipAddress := *device.IPAddress

		// 获取位置信息
		location := geoip.GetLocationWithCache(ipAddress)
		if location.Valid && location.String != "" {
			device.Location = &location.String
			if err := db.Model(&device).Update("location", location.String).Error; err != nil {
				log.Printf("更新设备 %d 位置失败: %v", device.ID, err)
				failed++
			} else {
				updated++
				if (i+1)%100 == 0 {
					log.Printf("已处理 %d/%d 个设备", i+1, len(devices))
				}
			}
		}
	}

	log.Printf("更新完成: 成功 %d 个, 失败 %d 个", updated, failed)
}
