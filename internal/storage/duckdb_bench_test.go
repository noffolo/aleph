package storage

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"testing"
	"time"
)

// BenchmarkConcurrentReads tests 10 goroutines performing 1000 reads each.
// This benchmark exposes the RWMutex bottleneck: only 1 reader at a time.
func BenchmarkConcurrentReads(b *testing.B) {
	db, err := NewDuckDB(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	// Setup: create table with test data
	_, err = db.Exec(context.Background(), "CREATE TABLE benchmark_data (id INTEGER, value VARCHAR)")
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 1000; i++ {
		_, err = db.Exec(context.Background(), "INSERT INTO benchmark_data VALUES (?, ?)", i, fmt.Sprintf("value_%d", i))
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()

	const numGoroutines = 10
	const readsPerGoroutine = 1000

	b.RunParallel(func(pb *testing.PB) {
		gid := rand.Intn(10000) // unique goroutine ID for query variation
		for pb.Next() {
			for i := 0; i < readsPerGoroutine; i++ {
				id := (gid + i) % 1000
				rows, err := db.Query("SELECT value FROM benchmark_data WHERE id = ?", id)
				if err != nil {
					b.Errorf("query failed: %v", err)
					continue
				}
				rows.Close()
			}
		}
	})
}

// BenchmarkConcurrentWrites tests 10 goroutines performing 100 writes each.
// This benchmark shows write serialization due to RWMutex.Lock().
func BenchmarkConcurrentWrites(b *testing.B) {
	db, err := NewDuckDB(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	// Setup: create table
	_, err = db.Exec(context.Background(), "CREATE TABLE benchmark_writes (id INTEGER, value VARCHAR, created_at TIMESTAMP)")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	const numGoroutines = 10
	const writesPerGoroutine = 100

	b.RunParallel(func(pb *testing.PB) {
		gid := rand.Intn(10000)
		for pb.Next() {
			for i := 0; i < writesPerGoroutine; i++ {
				id := gid*1000 + i
				_, err := db.Exec(context.Background(), "INSERT INTO benchmark_writes VALUES (?, ?, ?)", id, fmt.Sprintf("value_%d_%d", gid, i), time.Now())
				if err != nil {
					b.Errorf("insert failed: %v", err)
				}
			}
		}
	})
}

// BenchmarkMixedReadWrite tests 5 readers and 5 writers with 500 ops each.
// This simulates real-world workload with mixed access patterns.
func BenchmarkMixedReadWrite(b *testing.B) {
	db, err := NewDuckDB(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	// Setup: create table with initial data
	_, err = db.Exec(context.Background(), "CREATE TABLE benchmark_mixed (id INTEGER, value VARCHAR, updated_at TIMESTAMP)")
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 500; i++ {
		_, err = db.Exec(context.Background(), "INSERT INTO benchmark_mixed VALUES (?, ?, ?)", i, fmt.Sprintf("initial_%d", i), time.Now())
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()

	const numWorkers = 10
	const opsPerWorker = 500

	b.RunParallel(func(pb *testing.PB) {
		gid := rand.Intn(10000)
		isWriter := gid%2 == 0 // 50% writers, 50% readers
		for pb.Next() {
			for i := 0; i < opsPerWorker; i++ {
				id := (gid + i) % 500
				if isWriter {
					_, err := db.Exec(context.Background(), "UPDATE benchmark_mixed SET value = ?, updated_at = ? WHERE id = ?",
						fmt.Sprintf("updated_%d_%d", gid, i), time.Now(), id)
					if err != nil {
						b.Errorf("update failed: %v", err)
					}
				} else {
					rows, err := db.Query("SELECT value FROM benchmark_mixed WHERE id = ?", id)
					if err != nil {
						b.Errorf("query failed: %v", err)
						continue
					}
					rows.Close()
				}
			}
		}
	})
}

// percentile calculates the p-th percentile of a sorted slice
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p / 100.0)
	return sorted[idx]
}

// BenchmarkReadLatency measures p50, p95, p99 latency for read operations.
func BenchmarkReadLatency(b *testing.B) {
	db, err := NewDuckDB(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	// Setup
	_, err = db.Exec(context.Background(), "CREATE TABLE latency_test (id INTEGER, value VARCHAR)")
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		_, err = db.Exec(context.Background(), "INSERT INTO latency_test VALUES (?, ?)", i, fmt.Sprintf("value_%d", i))
		if err != nil {
			b.Fatal(err)
		}
	}

	var latencies []float64
	var mu sync.Mutex

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		id := i % 100
		start := time.Now()
		rows, err := db.Query("SELECT value FROM latency_test WHERE id = ?", id)
		latency := time.Since(start).Seconds() * 1000 // convert to ms

		mu.Lock()
		latencies = append(latencies, latency)
		mu.Unlock()

		if err != nil {
			b.Errorf("query failed: %v", err)
		}
		if rows != nil {
			rows.Close()
		}
	}

	// Calculate percentiles
	sort.Float64s(latencies)
	b.ReportMetric(percentile(latencies, 50), "p50_ms")
	b.ReportMetric(percentile(latencies, 95), "p95_ms")
	b.ReportMetric(percentile(latencies, 99), "p99_ms")
}

// BenchmarkWriteLatency measures p50, p95, p99 latency for write operations.
func BenchmarkWriteLatency(b *testing.B) {
	db, err := NewDuckDB(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	// Setup
	_, err = db.Exec(context.Background(), "CREATE TABLE write_latency_test (id INTEGER, value VARCHAR, ts TIMESTAMP)")
	if err != nil {
		b.Fatal(err)
	}

	var latencies []float64
	var mu sync.Mutex

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()
		_, err := db.Exec(context.Background(), "INSERT INTO write_latency_test VALUES (?, ?, ?)", i, fmt.Sprintf("value_%d", i), time.Now())
		latency := time.Since(start).Seconds() * 1000 // convert to ms

		mu.Lock()
		latencies = append(latencies, latency)
		mu.Unlock()

		if err != nil {
			b.Errorf("insert failed: %v", err)
		}
	}

	// Calculate percentiles
	sort.Float64s(latencies)
	b.ReportMetric(percentile(latencies, 50), "p50_ms")
	b.ReportMetric(percentile(latencies, 95), "p95_ms")
	b.ReportMetric(percentile(latencies, 99), "p99_ms")
}

// BenchmarkSemaphoreThroughput tests QueryContext with semaphore limiting.
// The semaphore (weight=5) allows 5 concurrent queries, but RWMutex serializes.
func BenchmarkSemaphoreThroughput(b *testing.B) {
	db, err := NewDuckDB(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	// Setup
	_, err = db.Exec(context.Background(), "CREATE TABLE semaphore_test (id INTEGER, value VARCHAR)")
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		_, err = db.Exec(context.Background(), "INSERT INTO semaphore_test VALUES (?, ?)", i, fmt.Sprintf("value_%d", i))
		if err != nil {
			b.Fatal(err)
		}
	}

	ctx := context.Background()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := rand.Intn(100)
			rows, err := db.QueryContext(ctx, "SELECT value FROM semaphore_test WHERE id = ?", id)
			if err != nil {
				// Expected: semaphore limit causes resource exhaustion
				// This benchmark measures throughput under semaphore constraints
				continue
			}
			rows.Close()
		}
	})
}

// BenchmarkDirectDBAccess tests bypassing the wrapper (no mutex, no semaphore).
// This shows the theoretical maximum throughput if DuckDB's native concurrency is used.
func BenchmarkDirectDBAccess(b *testing.B) {
	db, err := NewDuckDB(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	// Setup
	_, err = db.Exec(context.Background(), "CREATE TABLE direct_test (id INTEGER, value VARCHAR)")
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		_, err = db.Exec(context.Background(), "INSERT INTO direct_test VALUES (?, ?)", i, fmt.Sprintf("value_%d", i))
		if err != nil {
			b.Fatal(err)
		}
	}

	// Access raw *sql.DB directly (bypasses mutex and semaphore)
	rawDB := db.DB()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := rand.Intn(100)
			rows, err := rawDB.Query("SELECT value FROM direct_test WHERE id = ?", id)
			if err != nil {
				b.Errorf("query failed: %v", err)
				continue
			}
			rows.Close()
		}
	})
}

// BenchmarkConcurrentReadsDirectDB compares concurrent reads using raw *sql.DB.
// This tests if DuckDB's native connection pooling outperforms the mutex-wrapped approach.
func BenchmarkConcurrentReadsDirectDB(b *testing.B) {
	db, err := NewDuckDB(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	// Setup
	_, err = db.Exec(context.Background(), "CREATE TABLE direct_concurrent (id INTEGER, value VARCHAR)")
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 1000; i++ {
		_, err = db.Exec(context.Background(), "INSERT INTO direct_concurrent VALUES (?, ?)", i, fmt.Sprintf("value_%d", i))
		if err != nil {
			b.Fatal(err)
		}
	}

	rawDB := db.DB()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		gid := rand.Intn(10000)
		for pb.Next() {
			for i := 0; i < 100; i++ {
				id := (gid + i) % 1000
				rows, err := rawDB.Query("SELECT value FROM direct_concurrent WHERE id = ?", id)
				if err != nil {
					b.Errorf("query failed: %v", err)
					continue
				}
				rows.Close()
			}
		}
	})
}

// BenchmarkResourceExhaustion tests behavior when semaphore limit is exceeded.
// Expected: queries should fail with "duckdb resource exhausted" error.
func BenchmarkResourceExhaustion(b *testing.B) {
	db, err := NewDuckDB(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	// Setup
	_, err = db.Exec(context.Background(), "CREATE TABLE exhaustion_test (id INTEGER)")
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	b.ResetTimer()

	// Try to acquire more than semaphore weight (5)
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Long-running query to hold semaphore slot
			_, err := db.QueryContext(ctx, "SELECT COUNT(*) FROM exhaustion_test")
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Count exhaustion errors
	exhaustionCount := 0
	for err := range errors {
		if err != nil && err.Error() == "duckdb resource exhausted: too many concurrent queries" {
			exhaustionCount++
		}
	}

	b.Logf("resource exhaustion errors: %d", exhaustionCount)
}
