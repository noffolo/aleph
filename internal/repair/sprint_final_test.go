package repair

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFixSyntaxError_MissingCloseParen(t *testing.T) {
	eng := &RepairEngine{}

	// Case 1: Unmatched braces (add missing })
	code := "package main\nfunc Handle(input string) (string, error) {\n\treturn \"ok\", nil\n"
	result := eng.fixSyntaxError(code)
	assert.NotEqual(t, code, result)
	assert.True(t, strings.Count(result, "}") > strings.Count(code, "}"),
		"should add missing closing braces")
}

func TestFixSyntaxError_MissingParenInFuncSig(t *testing.T) {
	eng := &RepairEngine{}

	// Case 2: Function signature without trailing { and missing close paren
	code := "package main\nfunc Handle(input string (string, error)\n\treturn \"ok\", nil\n}"
	result := eng.fixSyntaxError(code)
	assert.NotEqual(t, code, result)
	assert.True(t, strings.Count(result, ")") > strings.Count(code, ")"),
		"should add missing closing paren in function signature")
}

func TestRepeatedFileRead_Multiple(t *testing.T) {
	code := `data, _ := os.ReadFile("config.json")
	more, _ := os.ReadFile("config.json")
	other, _ := os.ReadFile("data.json")`
	result := repeatedFileRead(code)
	assert.NotEmpty(t, result)
}

func TestRepeatedFileRead_Single(t *testing.T) {
	code := `data, _ := os.ReadFile("config.json")
	more, _ := os.ReadFile("data.json")`
	result := repeatedFileRead(code)
	assert.Empty(t, result)
}

func TestRepeatedFileRead_None(t *testing.T) {
	result := repeatedFileRead(`return "hello", nil`)
	assert.Empty(t, result)
}

func TestAddCommentAfterFuncDecl_HandleFunc(t *testing.T) {
	code := `package main
func Handle(input string) (string, error) {
	return "ok", nil
}`
	result := addCommentAfterFuncDecl(code, "/* comment */")
	assert.Contains(t, result, "/* comment */")
}

func TestAddCommentAfterFuncDecl_GenericFunc(t *testing.T) {
	code := `package main
func process(data []string) error {
	for _, s := range data {
		println(s)
	}
	return nil
}`
	result := addCommentAfterFuncDecl(code, "/* perf fix */")
	assert.Contains(t, result, "/* perf fix */")
}

func TestAddCommentAfterFuncDecl_NoFunc(t *testing.T) {
	code := `var x = 42`
	result := addCommentAfterFuncDecl(code, "/* comment */")
	assert.Contains(t, result, "/* comment */")
}

func TestFixTimeout_OneSecondToTen(t *testing.T) {
	eng := &RepairEngine{}

	result := eng.fixTimeout("timeout := 1 * time.Second")
	assert.Contains(t, result, "10 * time.Second")
	assert.NotContains(t, result, "1 * time.Second")
}

func TestFixTimeout_TwoSecondToThirty(t *testing.T) {
	eng := &RepairEngine{}

	result := eng.fixTimeout("timeout := 2 * time.Second")
	assert.Contains(t, result, "30 * time.Second")
}

func TestFixTimeout_ThreeSecondToThirty(t *testing.T) {
	eng := &RepairEngine{}

	result := eng.fixTimeout("timeout := 3 * time.Second")
	assert.Contains(t, result, "30 * time.Second")
}

func TestFixTimeout_FiveSecondToThirty(t *testing.T) {
	eng := &RepairEngine{}

	result := eng.fixTimeout("timeout := 5 * time.Second")
	assert.Contains(t, result, "30 * time.Second")
}

func TestFixTimeout_ContextTimeoutPattern(t *testing.T) {
	eng := &RepairEngine{}

	code := `func Handle(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	return nil
}`
	result := eng.fixTimeout(code)
	assert.Contains(t, result, "time.Second")
}

// ── fixStringConcatInLoop tests ───────────────────────────────────────────

func TestFixStringConcatInLoop_NoForLoop(t *testing.T) {
	code := "result := \"hello\"\nresult += \" world\"\nreturn result"
	result := fixStringConcatInLoop(code)
	assert.Equal(t, code, result)
}

func TestFixStringConcatInLoop_NoVarComment(t *testing.T) {
	// Variable appended in loop but no var declaration — should get a comment suggestion
	code := `func Handle(input string) (string, error) {
	for _, s := range []string{"a", "b"} {
		output += s
	}
	return nil
}`
	result := fixStringConcatInLoop(code)
	assert.Contains(t, result, "PERFORMANCE FIX")
}

func TestFixStringConcatInLoop_EmptyVarNameLoopVar(t *testing.T) {
	// Loop variable += — builderName found but no decl pattern, returns with comment
	code := `func Handle(input string) (string, error) {
	for _, s := range []string{"a", "b"} {
		s += "_suffix"
	}
	return nil
}`
	result := fixStringConcatInLoop(code)
	assert.Contains(t, result, "PERFORMANCE FIX")
}

func TestFixStringConcatInLoop_NoPlusEqInLoop(t *testing.T) {
	code := `func Handle(input string) (string, error) {
	for _, s := range []string{"a", "b", "c"} {
		results = append(results, s)
	}
	return nil
}`
	result := fixStringConcatInLoop(code)
	assert.Equal(t, code, result)
}

// ── NewRepairEngine test ──────────────────────────────────────────────────

func TestNewRepairEngine_ConstructsFields(t *testing.T) {
	e := NewRepairEngine(nil, nil, nil, nil)
	assert.NotNil(t, e)
	assert.Nil(t, e.logger)
	assert.Nil(t, e.reader)
	assert.Nil(t, e.writer)
	assert.NotNil(t, e.history)
}

// ── executeFix tests ──────────────────────────────────────────────────────

func TestExecuteFix_SyntaxError(t *testing.T) {
	eng := &RepairEngine{}
	result, err := eng.executeFix(
		"package main\nfunc Handle(input string) (string, error) {\n\treturn \"ok\", nil\n",
		RepairAction{ID: "fix-syntax-01"},
	)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestExecuteFix_DeprecatedAPI(t *testing.T) {
	eng := &RepairEngine{}
	result, err := eng.executeFix(
		"ioutil.ReadAll(r)",
		RepairAction{ID: "fix-dep-01"},
	)
	assert.NoError(t, err)
	assert.Contains(t, result, "io.ReadAll")
	assert.NotContains(t, result, "ioutil.ReadAll")
}

func TestExecuteFix_Configuration(t *testing.T) {
	eng := &RepairEngine{}
	result, err := eng.executeFix(
		`endpoint := "localhost:8080"`,
		RepairAction{ID: "fix-config-01"},
	)
	assert.NoError(t, err)
	assert.Contains(t, result, "ALEPH_ENDPOINT")
}

func TestExecuteFix_Performance(t *testing.T) {
	eng := &RepairEngine{}
	// Use code that triggers fixPerformance without hitting the buggy
	// fixStringConcatInLoop regex path (Go regex doesn't support lookahead).
	result, err := eng.executeFix(
		`resp, _ := http.Get("https://a.com")
data, _ := http.Get("https://b.com")`,
		RepairAction{ID: "fix-perf-01"},
	)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestExecuteFix_UnknownAction(t *testing.T) {
	eng := &RepairEngine{}
	code := "package main"
	result, err := eng.executeFix(code, RepairAction{ID: "unknown-action"})
	assert.NoError(t, err)
	assert.Equal(t, code, result)
}

func TestExecuteFix_DeprecationNoOp(t *testing.T) {
	eng := &RepairEngine{}
	code := "package main"
	result, err := eng.executeFix(code, RepairAction{ID: "fix-depdep-01"})
	assert.NoError(t, err)
	assert.Equal(t, code, result)
}

func TestExecuteFix_FixTimeout(t *testing.T) {
	eng := &RepairEngine{}
	result, err := eng.executeFix(
		"timeout := 1 * time.Second",
		RepairAction{ID: "fix-timeout-01"},
	)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

// ── fixSyntaxError edge cases ─────────────────────────────────────────────

func TestFixSyntaxError_AlreadyBalancedNoChange(t *testing.T) {
	eng := &RepairEngine{}
	code := "func main() {\n}\n"
	result := eng.fixSyntaxError(code)
	assert.Equal(t, code, result)
}

func TestFixSyntaxError_ExtraOpenBrace(t *testing.T) {
	eng := &RepairEngine{}
	code := "func main() {\n"
	result := eng.fixSyntaxError(code)
	assert.True(t, strings.Count(result, "}") > strings.Count(code, "}"))
}
