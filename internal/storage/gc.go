package storage

import (
	"sync"
	"time"

	"kv-cache/internal/storage/types"
)

// GC 过期键清理器
type GC struct {
	mu       sync.RWMutex
	running  bool
	stopChan chan struct{}
	data     func() map[string]*types.Value // 获取数据的回调
}

// NewGC 创建 GC 实例
func NewGC(dataFunc func() map[string]*types.Value) *GC {
	return &GC{
		data:     dataFunc,
		running:  false,
		stopChan: make(chan struct{}),
	}
}

// Start 启动后台 Goroutine 定期清理过期键
func (g *GC) Start(interval time.Duration) {
	g.mu.Lock()
	if g.running {
		g.mu.Unlock()
		return
	}
	g.running = true
	g.mu.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-g.stopChan:
				g.mu.Lock()
				g.running = false
				g.mu.Unlock()
				return
			case <-ticker.C:
				g.cleanup()
			}
		}
	}()
}

// Stop 停止后台 Goroutine
func (g *GC) Stop() {
	select {
	case g.stopChan <- struct{}{}:
	default:
	}
}

// cleanup 清理过期键
func (g *GC) cleanup() {
	data := g.data()
	if data == nil {
		return
	}

	now := time.Now()
	expiredKeys := make([]string, 0)

	for key, val := range data {
		if val != nil && val.ExpireAt != nil && now.After(*val.ExpireAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	if len(expiredKeys) > 0 {
		for _, key := range expiredKeys {
			if val, ok := data[key]; ok && val != nil &&
				val.ExpireAt != nil && val.ExpireAt.Before(now) {
				delete(data, key)
			}
		}
	}
}
