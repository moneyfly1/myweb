package config_update

import (
	"encoding/json"
	"sync"
)

// SSEManager 管理所有 SSE 连接和日志广播
type SSEManager struct {
	clients      map[chan []byte]bool
	mutex        sync.RWMutex
	historyLogs  []map[string]interface{}
	historyMutex sync.RWMutex
}

// NewSSEManager 创建新的 SSE 管理器
func NewSSEManager() *SSEManager {
	return &SSEManager{
		clients:     make(map[chan []byte]bool),
		historyLogs: make([]map[string]interface{}, 0, 500),
	}
}

// AddClient 添加新的 SSE 客户端连接
func (m *SSEManager) AddClient(ch chan []byte) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.clients[ch] = true
}

// RemoveClient 移除 SSE 客户端连接
func (m *SSEManager) RemoveClient(ch chan []byte) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if _, ok := m.clients[ch]; ok {
		delete(m.clients, ch)
		close(ch)
	}
}

// Broadcast 向所有连接的客户端广播日志
func (m *SSEManager) Broadcast(logEntry map[string]interface{}) {
	// 保存到历史记录
	m.historyMutex.Lock()
	m.historyLogs = append(m.historyLogs, logEntry)
	if len(m.historyLogs) > 500 {
		m.historyLogs = m.historyLogs[len(m.historyLogs)-500:]
	}
	m.historyMutex.Unlock()

	// 序列化日志
	data, err := json.Marshal(logEntry)
	if err != nil {
		return
	}

	// 广播给所有客户端
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for clientChan := range m.clients {
		select {
		case clientChan <- data:
		default:
			// 如果通道已满，跳过该客户端（避免阻塞）
		}
	}
}

// GetHistoryLogs 获取历史日志（用于新连接时发送）
func (m *SSEManager) GetHistoryLogs() []map[string]interface{} {
	m.historyMutex.RLock()
	defer m.historyMutex.RUnlock()

	// 返回副本，避免并发问题
	logs := make([]map[string]interface{}, len(m.historyLogs))
	copy(logs, m.historyLogs)
	return logs
}

// ClearHistory 清空历史日志
func (m *SSEManager) ClearHistory() {
	m.historyMutex.Lock()
	defer m.historyMutex.Unlock()
	m.historyLogs = make([]map[string]interface{}, 0, 500)
}

// ClientCount 返回当前连接的客户端数量
func (m *SSEManager) ClientCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.clients)
}
