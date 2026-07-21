// Package sse 通知系统实时层 进程内 ConnManager 单例 + Redis Pub/Sub 扇入
// 语义(N7):
//   - 只推信号不推明细 收到即回拉/list 保证漏推被下次拉取纠正
//   - 每个 SSE 连接一个缓冲 send channel 慢连接直接丢帧(drop) 不阻塞广播
//   - 多标签页多设备场景:同一 userId 可有多个 conn 逐 conn 推
package sse

import (
	"sync"
)

// Conn 表示一路 SSE 长连接 由 handler 侧构造 传给 ConnManager
// send 由 broadcaster 写 handler 主循环读;done 由 handler 用于识别 client 断开
type Conn struct {
	// Send 缓冲 channel handler 主循环 select 出帧写回 http.ResponseWriter
	// 缓冲用来吸收突发信号 满了直接 drop 保证广播不阻塞其它 conn
	Send chan []byte
	// Done 由 handler 侧用请求 ctx 派生 手动 close 或 ctx 取消都能识别断连
	Done <-chan struct{}
}

// DefaultSendBuffer 单 conn 缓冲长度 8 = 短时 8 个 push 都不丢
const DefaultSendBuffer = 8

// NewConn 建 send 缓冲 conn done 由外部传入(通常 r.Context().Done())
func NewConn(done <-chan struct{}) *Conn {
	return &Conn{
		Send: make(chan []byte, DefaultSendBuffer),
		Done: done,
	}
}

// ConnManager 进程内单例 userID → 该 user 所有活跃 conn 的集合
// RWMutex 读多写少(广播是读 增删是写)
type ConnManager struct {
	mu    sync.RWMutex
	conns map[int64]map[*Conn]struct{}
}

// NewConnManager 建单例 由 svc 持有
func NewConnManager() *ConnManager {
	return &ConnManager{conns: make(map[int64]map[*Conn]struct{})}
}

// Add 挂 conn 到 userID 下 一个 userID 可多 conn(多端多标签页)
func (m *ConnManager) Add(userID int64, c *Conn) {
	if userID <= 0 || c == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	set, ok := m.conns[userID]
	if !ok {
		set = make(map[*Conn]struct{}, 1)
		m.conns[userID] = set
	}
	set[c] = struct{}{}
}

// Remove 摘 conn 若 user 下已无 conn 则清空 map 项防泄漏
func (m *ConnManager) Remove(userID int64, c *Conn) {
	if userID <= 0 || c == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	set, ok := m.conns[userID]
	if !ok {
		return
	}
	delete(set, c)
	if len(set) == 0 {
		delete(m.conns, userID)
	}
}

// Broadcast 把 payload 逐条塞进 user 下所有 conn 的 send channel
// 慢连接(缓冲满)直接 drop 返回 dropped 数供观察
// 非阻塞语义保证一路慢客户端不拖住其它 client
func (m *ConnManager) Broadcast(userID int64, payload []byte) (delivered, dropped int) {
	if userID <= 0 || len(payload) == 0 {
		return 0, 0
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for c := range m.conns[userID] {
		select {
		case c.Send <- payload:
			delivered++
		default:
			dropped++
		}
	}
	return
}

// UserCount 观察用 当前挂了多少用户
func (m *ConnManager) UserCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.conns)
}

// ConnCount 观察用 某 user 当前几路连
func (m *ConnManager) ConnCount(userID int64) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.conns[userID])
}
