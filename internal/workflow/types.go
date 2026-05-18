package workflow

import "context"

// Status represents the current state of a workflow.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

// WorkflowID is a unique identifier for a workflow.
type WorkflowID string

// StepResult is the outcome of a single workflow step.
type StepResult struct {
	Name   string
	Error  error
	Output map[string]any
}

// Workflow represents a multi-step task execution.
type Workflow struct {
	ID     WorkflowID
	Status Status
	Steps  []Step
	Result []StepResult
}

// Step is a single unit of work within a workflow.
type Step struct {
	Name string
	Fn   StepFunc
}

// StepFunc is the signature for a workflow step function.
type StepFunc func(ctx context.Context, input map[string]any) (map[string]any, error)

// Engine defines the workflow execution interface.
type Engine interface {
	// RegisterStep registers a named step function.
	RegisterStep(name string, fn StepFunc)
	// Execute runs a workflow through its steps sequentially.
	Execute(ctx context.Context, w *Workflow) error
	// GetStatus returns the current status of a workflow.
	GetStatus(id WorkflowID) (Status, error)
}
