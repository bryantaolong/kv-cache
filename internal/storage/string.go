package store

import (
	"errors"
	"time"

	"kv-cache/internal/storage/types"
)

// SetString 设置字符串值
func (s *MemoryStore) SetString(key, value string, ttl time.Duration) error {
	return s.Set(key, types.Value{
		Type: types.TypeString,
		Data: value,
	}, ttl)
}

// GetString 获取字符串值
func (s *MemoryStore) GetString(key string) (string, bool) {
	v, ok := s.Get(key)
	if !ok || v.Type != types.TypeString {
		return "", false
	}
	str, ok := v.Data.(string)
	return str, ok
}

// Append 追加字符串
func (s *MemoryStore) Append(key, suffix string) (int, error) {
	v, ok := s.Get(key)

	var oldStr string
	if ok {
		if v.Type != types.TypeString {
			return 0, errors.New("WRONGTYPE")
		}
		oldStr = v.Data.(string)
	}

	newStr := oldStr + suffix

	// 保留过期时间
	ttl := time.Duration(0)
	if ok && v.ExpireAt != nil {
		ttl = time.Until(*v.ExpireAt)
		if ttl < 0 {
			ttl = 0
		}
	}

	err := s.SetString(key, newStr, ttl)
	if err != nil {
		return 0, err
	}
	return len(newStr), nil
}

// Incr 自增 1
func (s *MemoryStore) Incr(key string) (string, error) {
	return s.IncrBy(key, 1)
}

// IncrBy 自增指定值
func (s *MemoryStore) IncrBy(key string, delta int64) (string, error) {
	v, ok := s.Get(key)

	if !ok {
		// 不存在，初始化为 0 再自增
		newVal, err := types.StringIncrBy("0", delta)
		if err != nil {
			return "", err
		}
		s.SetString(key, newVal, 0)
		return newVal, nil
	}

	if v.Type != types.TypeString {
		return "", errors.New("WRONGTYPE")
	}

	str := v.Data.(string)
	newVal, err := types.StringIncrBy(str, delta)
	if err != nil {
		return "", err
	}

	// 保留过期时间
	ttl := time.Duration(0)
	if v.ExpireAt != nil {
		ttl = time.Until(*v.ExpireAt)
		if ttl < 0 {
			ttl = 0
		}
	}

	s.SetString(key, newVal, ttl)
	return newVal, nil
}

// Decr 自减 1
func (s *MemoryStore) Decr(key string) (string, error) {
	return s.IncrBy(key, -1)
}

// DecrBy 自减指定值
func (s *MemoryStore) DecrBy(key string, delta int64) (string, error) {
	return s.IncrBy(key, -delta)
}
