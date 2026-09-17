package handlers

import (
	"strings"
	"testing"

	"cboard-go/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupUsernameTestDB 创建内存 SQLite 测试库（含 users 表）。
func setupUsernameTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}

func createTestUser(t *testing.T, db *gorm.DB, username string) models.User {
	t.Helper()
	u := models.User{Username: username, Email: strings.ToLower(username) + "@example.com", Password: "x", IsActive: true}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("创建用户 %s 失败: %v", username, err)
	}
	return u
}

// TestUsernameTakenIgnoresCase 用户名唯一性必须忽略大小写：
// 数据库中唯一的索引区分大小写，仅靠它会出现 admin / ADMIN 两个肉眼无法区分的账号。
func TestUsernameTakenIgnoresCase(t *testing.T) {
	db := setupUsernameTestDB(t)
	createTestUser(t, db, "alex")

	for _, variant := range []string{"alex", "Alex", "ALEX", "aLex"} {
		taken, err := usernameTaken(db, variant, 0)
		if err != nil {
			t.Fatalf("查询 %s 失败: %v", variant, err)
		}
		if !taken {
			t.Errorf("变体 %q 应判定为已占用", variant)
		}
	}

	for _, free := range []string{"alex2", "alexa", "ale"} {
		taken, err := usernameTaken(db, free, 0)
		if err != nil {
			t.Fatalf("查询 %s 失败: %v", free, err)
		}
		if taken {
			t.Errorf("%q 不应判定为已占用", free)
		}
	}
}

// TestUsernameTakenExcludesSelf 改资料/后台编辑时，与自己同名的写入必须放行。
func TestUsernameTakenExcludesSelf(t *testing.T) {
	db := setupUsernameTestDB(t)
	self := createTestUser(t, db, "alex")

	taken, err := usernameTaken(db, "Alex", self.ID)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if taken {
		t.Error("排除自身后不应判定为已占用")
	}

	// 但与他人冲突仍要拦住
	other := createTestUser(t, db, "bob")
	taken, err = usernameTaken(db, "BOB", other.ID+1)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if !taken {
		t.Error("bob 存在时 BOB 应判定为已占用")
	}
}

// TestCheckUsernameAvailableRejectsBypass 覆盖历史漏洞：
// 改用户名接口此前不做格式校验，且唯一性区分大小写，
// 导致可以把自己改成 "ADMIN"、"admin "（尾随空格）或 HTML 片段。
func TestCheckUsernameAvailableRejectsBypass(t *testing.T) {
	db := setupUsernameTestDB(t)
	createTestUser(t, db, "admin")

	rejected := []struct {
		name   string
		reason string
	}{
		{"admin", "与现有用户同名"},
		{"ADMIN", "仅大小写不同（历史可绕过）"},
		{"Admin", "仅大小写不同（历史可绕过）"},
		{"admin ", "尾随空格（历史可绕过）"},
		{" admin", "前导空格（历史可绕过）"},
		{"<b>hack</b>", "HTML 片段（历史可绕过）"},
		{"a", "长度不足"},
		{strings.Repeat("a", 21), "长度超限"},
		{"emoji😀", "非法字符"},
	}
	for _, tc := range rejected {
		msg, err := checkUsernameAvailable(db, tc.name, 0)
		if err != nil {
			t.Fatalf("校验 %q 出错: %v", tc.name, err)
		}
		if msg == "" {
			t.Errorf("%q 应被拒绝（%s）", tc.name, tc.reason)
			continue
		}
		if !strings.Contains(msg, "格式不正确") && !strings.Contains(msg, "已被使用") {
			t.Errorf("%q 的提示不明确: %q", tc.name, msg)
		}
	}
}

// TestCheckUsernameAvailableAcceptsValid 合法用户名必须放行。
func TestCheckUsernameAvailableAcceptsValid(t *testing.T) {
	db := setupUsernameTestDB(t)
	createTestUser(t, db, "alex")

	for _, name := range []string{"张三", "用户_01", "newuser", "NewUser2", "ab"} {
		if strings.EqualFold(name, "alex") {
			t.Fatalf("用例 %q 与已存在用户冲突", name)
		}
		msg, err := checkUsernameAvailable(db, name, 0)
		if err != nil {
			t.Fatalf("校验 %q 出错: %v", name, err)
		}
		if msg != "" {
			t.Errorf("合法用户名 %q 被拒绝: %s", name, msg)
		}
	}
}
