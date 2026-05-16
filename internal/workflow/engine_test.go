package workflow

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestEngine_RegisterAndExecute(t *testing.T) {
	eng := NewEngine()
	eng.RegisterStep("greet", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"message": "hello"}, nil
	})

	w := &Workflow{
		ID: NewID(),
		Steps: []Step{
			{Name: "greet"},
		},
	}

	err := eng.Execute(context.Background(), w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if w.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s", w.Status)
	}

	if len(w.Result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(w.Result))
	}
}

func TestEngine_StepNotFound(t *testing.T) {
	eng := NewEngine()
	w := &Workflow{
		ID: NewID(),
		Steps: []Step{
			{Name: "nonexistent"},
		},
	}

	err := eng.Execute(context.Background(), w)
	if err == nil {
		t.Fatal("expected error for nonexistent step")
	}
}

func TestEngine_GetStatus(t *testing.T) {
	eng := NewEngine()
	id := NewID()
	_, err := eng.GetStatus(id)
	if err == nil {
		t.Fatal("expected error for nonexistent workflow")
	}

	eng.RegisterStep("ok", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{}, nil
	})

	w := &Workflow{ID: id, Steps: []Step{{Name: "ok"}}}
	eng.Execute(context.Background(), w)

	status, err := eng.GetStatus(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != StatusCompleted {
		t.Fatalf("expected completed, got %s", status)
	}
}

func TestOrchestrator_MaxAgents(t *testing.T) {
	eng := NewEngine()
	eng.RegisterStep("simple", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"result": "ok"}, nil
	})

	orch := NewOrchestrator(eng, 1)
	w, err := orch.DecomposeTask(context.Background(), []Step{{Name: "simple"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s", w.Status)
	}
}

func TestEngine_MultiStepExecution(t *testing.T) {
	eng := NewEngine()

	eng.RegisterStep("parse", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"value": 42}, nil
	})

	eng.RegisterStep("double", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		parseResult, ok := input["parse"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("missing parse result")
		}
		v, ok := parseResult["value"].(int)
		if !ok {
			return nil, fmt.Errorf("parse value is not int")
		}
		return map[string]interface{}{"value": v * 2}, nil
	})

	eng.RegisterStep("triple", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		doubleResult, ok := input["double"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("missing double result")
		}
		v, ok := doubleResult["value"].(int)
		if !ok {
			return nil, fmt.Errorf("double value is not int")
		}
		return map[string]interface{}{"value": v * 3}, nil
	})

	w := &Workflow{
		ID: NewID(),
		Steps: []Step{
			{Name: "parse"},
			{Name: "double"},
			{Name: "triple"},
		},
	}

	err := eng.Execute(context.Background(), w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if w.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s", w.Status)
	}

	if len(w.Result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(w.Result))
	}

	if w.Result[0].Error != nil {
		t.Fatalf("step parse has unexpected error: %v", w.Result[0].Error)
	}
	if w.Result[0].Output["value"] != 42 {
		t.Fatalf("step parse expected value 42, got %v", w.Result[0].Output["value"])
	}

	if w.Result[1].Error != nil {
		t.Fatalf("step double has unexpected error: %v", w.Result[1].Error)
	}
	if w.Result[1].Output["value"] != 84 {
		t.Fatalf("step double expected value 84, got %v", w.Result[1].Output["value"])
	}

	if w.Result[2].Error != nil {
		t.Fatalf("step triple has unexpected error: %v", w.Result[2].Error)
	}
	if w.Result[2].Output["value"] != 252 {
		t.Fatalf("step triple expected value 252, got %v", w.Result[2].Output["value"])
	}

	status, err := eng.GetStatus(w.ID)
	if err != nil {
		t.Fatalf("unexpected error getting status: %v", err)
	}
	if status != StatusCompleted {
		t.Fatalf("expected completed status, got %s", status)
	}
}

func TestEngine_ContextCancellation(t *testing.T) {
	eng := NewEngine()

	stepStarted := make(chan struct{})
	stepDone := make(chan struct{})

	eng.RegisterStep("blocker", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		close(stepStarted)
		select {
		case <-ctx.Done():
			close(stepDone)
			return nil, ctx.Err()
		case <-time.After(10 * time.Second):
			close(stepDone)
			return map[string]interface{}{"done": true}, nil
		}
	})

	eng.RegisterStep("never", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		t.Error("second step should not have been executed after cancellation")
		return nil, nil
	})

	w := &Workflow{
		ID: NewID(),
		Steps: []Step{
			{Name: "blocker"},
			{Name: "never"},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	execDone := make(chan error, 1)
	go func() {
		execDone <- eng.Execute(ctx, w)
	}()

	<-stepStarted
	cancel()
	<-stepDone

	err := <-execDone
	if err == nil {
		t.Fatal("expected error from cancelled execution")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	if len(w.Result) != 1 {
		t.Fatalf("expected 1 result (only blocker), got %d", len(w.Result))
	}
	if w.Result[0].Error == nil {
		t.Fatal("expected blocker step to have an error from cancellation")
	}
}

func TestEngine_StepErrorPropagation(t *testing.T) {
	eng := NewEngine()

	eng.RegisterStep("ok", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"first": "done"}, nil
	})

	eng.RegisterStep("bad", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		okResult, ok := input["ok"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("missing ok result")
		}
		if okResult["first"] != "done" {
			return nil, fmt.Errorf("unexpected value from ok step")
		}
		return nil, fmt.Errorf("intentional failure in step bad")
	})

	eng.RegisterStep("never", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		t.Error("third step should not have been executed after step error")
		return nil, nil
	})

	w := &Workflow{
		ID: NewID(),
		Steps: []Step{
			{Name: "ok"},
			{Name: "bad"},
			{Name: "never"},
		},
	}

	err := eng.Execute(context.Background(), w)
	if err == nil {
		t.Fatal("expected error from failed execution")
	}

	if len(w.Result) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(w.Result))
	}

	if w.Result[0].Name != "ok" {
		t.Fatalf("expected first result name 'ok', got %s", w.Result[0].Name)
	}
	if w.Result[0].Error != nil {
		t.Fatalf("first step should have no error, got %v", w.Result[0].Error)
	}
	if w.Result[0].Output == nil || w.Result[0].Output["first"] != "done" {
		t.Fatalf("first step output mismatch: %v", w.Result[0].Output)
	}

	if w.Result[1].Name != "bad" {
		t.Fatalf("expected second result name 'bad', got %s", w.Result[1].Name)
	}
	if w.Result[1].Error == nil {
		t.Fatal("second step should have an error")
	}

	if len(w.Result) != 2 {
		t.Fatalf("expected exactly 2 results, got %d", len(w.Result))
	}

	if w.Status != StatusFailed {
		t.Fatalf("expected StatusFailed, got %s", w.Status)
	}

	status, err := eng.GetStatus(w.ID)
	if err != nil {
		t.Fatalf("unexpected error getting status: %v", err)
	}
	if status != StatusFailed {
		t.Fatalf("expected StatusFailed via GetStatus, got %s", status)
	}
}


