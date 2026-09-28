package repo_sync

import (
	"errors"
	"testing"
	"time"

	"cboard-go/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupRepoSyncTestDB 内存 SQLite（含 system_configs 表）
func setupRepoSyncTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.SystemConfig{}); err != nil {
		t.Fatalf("迁移 system_configs 失败: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}

func seedRepoSyncConfig(t *testing.T, db *gorm.DB, enabled bool) {
	t.Helper()
	rows := []models.SystemConfig{
		{Key: KeyEnabled, Value: map[bool]string{true: "true", false: "false"}[enabled], Category: Category},
		{Key: KeyToken, Value: "ghp_dummy", Category: Category},
		{Key: KeyOwner, Value: "owner", Category: Category},
		{Key: KeyRepo, Value: "repo", Category: Category},
		{Key: KeyPath, Value: "nodes", Category: Category},
		{Key: KeyLastTime, Value: "2020-01-01T00:00:00", Category: Category},
	}
	for _, r := range rows {
		if err := db.Create(&r).Error; err != nil {
			t.Fatalf("写入配置失败: %v", err)
		}
	}
}

// waitUntil 轮询等待条件成立（最多 d）
func waitUntil(t *testing.T, d time.Duration, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return cond()
}

// 「立即下载」与定时任务各自 NewService()，实例内的锁无法跨实例互斥 ——
// 这里锁定修复后的语义：同步状态是进程级共享的。
func TestIsSyncingIsCrossInstance(t *testing.T) {
	svcA := &Service{}
	svcB := &Service{}

	if IsSyncing() {
		t.Fatal("初始状态不应在同步中")
	}
	if svcA.IsRunning() || svcB.IsRunning() {
		t.Fatal("初始 IsRunning 应为 false")
	}

	unlock, ok := tryLockSync()
	if !ok {
		t.Fatal("抢占同步锁失败")
	}
	if !IsSyncing() {
		t.Fatal("持锁期间 IsSyncing 应为 true")
	}
	if !svcB.IsRunning() {
		t.Error("另一个 Service 实例也应看到同步中（此前实例级锁恒为 false）")
	}

	unlock()
	if IsSyncing() {
		t.Fatal("释放后 IsSyncing 应为 false")
	}
}

// 已在同步时，同步版 SyncNow 必须立刻返回 ErrSyncRunning，不能并发跑第二次。
func TestSyncNowReturnsErrSyncRunning(t *testing.T) {
	unlock, ok := tryLockSync()
	if !ok {
		t.Fatal("抢占同步锁失败")
	}
	defer unlock()

	svc := &Service{}
	if _, err := svc.SyncNow(); !errors.Is(err, ErrSyncRunning) {
		t.Fatalf("应返回 ErrSyncRunning，实际: %v", err)
	}
}

// 已在同步时，异步入口必须返回「未启动」而不是排队第二次同步。
func TestStartSyncAsyncReturnsErrWhenBusy(t *testing.T) {
	unlock, ok := tryLockSync()
	if !ok {
		t.Fatal("抢占同步锁失败")
	}
	defer unlock()

	svc := &Service{}
	started, err := svc.StartSyncAsync()
	if started {
		t.Error("已占用时应返回 started=false")
	}
	if !errors.Is(err, ErrSyncRunning) {
		t.Fatalf("应返回 ErrSyncRunning，实际: %v", err)
	}
}

// 异步入口：立即返回 started=true，后台执行（配置缺失时快速失败），并最终释放锁。
func TestStartSyncAsyncRunsInBackgroundAndReleases(t *testing.T) {
	db := setupRepoSyncTestDB(t)
	svc := &Service{db: db}

	started, err := svc.StartSyncAsync()
	if err != nil || !started {
		t.Fatalf("应启动成功，实际 started=%v err=%v", started, err)
	}

	// 配置为空 → 后台任务立刻失败并写回失败状态，锁必须释放
	if !waitUntil(t, 3*time.Second, func() bool { return !IsSyncing() }) {
		t.Fatal("后台同步结束后未释放同步锁")
	}
	if !waitUntil(t, 3*time.Second, func() bool { return svc.getConfigValue(KeyLastStatus) == "failed" }) {
		t.Fatalf("应写回 failed 状态，实际: %q", svc.getConfigValue(KeyLastStatus))
	}
	msg := svc.getConfigValue(KeyLastMessage)
	if msg == "" {
		t.Error("失败原因应写入 last_message")
	}
}

// 手动同步进行中，定时任务必须跳过本轮（避免同一目录并发下载/清理）。
func TestTickSkipsWhileSyncing(t *testing.T) {
	db := setupRepoSyncTestDB(t)
	seedRepoSyncConfig(t, db, true)
	svc := &Service{db: db}

	// 配置里 last_time 很早 → ShouldRunNow 为 true
	if !svc.ShouldRunNow() {
		t.Fatal("前置条件：此刻应到同步时间")
	}

	unlock, ok := tryLockSync()
	if !ok {
		t.Fatal("抢占同步锁失败")
	}
	svc.Tick()
	unlock()

	if got := svc.getConfigValue(KeyLastStatus); got != "" {
		t.Errorf("同步进行中时 Tick 不应执行同步（status 被写成 %q）", got)
	}
}
