package sandbox

import (
	"regexp"
	"testing"
)

func TestNewSecurityScanner(t *testing.T) {
	s := NewSecurityScanner()
	if s == nil {
		t.Fatal("NewSecurityScanner returned nil")
	}
	rules := s.Rules()
	if len(rules) == 0 {
		t.Error("expected at least one default security rule")
	}
}

func TestSecurityScanner_Scan_XSS(t *testing.T) {
	s := NewSecurityScanner()

	code := `func render() string {
	return "<script>alert('xss')</script>"
}`
	issues := s.Scan(code)
	if len(issues) == 0 {
		t.Error("expected XSS script tag to be detected")
	}
	found := false
	for _, issue := range issues {
		if issue.RuleName == "xss-script-tag" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected xss-script-tag rule to match")
	}
}

func TestSecurityScanner_Scan_XSSEventHandler(t *testing.T) {
	s := NewSecurityScanner()

	code := `return "<div onclick='evil()'>click</div>"`
	issues := s.Scan(code)
	if len(issues) == 0 {
		t.Error("expected XSS event handler to be detected")
	}
}

func TestSecurityScanner_Scan_SQLInjection(t *testing.T) {
	s := NewSecurityScanner()

	code := `query := "SELECT * FROM users WHERE id = " + userInput`
	issues := s.Scan(code)
	if len(issues) == 0 {
		t.Error("expected SQL injection concat to be detected")
	}
}

func TestSecurityScanner_Scan_SQLInjectionFmt(t *testing.T) {
	s := NewSecurityScanner()

	code := `query := fmt.Sprintf("SELECT * FROM users WHERE id = %s", userInput)`
	issues := s.Scan(code)
	if len(issues) == 0 {
		t.Error("expected SQL injection fmt to be detected")
	}
}

func TestSecurityScanner_Scan_PathTraversal(t *testing.T) {
	s := NewSecurityScanner()

	code := `data, err := os.ReadFile(dir + "/" + filename)`
	issues := s.Scan(code)
	if len(issues) == 0 {
		t.Error("expected path traversal to be detected")
	}
}

func TestSecurityScanner_Scan_HardcodedAPIKey(t *testing.T) {
	s := NewSecurityScanner()

	code := `api_key = "sk-1234567890abcdef"`
	issues := s.Scan(code)
	if len(issues) == 0 {
		t.Error("expected hardcoded API key to be detected")
	}
}

func TestSecurityScanner_Scan_HardcodedPrivateKey(t *testing.T) {
	s := NewSecurityScanner()

	code := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
-----END RSA PRIVATE KEY-----`
	issues := s.Scan(code)
	if len(issues) == 0 {
		t.Error("expected private key to be detected")
	}
}

func TestSecurityScanner_Scan_CommandInjection(t *testing.T) {
	s := NewSecurityScanner()

	code := `cmd := exec.Command("bash", "-c", "cat " + filename)`
	issues := s.Scan(code)
	if len(issues) == 0 {
		t.Error("expected command injection to be detected")
	}
}

func TestSecurityScanner_Scan_CommandInjectionPython(t *testing.T) {
	s := NewSecurityScanner()

	code := `subprocess.run("ls " + user_input, shell=True)`
	issues := s.Scan(code)
	if len(issues) == 0 {
		t.Error("expected command injection to be detected")
	}
}

func TestSecurityScanner_Scan_InsecureDeserialization(t *testing.T) {
	s := NewSecurityScanner()

	code := `data = pickle.loads(blob)`
	issues := s.Scan(code)
	if len(issues) == 0 {
		t.Error("expected insecure deserialization to be detected")
	}
}

func TestSecurityScanner_Scan_CleanCode(t *testing.T) {
	s := NewSecurityScanner()

	code := `package main
import "fmt"
func main() {
	fmt.Println("hello world")
}`
	issues := s.Scan(code)
	if len(issues) != 0 {
		t.Errorf("expected 0 issues for clean code, got %d: %v", len(issues), issues)
	}
}

func TestSecurityScanner_ScanCode(t *testing.T) {
	s := NewSecurityScanner()

	cleanCode := `package main
func main() {}`
	if err := s.ScanCode(cleanCode); err != nil {
		t.Errorf("ScanCode() on clean code: %v", err)
	}

	dangerCode := `func render() string {
	return "<script>alert('xss')</script>"
}`
	if err := s.ScanCode(dangerCode); err == nil {
		t.Error("ScanCode() should return error for dangerous code")
	}
}

func TestNewSecurityScannerWithRules(t *testing.T) {
	customRules := []SecurityRule{
		{
			Name:        "test-rule",
			Description: "test description",
			Pattern:     regexp.MustCompile(`test`),
			Severity:    "low",
			Languages:   []string{"go"},
		},
	}
	s := NewSecurityScannerWithRules(customRules)
	if s == nil {
		t.Fatal("NewSecurityScannerWithRules returned nil")
	}
	rules := s.Rules()
	if len(rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(rules))
	}
}
