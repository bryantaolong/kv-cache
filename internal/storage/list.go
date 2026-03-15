package storage

import (
	"time"

	"kv-cache/internal/storage/types"
)

// ==================== List 命令 ====================

// LPush 从左边插入一个或多个元素
// 返回值：插入后列表的长度、error
func (s *MemoryStore) LPush(key string, values ...string) (int, error) {
	list, isNew, oldV, err := s.getOrCreateList(key)
	if err != nil {
		return 0, err
	}

	n := list.LPush(values...)

	// 保留过期时间
	var ttl time.Duration
	if !isNew && oldV.ExpireAt != nil {
		ttl = time.Until(*oldV.ExpireAt)
		if ttl < 0 {
			ttl = 0
		}
	}

	s.Set(key, types.Value{Type: types.TypeList, Data: list}, ttl)
	return n, nil
}

// RPush 从右边插入一个或多个元素
// 返回值：插入后列表的长度、error
func (s *MemoryStore) RPush(key string, values ...string) (int, error) {
	list, isNew, oldV, err := s.getOrCreateList(key)
	if err != nil {
		return 0, err
	}

	n := list.RPush(values...)

	// 保留过期时间
	var ttl time.Duration
	if !isNew && oldV.ExpireAt != nil {
		ttl = time.Until(*oldV.ExpireAt)
		if ttl < 0 {
			ttl = 0
		}
	}

	s.Set(key, types.Value{Type: types.TypeList, Data: list}, ttl)
	return n, nil
}

// LPop 从左边弹出一个元素
// 返回值：弹出的元素、是否存在、error
func (s *MemoryStore) LPop(key string) (string, bool, error) {
	list, oldV, exists, err := s.getList(key)
	if err != nil || !exists {
		return "", false, err
	}

	val, ok := list.LPop()
	if !ok {
		return "", false, nil
	}

	// 如果列表为空，删除 key
	if list.LLen() == 0 {
		s.Delete(key)
		return val, true, nil
	}

	// 保存回去，保留过期时间
	ttl := time.Duration(0)
	if oldV.ExpireAt != nil {
		ttl = time.Until(*oldV.ExpireAt)
		if ttl < 0 {
			ttl = 0
		}
	}
	s.Set(key, types.Value{Type: types.TypeList, Data: list}, ttl)
	return val, true, nil
}

// RPop 从右边弹出一个元素
// 返回值：弹出的元素、是否存在、error
func (s *MemoryStore) RPop(key string) (string, bool, error) {
	list, oldV, exists, err := s.getList(key)
	if err != nil || !exists {
		return "", false, err
	}

	val, ok := list.RPop()
	if !ok {
		return "", false, nil
	}

	// 如果列表为空，删除 key
	if list.LLen() == 0 {
		s.Delete(key)
		return val, true, nil
	}

	// 保存回去，保留过期时间
	ttl := time.Duration(0)
	if oldV.ExpireAt != nil {
		ttl = time.Until(*oldV.ExpireAt)
		if ttl < 0 {
			ttl = 0
		}
	}
	s.Set(key, types.Value{Type: types.TypeList, Data: list}, ttl)
	return val, true, nil
}

// LRange 返回指定区间的元素
// 返回值：元素列表、error
func (s *MemoryStore) LRange(key string, start, stop int) ([]string, error) {
	list, _, exists, err := s.getList(key)
	if err != nil || !exists {
		return []string{}, err
	}
	return list.LRange(start, stop), nil
}

// LLen 返回列表长度
// 返回值：长度、error
func (s *MemoryStore) LLen(key string) (int, error) {
	list, _, exists, err := s.getList(key)
	if err != nil || !exists {
		return 0, err
	}
	return list.LLen(), nil
}

// LIndex 返回指定索引的元素
// 返回值：元素、是否存在、error
func (s *MemoryStore) LIndex(key string, index int) (string, bool, error) {
	list, _, exists, err := s.getList(key)
	if err != nil || !exists {
		return "", false, err
	}
	val, ok := list.LIndex(index)
	return val, ok, nil
}

// ==================== 辅助方法 ====================

// getList 获取 List（只读）
func (s *MemoryStore) getList(key string) (types.List, *types.Value, bool, error) {
	v, exists := s.Get(key)
	if !exists {
		return nil, nil, false, nil
	}
	if v.Type != types.TypeList {
		return nil, nil, false, ErrWrongType
	}
	return v.Data.(types.List), &v, true, nil
}

// getOrCreateList 获取或创建 List
func (s *MemoryStore) getOrCreateList(key string) (types.List, bool, *types.Value, error) {
	v, exists := s.Get(key)
	if !exists {
		return types.NewList(), true, nil, nil
	}
	if v.Type != types.TypeList {
		return nil, false, nil, ErrWrongType
	}
	return v.Data.(types.List), false, &v, nil
}
