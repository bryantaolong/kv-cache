package store

import (
	"testing"
	"time"
)

func TestHSet(t *testing.T) {
	s := NewMemoryStore()

	// 测试新增 field
	n, err := s.HSet("user:1", "name", "tom")
	if err != nil {
		t.Fatalf("HSet failed: %v", err)
	}
	if n != 1 {
		t.Errorf("HSet new field should return 1, got %d", n)
	}

	// 测试更新 field
	n, err = s.HSet("user:1", "name", "jerry")
	if err != nil {
		t.Fatalf("HSet failed: %v", err)
	}
	if n != 0 {
		t.Errorf("HSet update should return 0, got %d", n)
	}

	// 验证值
	val, ok, err := s.HGet("user:1", "name")
	if err != nil {
		t.Fatalf("HGet failed: %v", err)
	}
	if !ok || val != "jerry" {
		t.Errorf("HGet got %s, want jerry", val)
	}
}

func TestHSetWrongType(t *testing.T) {
	s := NewMemoryStore()

	// 先设置 String
	s.SetString("str", "hello", 0)

	// 对 String 调用 HSet 应该报错
	_, err := s.HSet("str", "field", "value")
	if err == nil {
		t.Error("HSet on string should return WRONGTYPE error")
	}
	if !IsWrongTypeErr(err) {
		t.Errorf("Expected WRONGTYPE error, got %v", err)
	}
}

func TestHGetNotExist(t *testing.T) {
	s := NewMemoryStore()

	// key 不存在
	_, ok, err := s.HGet("not_exist", "field")
	if err != nil {
		t.Fatalf("HGet failed: %v", err)
	}
	if ok {
		t.Error("HGet on non-exist key should return false")
	}

	// key 存在但 field 不存在
	s.HSet("user:1", "name", "tom")
	_, ok, err = s.HGet("user:1", "age")
	if err != nil {
		t.Fatalf("HGet failed: %v", err)
	}
	if ok {
		t.Error("HGet on non-exist field should return false")
	}
}

func TestHGetAll(t *testing.T) {
	s := NewMemoryStore()

	// 不存在的 key
	_, ok, err := s.HGetAll("not_exist")
	if err != nil {
		t.Fatalf("HGetAll failed: %v", err)
	}
	if ok {
		t.Error("HGetAll on non-exist key should return false")
	}

	// 设置多个 field
	s.HSet("user:1", "name", "tom")
	s.HSet("user:1", "age", "20")
	s.HSet("user:1", "city", "beijing")

	hash, ok, err := s.HGetAll("user:1")
	if err != nil {
		t.Fatalf("HGetAll failed: %v", err)
	}
	if !ok {
		t.Error("HGetAll should return true")
	}
	if hash.HLen() != 3 {
		t.Errorf("HGetAll returned hash with %d fields, want 3", hash.HLen())
	}
}

func TestHDel(t *testing.T) {
	s := NewMemoryStore()

	s.HSet("user:1", "name", "tom")
	s.HSet("user:1", "age", "20")
	s.HSet("user:1", "city", "beijing")

	// 删除存在的 field
	n, err := s.HDel("user:1", "name", "age")
	if err != nil {
		t.Fatalf("HDel failed: %v", err)
	}
	if n != 2 {
		t.Errorf("HDel should return 2, got %d", n)
	}

	// 删除不存在的 field
	n, err = s.HDel("user:1", "not_exist")
	if err != nil {
		t.Fatalf("HDel failed: %v", err)
	}
	if n != 0 {
		t.Errorf("HDel non-exist should return 0, got %d", n)
	}

	// 验证只剩 city
	len, _ := s.HLen("user:1")
	if len != 1 {
		t.Errorf("Hash should have 1 field, got %d", len)
	}
}

func TestHLen(t *testing.T) {
	s := NewMemoryStore()

	// 不存在的 key
	len, err := s.HLen("not_exist")
	if err != nil {
		t.Fatalf("HLen failed: %v", err)
	}
	if len != 0 {
		t.Errorf("HLen on non-exist should return 0, got %d", len)
	}

	// 添加后
	s.HSet("user:1", "a", "1")
	s.HSet("user:1", "b", "2")
	len, err = s.HLen("user:1")
	if err != nil {
		t.Fatalf("HLen failed: %v", err)
	}
	if len != 2 {
		t.Errorf("HLen should return 2, got %d", len)
	}
}

func TestHashExpire(t *testing.T) {
	s := NewMemoryStore()

	// 设置带过期时间的 Hash
	s.HSet("temp", "field", "value")
	s.Expire("temp", 100*time.Millisecond)

	// 修改 Hash，过期时间应该保留
	s.HSet("temp", "field2", "value2")

	// 检查过期时间还在
	ttl := s.TTL("temp")
	if ttl <= 0 || ttl > 100*time.Millisecond {
		t.Errorf("TTL should be around 100ms, got %v", ttl)
	}

	// 等待过期
	time.Sleep(200 * time.Millisecond)

	// 应该过期了
	_, ok, _ := s.HGet("temp", "field")
	if ok {
		t.Error("Hash should be expired")
	}
}

func TestHKeysHVals(t *testing.T) {
	s := NewMemoryStore()

	s.HSet("user:1", "name", "tom")
	s.HSet("user:1", "age", "20")

	keys, err := s.HKeys("user:1")
	if err != nil {
		t.Fatalf("HKeys failed: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("HKeys should return 2 keys, got %d", len(keys))
	}

	vals, err := s.HVals("user:1")
	if err != nil {
		t.Fatalf("HVals failed: %v", err)
	}
	if len(vals) != 2 {
		t.Errorf("HVals should return 2 values, got %d", len(vals))
	}
}
