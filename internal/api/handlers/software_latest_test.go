package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/ghrelease"
	"cboard-go/internal/services/software_sync"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupSoftwareLatestDB 内存库（仅 system_configs）：处理器会读 github token / 加速前缀配置，
// 空库时必须安全退回默认值而不是崩。
func setupSoftwareLatestDB(t *testing.T) {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.SystemConfig{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	prev := database.DB
	database.DB = db
	t.Cleanup(func() {
		database.DB = prev
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
}

// moneyflyReleaseFixture v2.2.18 的真实资产清单（含校验和文件）
func moneyflyReleaseFixture() *ghrelease.Release {
	names := []string{
		"MoneyFly-android-2.2.18.aab",
		"MoneyFly-android-arm64-v8a-2.2.18.apk",
		"MoneyFly-macos-arm64-2.2.18.dmg",
		"MoneyFly-macos-universal-2.2.18.dmg",
		"MoneyFly-macos-x64-2.2.18.dmg",
		"MoneyFly-setup-2.2.18.exe",
		"SHA256SUMS-android-apk.txt",
		"SHA256SUMS-macos-arm64.txt",
		"SHA256SUMS-macos-x64.txt",
		"SHA256SUMS-windows.txt",
	}
	rel := &ghrelease.Release{TagName: "v2.2.18", Name: "v2.2.18"}
	for _, n := range names {
		rel.Assets = append(rel.Assets, ghrelease.Asset{
			Name:        n,
			Size:        1234,
			DownloadURL: "https://github.com/moneyfly004/moneyfly/releases/download/v2.2.18/" + n,
		})
	}
	return rel
}

// stubRelease 替换联网取数函数（测试内不联网），返回恢复函数
func stubRelease(t *testing.T, rel *ghrelease.Release, err error, sumsText string) {
	t.Helper()
	setupSoftwareLatestDB(t)
	prevRel, prevText := latestReleaseFor, releaseTextFetch
	latestReleaseFor = func(string, []string, string) (*ghrelease.Release, error) {
		return rel, err
	}
	releaseTextFetch = func(string) string { return sumsText }
	t.Cleanup(func() {
		latestReleaseFor, releaseTextFetch = prevRel, prevText
		sumsCache.Range(func(k, _ any) bool { sumsCache.Delete(k); return true })
	})
}

func newSoftwareLatestContext(key string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/software/latest?key="+key, nil)
	return c, w
}

func decodeData(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应不是 JSON: %v body=%s", err, w.Body.String())
	}
	return body.Data
}

// TestSoftwareLatestUnknownKey 未知配置键必须 404（不能拿任意 key 去打 GitHub）
func TestSoftwareLatestUnknownKey(t *testing.T) {
	stubRelease(t, moneyflyReleaseFixture(), nil, "")
	for _, key := range []string{"", "not_a_real_key", "moneyfly_version"} {
		c, w := newSoftwareLatestContext(key)
		SoftwareLatest(c)
		if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
			t.Errorf("key=%q 应拒绝（400/404），实际 %d", key, w.Code)
		}
	}
}

// TestSoftwareLatestReturnsAssetMetadata 正常路径：给出该平台正确的包名/体积/sha256
func TestSoftwareLatestReturnsAssetMetadata(t *testing.T) {
	sums := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  MoneyFly-setup-2.2.18.exe\n"
	stubRelease(t, moneyflyReleaseFixture(), nil, sums)

	c, w := newSoftwareLatestContext("moneyfly_windows_url")
	SoftwareLatest(c)
	if w.Code != http.StatusOK {
		t.Fatalf("状态码 %d body=%s", w.Code, w.Body.String())
	}
	data := decodeData(t, w)
	if data["version"] != "2.2.18" {
		t.Errorf("version = %v", data["version"])
	}
	if data["asset_name"] != "MoneyFly-setup-2.2.18.exe" {
		t.Errorf("asset_name = %v（应为 Windows 安装包，不能是 zip/dmg/apk）", data["asset_name"])
	}
	if data["sha256"] != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Errorf("sha256 = %v", data["sha256"])
	}
	if data["download_url"] == "" || data["download_url"] == nil {
		t.Error("download_url 不应为空")
	}
}

// TestSoftwareLatestPerPlatformAsset 每个入口都必须给出该平台自己的包
func TestSoftwareLatestPerPlatformAsset(t *testing.T) {
	stubRelease(t, moneyflyReleaseFixture(), nil, "")
	cases := map[string]string{
		"moneyfly_windows_url":   "MoneyFly-setup-2.2.18.exe",
		"moneyfly_macos_arm_url": "MoneyFly-macos-arm64-2.2.18.dmg",
		"moneyfly_macos_url":     "MoneyFly-macos-x64-2.2.18.dmg",
		"moneyfly_android_url":   "MoneyFly-android-arm64-v8a-2.2.18.apk",
	}
	for key, want := range cases {
		c, w := newSoftwareLatestContext(key)
		SoftwareLatest(c)
		if w.Code != http.StatusOK {
			t.Fatalf("%s 状态码 %d", key, w.Code)
		}
		if got := decodeData(t, w)["asset_name"]; got != want {
			t.Errorf("%s → %v，期望 %s", key, got, want)
		}
	}
}

// TestSoftwareLatestWithoutSums 拿不到校验和时返回空串（客户端按「无校验值则跳过」处理），不能报错
func TestSoftwareLatestWithoutSums(t *testing.T) {
	rel := moneyflyReleaseFixture()
	rel.Assets = rel.Assets[:6] // 去掉全部 SHA256SUMS-*.txt
	stubRelease(t, rel, nil, "")

	c, w := newSoftwareLatestContext("moneyfly_windows_url")
	SoftwareLatest(c)
	if w.Code != http.StatusOK {
		t.Fatalf("缺校验和不应失败，状态码 %d body=%s", w.Code, w.Body.String())
	}
	if sha := decodeData(t, w)["sha256"]; sha != "" {
		t.Errorf("sha256 应为空串，实际 %v", sha)
	}
}

// TestSoftwareLatestGitHubError 取 GitHub 失败 → 502（客户端据此回退，不误报「已是最新」）
func TestSoftwareLatestGitHubError(t *testing.T) {
	stubRelease(t, nil, http.ErrHandlerTimeout, "")
	c, w := newSoftwareLatestContext("moneyfly_windows_url")
	SoftwareLatest(c)
	if w.Code != http.StatusBadGateway {
		t.Fatalf("应返回 502，实际 %d body=%s", w.Code, w.Body.String())
	}
}

// TestParseSHA256FromSums 兼容 sha256sum 的文本/二进制两种行格式与 CRLF
func TestParseSHA256FromSums(t *testing.T) {
	const h = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	cases := []struct {
		name, text, want string
	}{
		{"文本模式", h + "  MoneyFly-setup-2.2.18.exe\n", h},
		{"二进制模式（Windows）", h + " *MoneyFly-setup-2.2.18.exe\r\n", h},
		{"多行里挑对的那行", "deadbeef  other.exe\n" + h + "  MoneyFly-setup-2.2.18.exe\n", h},
		{"文件名不匹配", h + "  MoneyFly-setup-2.2.17.exe\n", ""},
		{"摘要长度不对", "abc123  MoneyFly-setup-2.2.18.exe\n", ""},
		{"非十六进制", "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz  MoneyFly-setup-2.2.18.exe\n", ""},
		{"空文本", "", ""},
	}
	for _, tc := range cases {
		if got := parseSHA256FromSums(tc.text, "MoneyFly-setup-2.2.18.exe"); got != tc.want {
			t.Errorf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}

// TestFindSumsAsset 优先平台专属校验和文件，缺失时退回合并版；nil 安全
func TestFindSumsAsset(t *testing.T) {
	rel := moneyflyReleaseFixture()
	target := findMoneyflyTarget(t, "moneyfly_windows_url")
	got := findSumsAsset(rel, target)
	if got == nil || got.Name != "SHA256SUMS-windows.txt" {
		t.Fatalf("应选平台专属校验和文件，实际 %v", got)
	}
	// 只留合并版 → 退回它
	merged := "SHA256SUMS.txt"
	rel2 := &ghrelease.Release{TagName: "v1", Assets: []ghrelease.Asset{{Name: merged}}}
	if got := findSumsAsset(rel2, target); got == nil || got.Name != merged {
		t.Errorf("应退回合并版 %s，实际 %v", merged, got)
	}
	if got := findSumsAsset(nil, target); got != nil {
		t.Error("release 为 nil 时应返回 nil")
	}
	relWithMerged := &ghrelease.Release{TagName: "v1", Assets: []ghrelease.Asset{
		{Name: "MoneyFly-setup-2.2.18.exe"}, {Name: merged},
	}}
	if got := findSumsAsset(relWithMerged, nil); got == nil || got.Name != merged {
		t.Errorf("target 为 nil 时应退回合并版，实际 %v", got)
	}
}

// findMoneyflyTarget 供测试取目录里的目标定义
func findMoneyflyTarget(t *testing.T, key string) *software_sync.Target {
	t.Helper()
	target := software_sync.FindTarget(key)
	if target == nil {
		t.Fatalf("同步目录缺少 %s", key)
	}
	return target
}
