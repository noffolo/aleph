package mcp

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// mockPingerSuccess returns nil on every ping.
type mockPingerSuccess struct {
	mu     sync.Mutex
	count  int
	closed bool
}

func (m *mockPingerSuccess) Ping(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.count++
	return nil
}

func (m *mockPingerSuccess) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// mockPingerFail returns an error on every ping.
type mockPingerFail struct {
	mu     sync.Mutex
	count  int
	closed bool
	err    error
}

func (m *mockPingerFail) Ping(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.count++
	return m.err
}

func (m *mockPingerFail) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// mockPingerFailN returns success N times, then fails.
type mockPingerFailN struct {
	mu           sync.Mutex
	count        int
	successUntil int
	closed       bool
}

func (m *mockPingerFailN) Ping(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.count++
	if m.count <= m.successUntil {
		return nil
	}
	return errors.New("mock failure")
}

func (m *mockPingerFailN) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// mockPingerRestartFail always fails ping but supports restart (which also fails).
type mockPingerRestartFail struct {
	mu           sync.Mutex
	pingCount    int
	restartCount int
	closed       bool
}

func (m *mockPingerRestartFail) Ping(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pingCount++
	return errors.New("subprocess crashed")
}

func (m *mockPingerRestartFail) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockPingerRestartFail) Restart(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.restartCount++
	// Simulate restart failure (e.g., binary not found)
	return errors.New("exec format error")
}

// mockPingerRestartOK always fails ping but restart succeeds.
type mockPingerRestartOK struct {
	mu           sync.Mutex
	pingCount    int
	restartCount int
	closed       bool
	succeeded    bool
}

func (m *mockPingerRestartOK) Ping(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pingCount++
	if m.succeeded {
		return nil
	}
	return errors.New("subprocess crashed")
}

func (m *mockPingerRestartOK) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockPingerRestartOK) Restart(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.restartCount++
	m.succeeded = true
	return nil
}

func TestToolHealthMonitor_New(t *testing.T) {
	m := NewToolHealthMonitor(slog.Default(), "test-tool", &mockPingerSuccess{}, time.Second)
	assert.NotNil(t, m)
	assert.Len(t, m.tools, 1)
	assert.Equal(t, ToolStatusUnknown, m.tools["test-tool"].Status)
}

func TestToolHealthMonitor_New_Defaults(t *testing.T) {
	// nil logger and zero interval should get defaults
	m := NewToolHealthMonitor(nil, "test-tool", &mockPingerSuccess{}, 0)
	assert.NotNil(t, m.logger)
	assert.Equal(t, 30*time.Second, m.checkInterval)
}

func TestToolHealthMonitor_StartStop_Clean(t *testing.T) {
	m := NewToolHealthMonitor(slog.Default(), "test-tool", &mockPingerSuccess{}, 50*time.Millisecond)
	ctx := context.Background()
	m.Start(ctx)
	// Give it a moment to run a ping cycle
	time.Sleep(120 * time.Millisecond)
	m.Stop()

	status := m.Status()
	assert.Len(t, status, 1)
	assert.Equal(t, "test-tool", status[0].Name)
	assert.Equal(t, ToolStatusUp, status[0].Status)
}

func TestToolHealthMonitor_StartStop_NoGoroutineLeak(t *testing.T) {
	m := NewToolHealthMonitor(slog.Default(), "leak-test", &mockPingerSuccess{}, 20*time.Millisecond)
	ctx := context.Background()
	m.Start(ctx)

	// Let it run a few cycles
	time.Sleep(80 * time.Millisecond)

	// Stop and verify goroutine exits
	done := make(chan struct{})
	go func() {
		m.Stop()
		close(done)
	}()

	select {
	case <-done:
		// clean exit
	case <-time.After(5 * time.Second):
		t.Fatal("goroutine leak: Stop() did not complete within 5s")
	}
}

func TestToolHealthMonitor_MockPingerSuccess(t *testing.T) {
	pinger := &mockPingerSuccess{}
	m := NewToolHealthMonitor(slog.Default(), "test-tool", pinger, 50*time.Millisecond)
	ctx := context.Background()
	m.Start(ctx)
	time.Sleep(120 * time.Millisecond)
	m.Stop()

	// Pinger should have been called at least once
	assert.GreaterOrEqual(t, pinger.count, 1)
	assert.Equal(t, ToolStatusUp, m.GetLatestStatus("test-tool"))
}

func TestToolHealthMonitor_MockPingerFail_ThreeTimes(t *testing.T) {
	pinger := &mockPingerFail{err: errors.New("process crashed")}
	m := NewToolHealthMonitor(slog.Default(), "failing-tool", pinger, 30*time.Millisecond)

	// Set low consecutive threshold for test
	m.consecutive = 3

	ctx := context.Background()
	m.Start(ctx)

	// Wait enough time for at least 4 ping cycles
	time.Sleep(150 * time.Millisecond)
	m.Stop()

	assert.GreaterOrEqual(t, pinger.count, 3)
	assert.Equal(t, ToolStatusDown, m.GetLatestStatus("failing-tool"))
}

func TestToolHealthMonitor_DoubleStart_Safe(t *testing.T) {
	pinger := &mockPingerSuccess{}
	m := NewToolHealthMonitor(slog.Default(), "safe-tool", pinger, 50*time.Millisecond)
	ctx := context.Background()
	m.Start(ctx)
	// Second start should be a no-op
	m.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	m.Stop()
	assert.Equal(t, ToolStatusUp, m.GetLatestStatus("safe-tool"))
}

func TestToolHealthMonitor_StatusSnapshot(t *testing.T) {
	m := NewToolHealthMonitor(slog.Default(), "snapshot-tool", &mockPingerSuccess{}, 50*time.Millisecond)
	ctx := context.Background()
	m.Start(ctx)
	time.Sleep(120 * time.Millisecond)
	m.Stop()

	status := m.Status()
	assert.Len(t, status, 1)
	assert.Equal(t, "snapshot-tool", status[0].Name)
	assert.Equal(t, ToolStatusUp, status[0].Status)
	assert.False(t, status[0].LastPing.IsZero())
}

func TestToolHealthMonitor_UpdateToolName(t *testing.T) {
	m := NewToolHealthMonitor(slog.Default(), "original", &mockPingerSuccess{}, time.Second)
	m.UpdateToolName("new-tool")
	assert.Len(t, m.tools, 2)
	// Duplicate should not add
	m.UpdateToolName("original")
	assert.Len(t, m.tools, 2)
}

func TestToolHealthMonitor_ConsecutiveFailures(t *testing.T) {
	pinger := &mockPingerFail{err: errors.New("fail")}
	m := NewToolHealthMonitor(slog.Default(), "fail-tool", pinger, 30*time.Millisecond)
	m.consecutive = 3

	assert.Equal(t, 0, m.ConsecutiveFailures("nonexistent"))
	assert.Equal(t, 0, m.ConsecutiveFailures("fail-tool"))

	ctx := context.Background()
	m.Start(ctx)
	time.Sleep(150 * time.Millisecond)
	m.Stop()

	// Should have tracked failures
	assert.GreaterOrEqual(t, pinger.count, 3)
}

func TestToolHealthMonitor_GetLatestStatus_Unknown(t *testing.T) {
	m := NewToolHealthMonitor(slog.Default(), "unknown-tool", &mockPingerSuccess{}, time.Second)
	// Before starting, status should be unknown
	assert.Equal(t, ToolStatusUnknown, m.GetLatestStatus("unknown-tool"))
	assert.Equal(t, ToolStatusUnknown, m.GetLatestStatus("does-not-exist"))
}

func TestToolHealthMonitor_RestartAttempted_WhenSupported(t *testing.T) {
	pinger := &mockPingerRestartFail{}
	m := NewToolHealthMonitor(slog.Default(), "restart-fail", pinger, 30*time.Millisecond)
	m.consecutive = 3

	ctx := context.Background()
	m.Start(ctx)
	time.Sleep(150 * time.Millisecond)
	m.Stop()

	// Restart should have been attempted
	assert.GreaterOrEqual(t, pinger.restartCount, 1)
	// Status should be down since restart also failed
	assert.Equal(t, ToolStatusDown, m.GetLatestStatus("restart-fail"))
}

func TestToolHealthMonitor_RestartSuccess_Recovers(t *testing.T) {
	pinger := &mockPingerRestartOK{}
	m := NewToolHealthMonitor(slog.Default(), "restart-ok", pinger, 30*time.Millisecond)
	m.consecutive = 3

	ctx := context.Background()
	m.Start(ctx)

	// Wait enough time for failures + restart to trigger
	time.Sleep(200 * time.Millisecond)
	m.Stop()

	// Restart should have been attempted and should succeed
	assert.GreaterOrEqual(t, pinger.restartCount, 1)
	// After successful restart, next ping succeeds → status=up
	assert.Equal(t, ToolStatusUp, m.GetLatestStatus("restart-ok"))
}

func TestToolHealthMonitor_UpdateToolName_NotFound(t *testing.T) {
	m := NewToolHealthMonitor(slog.Default(), "tool-a", &mockPingerSuccess{}, time.Second)
	assert.Equal(t, ToolStatusUnknown, m.GetLatestStatus("tool-b"))
}

func TestNopPinger(t *testing.T) {
	p := &nopPinger{}
	assert.NoError(t, p.Ping(context.Background()))
	assert.NoError(t, p.Close())
}

func TestErrPinger(t *testing.T) {
	p := &ErrPinger{Err: errors.New("expected error")}
	assert.Error(t, p.Ping(context.Background()))
	assert.NoError(t, p.Close())
}
