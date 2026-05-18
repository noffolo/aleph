package sandbox

import (
	"testing"
)

func TestValidateGoCode_BlockedImports(t *testing.T) {
	testCases := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name: "Allowed import",
			code: `package main
import "fmt"
func main() { fmt.Println("hello") }`,
			wantErr: false,
		},
		{
			name: "Blocked os/exec",
			code: `package main
import "os/exec"
func main() { exec.Command("ls") }`,
			wantErr: true,
		},
		{
			name: "Blocked net",
			code: `package main
import "net"
func main() { net.Dial("tcp", "example.com:80") }`,
			wantErr: true,
		},
		{
			name: "Blocked net/http",
			code: `package main
import "net/http"
func main() { http.Get("http://example.com") }`,
			wantErr: true,
		},
		{
			name: "Blocked syscall",
			code: `package main
import "syscall"
func main() { syscall.Getpid() }`,
			wantErr: true,
		},
		{
			name: "Blocked unsafe",
			code: `package main
import "unsafe"
func main() { var x int; unsafe.Pointer(&x) }`,
			wantErr: true,
		},
		{
			name: "Blocked reflect",
			code: `package main
import "reflect"
func main() { reflect.TypeOf(42) }`,
			wantErr: true,
		},
		{
			name: "Multiple imports with one blocked",
			code: `package main
import (
	"fmt"
	"os/exec"
	"strings"
)
func main() { exec.Command("ls") }`,
			wantErr: true,
		},
		{
			name: "Import with alias still blocked",
			code: `package main
import exec "os/exec"
func main() { exec.Command("ls") }`,
			wantErr: true,
		},
		{
			name: "Net subpackage blocked",
			code: `package main
import "net/url"
func main() { url.Parse("http://example.com") }`,
			wantErr: true,
		},
		{
			name: "No imports",
			code: `package main
func main() { println("hello") }`,
			wantErr: false,
		},
		{
			name: "Malformed code still catches blocked imports",
			code: `package main
import "os/exec"
func main() { // missing closing brace`,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateGoCode(tc.code)
			if tc.wantErr && err == nil {
				t.Errorf("ValidateGoCode() expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("ValidateGoCode() unexpected error: %v", err)
			}
		})
	}
}

func TestValidatePythonCode_BlockedPatterns(t *testing.T) {
	testCases := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name: "Allowed Python code",
			code: `import json
def hello():
    print("world")`,
			wantErr: false,
		},
		{
			name: "Blocked subprocess import",
			code: `import subprocess
subprocess.run(["ls"])`,
			wantErr: true,
		},
		{
			name: "Blocked socket import",
			code: `import socket
s = socket.socket()`,
			wantErr: true,
		},
		{
			name: "Blocked ctypes import",
			code: `import ctypes
lib = ctypes.CDLL("libc.so.6")`,
			wantErr: true,
		},
		{
			name: "Blocked subprocess.call",
			code: `import os
subprocess.call(["ls"])`,
			wantErr: true,
		},
		{
			name: "Blocked os.system",
			code: `import os
os.system("ls")`,
			wantErr: true,
		},
		{
			name:    "Blocked eval",
			code:    `x = eval("2+2")`,
			wantErr: true,
		},
		{
			name:    "Blocked exec",
			code:    `exec("print('hello')")`,
			wantErr: true,
		},
		{
			name:    "Blocked __import__",
			code:    `module = __import__("subprocess")`,
			wantErr: true,
		},
		{
			name:    "Blocked open with http URL",
			code:    `f = open("http://example.com", "r")`,
			wantErr: true,
		},
		{
			name:    "Blocked open with https URL",
			code:    `f = open("https://example.com", "r")`,
			wantErr: true,
		},
		{
			name:    "Blocked open with ftp URL",
			code:    `f = open("ftp://example.com/file.txt", "r")`,
			wantErr: true,
		},
		{
			name:    "Allowed open with local file",
			code:    `f = open("/tmp/test.txt", "w")`,
			wantErr: false,
		},
		{
			name: "From import still blocked",
			code: `from subprocess import run
run(["ls"])`,
			wantErr: true,
		},
		{
			name: "Comment with dangerous word",
			code: `# This uses subprocess somewhere
print("hello")`,
			wantErr: false,
		},
		{
			name:    "Disguised import with spaces",
			code:    `import  subprocess  # extra spaces`,
			wantErr: true,
		},
		{
			name: "Multiple lines with one blocked",
			code: `import json
import socket
import os`,
			wantErr: true,
		},
		{
			name: "Blocked imaplib import",
			code: `import imaplib
mail = imaplib.IMAP4_SSL("imap.example.com")`,
			wantErr: true,
		},
		{
			name: "Blocked from imaplib import",
			code: `from imaplib import IMAP4_SSL
mail = IMAP4_SSL("imap.example.com")`,
			wantErr: true,
		},
		{
			name: "Multi-line eval bypass via backslash",
			code: `import json
result = ev\
al("2+2")
print(result)`,
			wantErr: true,
		},
		{
			name: "Multi-line exec bypass via backslash",
			code: `ex\
ec("import os; os.system('ls')")
`,
			wantErr: true,
		},
		{
			name: "Multi-line __import__ bypass via backslash",
			code: `mod = __imp\
ort__("subprocess")`,
			wantErr: true,
		},
		{
			name: "Normal line continuation (allowed)",
			code: `result = "hel\
lo"
print(result)`,
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePythonCode(tc.code)
			if tc.wantErr && err == nil {
				t.Errorf("ValidatePythonCode() expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("ValidatePythonCode() unexpected error: %v", err)
			}
		})
	}
}

func TestIsPythonCode(t *testing.T) {
	testCases := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "Python shebang",
			code:     "#!/usr/bin/env python\nprint('hello')",
			expected: true,
		},
		{
			name:     "Python comment header",
			code:     "# python\nprint('hello')",
			expected: true,
		},
		{
			name:     "Python def function",
			code:     "def hello():\n    return 'world'",
			expected: true,
		},
		{
			name:     "Python import",
			code:     "import sys\nsys.exit(0)",
			expected: true,
		},
		{
			name:     "Python print",
			code:     "print('hello world')",
			expected: true,
		},
		{
			name:     "Go code",
			code:     "package main\nfunc main() { println('hello') }",
			expected: false,
		},
		{
			name:     "Empty string",
			code:     "",
			expected: false,
		},
		{
			name:     "Mixed content",
			code:     "def foo():\n    println('bar')",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsPythonCode(tc.code)
			if got != tc.expected {
				t.Errorf("IsPythonCode() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestAnalyzeGoCodeQuality_Basic(t *testing.T) {
	source := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`
	metrics := AnalyzeGoCodeQuality(source)
	if metrics.LinesOfCode == 0 {
		t.Error("expected non-zero lines of code")
	}
	if metrics.CyclomaticComplexity < 1 {
		t.Error("expected at least base complexity of 1")
	}
}

func TestAnalyzeGoCodeQuality_Complexity(t *testing.T) {
	source := `package main
func f() {
	if true {
		if false {
		}
	}
	for i := 0; i < 10; i++ {
	}
}
`
	metrics := AnalyzeGoCodeQuality(source)
	if metrics.CyclomaticComplexity < 3 {
		t.Errorf("expected complexity >= 3, got %d", metrics.CyclomaticComplexity)
	}
}

func TestEstimateComplexity(t *testing.T) {
	tests := []struct {
		name          string
		source        string
		minComplexity int
	}{
		{"empty function", `package main
func main() {}`, 1},
		{"one if", `package main
func f() { if true {} }`, 2},
		{"if-else", `package main
func f() { if true {} else {} }`, 3},
		{"for loop", `package main
func f() { for i:=0;;i++ {} }`, 2},
		{"switch", `package main
func f() { switch x { case 1: } }`, 3},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := EstimateComplexity(tc.source)
			if got < tc.minComplexity {
				t.Errorf("EstimateComplexity() = %d, want >= %d", got, tc.minComplexity)
			}
		})
	}
}

func TestCheckGoFormat_Valid(t *testing.T) {
	issues := CheckGoFormat("package main\nfunc main() {}\n")
	if len(issues) != 0 {
		t.Errorf("expected no format issues, got %v", issues)
	}
}

func TestCheckGoFormat_InvalidSyntax(t *testing.T) {
	issues := CheckGoFormat("package main\nfunc main() {")
	if len(issues) == 0 {
		t.Error("expected format issues for invalid syntax")
	}
}

func TestCheckGoFormat_TrailingWhitespace(t *testing.T) {
	issues := CheckGoFormat("package main \nfunc main() {}\n")
	if len(issues) == 0 {
		t.Error("expected trailing whitespace detection")
	}
}

func TestCountNonBlankLines(t *testing.T) {
	source := "line1\n\nline3\n  \nline5"
	count := countNonBlankLines(source)
	if count != 3 {
		t.Errorf("expected 3 non-blank lines, got %d", count)
	}
}

func TestStripComments(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"single line", "a // comment\nb", "a \nb"},
		{"block removal", "a /* block */ b", "a  b"},
		{"multi-line block", "a /* start\nend */ b", "a \nb"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := stripComments(tc.input)
			if got != tc.want {
				t.Errorf("stripComments(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestEstimateCoverage_NoTests(t *testing.T) {
	cov := estimateCoverage("package main\nfunc main() {}\n")
	if cov != 0.0 {
		t.Errorf("expected 0 coverage for no tests, got %f", cov)
	}
}

func TestEstimateCoverage_WithTests(t *testing.T) {
	cov := estimateCoverage(`package main
import "testing"
func TestX(t *testing.T) {
	t.Error("fail")
	t.Fatal("fail")
}
`)
	if cov <= 0 {
		t.Errorf("expected positive coverage estimate, got %f", cov)
	}
}

// FuzzValidateGoCode verifies that ValidateGoCode never panics on arbitrary
// input, even malformed Go source or binary data.
func FuzzValidateGoCode(f *testing.F) {
	seeds := []string{
		`package main
import "fmt"
func main() { fmt.Println("hello") }`,
		`package main
import "os/exec"
func main() { exec.Command("ls") }`,
		`package main
import "os/exec"`,
		`package main
import (
	"fmt"
	"os/exec"
	"strings"
)`,
		`package main
import exec "os/exec"`,
		``,
		`this is not go code at all ; drop table users`,
		`import "syscall"`,
		`import "unsafe"`,
		`package main`,
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, source string) {
		// Must never panic
		_ = ValidateGoCode(source)
	})
}

// FuzzValidatePythonCode verifies that ValidatePythonCode never panics on
// arbitrary input, including malformed Python or binary data.
func FuzzValidatePythonCode(f *testing.F) {
	seeds := []string{
		`import json
def hello():
    print("world")`,
		`import subprocess
subprocess.run(["ls"])`,
		`import socket`,
		`x = eval("2+2")`,
		`exec("print('hello')")`,
		`module = __import__("subprocess")`,
		`import imaplib`,
		`from subprocess import run`,
		`f = open("http://example.com")`,
		`ex\
ec("import os; os.system('ls')")`,
		``,
		`not python at all ; drop table users`,
		`# comment`,
		`print("hello")`,
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, source string) {
		// Must never panic
		_ = ValidatePythonCode(source)
	})
}

// TestValidateGoCode_PropertyBased generates random Go-like sources and
// verifies that ValidateGoCode never panics, even on nonsensical input.
func TestValidateGoCode_PropertyBased(t *testing.T) {
	prefixes := []string{
		`package main`,
		`package main
import "fmt"`,
		`package main
func main() {}`,
		`package main
import (
	"fmt"
	"strings"
	"time"
)`,
	}
	bodies := []string{
		"",
		`func main() { println("hello") }`,
		`func main() {}`,
		`func f() int { return 42 }`,
	}
	suffixes := []string{
		"",
		"; DROP TABLE",
		"<script>alert(1)</script>",
		"\x00\x01\x02",
		"\n\n\n",
	}

	for _, prefix := range prefixes {
		for _, body := range bodies {
			for _, suffix := range suffixes {
				source := prefix + "\n" + body + "\n" + suffix
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Errorf("ValidateGoCode panicked on %q: %v", source, r)
						}
					}()
					_ = ValidateGoCode(source)
				}()
			}
		}
	}
}
