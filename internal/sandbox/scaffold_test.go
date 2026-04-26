package sandbox

import (
	"strings"
	"testing"
)

func TestScaffoldGenerator_GenerateGoTest(t *testing.T) {
	sg := NewScaffoldGenerator()
	inputs := map[string]string{
		"query":    "string",
		"limit":    "int",
	}
	outputs := []string{"results", "total"}

	code := sg.GenerateGoTest("search_tool", inputs, outputs)

	if !strings.Contains(code, "TestHandleSearch_tool_GivenWhenThen") {
		t.Error("expected test function name")
	}
	if !strings.Contains(code, "Given: a known input state") {
		t.Error("expected Given section")
	}
	if !strings.Contains(code, "When: the tool is invoked") {
		t.Error("expected When section")
	}
	if !strings.Contains(code, "Then: verify") {
		t.Error("expected Then section")
	}
	if !strings.Contains(code, `"results"`) {
		t.Error("expected results output check")
	}
	if !strings.Contains(code, `"total"`) {
		t.Error("expected total output check")
	}
}

func TestScaffoldGenerator_GeneratePythonTest(t *testing.T) {
	sg := NewScaffoldGenerator()
	inputs := map[string]string{
		"name": "string",
	}
	outputs := []string{"greeting"}

	code := sg.GeneratePythonTest("hello_tool", inputs, outputs)

	if !strings.Contains(code, "test_given_when_then") {
		t.Error("expected test method name")
	}
	if !strings.Contains(code, "given/when/then pattern") {
		t.Error("expected pattern comment")
	}
	if !strings.Contains(code, "assertIsInstance") {
		t.Error("expected assertIsInstance check")
	}
	if !strings.Contains(code, `self.assertIn("greeting", result)`) {
		t.Error("expected greeting output check")
	}
}

func TestScaffoldGenerator_GenerateGivenWhenThen(t *testing.T) {
	sg := NewScaffoldGenerator()
	inputExample := map[string]interface{}{
		"query": "test",
	}
	outputExample := map[string]interface{}{
		"results": "list",
	}

	desc := sg.GenerateGivenWhenThen("search", "Searches for items", inputExample, outputExample)

	if !strings.Contains(desc, "Given:") {
		t.Error("expected Given section")
	}
	if !strings.Contains(desc, "When:") {
		t.Error("expected When section")
	}
	if !strings.Contains(desc, "Then:") {
		t.Error("expected Then section")
	}
}

func TestGenerateExampleInput(t *testing.T) {
	inputs := map[string]string{
		"name":  "string",
		"count": "int",
		"ratio": "float",
		"flag":  "bool",
	}

	json := GenerateExampleInput(inputs)

	if !strings.Contains(json, `"name": "example_value"`) {
		t.Error("expected example string value")
	}
	if !strings.Contains(json, `"count": 42`) {
		t.Error("expected example int value")
	}
	if !strings.Contains(json, `"ratio": 3.14`) {
		t.Error("expected example float value")
	}
	if !strings.Contains(json, `"flag": true`) {
		t.Error("expected example bool value")
	}
}

func TestGenerateExampleOutput(t *testing.T) {
	outputs := []string{"result", "score"}
	json := GenerateExampleOutput(outputs)
	if !strings.Contains(json, `"result": null`) {
		t.Error("expected result output")
	}
	if !strings.Contains(json, `"score": null`) {
		t.Error("expected score output")
	}
}

func TestNewScaffoldGenerator(t *testing.T) {
	sg := NewScaffoldGenerator()
	if sg == nil {
		t.Fatal("NewScaffoldGenerator returned nil")
	}
}

func TestToUpperFirst(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "Hello"},
		{"Hello", "Hello"},
		{"", ""},
		{"a", "A"},
		{"123", "123"},
	}
	for _, tc := range tests {
		got := toUpperFirst(tc.input)
		if got != tc.expected {
			t.Errorf("toUpperFirst(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestGoZeroValue(t *testing.T) {
	tests := []struct {
		typ      string
		expected string
	}{
		{"string", `""`},
		{"int", "0"},
		{"float", "0.0"},
		{"float64", "0.0"},
		{"bool", "false"},
		{"unknown", `""`},
	}
	for _, tc := range tests {
		got := goZeroValue(tc.typ)
		if got != tc.expected {
			t.Errorf("goZeroValue(%q) = %q, want %q", tc.typ, got, tc.expected)
		}
	}
}

func TestPythonZeroValue(t *testing.T) {
	tests := []struct {
		typ      string
		expected string
	}{
		{"string", `""`},
		{"str", `""`},
		{"int", "0"},
		{"float", "0.0"},
		{"bool", "False"},
		{"unknown", "None"},
	}
	for _, tc := range tests {
		got := pythonZeroValue(tc.typ)
		if got != tc.expected {
			t.Errorf("pythonZeroValue(%q) = %q, want %q", tc.typ, got, tc.expected)
		}
	}
}
