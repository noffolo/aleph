package osint

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand" // #nosec G404 — safe: deterministic PRNG for synthetic threat data, not security-sensitive
	"time"

	"github.com/ff3300/aleph-v2/internal/repository"
)

// ThreatLevel represents a threat assessment for a target.
type ThreatLevel struct {
	Target      string  `json:"target"`
	Level       string  `json:"level"`       // low, medium, high, critical
	Confidence  float64 `json:"confidence"`  // 0.0–1.0
	Description string  `json:"description"`
	Vector      string  `json:"vector"`
	IsSynthetic bool    `json:"is_synthetic"`
	GeneratedAt string  `json:"generated_at"`
}

type ThreatLevelTool struct {
	broker *Shadowbroker
}

func NewThreatLevelTool(broker *Shadowbroker) *ThreatLevelTool {
	return &ThreatLevelTool{broker: broker}
}

// Assess evaluates the threat level for a given target.
func (t *ThreatLevelTool) Assess(ctx context.Context, target string) (map[string]interface{}, error) {
	if target == "" {
		return nil, fmt.Errorf("target is required")
	}
	return generateThreatAssessment(target), nil
}

// Execute implements the JSON→JSON tool interface.
func (t *ThreatLevelTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		Target string `json:"target"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if args.Target == "" {
		return "", fmt.Errorf("target is required")
	}
	result, err := t.Assess(ctx, args.Target)
	if err != nil {
		return "", err
	}
	out, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}
	return string(out), nil
}

func (t *ThreatLevelTool) Register(metaRepo *repository.MetadataRepository) error {
	return metaRepo.CreateTool(&repository.ToolRecord{
		ID:           "osint_threat_level",
		Name:         "osint_threat_level",
		Description:  "Threat level indicator (beta) | is_synthetic=true | privacy-preserving",
		Code:         "",
		Category:     "osint",
		Version:      "1.0.0",
		HealthStatus: "unknown",
		SourceType:   "package",
	})
}

var threatLevels = []string{"low", "medium", "high", "critical"}

func generateThreatAssessment(target string) map[string]interface{} {
	seed := int64(hashString(target))
	rng := rand.New(rand.NewSource(seed))

	levelIdx := rng.Intn(len(threatLevels))
	level := threatLevels[levelIdx]
	confidence := 0.5 + rng.Float64()*0.5 // 0.5–1.0

	descriptions := map[string]string{
		"low":      "No credible threats detected at this time.",
		"medium":   "Elevated indicators detected; routine monitoring recommended.",
		"high":     "Credible threat indicators present; immediate attention advised.",
		"critical": "Active threat detected; urgent response required.",
	}

	vectors := []string{"cyber", "physical", "economic", "social", "environmental"}
	vector := vectors[rng.Intn(len(vectors))]

	return map[string]interface{}{
		"target":       target,
		"level":        level,
		"confidence":   confidence,
		"description":  descriptions[level],
		"vector":       vector,
		"is_synthetic": true,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	}
}
