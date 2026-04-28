package middleware

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedOnSuccess(t *testing.T) {
	cb := NewCircuitBreaker(2, time.Second)

	for i := 0; i < 5; i++ {
		err := cb.Execute(func() error {
			return nil
		})
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
	}

	if cb.State() != StateClosed {
		t.Fatalf("expected closed, got %v", cb.State())
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(3, time.Minute)

	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return errors.New("fail")
		})
	}

	if cb.State() != StateOpen {
		t.Fatal("expected circuit to be open")
	}

	err := cb.Execute(func() error {
		return nil
	})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatal("expected ErrCircuitOpen")
	}
}

func TestCircuitBreaker_HalfOpenRecovery(t *testing.T) {
	cb := NewCircuitBreaker(3, 10*time.Millisecond)

	// Trip the breaker
	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return errors.New("fail")
		})
	}

	time.Sleep(20 * time.Millisecond)

	// Should half-open and allow execution
	err := cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Fatalf("expected recovery, got: %v", err)
	}

	if cb.State() != StateClosed {
		t.Fatal("expected circuit to be closed after recovery")
	}
}