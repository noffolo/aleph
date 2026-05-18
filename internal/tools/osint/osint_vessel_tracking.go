package osint

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand" // #nosec G404 — safe: deterministic PRNG for synthetic vessel tracking data, not security-sensitive
	"time"

	"github.com/ff3300/aleph-v2/internal/repository"
)

// VesselData represents tracked vessel information.
type VesselData struct {
	MMSI        string  `json:"mmsi"`
	VesselName  string  `json:"vessel_name"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Speed       float64 `json:"speed"`  // knots
	Course      float64 `json:"course"` // degrees 0–359
	Status      string  `json:"status"`
	IsSynthetic bool    `json:"is_synthetic"`
	GeneratedAt string  `json:"generated_at"`
}

type VesselTrackingTool struct {
	broker *Shadowbroker
}

func NewVesselTrackingTool(broker *Shadowbroker) *VesselTrackingTool {
	return &VesselTrackingTool{broker: broker}
}

// Track returns tracking data for a vessel identified by MMSI.
func (t *VesselTrackingTool) Track(ctx context.Context, mmsi string) (map[string]any, error) {
	if mmsi == "" {
		return nil, fmt.Errorf("mmsi is required")
	}
	return generateVesselData(mmsi), nil
}

// Execute implements the JSON→JSON tool interface.
func (t *VesselTrackingTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		MMSI string `json:"mmsi"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if args.MMSI == "" {
		return "", fmt.Errorf("mmsi is required")
	}
	result, err := t.Track(ctx, args.MMSI)
	if err != nil {
		return "", err
	}
	out, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}
	return string(out), nil
}

func (t *VesselTrackingTool) Register(metaRepo *repository.MetadataRepository) error {
	return metaRepo.CreateTool(&repository.ToolRecord{
		ID:           "osint_vessel_tracking",
		Name:         "osint_vessel_tracking",
		Description:  "Vessel/AIS tracking via Shadowbroker (beta) | is_synthetic=true | privacy-preserving",
		Code:         "",
		Category:     "osint",
		Version:      "1.0.0",
		HealthStatus: "unknown",
		SourceType:   "package",
	})
}

var vesselNames = []string{
	"MV Horizon Star", "Sea Guardian", "Northern Passage", "Ocean Explorer",
	"Maritime Trader", "Coastal Venture", "Deep Blue", "Atlantic Runner",
}

func generateVesselData(mmsi string) map[string]any {
	seed := int64(hashString(mmsi))
	rng := rand.New(rand.NewSource(seed))

	lat := (rng.Float64() * 180) - 90
	lon := (rng.Float64() * 360) - 180
	speed := rng.Float64() * 30
	course := rng.Float64() * 360
	statuses := []string{"underway", "anchored", "moored", "drifting"}
	status := statuses[rng.Intn(len(statuses))]

	return map[string]any{
		"mmsi":         mmsi,
		"vessel_name":  vesselNames[int(hashString(mmsi))%len(vesselNames)],
		"latitude":     roundFloat(lat, 4),
		"longitude":    roundFloat(lon, 4),
		"speed":        roundFloat(speed, 1),
		"course":       roundFloat(course, 1),
		"status":       status,
		"is_synthetic": true,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	}
}

func roundFloat(v float64, decimals int) float64 {
	pow := 1.0
	for i := 0; i < decimals; i++ {
		pow *= 10
	}
	return float64(int(v*pow+0.5)) / pow
}
