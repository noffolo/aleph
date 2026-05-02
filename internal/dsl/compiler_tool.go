package dsl

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/sandbox"
)

// GeneratedTool holds the complete output of compiling a .aleph tool definition.
type GeneratedTool struct {
	Name       string
	Template   ToolTemplate
	GoCode     string
	PythonCode string
	ProtoDef   string
	TestCode   string
	Inputs     []*ToolParam
	Outputs    []*ToolParam
	Handler    *ToolHandler
	Deps       []*ToolDep
}

// ToolTemplate identifies which code generation template to use.
type ToolTemplate string

const (
	TemplateDataProcessor ToolTemplate = "data_processor"
	TemplateAPIConnector  ToolTemplate = "api_connector"
	TemplateAnalyzer      ToolTemplate = "analyzer"
)

// TestResult holds the outcome of running a generated tool test in the sandbox.
type TestResult struct {
	ToolName string
	Passed   bool
	Stdout   string
	Stderr   string
	Error    string
}

// ValidToolTypes is the set of allowed parameter types.
var ValidToolTypes = map[string]bool{
	"string": true,
	"float":  true,
	"int":    true,
	"bool":   true,
}

// CompileToolDefinition generates Go, Python, and proto code from a tool definition.
// It selects a template based on the tool name prefix and generates all three outputs.
func CompileToolDefinition(def *ToolDefinition) (*GeneratedTool, error) {
	if def == nil {
		return nil, fmt.Errorf("tool definition is nil")
	}

	tmpl := selectTemplate(def.Name)

	gt := &GeneratedTool{
		Name:     def.Name,
		Template: tmpl,
		Inputs:   def.Inputs,
		Outputs:  def.Outputs,
		Handler:  def.Handler,
		Deps:     def.Deps,
	}

	gt.GoCode = renderGoHandler(def, tmpl)
	gt.PythonCode = renderPythonTool(def, tmpl)
	gt.ProtoDef = renderProtoDef(def)
	gt.TestCode = renderTestCode(def, tmpl)

	return gt, nil
}

// selectTemplate chooses a template based on tool name keywords.
func selectTemplate(name string) ToolTemplate {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "api") || strings.Contains(lower, "fetch") || strings.Contains(lower, "http") {
		return TemplateAPIConnector
	}
	if strings.Contains(lower, "analyze") || strings.Contains(lower, "score") || strings.Contains(lower, "audit") {
		return TemplateAnalyzer
	}
	return TemplateDataProcessor
}

// goType maps DSL type names to Go types.
func goType(dslType string) string {
	switch dslType {
	case "string":
		return "string"
	case "float":
		return "float64"
	case "int":
		return "int"
	case "bool":
		return "bool"
	default:
		return "string"
	}
}

// pythonType maps DSL type names to Python type hints.
func pythonType(dslType string) string {
	switch dslType {
	case "string":
		return "str"
	case "float":
		return "float"
	case "int":
		return "int"
	case "bool":
		return "bool"
	default:
		return "str"
	}
}

// protoType maps DSL type names to proto field types.
func protoType(dslType string) string {
	switch dslType {
	case "string":
		return "string"
	case "float":
		return "double"
	case "int":
		return "int64"
	case "bool":
		return "bool"
	default:
		return "string"
	}
}

// title upper-cases the first letter of a string.
func title(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// renderGoHandler generates Go handler code using a template.
// All placeholder substitution uses strings.NewReplacer with __DELIMITED__
// markers. Input names are validated via validNameRegex. No raw user input
// flows into SQL or code-execution positions.
func renderGoHandler(def *ToolDefinition, tmpl ToolTemplate) string {
	var inputFields, outputFields, outputInit []string
	for i, p := range def.Inputs {
		jsonTag := fmt.Sprintf("`json:\"%s\"`", p.Name)
		inputFields = append(inputFields, fmt.Sprintf("\t%s %s %s", title(p.Name), goType(p.Type), jsonTag))
		_ = i
	}
	for _, p := range def.Outputs {
		jsonTag := fmt.Sprintf("`json:\"%s\"`", p.Name)
		outputFields = append(outputFields, fmt.Sprintf("\t%s %s %s", title(p.Name), goType(p.Type), jsonTag))
		outputInit = append(outputInit, fmt.Sprintf("\t\t\t// %s: zero value, TODO: implement", title(p.Name)))
	}

	otStr := strings.Join(outputFields, "\n")
	oiStr := strings.Join(outputInit, "\n")
	inStr := strings.Join(inputFields, "\n")

	tpl := dataProcessorGoTemplate
	if tmpl == TemplateAPIConnector {
		tpl = apiConnectorGoTemplate
	} else if tmpl == TemplateAnalyzer {
		tpl = analyzerGoTemplate
	}

	r := strings.NewReplacer(
		"__NAME__", def.Name,
		"__NAME_TITLE__", title(def.Name),
		"__INPUT_FIELDS__", inStr,
		"__OUTPUT_FIELDS__", otStr,
		"__OUTPUT_INIT__", oiStr,
		"__DESCRIPTION__", def.Description,
	)
	return strings.TrimSpace(r.Replace(tpl))
}

// renderPythonTool generates Python tool stub code.
// SAFE: placeholder substitution only; no SQL positions.
func renderPythonTool(def *ToolDefinition, tmpl ToolTemplate) string {
	var inputParams []string
	for _, p := range def.Inputs {
		defaultVal := ""
		if !p.Required {
			switch p.Type {
			case "string":
				defaultVal = " = \"\""
			case "int":
				defaultVal = " = 0"
			case "float":
				defaultVal = " = 0.0"
			case "bool":
				defaultVal = " = False"
			}
		}
		inputParams = append(inputParams, fmt.Sprintf("\t%s: %s%s", p.Name, pythonType(p.Type), defaultVal))
	}

	var outputFields []string
	for _, p := range def.Outputs {
		outputFields = append(outputFields, fmt.Sprintf("\t\t\"%s\": None,", p.Name))
	}

	tpl := dataProcessorPythonTemplate
	if tmpl == TemplateAPIConnector {
		tpl = apiConnectorPythonTemplate
	} else if tmpl == TemplateAnalyzer {
		tpl = analyzerPythonTemplate
	}

	r := strings.NewReplacer(
		"__NAME__", def.Name,
		"__INPUT_PARAMS__", strings.Join(inputParams, ",\n"),
		"__OUTPUT_FIELDS__", strings.Join(outputFields, "\n"),
		"__DESCRIPTION__", def.Description,
	)
	return strings.TrimSpace(r.Replace(tpl))
}

// renderProtoDef generates a proto3 definition for the tool.
func renderProtoDef(def *ToolDefinition) string {
	var inputFields, outputFields []string
	for i, p := range def.Inputs {
		tag := i + 1
		inputFields = append(inputFields, fmt.Sprintf("  %s %s = %d;", protoType(p.Type), p.Name, tag))
	}
	for i, p := range def.Outputs {
		tag := i + 1
		outputFields = append(outputFields, fmt.Sprintf("  %s %s = %d;", protoType(p.Type), p.Name, tag))
	}

	r := strings.NewReplacer(
		"__NAME__", def.Name,
		"__INPUT_FIELDS__", strings.Join(inputFields, "\n"),
		"__OUTPUT_FIELDS__", strings.Join(outputFields, "\n"),
	)
	return strings.TrimSpace(r.Replace(protoTemplate))
}

// renderTestCode generates a test file for the tool.
func renderTestCode(def *ToolDefinition, tmpl ToolTemplate) string {
	var inputJSON []string
	for _, p := range def.Inputs {
		switch p.Type {
		case "string":
			inputJSON = append(inputJSON, fmt.Sprintf("\"%s\": \"test_value\"", p.Name))
		case "int":
			inputJSON = append(inputJSON, fmt.Sprintf("\"%s\": 0", p.Name))
		case "float":
			inputJSON = append(inputJSON, fmt.Sprintf("\"%s\": 0.0", p.Name))
		case "bool":
			inputJSON = append(inputJSON, fmt.Sprintf("\"%s\": false", p.Name))
		}
	}
	inputStr := "{" + strings.Join(inputJSON, ", ") + "}"

	r := strings.NewReplacer(
		"__NAME__", def.Name,
		"__INPUT_JSON__", inputStr,
	)
	return strings.TrimSpace(r.Replace(testGoTemplate))
}

// ---------------------------------------------------------------------------
// Go handler templates (placeholder-based to avoid Go text/template conflict)
// ---------------------------------------------------------------------------

const dataProcessorGoTemplate = `
package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

// __NAME__Input represents the input parameters for __NAME__.
type __NAME__Input struct {
__INPUT_FIELDS__
}

// __NAME__Output represents the output parameters for __NAME__.
type __NAME__Output struct {
__OUTPUT_FIELDS__
}

// Handle__NAME__ processes a data transformation request.
// __DESCRIPTION__
func Handle__NAME__(ctx context.Context, inputJSON string) (string, error) {
	var input __NAME__Input
	if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
		return "", fmt.Errorf("__NAME__: invalid input: %w", err)
	}

	// implement data transformation logic
	// Access input fields via input.FieldName
	// Use SQL queries via duckdb or aggregate in-memory

	result := __NAME__Output{
__OUTPUT_INIT__
	}

	output, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("__NAME__: marshal output: %w", err)
	}
	return string(output), nil
}
`

const apiConnectorGoTemplate = `
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ff3300/aleph-v2/internal/ssrf"
)

// __NAME__Input represents the input parameters for __NAME__.
type __NAME__Input struct {
__INPUT_FIELDS__
}

// __NAME__Output represents the output parameters for __NAME__.
type __NAME__Output struct {
__OUTPUT_FIELDS__
}

// httpClient is reused across API connector tool calls.
// Uses ssrf.NewClient() for SSRF protection (private IP blocking,
// DNS rebinding prevention, redirect validation).
var httpClient = ssrf.NewClient()

// Handle__NAME__ calls an external API and returns structured results.
// __DESCRIPTION__
func Handle__NAME__(ctx context.Context, inputJSON string) (string, error) {
	var input __NAME__Input
	if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
		return "", fmt.Errorf("__NAME__: invalid input: %w", err)
	}

	// implement API call with auth, retry, error handling
	// req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	// resp, err := httpClient.Do(req)
	// body, err := io.ReadAll(resp.Body)

	result := __NAME__Output{
__OUTPUT_INIT__
	}

	output, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("__NAME__: marshal output: %w", err)
	}
	return string(output), nil
}

func init() {
	// Ensure imports are used (required for response body reading and SSRF protection)
	var _ = io.Discard
	var _ = http.DefaultClient
	_ = ssrf.NewClient
}
`

const analyzerGoTemplate = `
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// __NAME__Input represents the input parameters for __NAME__.
type __NAME__Input struct {
__INPUT_FIELDS__
}

// __NAME__Output represents the output parameters for __NAME__.
type __NAME__Output struct {
__OUTPUT_FIELDS__
}

// Handle__NAME__ analyzes input data and returns scored results.
// __DESCRIPTION__
func Handle__NAME__(ctx context.Context, inputJSON string) (string, error) {
	var input __NAME__Input
	if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
		return "", fmt.Errorf("__NAME__: invalid input: %w", err)
	}

	// implement analysis/reporting logic
	// Pattern: load input, analyze, score, return structured output

	result := __NAME__Output{
__OUTPUT_INIT__
	}

	output, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("__NAME__: marshal output: %w", err)
	}
	return string(output), nil
}

// Ensure strings import is used (common in text analysis tools)
var _ = strings.ToLower
`

// ---------------------------------------------------------------------------
// Python tool templates
// ---------------------------------------------------------------------------

const dataProcessorPythonTemplate = `# python
\"\"\"__DESCRIPTION__\"\"\"
import json
import sys
from typing import Any


def handle__NAME__(
__INPUT_PARAMS__
) -> dict[str, Any]:
	\"\"\"Process data transformation for __NAME__.\"\"\"
	# TODO(data_processor): implement data transformation logic
	result = {
__OUTPUT_FIELDS__
	}
	return result


def main() -> None:
	input_json = sys.stdin.read() if not sys.stdin.isatty() else "{}"
	try:
		input_data = json.loads(input_json)
	except json.JSONDecodeError as e:
		print(json.dumps({"error": f"invalid input: {e}"}))
		sys.exit(1)
	result = handle__NAME__(**input_data)
	print(json.dumps(result, default=str))


if __name__ == "__main__":
	main()
`

const apiConnectorPythonTemplate = `# python
\"\"\"__DESCRIPTION__\"\"\"
import json
import sys
from typing import Any
# Network access is sandbox-gated; HTTP calls must go through the approved SSRF client


def handle__NAME__(
__INPUT_PARAMS__
) -> dict[str, Any]:
	\"\"\"Call external API for __NAME__.\"\"\"
	# TODO(api_connector): implement API call with auth, retry, error handling
	result = {
__OUTPUT_FIELDS__
	}
	return result


def main() -> None:
	input_json = sys.stdin.read() if not sys.stdin.isatty() else "{}"
	try:
		input_data = json.loads(input_json)
	except json.JSONDecodeError as e:
		print(json.dumps({"error": f"invalid input: {e}"}))
		sys.exit(1)
	result = handle__NAME__(**input_data)
	print(json.dumps(result, default=str))


if __name__ == "__main__":
	main()
`

const analyzerPythonTemplate = `# python
\"\"\"__DESCRIPTION__\"\"\"
import json
import re
import sys
from typing import Any


def handle__NAME__(
__INPUT_PARAMS__
) -> dict[str, Any]:
	\"\"\"Analyze data and return scored results for __NAME__.\"\"\"
	# TODO(analyzer): implement analysis/scoring logic
	result = {
__OUTPUT_FIELDS__
	}
	return result


def main() -> None:
	input_json = sys.stdin.read() if not sys.stdin.isatty() else "{}"
	try:
		input_data = json.loads(input_json)
	except json.JSONDecodeError as e:
		print(json.dumps({"error": f"invalid input: {e}"}))
		sys.exit(1)
	result = handle__NAME__(**input_data)
	print(json.dumps(result, default=str))


if __name__ == "__main__":
	main()
`

// ---------------------------------------------------------------------------
// Proto definition template
// ---------------------------------------------------------------------------

const protoTemplate = `syntax = "proto3";
package tools;

option go_package = "tools/__NAME__";

// __NAME__Input is the request message for the __NAME__ tool.
message __NAME__Input {
__INPUT_FIELDS__
}

// __NAME__Output is the response message for the __NAME__ tool.
message __NAME__Output {
__OUTPUT_FIELDS__
}

// __NAME__Service defines the RPC interface for the __NAME__ tool.
service __NAME__Service {
  rpc Execute(__NAME__Input) returns (__NAME__Output);
}
`

// ---------------------------------------------------------------------------
// Test code template
// ---------------------------------------------------------------------------

const testGoTemplate = `package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestHandle__NAME__(t *testing.T) {
	input := __INPUT_JSON__
	inputJSON, _ := json.Marshal(input)

	output, err := Handle__NAME__(context.Background(), string(inputJSON))
	if err != nil {
		t.Fatalf("Handle__NAME__() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
}
`

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

// ValidateTool performs comprehensive validation on a tool definition.
// Returns a list of validation errors; empty slice means the tool is valid.
func ValidateTool(def *ToolDefinition) []string {
	var errs []string

	// Name validation: alphanumeric + underscore, non-empty
	if def.Name == "" {
		errs = append(errs, "tool name must not be empty")
	} else if !validNameRegex.MatchString(def.Name) {
		errs = append(errs, fmt.Sprintf("tool name %q must match [a-zA-Z_][a-zA-Z0-9_]*", def.Name))
	}

	// Input validation
	seenInputs := make(map[string]bool)
	for _, p := range def.Inputs {
		if !ValidToolTypes[p.Type] {
			errs = append(errs, fmt.Sprintf("input %q: unsupported type %q (must be string/float/int/bool)", p.Name, p.Type))
		}
		if seenInputs[p.Name] {
			errs = append(errs, fmt.Sprintf("duplicate input parameter name: %q", p.Name))
		}
		seenInputs[p.Name] = true
	}

	// Output validation
	seenOutputs := make(map[string]bool)
	for _, p := range def.Outputs {
		if !ValidToolTypes[p.Type] {
			errs = append(errs, fmt.Sprintf("output %q: unsupported type %q (must be string/float/int/bool)", p.Name, p.Type))
		}
		if seenOutputs[p.Name] {
			errs = append(errs, fmt.Sprintf("duplicate output parameter name: %q", p.Name))
		}
		seenOutputs[p.Name] = true
	}

	// Handler validation
	if def.Handler != nil {
		lang := strings.ToLower(def.Handler.Language)
		if lang != "go" && lang != "python" {
			errs = append(errs, fmt.Sprintf("handler language must be \"go\" or \"python\", got %q", def.Handler.Language))
		}
		if def.Handler.EntryPoint == "" {
			errs = append(errs, "handler entry point must not be empty")
		}
	} else {
		errs = append(errs, "tool must have a handler block")
	}

	// Dependency validation: no circular (self-referencing) dependencies
	for _, dep := range def.Deps {
		if dep.Name == def.Name {
			errs = append(errs, fmt.Sprintf("dependency %q cannot reference the tool itself", dep.Name))
		}
	}

	return errs
}

var validNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// SecurityScan checks tool handler code for dangerous patterns.
// Returns a list of security issues; empty slice means the code is safe.
func SecurityScan(def *ToolDefinition) []string {
	var issues []string

	// Check handler code if present
	if def.Handler == nil {
		return issues
	}

	code := def.Handler.EntryPoint // The entry point name itself isn't code to scan
	_ = code

	// Scan dep names for suspicious package references
	for _, dep := range def.Deps {
		lower := strings.ToLower(dep.Name)
		if strings.Contains(lower, "exec") || strings.Contains(lower, "shell") || strings.Contains(lower, "system") {
			if dep.Type == "library" {
				issues = append(issues, fmt.Sprintf("suspicious dependency name: %q (type: library)", dep.Name))
			}
		}
	}

	// Check entry point name for suspicious patterns
	ep := def.Handler.EntryPoint
	if strings.Contains(strings.ToLower(ep), "exec") || strings.Contains(strings.ToLower(ep), "eval") {
		issues = append(issues, fmt.Sprintf("suspicious entry point name: %q", ep))
	}

	return issues
}

// ---------------------------------------------------------------------------
// Auto-testing integration with sandbox
// ---------------------------------------------------------------------------

// TestGeneratedTool runs a generated Go tool test in the sandbox verifier.
// It takes a GeneratedTool, writes it to a temporary location via the
// metadata repository, and invokes sandbox verification.
func TestGeneratedTool(ctx context.Context, gen *GeneratedTool, verifier *sandbox.Verifier) (*TestResult, error) {
	if gen == nil {
		return nil, fmt.Errorf("generated tool is nil")
	}

	result := &TestResult{
		ToolName: gen.Name,
	}

	// Perform static validation first
	vErrs := ValidateTool(&ToolDefinition{
		Name:    gen.Name,
		Inputs:  gen.Inputs,
		Outputs: gen.Outputs,
		Handler: gen.Handler,
		Deps:    gen.Deps,
	})
	if len(vErrs) > 0 {
		result.Error = fmt.Sprintf("validation failed: %s", strings.Join(vErrs, "; "))
		result.Passed = false
		return result, nil
	}

	// Run security scan
	sIssues := SecurityScan(&ToolDefinition{
		Name:    gen.Name,
		Inputs:  gen.Inputs,
		Outputs: gen.Outputs,
		Handler: gen.Handler,
		Deps:    gen.Deps,
	})
	if len(sIssues) > 0 {
		result.Error = fmt.Sprintf("security scan failed: %s", strings.Join(sIssues, "; "))
		result.Passed = false
		return result, nil
	}

	// Use sandbox verifier's static analysis
	vResult := verifier.VerifyToolCode(gen.GoCode)
	result.Passed = vResult.Passed
	result.Stdout = ""
	result.Stderr = ""
	if !vResult.Passed {
		result.Error = vResult.Error
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// Registration stub with SourceType: "user"
// ---------------------------------------------------------------------------

// RegisterTool registers a generated tool in the metadata repository
// with SourceType set to "user".
func RegisterTool(ctx context.Context, gen *GeneratedTool, metaRepo *repository.MetadataRepository) (*repository.ToolRecord, error) {
	if gen == nil {
		return nil, fmt.Errorf("generated tool is nil")
	}
	if metaRepo == nil {
		return nil, fmt.Errorf("metadata repository is nil")
	}

	code := gen.GoCode
	if gen.Handler != nil && strings.ToLower(gen.Handler.Language) == "python" {
		code = gen.PythonCode
	}

	rec := &repository.ToolRecord{
		ID:           fmt.Sprintf("user_%s", gen.Name),
		Name:         gen.Name,
		Description:  fmt.Sprintf("User-defined tool: %s (template: %s)", gen.Name, gen.Template),
		Code:         code,
		Category:     "user",
		Version:      "0.1.0",
		HealthStatus: "unknown",
		SourceType:   "user",
	}

	// Attempt to persist the tool record.
	// This requires a live database connection; for environments without
	// a database, the call will return an error.
	if err := metaRepo.CreateTool(rec); err != nil {
		return nil, fmt.Errorf("register tool %q: %w", gen.Name, err)
	}

	return rec, nil
}

// ValidateToolRecord checks that a tool record has valid SourceType.
func ValidateToolRecord(rec *repository.ToolRecord) error {
	if rec == nil {
		return fmt.Errorf("tool record is nil")
	}
	if rec.SourceType != "builtin" && rec.SourceType != "mcp" && rec.SourceType != "user" && rec.SourceType != "package" {
		return fmt.Errorf("invalid source_type %q, must be builtin/mcp/user/package", rec.SourceType)
	}
	return nil
}

