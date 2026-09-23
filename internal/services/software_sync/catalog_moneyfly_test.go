package software_sync

import (
	"strings"
	"testing"

	"cboard-go/internal/services/ghrelease"
)

// moneyflyV2218Assets 是 moneyfly004/moneyfly v2.2.18 Release 的真实资产清单
// （按 GitHub API 返回顺序：名称字典序）。用于锁定资产匹配规则，
// 防止 macos 三件套（arm64 / x64 / universal）与 android 的 apk / aab 互相误配。
var moneyflyV2218Assets = []string{
	"MoneyFly-android-2.2.18.aab",
	"MoneyFly-android-arm64-v8a-2.2.18.apk",
	"MoneyFly-android-armeabi-v7a-2.2.18.apk",
	"MoneyFly-android-x86_64-2.2.18.apk",
	"MoneyFly-ios-2.2.18.ipa",
	"MoneyFly-macos-arm64-2.2.18.dmg",
	"MoneyFly-macos-universal-2.2.18.dmg",
	"MoneyFly-macos-x64-2.2.18.dmg",
	"MoneyFly-setup-2.2.18.exe",
	"MoneyFly-windows-x64-portable-2.2.18.zip",
}

func moneyflyRelease(tag string) *ghrelease.Release {
	rel := &ghrelease.Release{TagName: tag, Name: tag}
	for _, name := range moneyflyV2218Assets {
		rel.Assets = append(rel.Assets, ghrelease.Asset{Name: name})
	}
	return rel
}

// TestMoneyFlyCatalogRegistered MoneyFly 必须注册进同步目录，
// 否则 pan:// 动态直链无法解析（历史上该软件只能手填网盘直链，每次发版都要人工替换）。
func TestMoneyFlyCatalogRegistered(t *testing.T) {
	sw := FindSoftwareByConfigKey("moneyfly_windows_url")
	if sw == nil {
		t.Fatal("MoneyFly 未注册进同步目录：moneyfly_windows_url 找不到所属软件")
	}
	if sw.Key != "moneyfly" || sw.Repo != "moneyfly004/moneyfly" {
		t.Fatalf("MoneyFly 仓库信息错误: key=%s repo=%s", sw.Key, sw.Repo)
	}
	for _, key := range []string{
		"moneyfly_windows_url",
		"moneyfly_macos_arm_url",
		"moneyfly_macos_url",
		"moneyfly_android_url",
	} {
		if FindTarget(key) == nil {
			t.Errorf("同步目录缺少目标: %s", key)
		}
	}
}

// TestMoneyFlyAssetMatching 按真实资产清单校验各入口选中的安装包。
func TestMoneyFlyAssetMatching(t *testing.T) {
	rel := moneyflyRelease("v2.2.18")
	if got := rel.Version(); got != "2.2.18" {
		t.Fatalf("版本号解析错误: %s", got)
	}

	cases := []struct {
		configKey string
		want      string
	}{
		{"moneyfly_windows_url", "MoneyFly-setup-2.2.18.exe"},
		{"moneyfly_macos_arm_url", "MoneyFly-macos-arm64-2.2.18.dmg"},
		{"moneyfly_macos_url", "MoneyFly-macos-x64-2.2.18.dmg"},
		{"moneyfly_android_url", "MoneyFly-android-arm64-v8a-2.2.18.apk"},
	}
	for _, c := range cases {
		target := FindTarget(c.configKey)
		if target == nil {
			t.Fatalf("未找到目标 %s", c.configKey)
		}
		asset, err := FindAssetFor(rel, target)
		if err != nil {
			t.Errorf("%s 匹配失败: %v", c.configKey, err)
			continue
		}
		if asset.Name != c.want {
			t.Errorf("%s 匹配到 %s，期望 %s", c.configKey, asset.Name, c.want)
		}
	}
}

// TestMoneyFlyWindowsSkipsPortableZip Windows 入口必须是安装包 exe，不能是免安装 zip。
func TestMoneyFlyWindowsSkipsPortableZip(t *testing.T) {
	rel := moneyflyRelease("v2.2.18")
	asset, err := FindAssetFor(rel, FindTarget("moneyfly_windows_url"))
	if err != nil {
		t.Fatal(err)
	}
	if asset.Name == "MoneyFly-windows-x64-portable-2.2.18.zip" {
		t.Fatal("Windows 入口误配为免安装 zip，应为 setup exe")
	}
}

// TestMoneyFlyAndroidSkipsAAB Android 入口必须给可直装的 apk，不能是上架用 aab。
func TestMoneyFlyAndroidSkipsAAB(t *testing.T) {
	rel := moneyflyRelease("v2.2.18")
	asset, err := FindAssetFor(rel, FindTarget("moneyfly_android_url"))
	if err != nil {
		t.Fatal(err)
	}
	if asset.Name == "MoneyFly-android-2.2.18.aab" {
		t.Fatal("Android 入口误配为上架包 aab，用户无法直装")
	}
}

// TestMoneyFlyMacOSUniversalNotCrossMatched 通用 dmg 不得被 arm64 / x64 入口选中，
// 否则 Apple 芯片用户会拿到体积更大的通用包（或反之 Intel 用户拿到 arm 包）。
func TestMoneyFlyMacOSUniversalNotCrossMatched(t *testing.T) {
	rel := moneyflyRelease("v2.2.18")
	for _, c := range []struct{ key, want string }{
		{"moneyfly_macos_arm_url", "arm64"},
		{"moneyfly_macos_url", "x64"},
	} {
		asset, err := FindAssetFor(rel, FindTarget(c.key))
		if err != nil {
			t.Fatalf("%s 匹配失败: %v", c.key, err)
		}
		if asset.Name == "MoneyFly-macos-universal-2.2.18.dmg" {
			t.Errorf("%s 误选通用包", c.key)
		}
		if !strings.Contains(asset.Name, c.want) {
			t.Errorf("%s 匹配到 %s，应含 %s", c.key, asset.Name, c.want)
		}
	}
}

// TestMoneyFlyFutureVersionStillMatches 后续版本（版本号变化）仍应命中同一规则，
// 保证发版后无需再改任何配置。
func TestMoneyFlyFutureVersionStillMatches(t *testing.T) {
	rel := &ghrelease.Release{TagName: "v2.3.0", Assets: []ghrelease.Asset{
		{Name: "MoneyFly-setup-2.3.0.exe"},
		{Name: "MoneyFly-macos-arm64-2.3.0.dmg"},
		{Name: "MoneyFly-macos-x64-2.3.0.dmg"},
		{Name: "MoneyFly-android-arm64-v8a-2.3.0.apk"},
	}}
	for _, c := range []struct{ key, want string }{
		{"moneyfly_windows_url", "MoneyFly-setup-2.3.0.exe"},
		{"moneyfly_macos_arm_url", "MoneyFly-macos-arm64-2.3.0.dmg"},
		{"moneyfly_macos_url", "MoneyFly-macos-x64-2.3.0.dmg"},
		{"moneyfly_android_url", "MoneyFly-android-arm64-v8a-2.3.0.apk"},
	} {
		asset, err := FindAssetFor(rel, FindTarget(c.key))
		if err != nil {
			t.Errorf("%s 匹配失败: %v", c.key, err)
			continue
		}
		if asset.Name != c.want {
			t.Errorf("%s 匹配到 %s，期望 %s", c.key, asset.Name, c.want)
		}
	}
}
