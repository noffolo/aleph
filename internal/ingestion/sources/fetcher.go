// Package sources implements W3 ingestion methods: rate-limited HTTP fetcher,
// chunker/worker pool, GitHub repos, sitemap XML, ProbeRunner, generic JSON/API, Google Sheets.
package sources

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ff3300/aleph-v2/internal/ssrf"
	"golang.org/x/time/rate"
)

// =============================================================================
// W3-06: Rate Limiter + Chunker
// =============================================================================

// RateLimitConfig defines per-source rate limits.
type RateLimitConfig struct {
	RequestsPerSecond float64
	Burst             int
}

var (
	// DefaultRate is a safe default (10 req/s, burst 20).
	DefaultRate = RateLimitConfig{RequestsPerSecond: 10, Burst: 20}
	// GitHubRate matches GitHub API limits (60 req/min for unauthenticated, 5000/hr authenticated).
	GitHubRate = RateLimitConfig{RequestsPerSecond: 2, Burst: 5}
	// GoogleRate for Google APIs.
	GoogleRate = RateLimitConfig{RequestsPerSecond: 10, Burst: 20}
)

// RateLimitedClient wraps http.Client with per-source rate limiting.
type RateLimitedClient struct {
	client  *http.Client
	limiter *rate.Limiter
}

// NewRateLimitedClient creates a new rate-limited HTTP client.
// The client reuses the global SSRF-safe transport.
// If cfg has zero rate, DefaultRate is used.
func NewRateLimitedClient(cfg RateLimitConfig) *RateLimitedClient {
	if cfg.RequestsPerSecond <= 0 {
		cfg = DefaultRate
	}
	burst := cfg.Burst
	if burst <= 0 {
		burst = int(cfg.RequestsPerSecond) * 2
		if burst < 1 {
			burst = 1
		}
	}
	return &RateLimitedClient{
		client:  ssrf.NewClient(),
		limiter: rate.NewLimiter(rate.Limit(cfg.RequestsPerSecond), burst),
	}
}

// Do waits for the rate limiter then executes the request.
func (c *RateLimitedClient) Do(req *http.Request) (*http.Response, error) {
	if err := c.limiter.Wait(req.Context()); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}
	return c.client.Do(req)
}

// Get issues a GET request through the rate limiter.
func (c *RateLimitedClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// =============================================================================
// W3-06: Chunker / Worker Pool
// =============================================================================

// ChunkConfig defines worker pool behaviour.
type ChunkConfig struct {
	Workers    int
	BatchSize  int
	MaxRetries int
	RetryDelay time.Duration
}

// DefaultChunkConfig is a safe default.
var DefaultChunkConfig = ChunkConfig{
	Workers:    4,
	BatchSize:  500,
	MaxRetries: 3,
	RetryDelay: time.Second,
}

// ChunkJob is a unit of work for the chunker pool.
type ChunkJob struct {
	Index int
	Data  []byte
}

// ChunkResult is the output of a single chunk job.
type ChunkResult struct {
	Index int
	Err   error
}

// WorkerPool processes ChunkJobs concurrently using a fixed pool.
type WorkerPool struct {
	config ChunkConfig
}

// NewWorkerPool creates a pool with the given config.
func NewWorkerPool(cfg ChunkConfig) *WorkerPool {
	if cfg.Workers <= 0 {
		cfg = DefaultChunkConfig
	}
	return &WorkerPool{config: cfg}
}

// Run sends jobs to the worker pool.
// processFn is called per job and must be idempotent (for retries).
func (wp *WorkerPool) Run(ctx context.Context, jobs []ChunkJob, processFn func(context.Context, ChunkJob) error) error {
	if len(jobs) == 0 {
		return nil
	}

	jobCh := make(chan ChunkJob, len(jobs))
	resultCh := make(chan ChunkResult, len(jobs))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Enqueue all jobs
	for _, j := range jobs {
		jobCh <- j
	}
	close(jobCh)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < wp.config.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobCh {
				select {
				case <-ctx.Done():
					resultCh <- ChunkResult{Index: j.Index, Err: ctx.Err()}
					return
				default:
				}
				var lastErr error
				for attempt := 0; attempt <= wp.config.MaxRetries; attempt++ {
					if attempt > 0 {
						select {
						case <-ctx.Done():
							lastErr = ctx.Err()
							goto done
						case <-time.After(wp.config.RetryDelay):
						}
					}
					if err := processFn(ctx, j); err != nil {
						lastErr = err
						log.Printf("[WorkerPool] chunk %d attempt %d failed: %v", j.Index, attempt+1, err)
						continue
					}
					lastErr = nil
					break
				}
			done:
				resultCh <- ChunkResult{Index: j.Index, Err: lastErr}
			}
		}()
	}

	// Wait for all workers
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	var firstErr error
	for r := range resultCh {
		if r.Err != nil && firstErr == nil {
			firstErr = r.Err
		}
	}
	return firstErr
}

// =============================================================================
// W3-06: Paginated Fetch Helpers
// =============================================================================

// FetchPages fetches all pages from a paginated JSON API.
// nextURLFn extracts the next page URL from a response (return "" to stop).
// Each response body is passed to consumeFn.
func FetchPages(ctx context.Context, client *RateLimitedClient, initialURL string,
	headers map[string]string,
	nextURLFn func(body []byte) (nextURL string),
	consumeFn func(body []byte) error) error {

	url := initialURL
	for url != "" {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("fetch %s: %w", url, err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("read body %s: %w", url, err)
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("HTTP %d fetching %s: %s", resp.StatusCode, url, string(body[:min(len(body), 500)]))
		}

		if err := consumeFn(body); err != nil {
			return fmt.Errorf("consume %s: %w", url, err)
		}

		url = ""
		if nextURLFn != nil {
			url = nextURLFn(body)
		}
	}
	return nil
}
