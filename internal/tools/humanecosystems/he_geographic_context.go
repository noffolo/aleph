package humanecosystems

import (
	"context"
	"math/rand"
	"time"
)

// GeographicContext provides geographic and spatial context analysis for
// human ecosystem interactions. Coordinates are approximate and synthetic.
type GeographicContext struct {
	db *DuckDBLayer
}

// Name returns the tool name.
func (g *GeographicContext) Name() string {
	return "he_geographic_context"
}

// Description returns the tool description.
func (g *GeographicContext) Description() string {
	return "Analyzes geographic and spatial context of human ecosystem interactions (beta) | is_synthetic=true | privacy-preserving"
}

// NewGeographicContext creates a GeographicContext tool.
func NewGeographicContext(db *DuckDBLayer) *GeographicContext {
	return &GeographicContext{db: db}
}

// Execute runs geographic context analysis. Expects args with optional
// "region" string. Returns structured geographic data with synthetic coordinates.
func (g *GeographicContext) Execute(ctx context.Context, args map[string]any) (any, error) {
	region, _ := args["region"].(string)
	if region == "" {
		region = "default"
	}

	if g.db.IsAvailable() {
		return g.queryGeographic(ctx, region)
	}
	return g.syntheticGeographic(region), nil
}

func (g *GeographicContext) queryGeographic(ctx context.Context, region string) (any, error) {
	rows, err := g.db.QueryContext(ctx,
		`SELECT COUNT(*) as total FROM system_tools WHERE category LIKE 'human-ecosystems'`)
	if err != nil {
		return g.syntheticGeographic(region), nil
	}
	defer rows.Close()

	var total int
	if rows.Next() {
		rows.Scan(&total)
	}

	return map[string]interface{}{
		"region":       region,
		"tool_density": total,
		"coordinates": map[string]interface{}{
			"latitude":  0.0,
			"longitude": 0.0,
		},
		"is_synthetic": false,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (g *GeographicContext) syntheticGeographic(region string) map[string]interface{} {
	seed := int64(hashString(region))
	rng := rand.New(rand.NewSource(seed))

	lat := (rng.Float64() * 140) - 70
	lon := (rng.Float64() * 260) - 130

	clusters := make([]map[string]interface{}, 2+rng.Intn(4))
	for i := range clusters {
		clusters[i] = map[string]interface{}{
			"cluster_id": sha256Hash(region + ":cluster:" + itoa(i)),
			"latitude":   roundFloat(lat+rng.Float64()*10-5, 4),
			"longitude":  roundFloat(lon+rng.Float64()*10-5, 4),
			"density":    rng.Intn(100),
			"label":      []string{"high", "medium", "low"}[rng.Intn(3)],
			"is_synthetic": true,
		}
	}

	return map[string]interface{}{
		"region":     region,
		"coordinates": map[string]interface{}{
			"latitude":  roundFloat(lat, 4),
			"longitude": roundFloat(lon, 4),
		},
		"clusters":     clusters,
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

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	s := ""
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	return s
}
