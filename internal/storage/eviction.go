package storage

import (
	"math/rand"
	"sync"

	"kv-cache/internal/storage/types"
)

// EvictionPolicy 淘汰策略
type EvictionPolicy int

const (
	EvictNoeviction     EvictionPolicy = iota // 不淘汰
	EvictAllKeysLRU                           // 所有键 LRU
	EvictVolatileLRU                          // 带过期键 LRU
	EvictAllKeysRandom                        // 所有键随机
	EvictVolatileRandom                       // 带过期键随机
)

// Evictor 淘汰器
type Evictor struct {
	mu             sync.RWMutex
	maxMemory      int64                          // 最大内存限制（字节）
	evictionPolicy EvictionPolicy                 // 淘汰策略
	data           func() map[string]*types.Value // 获取数据的回调
}

// NewEvictor 创建淘汰器
func NewEvictor(dataFunc func() map[string]*types.Value) *Evictor {
	return &Evictor{
		data:           dataFunc,
		evictionPolicy: EvictNoeviction,
	}
}

// SetMaxMemory 设置最大内存限制
func (e *Evictor) SetMaxMemory(maxBytes int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.maxMemory = maxBytes
}

// SetEvictionPolicy 设置淘汰策略
func (e *Evictor) SetEvictionPolicy(policy EvictionPolicy) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.evictionPolicy = policy
}

// GetEvictionPolicy 获取淘汰策略名称
func (e *Evictor) GetEvictionPolicy() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	switch e.evictionPolicy {
	case EvictNoeviction:
		return "noeviction"
	case EvictAllKeysLRU:
		return "allkeys-lru"
	case EvictVolatileLRU:
		return "volatile-lru"
	case EvictAllKeysRandom:
		return "allkeys-random"
	case EvictVolatileRandom:
		return "volatile-random"
	default:
		return "unknown"
	}
}

// EvictIfNeeded 检查是否需要淘汰数据
func (e *Evictor) EvictIfNeeded() {
	e.mu.RLock()
	maxMemory := e.maxMemory
	policy := e.evictionPolicy
	e.mu.RUnlock()

	if maxMemory <= 0 || policy == EvictNoeviction {
		return
	}

	// 估算当前使用
	currentUsage := e.estimateUsage()
	if currentUsage < maxMemory {
		return
	}

	// 需要淘汰到 75% 以下
	targetUsage := maxMemory * 75 / 100

	for currentUsage > targetUsage {
		evicted := e.doEvict()
		if !evicted {
			break
		}
		currentUsage = e.estimateUsage()
	}
}

// estimateUsage 估算内存使用
func (e *Evictor) estimateUsage() int64 {
	data := e.data()
	if data == nil {
		return 0
	}

	var total int64
	for _, val := range data {
		if val == nil {
			continue
		}
		switch v := val.Data.(type) {
		case string:
			total += int64(len(v))
		case map[string]string:
			for field, value := range v {
				total += int64(len(field) + len(value))
			}
		case []string:
			for _, item := range v {
				total += int64(len(item))
			}
		case types.Set:
			for m := range v {
				total += int64(len(m))
			}
		case *types.ZSet:
			members := v.Members()
			for _, member := range members {
				total += int64(len(member))
			}
		}
	}
	return total
}

// doEvict 执行一次淘汰
func (e *Evictor) doEvict() bool {
	data := e.data()
	if len(data) == 0 {
		return false
	}

	e.mu.RLock()
	policy := e.evictionPolicy
	e.mu.RUnlock()

	var candidates []string
	switch policy {
	case EvictAllKeysLRU, EvictVolatileLRU:
		for key := range data {
			if policy == EvictVolatileLRU {
				if val, ok := data[key]; !ok || val.ExpireAt == nil {
					continue
				}
			}
			candidates = append(candidates, key)
		}

		if len(candidates) == 0 {
			return false
		}

		// 随机淘汰
		if len(candidates) > 3 {
			candidates = candidates[:3]
		}
		idx := rand.Intn(len(candidates))
		key := candidates[idx]
		delete(data, key)

	case EvictAllKeysRandom, EvictVolatileRandom:
		for key := range data {
			if policy == EvictVolatileRandom {
				if val, ok := data[key]; !ok || val.ExpireAt == nil {
					continue
				}
			}
			candidates = append(candidates, key)
		}

		if len(candidates) == 0 {
			return false
		}

		idx := rand.Intn(len(candidates))
		key := candidates[idx]
		delete(data, key)

	default:
		return false
	}

	return true
}
