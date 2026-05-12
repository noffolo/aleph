package concurrency

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestSafeGo_RunsSuccessfully(t *testing.T) {
	done := make(chan struct{})
	ctx := context.Background()

	SafeGo(ctx, "test-success", func(ctx context.Context) {
		close(done)
	})

	select {
	case <-done:
		// success — fn executed
	case <-time.After(time.Second):
		t.Fatal("SafeGo did not execute the function within timeout")
	}
}

func TestSafeGo_ReceivesContext(t *testing.T) {
	receivedCtx := make(chan context.Context, 1)

	SafeGo(context.Background(), "test-ctx-pass", func(ctx context.Context) {
		receivedCtx <- ctx
	})

	select {
	case got := <-receivedCtx:
		if got == nil {
			t.Error("SafeGo passed a nil context to fn")
		}
	case <-time.After(time.Second):
		t.Fatal("SafeGo did not call fn within timeout")
	}
}

func TestSafeGo_PanicRecovery(t *testing.T) {
	// Verify SafeGo catches panics and doesn't crash the test process.
	// We use a wait group to confirm the goroutine exits.

	var wg sync.WaitGroup
	wg.Add(1)

	SafeGo(context.Background(), "test-panic", func(ctx context.Context) {
		defer wg.Done()
		panic("intentional test panic")
	})

	// If SafeGo did NOT recover, this test process would have crashed.
	// The WaitGroup confirms the goroutine completed (i.e. recover ran).
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// panic was recovered, goroutine exited cleanly
	case <-time.After(time.Second):
		t.Fatal("goroutine with panic did not complete — possible deadlock")
	}
}

func TestSafeGo_PanicRecovery_DoesNotCrash(t *testing.T) {
	// Additional verification: run multiple panic goroutines
	// and confirm they all complete without crashing the test.

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		SafeGo(context.Background(), "test-multi-panic", func(ctx context.Context) {
			defer wg.Done()
			panic("boom")
		})
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// all goroutines recovered
	case <-time.After(time.Second):
		t.Fatal("not all panic goroutines completed")
	}
}

func TestSafeGo_PreCancelledContext(t *testing.T) {
	// A pre-cancelled context: fn should be able to detect cancellation
	// and exit early.

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	exitedEarly := make(chan struct{})
	SafeGo(ctx, "test-precancel", func(ctx context.Context) {
		select {
		case <-ctx.Done():
			close(exitedEarly)
			return
		case <-time.After(5 * time.Second):
			// should not reach here
		}
	})

	select {
	case <-exitedEarly:
		// fn detected context cancellation and exited
	case <-time.After(time.Second):
		t.Fatal("function did not respond to pre-cancelled context")
	}
}

func TestSafeGo_ContextCancellationDuringExecution(t *testing.T) {
	// Cancel context after goroutine starts — fn should exit
	// when it checks ctx.Done().

	ctx, cancel := context.WithCancel(context.Background())

	started := make(chan struct{})
	exited := make(chan struct{})

	SafeGo(ctx, "test-mid-exec", func(ctx context.Context) {
		close(started)
		<-ctx.Done()
		close(exited)
	})

	// Wait for goroutine to start
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("goroutine did not start")
	}

	// Cancel while goroutine is running
	cancel()

	select {
	case <-exited:
		// goroutine detected cancellation and exited
	case <-time.After(time.Second):
		t.Fatal("goroutine did not respond to context cancellation")
	}
}

func TestSafeGo_TableDriven(t *testing.T) {
	tests := []struct {
		name        string
		ctxSetup    func() context.Context
		fn          func(t *testing.T, done chan struct{}) func(context.Context)
		expectEarly bool
	}{
		{
			name:     "normal-execution",
			ctxSetup: context.Background,
			fn: func(t *testing.T, done chan struct{}) func(context.Context) {
				return func(ctx context.Context) {
					close(done)
				}
			},
		},
		{
			name: "pre-cancelled-context",
			ctxSetup: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			fn: func(t *testing.T, done chan struct{}) func(context.Context) {
				return func(ctx context.Context) {
					if ctx.Err() != nil {
						close(done)
						return
					}
					t.Error("expected context to be cancelled, but it was not")
				}
			},
		},
		{
			name: "timeout-context",
			ctxSetup: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
				// Don't cancel early; let timeout fire
				_ = cancel
				return ctx
			},
			fn: func(t *testing.T, done chan struct{}) func(context.Context) {
				return func(ctx context.Context) {
					<-ctx.Done()
					if ctx.Err() == context.DeadlineExceeded {
						close(done)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			done := make(chan struct{})
			ctx := tt.ctxSetup()
			fn := tt.fn(t, done)

			SafeGo(ctx, tt.name, fn)

			select {
			case <-done:
				// test passed
			case <-time.After(2 * time.Second):
				t.Fatal("test timed out")
			}
		})
	}
}

func TestSafeGo_ConcurrentExecution(t *testing.T) {
	// Verify SafeGo works correctly under concurrent usage.
	// Launch many goroutines and verify all complete.

	const numGoroutines = 50
	var wg sync.WaitGroup
	var mu sync.Mutex
	completed := make(map[int]bool)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		i := i // capture
		SafeGo(context.Background(), "test-concurrent", func(ctx context.Context) {
			defer wg.Done()
			mu.Lock()
			completed[i] = true
			mu.Unlock()
		})
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		mu.Lock()
		count := len(completed)
		mu.Unlock()
		if count != numGoroutines {
			t.Errorf("expected %d goroutines to complete, got %d", numGoroutines, count)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent goroutines did not all complete")
	}
}

func TestSafeGo_EmptyName(t *testing.T) {
	// Verify SafeGo accepts an empty name (no panic, no crash).
	done := make(chan struct{})
	SafeGo(context.Background(), "", func(ctx context.Context) {
		close(done)
	})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("SafeGo with empty name did not execute")
	}
}

func TestSafeGo_NilFunction_Recovers(t *testing.T) {
	// If fn is nil, calling it would panic inside the goroutine.
	// SafeGo's recover should catch this. This test verifies
	// that the caller (test process) does not crash.

	SafeGo(context.Background(), "test-nil-fn", nil)

	// Give the goroutine a moment to panic and recover
	time.Sleep(50 * time.Millisecond)

	// If we reach here, the test process did not crash.
	// This is intentional — even a nil fn should not take down the process.
}
