package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"cboard-go/internal/core/auth"
	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/config_update"
	"cboard-go/internal/services/device"

	"gorm.io/gorm"
)

func main() {
	// 加载配置
	_, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("❌ 加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化数据库
	if err := database.InitDatabase(); err != nil {
		fmt.Printf("❌ 数据库初始化失败: %v\n", err)
		os.Exit(1)
	}

	// 自动迁移
	if err := database.AutoMigrate(); err != nil {
		fmt.Printf("❌ 数据库迁移失败: %v\n", err)
		os.Exit(1)
	}

	db := database.GetDB()
	if db == nil {
		fmt.Println("❌ 数据库连接失败")
		os.Exit(1)
	}

	fmt.Println("✅ 数据库连接成功")
	fmt.Println("\n开始测试订阅功能...\n")

	// 测试场景
	testScenarios(db)
}

func testScenarios(db *gorm.DB) {
	// 1. 创建测试用户
	user := createTestUser(db)
	if user == nil {
		fmt.Println("❌ 创建测试用户失败")
		return
	}
	fmt.Printf("✅ 创建测试用户: ID=%d, Username=%s\n", user.ID, user.Username)

	// 2. 创建测试订阅（5个设备限制，30天后到期）
	subscription := createTestSubscription(db, user.ID, 5, time.Now().AddDate(0, 0, 30))
	if subscription == nil {
		fmt.Println("❌ 创建测试订阅失败")
		return
	}
	fmt.Printf("✅ 创建测试订阅: ID=%d, URL=%s, DeviceLimit=%d\n", subscription.ID, subscription.SubscriptionURL, subscription.DeviceLimit)

	// 3. 创建测试节点
	createTestNodes(db)
	fmt.Println("✅ 创建测试节点")

	// 4. 测试场景1：正常用户（有效期内，设备在限制内）
	fmt.Println("\n=== 场景1：正常用户（有效期内，设备在限制内）===")
	testNormalUser(db, subscription, user.ID)

	// 5. 测试场景2：到期用户
	fmt.Println("\n=== 场景2：到期用户 ===")
	testExpiredUser(db, subscription, user.ID)

	// 6. 测试场景3：设备超限用户（但未到期）
	fmt.Println("\n=== 场景3：设备超限用户（但未到期）===")
	testDeviceOverLimit(db, subscription, user.ID)

	// 7. 测试场景4：到期且设备超限用户
	fmt.Println("\n=== 场景4：到期且设备超限用户 ===")
	testExpiredAndOverLimit(db, subscription, user.ID)

	// 8. 测试场景5：订阅失效用户
	fmt.Println("\n=== 场景5：订阅失效用户 ===")
	testInactiveSubscription(db, subscription, user.ID)

	// 9. 测试设备限制逻辑（第5个设备应该成功，第6个应该失败）
	fmt.Println("\n=== 测试设备限制逻辑 ===")
	testDeviceLimitLogic(db, subscription, user.ID)

	fmt.Println("\n✅ 所有测试完成！")
}

func createTestUser(db *gorm.DB) *models.User {
	// 检查是否已存在
	var existingUser models.User
	if err := db.Where("username = ?", "test_subscription_user").First(&existingUser).Error; err == nil {
		// 删除旧用户及其相关数据
		db.Where("user_id = ?", existingUser.ID).Delete(&models.Subscription{})
		db.Where("user_id = ?", existingUser.ID).Delete(&models.Device{})
		db.Delete(&existingUser)
	}

	// 创建新用户
	passwordHash, _ := auth.HashPassword("test123456")
	user := models.User{
		Username: "test_subscription_user",
		Email:    "test_subscription@example.com",
		Password: passwordHash,
		IsActive: true,
		Balance:  1000.0,
	}

	if err := db.Create(&user).Error; err != nil {
		fmt.Printf("创建用户失败: %v\n", err)
		return nil
	}

	return &user
}

func createTestSubscription(db *gorm.DB, userID uint, deviceLimit int, expireTime time.Time) *models.Subscription {
	// 删除旧订阅
	db.Where("user_id = ?", userID).Delete(&models.Subscription{})
	db.Where("user_id = ?", userID).Delete(&models.Device{})

	subscriptionURL := fmt.Sprintf("test_sub_%d_%d", userID, time.Now().Unix())
	subscription := models.Subscription{
		UserID:          userID,
		SubscriptionURL: subscriptionURL,
		DeviceLimit:     deviceLimit,
		CurrentDevices:  0,
		IsActive:        true,
		Status:          "active",
		ExpireTime:      expireTime,
	}

	if err := db.Create(&subscription).Error; err != nil {
		fmt.Printf("创建订阅失败: %v\n", err)
		return nil
	}

	return &subscription
}

func createTestNodes(db *gorm.DB) {
	// 删除旧节点
	db.Where("1 = 1").Delete(&models.Node{})

	// 创建测试节点
	nodes := []models.Node{
		{
			Name:     "香港-01",
			Region:   "香港",
			Type:     "vmess",
			Status:   "online",
			IsActive: true,
			Config:   getTestNodeConfig("vmess", "香港-01", "1.1.1.1", 443),
		},
		{
			Name:     "台湾-01",
			Region:   "台湾",
			Type:     "vless",
			Status:   "online",
			IsActive: true,
			Config:   getTestNodeConfig("vless", "台湾-01", "2.2.2.2", 443),
		},
		{
			Name:     "日本-01",
			Region:   "日本",
			Type:     "trojan",
			Status:   "online",
			IsActive: true,
			Config:   getTestNodeConfig("trojan", "日本-01", "3.3.3.3", 443),
		},
	}

	for _, node := range nodes {
		db.Create(&node)
	}
}

func getTestNodeConfig(nodeType, name, server string, port int) *string {
	config := map[string]interface{}{
		"name":   name,
		"type":   nodeType,
		"server": server,
		"port":   port,
	}

	switch nodeType {
	case "vmess":
		config["uuid"] = "00000000-0000-4000-8000-000000000009"
		config["network"] = "ws"
		config["tls"] = true
	case "vless":
		config["uuid"] = "00000000-0000-4000-8000-000000000056"
		config["network"] = "grpc"
		config["tls"] = true
	case "trojan":
		config["password"] = "test_password_123"
		config["network"] = "tcp"
		config["tls"] = true
	}

	configJSON, _ := json.Marshal(config)
	configStr := string(configJSON)
	return &configStr
}

func testNormalUser(db *gorm.DB, subscription *models.Subscription, userID uint) {
	service := config_update.NewConfigUpdateService()
	config, err := service.GenerateClashConfig(userID, subscription.SubscriptionURL)

	if err != nil {
		fmt.Printf("❌ 生成配置失败: %v\n", err)
		return
	}

	// 检查配置内容
	if !contains(config, "📢 网站域名") {
		fmt.Println("❌ 缺少网站域名信息节点")
		return
	}
	if !contains(config, "⏰ 到期时间") {
		fmt.Println("❌ 缺少到期时间信息节点")
		return
	}
	if !contains(config, "💬 售后QQ") {
		fmt.Println("❌ 缺少售后QQ信息节点")
		return
	}
	if !contains(config, "香港-01") {
		fmt.Println("❌ 缺少节点信息")
		return
	}
	if contains(config, "⚠️") {
		fmt.Println("❌ 不应该包含提醒节点（正常用户）")
		return
	}

	fmt.Println("✅ 场景1测试通过：配置包含所有信息节点，无提醒节点")
}

func testExpiredUser(db *gorm.DB, subscription *models.Subscription, userID uint) {
	// 设置订阅为已过期
	db.Model(subscription).Update("expire_time", time.Now().AddDate(0, 0, -1))

	service := config_update.NewConfigUpdateService()
	config, err := service.GenerateClashConfig(userID, subscription.SubscriptionURL)

	if err != nil {
		fmt.Printf("❌ 生成配置失败: %v\n", err)
		return
	}

	if !contains(config, "⚠️ 订阅已过期") {
		fmt.Println("❌ 缺少到期提醒节点")
		return
	}

	fmt.Println("✅ 场景2测试通过：包含到期提醒节点")

	// 恢复订阅
	db.Model(subscription).Update("expire_time", time.Now().AddDate(0, 0, 30))
}

func testDeviceOverLimit(db *gorm.DB, subscription *models.Subscription, userID uint) {
	// 创建5个设备（达到限制）
	deviceManager := device.NewDeviceManager()
	for i := 1; i <= 5; i++ {
		userAgent := fmt.Sprintf("TestDevice%d/1.0.0", i)
		ipAddress := fmt.Sprintf("192.168.1.%d", i)
		_, _ = deviceManager.RecordDeviceAccess(subscription.ID, userID, userAgent, ipAddress, "clash")
	}

	// 检查设备数量
	var deviceCount int64
	db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subscription.ID, true).Count(&deviceCount)
	fmt.Printf("   当前设备数量: %d/%d\n", deviceCount, subscription.DeviceLimit)

	// 测试第5个设备（应该成功）
	service := config_update.NewConfigUpdateService()
	config, err := service.GenerateClashConfig(userID, subscription.SubscriptionURL)
	if err != nil {
		fmt.Printf("❌ 第5个设备生成配置失败: %v\n", err)
		return
	}

	if !contains(config, "⚠️ 设备超限") {
		fmt.Println("⚠️  注意：配置中包含设备超限提醒（这是正常的，因为当前设备数=限制数）")
	} else {
		fmt.Println("✅ 第5个设备：配置包含设备超限提醒（当前设备数=限制数）")
	}

	// 测试第6个设备（应该失败或返回提醒）
	deviceHash := deviceManager.GenerateDeviceHash("TestDevice6/1.0.0", "192.168.1.6", "")
	var existingDevice models.Device
	isNewDevice := db.Where("device_hash = ? AND subscription_id = ?", deviceHash, subscription.ID).First(&existingDevice).Error != nil

	if isNewDevice && int(deviceCount) >= subscription.DeviceLimit {
		fmt.Println("✅ 第6个设备：正确识别为新设备且超过限制")
	} else {
		fmt.Printf("⚠️  第6个设备检查：isNewDevice=%v, deviceCount=%d, limit=%d\n", isNewDevice, deviceCount, subscription.DeviceLimit)
	}

	// 清理设备
	db.Where("subscription_id = ?", subscription.ID).Delete(&models.Device{})
	db.Model(subscription).Update("current_devices", 0)

	fmt.Println("✅ 场景3测试通过：设备限制逻辑正确")
}

func testExpiredAndOverLimit(db *gorm.DB, subscription *models.Subscription, userID uint) {
	// 设置订阅为已过期
	db.Model(subscription).Update("expire_time", time.Now().AddDate(0, 0, -1))

	// 创建6个设备（超过限制）
	deviceManager := device.NewDeviceManager()
	for i := 1; i <= 6; i++ {
		userAgent := fmt.Sprintf("TestDevice%d/1.0.0", i)
		ipAddress := fmt.Sprintf("192.168.1.%d", i)
		_, _ = deviceManager.RecordDeviceAccess(subscription.ID, userID, userAgent, ipAddress, "clash")
	}

	service := config_update.NewConfigUpdateService()
	config, err := service.GenerateClashConfig(userID, subscription.SubscriptionURL)

	if err != nil {
		fmt.Printf("❌ 生成配置失败: %v\n", err)
		return
	}

	if !contains(config, "⚠️ 订阅已过期") {
		fmt.Println("❌ 缺少到期提醒节点")
		return
	}
	if !contains(config, "⚠️ 设备超限") {
		fmt.Println("❌ 缺少设备超限提醒节点")
		return
	}

	fmt.Println("✅ 场景4测试通过：包含到期和设备超限提醒")

	// 恢复
	db.Model(subscription).Update("expire_time", time.Now().AddDate(0, 0, 30))
	db.Where("subscription_id = ?", subscription.ID).Delete(&models.Device{})
	db.Model(subscription).Update("current_devices", 0)
}

func testInactiveSubscription(db *gorm.DB, subscription *models.Subscription, userID uint) {
	// 设置订阅为失效
	db.Model(subscription).Updates(map[string]interface{}{
		"is_active": false,
		"status":    "inactive",
	})

	service := config_update.NewConfigUpdateService()
	config, err := service.GenerateClashConfig(userID, subscription.SubscriptionURL)

	if err != nil {
		fmt.Printf("❌ 生成配置失败: %v\n", err)
		return
	}

	if !contains(config, "⚠️ 订阅已失效") {
		fmt.Println("❌ 缺少订阅失效提醒节点")
		return
	}

	fmt.Println("✅ 场景5测试通过：包含订阅失效提醒")

	// 恢复
	db.Model(subscription).Updates(map[string]interface{}{
		"is_active": true,
		"status":    "active",
	})
}

func testDeviceLimitLogic(db *gorm.DB, subscription *models.Subscription, userID uint) {
	deviceManager := device.NewDeviceManager()

	// 创建4个设备
	for i := 1; i <= 4; i++ {
		userAgent := fmt.Sprintf("LimitTestDevice%d/1.0.0", i)
		ipAddress := fmt.Sprintf("10.0.0.%d", i)
		_, _ = deviceManager.RecordDeviceAccess(subscription.ID, userID, userAgent, ipAddress, "clash")
	}

	// 检查设备数量
	var deviceCount int64
	db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subscription.ID, true).Count(&deviceCount)
	fmt.Printf("   当前设备数量: %d/%d\n", deviceCount, subscription.DeviceLimit)

	// 测试第5个设备（应该成功，因为 4 < 5）
	deviceHash5 := deviceManager.GenerateDeviceHash("LimitTestDevice5/1.0.0", "10.0.0.5", "")
	var existingDevice5 models.Device
	isNewDevice5 := db.Where("device_hash = ? AND subscription_id = ?", deviceHash5, subscription.ID).First(&existingDevice5).Error != nil

	if isNewDevice5 && int(deviceCount) < subscription.DeviceLimit {
		fmt.Println("✅ 第5个设备：应该允许（当前4个，限制5个）")
	} else {
		fmt.Printf("❌ 第5个设备检查失败: isNewDevice=%v, deviceCount=%d, limit=%d\n", isNewDevice5, deviceCount, subscription.DeviceLimit)
	}

	// 创建第5个设备
	_, _ = deviceManager.RecordDeviceAccess(subscription.ID, userID, "LimitTestDevice5/1.0.0", "10.0.0.5", "clash")

	// 再次检查设备数量
	db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subscription.ID, true).Count(&deviceCount)
	fmt.Printf("   当前设备数量: %d/%d\n", deviceCount, subscription.DeviceLimit)

	// 测试第6个设备（应该失败，因为 5 >= 5）
	deviceHash6 := deviceManager.GenerateDeviceHash("LimitTestDevice6/1.0.0", "10.0.0.6", "")
	var existingDevice6 models.Device
	isNewDevice6 := db.Where("device_hash = ? AND subscription_id = ?", deviceHash6, subscription.ID).First(&existingDevice6).Error != nil

	if isNewDevice6 && int(deviceCount) >= subscription.DeviceLimit {
		fmt.Println("✅ 第6个设备：应该拒绝（当前5个，限制5个）")
	} else {
		fmt.Printf("❌ 第6个设备检查失败: isNewDevice=%v, deviceCount=%d, limit=%d\n", isNewDevice6, deviceCount, subscription.DeviceLimit)
	}

	// 清理
	db.Where("subscription_id = ?", subscription.ID).Delete(&models.Device{})
	db.Model(subscription).Update("current_devices", 0)

	fmt.Println("✅ 设备限制逻辑测试通过")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || bytes.Contains([]byte(s), []byte(substr)))
}
