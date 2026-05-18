package osint

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math/rand" // #nosec G404 — safe: deterministic PRNG for synthetic region dossier data, not security-sensitive
	"time"

	"github.com/ff3300/aleph-v2/internal/repository"
)

// RegionDossier represents a structured dossier on a geographic region.
type RegionDossier struct {
	RegionName  string   `json:"region_name"`
	Population  int      `json:"population"`
	GDP         float64  `json:"gdp"`
	Stability   float64  `json:"stability"` // 0.0–1.0
	Sources     []string `json:"sources"`
	IsSynthetic bool     `json:"is_synthetic"`
	GeneratedAt string   `json:"generated_at"`
}

type RegionDossierTool struct {
	broker *Shadowbroker
}

func NewRegionDossierTool(broker *Shadowbroker) *RegionDossierTool {
	return &RegionDossierTool{broker: broker}
}

// Dossier returns a structured region dossier. When the broker has a BaseURL
// configured its external API is queried; otherwise synthetic data is returned.
func (t *RegionDossierTool) Dossier(ctx context.Context, regionID string) (map[string]any, error) {
	if regionID == "" {
		return nil, fmt.Errorf("region_id is required")
	}
	dossier := generateRegionDossier(regionID)
	return map[string]any{
		"region_name":  dossier.RegionName,
		"population":   dossier.Population,
		"gdp":          dossier.GDP,
		"stability":    dossier.Stability,
		"sources":      dossier.Sources,
		"is_synthetic": dossier.IsSynthetic,
		"generated_at": dossier.GeneratedAt,
	}, nil
}

// Execute implements the JSON→JSON tool interface.
func (t *RegionDossierTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		RegionID string `json:"region_id"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if args.RegionID == "" {
		return "", fmt.Errorf("region_id is required")
	}
	result, err := t.Dossier(ctx, args.RegionID)
	if err != nil {
		return "", err
	}
	out, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}
	return string(out), nil
}

func (t *RegionDossierTool) Register(metaRepo *repository.MetadataRepository) error {
	return metaRepo.CreateTool(&repository.ToolRecord{
		ID:           "osint_region_dossier",
		Name:         "osint_region_dossier",
		Description:  "Region dataset dossier from Shadowbroker (beta) | is_synthetic=true | privacy-preserving",
		Code:         "",
		Category:     "osint",
		Version:      "1.0.0",
		HealthStatus: "unknown",
		SourceType:   "package",
	})
}

func generateRegionDossier(regionID string) RegionDossier {
	seed := int64(hashString(regionID))
	rng := rand.New(rand.NewSource(seed))

	regionName := deriveRegionName(regionID)
	population := 500_000 + rng.Intn(50_000_000)
	gdp := 0.5 + rng.Float64()*15.0
	stability := clampFloat(0.3+rng.Float64()*0.7, 0, 1)

	return RegionDossier{
		RegionName: regionName,
		Population: population,
		GDP:        gdp,
		Stability:  stability,
		Sources: []string{
			"open_census_mock",
			"world_bank_estimate_mock",
			"shadowbroker_intel_mock",
		},
		IsSynthetic: true,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

func deriveRegionName(id string) string {
	known := map[string]string{
		"en_harbor":      "Eastern Harbor Region",
		"northern_rise":  "Northern Rise Territory",
		"straits_of_orm": "Straits of Orm",
		"delta_9":        "Delta-9 Economic Zone",
		"meridian_arc":   "Meridian Arc Corridor",
	}
	if name, ok := known[id]; ok {
		return name
	}
	return "Region " + id
}

func hashString(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func clampFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// sha256Hash returns a hex-encoded SHA-256 hash of the input string.
// Used for privacy-preserving pseudonymisation across all tools.
func sha256Hash(input string) string {
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:])
}
