package storage

import (
	"sync"
	"time"

	"kv-cache/internal/storage/types"
)

// MemoryStore 是内存存储的实现
type MemoryStore struct {
	data    map[string]*types.Value
	mu      sync.RWMutex
	evictor *Evictor
	stopGC  chan struct{}
}

// NewMemoryStore 创建一个新的存储实例
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data:    make(map[string]*types.Value),
		evictor: NewEvictor(),
		stopGC:  make(chan struct{}),
	}
}

// Get 获取值（注意：要检查过期）
func (s *MemoryStore) Get(key string) (types.Value, bool) {
	s.mu.RLock()
	val, exists := s.data[key]
	if !exists || val == nil {
		s.mu.RUnlock()
		return types.Value{}, false
	}
	ret := *val
	expired := ret.ExpireAt != nil && time.Now().After(*ret.ExpireAt)
	s.mu.RUnlock()

	if expired {
		s.mu.Lock()
		delete(s.data, key)
		s.mu.Unlock()
		return types.Value{}, false
	}

	ret.AccessedAt = time.Now()
	s.mu.Lock()
	s.data[key] = &ret
	s.mu.Unlock()

	return ret, true
}

// Set 设置值
func (s *MemoryStore) Set(key string, value types.Value, ttl time.Duration) error {
	s.mu.Lock()
	if ttl > 0 {
		expireAt := time.Now().Add(ttl)
		value.ExpireAt = &expireAt
	}
	s.data[key] = &value
	s.mu.Unlock()

	// 先让 evictor 只返回候选 key，不直接删 map；再由 store 在锁内批量删除。
	s.mu.RLock()
	targets := s.evictor.SelectEvictionTargets(s.data)
	s.mu.RUnlock()

	if len(targets) > 0 {
		s.mu.Lock()
		for _, key := range targets {
			delete(s.data, key)
		}
		s.mu.Unlock()
	}

	return nil
}

// Delete 删除键
func (s *MemoryStore) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)
	return true
}

// Exists 检查键是否存在
func (s *MemoryStore) Exists(key string) bool {
	_, exists := s.Get(key)
	return exists
}

// Keys 返回所有未过期的键
func (s *MemoryStore) Keys() []string {
	s.mu.RLock()

	now := time.Now()
	expiredKeys := make([]string, 0)
	validKeys := make([]string, 0, len(s.data))

	for key, val := range s.data {
		if val != nil && val.ExpireAt != nil && now.After(*val.ExpireAt) {
			expiredKeys = append(expiredKeys, key)
		} else {
			validKeys = append(validKeys, key)
		}
	}
	s.mu.RUnlock()

	if len(expiredKeys) > 0 {
		s.mu.Lock()
		for _, key := range expiredKeys {
			if val, ok := s.data[key]; ok && val != nil &&
				val.ExpireAt != nil && val.ExpireAt.Before(now) {
				delete(s.data, key)
			}
		}
		s.mu.Unlock()
	}

	return validKeys
}

// Flush 清空数据
func (s *MemoryStore) Flush() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[string]*types.Value)
}

// Expire 给键设置过期时间
func (s *MemoryStore) Expire(key string, ttl time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	val, exists := s.data[key]
	if !exists || val == nil {
		return false
	}

	expireAt := time.Now().Add(ttl)
	val.ExpireAt = &expireAt

	return true
}

// TTL 返回剩余存活时间
func (s *MemoryStore) TTL(key string) time.Duration {
	val, exists := s.Get(key)
	if !exists {
		return -2
	}
	if val.ExpireAt == nil {
		return -1
	}
	return time.Until(*val.ExpireAt)
}

// DBSize 返回键数量（含过期）
func (s *MemoryStore) DBSize() int {
	return len(s.Keys())
}

// StartGC 启动后台 Goroutine 定期清理过期键
func (s *MemoryStore) StartGC(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-s.stopGC:
				return
			case <-ticker.C:
				s.cleanupExpired()
			}
		}
	}()
}

// StopGC 停止后台 Goroutine
func (s *MemoryStore) StopGC() {
	select {
	case s.stopGC <- struct{}{}:
	default:
	}
}

func (s *MemoryStore) cleanupExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for key, val := range s.data {
		if val != nil && val.ExpireAt != nil && now.After(*val.ExpireAt) {
			delete(s.data, key)
		}
	}
}

// SetMaxMemory 设置最大内存限制
func (s *MemoryStore) SetMaxMemory(maxBytes int64) {
	s.evictor.SetMaxMemory(maxBytes)
}

// SetEvictionPolicy 设置淘汰策略
func (s *MemoryStore) SetEvictionPolicy(policy EvictionPolicy) {
	s.evictor.SetEvictionPolicy(policy)
}

// GetEvictionPolicy 获取淘汰策略名称
func (s *MemoryStore) GetEvictionPolicy() string {
	return s.evictor.GetEvictionPolicy()
}
