package handlers

import (
	"testing"

	"cboard-go/internal/models"
)

func strPtr(s string) *string { return &s }

// 抽屉「设备」列的取值优先级（用户反馈：整列都是「未知设备」）
//
// 生产库里 device_model 只有 iOS/Android 的 UA 才解析得出，桌面客户端、
// 系统 UA 全为空。信息其实都在 os_name / software_name 里，只是以前没被用上。
// 本测试锁住「机型 > 品牌 > 系统+版本 > 客户端+版本 > 未知设备」这条兜底链。
func TestDeviceLabelOfFallbackChain(t *testing.T) {
	cases := []struct {
		name      string
		dev       models.Device
		wantLabel string
		wantSub   string
	}{
		{
			name: "iOS 有具体机型 → 主标题机型，副标题系统",
			dev: models.Device{
				DeviceModel: strPtr("iPhone 15 Pro"),
				OSName:      strPtr("iOS"), OSVersion: strPtr("18.6"),
				SoftwareName: strPtr("Shadowrocket"), SoftwareVersion: strPtr("3860.700.2"),
			},
			wantLabel: "iPhone 15 Pro",
			wantSub:   "iOS 18.6",
		},
		{
			name: "只有品牌 → 主标题品牌",
			dev: models.Device{
				DeviceBrand: strPtr("HUAWEI"),
				OSName:      strPtr("Android"), OSVersion: strPtr("14"),
			},
			wantLabel: "HUAWEI",
			wantSub:   "Android 14",
		},
		{
			name: "桌面客户端（无机型）→ 主标题系统，副标题客户端",
			dev: models.Device{
				OSName: strPtr("Windows"), OSVersion: strPtr("11"),
				SoftwareName: strPtr("Clash Verge"), SoftwareVersion: strPtr("2.5.2"),
			},
			wantLabel: "Windows 11",
			wantSub:   "Clash Verge 2.5.2",
		},
		{
			name: "UA 无系统段（跨平台客户端）→ 主标题 Desktop，副标题客户端",
			dev: models.Device{
				OSName:       strPtr("Desktop（Win/macOS）"),
				SoftwareName: strPtr("Mihomo Party"), SoftwareVersion: strPtr("2.0.0"),
			},
			wantLabel: "Desktop（Win/macOS）",
			wantSub:   "Mihomo Party 2.0.0",
		},
		{
			name: "系统未知只有客户端 → 用客户端兜底（不再是「未知设备」）",
			dev: models.Device{
				OSName:       strPtr("Unknown"),
				SoftwareName: strPtr("V2Box"), SoftwareVersion: strPtr("10.1.5"),
			},
			wantLabel: "V2Box 10.1.5",
			wantSub:   "",
		},
		{
			name: "自动化请求 → 明确标出非设备来源",
			dev: models.Device{
				OSName:       strPtr("自动化/非客户端"),
				SoftwareName: strPtr("curl"), SoftwareVersion: strPtr("8.9.1"),
			},
			wantLabel: "自动化/非客户端",
			wantSub:   "curl 8.9.1",
		},
		{
			name:      "真的一无所有 → 未知设备",
			dev:       models.Device{OSName: strPtr("Unknown")},
			wantLabel: "未知设备",
			wantSub:   "",
		},
		{
			name:      "空指针不应 panic",
			dev:       models.Device{},
			wantLabel: "未知设备",
			wantSub:   "",
		},
	}

	for _, c := range cases {
		label, sub := deviceLabelOf(&c.dev)
		if label != c.wantLabel {
			t.Errorf("%s: 主标题应为 %q，实际 %q", c.name, c.wantLabel, label)
		}
		if sub != c.wantSub {
			t.Errorf("%s: 副标题应为 %q，实际 %q", c.name, c.wantSub, sub)
		}
	}
}
