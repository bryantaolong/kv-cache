package storage

import (
	"sync"
	"testing"
	"time"

	"kv-cache/internal/storage/types"
)

func TestEvictorLRUOrder(t *testing.T) {
	e := NewEvictor()
	e.SetMaxMemory(5)
	e.SetEvictionPolicy(EvictLRU)

	now := time.Now()
	data := map[string]*types.Value{
		"oldest": {Type: types.TypeString, Data: "aaaa", AccessedAt: now.Add(-3 * time.Second)},
		"middle": {Type: types.TypeString, Data: "b", AccessedAt: now.Add(-2 * time.Second)},
		"newest": {Type: types.TypeString, Data: "cc", AccessedAt: now.Add(-1 * time.Second)},
	}

	target := e.SelectEvictionTargets(data)
	if len(target) != 1 || target[0] != "oldest" {
		t.Fatalf("expected oldest, got %v", target)
	}
}

func TestEvictorRandomEviction(t *testing.T) {
	e := NewEvictor()
	e.SetMaxMemory(4)
	e.SetEvictionPolicy(EvictRandom)

	now := time.Now()
	data := map[string]*types.Value{
		"a": {Type: types.TypeString, Data: "aaa", AccessedAt: now},
		"b": {Type: types.TypeString, Data: "bbb", AccessedAt: now},
	}

	targets := e.SelectEvictionTargets(data)
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d: %v", len(targets), targets)
	}
	if targets[0] != "a" && targets[0] != "b" {
		t.Fatalf("unexpected target %q", targets[0])
	}
}

func TestEvictorNoEvictionWhenUnderLimit(t *testing.T) {
	e := NewEvictor()
	e.SetMaxMemory(1000)
	e.SetEvictionPolicy(EvictRandom)

	now := time.Now()
	data := map[string]*types.Value{
		"a": {Type: types.TypeString, Data: "hello", AccessedAt: now},
	}

	targets := e.SelectEvictionTargets(data)
	if len(targets) != 0 {
		t.Fatalf("expected no targets, got %v", targets)
	}
}

func TestEvictorSkipWhenNoPolicy(t *testing.T) {
	e := NewEvictor()
	e.SetMaxMemory(1)
	e.SetEvictionPolicy(EvictNoEviction)

	data := map[string]*types.Value{
		"a": {Type: types.TypeString, Data: "hello", AccessedAt: time.Now()},
	}

	targets := e.SelectEvictionTargets(data)
	if len(targets) != 0 {
		t.Fatalf("expected no targets under no-eviction, got %v", targets)
	}
}

func TestEvictorHandlesNilData(t *testing.T) {
	e := NewEvictor()
	e.SetMaxMemory(1)
	e.SetEvictionPolicy(EvictLRU)

	// Should not panic on nil data
	e.SelectEvictionTargets(nil)
}

func TestMemoryStoreEvictionDoesNotDeadlock(t *testing.T) {
	s := NewMemoryStore()
	s.SetMaxMemory(1)
	s.SetEvictionPolicy(EvictLRU)

	// This should not hang, even though Set used to trigger eviction while holding the write lock
	if err := s.Set("key", types.Value{Type: types.TypeString, Data: "a"}, 0); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
}

func TestEvictorLRUReturnsOldestWhenTie(t *testing.T) {
	e := NewEvictor()
	e.SetMaxMemory(3)
	e.SetEvictionPolicy(EvictLRU)

	now := time.Now()
	data := map[string]*types.Value{
		"a": {Type: types.TypeString, Data: "a", AccessedAt: now},
		"b": {Type: types.TypeString, Data: "b", AccessedAt: now},
		"c": {Type: types.TypeString, Data: "c", AccessedAt: now},
	}

	targets := e.SelectEvictionTargets(data)
	// All keys share the same AccessedAt; algorithm must still return exactly 1 key.
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d: %v", len(targets), targets)
	}
}

func TestEvictorLRUEvictsUntilBelowTarget(t *testing.T) {
	e := NewEvictor()
	e.SetMaxMemory(3)
	e.SetEvictionPolicy(EvictLRU)

	now := time.Now()
	data := map[string]*types.Value{
		"k1": {Type: types.TypeString, Data: "aa", AccessedAt: now.Add(-3 * time.Second)},
		"k2": {Type: types.TypeString, Data: "bb", AccessedAt: now.Add(-2 * time.Second)},
		"k3": {Type: types.TypeString, Data: "cc", AccessedAt: now.Add(-1 * time.Second)},
		"k4": {Type: types.TypeString, Data: "d", AccessedAt: now},
	}

	targets := e.SelectEvictionTargets(data)
	if len(targets) != 3 {
		t.Fatalf("expected 3 targets, got %d: %v", len(targets), targets)
	}
	if targets[0] != "k1" || targets[1] != "k2" || targets[2] != "k3" {
		t.Fatalf("expected oldest first, got %v", targets)
	}
}

func TestEvictorSelectsOldestEvenWhenExpired(t *testing.T) {
	e := NewEvictor()
	e.SetMaxMemory(3)
	e.SetEvictionPolicy(EvictLRU)

	past := time.Now().Add(-time.Hour)
	now := time.Now()
	data := map[string]*types.Value{
		"expired": {Type: types.TypeString, Data: "x", ExpireAt: &past, AccessedAt: now.Add(-time.Minute)},
		"fresh":   {Type: types.TypeString, Data: "yy", AccessedAt: now},
	}

	targets := e.SelectEvictionTargets(data)
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d: %v", len(targets), targets)
	}
	if targets[0] != "expired" {
		t.Fatalf("expected LRU to pick expired key, got %q", targets[0])
	}
}

func TestEvictorMixedTypeEstimate(t *testing.T) {
	e := NewEvictor()
	e.SetMaxMemory(5)
	e.SetEvictionPolicy(EvictLRU)

	now := time.Now()
	data := map[string]*types.Value{
		"hash": {
			Type:       types.TypeHash,
			Data:       map[string]string{"f1": "v1"},
			AccessedAt: now.Add(-2 * time.Second),
		},
		"list": {
			Type:       types.TypeList,
			Data:       []string{"a", "b"},
			AccessedAt: now.Add(-1 * time.Second),
		},
		"set": {
			Type:       types.TypeSet,
			Data:       types.Set{"m1": struct{}{}},
			AccessedAt: now,
		},
	}

	targets := e.SelectEvictionTargets(data)
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d: %v", len(targets), targets)
	}
	if targets[0] != "hash" || targets[1] != "list" {
		t.Fatalf("expected LRU order among top-2 oldest, got %v", targets)
	}
}

func TestEvictorMaxMemoryZeroSkips(t *testing.T) {
	e := NewEvictor()
	e.SetMaxMemory(0)
	e.SetEvictionPolicy(EvictLRU)

	now := time.Now()
	data := map[string]*types.Value{
		"k": {Type: types.TypeString, Data: "anything", AccessedAt: now.Add(-time.Hour)},
	}

	targets := e.SelectEvictionTargets(data)
	if len(targets) != 0 {
		t.Fatalf("expected no targets when maxMemory=0, got %v", targets)
	}
}

func TestMemoryStoreSetOverMaxMemoryTriggersEviction(t *testing.T) {
	s := NewMemoryStore()
	s.SetMaxMemory(4)
	s.SetEvictionPolicy(EvictLRU)

	_ = s.Set("k1", types.Value{Type: types.TypeString, Data: "a"}, 0)
	_ = s.Set("k2", types.Value{Type: types.TypeString, Data: "b"}, 0)
	_ = s.Set("k3", types.Value{Type: types.TypeString, Data: "c"}, 0)

	if s.DBSize() != 3 {
		t.Fatalf("expected 3 keys before overflow, got %d", s.DBSize())
	}

	_ = s.Set("k4", types.Value{Type: types.TypeString, Data: "d"}, 0)

	if s.DBSize() != 3 {
		t.Fatalf("expected 3 keys after overflow, got %d", s.DBSize())
	}
}

func TestMemoryStoreAccessTimeUpdatesOnGet(t *testing.T) {
	s := NewMemoryStore()
	s.SetMaxMemory(4)
	s.SetEvictionPolicy(EvictLRU)

	_ = s.Set("k1", types.Value{Type: types.TypeString, Data: "a"}, 0)
	_ = s.Set("k2", types.Value{Type: types.TypeString, Data: "b"}, 0)
	_ = s.Set("k3", types.Value{Type: types.TypeString, Data: "c"}, 0)

	if s.DBSize() != 3 {
		t.Fatalf("expected 3 keys after initial sets, got %d", s.DBSize())
	}

	_, _ = s.Get("k1")
	time.Sleep(50 * time.Millisecond)
	_, _ = s.Get("k2")
	time.Sleep(50 * time.Millisecond)

	_ = s.Set("k4", types.Value{Type: types.TypeString, Data: "d"}, 0)

	if s.DBSize() != 3 {
		t.Fatalf("expected 3 keys after refresh + overflow, got %d", s.DBSize())
	}
	if _, ok := s.Get("k1"); !ok {
		t.Fatal("k1 should survive; it was refreshed most recently")
	}
}

func TestEvictorConcurrentAccess(t *testing.T) {
	e := NewEvictor()
	e.SetMaxMemory(10)
	e.SetEvictionPolicy(EvictLRU)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			now := time.Now()
			data := map[string]*types.Value{
				"key": {Type: types.TypeString, Data: "value", AccessedAt: now},
			}
			_ = e.SelectEvictionTargets(data)
		}(i)
	}
	wg.Wait()
}
