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
			return "Il servizio NLP non risponde entro il timeout. Possibile sovraccarico o problema di rete."
		}
		return "Operazione scaduta. Possibile sovraccarico del sistema o latenza di rete aumentata."
	case PatternAuth:
		return "Fallimenti ripetuti di autenticazione. Possibile token scaduto, credenziali non valide o tentativo di accesso non autorizzato."
	case PatternDataIntegrity:
		if pattern.Component == "ingestion" {
			return "Dati corrotti o incompleti durante l'ingestione. Possibile problema di checksum o formato dati non valido."
		}
		return "Integrità dei dati compromessa. Possibile corruzione dati o record mancanti."
	case PatternResourceExhaustion:
		return "Risorse del sistema esaurite. Possibile memory leak, troppe connessioni aperte o sovraccarico di CPU."
	case PatternDependencyFailure:
		return "Servizio esterno non disponibile. Verificare lo stato dei servizi dipendenti (NLP, storage, MCP)."
	case PatternConfiguration:
		return "Configurazione errata o mancante. Verificare le variabili d'ambiente e i parametri di configurazione."
	default:
		return "Causa radice non determinata. Raccomandata analisi manuale."
	}
}

// SuggestFix provides actionable suggestions based on pattern type.
func SuggestFix(pattern ErrorPattern) string {
	switch pattern.Type {
	case PatternTimeout:
		return "Aumentare i timeout, verificare la connessione di rete, ridurre il carico sul servizio."
	case PatternAuth:
		return "Verificare le credenziali, rinnovare i token, controllare i permessi dell'utente."
	case PatternDataIntegrity:
		return "Eseguire la verifica checksum, ripristinare da backup, controllare i formati dei dati in ingresso."
	case PatternResourceExhaustion:
		return "Aumentare le risorse (memoria/CPU), implementare il rate limiting, chiudere le connessioni inutilizzate."
	case PatternDependencyFailure:
		return "Verificare lo stato dei servizi esterni, implementare circuit breaker, aggiungere retry con backoff."
	case PatternConfiguration:
		return "Verificare le variabili d'ambiente, controllare i file di configurazione, validare i parametri all'avvio."
	default:
		return "Analisi manuale raccomandata."
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
		return "Autenticazione fallita: " + component
	case PatternDataIntegrity:
		return "Integrità dati: " + component + " - " + truncate(errMsg, 80)
	case PatternResourceExhaustion:
		return "Risorse esaurite: " + component
	case PatternDependencyFailure:
		return "Servizio dipendente non disponibile: " + component
	case PatternConfiguration:
		return "Configurazione errata: " + component
	default:
		return "Errore: " + component + " - " + truncate(errMsg, 80)
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