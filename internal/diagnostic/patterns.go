// Package diagnostic provides error pattern classification, root cause analysis,
// and severity assessment for the aleph-v2 system.
package diagnostic

import (
	"strings"
	"time"
)

type ErrorPattern struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Component   string `json:"component"`
	Count       int    `json:"count"`
	FirstSeen   string `json:"first_seen"`
	LastSeen    string `json:"last_seen"`
	RootCause   string `json:"root_cause"`
	Suggestion  string `json:"suggestion"`
}

const (
	PatternTimeout            = "timeout"
	PatternAuth               = "auth_failure"
	PatternDataIntegrity      = "data_integrity"
	PatternResourceExhaustion = "resource_exhaustion"
	PatternDependencyFailure  = "dependency_failure"
	PatternConfiguration      = "configuration"
)

const (
	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
	SeverityCritical = "critical"
)

func ClassifyError(code string, errMsg string) string {
	switch code {
	case "ERR_DEADLINE_EXCEEDED", "ERR_UNAVAILABLE":
		return PatternTimeout
	case "ERR_UNAUTHORIZED", "ERR_FORBIDDEN":
		return PatternAuth
	case "ERR_NOT_FOUND", "ERR_VALIDATION":
		return PatternDataIntegrity
	}

	msg := errMsg
	switch {
	case containsAny(msg, "timeout", "deadline", "context canceled", "TIMED OUT"):
		return PatternTimeout
	case containsAny(msg, "auth", "unauthorized", "forbidden", "permission", "token"):
		return PatternAuth
	case containsAny(msg, "checksum", "corrupt", "not found", "missing", "invalid data"):
		return PatternDataIntegrity
	case containsAny(msg, "OOM", "memory", "connection refused", "too many", "resource"):
		return PatternResourceExhaustion
	case containsAny(msg, "NLP", "grpc", "upstream", "dependency", "external"):
		return PatternDependencyFailure
	case containsAny(msg, "config", "env", "invalid argument", "misconfig"):
		return PatternConfiguration
	default:
		return PatternTimeout
	}
}

func AssessSeverity(patternType string, count int, impact string) string {
	highImpact := containsAny(impact, "user", "data", "production", "critical")

	switch patternType {
	case PatternAuth:
		if count >= 5 {
			return SeverityCritical
		}
		if count >= 3 {
			return SeverityHigh
		}
	case PatternDataIntegrity:
		if count >= 3 {
			return SeverityCritical
		}
	case PatternResourceExhaustion:
		if count >= 3 {
			return SeverityHigh
		}
	case PatternDependencyFailure:
		if count >= 5 {
			return SeverityHigh
		}
	case PatternTimeout:
		if count >= 10 {
			return SeverityHigh
		}
	}

	if highImpact {
		if count >= 2 {
			return SeverityHigh
		}
		return SeverityMedium
	}

	if count >= 5 {
		return SeverityMedium
	}
	return SeverityLow
}

// RootCauseAnalysis suggests a root cause based on pattern type and context.
func RootCauseAnalysis(pattern ErrorPattern) string {
	switch pattern.Type {
	case PatternTimeout:
		if pattern.Component == "nlp" {
			return "NLP service not responding within timeout. Possible overload or network issue."
		}
		return "Operation timed out. Possible system overload or increased network latency."
	case PatternAuth:
		return "Repeated authentication failures. Possible expired token, invalid credentials, or unauthorized access attempt."
	case PatternDataIntegrity:
		if pattern.Component == "ingestion" {
			return "Corrupted or incomplete data during ingestion. Possible checksum issue or invalid data format."
		}
		return "Data integrity compromised. Possible data corruption or missing records."
	case PatternResourceExhaustion:
		return "System resources exhausted. Possible memory leak, too many open connections, or CPU overload."
	case PatternDependencyFailure:
		return "External service unavailable. Check the status of dependent services (NLP, storage, MCP)."
	case PatternConfiguration:
		return "Configuration error or missing. Check environment variables and configuration parameters."
	default:
		return "Root cause undetermined. Manual analysis recommended."
	}
}

// SuggestFix provides actionable suggestions based on pattern type.
func SuggestFix(pattern ErrorPattern) string {
	switch pattern.Type {
	case PatternTimeout:
		return "Increase timeouts, check network connection, reduce service load."
	case PatternAuth:
		return "Check credentials, renew tokens, verify user permissions."
	case PatternDataIntegrity:
		return "Run checksum verification, restore from backup, check input data formats."
	case PatternResourceExhaustion:
		return "Increase resources (memory/CPU), implement rate limiting, close unused connections."
	case PatternDependencyFailure:
		return "Check external service status, implement circuit breaker, add retry with backoff."
	case PatternConfiguration:
		return "Check environment variables, verify configuration files, validate parameters at startup."
	default:
		return "Manual analysis recommended."
	}
}

type DiagnosticMonitor struct {
	patterns   map[string]*ErrorPattern
	alertCount int
	history    *HealthIntegration
}

type HealthIntegration struct {
	GetConsecutiveFailures func(toolID string) int
	GetToolHealthStatus    func(toolID string) string
}

func NewDiagnosticMonitor(alertCount int, healthIntegration *HealthIntegration) *DiagnosticMonitor {
	if alertCount <= 0 {
		alertCount = 3
	}
	return &DiagnosticMonitor{
		patterns:   make(map[string]*ErrorPattern),
		alertCount: alertCount,
		history:    healthIntegration,
	}
}

func (dm *DiagnosticMonitor) RecordError(code string, errMsg string, component string, impact string) ErrorPattern {
	patternType := ClassifyError(code, errMsg)
	key := patternType + ":" + component
	now := time.Now().Format(time.RFC3339)

	if existing, ok := dm.patterns[key]; ok {
		existing.Count++
		existing.LastSeen = now
		existing.Severity = AssessSeverity(patternType, existing.Count, impact)
		existing.RootCause = RootCauseAnalysis(*existing)
		existing.Suggestion = SuggestFix(*existing)
		return *existing
	}

	pattern := ErrorPattern{
		ID:          key,
		Type:        patternType,
		Description: describePattern(patternType, component, errMsg),
		Severity:    AssessSeverity(patternType, 1, impact),
		Component:   component,
		Count:       1,
		FirstSeen:   now,
		LastSeen:    now,
		RootCause:   "",
		Suggestion:  "",
	}
	pattern.RootCause = RootCauseAnalysis(pattern)
	pattern.Suggestion = SuggestFix(pattern)
	dm.patterns[key] = &pattern
	return pattern
}

func (dm *DiagnosticMonitor) GetPatterns() []ErrorPattern {
	result := make([]ErrorPattern, 0, len(dm.patterns))
	for _, p := range dm.patterns {
		result = append(result, *p)
	}
	return result
}

// SubsystemSummary aggregates error statistics for a single subsystem.
type SubsystemSummary struct {
	Subsystem    string `json:"subsystem"`
	TotalErrors  int    `json:"total_errors"`
	Patterns     int    `json:"patterns"`
	HighestSeverity string `json:"highest_severity"`
}

// CorrelateWithSubsystem groups recorded patterns by their Component field
// (which represents the subsystem) and returns per-subsystem summaries.
// This allows the DiagnosticMonitor to identify which subsystem is generating
// the most errors and at what severity level.
func (dm *DiagnosticMonitor) CorrelateWithSubsystem() []SubsystemSummary {
	bySubsystem := make(map[string]*SubsystemSummary)
	for _, p := range dm.patterns {
		comp := p.Component
		if comp == "" {
			comp = "unknown"
		}
		ss, ok := bySubsystem[comp]
		if !ok {
			ss = &SubsystemSummary{
				Subsystem:    comp,
				HighestSeverity: SeverityLow,
			}
			bySubsystem[comp] = ss
		}
		ss.TotalErrors += p.Count
		ss.Patterns++
		if severityRank(ss.HighestSeverity) < severityRank(p.Severity) {
			ss.HighestSeverity = p.Severity
		}
	}
	result := make([]SubsystemSummary, 0, len(bySubsystem))
	for _, ss := range bySubsystem {
		result = append(result, *ss)
	}
	return result
}

func severityRank(s string) int {
	switch s {
	case SeverityCritical:
		return 4
	case SeverityHigh:
		return 3
	case SeverityMedium:
		return 2
	case SeverityLow:
		return 1
	default:
		return 0
	}
}

func (dm *DiagnosticMonitor) GetCriticalPatterns() []ErrorPattern {
	result := make([]ErrorPattern, 0)
	for _, p := range dm.patterns {
		if p.Severity == SeverityHigh || p.Severity == SeverityCritical {
			result = append(result, *p)
		}
	}
	return result
}

func (dm *DiagnosticMonitor) ShouldAlert(pattern ErrorPattern) bool {
	return pattern.Count >= dm.alertCount && (pattern.Severity == SeverityHigh || pattern.Severity == SeverityCritical)
}

func (dm *DiagnosticMonitor) CorrelateWithHealth(pattern ErrorPattern) bool {
	if dm.history == nil {
		return false
	}
	if pattern.Component == "" {
		return false
	}
	consecutive := dm.history.GetConsecutiveFailures(pattern.Component)
	status := dm.history.GetToolHealthStatus(pattern.Component)
	return consecutive >= dm.alertCount || status == "down" || status == "degraded"
}

func describePattern(patternType string, component string, errMsg string) string {
	switch patternType {
	case PatternTimeout:
		return "Timeout: " + component + " - " + truncate(errMsg, 80)
	case PatternAuth:
		return "Authentication failed: " + component
	case PatternDataIntegrity:
		return "Data integrity: " + component + " - " + truncate(errMsg, 80)
	case PatternResourceExhaustion:
		return "Resources exhausted: " + component
	case PatternDependencyFailure:
		return "Dependent service unavailable: " + component
	case PatternConfiguration:
		return "Configuration error: " + component
	default:
		return "Error: " + component + " - " + truncate(errMsg, 80)
	}
}

func containsAny(s string, substrs ...string) bool {
	lower := strings.ToLower(s)
	for _, sub := range substrs {
		if strings.Contains(lower, strings.ToLower(sub)) {
			return true
		}
	}
	return false
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}