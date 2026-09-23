// Package software_sync 定义从 GitHub Release 同步安装包到 123 云盘的软件目录与同步引擎。
package software_sync

import "regexp"

// Target 一个下载入口（对应一个软件下载配置键）
type Target struct {
	// ConfigKey 软件下载配置键，如 clash_verge_windows_url
	ConfigKey string
	// OS 平台标识：windows / macos / android
	OS string
	// Arch 架构标识：x64 / intel / apple / universal
	Arch string
	// Label 展示名称，如 "Windows x64"
	Label string
	// Preferred 优先匹配规则（先尝试，如 arm64-v8a apk）
	Preferred []*regexp.Regexp
	// Patterns 匹配规则（对齐各仓库实际资产命名，已逐一核对 2026-08）
	Patterns []*regexp.Regexp
}

// Software 一个软件
type Software struct {
	// Key 标识，如 clash-verge
	Key string
	// Name 展示名称
	Name string
	// Repo GitHub 仓库，如 clash-verge-rev/clash-verge-rev
	Repo string
	// Targets 各平台下载目标
	Targets []Target
}

func rx(patterns ...string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		out = append(out, regexp.MustCompile(p))
	}
	return out
}

// 通用匹配（已对照各仓库真实资产名）
var apkAny = rx(`(?i)\.apk$`)
var apkPreferredArm = rx(`(?i)(arm64|arm64[-_]?v8a)[^.]*\.apk$`)
var dmgIntel = rx(`(?i)^.*(intel|x64|amd64|_x64|-64)\.(dmg|pkg)$`)
var dmgApple = rx(`(?i)^.*(apple|silicon|m[0-9]+|arm64|aarch64|_aarch64).*\.(dmg|pkg)$`)

// Catalog 同步目录：8 款软件 × 平台/架构（2026-08 已逐一核对 GitHub 最新 Release 资产名）
var Catalog = []Software{
	{
		// MoneyFly 为本站自研官方客户端，资产命名固定为 MoneyFly-<平台>-<版本>.<扩展名>
		// （见仓库 release.yml 的产物名），因此用精确前缀匹配，避免 macos 三件套互相误配：
		//   MoneyFly-macos-arm64-<V>.dmg       → Apple 芯片
		//   MoneyFly-macos-x64-<V>.dmg         → Intel
		//   MoneyFly-macos-universal-<V>.dmg   → 通用包（不映射到任何入口，避免抢占 arm/intel）
		// 注意：通用 dmgIntel/dmgApple 规则匹配不了 "macos-x64-2.2.18.dmg"（x64 后面还有版本号），
		// 故此处不复用，改用专属规则。
		Key: "moneyfly", Name: "MoneyFly", Repo: "moneyfly004/moneyfly",
		Targets: []Target{
			{ConfigKey: "moneyfly_windows_url", OS: "windows", Arch: "x64", Label: "Windows x64", Patterns: rx(`(?i)^MoneyFly-setup-.*\.exe$`)},
			{ConfigKey: "moneyfly_macos_arm_url", OS: "macos", Arch: "apple", Label: "macOS Apple 芯片", Patterns: rx(`(?i)^MoneyFly-macos-arm64-.*\.dmg$`)},
			{ConfigKey: "moneyfly_macos_url", OS: "macos", Arch: "intel", Label: "macOS Intel", Patterns: rx(`(?i)^MoneyFly-macos-x64-.*\.dmg$`)},
			// Android 只发 apk（同 Release 另有 .aab 上架包，被 \.apk$ 排除）
			{ConfigKey: "moneyfly_android_url", OS: "android", Arch: "universal", Label: "Android APK", Preferred: rx(`(?i)^MoneyFly-android-arm64-v8a-.*\.apk$`), Patterns: rx(`(?i)^MoneyFly-android-.*\.apk$`)},
		},
	},
	{
		Key: "v2rayn", Name: "V2rayN", Repo: "2dust/v2rayN",
		Targets: []Target{
			// windows-64 / windows-x64 专属，避免误配 windows-arm64
			{ConfigKey: "v2rayn_url", OS: "windows", Arch: "x64", Label: "Windows x64", Patterns: rx(`(?i)^.*windows-(64|x64)([-.]|$).*\.zip$`)},
			{ConfigKey: "v2rayn_macos_url", OS: "macos", Arch: "intel", Label: "macOS Intel", Patterns: rx(`(?i)^.*macos-64\.dmg$`)},
			{ConfigKey: "v2rayn_macos_arm_url", OS: "macos", Arch: "apple", Label: "macOS Apple 芯片", Patterns: dmgApple},
		},
	},
	{
		Key: "hiddify", Name: "Hiddify", Repo: "hiddify/hiddify-app",
		Targets: []Target{
			{ConfigKey: "hiddify_windows_url", OS: "windows", Arch: "x64", Label: "Windows x64", Patterns: rx(`(?i)^.*x64.*\.exe$`)},
			{ConfigKey: "hiddify_android_url", OS: "android", Arch: "universal", Label: "Android APK", Preferred: apkPreferredArm, Patterns: apkAny},
			// Hiddify 官方 macOS 为通用包 Hiddify-MacOS.dmg（Intel 与 Apple 芯片同一文件）
			{ConfigKey: "hiddify_macos_url", OS: "macos", Arch: "intel", Label: "macOS Intel", Patterns: rx(`(?i)^.*macos.*\.dmg$`)},
			{ConfigKey: "hiddify_macos_arm_url", OS: "macos", Arch: "apple", Label: "macOS Apple 芯片", Patterns: rx(`(?i)^.*macos.*\.dmg$`)},
		},
	},
	{
		Key: "clash-verge", Name: "Clash Verge", Repo: "clash-verge-rev/clash-verge-rev",
		Targets: []Target{
			{ConfigKey: "clash_verge_windows_url", OS: "windows", Arch: "x64", Label: "Windows x64", Patterns: rx(`(?i)^.*x64.*\.(exe|msi)$`)},
			{ConfigKey: "clash_verge_macos_url", OS: "macos", Arch: "intel", Label: "macOS Intel", Patterns: dmgIntel},
			{ConfigKey: "clash_verge_macos_arm_url", OS: "macos", Arch: "apple", Label: "macOS Apple 芯片", Patterns: dmgApple},
		},
	},
	{
		Key: "clash-part", Name: "Clash Part", Repo: "mihomo-party-org/clash-party",
		Targets: []Target{
			// 优先现代 windows-* x64（跳过 win7-* 旧版，且 arm64 中的 "64" 不会被误配）
			{ConfigKey: "clash_party_windows_url", OS: "windows", Arch: "x64", Label: "Windows x64", Patterns: rx(`(?i)^.*windows.*(x64|[^a-z]64).*\.exe$`)},
			{ConfigKey: "clash_party_macos_url", OS: "macos", Arch: "intel", Label: "macOS Intel", Patterns: dmgIntel},
			{ConfigKey: "clash_party_macos_arm_url", OS: "macos", Arch: "apple", Label: "macOS Apple 芯片", Patterns: dmgApple},
		},
	},
	{
		Key: "flclash", Name: "FlClash", Repo: "chen08209/FlClash",
		Targets: []Target{
			{ConfigKey: "flash_windows_url", OS: "windows", Arch: "x64", Label: "Windows x64", Patterns: rx(`(?i)^.*(x64|amd64).*\.exe$`)},
			{ConfigKey: "flash_android_url", OS: "android", Arch: "universal", Label: "Android APK", Preferred: apkPreferredArm, Patterns: apkAny},
			{ConfigKey: "flash_macos_url", OS: "macos", Arch: "intel", Label: "macOS Intel", Patterns: dmgIntel},
			{ConfigKey: "flash_macos_arm_url", OS: "macos", Arch: "apple", Label: "macOS Apple 芯片", Patterns: dmgApple},
		},
	},
	{
		Key: "v2rayng", Name: "V2rayNG", Repo: "2dust/v2rayNG",
		Targets: []Target{
			{ConfigKey: "v2rayng_url", OS: "android", Arch: "universal", Label: "Android APK", Preferred: apkPreferredArm, Patterns: apkAny},
		},
	},
	{
		Key: "clash-meta", Name: "Clash Meta", Repo: "MetaCubeX/ClashMetaForAndroid",
		Targets: []Target{
			{ConfigKey: "clash_android_url", OS: "android", Arch: "universal", Label: "Android APK", Preferred: apkPreferredArm, Patterns: apkAny},
		},
	},
}

// FindSoftwareByConfigKey 按配置键找所属软件（含 Repo）
func FindSoftwareByConfigKey(configKey string) *Software {
	for i := range Catalog {
		for j := range Catalog[i].Targets {
			if Catalog[i].Targets[j].ConfigKey == configKey {
				return &Catalog[i]
			}
		}
	}
	return nil
}

// FindTarget 按配置键找目标
func FindTarget(configKey string) *Target {
	for i := range Catalog {
		for j := range Catalog[i].Targets {
			if Catalog[i].Targets[j].ConfigKey == configKey {
				return &Catalog[i].Targets[j]
			}
		}
	}
	return nil
}
