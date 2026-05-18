package osint

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand" // #nosec G404 — safe: deterministic PRNG for synthetic alert data, not security-sensitive
	"time"

	"github.com/ff3300/aleph-v2/internal/repository"
)

// CorrelationEvent represents a single event in a correlation alert.
type CorrelationEvent struct {
	Type       string  `json:"type"`
	Source     string  `json:"source"`
	Severity   string  `json:"severity"`
	Timestamp  string  `json:"timestamp"`
	Confidence float64 `json:"confidence"`
}

// CorrelationAlert represents a multi-source correlation alert.
type CorrelationAlert struct {
	AlertID     string             `json:"alert_id"`
	Title       string             `json:"title"`
	Severity    string             `json:"severity"`
	Events      []CorrelationEvent `json:"events"`
	Summary     string             `json:"summary"`
	IsSynthetic bool               `json:"is_synthetic"`
	GeneratedAt string             `json:"generated_at"`
}

type CorrelationAlertsTool struct {
	broker *Shadowbroker
}

func NewCorrelationAlertsTool(broker *Shadowbroker) *CorrelationAlertsTool {
	return &CorrelationAlertsTool{broker: broker}
}

func (t *CorrelationAlertsTool) Correlate(ctx context.Context, signals []string) (map[string]any, error) {
	if len(signals) == 0 {
		return nil, fmt.Errorf("at least one signal is required")
	}
	return generateCorrelationAlerts(signals), nil
}

// Execute implements the JSON→JSON tool interface.
func (t *CorrelationAlertsTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		Signals []string `json:"signals"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if len(args.Signals) == 0 {
		return "", fmt.Errorf("at least one signal is required")
	}
	result, err := t.Correlate(ctx, args.Signals)
	if err != nil {
		return "", err
	}
	out, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}
	return string(out), nil
}

func (t *CorrelationAlertsTool) Register(metaRepo *repository.MetadataRepository) error {
	return metaRepo.CreateTool(&repository.ToolRecord{
		ID:           "osint_correlation_alerts",
		Name:         "osint_correlation_alerts",
		Description:  "Correlation alerts engine via Shadowbroker (beta) | is_synthetic=true | privacy-preserving",
		Code:         "",
		Category:     "osint",
		Version:      "1.0.0",
		HealthStatus: "unknown",
		SourceType:   "package",
	})
}

var eventTypes = []string{"movement", "communication", "economic", "cyber", "environmental", "social"}
var eventSources = []string{"satellite_imagery_mock", "sigint_mock", "humint_mock", "open_source_mock", "financial_monitor_mock"}
var severities = []string{"info", "low", "medium", "high", "critical"}

func generateCorrelationAlerts(signals []string) map[string]any {
	joined := ""
	for i, s := range signals {
		if i > 0 {
			joined += "_"
		}
		joined += s
	}
	seed := int64(hashString(joined))
	rng := rand.New(rand.NewSource(seed))

	numAlerts := 1 + rng.Intn(3)
	alerts := make([]map[string]any, numAlerts)

	for i := 0; i < numAlerts; i++ {
		numEvents := 2 + rng.Intn(4)
		events := make([]CorrelationEvent, numEvents)

		for j := 0; j < numEvents; j++ {
			events[j] = CorrelationEvent{
				Type:       eventTypes[rng.Intn(len(eventTypes))],
				Source:     eventSources[rng.Intn(len(eventSources))],
				Severity:   severities[rng.Intn(len(severities))],
				Timestamp:  time.Now().UTC().Add(-time.Duration(rng.Intn(3600)) * time.Second).Format(time.RFC3339),
				Confidence: roundFloat(0.5+rng.Float64()*0.5, 2),
			}
		}

		severity := severities[rng.Intn(len(severities))]
		alerts[i] = map[string]any{
			"alert_id":     fmt.Sprintf("CORR-%s-%d", fmt.Sprintf("%x", seed)[:8], i+1),
			"title":        fmt.Sprintf("Correlated event cluster %d: %s signals", i+1, signals[0]),
			"severity":     severity,
			"events":       events,
			"summary":      fmt.Sprintf("Correlation of %d events from %s signals indicates %s-level activity", numEvents, signals[0], severity),
			"is_synthetic": true,
			"generated_at": time.Now().UTC().Format(time.RFC3339),
		}
	}

	return map[string]any{
		"alerts":       alerts,
		"signal_count": len(signals),
		"is_synthetic": true,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	}
}
