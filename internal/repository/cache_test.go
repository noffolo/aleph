package repository

import (
	"testing"
	"time"
)

func TestToolCache_SetGet(t *testing.T) {
	tc := NewToolCache()
	defer tc.Close()

	tc.Set("key1", "value1")
	v, ok := tc.Get("key1")
	if !ok {
		t.Fatal("expected to find key1")
	}
	if v != "value1" {
		t.Fatalf("expected value1, got %v", v)
	}
}

func TestToolCache_Get_NonExistent(t *testing.T) {
	tc := NewToolCache()
	defer tc.Close()

	v, ok := tc.Get("nonexistent")
	if ok {
		t.Fatal("expected key not found")
	}
	if v != nil {
		t.Fatalf("expected nil, got %v", v)
	}
}

func TestToolCache_Invalidate(t *testing.T) {
	tc := NewToolCache()
	defer tc.Close()

	tc.Set("key1", "value1")
	tc.Invalidate("key1")

	v, ok := tc.Get("key1")
	if ok {
		t.Fatal("expected key to be invalidated")
	}
	if v != nil {
		t.Fatalf("expected nil after invalidate, got %v", v)
	}
}

func TestToolCache_Invalidate_NonExistent(t *testing.T) {
	tc := NewToolCache()
	defer tc.Close()

	// Should not panic or error
	tc.Invalidate("nonexistent")
}

func TestToolCache_SetMaxSize(t *testing.T) {
	tc := NewToolCache()
	defer tc.Close()

	tc.SetMaxSize(10)
	if tc.maxSize != 10 {
		t.Fatalf("expected maxSize 10, got %d", tc.maxSize)
	}
}

func TestToolCache_SetMaxSize_Zero(t *testing.T) {
	tc := NewToolCache()
	defer tc.Close()

	original := tc.maxSize
	tc.SetMaxSize(0)
	if tc.maxSize != original {
		t.Fatalf("expected maxSize to stay at %d, got %d", original, tc.maxSize)
	}
}

func TestToolCache_SetMaxSize_Negative(t *testing.T) {
	tc := NewToolCache()
	defer tc.Close()

	original := tc.maxSize
	tc.SetMaxSize(-1)
	if tc.maxSize != original {
		t.Fatalf("expected maxSize to stay at %d, got %d", original, tc.maxSize)
	}
}

func TestToolCache_SetMaxSize_Evicts(t *testing.T) {
	tc := NewToolCache()
	defer tc.Close()

	// Fill cache with 3 entries
	tc.Set("a", "1")
	tc.Set("b", "2")
	tc.Set("c", "3")

	if tc.len() != 3 {
		t.Fatalf("expected 3 entries, got %d", tc.len())
	}

	// Reduce max size to 2 — should evict oldest (by access time)
	tc.SetMaxSize(2)

	if tc.len() > 2 {
		t.Fatalf("expected at most 2 entries after eviction, got %d", tc.len())
	}
}

func TestToolCache_SetMaxSize_EvictsAll(t *testing.T) {
	tc := NewToolCache()
	defer tc.Close()

	tc.Set("a", "1")
	tc.Set("b", "2")
	tc.Set("c", "3")

	if tc.len() != 3 {
		t.Fatalf("expected 3 entries, got %d", tc.len())
	}

	// Reduce max size to 1 — evictLRU removes one entry per call
	tc.SetMaxSize(1)

	// evictLRU is a single-eviction, not a shrink-to-fit
	if tc.len() != 2 {
		t.Fatalf("expected 2 entries after single LRU eviction, got %d", tc.len())
	}
}

func TestToolCache_Close(t *testing.T) {
	tc := NewToolCache()
	tc.Close()
}

func TestToolCache_Close_NoDeadlock(t *testing.T) {
	tc := NewToolCache()

	// Close should return quickly regardless of cleanup interval
	done := make(chan struct{})
	go func() {
		tc.Close()
		close(done)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("Close() deadlocked or took too long")
	}
}

func TestToolCache_Set_EvictWhenFull(t *testing.T) {
	tc := NewToolCache()
	defer tc.Close()

	// Set max size to 2
	tc.SetMaxSize(2)

	tc.Set("a", "1")
	tc.Set("b", "2")
	// This should trigger eviction of the LRU entry
	tc.Set("c", "3")

	if tc.len() > 2 {
		t.Fatalf("expected at most 2 entries, got %d", tc.len())
	}
}

func TestToolCache_Get_UpdatesAccessed(t *testing.T) {
	tc := NewToolCache()
	defer tc.Close()

	tc.SetMaxSize(2)
	tc.Set("a", "1")
	tc.Set("b", "2")

	// Access "a" to make it more recently used
	tc.Get("a")

	// "b" should be LRU now. Inserting "c" should evict "b"
	tc.Set("c", "3")

	// "a" should still exist (was accessed)
	_, ok := tc.Get("a")
	if !ok {
		t.Fatal("expected a to survive LRU eviction after access")
	}
}

func TestToolCache_NewToolCache(t *testing.T) {
	tc := NewToolCache()
	defer tc.Close()

	if tc.cache == nil {
		t.Fatal("expected non-nil cache map")
	}
	if tc.ttl != defaultToolCacheTTL {
		t.Fatalf("expected default ttl %v, got %v", defaultToolCacheTTL, tc.ttl)
	}
	if tc.maxSize != defaultMaxSize {
		t.Fatalf("expected default maxSize %d, got %d", defaultMaxSize, tc.maxSize)
	}
	if tc.stopCh == nil {
		t.Fatal("expected non-nil stopCh")
	}
}
