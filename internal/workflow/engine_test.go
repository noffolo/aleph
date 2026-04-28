package workflow

import (
	"context"
	"testing"
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