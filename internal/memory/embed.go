package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ff3300/aleph-v2/internal/ssrf"
)

// ErrEmptyInput is returned by ProcessText when the input text is empty.
var ErrEmptyInput = errors.New("empty input text")

const (
	chunkSize    = 512
	chunkOverlap = 128
	embedTimeout = 30 * time.Second
)

type ollamaEmbedReq struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type ollamaEmbedResp struct {
	Embedding []float32 `json:"embedding"`
}

// cacheEntry holds a cached embedding with an expiration time.
type cacheEntry struct {
	embedding []float32
	expiresAt time.Time
}

// embeddingCache provides a thread-safe, TTL+LRU in-memory cache for embeddings.
type embeddingCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	order   []string // LRU eviction list; oldest first
	maxSize int
	ttl     time.Duration
}

// newEmbeddingCache creates a cache with the given max size and TTL.
func newEmbeddingCache(maxSize int, ttl time.Duration) *embeddingCache {
	if maxSize <= 0 {
		maxSize = 1000
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &embeddingCache{
		entries: make(map[string]*cacheEntry),
		order:   make([]string, 0, maxSize),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// normalizeKey produces a canonical cache key from input text.
func normalizeKey(text string) string {
	return strings.TrimSpace(strings.ToLower(text))
}

// get returns the cached embedding if present and not expired.
func (c *embeddingCache) get(key string) ([]float32, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.embedding, true
}

// put stores an embedding and evicts the LRU entry if over maxSize.
func (c *embeddingCache) put(key string, embedding []float32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// If key already exists, just update and move to end.
	if _, exists := c.entries[key]; exists {
		c.entries[key] = &cacheEntry{embedding: embedding, expiresAt: time.Now().Add(c.ttl)}
		// Move key to end of LRU order.
		for i, k := range c.order {
			if k == key {
				c.order = append(c.order[:i], c.order[i+1:]...)
				break
			}
		}
		c.order = append(c.order, key)
		return
	}
	// Evict oldest entries while over capacity.
	for len(c.entries) >= c.maxSize && len(c.order) > 0 {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.entries, oldest)
	}
	c.entries[key] = &cacheEntry{embedding: embedding, expiresAt: time.Now().Add(c.ttl)}
	c.order = append(c.order, key)
}

// Embedder generates embeddings via Ollama's embedding API.
type Embedder struct {
	baseURL string
	model   string
	client  *http.Client
	cache   *embeddingCache
}

// NewEmbedder creates an Embedder that calls Ollama at the given base URL with the given model.
// Cache size and TTL are configurable via EMBEDDING_CACHE_SIZE and EMBEDDING_CACHE_TTL_SECONDS env vars.
func NewEmbedder(baseURL, model string) *Embedder {
	if model == "" {
		model = "nomic-embed-text"
	}

	maxSize := 1000
	if v := os.Getenv("EMBEDDING_CACHE_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxSize = n
		}
	}

	ttl := 5 * time.Minute
	if v := os.Getenv("EMBEDDING_CACHE_TTL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			ttl = time.Duration(n) * time.Second
		}
	}

	return &Embedder{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		client:  ssrf.NewClient(),
		cache:   newEmbeddingCache(maxSize, ttl),
	}
}

// Embed generates an embedding vector for the given text, with TTL+LRU caching.
func (e *Embedder) Embed(ctx context.Context, text string) ([]float32, error) {
	key := normalizeKey(text)
	if cached, ok := e.cache.get(key); ok {
		slog.Debug("embedding cache HIT", "key_len", len(key))
		return cached, nil
	}
	slog.Debug("embedding cache MISS", "key_len", len(key))

	body := ollamaEmbedReq{Model: e.model, Prompt: text}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal embed request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/api/embeddings", bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("create embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embed request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embed API status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result ollamaEmbedResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode embed response: %w", err)
	}
	if len(result.Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding returned")
	}

	e.cache.put(key, result.Embedding)
	return result.Embedding, nil
}

// Chunk splits text into overlapping chunks of approximately chunkSize tokens.
func Chunk(text string) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var chunks []string
	start := 0
	for start < len(words) {
		end := start + chunkSize
		if end > len(words) {
			end = len(words)
		}
		chunk := strings.Join(words[start:end], " ")
		chunks = append(chunks, chunk)

		if end == len(words) {
			break
		}
		start = end - chunkOverlap
		if start < 0 {
			start = 0
		}
	}
	return chunks
}

// ProcessText splits text into chunks, embeds each, and returns chunk-embedding pairs.
func (e *Embedder) ProcessText(ctx context.Context, text string) ([]string, [][]float32, error) {
	chunks := Chunk(text)
	if len(chunks) == 0 {
		return nil, nil, ErrEmptyInput
	}

	embeddings := make([][]float32, 0, len(chunks))
	validChunks := make([]string, 0, len(chunks))

	for _, chunk := range chunks {
		emb, err := e.Embed(ctx, chunk)
		if err != nil {
			slog.Warn("embedding chunk failed, skipping", "error", err, "chunk_len", len(chunk))
			continue
		}
		embeddings = append(embeddings, emb)
		validChunks = append(validChunks, chunk)
	}

	return validChunks, embeddings, nil
}
