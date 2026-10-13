package activity

import (
	"testing"
	"time"

	"cboard-go/internal/core/database"
)

// 说明：这些测试不依赖 Redis —— GetStore() 在 Redis 不可用时自动降级为内存实现，
// 正好覆盖「没配 Redis 也要能用」这条降级路径。
func TestMain(m *testing.M) {
	ResetStoreForTest()
	m.Run()
}

func resetStore() {
	ResetStoreForTest()
	GetStore() // 触发重新选择（无 Redis → 内存）
}

func TestSubscriptionFetchMarksActive(t *testing.T) {
	resetStore()
	MarkSubscriptionFetch(101, "ClashforWindows/0.20.39", "1.2.3.4", "clash", "Clash for Windows")

	st := StateFor(101)
	if st.State != StateActive {
		t.Fatalf("拉订阅后状态应为 active，实际 %q", st.State)
	}
	if st.SubFetchCount != 1 {
		t.Fatalf("拉取次数应为 1，实际 %d", st.SubFetchCount)
	}
	if st.UA == "" || st.Format != "clash" || st.IP != "1.2.3.4" {
		t.Fatalf("订阅元信息未记录: %+v", st)
	}

	// 再拉一次 → 计数累加（元信息按 JSON 保存，能正确读回上一份）
	MarkSubscriptionFetch(101, "ClashforWindows/0.20.39", "1.2.3.4", "clash", "Clash for Windows")
	if st2 := StateFor(101); st2.SubFetchCount != 2 {
		t.Fatalf("第二次拉取后计数应为 2，实际 %d", st2.SubFetchCount)
	}
}

func TestStateTransitions(t *testing.T) {
	resetStore()
	oldActive, oldIdle := ActiveWindow, IdleWindow
	defer func() { ActiveWindow, IdleWindow = oldActive, oldIdle }()

	// 从未使用 → silent
	if st := StateFor(202); st.State != StateSilent {
		t.Fatalf("从未拉取应为 silent，实际 %q", st.State)
	}

	// 直接往 zset 写一个「很久以前」的时间戳 → idle（超过 ActiveWindow，但仍在 IdleWindow 内）
	s := GetStore()
	s.zadd(keyActive, 202, time.Now().Add(-2*time.Hour).Unix())
	if st := StateFor(202); st.State != StateIdle {
		t.Fatalf("2 小时前的信号应为 idle，实际 %q", st.State)
	}

	// 超过 IdleWindow → silent
	resetStore()
	GetStore().zadd(keyActive, 203, time.Now().Add(-48*time.Hour).Unix())
	if st := StateFor(203); st.State != StateSilent {
		t.Fatalf("48 小时前的信号应为 silent，实际 %q", st.State)
	}
}

func TestNodeReportMakesOnline(t *testing.T) {
	resetStore()
	MarkNodeUsers([]NodeUserStat{{UserID: 301, Conns: 3, Up: 10, Down: 20}})

	st := StateFor(301)
	if st.State != StateOnline {
		t.Fatalf("节点上报 conns>0 应为 online，实际 %q", st.State)
	}
	if st.OnlineConns != 3 {
		t.Fatalf("在线连接数应为 3，实际 %d", st.OnlineConns)
	}

	// conns=0 的上报 → 不再在线，但仍算活跃（节点还认这个用户）
	MarkNodeUsers([]NodeUserStat{{UserID: 301, Conns: 0}})
	if st := StateFor(301); st.State != StateActive {
		t.Fatalf("conns=0 后应为 active（不是 online），实际 %q", st.State)
	}

	// 空数组不应 panic，也不应影响状态
	MarkNodeUsers(nil)
	if st := StateFor(301); st.State != StateActive {
		t.Fatalf("空上报后状态不应变化，实际 %q", st.State)
	}
}

func TestCountsAndFilters(t *testing.T) {
	resetStore()
	MarkSubscriptionFetch(401, "v2rayN/6.31", "5.5.5.5", "universal", "v2rayN")
	MarkNodeUsers([]NodeUserStat{{UserID: 402, Conns: 2}})
	GetStore().zadd(keyActive, 403, time.Now().Add(-3*time.Hour).Unix())

	c := Counts()
	if c.Online != 1 {
		t.Fatalf("在线数应为 1，实际 %d", c.Online)
	}
	if c.Active != 2 { // 401 拉订阅 + 402 节点上报；403 是 3 小时前
		t.Fatalf("活跃数应为 2，实际 %d", c.Active)
	}
	if c.Idle != 3 {
		t.Fatalf("24h 内有信号数应为 3，实际 %d", c.Idle)
	}

	if ids := FilterIDs(StateOnline); len(ids) != 1 || ids[0] != 402 {
		t.Fatalf("online 过滤结果异常: %v", ids)
	}
	if ids := FilterIDs(StateIdle); len(ids) != 1 || ids[0] != 403 {
		t.Fatalf("idle 过滤应只含 403，实际 %v", ids)
	}
	if ids := SilentIDs(); len(ids) != 3 {
		t.Fatalf("24h 内有信号的应有 3 个，实际 %v", ids)
	}
}

func TestStatesForEmptyInput(t *testing.T) {
	resetStore()
	if got := StatesFor(nil); len(got) != 0 {
		t.Fatalf("空输入应返回空 map，实际 %v", got)
	}
	MarkSubscriptionFetch(0, "ua", "ip", "clash", "") // userID=0 应被忽略而不是写入
	if c := Counts(); c.Active != 0 {
		t.Fatalf("userID=0 不应产生信号，实际 active=%d", c.Active)
	}
}

func TestPersistSnapshotDegradesWithoutDB(t *testing.T) {
	// 未初始化数据库时 persist 必须安静返回（不能 panic / 不能阻塞）
	database.DB = nil
	resetStore()
	MarkNodeUsers([]NodeUserStat{{UserID: 501, Conns: 1}})
	MarkSubscriptionFetch(502, "ua", "1.1.1.1", "clash", "Clash")
	time.Sleep(50 * time.Millisecond) // 等后台落库协程走完（此处应直接返回）
	if st := StateFor(501); st.State != StateOnline {
		t.Fatalf("无数据库时在线状态仍应由内存存储提供，实际 %q", st.State)
	}
}
