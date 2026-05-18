package workflow

import (
	"context"
	"fmt"
	"sync"
)

const DefaultMaxAgents = 3

type Orchestrator struct {
	engine    Engine
	maxAgents int
	activeMu  sync.Mutex
	active    int
}

func NewOrchestrator(engine Engine, maxAgents int) *Orchestrator {
	if maxAgents <= 0 {
		maxAgents = DefaultMaxAgents
	}
	return &Orchestrator{
		engine:    engine,
		maxAgents: maxAgents,
	}
}

func (o *Orchestrator) DecomposeTask(ctx context.Context, steps []Step) (*Workflow, error) {
	o.activeMu.Lock()
	if o.active >= o.maxAgents {
		o.activeMu.Unlock()
		return nil, fmt.Errorf("max concurrent agents reached (%d)", o.maxAgents)
	}
	o.active++
	o.activeMu.Unlock()

	defer func() {
		o.activeMu.Lock()
		o.active--
		o.activeMu.Unlock()
	}()

	w := &Workflow{
		ID:     NewID(),
		Steps:  steps,
		Status: StatusPending,
	}

	if err := o.engine.Execute(ctx, w); err != nil {
		return nil, fmt.Errorf("engine execute failed: %w", err)
	}

	return w, nil
}
