package memory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func fakeEmbedResponse(t *testing.T, embedding []float32) []byte {
	t.Helper()
	resp := ollamaEmbedResp{Embedding: embedding}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal fake response: %v", err)
	}
	return b
}

func newTestEmbedder(t *testing.T, handler http.HandlerFunc) *Embedder {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return &Embedder{
		baseURL: srv.URL,
		model:   "test-model",
		client:  srv.Client(),
		cache:   newEmbeddingCache(1000, 5*time.Minute),
	}
}

func TestCacheHitReturnsSameEmbedding(t *testing.T) {
	callCount := 0
	emb := newTestEmbedder(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.Write(fakeEmbedResponse(t, []float32{0.1, 0.2, 0.3}))
	}))

	ctx := context.Background()

	result1, err := emb.Embed(ctx, "hello world")
	if err != nil {
		t.Fatalf("first Embed: %v", err)
	}
	result2, err := emb.Embed(ctx, "hello world")
	if err != nil {
		t.Fatalf("second Embed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("expected 1 HTTP call, got %d", callCount)
	}
	if len(result1) != len(result2) {
		t.Fatalf("expected same length, got %d vs %d", len(result1), len(result2))
	}
	for i := range result1 {
		if result1[i] != result2[i] {
			t.Fatalf("expected identical embeddings at index %d: %f vs %f", i, result1[i], result2[i])
		}
	}
}

func TestCacheNormalizesKey(t *testing.T) {
	callCount := 0
	emb := newTestEmbedder(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.Write(fakeEmbedResponse(t, []float32{0.5}))
	}))

	ctx := context.Background()

	_, _ = emb.Embed(ctx, "Hello World")
	_, _ = emb.Embed(ctx, "  hello world  ")
	_, _ = emb.Embed(ctx, "HELLO WORLD")

	if callCount != 1 {
		t.Errorf("expected 1 HTTP call with normalized keys, got %d", callCount)
	}
}

func TestCacheLRUEviction(t *testing.T) {
	callCount := 0
	emb := newTestEmbedder(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.Write(fakeEmbedResponse(t, []float32{float32(callCount)}))
	}))
	emb.cache = newEmbeddingCache(2, 5*time.Minute)

	ctx := context.Background()

	_, _ = emb.Embed(ctx, "a")
	_, _ = emb.Embed(ctx, "b")
	_, _ = emb.Embed(ctx, "c")

	emb.cache.mu.RLock()
	_, hasA := emb.cache.entries[normalizeKey("a")]
	_, hasB := emb.cache.entries[normalizeKey("b")]
	_, hasC := emb.cache.entries[normalizeKey("c")]
	emb.cache.mu.RUnlock()

	if hasA {
		t.Error("expected 'a' to be evicted (LRU)")
	}
	if !hasB {
		t.Error("expected 'b' to remain in cache")
	}
	if !hasC {
		t.Error("expected 'c' to remain in cache")
	}
}

func TestCacheExpireTriggersRefetch(t *testing.T) {
	callCount := 0
	emb := newTestEmbedder(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.Write(fakeEmbedResponse(t, []float32{float32(callCount)}))
	}))
	emb.cache = newEmbeddingCache(100, 50*time.Millisecond)

	ctx := context.Background()

	result1, _ := emb.Embed(ctx, "test text")
	if callCount != 1 {
		t.Errorf("expected 1 call after first embed, got %d", callCount)
	}

	time.Sleep(80 * time.Millisecond)

	result2, _ := emb.Embed(ctx, "test text")
	if callCount != 2 {
		t.Errorf("expected 2 calls after TTL expiry, got %d", callCount)
	}
	if result1[0] == result2[0] {
		t.Error("expected different embeddings after re-fetch, got same values")
	}
}

func TestCacheConcurrentAccess(t *testing.T) {
	emb := newTestEmbedder(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(fakeEmbedResponse(t, []float32{0.42}))
	}))
	emb.cache = newEmbeddingCache(100, 5*time.Minute)

	ctx := context.Background()
	var wg sync.WaitGroup
	const goroutines = 50

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := emb.Embed(ctx, "concurrent test")
			if err != nil {
				t.Errorf("concurrent Embed failed: %v", err)
			}
		}()
	}
	wg.Wait()
}

func TestNewEmbedderEnvVars(t *testing.T) {
	t.Setenv("EMBEDDING_CACHE_SIZE", "500")
	t.Setenv("EMBEDDING_CACHE_TTL_SECONDS", "60")

	emb := NewEmbedder("http://localhost:11434", "test-model")

	if emb.cache.maxSize != 500 {
		t.Errorf("expected cache maxSize 500, got %d", emb.cache.maxSize)
	}
	if emb.cache.ttl != 60*time.Second {
		t.Errorf("expected cache TTL 60s, got %v", emb.cache.ttl)
	}
}

func TestNewEmbedderDefaults(t *testing.T) {
	emb := NewEmbedder("http://localhost:11434", "")
	if emb.model != "nomic-embed-text" {
		t.Errorf("expected default model 'nomic-embed-text', got %q", emb.model)
	}
	if emb.cache.maxSize != 1000 {
		t.Errorf("expected default maxSize 1000, got %d", emb.cache.maxSize)
	}
	if emb.cache.ttl != 5*time.Minute {
		t.Errorf("expected default TTL 5m, got %v", emb.cache.ttl)
	}
}
