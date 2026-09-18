package geoip

import "testing"

// TestTranslateCityNameReturnsEmptyWhenUnknown 翻译表查不到时返回空串。
//
// 空串表示"翻不出来"：调用方据此丢弃英文城市名、退回中文省份，
// 而不是像历史实现那样拼出"河南 Guancheng"这种中英混排。
// （DB-IP City Lite 只有英文名，中文完全依赖这张手写映射表。）
func TestTranslateCityNameReturnsEmptyWhenUnknown(t *testing.T) {
	cases := []struct {
		cityEN string
		want   string
		why    string
	}{
		{"Guangzhou", "广州", "表内城市"},
		{"Zhu Cheng City", "诸城", "去 City 后缀后命中表内特例"},
		{"Jinrongjie (Xicheng District)", "金融街", "去括号后命中表内特例"},
		{"Zhengzhou", "郑州", "表内城市"},
		{"Guancheng", "", "表内没有 → 空串（由展示层退回中文省份）"},
		{"Youyilu", "", "街道级英文名同样翻不出来"},
		{"Chengde", "", "承德不在表内"},
		{"Wenquan", "", "区级名不在表内"},
	}
	for _, c := range cases {
		if got := translateCityName(c.cityEN); got != c.want {
			t.Errorf("translateCityName(%q) = %q，期望 %q（%s）", c.cityEN, got, c.want, c.why)
		}
	}
}

// TestTranslateRegionName 省份翻译：表内命中中文，表外原样返回英文
func TestTranslateRegionName(t *testing.T) {
	if got := translateRegionName("Guangdong"); got != "广东" {
		t.Errorf("Guangdong 应译为广东，实际 %q", got)
	}
	if got := translateRegionName("California"); got != "California" {
		t.Errorf("表外地区应原样返回，实际 %q", got)
	}
}

// TestCleanCityName 城市名清理：去括号补充、去常见后缀（各国通用）
func TestCleanCityName(t *testing.T) {
	cases := map[string]string{
		"Jinrongjie (Xicheng District)":      "Jinrongjie",
		"Los Angeles (Downtown Los Angeles)": "Los Angeles",
		"Singapore (Pioneer)":                "Singapore",
		"Zhu Cheng City":                     "Zhu Cheng",
		"  Guangzhou  ":                      "Guangzhou",
	}
	for in, want := range cases {
		if got := cleanCityName(in); got != want {
			t.Errorf("cleanCityName(%q) = %q，期望 %q", in, got, want)
		}
	}
}

// TestFinalizeLocationNamesChinaOnlyChinese 中国 IP 的位置只保留中文：
// 城市翻不出来就丢城市（退回省份）、省份也不是中文就丢省份（只留国家）
func TestFinalizeLocationNamesChinaOnlyChinese(t *testing.T) {
	// 城市翻不出来 → 丢城市，保留中文省份
	loc := &LocationInfo{Country: "中国", CountryCode: "CN", City: "Guancheng", Region: "Henan"}
	finalizeLocationNames(loc)
	if loc.City != "" || loc.Region != "河南" {
		t.Errorf("应丢英文城市并保留中文省份，实际 city=%q region=%q", loc.City, loc.Region)
	}

	// 城市能翻出来 → 用中文城市
	loc2 := &LocationInfo{Country: "中国", CountryCode: "CN", City: "Guangzhou (Tianhe)", Region: "Guangdong"}
	finalizeLocationNames(loc2)
	if loc2.City != "广州" || loc2.Region != "广东" {
		t.Errorf("应译为中文城市/省份，实际 city=%q region=%q", loc2.City, loc2.Region)
	}

	// 省份也翻不出来 → 一并丢掉，避免位置列出现英文
	loc3 := &LocationInfo{Country: "中国", CountryCode: "CN", City: "Unknownville", Region: "Nowhere"}
	finalizeLocationNames(loc3)
	if loc3.City != "" || loc3.Region != "" {
		t.Errorf("中英混排都应丢弃，实际 city=%q region=%q", loc3.City, loc3.Region)
	}

	// 非中国：保留清理后的当地名称
	loc4 := &LocationInfo{Country: "美国", CountryCode: "US", City: "Los Angeles (Downtown Los Angeles)", Region: "California"}
	finalizeLocationNames(loc4)
	if loc4.City != "Los Angeles" {
		t.Errorf("非中国城市应保留并去括号，实际 %q", loc4.City)
	}
}

// TestLocationDisplayOfCleansHistoricalValues 展示层清洗历史存量值：
// 库里已存的"中国, 河南 Guancheng"这类中英混排、以及带括号的英文城市，
// 在展示时被整理成干净文本（历史数据无法回填，只能在这一层兜底）。
func TestLocationDisplayOfCleansHistoricalValues(t *testing.T) {
	cases := []struct{ country, city, region, want, why string }{
		{"中国", "河南 Guancheng", "Henan", "中国, 河南", "中英混排 → 退回省份"},
		{"中国", "天津 Youyilu", "Tianjin", "中国, 天津", "同上"},
		{"中国", "河北 Chengde", "Hebei", "中国, 河北", "同上"},
		{"中国", "Guangzhou", "Guangdong", "中国, 广州", "历史英文城市名就地翻译为中文"},
		{"中国", "北京", "北京市", "中国, 北京", "GeoLite2 已是中文名 → 原样保留"},
		{"中国", "广州", "广东", "中国, 广州", "中文城市照常显示"},
		{"美国", "Los Angeles (Downtown Los Angeles)", "California", "美国, Los Angeles", "非中国：去括号"},
		{"日本", "Shinagawa (1 Chome)", "Tokyo", "日本, Shinagawa", "非中国：去括号"},
	}
	for _, c := range cases {
		if got := LocationDisplayOf(c.country, c.city, c.region); got != c.want {
			t.Errorf("LocationDisplayOf(%q,%q,%q) = %q，期望 %q（%s）",
				c.country, c.city, c.region, got, c.want, c.why)
		}
	}
}

// TestLocationDisplayOfRule 拼接规则：国家 + ", " + 城市；无城市退回省份；无省份只剩国家
func TestLocationDisplayOfRule(t *testing.T) {
	if got := LocationDisplayOf("中国", "广州", "广东"); got != "中国, 广州" {
		t.Errorf("有城市时应显示城市，实际 %q", got)
	}
	if got := LocationDisplayOf("中国", "", "河南"); got != "中国, 河南" {
		t.Errorf("无城市时应退回省份，实际 %q", got)
	}
	if got := LocationDisplayOf("中国", "", ""); got != "中国" {
		t.Errorf("只有国家时应只显示国家，实际 %q", got)
	}
	if got := LocationDisplayOf("", "广州", "广东"); got != "" {
		t.Errorf("无国家时应返回空，实际 %q", got)
	}
}
