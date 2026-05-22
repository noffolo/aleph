package genesis

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// NewVetoRegistry
// ---------------------------------------------------------------------------

func TestNewVetoRegistry_Happy(t *testing.T) {
	ctx := context.Background()
	ttl := 1 * time.Hour
	v := NewVetoRegistry(ctx, ttl)
	defer v.Shutdown()

	assert.NotNil(t, v)
	assert.Equal(t, ttl, v.ttl)
	assert.NotNil(t, v.cancel)
}

func TestNewVetoRegistry_VeryShortTTL(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 1*time.Nanosecond)
	defer v.Shutdown()

	assert.NotNil(t, v)
	assert.Equal(t, 1*time.Nanosecond, v.ttl)
}

func TestNewVetoRegistry_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	v := NewVetoRegistry(ctx, 1*time.Hour)
	defer v.Shutdown()

	assert.NotNil(t, v)
	assert.NotNil(t, v.suggestions)
}

// ---------------------------------------------------------------------------
// Register
// ---------------------------------------------------------------------------

func TestVetoRegistry_Register_Defaults(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 1*time.Hour)
	defer v.Shutdown()

	s := Suggestion{ID: "s1", Name: "test-defaults"}
	v.Register(s)

	pending, err := v.ListPending(ctx)
	assert.NoError(t, err)
	assert.Len(t, pending, 1)
	assert.Equal(t, "pending", pending[0].Status)
	assert.False(t, pending[0].CreatedAt.IsZero())
	assert.False(t, pending[0].ExpiresAt.IsZero())
	assert.True(t, pending[0].ExpiresAt.After(pending[0].CreatedAt))
}

func TestVetoRegistry_Register_Overwrite(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 1*time.Hour)
	defer v.Shutdown()

	s1 := Suggestion{ID: "s1", Name: "first", Status: "pending"}
	s2 := Suggestion{ID: "s1", Name: "second", Status: "pending"}

	v.Register(s1)
	v.Register(s2)

	pending, err := v.ListPending(ctx)
	assert.NoError(t, err)
	assert.Len(t, pending, 1)
	assert.Equal(t, "second", pending[0].Name)
}

func TestVetoRegistry_Register_PreservesStatus(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 1*time.Hour)
	defer v.Shutdown()

	s := Suggestion{ID: "s1", Name: "prefiled", Status: "approved"}
	v.Register(s)

	pending, err := v.ListPending(ctx)
	assert.NoError(t, err)
	assert.Len(t, pending, 0, "non-pending status should not appear in ListPending")
}

// ---------------------------------------------------------------------------
// Approve
// ---------------------------------------------------------------------------

func TestVetoRegistry_Approve_Happy(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 1*time.Hour)
	defer v.Shutdown()

	v.Register(Suggestion{ID: "s1", Name: "approve-me"})
	err := v.Approve(ctx, "s1")
	assert.NoError(t, err)

	pending, err := v.ListPending(ctx)
	assert.NoError(t, err)
	assert.Len(t, pending, 0)
}

func TestVetoRegistry_Approve_AlreadyApproved(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 1*time.Hour)
	defer v.Shutdown()

	v.Register(Suggestion{ID: "s1", Name: "already-ok"})
	_ = v.Approve(ctx, "s1")
	err := v.Approve(ctx, "s1")
	assert.NoError(t, err)
}

func TestVetoRegistry_Approve_EmptyID(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 1*time.Hour)
	defer v.Shutdown()

	err := v.Approve(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// Reject
// ---------------------------------------------------------------------------

func TestVetoRegistry_Reject_Happy(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 1*time.Hour)
	defer v.Shutdown()

	v.Register(Suggestion{ID: "s1", Name: "reject-me"})
	err := v.Reject(ctx, "s1")
	assert.NoError(t, err)

	pending, err := v.ListPending(ctx)
	assert.NoError(t, err)
	assert.Len(t, pending, 0)
}

func TestVetoRegistry_Reject_AlreadyRejected(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 1*time.Hour)
	defer v.Shutdown()

	v.Register(Suggestion{ID: "s1", Name: "already-bad"})
	_ = v.Reject(ctx, "s1")
	err := v.Reject(ctx, "s1")
	assert.NoError(t, err)
}

func TestVetoRegistry_Reject_EmptyID(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 1*time.Hour)
	defer v.Shutdown()

	err := v.Reject(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// ListPending — mixed states
// ---------------------------------------------------------------------------

func TestVetoRegistry_ListPending_MixedStates(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 1*time.Hour)
	defer v.Shutdown()

	v.Register(Suggestion{ID: "p1", Name: "pending-1", Status: "pending"})
	v.Register(Suggestion{ID: "p2", Name: "pending-2", Status: "pending"})
	v.Register(Suggestion{ID: "a1", Name: "approved", Status: "approved"})
	v.Register(Suggestion{ID: "r1", Name: "rejected", Status: "rejected"})

	pending, err := v.ListPending(ctx)
	assert.NoError(t, err)
	assert.Len(t, pending, 2)

	ids := make(map[string]bool)
	for _, s := range pending {
		ids[s.ID] = true
	}
	assert.True(t, ids["p1"])
	assert.True(t, ids["p2"])
	assert.False(t, ids["a1"])
	assert.False(t, ids["r1"])
}

func TestVetoRegistry_ListPending_Empty(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 1*time.Hour)
	defer v.Shutdown()

	pending, err := v.ListPending(ctx)
	assert.NoError(t, err)
	assert.Len(t, pending, 0)
}

func TestVetoRegistry_ListPending_AfterShutdown(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 1*time.Hour)

	v.Register(Suggestion{ID: "p1", Name: "pre-shutdown"})
	v.Shutdown()

	time.Sleep(50 * time.Millisecond)

	pending, err := v.ListPending(ctx)
	assert.NoError(t, err)
	assert.Len(t, pending, 1)
}

// ---------------------------------------------------------------------------
// Shutdown
// ---------------------------------------------------------------------------

func TestVetoRegistry_Shutdown_Once(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 1*time.Hour)
	v.Shutdown()
	time.Sleep(20 * time.Millisecond)
}

func TestVetoRegistry_Shutdown_Double(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 1*time.Hour)
	v.Shutdown()
	v.Shutdown()
	time.Sleep(20 * time.Millisecond)
}

func TestVetoRegistry_Shutdown_OperationsAfter(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 1*time.Hour)
	v.Shutdown()
	time.Sleep(20 * time.Millisecond)
	v.Register(Suggestion{ID: "post", Name: "after-shutdown"})
	pending, err := v.ListPending(ctx)
	assert.NoError(t, err)
	assert.Len(t, pending, 1)
}

// ---------------------------------------------------------------------------
// cleanupLoop — expiry and deduplication
// ---------------------------------------------------------------------------

func TestVetoRegistry_CleanupLoop_Expiry(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 20*time.Millisecond)
	defer v.Shutdown()

	v.Register(Suggestion{ID: "e1", Name: "expiring-soon"})

	pendingBefore, err := v.ListPending(ctx)
	assert.NoError(t, err)
	assert.Len(t, pendingBefore, 1)

	time.Sleep(50 * time.Millisecond)

	pendingAfter, err := v.ListPending(ctx)
	assert.NoError(t, err)
	assert.Len(t, pendingAfter, 0, "expired suggestion should be cleaned up")
}

func TestVetoRegistry_CleanupLoop_PreservesNonExpired(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 2*time.Second)
	defer v.Shutdown()

	v.Register(Suggestion{ID: "keep", Name: "keeps"})
	time.Sleep(30 * time.Millisecond)

	pending, err := v.ListPending(ctx)
	assert.NoError(t, err)
	assert.Len(t, pending, 1)
	assert.Equal(t, "keep", pending[0].ID)
}

func TestVetoRegistry_CleanupLoop_MultiRegister(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 50*time.Millisecond)
	defer v.Shutdown()

	count := 100
	for i := 0; i < count; i++ {
		id := "reg-" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		v.Register(Suggestion{ID: id, Name: "many"})
	}

	pending, err := v.ListPending(ctx)
	assert.NoError(t, err)
	assert.Len(t, pending, count)

	time.Sleep(100 * time.Millisecond)

	pendingAfter, err := v.ListPending(ctx)
	assert.NoError(t, err)
	assert.Len(t, pendingAfter, 0)
}

// ---------------------------------------------------------------------------
// Concurrent mutation patterns
// ---------------------------------------------------------------------------

func TestVetoRegistry_Concurrent_RegisterApproveReject(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 1*time.Hour)
	defer v.Shutdown()

	var wg sync.WaitGroup
	numGoroutines := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			id := "conc-" + string(rune('a'+idx%26))
			v.Register(Suggestion{ID: id, Name: "concurrent"})
		}(i)
	}

	wg.Wait()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			id := "conc-" + string(rune('a'+idx%26))
			_ = v.Approve(ctx, id)
		}(i)
	}

	wg.Wait()

	pending, err := v.ListPending(ctx)
	assert.NoError(t, err)
	assert.Len(t, pending, 0)
}

func TestVetoRegistry_Concurrent_ListPendingDuringMutation(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 1*time.Hour)
	defer v.Shutdown()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_ = v.Reject(ctx, "does-not-exist")
		}(i)
	}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = v.ListPending(ctx)
		}()
	}

	wg.Wait()
}

// ---------------------------------------------------------------------------
// Context propagation through ListPending
// ---------------------------------------------------------------------------

func TestVetoRegistry_ListPending_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	v := NewVetoRegistry(ctx, 1*time.Hour)
	defer v.Shutdown()

	v.Register(Suggestion{ID: "s1", Name: "test"})
	pending, err := v.ListPending(ctx)
	assert.NoError(t, err)
	assert.Len(t, pending, 1)
}

// ---------------------------------------------------------------------------
// Edge: Register with empty ID
// ---------------------------------------------------------------------------

func TestVetoRegistry_Register_EmptyID(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 1*time.Hour)
	defer v.Shutdown()

	v.Register(Suggestion{ID: "", Name: "no-id"})
	pending, err := v.ListPending(ctx)
	assert.NoError(t, err)
	assert.Len(t, pending, 1)
	assert.Equal(t, "", pending[0].ID)
}

func TestVetoRegistry_Register_MultipleEmptyIDs(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 1*time.Hour)
	defer v.Shutdown()

	v.Register(Suggestion{ID: "", Name: "first-empty"})
	v.Register(Suggestion{ID: "", Name: "second-empty"})

	pending, err := v.ListPending(ctx)
	assert.NoError(t, err)
	assert.Len(t, pending, 1)
	assert.Equal(t, "second-empty", pending[0].Name)
}

// ---------------------------------------------------------------------------
// GenesisEngine methods (simple delegation tests)
// ---------------------------------------------------------------------------

func TestGenesisEngine_Approve_NotFound(t *testing.T) {
	ctx := context.Background()
	veto := NewVetoRegistry(ctx, 1*time.Hour)
	defer veto.Shutdown()
	engine := NewGenesisEngine(NewSuggester(), NewSandbox(5*time.Second), veto)

	err := engine.Approve(ctx, "does-not-exist")
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "not found"))
}

func TestGenesisEngine_Reject_NotFound(t *testing.T) {
	ctx := context.Background()
	veto := NewVetoRegistry(ctx, 1*time.Hour)
	defer veto.Shutdown()
	engine := NewGenesisEngine(NewSuggester(), NewSandbox(5*time.Second), veto)

	err := engine.Reject(ctx, "does-not-exist")
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "not found"))
}

func TestGenesisEngine_Suggest_PartialValidation(t *testing.T) {
	ctx := context.Background()
	veto := NewVetoRegistry(ctx, 1*time.Hour)
	defer veto.Shutdown()
	engine := NewGenesisEngine(NewSuggester(), NewSandbox(5*time.Second), veto)

	suggestions, err := engine.Suggest(ctx, "proj-empty", "agent-empty")
	assert.NoError(t, err)
	assert.NotNil(t, suggestions)
	assert.Len(t, suggestions, 0)
}
