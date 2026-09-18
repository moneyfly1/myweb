package geoip

import "testing"

// TestTranslateCityNameRealData 用生产库（DB-IP City Lite）真实返回的名称，
// 固化"城市名翻译"的现状，便于判断"位置不准"的成因。
//
// DB-IP City Lite 只有英文名（其 city/subdivision 没有 zh-CN 键），因此中文名
// 完全依赖这张手写映射表；表里没有的城市会退化成"中文省 + 英文城市"，
// 在界面上就表现为"中国, 河南 Guancheng""中国, 天津 Youyilu"这类中英混排。
func TestTranslateCityNameRealData(t *testing.T) {
	cases := []struct {
		cityEN   string
		regionEN string
		want     string
		why      string
	}{
		{"Guangzhou", "Guangdong", "广州", "表内城市：正常翻成中文"},
		{"Zhu Cheng City", "Shandong", "诸城", "去掉 City 后缀后命中表内特例"},
		{"Jinrongjie (Xicheng District)", "Beijing", "金融街", "去括号后命中表内特例"},
		{"Zhengzhou", "Henan", "郑州", "表内城市"},
		{"Guancheng", "Henan", "河南 Guancheng", "表内没有 → 中文省 + 英文城市（中英混排）"},
		{"Youyilu", "Tianjin", "天津 Youyilu", "同上，街道级英文名无法翻译"},
		{"Chengde", "Hebei", "河北 Chengde", "同上，承德不在表内"},
		{"Wenquan", "Fujian", "福建 Wenquan", "同上，区级名不在表内"},
	}
	for _, c := range cases {
		if got := translateCityName(c.cityEN, c.regionEN); got != c.want {
			t.Errorf("translateCityName(%q, %q) = %q，期望 %q（%s）",
				c.cityEN, c.regionEN, got, c.want, c.why)
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

// TestLocationDisplayOfNonCNKeepsParenthetical 非中国城市的括号内容不会被清理
// （去括号只发生在 translateCityName 内，而它只对中国 IP 调用），
// 所以美国 IP 会显示成"美国, Los Angeles (Downtown Los Angeles)"。
func TestLocationDisplayOfNonCNKeepsParenthetical(t *testing.T) {
	got := LocationDisplayOf("美国", "Los Angeles (Downtown Los Angeles)", "California")
	want := "美国, Los Angeles (Downtown Los Angeles)"
	if got != want {
		t.Errorf("当前行为应为 %q，实际 %q", want, got)
	}
}

// TestLocationDisplayOfRule 展示拼接规则：国家 + ", " + 城市；无城市退回省份；无省份只剩国家
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
