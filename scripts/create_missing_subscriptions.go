package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"gorm.io/gorm"
)

func main() {
	_, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	if err := database.InitDatabase(); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	db := database.GetDB()
	if db == nil {
		log.Fatal("数据库连接失败")
	}

	var users []models.User
	if err := db.Find(&users).Error; err != nil {
		log.Fatalf("获取用户列表失败: %v", err)
	}

	fmt.Printf("找到 %d 个用户\n", len(users))

	createdCount := 0
	for _, user := range users {
		var existing models.Subscription
		if err := db.Where("user_id = ?", user.ID).First(&existing).Error; err == nil {
			fmt.Printf("用户 %s (ID: %d) 已有订阅，跳过\n", user.Username, user.ID)
			continue
		}

		deviceLimit, durationMonths := getDefaultSubscriptionSettings(db)

		subscriptionURL := utils.GenerateSubscriptionURL()

		nowUTC := time.Now().UTC()
		var expireTime time.Time
		if durationMonths <= 0 {
			expireTime = time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 23, 59, 59, 0, nowUTC.Location())
		} else {
			expireTime = nowUTC.AddDate(0, durationMonths, 0)
		}

		sub := models.Subscription{
			UserID:          user.ID,
			SubscriptionURL: subscriptionURL,
			DeviceLimit:     deviceLimit,
			CurrentDevices:  0,
			IsActive:        true,
			Status:          "active",
			ExpireTime:      expireTime,
		}

		if err := db.Create(&sub).Error; err != nil {
			log.Printf("为用户 %s (ID: %d) 创建订阅失败: %v\n", user.Username, user.ID, err)
			continue
		}

		fmt.Printf("✅ 为用户 %s (ID: %d) 创建订阅成功 (订阅ID: %d, 设备限制: %d, 到期时间: %s)\n",
			user.Username, user.ID, sub.ID, deviceLimit, expireTime.Format("2006-01-02 15:04:05"))
		createdCount++
	}

	fmt.Printf("\n完成！共创建了 %d 个订阅\n", createdCount)
}

func getDefaultSubscriptionSettings(db *gorm.DB) (deviceLimit int, durationMonths int) {
	deviceLimit = 0
	durationMonths = 0

	var deviceLimitConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "default_subscription_device_limit", "registration").First(&deviceLimitConfig).Error; err != nil {
		if err := db.Where("key = ? AND category = ?", "default_subscription_device_limit", "general").First(&deviceLimitConfig).Error; err == nil {
			if deviceLimitConfig.Value != "" {
				if limit, err := strconv.Atoi(deviceLimitConfig.Value); err == nil && limit >= 0 {
					deviceLimit = limit
				}
			}
		}
	} else {
		if deviceLimitConfig.Value != "" {
			if limit, err := strconv.Atoi(deviceLimitConfig.Value); err == nil && limit >= 0 {
				deviceLimit = limit
			}
		}
	}

	var durationConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "default_subscription_duration_months", "registration").First(&durationConfig).Error; err != nil {
		if err := db.Where("key = ? AND category = ?", "default_subscription_duration_months", "general").First(&durationConfig).Error; err == nil {
			if durationConfig.Value != "" {
				if months, err := strconv.Atoi(durationConfig.Value); err == nil && months >= 0 {
					durationMonths = months
				}
			}
		}
	} else {
		if durationConfig.Value != "" {
			if months, err := strconv.Atoi(durationConfig.Value); err == nil && months >= 0 {
				durationMonths = months
			}
		}
	}

	return deviceLimit, durationMonths
}
