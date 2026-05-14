package sandbox

import (
	"fmt"
	"strings"
)

// TestScaffold holds the generated test code for a tool.
type TestScaffold struct {
	ToolName       string
	Language       string // "go" or "python"
	GoTestCode     string
	PythonTestCode string
	GivenWhenThen  string
	ExampleInput   string
	ExampleOutput  string
}

// ScaffoldGenerator creates test scaffolding for tools using the
// given/when/then pattern.
type ScaffoldGenerator struct{}

// NewScaffoldGenerator creates a new scaffold generator.
func NewScaffoldGenerator() *ScaffoldGenerator {
	return &ScaffoldGenerator{}
}

// GenerateGoTest produces a Go test file with given/when/then pattern
// for a tool with the specified inputs and outputs.
// inputs maps parameter names to their types ("string", "int", "float", "bool").
func (sg *ScaffoldGenerator) GenerateGoTest(toolName string, inputs map[string]string, outputs []string) string {
	var givenFields []string
	for name, typ := range inputs {
		givenFields = append(givenFields, fmt.Sprintf("\t\t%s: %s,", name, goZeroValue(typ)))
	}

	var thenChecks []string
	for _, out := range outputs {
		thenChecks = append(thenChecks, fmt.Sprintf("\t\t\t// Then: verify \"%s\" is present in output", out))
		thenChecks = append(thenChecks, fmt.Sprintf("\t\t\tif _, ok := output[%q]; !ok {", out))
		thenChecks = append(thenChecks, fmt.Sprintf("\t\t\t\tt.Errorf(\"missing expected output key: %%s\", %q)", out))
		thenChecks = append(thenChecks, "\t\t\t}")
	}

	givenStr := strings.Join(givenFields, "\n")
	thenStr := strings.Join(thenChecks, "\n")

	return fmt.Sprintf(`package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestHandle%s_GivenWhenThen(t *testing.T) {
	// Given: a known input state
	given := %sInput{
%s
	}
	inputJSON, _ := json.Marshal(given)

	// When: the tool is invoked
	result, err := Handle%s(context.TODO(), string(inputJSON))
	if err != nil {
		t.Fatalf("Handle%s() error = %%v", err)
	}

	// Then: verify expected outcomes
	var output map[string]interface{}
	if err := json.Unmarshal([]byte(result), &output); err != nil {
		t.Fatalf("invalid JSON output: %%v", err)
	}
%s
}
`, toUpperFirst(toolName), toUpperFirst(toolName),
		givenStr,
		toUpperFirst(toolName), toUpperFirst(toolName),
		thenStr)
}

// GeneratePythonTest produces a Python test file with given/when/then pattern.
func (sg *ScaffoldGenerator) GeneratePythonTest(toolName string, inputs map[string]string, outputs []string) string {
	var givenVars []string
	for name, typ := range inputs {
		givenVars = append(givenVars, fmt.Sprintf("        %s = %s", name, pythonZeroValue(typ)))
	}

	var thenChecks []string
	for _, out := range outputs {
		thenChecks = append(thenChecks, fmt.Sprintf("        # Then: verify \"%s\" in result", out))
		thenChecks = append(thenChecks, fmt.Sprintf("        self.assertIn(%q, result)", out))
	}

	return fmt.Sprintf(`"""Tests for %s using the given/when/then pattern."""
import json
import unittest
from typing import Any


class Test%s(unittest.TestCase):
    """Test suite for %s."""

    def test_given_when_then(self) -> None:
        """Given a known input, when the tool runs, then verify output."""
        # Given: a known input state
%s

        # When: the tool is invoked
        result = handle_%s(**{
%s
        })

        # Then: verify expected outcomes
        self.assertIsInstance(result, dict)
%s


if __name__ == "__main__":
    unittest.main()
`, toolName, toUpperFirst(toolName), toolName,
		strings.Join(givenVars, "\n"),
		toolName,
		strings.Join(givenVars, "\n"),
		strings.Join(thenChecks, "\n"))
}

// GenerateGivenWhenThen produces a structured given/when/then description.
// It describes the test scenario with example input and expected output values.
func (sg *ScaffoldGenerator) GenerateGivenWhenThen(toolName, description string, inputExample, outputExample map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s Test Scenarios\n", toolName))
	b.WriteString(fmt.Sprintf("  %s\n", description))
	b.WriteString("\n  Given:\n")
	for k, v := range inputExample {
		b.WriteString(fmt.Sprintf("    %s = %v\n", k, v))
	}
	b.WriteString("  When: tool is executed\n")
	b.WriteString("  Then:\n")
	for k, v := range outputExample {
		b.WriteString(fmt.Sprintf("    %s = %v\n", k, v))
	}
	return b.String()
}

// GenerateExampleInput creates example input JSON from a parameter map.
func GenerateExampleInput(inputs map[string]string) string {
	var pairs []string
	for name, typ := range inputs {
		pairs = append(pairs, fmt.Sprintf("  %q: %s", name, exampleValue(typ)))
	}
	return "{\n" + strings.Join(pairs, ",\n") + "\n}"
}

// GenerateExampleOutput creates example output JSON from a list of output names.
func GenerateExampleOutput(outputs []string) string {
	var pairs []string
	for _, name := range outputs {
		pairs = append(pairs, fmt.Sprintf("  %q: null", name))
	}
	return "{\n" + strings.Join(pairs, ",\n") + "\n}"
}

// toUpperFirst upper-cases the first rune of s.
func toUpperFirst(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// goZeroValue returns the Go zero-value literal for a type string.
func goZeroValue(typ string) string {
	switch typ {
	case "string":
		return `""`
	case "int":
		return "0"
	case "float", "float64":
		return "0.0"
	case "bool":
		return "false"
	default:
		return `""`
	}
}

// pythonZeroValue returns the Python zero-value literal for a type string.
func pythonZeroValue(typ string) string {
	switch typ {
	case "string", "str":
		return `""`
	case "int":
		return "0"
	case "float":
		return "0.0"
	case "bool":
		return "False"
	default:
		return "None"
	}
}

// exampleValue returns an example literal value for a given type.
func exampleValue(typ string) string {
	switch typ {
	case "string":
		return `"example_value"`
	case "int":
		return "42"
	case "float", "float64":
		return "3.14"
	case "bool":
		return "true"
	default:
		return `"example_value"`
	}
}
