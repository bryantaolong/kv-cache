package store

import (
	"sync"
	"testing"
	"time"

	"kv-cache/internal/storage/types"
)

func TestNewMemoryStore(t *testing.T) {
	s := NewMemoryStore()
	if s == nil {
		t.Fatal("NewMemoryStore should not return nil")
	}
	if s.data == nil {
		t.Fatal("data map should be initialized")
	}
}

func TestSetAndGet(t *testing.T) {
	s := NewMemoryStore()

	// 测试 Set 和 Get
	val := types.Value{Type: types.TypeString, Data: "hello"}
	err := s.Set("key1", val, 0)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	v, ok := s.Get("key1")
	if !ok {
		t.Fatal("Get should return true for existing key")
	}
	if v.Data != "hello" {
		t.Fatalf("expected 'hello', got %v", v.Data)
	}
}

func TestGetNotExist(t *testing.T) {
	s := NewMemoryStore()

	_, ok := s.Get("not_exist")
	if ok {
		t.Fatal("Get should return false for non-existing key")
	}
}

func TestSetWithTTL(t *testing.T) {
	s := NewMemoryStore()

	val := types.Value{Type: types.TypeString, Data: "temp"}
	s.Set("key", val, 100*time.Millisecond)

	// 立即获取应该成功
	_, ok := s.Get("key")
	if !ok {
		t.Fatal("key should exist immediately after Set")
	}

	// 等待过期
	time.Sleep(200 * time.Millisecond)

	_, ok = s.Get("key")
	if ok {
		t.Fatal("key should be expired after TTL")
	}
}

func TestSetNoTTL(t *testing.T) {
	s := NewMemoryStore()

	val := types.Value{Type: types.TypeString, Data: "permanent"}
	s.Set("key", val, 0) // 永不过期

	v, ok := s.Get("key")
	if !ok {
		t.Fatal("key should exist")
	}
	if v.ExpireAt != nil {
		t.Fatal("ExpireAt should be nil for no TTL")
	}
}

func TestDelete(t *testing.T) {
	s := NewMemoryStore()

	val := types.Value{Type: types.TypeString, Data: "to_delete"}
	s.Set("key", val, 0)

	deleted := s.Delete("key")
	if !deleted {
		t.Fatal("Delete should return true")
	}

	_, ok := s.Get("key")
	if ok {
		t.Fatal("key should not exist after Delete")
	}
}

func TestExists(t *testing.T) {
	s := NewMemoryStore()

	if s.Exists("key") {
		t.Fatal("Exists should return false for non-existing key")
	}

	val := types.Value{Type: types.TypeString, Data: "exists"}
	s.Set("key", val, 0)

	if !s.Exists("key") {
		t.Fatal("Exists should return true for existing key")
	}
}

func TestKeys(t *testing.T) {
	s := NewMemoryStore()

	s.Set("a", types.Value{Type: types.TypeString, Data: "1"}, 0)
	s.Set("b", types.Value{Type: types.TypeString, Data: "2"}, 0)
	s.Set("c", types.Value{Type: types.TypeString, Data: "3"}, 0)

	keys := s.Keys()
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}
}

func TestFlush(t *testing.T) {
	s := NewMemoryStore()

	s.Set("key1", types.Value{Type: types.TypeString, Data: "1"}, 0)
	s.Set("key2", types.Value{Type: types.TypeString, Data: "2"}, 0)

	s.Flush()

	if s.Exists("key1") || s.Exists("key2") {
		t.Fatal("all keys should be deleted after Flush")
	}
}

func TestExpire(t *testing.T) {
	s := NewMemoryStore()

	val := types.Value{Type: types.TypeString, Data: "value"}
	s.Set("key", val, 0) // 永不过期

	// 设置过期时间
	ok := s.Expire("key", 100*time.Millisecond)
	if !ok {
		t.Fatal("Expire should return true for existing key")
	}

	// 检查 TTL
	ttl := s.TTL("key")
	if ttl < 0 || ttl > 100*time.Millisecond {
		t.Fatalf("TTL should be positive and <= 100ms, got %v", ttl)
	}

	// 等待过期
	time.Sleep(200 * time.Millisecond)

	if s.Exists("key") {
		t.Fatal("key should be expired")
	}
}

func TestExpireNotExist(t *testing.T) {
	s := NewMemoryStore()

	ok := s.Expire("not_exist", time.Second)
	if ok {
		t.Fatal("Expire should return false for non-existing key")
	}
}

func TestTTL(t *testing.T) {
	s := NewMemoryStore()

	// 不存在的键
	ttl := s.TTL("not_exist")
	if ttl != -2 {
		t.Fatalf("TTL for non-existing key should be -2, got %v", ttl)
	}

	// 永不过期的键
	val := types.Value{Type: types.TypeString, Data: "value"}
	s.Set("no_ttl", val, 0)
	ttl = s.TTL("no_ttl")
	if ttl != -1 {
		t.Fatalf("TTL for key without expire should be -1, got %v", ttl)
	}

	// 有过期时间的键
	s.Set("with_ttl", val, 10*time.Second)
	ttl = s.TTL("with_ttl")
	if ttl < 0 || ttl > 10*time.Second {
		t.Fatalf("TTL should be between 0 and 10s, got %v", ttl)
	}
}

func TestDBSize(t *testing.T) {
	s := NewMemoryStore()

	if s.DBSize() != 0 {
		t.Fatalf("DBSize should be 0 for empty store, got %d", s.DBSize())
	}

	s.Set("a", types.Value{Type: types.TypeString, Data: "1"}, 0)
	s.Set("b", types.Value{Type: types.TypeString, Data: "2"}, 0)

	if s.DBSize() != 2 {
		t.Fatalf("DBSize should be 2, got %d", s.DBSize())
	}
}

func TestConcurrentAccess(t *testing.T) {
	s := NewMemoryStore()

	var wg sync.WaitGroup
	numGoroutines := 100
	numOps := 100

	// 并发写入
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				key := string(rune('a' + id%26))
				val := types.Value{Type: types.TypeString, Data: id}
				s.Set(key, val, 0)
			}
		}(i)
	}

	// 并发读取
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				key := string(rune('a' + id%26))
				s.Get(key)
			}
		}(i)
	}

	// 并发删除
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				key := string(rune('a' + id%26))
				s.Delete(key)
			}
		}(i)
	}

	wg.Wait()
}

func TestLazyExpire(t *testing.T) {
	s := NewMemoryStore()

	// 设置一个短过期时间的键
	val := types.Value{Type: types.TypeString, Data: "lazy"}
	s.Set("key", val, 50*time.Millisecond)

	// 等待过期
	time.Sleep(100 * time.Millisecond)

	// 通过 Get 触发惰性删除
	_, ok := s.Get("key")
	if ok {
		t.Fatal("Get should return false for expired key")
	}

	// 确认键已被删除
	if s.DBSize() != 0 {
		t.Fatal("expired key should be deleted from store")
	}
}
