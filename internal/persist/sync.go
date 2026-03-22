package persist

import (
	"sync"
	"time"
)

// SyncPolicy AOF 同步策略
type SyncPolicy int

const (
	SyncAlways   SyncPolicy = iota // 每次写入都 fsync
	SyncEverySec                   // 每秒 fsync 一次
	SyncNo                         // 不主动 fsync，依赖 OS
)

// String 返回策略名称
func (s SyncPolicy) String() string {
	switch s {
	case SyncAlways:
		return "always"
	case SyncEverySec:
		return "everysec"
	case SyncNo:
		return "no"
	default:
		return "unknown"
	}
}

// ParseSyncPolicy 从字符串解析同步策略
func ParseSyncPolicy(s string) SyncPolicy {
	switch s {
	case "always":
		return SyncAlways
	case "everysec":
		return SyncEverySec
	case "no":
		return SyncNo
	default:
		return SyncEverySec // 默认使用 everysec
	}
}

// Syncer 管理 AOF 同步策略
type Syncer struct {
	mu       sync.RWMutex
	policy   SyncPolicy    // 当前同步策略
	needSync bool          // 是否需要 sync（everysec 模式用）
	ticker   *time.Ticker  // 定时器
	stopChan chan struct{} // 停止信号
	syncFunc func() error  // 实际的 sync 函数回调
	running  bool
}

// NewSyncer 创建同步管理器
func NewSyncer(syncFunc func() error) *Syncer {
	return &Syncer{
		policy:   SyncEverySec, // 默认策略
		syncFunc: syncFunc,
		stopChan: make(chan struct{}),
	}
}

// SetPolicy 设置同步策略
func (s *Syncer) SetPolicy(policy SyncPolicy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policy = policy
}

// GetPolicy 获取当前同步策略
func (s *Syncer) GetPolicy() SyncPolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.policy
}

// Start 启动后台同步协程（用于 everysec 模式）
func (s *Syncer) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running || s.policy != SyncEverySec {
		return
	}

	s.running = true
	s.ticker = time.NewTicker(time.Second)

	go s.run()
}

// Stop 停止后台同步协程
func (s *Syncer) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	select {
	case s.stopChan <- struct{}{}:
	default:
	}
}

// run 后台协程：每秒检查并执行 sync
func (s *Syncer) run() {
	for {
		select {
		case <-s.stopChan:
			s.mu.Lock()
			s.ticker.Stop()
			s.running = false
			s.mu.Unlock()
			return
		case <-s.ticker.C:
			s.doSync()
		}
	}
}

// doSync 执行实际的 sync 操作
func (s *Syncer) doSync() {
	s.mu.Lock()
	need := s.needSync
	s.needSync = false
	s.mu.Unlock()

	if need && s.syncFunc != nil {
		s.syncFunc()
	}
}

// AfterWrite 写入后根据策略执行相应操作
// 返回值：是否需要立即执行 sync（always 模式返回 true）
func (s *Syncer) AfterWrite() bool {
	s.mu.RLock()
	policy := s.policy
	s.mu.RUnlock()

	switch policy {
	case SyncAlways:
		return true // 调用者需要立即执行 sync
	case SyncEverySec:
		// 标记需要 sync，由后台协程处理
		s.mu.Lock()
		s.needSync = true
		s.mu.Unlock()
		return false
	case SyncNo:
		return false
	default:
		return false
	}
}

// NeedFlush 是否需要执行 Flush（always 和 everysec 需要 flush 到 OS 缓冲区）
func (s *Syncer) NeedFlush() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.policy == SyncAlways || s.policy == SyncEverySec
}
