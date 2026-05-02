package sandbox

import (
	"fmt"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
	"sync"
)

var (
	blocklistedGoImports = map[string]struct{}{
		"os":             {},
		"os/exec":        {},
		"net":           {},
		"net/http":      {},
		"net/url":       {},
		"net/smtp":      {},
		"net/rpc":       {},
		"net/mail":      {},
		"syscall":       {},
		"unsafe":        {},
		"reflect":       {},
		"plugin":        {},
		"runtime":       {},
		"runtime/cgo":   {},
		"runtime/pprof":{},
		"crypto":              {},
		"crypto/md5":          {},
		"crypto/sha1":         {},
		"crypto/sha256":       {},
		"crypto/sha512":       {},
		"crypto/aes":          {},
		"crypto/cipher":       {},
		"crypto/des":          {},
		"crypto/ecdsa":        {},
		"crypto/ed25519":      {},
		"crypto/elliptic":     {},
		"crypto/hmac":         {},
		"crypto/rand":         {},
		"crypto/rsa":          {},
		"crypto/subtle":       {},
		"crypto/tls":          {},
		"crypto/x509":         {},
		"encoding":            {},
		"encoding/hex":        {},
		"encoding/base64":     {},
		"encoding/json":       {},
		"encoding/gob":        {},
		"encoding/pem":        {},
		"encoding/asn1":       {},
		"encoding/binary":     {},
		"io/ioutil":           {},
		"mime/multipart":      {},
		"text/template":       {},
		"html/template":       {},
		"debug/dwarf":         {},
		"debug/elf":           {},
		"debug/gosym":         {},
		"debug/macho":         {},
		"debug/pe":            {},
		"debug/plan9obj":      {},
	}

	blocklistedPythonPatterns = []string{
		`^\s*import\s+(subprocess|socket|ctypes|__import__|imaplib)`,
		`^\s*from\s+(subprocess|socket|ctypes|imaplib)\s+import`,
		`subprocess\.(run|call|Popen|check_output)`,
		`socket\.(socket|create_connection|gethostbyname)`,
		`os\.system`,
		`os\.popen`,
		`eval\s*\(`,
		`exec\s*\(`,
		`__import__\s*\(`,
		`open\s*\([^)]*['\"]http[s]?:`,
		`open\s*\([^)]*['\"]ftp://`,
		`^\s*import\s+(importlib|runpy|pickle|shelve|shutil|code)`,
		`^\s*from\s+(importlib|runpy|pickle|shelve|shutil|code)\s+import`,
		`^\s*import\s+(requests|httpx|urllib3|aiohttp|websockets)`,
		`^\s*from\s+(requests|httpx|urllib3|aiohttp|websockets)\s+import`,
		`^\s*import\s+smtplib`,
		`^\s*from\s+smtplib\s+import`,
		`getattr\s*\(`,
		`__getattribute__`,
		`__dict__`,
		`__class__`,
		`globals\s*\(`,
		`locals\s*\(`,
		`vars\s*\(`,
		`__builtins__`,
		`compile\s*\(`,
	}

	pythonPatternRegexes []*regexp.Regexp
	compileOnce          sync.Once
	compileErr           error
)

// ValidateConfig compiles the Python blocklist regexes lazily.
func ValidateConfig() error {
	compileOnce.Do(func() {
		pythonPatternRegexes = make([]*regexp.Regexp, 0, len(blocklistedPythonPatterns))
		for _, pattern := range blocklistedPythonPatterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				compileErr = fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
				return
			}
			pythonPatternRegexes = append(pythonPatternRegexes, re)
		}
	})
	return compileErr
}

func ValidateGoCode(source string) error {
	if strings.Contains(source, "import") {
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, "", source, parser.ImportsOnly)
		if err != nil {
			return simpleGoImportCheck(source)
		}

		for _, imp := range node.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if _, blocked := blocklistedGoImports[importPath]; blocked {
				return fmt.Errorf("blocklisted import %q", importPath)
			}
			if strings.HasPrefix(importPath, "net/") {
				return fmt.Errorf("blocklisted net subpackage %q", importPath)
			}
			if strings.HasPrefix(importPath, "internal/") {
				return fmt.Errorf("blocklisted internal subpackage %q", importPath)
			}
		}
	}
	return nil
}

func simpleGoImportCheck(source string) error {
	lines := strings.Split(source, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "import") {
			continue
		}
		for blocked := range blocklistedGoImports {
			if strings.Contains(line, `"`+blocked+`"`) {
				return fmt.Errorf("blocklisted import %q detected", blocked)
			}
		}
		if strings.Contains(line, `"net/`) {
			return fmt.Errorf("blocklisted net subpackage detected")
		}
		if strings.Contains(line, `"internal/`) {
			return fmt.Errorf("blocklisted internal subpackage detected")
		}
	}
	return nil
}

// collapseBackslashContinuations joins lines ending with a backslash
// to prevent multi-line evasion of blocklisted patterns.
// Example: "ev\\\nal(" becomes "eval(".
func collapseBackslashContinuations(source string) string {
	lines := strings.Split(source, "\n")
	var out []string
	var buf strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t\r")
		if strings.HasSuffix(trimmed, "\\") {
			buf.WriteString(trimmed[:len(trimmed)-1])
			continue
		}
		if buf.Len() > 0 {
			buf.WriteString(line)
			out = append(out, buf.String())
			buf.Reset()
			continue
		}
		out = append(out, line)
	}
	if buf.Len() > 0 {
		out = append(out, buf.String())
	}
	return strings.Join(out, "\n")
}

func ValidatePythonCode(source string) error {
	if err := ValidateConfig(); err != nil {
		return fmt.Errorf("sandbox config: %w", err)
	}
	// Collapse backslash-continuation lines to prevent multi-line evasion
	// of blocklisted patterns (e.g. "ev\\\nal(" splitting across lines).
	source = collapseBackslashContinuations(source)
	lines := strings.Split(source, "\n")
	for i, line := range lines {
		lineTrimmed := strings.TrimSpace(line)
		if strings.HasPrefix(lineTrimmed, "#") {
			continue
		}
		
		for _, re := range pythonPatternRegexes {
			if re.MatchString(line) {
				return fmt.Errorf("line %d: blocklisted pattern %q", i+1, re.String())
			}
		}
		
		if strings.Contains(line, "import") {
			if strings.Contains(line, "subprocess") || strings.Contains(line, "socket") ||
				strings.Contains(line, "ctypes") || strings.Contains(line, "__import__") ||
				strings.Contains(line, "imaplib") || strings.Contains(line, "importlib") ||
				strings.Contains(line, "runpy") || strings.Contains(line, "pickle") ||
				strings.Contains(line, "shelve") || strings.Contains(line, "shutil") ||
				strings.Contains(line, "code") || strings.Contains(line, "requests") ||
				strings.Contains(line, "httpx") || strings.Contains(line, "urllib3") ||
				strings.Contains(line, "aiohttp") || strings.Contains(line, "websockets") ||
				strings.Contains(line, "smtplib") {
				return fmt.Errorf("line %d: blocklisted import detected", i+1)
			}
		}
	}
	return nil
}

func IsPythonCode(code string) bool {
	return strings.HasPrefix(code, "# python") || 
	       strings.HasPrefix(code, "#!/usr/bin/env python") ||
	       strings.Contains(code, "def ") || 
	       strings.Contains(code, "import ") ||
	       strings.Contains(code, "print(")
}

// CodeMetrics holds quality metrics extracted from tool source code.
type CodeMetrics struct {
	LinesOfCode        int
	CyclomaticComplexity int
	GofmtErrors        []string
	HasGofmtViolations bool
	EstimatedCoverage  float64
}

// AnalyzeGoCodeQuality runs quality checks on Go source code.
// It estimates cyclomatic complexity, checks gofmt compliance, and
// returns a comprehensive CodeMetrics result.
func AnalyzeGoCodeQuality(source string) CodeMetrics {
	metrics := CodeMetrics{
		LinesOfCode: countNonBlankLines(source),
	}

	// Cyclomatic complexity estimate: count decision points
	metrics.CyclomaticComplexity = EstimateComplexity(source)

	// gofmt compliance check
	metrics.GofmtErrors = CheckGoFormat(source)
	metrics.HasGofmtViolations = len(metrics.GofmtErrors) > 0

	// Estimated coverage: simple heuristic based on test-like patterns
	metrics.EstimatedCoverage = estimateCoverage(source)

	return metrics
}

// EstimateComplexity estimates cyclomatic complexity by counting decision
// points: if, else if, for, switch, select, case (non-type), range, &&, ||.
// This is a simplified estimate; production tools use go/ast walkers.
func EstimateComplexity(source string) int {
	complexity := 1 // base complexity
	noComments := stripComments(source)
	tokens := strings.Fields(noComments)

	for _, tok := range tokens {
		switch tok {
		case "if", "else", "for", "range", "select":
			complexity++
		case "switch":
			complexity++
		case "case":
			complexity++
		case "&&", "||":
			complexity++
		}
	}
	return complexity
}

// CheckGoFormat checks whether Go source complies with gofmt.
// It validates parsing and common formatting issues.
func CheckGoFormat(source string) []string {
	var issues []string

	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "", source, parser.AllErrors)
	if err != nil {
		issues = append(issues, fmt.Sprintf("parse error: %v", err))
		return issues
	}

	// Check for common formatting violations
	lines := strings.Split(source, "\n")
	for i, line := range lines {
		if strings.HasSuffix(line, " ") || strings.HasSuffix(line, "\t") {
			issues = append(issues, fmt.Sprintf("line %d: trailing whitespace", i+1))
		}
		if strings.Contains(line, "\t") {
			// gofmt uses tabs for indentation; this is fine.
			// Only flag mixed tabs+spaces at line start.
			trimmed := strings.TrimLeft(line, " \t")
			if len(trimmed) < len(line) {
				lead := line[:len(line)-len(trimmed)]
				if strings.Contains(lead, " ") && strings.Contains(lead, "\t") {
					issues = append(issues, fmt.Sprintf("line %d: mixed tabs and spaces in indentation", i+1))
				}
			}
		}
	}

	return issues
}

// countNonBlankLines returns the number of non-blank lines in source.
func countNonBlankLines(source string) int {
	count := 0
	for _, line := range strings.Split(source, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

// stripComments removes single-line (//) and block (/* */) comments from source.
func stripComments(source string) string {
	var result []string
	lines := strings.Split(source, "\n")
	inBlock := false

	for _, line := range lines {
		if inBlock {
			if idx := strings.Index(line, "*/"); idx >= 0 {
				result = append(result, strings.TrimSpace(line[idx+2:]))
				inBlock = false
			}
			continue
		}
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		if idx := strings.Index(line, "/*"); idx >= 0 {
			before := line[:idx]
			if end := strings.Index(line[idx+2:], "*/"); end >= 0 {
				after := line[idx+2+end+2:]
				result = append(result, before+after)
			} else {
				result = append(result, before)
				inBlock = true
			}
			continue
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}

// estimateCoverage provides a simple coverage heuristic based on whether
// test-like functions and assertions exist in the same file.
func estimateCoverage(source string) float64 {
	noComments := stripComments(source)
	if !strings.Contains(noComments, "func Test") && !strings.Contains(noComments, "func Test") {
		return 0.0
	}

	assertCount := strings.Count(noComments, "t.Error") +
		strings.Count(noComments, "t.Fatal") +
		strings.Count(noComments, "assert.")
	if assertCount == 0 {
		return 10.0
	}

	// Heuristic: each assertion ≅ 5% coverage, capped at 85%
	est := float64(assertCount) * 5.0
	if est > 85.0 {
		return 85.0
	}
	return est
}