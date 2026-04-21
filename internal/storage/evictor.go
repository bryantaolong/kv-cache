package storage

import (
	"math/rand"
	"strings"
	"sync"
	"time"

	"kv-cache/internal/storage/types"
)

// EvictionPolicy 淘汰策略
type EvictionPolicy int

const (
	EvictNoEviction EvictionPolicy = iota // 不淘汰
	EvictLRU                              // LRU
	EvictRandom                           // 随机
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
		evictionPolicy: EvictLRU, // 默认 LRU
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

// ParseEvictionPolicy 从字符串解析淘汰策略
func ParseEvictionPolicy(s string) EvictionPolicy {
	switch strings.ToLower(s) {
	case "lru":
		return EvictLRU
	case "random":
		return EvictRandom
	default:
		return EvictLRU // 默认LRU
	}
}

// GetEvictionPolicy 获取淘汰策略名称
func (e *Evictor) GetEvictionPolicy() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	switch e.evictionPolicy {
	case EvictNoEviction:
		return "no-eviction"
	case EvictLRU:
		return "lru"
	case EvictRandom:
		return "random"
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

	if maxMemory <= 0 || policy == EvictNoEviction {
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

	// 先清理过期键
	e.cleanup(data)

	// 如果清理后数据为空，返回 false
	if len(data) == 0 {
		return false
	}

	e.mu.RLock()
	policy := e.evictionPolicy
	e.mu.RUnlock()

	var candidates []string
	switch policy {
	case EvictLRU:
		// 收集所有键（清理后剩余的键）
		for key := range data {
			candidates = append(candidates, key)
		}

		if len(candidates) == 0 {
			return false
		}

		// 简化：随机从候选中选
		if len(candidates) > 3 {
			candidates = candidates[:3]
		}
		idx := rand.Intn(len(candidates))
		key := candidates[idx]
		delete(data, key)

	case EvictRandom:
		// 收集所有键
		for key := range data {
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

// cleanup 清理过期键
func (e *Evictor) cleanup(data map[string]*types.Value) {
	now := time.Now()
	expiredKeys := make([]string, 0)

	for key, val := range data {
		if val != nil && val.ExpireAt != nil && now.After(*val.ExpireAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(data, key)
	}
}
