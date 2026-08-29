package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
	if redisAddr == "" {
		return fmt.Errorf("队列初始化失败：REDIS_ADDR 为空")
	}
	opts := asynq.RedisClientOpt{Addr: redisAddr}
	client = asynq.NewClient(opts)
	server = asynq.NewServer(opts, asynq.Config{
		Concurrency: 10,
		Queues: map[string]int{
			"critical": 6, // 高优先级（如邮件验证码）
			"default":  3, // 默认
			"low":      1, // 低优先级（如审计日志）
		},
		// 失败重试：间隔递增（1min、2min、4min...），最多 5 次
		RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
			return time.Duration(1<<uint(n-1)) * time.Minute
		},
	})
	return nil
}

// Enqueue 入队任务（JSON 序列化 payload）。Redis 不可用时返回 error，
// 调用方应 fallback 到同步执行，保证核心逻辑不因队列故障而丢失。
func Enqueue(ctx context.Context, taskType string, payload interface{}, opts ...asynq.Option) error {
	if client == nil {
		return fmt.Errorf("队列客户端未初始化")
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化任务负载失败: %w", err)
	}
	task := asynq.NewTask(taskType, data)
	_, err = client.EnqueueContext(ctx, task, opts...)
	return err
}

// EnqueueIn 入队延迟任务（delay 后执行）
func EnqueueIn(ctx context.Context, delay time.Duration, taskType string, payload interface{}) error {
	return Enqueue(ctx, taskType, payload, asynq.ProcessIn(delay))
}

// StartWorker 启动 worker 处理任务（阻塞，通常用 goroutine 调用）。
// handlers 为任务类型到处理器的映射。
func StartWorker(handlers map[string]Handler) error {
	if server == nil {
		return fmt.Errorf("队列服务端未初始化")
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
