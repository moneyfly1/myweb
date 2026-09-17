package cache

import (
	"testing"
	"time"
)

// TestSubscriptionConfigKeysCoverAllFormats 订阅配置缓存的 key 清单必须覆盖写入侧的全部格式。
// 历史缺陷：清除侧只删 clash/base64，漏了 ssr、base64_v2rayn、ssr_v2rayn，
// 导致设备踢下线 / 改订阅后这两类客户端仍能拿到变更前的节点。
func TestSubscriptionConfigKeysCoverAllFormats(t *testing.T) {
	keys := SubscriptionConfigKeys("https://example.com/sub?token=abc")
	if len(keys) != len(SubscriptionConfigFormats) {
		t.Fatalf("key 数量 %d 与格式数量 %d 不一致", len(keys), len(SubscriptionConfigFormats))
	}

	want := map[string]string{
		"clash":         SubscriptionConfigKey("https://example.com/sub?token=abc", "clash"),
		"base64":        SubscriptionConfigKey("https://example.com/sub?token=abc", "base64"),
		"ssr":           SubscriptionConfigKey("https://example.com/sub?token=abc", "ssr"),
		"base64_v2rayn": SubscriptionConfigKey("https://example.com/sub?token=abc", "base64_v2rayn"),
		"ssr_v2rayn":    SubscriptionConfigKey("https://example.com/sub?token=abc", "ssr_v2rayn"),
	}
	got := make(map[string]bool, len(keys))
	for _, k := range keys {
		got[k] = true
	}
	for format, key := range want {
		if !got[key] {
			t.Errorf("缺少格式 %q 的缓存 key: %s", format, key)
		}
	}
}

// TestGenerationGuardsStaleWrite 验证"陈旧回填"防护：
// 读路径先取版本号 → 数据变更方清缓存（版本 +1）→ 读路径再写回时必须被拒绝，
// 否则旧数据会被缓存住直到 TTL 到期（套餐改价后前台仍显示旧价格）。
//
// 需要真实 Redis；未配置时跳过（本地/CI 无 Redis 也能跑通其余测试）。
func TestGenerationGuardsStaleWrite(t *testing.T) {
	if err := InitRedis(); err != nil {
		t.Skipf("跳过：Redis 不可用 (%v)", err)
	}
	defer func() { _ = Close() }()

	const key = "test:stale-repopulate"
	_ = DelWithGeneration(key) // 从干净状态开始

	// 1) 读路径查库前取版本号
	generation := Generation(key)

	// 2) 期间数据变更：清除缓存（版本 +1）
	if err := DelWithGeneration(key); err != nil {
		t.Fatalf("清除缓存失败: %v", err)
	}

	// 3) 陈旧回填必须被拒绝
	written, err := SetIfGeneration(key, generation, "旧数据", time.Minute)
	if err != nil {
		t.Fatalf("SetIfGeneration 出错: %v", err)
	}
	if written {
		t.Error("版本已变，陈旧写入必须被拒绝")
	}
	if _, err := Get(key); err == nil {
		t.Error("陈旧写入被拒绝后缓存应仍然为空")
	}

	// 4) 用新版本号写入应当成功
	fresh := Generation(key)
	written, err = SetIfGeneration(key, fresh, "新数据", time.Minute)
	if err != nil {
		t.Fatalf("SetIfGeneration 出错: %v", err)
	}
	if !written {
		t.Fatal("版本未变，写入应当成功")
	}
	got, err := Get(key)
	if err != nil || got != "新数据" {
		t.Fatalf("缓存内容应为新数据，实际 %q (err=%v)", got, err)
	}

	// 5) 版本号确实随清除递增
	before := Generation(key)
	_ = DelWithGeneration(key)
	if after := Generation(key); after <= before {
		t.Errorf("清除缓存后版本号应递增：before=%d after=%d", before, after)
	}
	_ = Del(key)
}
