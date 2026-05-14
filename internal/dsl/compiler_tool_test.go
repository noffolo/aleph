package dsl

import (
	"context"
	"testing"

	"github.com/ff3300/aleph-v2/internal/sandbox"
)

func TestTestGeneratedTool_NilGen(t *testing.T) {
	v := sandbox.NewVerifier(nil, nil, "", "")
	result, err := TestGeneratedTool(context.Background(), nil, v)
	if err == nil {
		t.Fatal("expected error for nil generated tool")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestTestGeneratedTool_ValidationError(t *testing.T) {
	v := sandbox.NewVerifier(nil, nil, "", "")
	gen := &GeneratedTool{
		Name:    "",
		GoCode:  "package main\nfunc main() {}",
	}
	result, err := TestGeneratedTool(context.Background(), gen, v)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected TestResult.Passed=false for empty name (validation should fail)")
	}
}

func TestTestGeneratedTool_ValidPasses(t *testing.T) {
	v := sandbox.NewVerifier(nil, nil, "", "")
	gen := &GeneratedTool{
		Name:    "hello_world",
		GoCode:  "package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"hello\") }",
		Inputs: []*ToolParam{
			{Name: "name", Type: "string", Required: false},
		},
		Outputs: []*ToolParam{
			{Name: "greeting", Type: "string"},
		},
		Handler: &ToolHandler{
			Language: "go",
			EntryPoint: "handler.go",
		},
	}
	result, err := TestGeneratedTool(context.Background(), gen, v)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Errorf("expected pass, got error: %s", result.Error)
	}
}

func TestTestGeneratedTool_ValidPython(t *testing.T) {
	v := sandbox.NewVerifier(nil, nil, "", "")
	gen := &GeneratedTool{
		Name:       "hello_world",
		GoCode:     "package main\nimport \"fmt\"\nfunc main() {}",
		PythonCode: "def main():\n    return 'hello'",
		Inputs: []*ToolParam{
			{Name: "name", Type: "string", Required: false},
		},
		Outputs: []*ToolParam{
			{Name: "greeting", Type: "string"},
		},
		Handler: &ToolHandler{
			Language:    "python",
			EntryPoint:  "handler.py",
		},
	}
	result, err := TestGeneratedTool(context.Background(), gen, v)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Errorf("expected pass, got error: %s", result.Error)
	}
}

func TestTestGeneratedTool_MissingHandler(t *testing.T) {
	v := sandbox.NewVerifier(nil, nil, "", "")
	gen := &GeneratedTool{
		Name:   "hello_world",
		GoCode: "package main\nfunc main() {}",
		Inputs: []*ToolParam{
			{Name: "name", Type: "string", Required: false},
		},
		Outputs: []*ToolParam{
			{Name: "greeting", Type: "string"},
		},
	}
	result, err := TestGeneratedTool(context.Background(), gen, v)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected failure due to missing handler")
	}
}

func TestRegisterTool_NilGen(t *testing.T) {
	_, err := RegisterTool(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error for nil generated tool")
	}
}

func TestRegisterTool_NilMetaRepo(t *testing.T) {
	gen := &GeneratedTool{
		Name:    "my_tool",
		GoCode:  "package main\nfunc main() {}",
		Template: TemplateDataProcessor,
	}
	rec, err := RegisterTool(context.Background(), gen, nil)
	if err == nil {
		t.Fatal("expected error for nil metadata repository")
	}
	if rec != nil {
		t.Errorf("expected nil record, got %+v", rec)
	}
}
