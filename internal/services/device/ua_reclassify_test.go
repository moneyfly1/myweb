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
