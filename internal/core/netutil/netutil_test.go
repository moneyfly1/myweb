package netutil

import (
	"net"
	"testing"
)

// IP 判定与规范化的统一口径测试。
//
// 背景：历史上内网判定有 8 份互不等价的实现，导致
//   - 192.168.x.x 被当成公网地址外发到第三方查询接口（内网拓扑泄漏）
//   - "内网"/"本地" 占位值被当成国家名进入地区统计
//   - 同一 IP 在不同模块结论不同
// 这里锁定并集口径，防止再次分叉。

func TestIsPrivateOrReserved(t *testing.T) {
	private := []string{
		"127.0.0.1", "127.1.2.3", "10.0.0.1", "10.255.255.255",
		"172.16.0.1", "172.31.255.254", // 172.16/12 边界
		"192.168.0.1", "192.168.1.5",
		"169.254.1.1",   // 链路本地
		"100.64.0.1",    // CGNAT
		"100.127.255.1", // CGNAT 上界
		"0.0.0.0", "224.0.0.1", "239.255.255.255", "240.0.0.1",
		"::1", "fe80::1", "fc00::1", "fd12:3456::1",
		"::ffff:192.168.1.1", // IPv4-mapped 内网
	}
	for _, s := range private {
		if !IsPrivateOrReserved(net.ParseIP(s)) {
			t.Errorf("%s 应判定为内网/保留地址", s)
		}
		if !IsPrivateOrReservedString(s) {
			t.Errorf("%s 字符串版也应判定为内网/保留", s)
		}
	}

	public := []string{
		"8.8.8.8", "1.1.1.1", "203.0.114.1",
		"172.15.0.1", // 172.16/12 之外，是公网（历史前端曾误判为内网）
		"172.32.0.1", // 同上
		"100.63.0.1", // CGNAT 之外
		"100.128.0.1",
		"2606:4700:4700::1111",
	}
	for _, s := range public {
		if IsPrivateOrReserved(net.ParseIP(s)) {
			t.Errorf("%s 是公网地址，不应判定为内网", s)
		}
	}
}

func TestIsPrivateOrReservedInvalid(t *testing.T) {
	// 非法/空值按"不可用"处理，避免被当成公网去外发查询
	for _, s := range []string{"", "not-an-ip", "999.999.999.999", "192.168.1.1:8080"} {
		if !IsPrivateOrReservedString(s) && s != "192.168.1.1:8080" {
			t.Errorf("%q 应判定为不可用（视为内网跳过解析）", s)
		}
	}
	if !IsPrivateOrReserved(nil) {
		t.Error("nil IP 应视为内网/不可用")
	}
}

func TestNormalize(t *testing.T) {
	cases := []struct{ in, want string }{
		{"127.0.0.1", "127.0.0.1"},
		{"localhost", "127.0.0.1"},
		{"::1", "127.0.0.1"},
		{"192.168.1.1:8080", "192.168.1.1"},
		{"[::1]:8080", "127.0.0.1"},
		{"::ffff:203.0.113.5", "203.0.113.5"},
		{"  8.8.8.8  ", "8.8.8.8"},
	}
	for _, c := range cases {
		if got := Normalize(c.in, false); got != c.want {
			t.Errorf("Normalize(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	// strict 模式：非法值丢弃（落库场景），非 strict 原样返回（展示场景）
	if got := Normalize("garbage", true); got != "" {
		t.Errorf("strict 模式非法值应返回空串，实际 %q", got)
	}
	if got := Normalize("garbage", false); got != "garbage" {
		t.Errorf("非 strict 模式应原样返回，实际 %q", got)
	}
}

// TestPrivateWithPortIsNotMisjudged 带端口的字符串必须先剥端口再判定，
// 否则会被当作公网继续外发查询（#192.168.1.1:8080 场景）
func TestPrivateWithPortIsNotMisjudged(t *testing.T) {
	if !IsPrivateOrReservedString(Normalize("192.168.1.1:8080", false)) {
		t.Error("带端口的内网地址规范化后应判定为内网")
	}
}
