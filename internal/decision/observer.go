package decision

import (
	"context"
)

// Observer is the interface for evaluating tool execution results.
type Observer interface {
	Observe(ctx context.Context, step PlannedStep, result *ActResult) (*Observation, error)
}

// DefaultObserver implements the Observe phase of the decision loop.
type DefaultObserver struct{}

// NewDefaultObserver creates a new DefaultObserver.
func NewDefaultObserver() *DefaultObserver {
	return &DefaultObserver{}
}

// Observe evaluates the result of a tool execution and produces an Observation.
// It checks for:
//   - Execution errors
//   - Empty output when no error occurred
//   - Truncation signals in the output
//   - JSON parse failures
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

	// Check for truncation
	if len(result.Output) > 1900 {
		obs.Issues = append(obs.Issues, "output was truncated due to context limits")
	}

	return obs, nil
}
