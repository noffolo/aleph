// Package humanecosystems provides Human Ecosystems tool implementations.
// Category: "human-ecosystems", SourceType: "package"
// is_synthetic=true, Labels=["beta"]
package humanecosystems

import (
	"context"
)

// ToolExecutor interface for all human ecosystems tools.
type ToolExecutor interface {
	Execute(ctx context.Context, args map[string]any) (any, error)
	Name() string
	Description() string
}

// ListTools returns all available human ecosystems tools.
// When dbl is nil the tools gracefully degrade to synthetic operation.
func ListTools(dbl *DuckDBLayer) []ToolExecutor {
	return []ToolExecutor{
		NewResearchProfiles(dbl),
		NewRelationalEngine(dbl),
		NewGeographicContext(dbl),
		NewPatternClassifier(dbl),
		NewPluginViz(dbl),
		NewDemographicProfileTool(dbl),
		NewSocioeconomicIndicatorsTool(dbl),
		NewCulturalMetricsTool(dbl),
		NewUrbanRuralDistributionTool(dbl),
		NewMigrationPatternsTool(dbl),
	}
}
