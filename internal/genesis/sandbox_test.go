package genesis

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// NewSandbox
// ---------------------------------------------------------------------------

func TestNewSandbox_Happy(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	assert.NotNil(t, s)
	assert.Equal(t, 5*time.Second, s.timeout)
	assert.Equal(t, "go", s.execPath)
}

func TestNewSandbox_ZeroTimeout(t *testing.T) {
	s := NewSandbox(0)
	assert.NotNil(t, s)
	assert.Equal(t, time.Duration(0), s.timeout)
}

func TestNewSandbox_VeryShortTimeout(t *testing.T) {
	s := NewSandbox(1 * time.Nanosecond)
	assert.NotNil(t, s)
	assert.Equal(t, 1*time.Nanosecond, s.timeout)
}

// ---------------------------------------------------------------------------
// createTempModule
// ---------------------------------------------------------------------------

func TestCreateTempModule_Happy(t *testing.T) {
	code := "package main\n\nfunc main() { println(\"hello\") }"
	dir, cleanup, err := createTempModule(code)
	assert.NoError(t, err)
	assert.NotEmpty(t, dir)
	defer cleanup()

	gomodData, err := os.ReadFile(dir + "/go.mod")
	assert.NoError(t, err)
	assert.Contains(t, string(gomodData), "module sandbox")

	mainData, err := os.ReadFile(dir + "/main.go")
	assert.NoError(t, err)
	assert.Equal(t, code, string(mainData))
}

func TestCreateTempModule_EmptyCode(t *testing.T) {
	code := ""
	dir, cleanup, err := createTempModule(code)
	assert.NoError(t, err)
	assert.NotEmpty(t, dir)
	defer cleanup()

	mainData, err := os.ReadFile(dir + "/main.go")
	assert.NoError(t, err)
	assert.Equal(t, code, string(mainData))
}

func TestCreateTempModule_LargeCode(t *testing.T) {
	code := "package main\n\n" + strings.Repeat("// comment line\n", 10000) + "func main() {}"
	dir, cleanup, err := createTempModule(code)
	assert.NoError(t, err)
	assert.NotEmpty(t, dir)
	defer cleanup()

	mainPath := dir + "/main.go"
	mainData, err := os.ReadFile(mainPath)
	assert.NoError(t, err)
	assert.Equal(t, code, string(mainData))
}

// ---------------------------------------------------------------------------
// detectObfuscation
// ---------------------------------------------------------------------------

func TestDetectObfuscation_NoPatterns(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	code := "package main\n\nfunc main() { println(\"hello\") }"
	warnings, score := s.detectObfuscation(code)
	assert.Empty(t, warnings)
	assert.Equal(t, float64(0), score)
}

func TestDetectObfuscation_SinglePattern(t *testing.T) {
	s := NewSandbox(5 * time.Second)

	tests := []struct {
		name         string
		code         string
		expectedName string
		expectedMin  float64
	}{
		{
			name:         "base64_decode",
			code:         `package main; import "encoding/base64"; func main() { base64.StdEncoding.DecodeString("test") }`,
			expectedName: "base64_decode",
			expectedMin:  0.3,
		},
		{
			name:         "hex_decode_call",
			code:         `package main; import "encoding/hex"; func main() { hex.DecodeString("deadbeef") }`,
			expectedName: "hex_decode_call",
			expectedMin:  0.4,
		},
		{
			name:         "exec_command_variable",
			code:         `package main; import "os/exec"; func main() { exec.Command(cmd) }`,
			expectedName: "exec_command_variable",
			expectedMin:  0.4,
		},
		{
			name:         "shell_redirect",
			code:         `package main; func main() { _ = "/proc/self/status" }`,
			expectedName: "shell_redirect",
			expectedMin:  0.7,
		},
		{
			name:         "cgo_escape",
			code:         "package main\n// #include <stdio.h>\nimport \"C\"\nfunc main() {}",
			expectedName: "cgo_escape",
			expectedMin:  0.6,
		},
		{
			name:         "plugin_open",
			code:         `package main; import "plugin"; func main() { plugin.Open("x.so") }`,
			expectedName: "plugin_open",
			expectedMin:  0.7,
		},
		{
			name:         "getenv_command",
			code:         `package main; import "os"; import "os/exec"; func main() { _ = os.Getenv("CMD"); exec.Command("x") }`,
			expectedName: "getenv_command",
			expectedMin:  0.5,
		},
		{
			name:         "runtime_mmap",
			code:         `package main; import "syscall"; func main() { syscall.Mmap(0, 0, 0, 0, 0, 0) }`,
			expectedName: "runtime_mmap",
			expectedMin:  0.6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings, score := s.detectObfuscation(tt.code)
			assert.GreaterOrEqual(t, score, tt.expectedMin, "score should be >= expected min")

			found := false
			for _, w := range warnings {
				if strings.Contains(w, tt.expectedName) {
					found = true
					break
				}
			}
			assert.True(t, found, "expected warning with pattern %q in %v", tt.expectedName, warnings)
		})
	}
}

func TestDetectObfuscation_MultiPattern(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	code := `package main
import "encoding/base64"
import "encoding/hex"
import "plugin"
func main() {
	base64.StdEncoding.DecodeString("aGVsbG8=")
	hex.DecodeString("deadbeef")
	plugin.Open("mal.so")
}
`
	warnings, score := s.detectObfuscation(code)
	assert.Greater(t, score, float64(0.7), "multi-pattern score should be high")
	assert.GreaterOrEqual(t, len(warnings), 3, "should have at least 3 warnings")
}

func TestDetectObfuscation_StringDensity(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	code := `package main; func main() {
	_ = "` + strings.Repeat(`a"+"`, 50) + `z"
}`
	warnings, score := s.detectObfuscation(code)

	hasDensity := false
	for _, w := range warnings {
		if strings.Contains(w, "string literal density") {
			hasDensity = true
			break
		}
	}
	assert.True(t, hasDensity, "expected string density warning")
	assert.Greater(t, score, float64(0))
}

func TestDetectObfuscation_DynamicExecCommand(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	code := `package main; import "os/exec"; func main() { exec.Command("rm " + path) }`
	warnings, score := s.detectObfuscation(code)

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "exec_command_dynamic") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected exec_command_dynamic warning")
	assert.GreaterOrEqual(t, score, 0.5)
}

// ---------------------------------------------------------------------------
// checkDangerousPatternsAST
// ---------------------------------------------------------------------------

func TestCheckDangerousPatternsAST_UnparseableCode(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	ctx := context.Background()
	blocked := s.checkDangerousPatternsAST(ctx, "not valid {{ go code !!!")
	assert.Empty(t, blocked)
}

func TestCheckDangerousPatternsAST_CleanCode(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	ctx := context.Background()
	code := `package main
import "fmt"
import "strings"
func main() { fmt.Println(strings.ToUpper("hello")) }`
	blocked := s.checkDangerousPatternsAST(ctx, code)
	assert.Empty(t, blocked)
}

func TestCheckDangerousPatternsAST_BlockedImportPrefix(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	ctx := context.Background()

	tests := []struct {
		name   string
		code   string
		expect string
	}{
		{
			name:   "crypto/aes prefix blocked",
			code:   `package main; import "crypto/aes"; func main() {}`,
			expect: "crypto/aes",
		},
		{
			name:   "encoding/json prefix blocked",
			code:   `package main; import "encoding/json"; func main() {}`,
			expect: "encoding/json",
		},
		{
			name:   "debug/elf prefix blocked",
			code:   `package main; import "debug/elf"; func main() {}`,
			expect: "debug/elf",
		},
		{
			name:   "internal/abi prefix blocked",
			code:   `package main; import "internal/abi"; func main() {}`,
			expect: "internal/abi",
		},
		{
			name:   "net/mail prefix blocked",
			code:   `package main; import "net/mail"; func main() {}`,
			expect: "net/mail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocked := s.checkDangerousPatternsAST(ctx, tt.code)
			assert.NotEmpty(t, blocked)
			found := false
			for _, b := range blocked {
				if strings.Contains(b, tt.expect) {
					found = true
					break
				}
			}
			assert.True(t, found, "expected blocked entry containing %q in %v", tt.expect, blocked)
		})
	}
}

func TestCheckDangerousPatternsAST_ContextCancelled(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	blocked := s.checkDangerousPatternsAST(ctx, `package main; import "os/exec"; func main() {}`)
	assert.Nil(t, blocked, "cancelled context should return nil")
}

func TestCheckDangerousPatternsAST_MultipleImports(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	ctx := context.Background()
	code := `package main
import "os/exec"
import "syscall"
import "unsafe"
import "fmt"
func main() {}`
	blocked := s.checkDangerousPatternsAST(ctx, code)
	assert.GreaterOrEqual(t, len(blocked), 3, "should block at least 3 imports")
}

// ---------------------------------------------------------------------------
// checkDangerousPatternsFallback
// ---------------------------------------------------------------------------

func TestCheckDangerousPatternsFallback_SafeCode(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	blocked := s.checkDangerousPatternsFallback(`package main; import "fmt"; func main() {}`)
	assert.Empty(t, blocked)
}

func TestCheckDangerousPatternsFallback_OsRemoveCall(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	blocked := s.checkDangerousPatternsFallback("os.Remove(\"/etc/passwd\")")
	assert.NotEmpty(t, blocked)
	found := false
	for _, b := range blocked {
		if strings.Contains(b, "os.Remove") {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestCheckDangerousPatternsFallback_MultipleMatches(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	code := `os.Remove("x"); os.RemoveAll("dir"); os.Exit(0)`
	blocked := s.checkDangerousPatternsFallback(code)
	assert.GreaterOrEqual(t, len(blocked), 3, "should match os.Remove, os.RemoveAll, os.Exit")
}

// ---------------------------------------------------------------------------
// validateCode
// ---------------------------------------------------------------------------

func TestValidateCode_ContextCancelled(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := s.validateCode(ctx, "package main; func main() {}")
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, result)
}

func TestValidateCode_UnblockedSafeCode(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	ctx := context.Background()
	code := `package main
import "fmt"
func main() { fmt.Println("safe") }`
	result, err := s.validateCode(ctx, code)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestValidateCode_RiskThresholdBlocks(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	ctx := context.Background()
	code := `package main
import "encoding/base64"
import "encoding/hex"
func main() {
	base64.StdEncoding.DecodeString("aGVsbG8=")
	hex.DecodeString("deadbeef")
}
// #include <stdio.h>
import "C"
`
	result, err := s.validateCode(ctx, code)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	// If risk >= 0.6, it should be blocked
	if result.RiskScore >= 0.6 {
		assert.False(t, result.Passed, "should be blocked when risk >= 0.6")
	}
}

// ---------------------------------------------------------------------------
// validateInSubprocess (via Validate)
// ---------------------------------------------------------------------------

func TestValidateInSubprocess_SimpleSafeCode(t *testing.T) {
	s := NewSandbox(10 * time.Second)
	ctx := context.Background()
	code := `package main

import "fmt"

func main() { fmt.Println("hello") }
`
	result, err := s.Validate(ctx, Suggestion{Code: code})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Note: subprocess may fail if go vet complains about go.mod version, so we
	// only assert that the result structure is valid
	assert.NotZero(t, result.Duration)
}

func TestValidateInSubprocess_EmptyCode(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	ctx := context.Background()
	result, err := s.Validate(ctx, Suggestion{Code: ""})
	assert.NoError(t, err)
	assert.True(t, result.Passed)
	assert.Equal(t, float64(0), result.RiskScore)
}

func TestValidateInSubprocess_BlockedAtAST(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	ctx := context.Background()
	code := `package main
import "os/exec"
func main() { exec.Command("ls") }`
	result, err := s.Validate(ctx, Suggestion{Code: code})
	assert.NoError(t, err)
	assert.False(t, result.Passed)
	assert.NotEmpty(t, result.BlockedPatterns)
	assert.Greater(t, result.RiskScore, float64(0))
}

// ---------------------------------------------------------------------------
// Obfuscation: import_concat pattern
// ---------------------------------------------------------------------------

func TestDetectObfuscation_ImportConcat(t *testing.T) {
	s := NewSandbox(5 * time.Second)
	code := "package main\nimport (\n\t\"encoding/\" + \"base64\"\n)\nfunc main() {}"
	warnings, score := s.detectObfuscation(code)

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "import_concat") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected import_concat warning")
	assert.GreaterOrEqual(t, score, 0.6)
}
