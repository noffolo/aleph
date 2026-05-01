package sources

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRateLimitedClient(t *testing.T) {
	cfg := RateLimitConfig{
		RequestsPerSecond: 100,
		Burst:             10,
	}
	client := NewRateLimitedClient(cfg)
	require.NotNil(t, client, "client should not be nil")
	assert.NotNil(t, client.limiter, "limiter should be initialized")
	assert.NotNil(t, client.client, "http client should be initialized")
}

func TestNewRateLimitedClient_DefaultRate(t *testing.T) {
	cfg := RateLimitConfig{
		RequestsPerSecond: 0,
		Burst:             0,
	}
	client := NewRateLimitedClient(cfg)
	require.NotNil(t, client, "client should not be nil with zero config")
	assert.NotNil(t, client.limiter, "limiter should be initialized with defaults")
}

func TestNewWorkerPool(t *testing.T) {
	cfg := ChunkConfig{
		Workers:    4,
		BatchSize:  100,
		MaxRetries: 2,
		RetryDelay: time.Second,
	}
	pool := NewWorkerPool(cfg)
	require.NotNil(t, pool, "pool should not be nil")
	assert.Equal(t, cfg.Workers, pool.config.Workers)
}

func TestWorkerPool_Run(t *testing.T) {
	pool := NewWorkerPool(ChunkConfig{
		Workers:    2,
		BatchSize:  10,
		MaxRetries: 0,
		RetryDelay: time.Millisecond,
	})

	jobs := []ChunkJob{
		{Index: 0, Data: []byte("a")},
		{Index: 1, Data: []byte("b")},
		{Index: 2, Data: []byte("c")},
	}

	results := make([]int, len(jobs))
	var mu sync.Mutex

	ctx := t.Context()
	err := pool.Run(ctx, jobs, func(ctx context.Context, job ChunkJob) error {
		mu.Lock()
		results[job.Index] = 1
		mu.Unlock()
		return nil
	})

	require.NoError(t, err, "Run should not error")
	assert.Equal(t, []int{1, 1, 1}, results, "all jobs should be processed")
}

func TestWorkerPool_Run_EmptyJobs(t *testing.T) {
	pool := NewWorkerPool(DefaultChunkConfig)
	ctx := t.Context()
	err := pool.Run(ctx, []ChunkJob{}, func(ctx context.Context, job ChunkJob) error {
		return nil
	})
	require.NoError(t, err, "Run with empty jobs should not error")
}

func TestNewSitemapIngester(t *testing.T) {
	ingester := NewSitemapIngester()
	require.NotNil(t, ingester, "ingester should not be nil")
	assert.NotNil(t, ingester.client, "client should be initialized")
}

func TestDetectRootElement(t *testing.T) {
	tests := []struct {
		name     string
		xml      string
		expected string
		wantErr  bool
	}{
		{
			name:     "urlset",
			xml:      `<?xml version="1.0"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>https://example.com/</loc></url></urlset>`,
			expected: "urlset",
			wantErr:  false,
		},
		{
			name:     "sitemapindex",
			xml:      `<?xml version="1.0"?><sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><sitemap><loc>https://example.com/sitemap.xml</loc></sitemap></sitemapindex>`,
			expected: "sitemapindex",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, err := detectRootElement([]byte(tt.xml))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, root)
			}
		})
	}
}

func TestIsAllowedContentType(t *testing.T) {
	tests := []struct {
		name      string
		mediaType string
		expected  bool
	}{
		{"text/html", "text/html", true},
		{"text/plain", "text/plain", true},
		{"application/json", "application/json", true},
		{"application/xml", "application/xml", true},
		{"text/xml", "text/xml", true},
		{"image/jpeg", "image/jpeg", false},
		{"application/pdf", "application/pdf", false},
		{"application/atom+xml", "application/atom+xml", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAllowedContentType(tt.mediaType)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestResolveURL(t *testing.T) {
	tests := []struct {
		name    string
		base    string
		rawURL  string
		want    string
		wantErr bool
	}{
		{
			name:   "absolute URL",
			base:   "https://example.com/sitemap.xml",
			rawURL: "https://other.com/page.html",
			want:   "https://other.com/page.html",
		},
		{
			name:   "relative URL",
			base:   "https://example.com/sitemap.xml",
			rawURL: "/page.html",
			want:   "https://example.com/page.html",
		},
		{
			name:   "relative URL with path",
			base:   "https://example.com/dir/sitemap.xml",
			rawURL: "page.html",
			want:   "https://example.com/dir/page.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveURL(tt.base, tt.rawURL)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestRateLimitConfig_Struct(t *testing.T) {
	cfg := RateLimitConfig{
		RequestsPerSecond: 50,
		Burst:             20,
	}
	assert.Equal(t, 50.0, cfg.RequestsPerSecond)
	assert.Equal(t, 20, cfg.Burst)
}

func TestChunkJob_Struct(t *testing.T) {
	job := ChunkJob{
		Index: 42,
		Data:  []byte("test data"),
	}
	assert.Equal(t, 42, job.Index)
	assert.Equal(t, []byte("test data"), job.Data)
}

func TestChunkResult_Struct(t *testing.T) {
	result := ChunkResult{
		Index: 1,
		Err:   nil,
	}
	assert.Equal(t, 1, result.Index)
	assert.Nil(t, result.Err)
}

func TestCrawlResult_Struct(t *testing.T) {
	result := CrawlResult{
		SitemapURL: "https://example.com/sitemap.xml",
		URLs: []PageResult{
			{URL: "https://example.com/page1", Status: 200},
		},
	}
	assert.Equal(t, "https://example.com/sitemap.xml", result.SitemapURL)
	assert.Len(t, result.URLs, 1)
}

func TestPageResult_Struct(t *testing.T) {
	result := PageResult{
		URL:     "https://example.com/page",
		Content: []byte("content"),
		Size:    100,
		Status:  200,
		Err:     nil,
	}
	assert.Equal(t, "https://example.com/page", result.URL)
	assert.Equal(t, int64(100), result.Size)
	assert.Equal(t, 200, result.Status)
}

func TestSitemapIndex_Struct(t *testing.T) {
	index := SitemapIndex{
		Sitemaps: []SitemapEntry{
			{Loc: "https://example.com/sitemap1.xml", LastMod: "2024-01-01"},
		},
	}
	assert.Len(t, index.Sitemaps, 1)
	assert.Equal(t, "https://example.com/sitemap1.xml", index.Sitemaps[0].Loc)
}

func TestURLSet_Struct(t *testing.T) {
	urlset := URLSet{
		URLs: []URLEntry{
			{Loc: "https://example.com/", LastMod: "2024-01-01", ChangeFreq: "daily", Priority: "1.0"},
		},
	}
	assert.Len(t, urlset.URLs, 1)
	assert.Equal(t, "https://example.com/", urlset.URLs[0].Loc)
}
