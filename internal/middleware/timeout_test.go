package middleware

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeoutInterceptor_WrapUnary(t *testing.T) {
	customConfig := &TimeoutConfig{
		DBTimeout:           500 * time.Millisecond,
		LLMTimeout:          2 * time.Second,
		NLPTimeout:          1 * time.Second,
		ExternalHTTPTimeout: 1500 * time.Millisecond,
		DefaultTimeout:      3 * time.Second,
	}

	tests := []struct {
		name       string
		procedure  string
		wantErr    bool
		wantMinDur time.Duration
		wantMaxDur time.Duration
	}{
		{
			name:       "QueryService gets DB timeout",
			procedure:  "/aleph.v1.QueryService/GetChatHistory",
			wantErr:    true,
			wantMinDur: 400 * time.Millisecond,
			wantMaxDur: 700 * time.Millisecond,
		},
		{
			name:       "NLPService gets NLP timeout",
			procedure:  "/aleph.nlp.v1.NLPService/AnalyzeSentiment",
			wantErr:    true,
			wantMinDur: 800 * time.Millisecond,
			wantMaxDur: 1300 * time.Millisecond,
		},
		{
			name:       "AgentService gets LLM timeout",
			procedure:  "/aleph.v1.AgentService/CreateAgent",
			wantErr:    true,
			wantMinDur: 1800 * time.Millisecond,
			wantMaxDur: 2300 * time.Millisecond,
		},
		{
			name:       "IngestionService gets ExternalHTTP timeout",
			procedure:  "/aleph.v1.IngestionService/IngestURL",
			wantErr:    true,
			wantMinDur: 1300 * time.Millisecond,
			wantMaxDur: 1800 * time.Millisecond,
		},
		{
			name:       "Unknown service gets default timeout",
			procedure:  "/aleph.v1.UnknownService/UnknownMethod",
			wantErr:    true,
			wantMinDur: 2800 * time.Millisecond,
			wantMaxDur: 3300 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			interceptor := NewTimeoutInterceptor(customConfig)
			
			// sleep must exceed the procedure timeout so context expires first.
			sleepDur := tt.wantMaxDur + 200*time.Millisecond

			handler := interceptor.WrapUnary(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				// Simulate a long-running operation that should timeout
				select {
				case <-time.After(sleepDur):
					return nil, nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			})

			start := time.Now()
			_, err := handler(context.Background(), newMockRequest(tt.procedure))
			elapsed := time.Since(start)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "context deadline exceeded")
				assert.Less(t, elapsed, tt.wantMaxDur, "should have timed out before maximum duration")
				assert.Greater(t, elapsed, tt.wantMinDur/2, "should have run at least half the expected timeout")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTimeoutFromContext(t *testing.T) {
	interceptor := NewTimeoutInterceptor(nil)
	
	handler := interceptor.WrapUnary(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		timeout := TimeoutFromContext(ctx)
		assert.Equal(t, 5*time.Second, timeout, "QueryService should have DB timeout")
		return nil, nil
	})

	_, err := handler(context.Background(), newMockRequest("/aleph.v1.QueryService/GetChatHistory"))
	require.NoError(t, err)
}

func TestTimeoutInterceptor_StreamingHandler(t *testing.T) {
	interceptor := NewTimeoutInterceptor(nil)
	
	handler := interceptor.WrapStreamingHandler(func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		select {
		case <-time.After(6 * time.Second): // Exceeds DB timeout of 5s
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	start := time.Now()
	err := handler(context.Background(), &mockStreamingHandlerConn{procedure: "/aleph.v1.QueryService/StreamQuery"})
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
	assert.Less(t, elapsed, 6*time.Second, "should have timed out before 6 seconds")
}

func TestCustomTimeoutConfig(t *testing.T) {
	customConfig := &TimeoutConfig{
		DBTimeout:           1 * time.Second,
		LLMTimeout:          2 * time.Second,
		NLPTimeout:          3 * time.Second,
		ExternalHTTPTimeout: 4 * time.Second,
		DefaultTimeout:      5 * time.Second,
	}
	
	interceptor := NewTimeoutInterceptor(customConfig)
	
	handler := interceptor.WrapUnary(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		select {
		case <-time.After(1500 * time.Millisecond): // Exceeds 1s DB timeout
			return nil, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})

	start := time.Now()
	_, err := handler(context.Background(), newMockRequest("/aleph.v1.QueryService/GetChatHistory"))
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
	assert.Less(t, elapsed, 2*time.Second, "should have timed out before 2 seconds")
}

