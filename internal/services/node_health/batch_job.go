package node_health

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"cboard-go/internal/utils"
)

// 批量测速任务（异步）：HTTP 请求只负责启动，真正的测速在后台跑，前端轮询进度。
//
// 背景：此前「批量测速」在 HTTP 请求内同步执行。600+ 个节点要 20 秒以上、专线节点
// 更多，而浏览器默认 10 秒超时 —— 前端看到「测速超时」并放弃请求，界面又只在成功
// 分支刷新列表，于是「点了没结果」；服务端其实已经测完并写库。

// BatchJobKind 批量测速任务类型
type BatchJobKind string

const (
	// BatchJobNodes 普通节点列表
	BatchJobNodes BatchJobKind = "nodes"
	// BatchJobCustom 专线节点
	BatchJobCustom BatchJobKind = "custom"
)

// 任务状态
const (
	BatchJobRunning = "running"
	BatchJobDone    = "done"
	BatchJobFailed  = "failed"
)

// ErrBatchJobRunning 同类型任务已在运行
var ErrBatchJobRunning = errors.New("已有测速任务正在进行")

// BatchJobItem 单个节点的测速结果
type BatchJobItem struct {
	NodeID   uint   `json:"node_id"`
	Status   string `json:"status"`
	Latency  int    `json:"latency"`
	Message  string `json:"message,omitempty"`
	IsActive *bool  `json:"is_active,omitempty"`
}

// BatchJob 批量测速任务
type BatchJob struct {
	ID          string         `json:"job_id"`
	Kind        BatchJobKind   `json:"kind"`
	Total       int            `json:"total"`
	Done        int            `json:"done"`
	Online      int            `json:"online"`
	Failed      int            `json:"failed"`
	Unsupported int            `json:"unsupported"`
	Status      string         `json:"status"`
	Message     string         `json:"message,omitempty"`
	StartedAt   time.Time      `json:"started_at"`
	FinishedAt  *time.Time     `json:"finished_at,omitempty"`
	Results     []BatchJobItem `json:"results"`
}

// 任务注册表：同类型同时只允许一个任务（避免连点把服务器打满），
// 并保留最近一次任务，便于页面刷新后继续轮询。
var (
	batchMu      sync.Mutex
	batchJobs    = make(map[string]*BatchJob)
	batchLatest  = make(map[BatchJobKind]string)
	batchRunning = make(map[BatchJobKind]string)
	batchSeq     uint64
)

// 已完成任务保留上限（防止长时间运行后内存堆积）
const batchJobKeep = 20

// StartBatchJob 启动一个异步批量测速任务，立即返回任务快照；真正的测速在后台执行。
// 同类型已有任务在运行时返回该任务与 ErrBatchJobRunning。
func StartBatchJob(kind BatchJobKind, total int, runner func(*BatchJob)) (BatchJob, error) {
	batchMu.Lock()
	if id, ok := batchRunning[kind]; ok {
		if j := batchJobs[id]; j != nil {
			snap := j.snapshotLocked()
			batchMu.Unlock()
			return snap, ErrBatchJobRunning
		}
	}
	batchSeq++
	job := &BatchJob{
		ID:        fmt.Sprintf("%s-%d-%d", kind, time.Now().Unix(), batchSeq),
		Kind:      kind,
		Total:     total,
		Status:    BatchJobRunning,
		StartedAt: time.Now(),
		Results:   make([]BatchJobItem, 0, total),
	}
	batchJobs[job.ID] = job
	batchRunning[kind] = job.ID
	batchLatest[kind] = job.ID
	pruneLocked()
	batchMu.Unlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				utils.LogErrorMsg("批量测速任务 panic: %v", r)
				batchMu.Lock()
				job.Status = BatchJobFailed
				job.Message = fmt.Sprintf("测速任务异常: %v", r)
				now := time.Now()
				job.FinishedAt = &now
				if batchRunning[kind] == job.ID {
					delete(batchRunning, kind)
				}
				batchMu.Unlock()
			}
		}()

		runner(job)

		batchMu.Lock()
		if job.Status == BatchJobRunning {
			job.Status = BatchJobDone
		}
		if job.FinishedAt == nil {
			now := time.Now()
			job.FinishedAt = &now
		}
		if batchRunning[kind] == job.ID {
			delete(batchRunning, kind)
		}
		batchMu.Unlock()
	}()

	return job.snapshot(), nil
}

// GetBatchJob 按 id 取任务快照（含进度与结果，均为副本）
func GetBatchJob(id string) (BatchJob, bool) {
	batchMu.Lock()
	defer batchMu.Unlock()
	j, ok := batchJobs[id]
	if !ok {
		return BatchJob{}, false
	}
	return j.snapshotLocked(), true
}

// LatestBatchJob 取某类型最近一次任务快照（页面刷新后仍可继续轮询）
func LatestBatchJob(kind BatchJobKind) (BatchJob, bool) {
	batchMu.Lock()
	defer batchMu.Unlock()
	id, ok := batchLatest[kind]
	if !ok {
		return BatchJob{}, false
	}
	j, ok := batchJobs[id]
	if !ok {
		return BatchJob{}, false
	}
	return j.snapshotLocked(), true
}

// AddResult 记录一个节点的结果并推进进度（线程安全：普通节点测速是 10 并发回调）
func (j *BatchJob) AddResult(item BatchJobItem) {
	batchMu.Lock()
	defer batchMu.Unlock()
	j.Done++
	switch item.Status {
	case StatusOnline:
		j.Online++
	case StatusUnsupported:
		j.Unsupported++
	default:
		j.Failed++
	}
	j.Results = append(j.Results, item)
}

// Finish 标记任务完成
func (j *BatchJob) Finish(message string) {
	batchMu.Lock()
	defer batchMu.Unlock()
	if j.Status == BatchJobRunning {
		j.Status = BatchJobDone
	}
	if message != "" {
		j.Message = message
	}
}

// Fail 标记任务失败
func (j *BatchJob) Fail(message string) {
	batchMu.Lock()
	defer batchMu.Unlock()
	j.Status = BatchJobFailed
	j.Message = message
}

// snapshot 返回任务副本（调用方持有锁）
func (j *BatchJob) snapshotLocked() BatchJob {
	cp := *j
	cp.Results = append([]BatchJobItem(nil), j.Results...)
	return cp
}

// Snapshot 返回任务快照（副本，供后台任务结束时汇总用）
func (j *BatchJob) Snapshot() BatchJob {
	return j.snapshot()
}

func (j *BatchJob) snapshot() BatchJob {
	batchMu.Lock()
	defer batchMu.Unlock()
	return j.snapshotLocked()
}

// pruneLocked 清理超过保留上限的已完成任务（调用方持有锁）
func pruneLocked() {
	if len(batchJobs) <= batchJobKeep {
		return
	}
	// 先删非最新、且已结束的旧任务
	for id, j := range batchJobs {
		if len(batchJobs) <= batchJobKeep {
			break
		}
		if j.Status == BatchJobRunning {
			continue
		}
		if batchLatest[j.Kind] == id {
			continue
		}
		delete(batchJobs, id)
	}
}

// IsBatchJobRunningForTest 仅测试使用：查询某类型是否仍被任务占位
func IsBatchJobRunningForTest(kind BatchJobKind) bool {
	batchMu.Lock()
	defer batchMu.Unlock()
	_, ok := batchRunning[kind]
	return ok
}

// ResetBatchJobsForTest 仅测试使用：清空任务表
func ResetBatchJobsForTest() {
	batchMu.Lock()
	defer batchMu.Unlock()
	batchJobs = make(map[string]*BatchJob)
	batchLatest = make(map[BatchJobKind]string)
	batchRunning = make(map[BatchJobKind]string)
	batchSeq = 0
}
