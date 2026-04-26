package diagnostic

import "testing"

func TestClassifyError_Timeout(t *testing.T) {
	tests := []struct {
		code    string
		msg     string
		want    string
	}{
		{"ERR_DEADLINE_EXCEEDED", "operation timed out", PatternTimeout},
		{"ERR_UNAVAILABLE", "service unavailable", PatternTimeout},
		{"ERR_UNAUTHORIZED", "invalid token", PatternAuth},
		{"ERR_FORBIDDEN", "access denied", PatternAuth},
		{"ERR_NOT_FOUND", "resource missing", PatternDataIntegrity},
		{"ERR_VALIDATION", "invalid data", PatternDataIntegrity},
		{"", "context canceled during processing", PatternTimeout},
		{"", "authentication failed for user", PatternAuth},
		{"", "checksum mismatch detected", PatternDataIntegrity},
		{"", "OOM: out of memory", PatternResourceExhaustion},
		{"", "NLP service grpc error", PatternDependencyFailure},
		{"", "invalid argument misconfig", PatternConfiguration},
		{"", "unknown error occurred", PatternTimeout},
	}

	for _, tt := range tests {
		got := ClassifyError(tt.code, tt.msg)
		if got != tt.want {
			t.Errorf("ClassifyError(%q, %q) = %q, want %q", tt.code, tt.msg, got, tt.want)
		}
	}
}

func TestAssessSeverity(t *testing.T) {
	tests := []struct {
		patternType string
		count       int
		impact      string
		want        string
	}{
		{PatternAuth, 5, "", SeverityCritical},
		{PatternAuth, 3, "", SeverityHigh},
		{PatternAuth, 1, "", SeverityLow},
		{PatternDataIntegrity, 3, "", SeverityCritical},
		{PatternDataIntegrity, 1, "", SeverityLow},
		{PatternResourceExhaustion, 3, "", SeverityHigh},
		{PatternDependencyFailure, 5, "", SeverityHigh},
		{PatternTimeout, 10, "", SeverityHigh},
		{PatternTimeout, 2, "user impact high", SeverityHigh},
		{PatternTimeout, 1, "user data affected", SeverityMedium},
		{PatternTimeout, 1, "low impact", SeverityLow},
		{PatternConfiguration, 5, "", SeverityMedium},
		{PatternConfiguration, 1, "", SeverityLow},
	}

	for _, tt := range tests {
		got := AssessSeverity(tt.patternType, tt.count, tt.impact)
		if got != tt.want {
			t.Errorf("AssessSeverity(%q, %d, %q) = %q, want %q", tt.patternType, tt.count, tt.impact, got, tt.want)
		}
	}
}

func TestRootCauseAnalysis(t *testing.T) {
	patterns := []ErrorPattern{
		{Type: PatternTimeout, Component: "nlp"},
		{Type: PatternTimeout, Component: "api"},
		{Type: PatternAuth, Component: "auth"},
		{Type: PatternDataIntegrity, Component: "ingestion"},
		{Type: PatternDataIntegrity, Component: "storage"},
		{Type: PatternResourceExhaustion, Component: "system"},
		{Type: PatternDependencyFailure, Component: "mcp"},
		{Type: PatternConfiguration, Component: "config"},
	}

	for _, p := range patterns {
		result := RootCauseAnalysis(p)
		if result == "" {
			t.Errorf("RootCauseAnalysis(%+v) returned empty string", p)
		}
	}
}

func TestSuggestFix(t *testing.T) {
	types := []string{
		PatternTimeout, PatternAuth, PatternDataIntegrity,
		PatternResourceExhaustion, PatternDependencyFailure, PatternConfiguration,
	}

	for _, pt := range types {
		p := ErrorPattern{Type: pt}
		result := SuggestFix(p)
		if result == "" {
			t.Errorf("SuggestFix(%+v) returned empty string", p)
		}
	}
}

func TestDiagnosticMonitor_RecordError(t *testing.T) {
	dm := NewDiagnosticMonitor(3, nil)

	p1 := dm.RecordError("ERR_UNAUTHORIZED", "invalid token", "auth", "user login")
	if p1.Type != PatternAuth {
		t.Errorf("expected PatternAuth, got %q", p1.Type)
	}
	if p1.Count != 1 {
		t.Errorf("expected Count=1, got %d", p1.Count)
	}

	p2 := dm.RecordError("ERR_UNAUTHORIZED", "expired token", "auth", "user login")
	if p2.Count != 2 {
		t.Errorf("expected Count=2 after second error, got %d", p2.Count)
	}

	p3 := dm.RecordError("ERR_DEADLINE_EXCEEDED", "timeout", "nlp", "production")
	if p3.Type != PatternTimeout {
		t.Errorf("expected PatternTimeout, got %q", p3.Type)
	}

	patterns := dm.GetPatterns()
	if len(patterns) != 2 {
		t.Errorf("expected 2 patterns, got %d", len(patterns))
	}
}

func TestDiagnosticMonitor_GetCriticalPatterns(t *testing.T) {
	dm := NewDiagnosticMonitor(3, nil)

	for i := 0; i < 5; i++ {
		dm.RecordError("ERR_UNAUTHORIZED", "auth failure", "auth", "production")
	}
	dm.RecordError("ERR_VALIDATION", "bad data", "api", "low")

	critical := dm.GetCriticalPatterns()
	found := false
	for _, p := range critical {
		if p.Type == PatternAuth && (p.Severity == SeverityHigh || p.Severity == SeverityCritical) {
			found = true
		}
	}
	if !found {
		t.Errorf("expected auth pattern to be high/critical severity, got %v", critical)
	}
}

func TestDiagnosticMonitor_ShouldAlert(t *testing.T) {
	dm := NewDiagnosticMonitor(3, nil)

	p1 := dm.RecordError("ERR_UNAUTHORIZED", "auth failure", "auth", "production")
	if dm.ShouldAlert(p1) {
		t.Error("should not alert after 1 occurrence")
	}

	for i := 0; i < 4; i++ {
		dm.RecordError("ERR_UNAUTHORIZED", "auth failure", "auth", "production")
	}
	patterns := dm.GetPatterns()
	for _, p := range patterns {
		if p.Type == PatternAuth && dm.ShouldAlert(p) {
			return
		}
	}
	t.Error("expected alert after 5 auth failures")
}

func TestDiagnosticMonitor_CorrelateWithHealth(t *testing.T) {
	hi := &HealthIntegration{
		GetConsecutiveFailures: func(toolID string) int { return 5 },
		GetToolHealthStatus:    func(toolID string) string { return "down" },
	}
	dm := NewDiagnosticMonitor(3, hi)

	p := ErrorPattern{Type: PatternDependencyFailure, Component: "nlp"}
	if !dm.CorrelateWithHealth(p) {
		t.Error("expected correlation with down health status")
	}
}