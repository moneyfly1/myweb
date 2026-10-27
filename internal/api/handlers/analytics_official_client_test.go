package handlers

import "testing"

// 「官方 / 第三方」口径：MoneyFly 与它的后续版本 MBox 必须**并存**为官方。
//
// 背景：品牌从 MoneyFly 改名 MBox 后，后台统计只把 software_name=MoneyFly
// 算官方 → MBox 用户被错算进「第三方客户端（待升级）」，劝升级通知会发给
// 本来就用着官方客户端的人。
func TestOfficialClientNamesIncludeMoneyFlyAndMBox(t *testing.T) {
	cases := map[string]bool{
		"MoneyFly":    true,
		"MBox":        true,
		"  mbox  ":    true, // 大小写/空白不敏感
		"moneyfly":    true,
		"Clash Verge": false,
		"v2rayN":      false,
		"Unknown":     false,
		"":            false,
	}
	for name, want := range cases {
		if got := isOfficialClientName(name); got != want {
			t.Errorf("isOfficialClientName(%q) = %v, 期望 %v", name, got, want)
		}
	}

	// 传给 SQL 的小写名单必须两套都有（official 用 IN、third_party 用 NOT IN）
	lower := lowerOfficialClientNames()
	seen := map[string]bool{}
	for _, n := range lower {
		seen[n] = true
	}
	for _, want := range []string{"moneyfly", "mbox"} {
		if !seen[want] {
			t.Fatalf("官方客户端名单缺少 %q（实际 %v）", want, lower)
		}
	}
}
