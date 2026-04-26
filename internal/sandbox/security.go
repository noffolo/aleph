package sandbox

import (
	"fmt"
	"regexp"
	"strings"
)

// SecurityRule defines a single security check pattern for scanning tool code.
type SecurityRule struct {
	Name        string
	Description string
	Pattern     *regexp.Regexp
	Severity    string // "critical", "high", "medium", "low"
	Languages   []string // "go", "python", or both
}

// SecurityIssue represents a single security finding from a scan.
type SecurityIssue struct {
	RuleName    string
	Description string
	Severity    string
	Line        int
	Code        string
}

// SecurityScanner checks tool code for security vulnerabilities.
type SecurityScanner struct {
	rules []SecurityRule
}

// NewSecurityScanner creates a scanner with built-in security rules.
func NewSecurityScanner() *SecurityScanner {
	return &SecurityScanner{
		rules: defaultSecurityRules(),
	}
}

// NewSecurityScannerWithRules creates a scanner with custom security rules.
func NewSecurityScannerWithRules(rules []SecurityRule) *SecurityScanner {
	return &SecurityScanner{
		rules: rules,
	}
}

// Scan examines code for security issues and returns all findings.
func (s *SecurityScanner) Scan(code string) []SecurityIssue {
	var issues []SecurityIssue
	lines := strings.Split(code, "\n")

	for _, rule := range s.rules {
		for i, line := range lines {
			if rule.Pattern.MatchString(line) {
				issues = append(issues, SecurityIssue{
					RuleName:    rule.Name,
					Description: rule.Description,
					Severity:    rule.Severity,
					Line:        i + 1,
					Code:        strings.TrimSpace(line),
				})
			}
		}
	}

	return issues
}

// ScanCode is a convenience wrapper around Scan that returns a single
// formatted error if any issues are found, or nil if the code is clean.
func (s *SecurityScanner) ScanCode(code string) error {
	issues := s.Scan(code)
	if len(issues) == 0 {
		return nil
	}

	var msgs []string
	for _, issue := range issues {
		msgs = append(msgs, fmt.Sprintf(
			"[%s] line %d: %s (%s)",
			issue.Severity, issue.Line, issue.Description, issue.RuleName,
		))
	}
	return fmt.Errorf("security issues found:\n%s", strings.Join(msgs, "\n"))
}

// Rules returns a copy of the scanner's rule set.
func (s *SecurityScanner) Rules() []SecurityRule {
	r := make([]SecurityRule, len(s.rules))
	copy(r, s.rules)
	return r
}

// defaultSecurityRules returns the built-in security rule set.
func defaultSecurityRules() []SecurityRule {
	return []SecurityRule{
		{
			Name:        "xss-script-tag",
			Description: "direct script tag injection in rendered output",
			Pattern:     regexp.MustCompile(`<script\b[^>]*>.*?</script>`),
			Severity:    "high",
			Languages:   []string{"go", "python"},
		},
		{
			Name:        "xss-event-handler",
			Description: "inline event handler attribute in output",
			Pattern:     regexp.MustCompile(`\bon\w+\s*=`),
			Severity:    "high",
			Languages:   []string{"go", "python"},
		},
		{
			Name:        "sql-injection-concat",
			Description: "possible SQL injection: string concatenation in query context",
			Pattern:     regexp.MustCompile(`(?i)(select|insert|update|delete|drop|alter|truncate)\s+.*\+`),
			Severity:    "high",
			Languages:   []string{"go", "python"},
		},
		{
			Name:        "sql-injection-fmt",
			Description: "possible SQL injection via fmt.Sprintf with query",
			Pattern:     regexp.MustCompile(`(?i)fmt\.Sprintf.*\b(select|insert|update|delete|drop)\b`),
			Severity:    "high",
			Languages:   []string{"go"},
		},
		{
			Name:        "path-traversal",
			Description: "possible path traversal via string concatenation in file ops",
			Pattern:     regexp.MustCompile(`(os\.Open|os\.ReadFile|os\.Create|ioutil\.ReadFile|open\s*\()\s*[^)]*\+`),
			Severity:    "high",
			Languages:   []string{"go", "python"},
		},
		{
			Name:        "hardcoded-api-key",
			Description: "possible hardcoded API key or secret token",
			Pattern:     regexp.MustCompile(`(?i)(api[_-]?key|apikey|secret|token|password|passwd)\s*[=:]\s*['\"][^'\"]{8,}['\"]`),
			Severity:    "high",
			Languages:   []string{"go", "python"},
		},
		{
			Name:        "hardcoded-private-key",
			Description: "possible hardcoded private key block",
			Pattern:     regexp.MustCompile(`-----BEGIN\s+(RSA|DSA|EC|OPENSSH)\s+PRIVATE\s+KEY-----`),
			Severity:    "critical",
			Languages:   []string{"go", "python"},
		},
		{
			Name:        "command-injection",
			Description: "possible command injection via external call with concatenation",
			Pattern:     regexp.MustCompile(`(exec\.Command|subprocess\.(run|call|Popen|check_output)|os\.system)\s*\([^)]*\+`),
			Severity:    "critical",
			Languages:   []string{"go", "python"},
		},
		{
			Name:        "insecure-deserialization",
			Description: "unsafe deserialization via eval/pickle/yaml.load",
			Pattern:     regexp.MustCompile(`\b(eval|pickle\.loads|yaml\.load)\s*\(`),
			Severity:    "high",
			Languages:   []string{"python"},
		},
		{
			Name:        "template-injection",
			Description: "possible template injection via unescaped rendering helpers",
			Pattern:     regexp.MustCompile(`(?i)(template\.HTML|template\.JS|template\.URL|template\.HTMLEscapeString)\s*\(`),
			Severity:    "medium",
			Languages:   []string{"go", "python"},
		},
	}
}
