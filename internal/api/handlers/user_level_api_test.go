package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 等级功能接口测试（2026-10-28）：
//
// 覆盖三件用户可见的事：
//  1. 后台「手动添加等级」能正常落库（名称/门槛/折扣/颜色）；
//  2. 后台「添加用户」可以手工指定等级（不传则留空，由累计消费自动匹配）；
//  3. 等级列表可按 active_only 过滤（前台下拉只给启用中的等级）。
func setupUserLevelAPIEnv(t *testing.T) *gorm.DB {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.UserLevel{}, &models.SystemConfig{},
		&models.Subscription{}, &models.AuditLog{}, &models.RegistrationLog{},
	); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	old := database.DB
	database.DB = db
	t.Cleanup(func() {
		database.DB = old
		if s, err := db.DB(); err == nil {
			s.Close()
		}
	})
	return db
}

func doJSON(t *testing.T, method, path, body string, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler(c)
	return w
}

func decodeLevel(t *testing.T, w *httptest.ResponseRecorder) models.UserLevel {
	t.Helper()
	var resp struct {
		Success bool             `json:"success"`
		Data    models.UserLevel `json:"data"`
		Message string           `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v body=%s", err, w.Body.String())
	}
	if !resp.Success {
		t.Fatalf("接口返回失败: %s", resp.Message)
	}
	return resp.Data
}

// 后台手动添加等级：字段完整落库，颜色/折扣原样返回
func TestCreateUserLevel_manual(t *testing.T) {
	db := setupUserLevelAPIEnv(t)

	body := `{"level_name":"VIP1","level_order":1,"min_consumption":200,"discount_rate":0.99,"color":"#909399","benefits":"{\"priority\":false}","is_active":true}`
	w := doJSON(t, http.MethodPost, "/api/v1/admin/user-levels", body, CreateUserLevel)
	if w.Code != http.StatusCreated {
		t.Fatalf("期望 201，实际 %d：%s", w.Code, w.Body.String())
	}
	got := decodeLevel(t, w)
	if got.LevelName != "VIP1" || got.MinConsumption != 200 || got.DiscountRate != 0.99 || got.Color != "#909399" {
		t.Fatalf("返回字段不符: %+v", got)
	}

	var saved models.UserLevel
	if err := db.First(&saved, got.ID).Error; err != nil {
		t.Fatalf("落库失败: %v", err)
	}
	if saved.LevelOrder != 1 || !saved.IsActive || saved.Color != "#909399" {
		t.Fatalf("落库字段不符: %+v", saved)
	}

	// 重名 / 重排序必须被拒，否则前台分组会错乱
	if w2 := doJSON(t, http.MethodPost, "/api/v1/admin/user-levels", body, CreateUserLevel); w2.Code != http.StatusBadRequest {
		t.Fatalf("重复等级名应返回 400，实际 %d", w2.Code)
	}
}

// 等级列表：active_only=true 只回启用中的等级（下拉数据源）
func TestGetAdminUserLevels_activeOnly(t *testing.T) {
	db := setupUserLevelAPIEnv(t)
	db.Create(&models.UserLevel{LevelName: "VIP1", LevelOrder: 1, MinConsumption: 200, DiscountRate: 0.99, Color: "#909399", IsActive: true})
	vip2 := models.UserLevel{LevelName: "VIP2", LevelOrder: 2, MinConsumption: 400, DiscountRate: 0.98, Color: "#67C23A", IsActive: true}
	db.Create(&vip2)
	db.Model(&models.UserLevel{}).Where("id = ?", vip2.ID).Update("is_active", false)

	w := doJSON(t, http.MethodGet, "/api/v1/admin/user-levels?active_only=true", "", GetAdminUserLevels)
	var resp struct {
		Data []models.UserLevel `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析失败: %v body=%s", err, w.Body.String())
	}
	if len(resp.Data) != 1 || resp.Data[0].LevelName != "VIP1" {
		t.Fatalf("active_only 过滤失败，返回 %d 条: %+v", len(resp.Data), resp.Data)
	}
	// 颜色字段必须随响应返回（前端用它渲染等级色）
	if resp.Data[0].Color != "#909399" {
		t.Fatalf("颜色未返回: %+v", resp.Data[0])
	}
}

// 后台「添加用户 → 用户等级」的落库逻辑：手工指定 VIP1 生效、0 表示清空。
// （CreateUser 处理器本体会在返回后异步发邮件/通知，测试环境跑不了；
//
//	这里直接覆盖它使用的同一个 helper，接口层由后台页面人工确认。）
func TestSetUserLevelFromID(t *testing.T) {
	db := setupUserLevelAPIEnv(t)
	vip1 := models.UserLevel{LevelName: "VIP1", LevelOrder: 1, MinConsumption: 200, DiscountRate: 0.99, Color: "#909399", IsActive: true}
	db.Create(&vip1)

	user := models.User{Username: "u", Email: "u@test.local", Password: "x"}
	db.Create(&user)

	if err := setUserLevelFromID(db, &user, vip1.ID); err != nil {
		t.Fatalf("指定等级失败: %v", err)
	}
	if !user.UserLevelID.Valid || user.UserLevelID.Int64 != int64(vip1.ID) {
		t.Fatalf("手工开通等级未生效: %+v", user.UserLevelID)
	}
	if err := db.Save(&user).Error; err != nil {
		t.Fatalf("保存失败: %v", err)
	}
	var reloaded models.User
	db.First(&reloaded, user.ID)
	if !reloaded.UserLevelID.Valid || reloaded.UserLevelID.Int64 != int64(vip1.ID) {
		t.Fatalf("落库后等级丢失: %+v", reloaded.UserLevelID)
	}

	// 0 = 清空等级
	if err := setUserLevelFromID(db, &user, 0); err != nil {
		t.Fatalf("清空等级失败: %v", err)
	}
	if user.UserLevelID.Valid {
		t.Fatalf("0 应清空等级: %+v", user.UserLevelID)
	}

	// 不存在的等级必须报错，不能静默挂空
	if err := setUserLevelFromID(db, &user, 9999); err == nil {
		t.Fatal("不存在的等级应返回错误")
	}
}
