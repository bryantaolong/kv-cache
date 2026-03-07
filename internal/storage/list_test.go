package store

import (
	"testing"
	"time"
)

func TestLPush(t *testing.T) {
	s := NewMemoryStore()

	// 测试单个元素
	n, err := s.LPush("list", "a")
	if err != nil {
		t.Fatalf("LPush failed: %v", err)
	}
	if n != 1 {
		t.Errorf("LPush should return 1, got %d", n)
	}

	// 测试多个元素 LPUSH list b c => [c, b, a]
	n, err = s.LPush("list", "b", "c")
	if err != nil {
		t.Fatalf("LPush failed: %v", err)
	}
	if n != 3 {
		t.Errorf("LPush should return 3, got %d", n)
	}

	// 验证顺序
	rangeVals, _ := s.LRange("list", 0, -1)
	if len(rangeVals) != 3 || rangeVals[0] != "c" || rangeVals[2] != "a" {
		t.Errorf("List order wrong: %v", rangeVals)
	}
}

func TestRPush(t *testing.T) {
	s := NewMemoryStore()

	// RPush a b c => [a, b, c]
	n, err := s.RPush("list", "a", "b", "c")
	if err != nil {
		t.Fatalf("RPush failed: %v", err)
	}
	if n != 3 {
		t.Errorf("RPush should return 3, got %d", n)
	}

	rangeVals, _ := s.LRange("list", 0, -1)
	if len(rangeVals) != 3 || rangeVals[0] != "a" || rangeVals[2] != "c" {
		t.Errorf("List order wrong: %v", rangeVals)
	}
}

func TestLPop(t *testing.T) {
	s := NewMemoryStore()

	// 空列表
	_, ok, err := s.LPop("not_exist")
	if err != nil || ok {
		t.Error("LPop on empty list should return false")
	}

	s.RPush("list", "a", "b", "c")

	val, ok, err := s.LPop("list")
	if err != nil {
		t.Fatalf("LPop failed: %v", err)
	}
	if !ok || val != "a" {
		t.Errorf("LPop got %s, want a", val)
	}

	// 弹出所有元素
	s.LPop("list")
	s.LPop("list")
	_, _, _ = s.LPop("list") // 弹空

	// 验证 key 被删除
	if s.Exists("list") {
		t.Error("Key should be deleted when list is empty")
	}
}

func TestRPop(t *testing.T) {
	s := NewMemoryStore()

	s.RPush("list", "a", "b", "c")

	val, ok, err := s.RPop("list")
	if err != nil {
		t.Fatalf("RPop failed: %v", err)
	}
	if !ok || val != "c" {
		t.Errorf("RPop got %s, want c", val)
	}
}

func TestLRange(t *testing.T) {
	s := NewMemoryStore()

	s.RPush("list", "a", "b", "c", "d", "e")

	// 正常范围
	vals, err := s.LRange("list", 1, 3)
	if err != nil {
		t.Fatalf("LRange failed: %v", err)
	}
	if len(vals) != 3 || vals[0] != "b" || vals[2] != "d" {
		t.Errorf("LRange got %v, want [b c d]", vals)
	}

	// 负数索引
	vals, _ = s.LRange("list", -2, -1)
	if len(vals) != 2 || vals[0] != "d" || vals[1] != "e" {
		t.Errorf("LRange negative index got %v, want [d e]", vals)
	}

	// 全部
	vals, _ = s.LRange("list", 0, -1)
	if len(vals) != 5 {
		t.Errorf("LRange 0 -1 got %d elements, want 5", len(vals))
	}

	// 越界
	vals, _ = s.LRange("list", 10, 20)
	if len(vals) != 0 {
		t.Error("LRange out of range should return empty")
	}
}

func TestLLen(t *testing.T) {
	s := NewMemoryStore()

	// 不存在的 key
	len, err := s.LLen("not_exist")
	if err != nil || len != 0 {
		t.Error("LLen on non-exist should return 0")
	}

	s.RPush("list", "a", "b")
	len, err = s.LLen("list")
	if err != nil || len != 2 {
		t.Errorf("LLen should return 2, got %d", len)
	}
}

func TestLIndex(t *testing.T) {
	s := NewMemoryStore()

	s.RPush("list", "a", "b", "c")

	// 正常索引
	val, ok, _ := s.LIndex("list", 1)
	if !ok || val != "b" {
		t.Errorf("LIndex 1 got %s, want b", val)
	}

	// 负数索引
	val, ok, _ = s.LIndex("list", -1)
	if !ok || val != "c" {
		t.Errorf("LIndex -1 got %s, want c", val)
	}

	// 越界
	_, ok, _ = s.LIndex("list", 10)
	if ok {
		t.Error("LIndex out of range should return false")
	}
}

func TestListWrongType(t *testing.T) {
	s := NewMemoryStore()

	s.SetString("str", "hello", 0)

	_, err := s.LPush("str", "a")
	if err == nil {
		t.Error("LPush on string should return WRONGTYPE")
	}

	_, _, err = s.LPop("str")
	if err == nil {
		t.Error("LPop on string should return WRONGTYPE")
	}
}

func TestListExpire(t *testing.T) {
	s := NewMemoryStore()

	s.LPush("list", "a")
	s.Expire("list", 100*time.Millisecond)

	// 修改列表，过期时间应该保留
	s.RPush("list", "b")

	ttl := s.TTL("list")
	if ttl <= 0 || ttl > 100*time.Millisecond {
		t.Errorf("TTL should be around 100ms, got %v", ttl)
	}
}
