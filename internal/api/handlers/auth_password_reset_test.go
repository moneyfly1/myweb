package handlers

import (
	"cboard-go/internal/core/auth"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAuthPasswordResetTestDB(t *testing.T) *gorm.DB {
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
	if err := db.AutoMigrate(
		&models.User{},
		&models.SystemConfig{},
		&models.VerificationCode{},
		&models.EmailQueue{},
		&models.Notification{},
		&models.AuditLog{},
	); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}
	database.DB = db
	return db
}

func performAuthPasswordResetRequest(method string, routePath string, body string, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, routePath, handler)
	req := httptest.NewRequest(method, routePath, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func performAuthPasswordResetRequestAsUser(method string, routePath string, body string, user *models.User, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, routePath, func(c *gin.Context) {
		c.Set("user", user)
		c.Set("user_id", user.ID)
		handler(c)
	})
	req := httptest.NewRequest(method, routePath, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestForgotPasswordQueuesResetEmail(t *testing.T) {
	db := setupAuthPasswordResetTestDB(t)
	user := models.User{
		Username: "reset_user",
		Email:    "reset@example.com",
		Password: "old-password-hash",
		IsActive: true,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	recorder := performAuthPasswordResetRequest(
		http.MethodPost,
		"/auth/forgot-password",
		`{"email":"RESET@example.com"}`,
		ForgotPassword,
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var code models.VerificationCode
	if err := db.Where("email = ? AND purpose = ?", "reset@example.com", "reset_password").First(&code).Error; err != nil {
		t.Fatalf("expected reset verification code: %v", err)
	}
	if code.Code == "" || code.Used != 0 {
		t.Fatalf("unexpected verification code state: code=%q used=%d", code.Code, code.Used)
	}

	var queued models.EmailQueue
	if err := db.Where("to_email = ? AND email_type = ?", "reset@example.com", "password_reset").First(&queued).Error; err != nil {
		t.Fatalf("expected queued password reset email: %v", err)
	}
	if !strings.Contains(queued.Content, code.Code) {
		t.Fatalf("queued email content does not include verification code")
	}
}

func TestResetPasswordByCodeUpdatesPasswordAndConsumesCode(t *testing.T) {
	db := setupAuthPasswordResetTestDB(t)
	oldHash, err := auth.HashPassword("OldPass1!")
	if err != nil {
		t.Fatal(err)
	}
	user := models.User{
		Username: "reset_user",
		Email:    "reset@example.com",
		Password: oldHash,
		IsActive: true,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	verificationCode := models.VerificationCode{
		Email:     "reset@example.com",
		Code:      "123456",
		ExpiresAt: utils.GetBeijingTime().Add(10 * time.Minute),
		Used:      0,
		Purpose:   "reset_password",
	}
	if err := db.Create(&verificationCode).Error; err != nil {
		t.Fatal(err)
	}

	recorder := performAuthPasswordResetRequest(
		http.MethodPost,
		"/auth/reset-password",
		`{"email":"reset@example.com","verification_code":"123456","new_password":"NewPass1!"}`,
		ResetPasswordByCode,
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("expected success response, got: %s", response.Message)
	}

	var updatedUser models.User
	if err := db.First(&updatedUser, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updatedUser.Password == oldHash {
		t.Fatalf("expected password hash to change")
	}
	if !auth.VerifyPassword("NewPass1!", updatedUser.Password) {
		t.Fatalf("new password hash does not verify")
	}

	var usedCode models.VerificationCode
	if err := db.First(&usedCode, verificationCode.ID).Error; err != nil {
		t.Fatal(err)
	}
	if usedCode.Used != 1 {
		t.Fatalf("expected verification code to be marked used, got %d", usedCode.Used)
	}
}

func TestChangePasswordUsesConfiguredMinPasswordLength(t *testing.T) {
	db := setupAuthPasswordResetTestDB(t)
	currentHash, err := auth.HashPassword("OldPass1!")
	if err != nil {
		t.Fatal(err)
	}
	user := models.User{
		Username: "change_user",
		Email:    "change@example.com",
		Password: currentHash,
		IsActive: true,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.SystemConfig{
		Key:      "min_password_length",
		Value:    "12",
		Category: "registration",
	}).Error; err != nil {
		t.Fatal(err)
	}

	recorder := performAuthPasswordResetRequestAsUser(
		http.MethodPost,
		"/users/change-password",
		`{"current_password":"OldPass1!","new_password":"NewPass1!"}`,
		&user,
		ChangePassword,
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Success {
		t.Fatalf("expected failure response")
	}
	if response.Message != "密码长度至少12位" {
		t.Fatalf("expected configured min length message, got %q", response.Message)
	}
}

func TestChangePasswordMissingCurrentPasswordMessage(t *testing.T) {
	db := setupAuthPasswordResetTestDB(t)
	currentHash, err := auth.HashPassword("OldPass1!")
	if err != nil {
		t.Fatal(err)
	}
	user := models.User{
		Username: "change_user",
		Email:    "change@example.com",
		Password: currentHash,
		IsActive: true,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	recorder := performAuthPasswordResetRequestAsUser(
		http.MethodPost,
		"/users/change-password",
		`{"new_password":"NewPass1!"}`,
		&user,
		ChangePassword,
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Message != "请输入当前密码" {
		t.Fatalf("expected missing current password message, got %q", response.Message)
	}
}
