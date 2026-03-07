package store

import (
	"testing"
	"time"
)

func TestSAdd(t *testing.T) {
	s := NewMemoryStore()

	// 添加新成员
	n, err := s.SAdd("set", "a", "b", "c")
	if err != nil {
		t.Fatalf("SAdd failed: %v", err)
	}
	if n != 3 {
		t.Errorf("SAdd should return 3, got %d", n)
	}

	// 添加重复成员
	n, err = s.SAdd("set", "a", "d")
	if err != nil {
		t.Fatalf("SAdd failed: %v", err)
	}
	if n != 1 {
		t.Errorf("SAdd should return 1 (only d is new), got %d", n)
	}
}

func TestSRem(t *testing.T) {
	s := NewMemoryStore()

	s.SAdd("set", "a", "b", "c")

	// 移除存在的成员
	n, err := s.SRem("set", "a", "b")
	if err != nil {
		t.Fatalf("SRem failed: %v", err)
	}
	if n != 2 {
		t.Errorf("SRem should return 2, got %d", n)
	}

	// 移除不存在的成员
	n, err = s.SRem("set", "not_exist")
	if err != nil {
		t.Fatalf("SRem failed: %v", err)
	}
	if n != 0 {
		t.Errorf("SRem should return 0, got %d", n)
	}

	// 删除所有成员后，key 应该被删除
	s.SRem("set", "c")
	if s.Exists("set") {
		t.Error("Key should be deleted when set is empty")
	}
}

func TestSIsMember(t *testing.T) {
	s := NewMemoryStore()

	// 不存在的 key
	ok, err := s.SIsMember("not_exist", "a")
	if err != nil || ok {
		t.Error("SIsMember on non-exist should return false")
	}

	s.SAdd("set", "a", "b")

	// 存在的成员
	ok, err = s.SIsMember("set", "a")
	if err != nil || !ok {
		t.Error("SIsMember should return true for existing member")
	}

	// 不存在的成员
	ok, err = s.SIsMember("set", "c")
	if err != nil || ok {
		t.Error("SIsMember should return false for non-existing member")
	}
}

func TestSMembers(t *testing.T) {
	s := NewMemoryStore()

	// 不存在的 key
	members, err := s.SMembers("not_exist")
	if err != nil || len(members) != 0 {
		t.Error("SMembers on non-exist should return empty")
	}

	s.SAdd("set", "a", "b", "c")

	members, err = s.SMembers("set")
	if err != nil {
		t.Fatalf("SMembers failed: %v", err)
	}
	if len(members) != 3 {
		t.Errorf("SMembers should return 3 members, got %d", len(members))
	}
}

func TestSCard(t *testing.T) {
	s := NewMemoryStore()

	// 不存在的 key
	card, err := s.SCard("not_exist")
	if err != nil || card != 0 {
		t.Error("SCard on non-exist should return 0")
	}

	s.SAdd("set", "a", "b")

	card, err = s.SCard("set")
	if err != nil || card != 2 {
		t.Errorf("SCard should return 2, got %d", card)
	}
}

func TestSPop(t *testing.T) {
	s := NewMemoryStore()

	s.SAdd("set", "a", "b", "c", "d", "e")

	// 弹出 2 个
	members, err := s.SPop("set", 2)
	if err != nil {
		t.Fatalf("SPop failed: %v", err)
	}
	if len(members) != 2 {
		t.Errorf("SPop should return 2 members, got %d", len(members))
	}

	// 剩余 3 个
	card, _ := s.SCard("set")
	if card != 3 {
		t.Errorf("Set should have 3 members left, got %d", card)
	}

	// 弹出所有
	s.SPop("set", 10)
	if s.Exists("set") {
		t.Error("Key should be deleted when set is empty")
	}
}

func TestSUnion(t *testing.T) {
	s := NewMemoryStore()

	s.SAdd("set1", "a", "b", "c")
	s.SAdd("set2", "b", "c", "d")
	s.SAdd("set3", "c", "d", "e")

	// 并集
	members, err := s.SUnion("set1", "set2", "set3")
	if err != nil {
		t.Fatalf("SUnion failed: %v", err)
	}
	if len(members) != 5 { // a, b, c, d, e
		t.Errorf("SUnion should return 5 members, got %d", len(members))
	}
}

func TestSInter(t *testing.T) {
	s := NewMemoryStore()

	s.SAdd("set1", "a", "b", "c")
	s.SAdd("set2", "b", "c", "d")
	s.SAdd("set3", "c", "d", "e")

	// 交集
	members, err := s.SInter("set1", "set2", "set3")
	if err != nil {
		t.Fatalf("SInter failed: %v", err)
	}
	if len(members) != 1 { // only c
		t.Errorf("SInter should return 1 member (c), got %v", members)
	}
}

func TestSDiff(t *testing.T) {
	s := NewMemoryStore()

	s.SAdd("set1", "a", "b", "c")
	s.SAdd("set2", "b", "c", "d")

	// 差集 set1 - set2
	members, err := s.SDiff("set1", "set2")
	if err != nil {
		t.Fatalf("SDiff failed: %v", err)
	}
	if len(members) != 1 || members[0] != "a" {
		t.Errorf("SDiff should return [a], got %v", members)
	}
}

func TestSetWrongType(t *testing.T) {
	s := NewMemoryStore()

	s.SetString("str", "hello", 0)

	_, err := s.SAdd("str", "a")
	if err == nil {
		t.Error("SAdd on string should return WRONGTYPE")
	}

	_, err = s.SCard("str")
	if err == nil {
		t.Error("SCard on string should return WRONGTYPE")
	}
}

func TestSetExpire(t *testing.T) {
	s := NewMemoryStore()

	s.SAdd("set", "a")
	s.Expire("set", 100*time.Millisecond)

	// 修改，过期时间应该保留
	s.SAdd("set", "b")

	ttl := s.TTL("set")
	if ttl <= 0 || ttl > 100*time.Millisecond {
		t.Errorf("TTL should be around 100ms, got %v", ttl)
	}
}
