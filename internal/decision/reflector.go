package decision

import (
	"context"
	"strings"
)

// Reflector is the interface for the Reflect phase.
type Reflector interface {
	Reflect(ctx context.Context, plan *PlanResult, observations []Observation) (*PlanResult, error)
}

// DefaultReflector implements the Reflect phase of the decision loop.
// It analyzes observations from completed actions and determines:
//   - Whether to continue or stop
//   - Which tools to call next
//   - Whether the goal has been achieved
type DefaultReflector struct{}

// NewDefaultReflector creates a new DefaultReflector.
func NewDefaultReflector() *DefaultReflector {
	return &DefaultReflector{}
}

// Reflect analyzes the observations and produces an updated plan.
func (r *DefaultReflector) Reflect(ctx context.Context, plan *PlanResult, observations []Observation) (*PlanResult, error) {
	if plan == nil {
		return nil, nil
	}

	// If the plan already says we can't proceed, don't change it
	if !plan.CanProceed {
		return plan, nil
	}

	// No observations yet — nothing to reflect on
	if len(observations) == 0 {
		return plan, nil
	}

	lastObs := observations[len(observations)-1]

	// If the last observation failed, mark plan as unable to proceed
	if !lastObs.Success {
		return &PlanResult{
			Intent:     plan.Intent,
			Steps:      plan.Steps,
			CanProceed: false,
			Reason:     "tool execution failed: " + strings.Join(lastObs.Issues, "; "),
		}, nil
	}

	// If the last step succeeded, we typically need another LLM call to decide next action
	// The Chat loop handles calling the LLM again; we just need to keep the plan alive
	newSteps := make([]PlannedStep, len(plan.Steps))
	copy(newSteps, plan.Steps)

	return &PlanResult{
		Intent:     plan.Intent,
		Steps:      newSteps,
		CanProceed: true,
		Reason:     "step completed, ready for next iteration",
	}, nil
}
