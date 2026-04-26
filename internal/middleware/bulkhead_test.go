package middleware

import (
	"context"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBulkheadInterceptor_WrapUnary(t *testing.T) {
	config := &BulkheadConfig{
		QueryPoolSize:     2,
		IngestionPoolSize: 1,
		ChatPoolSize:      1,
	}
	interceptor := NewBulkheadInterceptor(config)

	tests := []struct {
		name          string
		procedure     string
		concurrent    int
		expectSuccess int
		expectError   int
	}{
		{
			name:          "QueryService accepts up to 2 concurrent",
			procedure:     "/aleph.v1.QueryService/GetChatHistory",
			concurrent:    3,
			expectSuccess: 2,
			expectError:   1,
		},
		{
			name:          "IngestionService accepts only 1 concurrent",
			procedure:     "/aleph.v1.IngestionService/IngestURL",
			concurrent:    2,
			expectSuccess: 1,
			expectError:   1,
		},
		{
			name:          "NLPService uses chat pool (1 concurrent)",
			procedure:     "/aleph.nlp.v1.NLPService/AnalyzeSentiment",
			concurrent:    2,
			expectSuccess: 1,
			expectError:   1,
		},
		{
			name:          "Unknown service no bulkhead",
			procedure:     "/aleph.v1.UnknownService/UnknownMethod",
			concurrent:    5,
			expectSuccess: 5,
			expectError:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := interceptor.WrapUnary(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				time.Sleep(50 * time.Millisecond)
				return nil, nil
			})

			var wg sync.WaitGroup
			successCount := 0
			errorCount := 0
			var mu sync.Mutex

			for i := 0; i < tt.concurrent; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_, err := handler(context.Background(), newMockRequest(tt.procedure))

					mu.Lock()
					if err == nil {
						successCount++
					} else {
						errorCount++
						assert.Contains(t, err.Error(), "service overloaded")
					}
					mu.Unlock()
				}()
			}

			wg.Wait()

			assert.Equal(t, tt.expectSuccess, successCount)
			assert.Equal(t, tt.expectError, errorCount)
		})
	}
}

func TestBulkheadDomainFromContext(t *testing.T) {
	interceptor := NewBulkheadInterceptor(nil)

	handler := interceptor.WrapUnary(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		domain := BulkheadDomainFromContext(ctx)

		switch req.Spec().Procedure {
		case "/aleph.v1.QueryService/GetChatHistory":
			assert.Equal(t, "query", domain)
		case "/aleph.v1.IngestionService/IngestURL":
			assert.Equal(t, "ingestion", domain)
		case "/aleph.nlp.v1.NLPService/AnalyzeSentiment":
			assert.Equal(t, "chat", domain)
		case "/aleph.v1.UnknownService/UnknownMethod":
			assert.Equal(t, "", domain)
		}
		return nil, nil
	})

	tests := []struct {
		procedure string
	}{
		{"/aleph.v1.QueryService/GetChatHistory"},
		{"/aleph.v1.IngestionService/IngestURL"},
		{"/aleph.nlp.v1.NLPService/AnalyzeSentiment"},
		{"/aleph.v1.UnknownService/UnknownMethod"},
	}

	for _, tt := range tests {
		_, err := handler(context.Background(), newMockRequest(tt.procedure))
		require.NoError(t, err)
	}
}

func TestBulkheadInterceptor_StreamingHandler(t *testing.T) {
	config := &BulkheadConfig{
		QueryPoolSize: 1,
	}
	interceptor := NewBulkheadInterceptor(config)

	handler := interceptor.WrapStreamingHandler(func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		domain := BulkheadDomainFromContext(ctx)
		assert.Equal(t, "query", domain)
		return nil
	})

	err := handler(context.Background(), &mockStreamingHandlerConn{procedure: "/aleph.v1.QueryService/StreamQuery"})
	require.NoError(t, err)
}

func TestBulkheadInterceptor_ConcurrentLimit(t *testing.T) {
	config := &BulkheadConfig{
		QueryPoolSize: 3,
	}
	interceptor := NewBulkheadInterceptor(config)

	started := make(chan struct{}, 3)
	block := make(chan struct{})

	handler := interceptor.WrapUnary(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		started <- struct{}{}
		<-block
		return nil, nil
	})

	var wg sync.WaitGroup
	errorCount := 0
	var mu sync.Mutex

	// Launch exactly 3 goroutines — these will acquire the semaphore and block.
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := handler(context.Background(), newMockRequest("/aleph.v1.QueryService/GetChatHistory"))
			if err != nil {
				mu.Lock()
				errorCount++
				mu.Unlock()
			}
		}()
	}

	// Wait for all 3 to be blocked inside the handler (holding the semaphore).
	for i := 0; i < 3; i++ {
		<-started
	}

	// Now launch the 4th synchronously — semaphore is full, so TryAcquire must fail.
	_, err := handler(context.Background(), newMockRequest("/aleph.v1.QueryService/GetChatHistory"))
	if err != nil {
		errorCount++
	}

	// Release the 3 blocked handlers.
	close(block)
	wg.Wait()

	assert.Equal(t, 1, errorCount)
}

func TestBulkheadStats(t *testing.T) {
	config := &BulkheadConfig{
		QueryPoolSize:     20,
		IngestionPoolSize: 5,
		ChatPoolSize:      10,
	}
	interceptor := NewBulkheadInterceptor(config)

	stats := interceptor.Stats()
	assert.Equal(t, int64(20), stats["query"])
	assert.Equal(t, int64(5), stats["ingestion"])
	assert.Equal(t, int64(10), stats["chat"])
}

func TestCustomBulkheadConfig(t *testing.T) {
	config := &BulkheadConfig{
		QueryPoolSize:     1,
		IngestionPoolSize: 1,
		ChatPoolSize:      1,
	}
	interceptor := NewBulkheadInterceptor(config)

	started := make(chan struct{}, 1)
	block := make(chan struct{})

	handler := interceptor.WrapUnary(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		started <- struct{}{}
		<-block
		return nil, nil
	})

	var wg sync.WaitGroup

	// Launch 1 goroutine — it acquires the semaphore and blocks.
	wg.Add(1)
	go func() {
		defer wg.Done()
		handler(context.Background(), newMockRequest("/aleph.v1.QueryService/GetChatHistory"))
	}()

	<-started

	// Launch the 2nd synchronously — semaphore is full, so TryAcquire must fail.
	_, err := handler(context.Background(), newMockRequest("/aleph.v1.QueryService/GetChatHistory"))
	require.Error(t, err)

	close(block)
	wg.Wait()
}
