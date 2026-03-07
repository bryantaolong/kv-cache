package store

import (
	"sync"
	"time"

	"kv-cache/internal/storage/types"
)

// MemoryStore 是内存存储的实现
type MemoryStore struct {
	data map[string]*types.Value
	mu   sync.RWMutex
}

// NewMemoryStore 创建一个新的存储实例
func NewMemoryStore() *MemoryStore {
	// 初始化 map 和 mutex
	return &MemoryStore{
		data: make(map[string]*types.Value),
		mu:   sync.RWMutex{},
	}
}

// Get 获取值（注意：要检查过期）
func (s *MemoryStore) Get(key string) (types.Value, bool) {
	s.mu.RLock()
	val, exists := s.data[key]
	s.mu.RUnlock() // 先解锁

	if !exists {
		return types.Value{}, false
	}

	// 检查过期
	if val.ExpireAt != nil && time.Now().After(*val.ExpireAt) {
		s.mu.Lock()
		delete(s.data, key)
		s.mu.Unlock()
		return types.Value{}, false
	}

	return *val, true
}

// Set 设置值
func (s *MemoryStore) Set(key string, value types.Value, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ttl > 0 {
		expireAt := time.Now().Add(ttl)
		value.ExpireAt = &expireAt
	}

	s.data[key] = &value
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

	// 第一遍：只读，收集信息
	now := time.Now()
	expiredKeys := make([]string, 0)
	validKeys := make([]string, 0, len(s.data))

	for key, val := range s.data {
		if val != nil && val.ExpireAt != nil && now.After(*val.ExpireAt) {
			expiredKeys = append(expiredKeys, key) // 记录过期键
		} else {
			validKeys = append(validKeys, key) // 记录有效键
		}
	}
	s.mu.RUnlock() // 读锁释放

	// 第二遍：加写锁删除过期键
	if len(expiredKeys) > 0 {
		s.mu.Lock()
		for _, key := range expiredKeys {
			// 双重检查：确认仍然过期（避免重复删除已被删掉的键）
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
		return -2 // 不存在
	}
	if val.ExpireAt == nil {
		return -1 // 未设置过期
	}
	return time.Until(*val.ExpireAt)
}

// DBSize 返回键数量（含过期）
func (s *MemoryStore) DBSize() int {
	return len(s.Keys())
}
