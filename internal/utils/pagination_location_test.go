package utils

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// 分页参数解析的回归测试。
//
// 线上缺陷：管理端订单页发的是 {skip, limit}，而解析函数先算 page 再覆盖 size，
// 且换算用的是**旧 size**（守卫还是 size == 10），于是每页 20/50/100 时
// offset 被算成 40/250/1000（应为 20/50/100）→ 切换每页条数后订单行整段漏显。

func newPaginationContext(query string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?"+query, nil)
	return c
}

func TestParsePaginationSkipWithLimit(t *testing.T) {
	cases := []struct {
		query      string
		wantPage   int
		wantSize   int
		wantOffset int
	}{
		// 每页 10（默认）：skip=20 → 第 3 页，offset 20
		{"skip=20&limit=10", 3, 10, 20},
		// 每页 20：skip=20 表示"跳过 20 条"→ 第 2 页，offset 必须是 20（旧实现为 40）
		{"skip=20&limit=20", 2, 20, 20},
		// 每页 50：skip=100 → 第 3 页，offset 100（旧实现为 250）
		{"skip=100&limit=50", 3, 50, 100},
		// 每页 100：skip=200 → 第 3 页，offset 200（旧实现为 1000）
		{"skip=200&limit=100", 3, 100, 200},
		// 首页
		{"skip=0&limit=20", 1, 20, 0},
	}
	for _, c := range cases {
		p := ParsePagination(newPaginationContext(c.query))
		if p.Page != c.wantPage || p.Size != c.wantSize {
			t.Errorf("%s: page/size = %d/%d, want %d/%d", c.query, p.Page, p.Size, c.wantPage, c.wantSize)
		}
		if got := p.GetOffset(); got != c.wantOffset {
			t.Errorf("%s: offset = %d, want %d（漏显就是这里算错）", c.query, got, c.wantOffset)
		}
	}
}

func TestParsePaginationPageAndSize(t *testing.T) {
	p := ParsePagination(newPaginationContext("page=4&page_size=25"))
	if p.Page != 4 || p.Size != 25 {
		t.Errorf("page/size = %d/%d, want 4/25", p.Page, p.Size)
	}
	if p.GetOffset() != 75 {
		t.Errorf("offset = %d, want 75", p.GetOffset())
	}

	// 越界值应被收敛，避免一次性拉全表
	p = ParsePagination(newPaginationContext("page=0&size=100000"))
	if p.Page != 1 {
		t.Errorf("page<1 应收敛为 1，实际 %d", p.Page)
	}
	if p.Size > 100 {
		t.Errorf("size 应被限制在 100 以内，实际 %d", p.Size)
	}
}

// FormatLocation 的统一口径测试：
//   - JSON（含逗号）不能被当成 "国家,城市" 拆分
//   - "本地"/"内网" 原样返回
//   - 展示文本统一用 "国家, 城市"（历史上有 "国家 - 城市" 的异类写法）
func TestFormatLocation(t *testing.T) {
	cases := []struct{ in, want string }{
		{`{"country":"中国","city":"杭州"}`, "中国, 杭州"},
		{`{"country":"美国","city":"","region":"California"}`, "美国, California"},
		{`{"country":"日本"}`, "日本"},
		{"本地", "本地"},
		{"内网", "内网"},
		{"中国, 杭州", "中国, 杭州"},
		{"", ""},
	}
	for _, c := range cases {
		if got := FormatLocation(c.in); got != c.want {
			t.Errorf("FormatLocation(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestFormatLocationJSONNotSplitByComma 锁死历史缺陷：JSON 自带逗号，
// 若先按逗号拆分会把国家解析成 `{"country":"中国"`（前端筛选框曾被填成这种垃圾值）。
func TestFormatLocationJSONNotSplitByComma(t *testing.T) {
	got := FormatLocation(`{"country":"中国","city":"杭州","region":"浙江"}`)
	if got != "中国, 杭州" {
		t.Errorf("含逗号的 JSON 必须按 JSON 解析，实际 %q", got)
	}
	if len(got) > 0 && got[0] == '{' {
		t.Errorf("展示文本不应以 { 开头（说明返回了原始 JSON）：%q", got)
	}
}
