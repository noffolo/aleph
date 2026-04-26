package decision

import (
	"context"
)

// Admitter is the interface for the Admit phase.
type Admitter interface {
	Admit(ctx context.Context, results []*ActResult, maxAttempts int) (bool, error)
}

// DefaultAdmitter implements the Admit phase of the decision loop.
// It decides when to stop the Plan→Act→Observe→Reflect cycle:
//   - Maximum attempts reached
//   - Error encountered in last execution
//   - Goal achieved (all steps completed successfully)
type DefaultAdmitter struct{}

// NewDefaultAdmitter creates a new DefaultAdmitter.
func NewDefaultAdmitter() *DefaultAdmitter {
	return &DefaultAdmitter{}
}

// Admit checks whether the loop should be terminated.
// Returns true when the loop should stop (admit completion).
func (a *DefaultAdmitter) Admit(ctx context.Context, results []*ActResult, maxAttempts int) (bool, error) {
	// No results yet — don't admit
	if len(results) == 0 {
		return false, nil
	}

	// Check max attempts
	if maxAttempts > 0 && len(results) >= maxAttempts {
		return true, nil
	}

	// Check if last result had an error
	last := results[len(results)-1]
	if last.Error != "" {
		return true, nil
	}

	// No tool calls means we're done (handled by Chat loop)
	// Continue by default
	return false, nil
}
