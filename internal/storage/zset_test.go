package store

import (
	"testing"
	"time"
)

func TestZAdd(t *testing.T) {
	s := NewMemoryStore()

	// 新增
	n, err := s.ZAdd("zset", 1.0, "one")
	if err != nil {
		t.Fatalf("ZAdd failed: %v", err)
	}
	if n != 1 {
		t.Errorf("ZAdd new should return 1, got %d", n)
	}

	// 再新增
	s.ZAdd("zset", 2.0, "two")
	s.ZAdd("zset", 3.0, "three")

	// 更新
	n, err = s.ZAdd("zset", 10.0, "one")
	if err != nil {
		t.Fatalf("ZAdd failed: %v", err)
	}
	if n != 0 {
		t.Errorf("ZAdd update should return 0, got %d", n)
	}

	// 验证更新后的 score
	score, ok, _ := s.ZScore("zset", "one")
	if !ok || score != 10.0 {
		t.Errorf("ZScore should be 10.0, got %f", score)
	}
}

func TestZScore(t *testing.T) {
	s := NewMemoryStore()

	// 不存在
	_, ok, err := s.ZScore("not_exist", "member")
	if err != nil || ok {
		t.Error("ZScore on non-exist should return false")
	}

	s.ZAdd("zset", 5.0, "member")

	score, ok, err := s.ZScore("zset", "member")
	if err != nil {
		t.Fatalf("ZScore failed: %v", err)
	}
	if !ok || score != 5.0 {
		t.Errorf("ZScore got %f, want 5.0", score)
	}
}

func TestZRank(t *testing.T) {
	s := NewMemoryStore()

	s.ZAdd("zset", 10.0, "a")
	s.ZAdd("zset", 20.0, "b")
	s.ZAdd("zset", 30.0, "c")

	// 升序排名：a(0), b(1), c(2)
	rank, ok, err := s.ZRank("zset", "b")
	if err != nil {
		t.Fatalf("ZRank failed: %v", err)
	}
	if !ok || rank != 1 {
		t.Errorf("ZRank got %d, want 1", rank)
	}

	// 倒序排名：c(0), b(1), a(2)
	rank, ok, err = s.ZRevRank("zset", "b")
	if err != nil {
		t.Fatalf("ZRevRank failed: %v", err)
	}
	if !ok || rank != 1 {
		t.Errorf("ZRevRank got %d, want 1", rank)
	}
}

func TestZRange(t *testing.T) {
	s := NewMemoryStore()

	s.ZAdd("zset", 1.0, "one")
	s.ZAdd("zset", 2.0, "two")
	s.ZAdd("zset", 3.0, "three")

	// 升序范围 [0, 1] => one, two
	members, err := s.ZRange("zset", 0, 1)
	if err != nil {
		t.Fatalf("ZRange failed: %v", err)
	}
	if len(members) != 2 || members[0].Member != "one" || members[1].Member != "two" {
		t.Errorf("ZRange got %v", members)
	}

	// 倒序范围 [0, 1] => three, two
	members, err = s.ZRevRange("zset", 0, 1)
	if err != nil {
		t.Fatalf("ZRevRange failed: %v", err)
	}
	if len(members) != 2 || members[0].Member != "three" || members[1].Member != "two" {
		t.Errorf("ZRevRange got %v", members)
	}
}

func TestZRangeByScore(t *testing.T) {
	s := NewMemoryStore()

	s.ZAdd("zset", 1.0, "a")
	s.ZAdd("zset", 2.0, "b")
	s.ZAdd("zset", 3.0, "c")
	s.ZAdd("zset", 4.0, "d")
	s.ZAdd("zset", 5.0, "e")

	// score 在 [2, 4] 范围内
	members, err := s.ZRangeByScore("zset", 2.0, 4.0)
	if err != nil {
		t.Fatalf("ZRangeByScore failed: %v", err)
	}
	if len(members) != 3 {
		t.Errorf("ZRangeByScore should return 3 members, got %d", len(members))
	}
}

func TestZRem(t *testing.T) {
	s := NewMemoryStore()

	s.ZAdd("zset", 1.0, "a")
	s.ZAdd("zset", 2.0, "b")
	s.ZAdd("zset", 3.0, "c")

	n, err := s.ZRem("zset", "a", "b")
	if err != nil {
		t.Fatalf("ZRem failed: %v", err)
	}
	if n != 2 {
		t.Errorf("ZRem should return 2, got %d", n)
	}

	// 验证只剩 c
	card, _ := s.ZCard("zset")
	if card != 1 {
		t.Errorf("ZCard should be 1, got %d", card)
	}

	// 删除所有，key 应该被删除
	s.ZRem("zset", "c")
	if s.Exists("zset") {
		t.Error("Key should be deleted when zset is empty")
	}
}

func TestZCard(t *testing.T) {
	s := NewMemoryStore()

	// 不存在
	card, err := s.ZCard("not_exist")
	if err != nil || card != 0 {
		t.Error("ZCard on non-exist should return 0")
	}

	s.ZAdd("zset", 1.0, "a")
	s.ZAdd("zset", 2.0, "b")

	card, err = s.ZCard("zset")
	if err != nil || card != 2 {
		t.Errorf("ZCard should be 2, got %d", card)
	}
}

func TestZCount(t *testing.T) {
	s := NewMemoryStore()

	s.ZAdd("zset", 1.0, "a")
	s.ZAdd("zset", 2.0, "b")
	s.ZAdd("zset", 3.0, "c")
	s.ZAdd("zset", 4.0, "d")

	count, err := s.ZCount("zset", 2.0, 3.0)
	if err != nil {
		t.Fatalf("ZCount failed: %v", err)
	}
	if count != 2 {
		t.Errorf("ZCount should return 2, got %d", count)
	}
}

func TestZIncrBy(t *testing.T) {
	s := NewMemoryStore()

	// 新增
	score, err := s.ZIncrBy("zset", 5.0, "member")
	if err != nil {
		t.Fatalf("ZIncrBy failed: %v", err)
	}
	if score != 5.0 {
		t.Errorf("ZIncrBy should return 5.0, got %f", score)
	}

	// 增加
	score, err = s.ZIncrBy("zset", 3.0, "member")
	if err != nil {
		t.Fatalf("ZIncrBy failed: %v", err)
	}
	if score != 8.0 {
		t.Errorf("ZIncrBy should return 8.0, got %f", score)
	}
}

func TestZSetWrongType(t *testing.T) {
	s := NewMemoryStore()

	s.SetString("str", "hello", 0)

	_, err := s.ZAdd("str", 1.0, "member")
	if err == nil {
		t.Error("ZAdd on string should return WRONGTYPE")
	}

	_, _, err = s.ZRank("str", "member")
	if err == nil {
		t.Error("ZRank on string should return WRONGTYPE")
	}
}

func TestZSetExpire(t *testing.T) {
	s := NewMemoryStore()

	s.ZAdd("zset", 1.0, "a")
	s.Expire("zset", 100*time.Millisecond)

	// 修改，过期时间应该保留
	s.ZAdd("zset", 2.0, "b")

	ttl := s.TTL("zset")
	if ttl <= 0 || ttl > 100*time.Millisecond {
		t.Errorf("TTL should be around 100ms, got %v", ttl)
	}
}
