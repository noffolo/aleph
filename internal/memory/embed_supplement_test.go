package memory

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestChunk_EmptyText(t *testing.T) {
	chunks := Chunk("")
	if chunks != nil {
		t.Errorf("Chunk on empty string: expected nil, got %v", chunks)
	}
}

func TestChunk_SingleWord(t *testing.T) {
	chunks := Chunk("hello")
	if len(chunks) != 1 {
		t.Fatalf("Chunk single word: expected 1, got %d", len(chunks))
	}
	if chunks[0] != "hello" {
		t.Errorf("Chunk single word: expected 'hello', got %q", chunks[0])
	}
}

func TestChunk_ShortText(t *testing.T) {
	text := "the quick brown fox"
	chunks := Chunk(text)
	if len(chunks) != 1 {
		t.Fatalf("Chunk short text: expected 1, got %d", len(chunks))
	}
	if chunks[0] != text {
		t.Errorf("Chunk short text: expected %q, got %q", text, chunks[0])
	}
}

func TestChunk_LongText(t *testing.T) {
	words := make([]string, 1200)
	for i := range words {
		words[i] = "word"
	}
	text := strings.Join(words, " ")
	chunks := Chunk(text)

	if len(chunks) < 2 {
		t.Errorf("Chunk long text: expected at least 2 chunks, got %d", len(chunks))
	}

	for i, chunk := range chunks {
		wordCount := len(strings.Fields(chunk))
		if wordCount > chunkSize {
			t.Errorf("Chunk[%d]: expected at most %d words, got %d", i, chunkSize, wordCount)
		}
	}
}

func TestChunk_ExactBoundary(t *testing.T) {
	words := make([]string, chunkSize)
	for i := range words {
		words[i] = "w"
	}
	text := strings.Join(words, " ")
	chunks := Chunk(text)
	if len(chunks) != 1 {
		t.Errorf("Chunk exact boundary: expected 1 chunk, got %d", len(chunks))
	}
}

func TestChunk_BoundaryPlusOne(t *testing.T) {
	words := make([]string, chunkSize+1)
	for i := range words {
		words[i] = "w"
	}
	text := strings.Join(words, " ")
	chunks := Chunk(text)
	if len(chunks) != 2 {
		t.Errorf("Chunk boundary+1: expected 2 chunks, got %d", len(chunks))
	}
}

func TestChunk_WithOverlap(t *testing.T) {
	words := make([]string, 600)
	for i := range words {
		words[i] = "x"
	}
	text := strings.Join(words, " ")
	chunks := Chunk(text)

	if len(chunks) < 2 {
		t.Fatal("expected at least 2 chunks")
	}

	firstWords := strings.Fields(chunks[0])
	secondWords := strings.Fields(chunks[1])

	overlapCount := 0
	lastFirst := firstWords[len(firstWords)-1]
	for _, w := range secondWords {
		if w == lastFirst {
			overlapCount++
		}
	}
	if overlapCount == 0 {
		t.Error("expected overlap between consecutive chunks")
	}
}

func TestNormalizeKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"  Hello  ", "hello"},
		{"HELLO", "hello"},
		{"  \t  HeLLo World \n  ", "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeKey(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeKey(%q): expected %q, got %q", tt.input, tt.expected, got)
			}
		})
	}
}

func TestEmbeddingCache_PutAndGet(t *testing.T) {
	c := newEmbeddingCache(10, time.Minute)
	key := "test-key"
	emb := []float32{0.1, 0.2, 0.3}

	c.put(key, emb)
	got, ok := c.get(key)
	if !ok {
		t.Fatal("expected cache hit after put")
	}
	if len(got) != len(emb) {
		t.Fatalf("expected len %d, got %d", len(emb), len(got))
	}
	for i := range emb {
		if got[i] != emb[i] {
			t.Errorf("index %d: expected %f, got %f", i, emb[i], got[i])
		}
	}
}

func TestEmbeddingCache_NotFound(t *testing.T) {
	c := newEmbeddingCache(10, time.Minute)
	_, ok := c.get("missing-key")
	if ok {
		t.Error("expected cache miss for nonexistent key")
	}
}

func TestEmbeddingCache_Expired(t *testing.T) {
	c := newEmbeddingCache(10, 10*time.Millisecond)
	c.put("expiring", []float32{0.5})

	time.Sleep(15 * time.Millisecond)

	_, ok := c.get("expiring")
	if ok {
		t.Error("expected cache miss for expired entry")
	}
}

func TestEmbeddingCache_UpdateExisting(t *testing.T) {
	c := newEmbeddingCache(5, time.Minute)
	c.put("key", []float32{0.1})
	c.put("key", []float32{0.2, 0.3})

	got, ok := c.get("key")
	if !ok {
		t.Fatal("expected cache hit after update")
	}
	if len(got) != 2 {
		t.Fatalf("expected len 2 after update, got %d", len(got))
	}
	if got[0] != 0.2 || got[1] != 0.3 {
		t.Errorf("expected [0.2, 0.3], got %v", got)
	}
}

func TestEmbeddingCache_MoveToEndOnUpdate(t *testing.T) {
	c := newEmbeddingCache(3, time.Minute)
	c.put("a", []float32{1})
	c.put("b", []float32{2})
	c.put("a", []float32{1.1})
	c.put("c", []float32{3})

	_, hasA := c.get("a")
	_, hasB := c.get("b")
	_, hasC := c.get("c")

	if !hasA {
		t.Error("'a' should remain (updated, moved to end)")
	}
	if !hasB {
		t.Error("'b' should remain (not oldest after 'a' moved)")
	}
	if !hasC {
		t.Error("'c' should remain")
	}
}

func TestEmbeddingCache_LRUEvictionMaxSize(t *testing.T) {
	c := newEmbeddingCache(2, time.Minute)
	c.put("first", []float32{1})
	c.put("second", []float32{2})
	c.put("third", []float32{3})

	_, hasFirst := c.get("first")
	_, hasSecond := c.get("second")
	_, hasThird := c.get("third")

	if hasFirst {
		t.Error("'first' should be evicted (LRU)")
	}
	if !hasSecond {
		t.Error("'second' should remain")
	}
	if !hasThird {
		t.Error("'third' should remain")
	}
}

func TestNewEmbeddingCache_ZeroSizeDefaults(t *testing.T) {
	c := newEmbeddingCache(0, 0)
	if c.maxSize != 1000 {
		t.Errorf("expected default maxSize 1000, got %d", c.maxSize)
	}
	if c.ttl != 5*time.Minute {
		t.Errorf("expected default TTL 5m, got %v", c.ttl)
	}
}

func TestNewEmbeddingCache_NegativeSizeDefaults(t *testing.T) {
	c := newEmbeddingCache(-5, -100*time.Second)
	if c.maxSize != 1000 {
		t.Errorf("expected default maxSize 1000, got %d", c.maxSize)
	}
	if c.ttl != 5*time.Minute {
		t.Errorf("expected default TTL 5m, got %v", c.ttl)
	}
}

func TestProcessText_EmptyInput(t *testing.T) {
	emb := &Embedder{cache: newEmbeddingCache(100, time.Minute)}
	chunks, embeddings, err := emb.ProcessText(context.Background(), "")
	if err != ErrEmptyInput {
		t.Errorf("ProcessText empty: expected ErrEmptyInput, got %v", err)
	}
	if chunks != nil {
		t.Error("chunks should be nil for empty input")
	}
	if embeddings != nil {
		t.Error("embeddings should be nil for empty input")
	}
}

func TestEmbedder_BaseURLTrimmed(t *testing.T) {
	t.Setenv("EMBEDDING_CACHE_SIZE", "100")
	t.Setenv("EMBEDDING_CACHE_TTL_SECONDS", "30")
	emb := NewEmbedder("http://localhost:11434/", "model")
	if emb.baseURL != "http://localhost:11434" {
		t.Errorf("expected trailing slash trimmed, got %q", emb.baseURL)
	}
}

func TestEmbedder_BaseURLNoTrailingSlash(t *testing.T) {
	t.Setenv("EMBEDDING_CACHE_SIZE", "100")
	t.Setenv("EMBEDDING_CACHE_TTL_SECONDS", "30")
	emb := NewEmbedder("http://localhost:11434", "model")
	if emb.baseURL != "http://localhost:11434" {
		t.Errorf("expected unchanged baseURL, got %q", emb.baseURL)
	}
}

func TestEmbedder_InvalidEnvCacheSize(t *testing.T) {
	t.Setenv("EMBEDDING_CACHE_SIZE", "not-a-number")
	t.Setenv("EMBEDDING_CACHE_TTL_SECONDS", "30")
	emb := NewEmbedder("http://localhost:11434", "")
	if emb.cache.maxSize != 1000 {
		t.Errorf("expected default cache size 1000 when env var is invalid, got %d", emb.cache.maxSize)
	}
}

func TestEmbedder_InvalidEnvTTL(t *testing.T) {
	t.Setenv("EMBEDDING_CACHE_SIZE", "100")
	t.Setenv("EMBEDDING_CACHE_TTL_SECONDS", "not-a-number")
	emb := NewEmbedder("http://localhost:11434", "")
	if emb.cache.ttl != 5*time.Minute {
		t.Errorf("expected default TTL 5m when env var is invalid, got %v", emb.cache.ttl)
	}
}

func TestEmbedder_NegativeEnvCacheSize(t *testing.T) {
	t.Setenv("EMBEDDING_CACHE_SIZE", "-50")
	t.Setenv("EMBEDDING_CACHE_TTL_SECONDS", "30")
	emb := NewEmbedder("http://localhost:11434", "")
	if emb.cache.maxSize != 1000 {
		t.Errorf("expected default cache size 1000 when env var is negative, got %d", emb.cache.maxSize)
	}
}

func TestEmbedder_NegativeEnvTTL(t *testing.T) {
	t.Setenv("EMBEDDING_CACHE_SIZE", "100")
	t.Setenv("EMBEDDING_CACHE_TTL_SECONDS", "-30")
	emb := NewEmbedder("http://localhost:11434", "")
	if emb.cache.ttl != 5*time.Minute {
		t.Errorf("expected default TTL 5m when env var is negative, got %v", emb.cache.ttl)
	}
}

func TestOllamaEmbedReq_JSON(t *testing.T) {
	req := ollamaEmbedReq{
		Model:  "test-model",
		Prompt: "test prompt",
	}
	if req.Model != "test-model" {
		t.Error("model field mismatch")
	}
	if req.Prompt != "test prompt" {
		t.Error("prompt field mismatch")
	}
}
