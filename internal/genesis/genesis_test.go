package genesis

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestSandbox_Validate_EmptyCode(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	ctx := context.Background()
	suggestion := Suggestion{Code: ""}
	valid, err := s.Validate(ctx, suggestion)
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if !valid {
		t.Error("expected valid=true for empty code")
	}
}

func TestSandbox_Validate_SafeCode(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	ctx := context.Background()
	suggestion := Suggestion{Code: "package main\n\nfunc main() { println(\"hello\") }"}
	valid, err := s.Validate(ctx, suggestion)
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if !valid {
		t.Error("expected valid=true for safe code")
	}
}

func TestSandbox_Validate_DangerousPattern(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	ctx := context.Background()

	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{"os/exec", "package main\n\nimport \"os/exec\"\n\nfunc main() { exec.Command(\"ls\") }", false},
		{"syscall", "package main\n\nimport \"syscall\"\n\nfunc main() {}", false},
		{"unsafe", "package main\n\nimport \"unsafe\"\n\nfunc main() {}", false},
		{"reflect", "package main\n\nimport \"reflect\"\n\nfunc main() {}", false},
		{"os.Remove", "package main\n\nimport \"os\"\n\nfunc main() { os.Remove(\"x\") }", false},
		{"os.RemoveAll", "package main\n\nimport \"os\"\n\nfunc main() { os.RemoveAll(\"dir\") }", false},
		{"os.Chmod", "package main\n\nimport \"os\"\n\nfunc main() { os.Chmod(\"f\", 0644) }", false},
		{"net.Listen", "package main\n\nimport \"net\"\n\nfunc main() { net.Listen(\"tcp\", \":8080\") }", false},
		{"net.Dial", "package main\n\nimport \"net\"\n\nfunc main() { net.Dial(\"tcp\", \"localhost:8080\") }", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			suggestion := Suggestion{Code: tc.code}
			valid, err := s.Validate(ctx, suggestion)
			if err != nil {
				t.Fatalf("Validate returned error: %v", err)
			}
			if valid != tc.expected {
				t.Errorf("expected valid=%v, got %v for pattern %s", tc.expected, valid, tc.name)
			}
		})
	}
}

func TestSandbox_Validate_ContextCancellation(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	suggestion := Suggestion{Code: "some code"}
	valid, err := s.Validate(ctx, suggestion)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
	if valid {
		t.Error("expected valid=false on cancelled context")
	}
}

func TestVetoRegistry_Register(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 1*time.Hour)
	defer v.Shutdown()

	s := Suggestion{ID: "s1", Name: "test", Description: "test suggestion", Status: "pending"}
	v.Register(s)

	pending, err := v.ListPending(ctx)
	if err != nil {
		t.Fatalf("ListPending returned error: %v", err)
	}
	if len(pending) != 1 {
		t.Errorf("expected 1 pending suggestion, got %d", len(pending))
	}
	if pending[0].ID != "s1" {
		t.Errorf("expected suggestion ID s1, got %s", pending[0].ID)
	}
}

func TestVetoRegistry_Approve(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 1*time.Hour)
	defer v.Shutdown()

	s := Suggestion{ID: "s1", Name: "test"}
	v.Register(s)

	err := v.Approve(ctx, "s1")
	if err != nil {
		t.Fatalf("Approve returned error: %v", err)
	}

	pending, _ := v.ListPending(ctx)
	if len(pending) != 0 {
		t.Error("expected no pending suggestions after approve")
	}
}

func TestVetoRegistry_Reject(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 1*time.Hour)
	defer v.Shutdown()

	s := Suggestion{ID: "s1", Name: "test"}
	v.Register(s)

	err := v.Reject(ctx, "s1")
	if err != nil {
		t.Fatalf("Reject returned error: %v", err)
	}

	pending, _ := v.ListPending(ctx)
	if len(pending) != 0 {
		t.Error("expected no pending suggestions after reject")
	}
}

func TestVetoRegistry_ListPending_ExcludesExpired(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 10*time.Millisecond)
	defer v.Shutdown()

	s := Suggestion{ID: "s1", Name: "test", Status: "pending"}
	v.Register(s)

	time.Sleep(20 * time.Millisecond)

	pending, err := v.ListPending(ctx)
	if err != nil {
		t.Fatalf("ListPending returned error: %v", err)
	}
	if len(pending) != 0 {
		t.Error("expected expired suggestion excluded from pending list")
	}
}

func TestVetoRegistry_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 1*time.Hour)
	defer v.Shutdown()

	var wg sync.WaitGroup
	numOps := 100

	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			s := Suggestion{ID: string(rune('a' + id%26)), Name: "concurrent"}
			v.Register(s)
		}(i)

		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, _ = v.ListPending(ctx)
		}(i)
	}

	wg.Wait()
}

func TestVetoRegistry_Shutdown(t *testing.T) {
	ctx := context.Background()
	v := NewVetoRegistry(ctx, 1*time.Hour)
	v.Shutdown()

	time.Sleep(50 * time.Millisecond)
}

func TestSuggester_Analyze_Stub(t *testing.T) {
	s := NewSuggester()
	ctx := context.Background()
	input := SuggesterInput{
		ProjectID: "proj1",
		AgentID:   "agent1",
	}
	suggestions, err := s.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(suggestions) != 0 {
		t.Errorf("expected empty suggestions from stub, got %d", len(suggestions))
	}
}

func TestGenesisEngine_Suggest_Empty(t *testing.T) {
	ctx := context.Background()
	suggester := NewSuggester()
	sandbox := NewSandbox(5 * time.Second)
	veto := NewVetoRegistry(ctx, 1*time.Hour)
	defer veto.Shutdown()

	engine := NewGenesisEngine(suggester, sandbox, veto)
	suggestions, err := engine.Suggest(ctx, "proj1", "agent1")
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if len(suggestions) != 0 {
		t.Errorf("expected empty suggestions, got %d", len(suggestions))
	}
}

func TestGenesisEngine_Approve(t *testing.T) {
	ctx := context.Background()
	suggester := NewSuggester()
	sandbox := NewSandbox(5 * time.Second)
	veto := NewVetoRegistry(ctx, 1*time.Hour)
	defer veto.Shutdown()

	engine := NewGenesisEngine(suggester, sandbox, veto)

	s := Suggestion{ID: "s1", Name: "test", Status: "pending"}
	veto.Register(s)

	err := engine.Approve(ctx, "s1")
	if err != nil {
		t.Fatalf("Approve returned error: %v", err)
	}
}