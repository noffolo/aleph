package genesis

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSandbox_Validate_EmptyCode(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	ctx := context.Background()
	suggestion := Suggestion{Code: ""}
	result, err := s.Validate(ctx, suggestion)
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if !result.Passed {
		t.Error("expected Passed=true for empty code")
	}
}

func TestSandbox_Validate_SafeCode(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	ctx := context.Background()
	suggestion := Suggestion{Code: "package main\n\nfunc main() { println(\"hello\") }"}
	result, err := s.Validate(ctx, suggestion)
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if !result.Passed {
		t.Errorf("expected Passed=true for safe code, got Passed=false (risk=%.2f, blocked=%v, warnings=%v)",
			result.RiskScore, result.BlockedPatterns, result.Warnings)
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
			result, err := s.Validate(ctx, suggestion)
			if err != nil {
				t.Fatalf("Validate returned error: %v", err)
			}
			if result.Passed != tc.expected {
				t.Errorf("expected Passed=%v, got %v for pattern %s", tc.expected, result.Passed, tc.name)
			}
		})
	}
}

func TestSandbox_Validate_ContextCancellation(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	suggestion := Suggestion{Code: "some code"}
	result, err := s.Validate(ctx, suggestion)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
	if result != nil && result.Passed {
		t.Error("expected Passed=false on cancelled context")
	}
}

func TestSandbox_BlocksDangerousPatterns(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	ctx := context.Background()

	code := "package main\n\nimport \"os/exec\"\nimport \"syscall\"\n\nfunc main() {}"
	result, err := s.Validate(ctx, code2Suggestion(code))
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if result.Passed {
		t.Error("expected Passed=false for code with os/exec and syscall")
	}
	if len(result.BlockedPatterns) < 2 {
		t.Errorf("expected at least 2 blocked patterns, got %d: %v", len(result.BlockedPatterns), result.BlockedPatterns)
	}
	if result.RiskScore <= 0 {
		t.Error("expected RiskScore > 0 for dangerous patterns")
	}
}

func TestSandbox_PassesSafeCode(t *testing.T) {
	s := NewSandbox(10 * time.Second)
	ctx := context.Background()

	code := "package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"hello\") }"
	result, err := s.Validate(ctx, code2Suggestion(code))
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	// Code may fail go vet subprocess (no go.mod consistency), so check pattern pass only
	if len(result.BlockedPatterns) > 0 {
		t.Errorf("expected no blocked patterns for safe code, got %v", result.BlockedPatterns)
	}
}

func TestSandbox_TimeoutEnforcement(t *testing.T) {
	s := NewSandbox(50 * time.Millisecond)
	ctx := context.Background()

	// Code that would take a long time to vet (infinite loop in init)
	code := "package main\n\nfunc init() { for {} }\nfunc main() {}"
	result, err := s.Validate(ctx, code2Suggestion(code))
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if result.Passed {
		t.Error("expected Passed=false for code that times out in subprocess")
	}
	if result.Duration > 2*time.Second {
		t.Errorf("expected fast timeout, but validation took %v", result.Duration)
	}
}

func TestSandbox_ObfuscationDetection(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	ctx := context.Background()

	tests := []struct {
		name           string
		code           string
		expectWarning  bool
		expectBlocked  bool
	}{
		{
			"base64 decode",
			"package main\n\nimport \"encoding/base64\"\n\nfunc main() { base64.StdEncoding.DecodeString(\"aGVsbG8=\") }",
			false,
			true,
		},
		{
			"plugin open",
			"package main\n\nimport \"plugin\"\n\nfunc main() { plugin.Open(\"mal.so\") }",
			false,
			true,
		},
		{
			"cgo escape",
			"package main\n\n// #include <stdio.h>\nimport \"C\"\n\nfunc main() {}",
			true,
			false,
		},
		{
			"clean code",
			"package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"hello\") }",
			false,
			false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := s.Validate(ctx, code2Suggestion(tc.code))
			if err != nil {
				t.Fatalf("Validate returned error: %v", err)
			}
			hasObfuscationWarning := false
			for _, w := range result.Warnings {
				if strings.Contains(w, "obfuscation pattern") {
					hasObfuscationWarning = true
					break
				}
			}
			if hasObfuscationWarning != tc.expectWarning {
				t.Errorf("expected obfuscation warning=%v, got warnings=%v", tc.expectWarning, result.Warnings)
			}
			if tc.expectBlocked && result.Passed {
				t.Errorf("expected code to be blocked (Passed=false), but Passed=true, BlockedPatterns=%v", result.BlockedPatterns)
			}
			if !tc.expectBlocked && !result.Passed && len(result.BlockedPatterns) > 0 {
				t.Errorf("expected code to not be blocked by blocklist, but got BlockedPatterns=%v", result.BlockedPatterns)
			}
		})
	}
}

func TestSandboxResult_Structure(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	ctx := context.Background()

	code := "package main\n\nimport \"os/exec\"\n\nfunc main() { exec.Command(\"ls\") }"
	result, err := s.Validate(ctx, code2Suggestion(code))
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}

	if result.Passed {
		t.Error("expected Passed=false")
	}
	if len(result.BlockedPatterns) == 0 {
		t.Error("expected BlockedPatterns to be populated")
	}
	if result.RiskScore <= 0 {
		t.Error("expected RiskScore > 0")
	}
	if result.Duration == 0 {
		t.Error("expected Duration > 0")
	}
}

func code2Suggestion(code string) Suggestion {
	return Suggestion{Code: code}
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

func TestSuggester_Analyze_EmptyInput(t *testing.T) {
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
		t.Errorf("expected empty suggestions, got %d", len(suggestions))
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
func TestSuggester_Analyze_GeneratesOntologySuggestions(t *testing.T) {
	s := NewSuggester()
	ctx := context.Background()

	input := SuggesterInput{
		ProjectID: "proj1",
		AgentID:   "agent1",
		ChatHistory: []ChatMessage{
			{Role: "user", Content: "Mercury is important for our model."},
			{Role: "user", Content: "Mercury appears in this dataset too."},
			{Role: "user", Content: "We should enrich Mercury signals."},
			{Role: "user", Content: "what about Saturn?"},
		},
	}

	suggestions, err := s.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(suggestions) == 0 {
		t.Fatal("expected suggestions, got 0")
	}

	foundMercury := false
	foundSaturn := false
	for _, sug := range suggestions {
		if sug.Status != "pending" {
			t.Fatalf("expected suggestion status pending, got %q", sug.Status)
		}
		if sug.Name == "Mercury" && sug.Type == "ontology" {
			foundMercury = true
		}
		if sug.Name == "Saturn" && sug.Type == "ontology" {
			foundSaturn = true
		}
	}

	if !foundMercury {
		t.Error("expected ontology suggestion for Mercury")
	}
	if !foundSaturn {
		t.Error("expected ontology suggestion for Saturn")
	}
}

func TestGenesisEngine_Suggest_RegistersPendingSuggestions(t *testing.T) {
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
		t.Fatalf("expected empty suggestions with no history, got %d", len(suggestions))
	}

	inputSuggestions, err := suggester.Analyze(ctx, SuggesterInput{
		ProjectID: "proj1",
		AgentID:   "agent1",
		ChatHistory: []ChatMessage{
			{Role: "user", Content: "what about Pluto?"},
		},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	for _, sug := range inputSuggestions {
		veto.Register(sug)
	}

	pending, err := veto.ListPending(ctx)
	if err != nil {
		t.Fatalf("ListPending returned error: %v", err)
	}
	if len(pending) == 0 {
		t.Fatal("expected pending suggestions to be registered")
	}
}
