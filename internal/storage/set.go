package storage

import (
	"time"

	"kv-cache/internal/storage/types"
)

// ==================== Set 命令 ====================

// SAdd 添加成员到集合
// 返回值：成功添加的成员数量、error
func (s *MemoryStore) SAdd(key string, members ...string) (int, error) {
	set, isNew, oldV, err := s.getOrCreateSet(key)
	if err != nil {
		return 0, err
	}

	n := set.SAdd(members...)

	// 保留过期时间
	var ttl time.Duration
	if !isNew && oldV.ExpireAt != nil {
		ttl = time.Until(*oldV.ExpireAt)
		if ttl < 0 {
			ttl = 0
		}
	}

	s.Set(key, types.Value{Type: types.TypeSet, Data: set}, ttl)
	return n, nil
}

// SRem 从集合中移除成员
// 返回值：成功移除的成员数量、error
func (s *MemoryStore) SRem(key string, members ...string) (int, error) {
	set, oldV, exists, err := s.getSet(key)
	if err != nil || !exists {
		return 0, err
	}

	count := set.SRem(members...)

	// 如果 Set 为空，删除 key
	if set.SCard() == 0 {
		s.Delete(key)
		return count, nil
	}

	// 保存回去
	ttl := time.Duration(0)
	if oldV.ExpireAt != nil {
		ttl = time.Until(*oldV.ExpireAt)
		if ttl < 0 {
			ttl = 0
		}
	}
	s.Set(key, types.Value{Type: types.TypeSet, Data: set}, ttl)
	return count, nil
}

// SIsMember 判断成员是否在集合中
// 返回值：是否存在、error
func (s *MemoryStore) SIsMember(key, member string) (bool, error) {
	set, _, exists, err := s.getSet(key)
	if err != nil || !exists {
		return false, err
	}
	return set.SIsMember(member), nil
}

// SMembers 返回集合中的所有成员
// 返回值：成员列表、error
func (s *MemoryStore) SMembers(key string) ([]string, error) {
	set, _, exists, err := s.getSet(key)
	if err != nil || !exists {
		return []string{}, err
	}
	return set.SMembers(), nil
}

// SCard 返回集合的成员数量
// 返回值：数量、error
func (s *MemoryStore) SCard(key string) (int, error) {
	set, _, exists, err := s.getSet(key)
	if err != nil || !exists {
		return 0, err
	}
	return set.SCard(), nil
}

// SPop 随机弹出指定数量的成员
// 返回值：被弹出的成员列表、error
func (s *MemoryStore) SPop(key string, count int) ([]string, error) {
	set, oldV, exists, err := s.getSet(key)
	if err != nil || !exists {
		return []string{}, err
	}

	members := set.SPop(count)

	// 如果 Set 为空，删除 key
	if set.SCard() == 0 {
		s.Delete(key)
		return members, nil
	}

	// 保存回去
	ttl := time.Duration(0)
	if oldV.ExpireAt != nil {
		ttl = time.Until(*oldV.ExpireAt)
		if ttl < 0 {
			ttl = 0
		}
	}
	s.Set(key, types.Value{Type: types.TypeSet, Data: set}, ttl)
	return members, nil
}

// SUnion 返回多个集合的并集
// 返回值：并集成员列表、error
func (s *MemoryStore) SUnion(keys ...string) ([]string, error) {
	if len(keys) == 0 {
		return []string{}, nil
	}

	// 获取第一个集合
	set, _, exists, err := s.getSet(keys[0])
	if err != nil {
		return nil, err
	}
	if !exists {
		set = types.NewSet()
	}

	// 与其他集合求并集
	result := set
	for i := 1; i < len(keys); i++ {
		other, _, exists, err := s.getSet(keys[i])
		if err != nil {
			return nil, err
		}
		if exists {
			result = result.SUnion(other)
		}
	}

	return result.SMembers(), nil
}

// SInter 返回多个集合的交集
// 返回值：交集成员列表、error
func (s *MemoryStore) SInter(keys ...string) ([]string, error) {
	if len(keys) == 0 {
		return []string{}, nil
	}

	// 获取第一个集合
	set, _, exists, err := s.getSet(keys[0])
	if err != nil {
		return nil, err
	}
	if !exists {
		return []string{}, nil
	}

	// 与其他集合求交集
	result := set
	for i := 1; i < len(keys); i++ {
		other, _, exists, err := s.getSet(keys[i])
		if err != nil {
			return nil, err
		}
		if !exists {
			return []string{}, nil
		}
		result = result.SInter(other)
		if len(result) == 0 {
			break
		}
	}

	return result.SMembers(), nil
}

// SDiff 返回第一个集合与其他集合的差集
// 返回值：差集成员列表、error
func (s *MemoryStore) SDiff(keys ...string) ([]string, error) {
	if len(keys) == 0 {
		return []string{}, nil
	}

	// 获取第一个集合
	set, _, exists, err := s.getSet(keys[0])
	if err != nil {
		return nil, err
	}
	if !exists {
		return []string{}, nil
	}

	// 与其他集合求差集
	result := set
	for i := 1; i < len(keys); i++ {
		other, _, exists, err := s.getSet(keys[i])
		if err != nil {
			return nil, err
		}
		if exists {
			result = result.SDiff(other)
		}
	}

	return result.SMembers(), nil
}

// ==================== 辅助方法 ====================

// getOrCreateSet 获取或创建 Set
func (s *MemoryStore) getOrCreateSet(key string) (types.Set, bool, *types.Value, error) {
	v, exists := s.Get(key)
	if !exists {
		return types.NewSet(), true, nil, nil
	}
	if v.Type != types.TypeSet {
		return nil, false, nil, ErrWrongType
	}
	return v.Data.(types.Set), false, &v, nil
}

// getSet 获取 Set（只读）
func (s *MemoryStore) getSet(key string) (types.Set, *types.Value, bool, error) {
	v, exists := s.Get(key)
	if !exists {
		return nil, nil, false, nil
	}
	if v.Type != types.TypeSet {
		return nil, nil, false, ErrWrongType
	}
	return v.Data.(types.Set), &v, true, nil
}
