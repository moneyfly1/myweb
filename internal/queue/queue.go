package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"cboard-go/internal/core/cache"

	"github.com/hibiken/asynq"
)

// 任务类型常量
const (
	TypeEmailSend       = "email:send"       // 发送邮件
	TypeSubscriptionLog = "subscription:log" // 订阅日志
	TypeAuditLog        = "audit:log"        // 审计日志
)

var (
	client *asynq.Client
	server *asynq.Server
)

// Handler 任务处理器签名
type Handler func(ctx context.Context, payload []byte) error

// InitQueue 初始化队列客户端和服务端（基于 Redis）。
// 若 Redis 未启用，本函数不应被调用（调用方先判断 cache 是否可用）。
func InitQueue(redisAddr string) error {
	if !cache.IsRedisEnabled() {
		// Redis 没连上就别创建 asynq client/server：否则 worker 会在后台无限重连、
		// 刷屏 "failed to dial"，而每次入队都要先付一次连接失败的时间。
		return fmt.Errorf("队列未启动：Redis 不可用")
	}
	conn := cache.CurrentRedisConn()
	if redisAddr != "" {
		conn.Addr = redisAddr
	}
	if conn.Addr == "" {
		return fmt.Errorf("队列初始化失败：REDIS_ADDR 为空")
	}
	// 连接参数必须与缓存共用（密码/DB/TLS）：此前只传 Addr，
	// 配了密码的 Redis 上每一次入队都会失败，而调用方只看到"队列未就绪"的兜底路径。
	opts := asynq.RedisClientOpt{Addr: conn.Addr, Password: conn.Password, DB: conn.DB, TLSConfig: conn.TLS}
	client = asynq.NewClient(opts)
	server = asynq.NewServer(opts, asynq.Config{
		Concurrency: 10,
		Queues: map[string]int{
			"critical": 6, // 高优先级（如邮件验证码）
			"default":  3, // 默认
			"low":      1, // 低优先级（如审计日志）
		},
		// 失败重试：间隔递增（1min、2min、4min...）。重试次数在入队时用 asynq.MaxRetry(5)
		// 指定（asynq 服务端默认 25 次，与"最多 5 次"的预期不符，见 EnqueueWithRetry）。
		RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
			return time.Duration(1<<uint(n-1)) * time.Minute
		},
		// 失败任务打日志，避免"邮件没发出去"完全无声
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, t *asynq.Task, err error) {
			defaultQueueLogger.Printf("任务失败: type=%s payload=%s err=%v", t.Type(), string(t.Payload()), err)
		}),
	})
	return nil
}

// defaultQueueLogger 队列侧日志（不引入循环依赖，直接标准库输出）
var defaultQueueLogger = log.New(os.Stdout, "[queue] ", log.LstdFlags)

// defaultMaxRetry 默认重试次数（覆盖 asynq 服务端的 25 次默认值）
const defaultMaxRetry = 5

// Enqueue 入队任务（JSON 序列化 payload）。Redis 不可用时返回 error，
// 调用方应 fallback 到同步执行，保证核心逻辑不因队列故障而丢失。
// 未显式指定重试次数时用 defaultMaxRetry。
func Enqueue(ctx context.Context, taskType string, payload interface{}, opts ...asynq.Option) error {
	if client == nil {
		return fmt.Errorf("队列客户端未初始化")
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化任务负载失败: %w", err)
	}
	task := asynq.NewTask(taskType, data)
	if !hasRetryOption(opts) {
		opts = append(opts, asynq.MaxRetry(defaultMaxRetry))
	}
	_, err = client.EnqueueContext(ctx, task, opts...)
	return err
}

// StartWorker 启动 worker 处理任务（阻塞，通常用 goroutine 调用）。
// handlers 为任务类型到处理器的映射。
var workerOnce sync.Once

// StartWorker 启动 worker（可安全重复调用：Redis 断线恢复后会被再次触发，靠 once 保证只起一份）。
func StartWorker(handlers map[string]Handler) error {
	if server == nil {
		return fmt.Errorf("队列服务端未初始化")
	}
	started := false
	workerOnce.Do(func() { started = true })
	if !started {
		return nil
	}
	mux := asynq.NewServeMux()
	for taskType, h := range handlers {
		handler := h
		mux.HandleFunc(taskType, func(ctx context.Context, t *asynq.Task) error {
			return handler(ctx, t.Payload())
		})
	}
	return server.Run(mux)
}

// hasRetryOption 调用方是否已显式指定重试次数
func hasRetryOption(opts []asynq.Option) bool {
	for _, opt := range opts {
		if opt.Type() == asynq.MaxRetryOpt {
			return true
		}
	}
	return false
}

// Close 关闭队列客户端连接
func Close() {
	if client != nil {
		client.Close()
	}
}

// IsReady 队列是否已初始化（Redis 可用）
func IsReady() bool {
	return client != nil && server != nil
}
