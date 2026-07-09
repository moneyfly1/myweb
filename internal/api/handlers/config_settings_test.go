package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupConfigSettingsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName)), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open sqlite sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	if err := db.AutoMigrate(&models.SystemConfig{}, &models.AuditLog{}); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}
	database.DB = db
	return db
}

func performConfigSettingsRequest(method, routePath, body string, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, routePath, handler)
	req := httptest.NewRequest(method, routePath, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestAdminNotificationSettingsPersistNewSensitiveFields(t *testing.T) {
	db := setupConfigSettingsTestDB(t)

	saveBody := `{
		"admin_notification_enabled": "true",
		"admin_telegram_notification": "true",
		"admin_telegram_bot_token": "123456:telegram-token",
		"admin_telegram_chat_id": "@example_admin",
		"admin_bark_notification": "true",
		"admin_bark_server_url": "https://api.day.app",
		"admin_bark_device_key": "bark-device-key"
	}`
	saveRecorder := performConfigSettingsRequest(http.MethodPut, "/admin/settings/admin-notification", saveBody, UpdateAdminNotificationSystemSettings)
	if saveRecorder.Code != http.StatusOK {
		t.Fatalf("expected save status 200, got %d: %s", saveRecorder.Code, saveRecorder.Body.String())
	}

	var saved []models.SystemConfig
	if err := db.Where("category = ?", CatAdminNotification).Find(&saved).Error; err != nil {
		t.Fatalf("query saved configs: %v", err)
	}
	if len(saved) != 7 {
		t.Fatalf("expected 7 admin notification configs, got %d", len(saved))
	}
	for _, cfg := range saved {
		if cfg.Type == "" || cfg.DisplayName == "" {
			t.Fatalf("config %s missing metadata: type=%q display_name=%q", cfg.Key, cfg.Type, cfg.DisplayName)
		}
	}

	getRecorder := performConfigSettingsRequest(http.MethodGet, "/admin/settings", "", GetAdminSettings)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected get status 200, got %d: %s", getRecorder.Code, getRecorder.Body.String())
	}

	var response struct {
		Data map[string]map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(getRecorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	adminNotification := response.Data[CatAdminNotification]
	if got := adminNotification["admin_telegram_bot_token"]; got != "123456:telegram-token" {
		t.Fatalf("unexpected telegram token: %#v", got)
	}
	if got := adminNotification["admin_telegram_chat_id"]; got != "@example_admin" {
		t.Fatalf("unexpected telegram chat id: %#v", got)
	}
	if got := adminNotification["admin_bark_device_key"]; got != "bark-device-key" {
		t.Fatalf("unexpected bark device key: %#v", got)
	}
}
