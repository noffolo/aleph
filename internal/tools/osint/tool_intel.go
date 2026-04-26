package osint

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// RiskLevel represents the severity of a security risk.
type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

// IntelReport represents a security intelligence report for a tool.
type IntelReport struct {
	ToolName        string    `json:"tool_name"`
	RiskScore       float64   `json:"risk_score"` // 0-100
	RiskLevel       RiskLevel `json:"risk_level"`
	Warnings        []string  `json:"warnings"`
	Recommendations []string  `json:"recommendations"`
	Sources         []string  `json:"sources"`
	LastScan        time.Time `json:"last_scan"`
	ScannedImports  []string  `json:"scanned_imports,omitempty"`
	PatternsFound   []string  `json:"patterns_found,omitempty"`
}

// ToolIntel provides security intelligence analysis for tool code.
type ToolIntel struct {
	patterns []securityPattern
}

type securityPattern struct {
	name     string
	severity RiskLevel
	weight   float64
	search   func(code string) bool
}

// NewToolIntel creates a new ToolIntel instance with built-in security patterns.
func NewToolIntel() *ToolIntel {
	return &ToolIntel{
		patterns: []securityPattern{
			{name: "os/exec", severity: RiskHigh, weight: 30, search: hasImport("os/exec")},
			{name: "os/exec.Command", severity: RiskHigh, weight: 25, search: containsPattern("exec.Command")},
			{name: "eval/exec call", severity: RiskCritical, weight: 40, search: containsPattern("exec(")},
			{name: "subprocess", severity: RiskHigh, weight: 30, search: containsPattern("subprocess")},
			{name: "os.StartProcess", severity: RiskHigh, weight: 25, search: containsPattern("os.StartProcess")},
			{name: "syscall", severity: RiskHigh, weight: 25, search: hasImport("syscall")},
			{name: "unsafe", severity: RiskHigh, weight: 20, search: hasImport("unsafe")},
			{name: "crypto/rand", severity: RiskLow, weight: -5, search: hasImport("crypto/rand")},
			{name: "encoding/json (unsafe)", severity: RiskMedium, weight: 10, search: containsPattern("json.Unmarshal")},
			{name: "ioutil (deprecated)", severity: RiskLow, weight: 5, search: hasImport("io/ioutil")},
			{name: "net/http (basic)", severity: RiskLow, weight: 5, search: hasImport("net/http")},
			{name: "hardcoded credentials", severity: RiskCritical, weight: 35, search: containsPattern("password")},
			{name: "hardcoded API key", severity: RiskCritical, weight: 35, search: containsPattern("api_key")},
			{name: "hardcoded secret", severity: RiskCritical, weight: 35, search: containsPattern("secret")},
			{name: "SQL injection risk", severity: RiskHigh, weight: 30, search: containsPattern("fmt.Sprintf.*SELECT")},
			{name: "command injection", severity: RiskCritical, weight: 40, search: containsPattern("exec.Command.*fmt")},
			{name: "reflect", severity: RiskMedium, weight: 15, search: hasImport("reflect")},
			{name: "cgo", severity: RiskMedium, weight: 15, search: containsPattern("import \"C\"")},
			{name: "os.Remove", severity: RiskMedium, weight: 15, search: containsPattern("os.Remove")},
			{name: "ioutil.WriteFile", severity: RiskLow, weight: 5, search: containsPattern("ioutil.WriteFile")},
		},
	}
}

func hasImport(pkg string) func(code string) bool {
	return func(code string) bool {
		return strings.Contains(code, fmt.Sprintf("\"%s\"", pkg)) ||
			strings.Contains(code, fmt.Sprintf("'%s'", pkg))
	}
}

func containsPattern(pattern string) func(code string) bool {
	return func(code string) bool {
		return strings.Contains(code, pattern)
	}
}

func (ti *ToolIntel) extractImports(code string) []string {
	var imports []string
	lines := strings.Split(code, "\n")
	inImport := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "import (") {
			inImport = true
			continue
		}
		if inImport {
			if strings.HasPrefix(trimmed, ")") {
				break
			}
			cleaned := strings.Trim(trimmed, "\" \t")
			if cleaned != "" && !strings.HasPrefix(cleaned, "//") {
				imports = append(imports, cleaned)
			}
		}
		if strings.HasPrefix(trimmed, "import ") && !strings.HasPrefix(trimmed, "import (") {
			parts := strings.Fields(trimmed)
			for _, p := range parts {
				if strings.HasPrefix(p, "\"") && strings.HasSuffix(p, "\"") {
					imports = append(imports, strings.Trim(p, "\""))
				}
			}
		}
	}
	return imports
}

// ScanTool performs static analysis on tool source code and returns an IntelReport.
func (ti *ToolIntel) ScanTool(name, code string) (IntelReport, error) {
	if name == "" {
		return IntelReport{}, fmt.Errorf("tool name cannot be empty")
	}

	warnings := make([]string, 0)
	recommendations := make([]string, 0)
	patternsFound := make([]string, 0)
	imports := ti.extractImports(code)
	var totalRisk float64
	severeFound := false

	for _, pattern := range ti.patterns {
		if !pattern.search(code) {
			continue
		}

		patternsFound = append(patternsFound, pattern.name)
		totalRisk += pattern.weight

		switch pattern.severity {
		case RiskCritical:
			warnings = append(warnings, fmt.Sprintf("Critical: %s detected in tool code", pattern.name))
			recommendations = append(recommendations,
				fmt.Sprintf("Review and sandbox '%s' usage — high security risk", pattern.name))
			severeFound = true
		case RiskHigh:
			warnings = append(warnings, fmt.Sprintf("Warning: %s usage detected", pattern.name))
			recommendations = append(recommendations,
				fmt.Sprintf("Consider alternative to '%s' if possible", pattern.name))
			severeFound = true
		case RiskMedium:
			warnings = append(warnings, fmt.Sprintf("Note: %s usage — review required", pattern.name))
		case RiskLow:
			if pattern.weight > 0 {
				warnings = append(warnings, fmt.Sprintf("Info: %s detected", pattern.name))
			}
		}
	}

	// Negative weight reduces risk (e.g. crypto/rand is good)
	if totalRisk < 0 {
		totalRisk = 0
	}
	if totalRisk > 100 {
		totalRisk = 100
	}

	riskLevel := RiskLow
	switch {
	case totalRisk >= 70:
		riskLevel = RiskCritical
	case totalRisk >= 50:
		riskLevel = RiskHigh
	case totalRisk >= 25:
		riskLevel = RiskMedium
	}

	if totalRisk > 0 && !severeFound {
		recommendations = append(recommendations, "Perform additional security review")
	}

	sources := []string{"static_analysis"}
	if len(imports) > 0 {
		importNotice := fmt.Sprintf("Found %d imports for analysis", len(imports))
		sources = append(sources, importNotice)
	}

	return IntelReport{
		ToolName:        name,
		RiskScore:       totalRisk,
		RiskLevel:       riskLevel,
		Warnings:        deduplicate(warnings),
		Recommendations: deduplicate(recommendations),
		Sources:         sources,
		LastScan:        time.Now(),
		ScannedImports:  imports,
		PatternsFound:   deduplicate(patternsFound),
	}, nil
}

func deduplicate(items []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(items))
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

// ToolSecurityProfile represents a comprehensive security profile for a tool.
type ToolSecurityProfile struct {
	ToolName       string   `json:"tool_name"`
	RiskScore      float64  `json:"risk_score"` // 0-100
	RiskLevel      string   `json:"risk_level"`
	Warnings       []string `json:"warnings"`
	Recommendations []string `json:"recommendations"`
	Sources        []string `json:"sources"`
}

// DiscoverToolSecurity discovers the security profile for a given tool name.
// This method is added to Shadowbroker to provide tool security intelligence.
func (s *Shadowbroker) DiscoverToolSecurity(ctx context.Context, toolName string) (ToolSecurityProfile, error) {
	if toolName == "" {
		return ToolSecurityProfile{}, fmt.Errorf("toolName cannot be empty")
	}

	// Check cache first
	cacheKey := "tool_security:" + toolName
	if cached, ok := s.cache.Get(cacheKey); ok {
		if profile, ok := cached.(ToolSecurityProfile); ok {
			return profile, nil
		}
	}

	// Try to look up known vulnerabilities via the external API
	var warnings []string
	var recommendations []string
	var riskScore float64
	sources := []string{"shadowbroker_intel"}

	if s.config.BaseURL != "" {
		params := map[string]string{"tool": toolName}
		result, err := s.Request(ctx, "/tool/security", params)
		if err == nil {
			sources = append(sources, "external_intel_api")
			if risk, ok := result["risk_score"].(float64); ok {
				riskScore = risk
			}
			if warns, ok := result["warnings"].([]string); ok {
				warnings = append(warnings, warns...)
			}
			if recs, ok := result["recommendations"].([]string); ok {
				recommendations = append(recommendations, recs...)
			}
		}
	}

	// Add default recommendations
	if riskScore > 50 {
		warnings = append(warnings, fmt.Sprintf("Elevated risk score (%.0f/100) for tool: %s", riskScore, toolName))
		recommendations = append(recommendations, "Review tool before deployment")
		recommendations = append(recommendations, "Enable sandbox execution")
	}

	riskLevel := "low"
	switch {
	case riskScore >= 70:
		riskLevel = "critical"
	case riskScore >= 50:
		riskLevel = "high"
	case riskScore >= 25:
		riskLevel = "medium"
	}

	profile := ToolSecurityProfile{
		ToolName:       toolName,
		RiskScore:      riskScore,
		RiskLevel:      riskLevel,
		Warnings:       warnings,
		Recommendations: recommendations,
		Sources:        sources,
	}

	s.cache.Set(cacheKey, profile)
	return profile, nil
}
