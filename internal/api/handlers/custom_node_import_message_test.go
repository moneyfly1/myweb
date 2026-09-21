package handlers

import (
	"strings"
	"testing"
)

// TestCustomNodeImportMessage 导入结果提示必须把「解析 / 新增 / 已存在跳过 / 失败」分开说。
//
// 线上真实问题：11 个专线节点早已导入，管理员再次导入时后端只回报「成功 0 个」，
// 前端于是统一提示「没有解析到可导入的节点」，管理员因此以为「链接无法解析」——
// 实际是全部被「已存在」跳过了。这组用例把提示语锁死，避免同类误导再出现。
func TestCustomNodeImportMessage(t *testing.T) {
	cases := []struct {
		name                      string
		parsed, imported, skipped int
		failed                    int
		wantContains              []string
		wantNotContains           []string
	}{
		{
			name:   "全部已存在（线上实际场景）",
			parsed: 11, imported: 0, skipped: 11, failed: 0,
			wantContains: []string{"11", "已存在", "本次未新增"},
			// 关键：不能说成「没有解析到 / 请检查链接格式」，那会让人以为解析失败
			wantNotContains: []string{"没有解析到", "请检查链接格式"},
		},
		{
			name:   "全部新增",
			parsed: 5, imported: 5, skipped: 0, failed: 0,
			wantContains: []string{"成功导入 5 个"},
		},
		{
			name:   "部分新增、部分已存在",
			parsed: 5, imported: 3, skipped: 2, failed: 0,
			wantContains: []string{"成功导入 3 个", "跳过 2 个", "已存在"},
		},
		{
			name:   "新增 + 跳过 + 失败",
			parsed: 6, imported: 3, skipped: 1, failed: 2,
			wantContains: []string{"成功导入 3 个", "跳过 1 个", "失败 2 个"},
		},
		{
			name:   "全部解析失败",
			parsed: 4, imported: 0, skipped: 0, failed: 4,
			wantContains: []string{"失败 4 个"},
		},
		{
			name:   "真的没解析到任何东西",
			parsed: 0, imported: 0, skipped: 0, failed: 0,
			wantContains: []string{"没有解析到", "链接格式"},
		},
	}

	for _, c := range cases {
		got := customNodeImportMessage(c.parsed, c.imported, c.skipped, c.failed)
		for _, want := range c.wantContains {
			if !strings.Contains(got, want) {
				t.Errorf("%s：提示 %q 应包含 %q", c.name, got, want)
			}
		}
		for _, bad := range c.wantNotContains {
			if strings.Contains(got, bad) {
				t.Errorf("%s：提示 %q 不应包含 %q（会误导成解析失败）", c.name, got, bad)
			}
		}
	}
}
