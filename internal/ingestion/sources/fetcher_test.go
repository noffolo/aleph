package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── NewRateLimitedClient ────────────────────────────────────────────────────

func TestNewRateLimitedClient_HappyPath(t *testing.T) {
	cfg := RateLimitConfig{RequestsPerSecond: 50, Burst: 10}
	c := NewRateLimitedClient(cfg)
	require.NotNil(t, c)
	assert.NotNil(t, c.client)
	assert.NotNil(t, c.limiter)
}

func TestNewRateLimitedClient_DefaultRate_ZeroRate(t *testing.T) {
	cfg := RateLimitConfig{RequestsPerSecond: 0, Burst: 5}
	c := NewRateLimitedClient(cfg)
	require.NotNil(t, c)
}

func TestNewRateLimitedClient_ZeroBurst_New(t *testing.T) {
	cfg := RateLimitConfig{RequestsPerSecond: 10, Burst: 0}
	c := NewRateLimitedClient(cfg)
	require.NotNil(t, c)
	assert.NotNil(t, c.limiter)
}

func TestNewRateLimitedClient_NegativeBurst(t *testing.T) {
	cfg := RateLimitConfig{RequestsPerSecond: 10, Burst: -1}
	c := NewRateLimitedClient(cfg)
	require.NotNil(t, c)
}

func TestNewRateLimitedClient_LowRateSmallBurst(t *testing.T) {
	cfg := RateLimitConfig{RequestsPerSecond: 0.5, Burst: 0}
	c := NewRateLimitedClient(cfg)
	require.NotNil(t, c)
}

// ─── RateLimitedClient.Do ────────────────────────────────────────────────────

func TestRateLimitedClient_Do_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer srv.Close()

	c := NewTestRateLimitedClient()
	req, err := http.NewRequest("GET", srv.URL, nil)
	require.NoError(t, err)

	resp, err := c.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestRateLimitedClient_Do_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewTestRateLimitedClient()
	req, err := http.NewRequest("GET", srv.URL, nil)
	require.NoError(t, err)

	resp, err := c.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}

func TestRateLimitedClient_Do_CancelledContext_New(t *testing.T) {
	c := NewTestRateLimitedClient()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://127.0.0.1:1/nope", nil)
	require.NoError(t, err)

	_, err = c.Do(req)
	require.Error(t, err)
}

// ─── RateLimitedClient.Get ───────────────────────────────────────────────────

func TestRateLimitedClient_Get_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": "ok"}`))
	}))
	defer srv.Close()

	c := NewTestRateLimitedClient()
	resp, err := c.Get(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestRateLimitedClient_Get_InvalidURL(t *testing.T) {
	c := NewTestRateLimitedClient()
	_, err := c.Get(context.Background(), "://invalid-url")
	require.Error(t, err)
}

func TestRateLimitedClient_Get_CancelledContext(t *testing.T) {
	c := NewTestRateLimitedClient()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.Get(ctx, "http://example.com")
	require.Error(t, err)
}

// ─── NewWorkerPool ───────────────────────────────────────────────────────────

func TestNewWorkerPool_HappyPath(t *testing.T) {
	cfg := ChunkConfig{Workers: 4, BatchSize: 100, MaxRetries: 2}
	pool := NewWorkerPool(cfg)
	require.NotNil(t, pool)
	assert.Equal(t, 4, pool.config.Workers)
	assert.Equal(t, 100, pool.config.BatchSize)
}

func TestNewWorkerPool_ZeroWorkersDefaults(t *testing.T) {
	pool := NewWorkerPool(ChunkConfig{Workers: 0, BatchSize: 10, MaxRetries: 0})
	require.NotNil(t, pool)
	assert.Equal(t, DefaultChunkConfig.Workers, pool.config.Workers)
}

func TestNewWorkerPool_NegativeConfig(t *testing.T) {
	pool := NewWorkerPool(ChunkConfig{Workers: -1})
	require.NotNil(t, pool)
	assert.Greater(t, pool.config.Workers, 0)
}

// ─── WorkerPool.Run ──────────────────────────────────────────────────────────

func TestWorkerPool_Run_HappyPath(t *testing.T) {
	pool := NewWorkerPool(ChunkConfig{Workers: 2, BatchSize: 10, MaxRetries: 0})
	jobs := []ChunkJob{
		{Index: 0, Data: []byte("a")},
		{Index: 1, Data: []byte("b")},
		{Index: 2, Data: []byte("c")},
	}

	processed := make([]bool, len(jobs))
	ctx := context.Background()
	err := pool.Run(ctx, jobs, func(ctx context.Context, job ChunkJob) error {
		processed[job.Index] = true
		return nil
	})
	require.NoError(t, err)
	for i, p := range processed {
		assert.True(t, p, "job %d was not processed", i)
	}
}

func TestWorkerPool_Run_EmptyJobs_New(t *testing.T) {
	pool := NewWorkerPool(DefaultChunkConfig)
	ctx := context.Background()
	err := pool.Run(ctx, []ChunkJob{}, func(ctx context.Context, job ChunkJob) error {
		return nil
	})
	require.NoError(t, err)
}

func TestWorkerPool_Run_WithRetries_New(t *testing.T) {
	pool := NewWorkerPool(ChunkConfig{
		Workers:    1,
		BatchSize:  10,
		MaxRetries: 2,
		RetryDelay: 1,
	})

	jobs := []ChunkJob{{Index: 0, Data: []byte("a")}}
	attempts := 0

	ctx := context.Background()
	err := pool.Run(ctx, jobs, func(ctx context.Context, job ChunkJob) error {
		attempts++
		if attempts <= 2 {
			return assert.AnError
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestWorkerPool_Run_CtxCancellation_New(t *testing.T) {
	pool := NewWorkerPool(ChunkConfig{Workers: 2, BatchSize: 10, MaxRetries: 0})
	jobs := []ChunkJob{
		{Index: 0, Data: []byte("x")},
		{Index: 1, Data: []byte("y")},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := pool.Run(ctx, jobs, func(ctx context.Context, job ChunkJob) error {
		return nil
	})
	assert.Error(t, err)
}

func TestWorkerPool_Run_ErrorPropagation(t *testing.T) {
	pool := NewWorkerPool(ChunkConfig{Workers: 2, BatchSize: 10, MaxRetries: 0})
	jobs := []ChunkJob{
		{Index: 0, Data: []byte("fail")},
		{Index: 1, Data: []byte("ok")},
	}

	ctx := context.Background()
	err := pool.Run(ctx, jobs, func(ctx context.Context, job ChunkJob) error {
		if string(job.Data) == "fail" {
			return assert.AnError
		}
		return nil
	})
	assert.Error(t, err)
}

// ─── FetchPages ──────────────────────────────────────────────────────────────

func TestFetchPages_HappyPath_SinglePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id": 1}]`))
	}))
	defer srv.Close()

	c := NewTestRateLimitedClient()
	var pages [][]byte
	err := FetchPages(context.Background(), c, srv.URL, nil,
		func(body []byte) string { return "" },
		func(body []byte) error {
			pages = append(pages, body)
			return nil
		},
	)
	require.NoError(t, err)
	assert.Len(t, pages, 1)
}

func TestFetchPages_HTTPError_New(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`error`))
	}))
	defer srv.Close()

	c := NewTestRateLimitedClient()
	err := FetchPages(context.Background(), c, srv.URL, nil, nil,
		func(body []byte) error { return nil },
	)
	assert.Error(t, err)
}

func TestFetchPages_InvalidURL_New(t *testing.T) {
	c := NewTestRateLimitedClient()
	err := FetchPages(context.Background(), c, "://invalid", nil, nil,
		func(body []byte) error { return nil },
	)
	assert.Error(t, err)
}

func TestFetchPages_MultiPage_New(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"items": [{}]}`))
	}))
	defer srv.Close()

	c := NewTestRateLimitedClient()
	var pages [][]byte
	pagesRequested := 0
	err := FetchPages(context.Background(), c, srv.URL, nil,
		func(body []byte) string {
			pagesRequested++
			if pagesRequested < 3 {
				return srv.URL
			}
			return ""
		},
		func(body []byte) error {
			pages = append(pages, body)
			return nil
		},
	)
	require.NoError(t, err)
	assert.Len(t, pages, 3)
}

func TestFetchPages_ConsumeError_New(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id": 1}]`))
	}))
	defer srv.Close()

	c := NewTestRateLimitedClient()
	err := FetchPages(context.Background(), c, srv.URL, nil,
		func(body []byte) string { return "" },
		func(body []byte) error {
			return assert.AnError
		},
	)
	assert.Error(t, err)
}

func TestFetchPages_WithHeaders_New(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "value", r.Header.Get("X-Custom"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := NewTestRateLimitedClient()
	err := FetchPages(context.Background(), c, srv.URL,
		map[string]string{"X-Custom": "value"},
		func(body []byte) string { return "" },
		func(body []byte) error { return nil },
	)
	require.NoError(t, err)
}
