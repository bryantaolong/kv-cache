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
	maxMemory      int64          // 最大内存限制（字节）
	evictionPolicy EvictionPolicy // 淘汰策略
}

// NewEvictor 创建淘汰器
func NewEvictor() *Evictor {
	return &Evictor{
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

// SelectEvictionTargets 估算内存并在超过阈值时返回应被淘汰的 key 列表（不修改原 map）。
func (e *Evictor) SelectEvictionTargets(data map[string]*types.Value) []string {
	e.mu.RLock()
	maxMemory := e.maxMemory
	policy := e.evictionPolicy
	e.mu.RUnlock()

	if maxMemory <= 0 || policy == EvictNoEviction || len(data) == 0 {
		return nil
	}

	// 在本地副本上做批次选择，避免同一轮重复选中同一个 key。
	snapshot := make(map[string]*types.Value, len(data))
	for k, v := range data {
		snapshot[k] = v
	}

	currentUsage := e.estimateUsageOn(snapshot)
	if currentUsage < maxMemory {
		return nil
	}

	targetUsage := maxMemory * 75 / 100
	var targets []string
	for currentUsage > targetUsage {
		var target string
		switch policy {
		case EvictLRU:
			target = e.selectLRU(snapshot)
		case EvictRandom:
			target = e.selectRandom(snapshot)
		}
		if target == "" {
			break
		}
		targets = append(targets, target)
		delete(snapshot, target)
		if val, ok := data[target]; ok && val != nil {
			currentUsage -= e.estimateValueUsage(val)
		}
	}
	return targets
}

// estimateValueUsage 估算单个 Value 的内存占用。
func (e *Evictor) estimateValueUsage(val *types.Value) int64 {
	if val == nil {
		return 0
	}
	var n int64
	switch v := val.Data.(type) {
	case string:
		n += int64(len(v))
	case map[string]string:
		for field, value := range v {
			n += int64(len(field) + len(value))
		}
	case []string:
		for _, item := range v {
			n += int64(len(item))
		}
	case types.Set:
		for m := range v {
			n += int64(len(m))
		}
	case *types.ZSet:
		for _, member := range v.Members() {
			n += int64(len(member))
		}
	}
	return n
}

// estimateUsageOn 根据传入快照估算内存使用
func (e *Evictor) estimateUsageOn(data map[string]*types.Value) int64 {
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

// selectLRU 选择最久未访问的 key
func (e *Evictor) selectLRU(data map[string]*types.Value) string {
	var (
		oldestKey string
		oldest    time.Time
		found     bool
	)
	for key, val := range data {
		if val == nil {
			continue
		}
		if !found || val.AccessedAt.Before(oldest) {
			oldestKey = key
			oldest = val.AccessedAt
			found = true
		}
	}
	if !found {
		return ""
	}
	return oldestKey
}

// selectRandom 随机选择一个 key
func (e *Evictor) selectRandom(data map[string]*types.Value) string {
	candidates := make([]string, 0, len(data))
	for key := range data {
		candidates = append(candidates, key)
	}
	if len(candidates) == 0 {
		return ""
	}
	idx := rand.Intn(len(candidates))
	return candidates[idx]
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
