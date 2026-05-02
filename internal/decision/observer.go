package decision

import (
	"context"
	"fmt"
)

// Observer is the interface for evaluating tool execution results.
type Observer interface {
	Observe(ctx context.Context, step PlannedStep, result *ActResult) (*Observation, error)
}

// DefaultObserver implements the Observe phase of the decision loop.
type DefaultObserver struct {
	truncationThreshold int
}

// NewDefaultObserver creates a new DefaultObserver.
// The truncation threshold defaults to 1900.
func NewDefaultObserver() *DefaultObserver {
	return &DefaultObserver{
		truncationThreshold: 1900,
	}
}

// NewDefaultObserverWithThreshold creates a DefaultObserver with a custom truncation threshold.
// A threshold <= 0 falls back to the default (1900).
func NewDefaultObserverWithThreshold(threshold int) *DefaultObserver {
	if threshold <= 0 {
		threshold = 1900
	}
	return &DefaultObserver{
		truncationThreshold: threshold,
	}
}

// Observe evaluates the result of a tool execution and produces an Observation.
// It checks for:
//   - Execution errors
//   - Empty output when no error occurred
//   - Truncation signals in the output (threshold configurable via NewDefaultObserverWithThreshold)
func (o *DefaultObserver) Observe(ctx context.Context, step PlannedStep, result *ActResult) (*Observation, error) {
	obs := &Observation{
		Step:       step,
		ActResult:  *result,
		Success:    true,
		TrustDelta: 0,
		Issues:     make([]string, 0),
	}

	// Check for execution errors
	if result.Error != "" {
		obs.Success = false
		obs.TrustDelta = -0.1
		obs.Issues = append(obs.Issues, result.Error)
	}

	// Check for empty output
	if result.Output == "" && result.Error == "" {
		obs.Issues = append(obs.Issues, "tool returned empty output")
	}

	// Check for truncation using configurable threshold
	if len(result.Output) > o.truncationThreshold {
		obs.Issues = append(obs.Issues, fmt.Sprintf("output was truncated due to context limits (%d > %d)", len(result.Output), o.truncationThreshold))
	}

	return obs, nil
}
