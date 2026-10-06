package node_health

import (
	"errors"
	"testing"
	"time"
)

// waitJob 轮询等待任务结束（最多 3 秒）
func waitJob(t *testing.T, id string) BatchJob {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		job, ok := GetBatchJob(id)
		if ok && job.Status != BatchJobRunning {
			return job
		}
		time.Sleep(10 * time.Millisecond)
	}
	job, _ := GetBatchJob(id)
	t.Fatalf("任务未在预期时间内结束: %+v", job)
	return BatchJob{}
}

func TestStartBatchJobRunsAndReportsProgress(t *testing.T) {
	ResetBatchJobsForTest()

	started, err := StartBatchJob(BatchJobNodes, 3, func(job *BatchJob) {
		job.AddResult(BatchJobItem{NodeID: 1, Status: StatusOnline, Latency: 42})
		job.AddResult(BatchJobItem{NodeID: 2, Status: StatusOffline, Message: "连接失败"})
		job.AddResult(BatchJobItem{NodeID: 3, Status: StatusUnsupported})
		job.Finish("完成")
	})
	if err != nil {
		t.Fatalf("启动任务失败: %v", err)
	}
	if started.Total != 3 || started.Status != BatchJobRunning {
		t.Fatalf("初始快照不符: %+v", started)
	}

	job := waitJob(t, started.ID)
	if job.Status != BatchJobDone {
		t.Errorf("任务状态应为 done，实际 %s", job.Status)
	}
	if job.Done != 3 || job.Online != 1 || job.Failed != 1 || job.Unsupported != 1 {
		t.Errorf("计数不符: done=%d online=%d failed=%d unsupported=%d", job.Done, job.Online, job.Failed, job.Unsupported)
	}
	if len(job.Results) != 3 {
		t.Errorf("结果条数应为 3，实际 %d", len(job.Results))
	}
	if job.FinishedAt == nil {
		t.Error("完成任务应有 FinishedAt")
	}
}

func TestStartBatchJobRejectsConcurrentSameKind(t *testing.T) {
	ResetBatchJobsForTest()

	release := make(chan struct{})
	first, err := StartBatchJob(BatchJobNodes, 1, func(job *BatchJob) {
		<-release
		job.AddResult(BatchJobItem{NodeID: 1, Status: StatusOnline})
	})
	if err != nil {
		t.Fatalf("首次启动应成功: %v", err)
	}

	second, err := StartBatchJob(BatchJobNodes, 5, func(job *BatchJob) {
		t.Error("同类型任务在跑时不应再启动新任务")
	})
	if !errors.Is(err, ErrBatchJobRunning) {
		t.Fatalf("应返回 ErrBatchJobRunning，实际 %v", err)
	}
	if second.ID != first.ID {
		t.Errorf("应返回正在运行的任务，实际 id=%s want=%s", second.ID, first.ID)
	}

	// 不同类型可以并行
	if _, err := StartBatchJob(BatchJobCustom, 1, func(job *BatchJob) {}); err != nil {
		t.Errorf("不同类型任务应可同时启动: %v", err)
	}

	close(release)
	waitJob(t, first.ID)

	// 前一个任务结束后可以再次启动同类型任务
	if _, err := StartBatchJob(BatchJobNodes, 1, func(job *BatchJob) {}); err != nil {
		t.Errorf("任务结束后应可再次启动: %v", err)
	}
}

func TestStartBatchJobPanicMarksFailed(t *testing.T) {
	ResetBatchJobsForTest()

	job, err := StartBatchJob(BatchJobCustom, 1, func(job *BatchJob) {
		panic("boom")
	})
	if err != nil {
		t.Fatalf("启动失败: %v", err)
	}

	done := waitJob(t, job.ID)
	if done.Status != BatchJobFailed {
		t.Errorf("panic 后任务应为 failed，实际 %s", done.Status)
	}
	if done.Message == "" {
		t.Error("应记录异常信息")
	}
	// panic 兜底必须释放占位，否则后续测速永远提示“已有任务”
	if IsBatchJobRunningForTest(BatchJobCustom) {
		t.Error("panic 后应释放同类型占位")
	}
}

func TestLatestBatchJobAndSnapshotCopy(t *testing.T) {
	ResetBatchJobsForTest()

	job, err := StartBatchJob(BatchJobNodes, 1, func(job *BatchJob) {
		job.AddResult(BatchJobItem{NodeID: 7, Status: StatusOnline, Latency: 5})
	})
	if err != nil {
		t.Fatalf("启动失败: %v", err)
	}
	waitJob(t, job.ID)

	latest, ok := LatestBatchJob(BatchJobNodes)
	if !ok || latest.ID != job.ID {
		t.Fatalf("LatestBatchJob 应返回最近任务，实际 ok=%v id=%s", ok, latest.ID)
	}

	// 快照是副本：外部改动不应影响内部状态
	latest.Results[0].Status = "tampered"
	latest.Done = 999
	again, _ := GetBatchJob(job.ID)
	if again.Done != 1 || again.Results[0].Status != StatusOnline {
		t.Errorf("快照应为副本，内部状态被污染: done=%d status=%s", again.Done, again.Results[0].Status)
	}

	if _, ok := GetBatchJob("not-exist"); ok {
		t.Error("不存在的任务应返回 false")
	}
}
