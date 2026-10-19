package device

import "testing"

// 「未知设备」重新分类（2026-10-19 用户反馈）
//
// 生产库抽样发现：大量"未知"并不是识别失败，而是 UA **本身不含 OS 段**
// （mihomo.party / clash.meta / sing-box 这类跨平台客户端与裸内核），
// 以及 curl / Go-http-client 这类**不是用户设备**的自动化请求。
// 本测试锁住增强后的归类口径。
func TestParseUserAgentReclassifiesUnknown(t *testing.T) {
	dm := NewDeviceManager()

	cases := []struct {
		ua       string
		software string
		osName   string // 期望的 OS（含"自动化/非客户端"这类明确归类）
	}{
		// 跨平台桌面客户端：UA 无 OS 段 → 不能落成 Clash / Unknown
		{"mihomo.party/v2.0.0 (clash.meta)", "Mihomo Party", ""},
		{"mihomo.party/v1.9.5 (clash.meta)", "Mihomo Party", ""},
		// FlClash 的 UA 带 clash-verge 兼容串 → 不能被误判成 Clash Verge
		{"FlClash/v0.9.1 clash-verge Platform/macos", "FlClash", ""},
		// 裸内核类
		{"clash.meta/alpha-de19f92", "Clash Meta", ""},
		{"sing-box/1.13.2", "sing-box", ""},
		// Android 客户端仍要识别成 Android
		{"ClashMetaForAndroid/2.11.24.Meta", "Clash Meta", "Android"},
		// 自动化/抓取类：明确归类，不再混进"未知设备"
		{"curl/8.9.1", "curl", "自动化/非客户端"},
		{"Go-http-client/2.0", "Go-http-client", "自动化/非客户端"},
		{"python-requests/2.31.0", "python-requests", "自动化/非客户端"},
		{"okhttp/4.12.0", "OkHttp", "自动化/非客户端"},
		// 正常客户端不受影响
		{"MoneyFly/2.2.30 (Windows NT 10.0; Win64; x64)", "MoneyFly", "Windows"},
		{"Shadowrocket/3860.700.2 CFNetwork/3860.700.1 Darwin/24.6.0", "Shadowrocket", "iOS"},
		{"v2rayN/7.14.12", "v2rayN", "Windows"},
	}

	for _, c := range cases {
		info := dm.ParseUserAgent(c.ua)
		if info.SoftwareName != c.software {
			t.Errorf("UA=%q 软件名应为 %q，实际 %q", c.ua, c.software, info.SoftwareName)
		}
		if c.osName != "" && info.OSName != c.osName {
			t.Errorf("UA=%q 系统应为 %q，实际 %q", c.ua, c.osName, info.OSName)
		}
		// 软件名不允许再是 Unknown（这正是"未知设备"的来源）
		if info.SoftwareName == "Unknown" {
			t.Errorf("UA=%q 不应再解析成 Unknown", c.ua)
		}
	}
}

// 自动化客户端必须与「用户设备」分开，便于统计时排除
func TestAutomationUAClassification(t *testing.T) {
	dm := NewDeviceManager()
	for _, ua := range []string{"curl/7.64.0", "Go-http-client/2.0", "python-requests/2.31.0", "PostmanRuntime/7.36.0"} {
		info := dm.ParseUserAgent(ua)
		if info.OSName != "自动化/非客户端" {
			t.Errorf("UA=%q 应归类为自动化/非客户端，实际 %q", ua, info.OSName)
		}
	}
}

// 第二轮再分类：把生产库里剩下的「未知」一条条对回真实来源。
//
// 抽样（老站活跃设备）后剩下的 Unknown 只有三类：
//  1. UA 本身就是我们的订阅链接（脚本/中间层转发）；
//  2. 社交/IM 的链接预览爬虫（钉钉、WhatsApp、微信、TelegramBot…）；
//  3. 客户端写了但规则没覆盖（FlC=FlClash、CMFAndroid、ClashX、V2Box、SagerNet…）。
// 前两类不是设备，第三类必须识别出来 —— 本测试逐条锁住。
func TestParseUserAgentReclassifiesUnknownRound2(t *testing.T) {
	dm := NewDeviceManager()

	cases := []struct {
		ua       string
		software string
		osName   string // 空表示"不校验（跨平台/无 OS 段）"
	}{
		// ① UA 就是订阅地址
		{"https://dy.moneyfly.top/api/v1/subscriptions/universal/oSEtBtfvh4xELKDZDeqkow==", "直接抓取订阅", "自动化/非客户端"},
		// ② 链接预览爬虫：不是设备
		{"DingTalkBot-LinkService/1.0 (+https://open-doc.dingtalk.com/microapp/faquestions/ftpfeu)", "钉钉链接预览", "自动化/非客户端"},
		{"WhatsApp/2.23.20.0", "WhatsApp 链接预览", "自动化/非客户端"},
		{"WhatsApp/3.0.0.0 A", "WhatsApp 链接预览", "自动化/非客户端"},
		{"Mobile/15E148 wxwork/5.0.6 MicroMessenger/7.0.1 Language/zh ColorScheme/Light wwmver/3.26.120.400", "企业微信链接预览", "自动化/非客户端"},
		{"GoogleMessages/20.2 facebookexternalhit/1.1 Facebot Twitterbot/1.0", "Google 链接预览", "自动化/非客户端"},
		{"TelegramBot (like TwitterBot)", "Telegram 链接预览", "自动化/非客户端"},
		// ③ 规则未覆盖的真实客户端
		{"FlC/0.8.61/meta/verge/Android 10; HJC-AN90", "FlClash", "Android"},
		{"CMFAndroid/2.11.1x", "Clash Meta", "Android"},
		{"HiddifyNext/1.1.1 (macos) like ClashMeta v2ray sing-box", "Hiddify", "macOS"},
		{"HiddifyNext/4.0.0 (ios) like ClashMeta v2ray sing-box", "Hiddify", "iOS"},
		{"V2rayU/4.2.2 CFNetwork/1410.4 Darwin/22.6.0", "V2rayU", "macOS"},
		{"ClashX Meta/v1.0 (com.metacubex.ClashX.meta; build:0; macOS 12.6.9) Alamofire/5.10.2", "ClashX Meta", "macOS"},
		{"ClashX/1.119.0 (com.west2online.ClashX; build:1.119.0; macOS 26.7.1) Alamofire/5.10.2", "ClashX", "macOS"},
		{"V2Box 10.1.5/iOS 26.5.2", "V2Box", "iOS"},
		{"SagerNet/0.4-rc06", "SagerNet", "Android"},
		{"Matsuri/0.4 (Prefer Clash Format)", "Matsuri", "Android"},
		{"AxXray/0.2-rc02", "ANXRay", "iOS"},
		{"ToMoonClash/0.2.8", "ToMoon Clash", ""},
	}

	for _, c := range cases {
		info := dm.ParseUserAgent(c.ua)
		if info.SoftwareName != c.software {
			t.Errorf("UA=%q 软件名应为 %q，实际 %q", c.ua, c.software, info.SoftwareName)
		}
		if c.osName != "" && info.OSName != c.osName {
			t.Errorf("UA=%q 系统应为 %q，实际 %q", c.ua, c.osName, info.OSName)
		}
	}

	// V2Box 的 iOS 版本号要能从 "/iOS 26.5.2" 里取到
	if info := dm.ParseUserAgent("V2Box 10.1.5/iOS 26.5.2"); info.OSVersion != "26.5.2" {
		t.Errorf("V2Box iOS 版本应为 26.5.2，实际 %q", info.OSVersion)
	}
}

// 没有客户端标识、但 UA 带系统信息的请求（Android 系统 UA / Dalvik）：
// 系统要能识别出来，软件名保持 Unknown 是诚实的（不能瞎猜客户端）。
func TestSystemOnlyUserAgentKeepsHonestUnknown(t *testing.T) {
	dm := NewDeviceManager()
	ua := "Dalvik/2.1.0 (Linux; U; Android 16; PKB110 Build/BP2A.250605.015)"
	info := dm.ParseUserAgent(ua)
	if info.OSName != "Android" {
		t.Errorf("Dalvik UA 系统应为 Android，实际 %q", info.OSName)
	}
	if info.SoftwareName != "Unknown" {
		t.Errorf("Dalvik UA 没有客户端标识，软件名应保持 Unknown，实际 %q", info.SoftwareName)
	}
}
