package storage

import (
	"errors"
	"time"

	"kv-cache/internal/storage/types"
)

// ==================== Hash 命令 ====================
// 这些方法是 MemoryStore 的方法，负责将 key 操作和 Hash 数据结构连接起来
// 职责：
//   1. 通过 Get/Set 操作 key
//   2. 类型检查（WRONGTYPE）
//   3. 自动创建默认值
//   4. 保留原有过期时间

// HSet 设置 Hash 的 field
// 返回值：1 表示新增 field，0 表示更新 field，error 表示类型错误
func (s *MemoryStore) HSet(key, field, value string) (int, error) {
	// 获取或创建 Hash
	h, isNew, oldV, err := s.getOrCreateHash(key)
	if err != nil {
		return 0, err
	}

	// 调用 hash.HSet(field, value) 修改
	n := h.HSet(field, value)

	// 调用 s.Set() 保存回去，保留原有过期时间（如果存在）
	var ttl time.Duration
	if !isNew && oldV.ExpireAt != nil {
		ttl = time.Until(*oldV.ExpireAt)
		if ttl < 0 {
			ttl = 0
		}
	}

	// 保存
	s.Set(key, types.Value{Type: types.TypeHash, Data: h}, ttl)
	return n, nil
}

// HGet 获取 Hash 的 field
// 返回值：value、是否存在、error
func (s *MemoryStore) HGet(key, field string) (string, bool, error) {
	h, _, exists, err := s.getHash(key)
	if err != nil || !exists {
		return "", false, err
	}
	val, ok := h.HGet(field)
	return val, ok, nil
}

// HGetAll 获取 Hash 的所有 field 和 value
// 返回值：Hash、是否存在、error
func (s *MemoryStore) HGetAll(key string) (types.Hash, bool, error) {
	// 获取 Value
	v, exists := s.Get(key)
	if !exists {
		return nil, false, nil
	}
	if v.Type != types.TypeHash {
		return nil, false, ErrWrongType
	}

	// 取出 Hash，返回副本（避免外部修改影响内部）
	h := v.Data.(types.Hash)
	copy := make(types.Hash, len(h))
	for k, v := range h {
		copy[k] = v
	}
	return copy, true, nil
}

// HDel 删除 Hash 的一个或多个 field
// 返回值：成功删除的 field 数量、error
func (s *MemoryStore) HDel(key string, fields ...string) (int, error) {
	// 获取 Value
	v, exists := s.Get(key)
	if !exists {
		return 0, nil
	}
	if v.Type != types.TypeHash {
		return 0, ErrWrongType
	}

	// 取出 Hash
	h := v.Data.(types.Hash)

	// 删除字段
	deletedCount := h.HDel(fields...)
	if h.HLen() == 0 {
		s.Delete(key)
		return deletedCount, nil
	}

	// 保存回去（v.Data 已经被 h 修改了，因为 map 是引用类型）
	// 注意：需要保留原有过期时间
	if deletedCount > 0 {
		ttl := time.Duration(0)
		if v.ExpireAt != nil {
			ttl = time.Until(*v.ExpireAt)
			if ttl < 0 {
				ttl = 0
			}
		}
		s.Set(key, types.Value{Type: types.TypeHash, Data: h}, ttl)
	}

	return deletedCount, nil
}

// HExists 判断 field 是否存在
// 返回值：是否存在、error
func (s *MemoryStore) HExists(key, field string) (bool, error) {
	h, _, exists, err := s.getHash(key)
	if err != nil || !exists {
		return false, err
	}
	return h.HExists(field), nil
}

// HLen 返回 field 数量
// 返回值：数量、error
func (s *MemoryStore) HLen(key string) (int, error) {
	h, _, exists, err := s.getHash(key)
	if err != nil || !exists {
		return 0, err
	}
	return h.HLen(), nil
}

// HKeys 返回所有 field
// 返回值：field 列表、error
func (s *MemoryStore) HKeys(key string) ([]string, error) {
	h, _, exists, err := s.getHash(key)
	if err != nil || !exists {
		return nil, err
	}
	return h.HKeys(), nil
}

// HVals 返回所有 value
// 返回值：value 列表、error
func (s *MemoryStore) HVals(key string) ([]string, error) {
	h, _, exists, err := s.getHash(key)
	if err != nil || !exists {
		return nil, err
	}
	return h.HVals(), nil
}

// getHash 辅助方法：获取 key 对应的 Hash（只读场景）
// 返回值：Hash、原 Value（用于保留过期时间）、是否存在、error
func (s *MemoryStore) getHash(key string) (types.Hash, *types.Value, bool, error) {
	v, exists := s.Get(key)
	if !exists {
		return nil, nil, false, nil
	}
	if v.Type != types.TypeHash {
		return nil, nil, false, ErrWrongType
	}
	return v.Data.(types.Hash), &v, true, nil
}

// getOrCreateHash 辅助方法：获取 key 对应的 Hash，如果不存在则创建
// 返回值：Hash、是否新建、原 Value（用于保留过期时间）、error
func (s *MemoryStore) getOrCreateHash(key string) (types.Hash, bool, *types.Value, error) {
	v, exists := s.Get(key)
	if !exists {
		return types.NewHash(), true, nil, nil
	}
	if v.Type != types.TypeHash {
		return nil, false, nil, ErrWrongType
	}
	h := v.Data.(types.Hash)
	return h, false, &v, nil
}

// ==================== 错误定义 ====================

// ErrWrongType 类型错误
var ErrWrongType = errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")

// 辅助函数：检查是否为 WRONGTYPE 错误
func IsWrongTypeErr(err error) bool {
	return err == ErrWrongType
}
