package store

import (
	"time"

	"kv-cache/internal/storage/types"
)

// ==================== ZSet 命令 ====================

// ZAdd 添加成员
// 返回值：新增成员数量、error
func (s *MemoryStore) ZAdd(key string, score float64, member string) (int, error) {
	zset, isNew, oldV, err := s.getOrCreateZSet(key)
	if err != nil {
		return 0, err
	}

	n := zset.ZAdd(score, member)

	// 保留过期时间
	var ttl time.Duration
	if !isNew && oldV.ExpireAt != nil {
		ttl = time.Until(*oldV.ExpireAt)
		if ttl < 0 {
			ttl = 0
		}
	}

	s.Set(key, types.Value{Type: types.TypeZSet, Data: zset}, ttl)
	return n, nil
}

// ZRem 移除成员
// 返回值：成功移除的数量、error
func (s *MemoryStore) ZRem(key string, members ...string) (int, error) {
	zset, oldV, exists, err := s.getZSet(key)
	if err != nil || !exists {
		return 0, err
	}

	count := zset.ZRem(members...)

	// 如果 ZSet 为空，删除 key
	if zset.ZCard() == 0 {
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
	s.Set(key, types.Value{Type: types.TypeZSet, Data: zset}, ttl)
	return count, nil
}

// ZScore 返回成员的 score
// 返回值：score、是否存在、error
func (s *MemoryStore) ZScore(key, member string) (float64, bool, error) {
	zset, _, exists, err := s.getZSet(key)
	if err != nil || !exists {
		return 0, false, err
	}
	score, ok := zset.ZScore(member)
	return score, ok, nil
}

// ZRank 返回成员的排名（升序，从0开始）
// 返回值：排名、是否存在、error
func (s *MemoryStore) ZRank(key, member string) (int, bool, error) {
	zset, _, exists, err := s.getZSet(key)
	if err != nil || !exists {
		return 0, false, err
	}
	rank, ok := zset.ZRank(member)
	return rank, ok, nil
}

// ZRevRank 返回成员的排名（降序，从0开始）
// 返回值：排名、是否存在、error
func (s *MemoryStore) ZRevRank(key, member string) (int, bool, error) {
	zset, _, exists, err := s.getZSet(key)
	if err != nil || !exists {
		return 0, false, err
	}
	rank, ok := zset.ZRevRank(member)
	return rank, ok, nil
}

// ZRange 按排名范围返回成员（升序）
// 返回值：成员列表、error
func (s *MemoryStore) ZRange(key string, start, stop int) ([]types.ZSetMember, error) {
	zset, _, exists, err := s.getZSet(key)
	if err != nil || !exists {
		return []types.ZSetMember{}, err
	}
	return zset.ZRange(start, stop), nil
}

// ZRevRange 按排名范围返回成员（降序）
// 返回值：成员列表、error
func (s *MemoryStore) ZRevRange(key string, start, stop int) ([]types.ZSetMember, error) {
	zset, _, exists, err := s.getZSet(key)
	if err != nil || !exists {
		return []types.ZSetMember{}, err
	}
	return zset.ZRevRange(start, stop), nil
}

// ZRangeByScore 按 score 范围返回成员
// 返回值：成员列表、error
func (s *MemoryStore) ZRangeByScore(key string, min, max float64) ([]types.ZSetMember, error) {
	zset, _, exists, err := s.getZSet(key)
	if err != nil || !exists {
		return []types.ZSetMember{}, err
	}
	return zset.ZRangeByScore(min, max), nil
}

// ZCard 返回成员数量
// 返回值：数量、error
func (s *MemoryStore) ZCard(key string) (int, error) {
	zset, _, exists, err := s.getZSet(key)
	if err != nil || !exists {
		return 0, err
	}
	return zset.ZCard(), nil
}

// ZCount 返回 score 在范围内的成员数量
// 返回值：数量、error
func (s *MemoryStore) ZCount(key string, min, max float64) (int, error) {
	zset, _, exists, err := s.getZSet(key)
	if err != nil || !exists {
		return 0, err
	}
	return zset.ZCount(min, max), nil
}

// ZIncrBy 将成员的 score 增加 delta
// 返回值：新的 score、error
func (s *MemoryStore) ZIncrBy(key string, delta float64, member string) (float64, error) {
	zset, isNew, oldV, err := s.getOrCreateZSet(key)
	if err != nil {
		return 0, err
	}

	newScore, err := zset.ZIncrBy(member, delta)
	if err != nil {
		return 0, err
	}

	// 保留过期时间
	var ttl time.Duration
	if !isNew && oldV.ExpireAt != nil {
		ttl = time.Until(*oldV.ExpireAt)
		if ttl < 0 {
			ttl = 0
		}
	}

	s.Set(key, types.Value{Type: types.TypeZSet, Data: zset}, ttl)
	return newScore, nil
}

// ==================== 辅助方法 ====================

// getZSet 获取 ZSet（只读）
func (s *MemoryStore) getZSet(key string) (*types.ZSet, *types.Value, bool, error) {
	v, exists := s.Get(key)
	if !exists {
		return nil, nil, false, nil
	}
	if v.Type != types.TypeZSet {
		return nil, nil, false, ErrWrongType
	}
	return v.Data.(*types.ZSet), &v, true, nil
}

// getOrCreateZSet 获取或创建 ZSet
func (s *MemoryStore) getOrCreateZSet(key string) (*types.ZSet, bool, *types.Value, error) {
	v, exists := s.Get(key)
	if !exists {
		return types.NewZSet(), true, nil, nil
	}
	if v.Type != types.TypeZSet {
		return nil, false, nil, ErrWrongType
	}
	return v.Data.(*types.ZSet), false, &v, nil
}
