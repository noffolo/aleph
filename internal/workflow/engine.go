package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type engine struct {
	mu       sync.RWMutex
	steps    map[string]StepFunc
	statuses map[WorkflowID]Status
}

// NewEngine creates a new WorkflowEngine.
func NewEngine() Engine {
	return &engine{
		steps:    make(map[string]StepFunc),
		statuses: make(map[WorkflowID]Status),
	}
}

func (e *engine) RegisterStep(name string, fn StepFunc) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.steps[name] = fn
}

func (e *engine) Execute(ctx context.Context, w *Workflow) error {
	e.mu.Lock()
	w.Status = StatusRunning
	e.statuses[w.ID] = StatusRunning
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		if w.Status != StatusCompleted {
			w.Status = StatusFailed
		}
		e.statuses[w.ID] = w.Status
		e.mu.Unlock()
	}()

	for _, step := range w.Steps {
		select {
		case <-ctx.Done():
			w.Status = StatusCancelled
			return ctx.Err()
		default:
		}

		e.mu.RLock()
		fn, exists := e.steps[step.Name]
		e.mu.RUnlock()

		if !exists {
			return fmt.Errorf("workflow %s: step %s not registered", w.ID, step.Name)
		}

		result, err := fn(ctx, e.collectInputs(w.Result))
		if err != nil {
			w.Result = append(w.Result, StepResult{
				Name:   step.Name,
				Error:  err,
				Output: nil,
			})
			return fmt.Errorf("workflow %s: step %s failed: %w", w.ID, step.Name, err)
		}

		w.Result = append(w.Result, StepResult{
			Name:   step.Name,
			Output: result,
		})
	}

	w.Status = StatusCompleted
	return nil
}

func (e *engine) GetStatus(id WorkflowID) (Status, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	status, exists := e.statuses[id]
	if !exists {
		return "", fmt.Errorf("workflow %s not found", id)
	}
	return status, nil
}

func (e *engine) collectInputs(results []StepResult) map[string]interface{} {
	input := make(map[string]interface{})
	for _, r := range results {
		if r.Output != nil {
			input[r.Name] = r.Output
		}
	}
	return input
}

// NewID generates a workflow ID from timestamp.
func NewID() WorkflowID {
	return WorkflowID(fmt.Sprintf("wf-%d", time.Now().UnixNano()))
}