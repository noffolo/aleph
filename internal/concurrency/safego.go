// Package concurrency provides safe goroutine patterns and helpers.
package concurrency

import (
	"context"
	"log/slog"
)

// SafeGo launches a goroutine with built-in panic recovery.
// The context controls the goroutine lifetime — when ctx is cancelled,
// the goroutine should exit gracefully.
//
// Usage:
//
//	concurrency.SafeGo(ctx, "health-checker", func(ctx context.Context) {
//	    ticker := time.NewTicker(interval)
//	    defer ticker.Stop()
//	    for {
//	        select {
//	        case <-ctx.Done():
//	            return
//	        case <-ticker.C:
//	            // ...
//	        }
//	    }
//	})
func SafeGo(ctx context.Context, name string, fn func(context.Context)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("goroutine panic recovered",
					"name", name,
					"panic", r,
				)
			}
		}()
		fn(ctx)
	}()
}
