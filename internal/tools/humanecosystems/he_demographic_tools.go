package humanecosystems

import (
	"context"
	"fmt"
)

// DemographicProfileTool provides demographic profile data for a country.
type DemographicProfileTool struct {
	dbl *DuckDBLayer
}

// NewDemographicProfileTool creates a new DemographicProfileTool.
func NewDemographicProfileTool(dbl *DuckDBLayer) *DemographicProfileTool {
	return &DemographicProfileTool{dbl: dbl}
}

func (t *DemographicProfileTool) Name() string { return "demographicProfile" }
func (t *DemographicProfileTool) Description() string {
	return "Returns demographic profile data for a country (ISO alpha-3 code). Fields: population, GDP per capita, urbanization rate, median age, life expectancy."
}

func (t *DemographicProfileTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	code, _ := args["countryCode"].(string)
	if code == "" {
		return nil, fmt.Errorf("countryCode is required (ISO 3166-1 alpha-3)")
	}
	d, ok := lookupCountry(code)
	if !ok {
		return nil, fmt.Errorf("unknown country code: %s", code)
	}
	return map[string]any{
		"country_code":     d.Profile.CountryCode,
		"country_name":     d.Profile.CountryName,
		"population":       d.Profile.Population,
		"gdp_per_capita":   d.Profile.GDPPerCapita,
		"urbanization_pct": d.Profile.UrbanizationPct,
		"median_age":       d.Profile.MedianAge,
		"life_expectancy":  d.Profile.LifeExpectancy,
		"is_synthetic":     !t.dbl.IsAvailable(),
	}, nil
}

// SocioeconomicIndicatorsTool provides socioeconomic data for a country.
type SocioeconomicIndicatorsTool struct {
	dbl *DuckDBLayer
}

// NewSocioeconomicIndicatorsTool creates a new SocioeconomicIndicatorsTool.
func NewSocioeconomicIndicatorsTool(dbl *DuckDBLayer) *SocioeconomicIndicatorsTool {
	return &SocioeconomicIndicatorsTool{dbl: dbl}
}

func (t *SocioeconomicIndicatorsTool) Name() string { return "socioeconomicIndicators" }
func (t *SocioeconomicIndicatorsTool) Description() string {
	return "Returns socioeconomic indicators for a country: Gini coefficient, poverty rate, literacy rate, unemployment rate."
}

func (t *SocioeconomicIndicatorsTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	code, _ := args["countryCode"].(string)
	if code == "" {
		return nil, fmt.Errorf("countryCode is required (ISO 3166-1 alpha-3)")
	}
	d, ok := lookupCountry(code)
	if !ok {
		return nil, fmt.Errorf("unknown country code: %s", code)
	}
	return map[string]any{
		"country_code":      d.Socio.CountryCode,
		"country_name":      d.Socio.CountryName,
		"gini_coefficient":  d.Socio.GiniCoefficient,
		"poverty_rate":      d.Socio.PovertyRate,
		"literacy_rate":     d.Socio.LiteracyRate,
		"unemployment_rate": d.Socio.UnemploymentRate,
		"is_synthetic":      !t.dbl.IsAvailable(),
	}, nil
}

// CulturalMetricsTool provides cultural and infrastructure metrics for a country.
type CulturalMetricsTool struct {
	dbl *DuckDBLayer
}

// NewCulturalMetricsTool creates a new CulturalMetricsTool.
func NewCulturalMetricsTool(dbl *DuckDBLayer) *CulturalMetricsTool {
	return &CulturalMetricsTool{dbl: dbl}
}

func (t *CulturalMetricsTool) Name() string { return "culturalMetrics" }
func (t *CulturalMetricsTool) Description() string {
	return "Returns cultural metrics for a country: language diversity index, internet penetration, mobile adoption."
}

func (t *CulturalMetricsTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	code, _ := args["countryCode"].(string)
	if code == "" {
		return nil, fmt.Errorf("countryCode is required (ISO 3166-1 alpha-3)")
	}
	d, ok := lookupCountry(code)
	if !ok {
		return nil, fmt.Errorf("unknown country code: %s", code)
	}
	return map[string]any{
		"country_code":        d.Cultural.CountryCode,
		"country_name":        d.Cultural.CountryName,
		"language_diversity":  d.Cultural.LanguageDiversity,
		"internet_pct":        d.Cultural.InternetPct,
		"mobile_adoption_pct": d.Cultural.MobileAdoptionPct,
		"is_synthetic":        !t.dbl.IsAvailable(),
	}, nil
}

// UrbanRuralDistributionTool computes urban vs rural population split for a country.
type UrbanRuralDistributionTool struct {
	dbl *DuckDBLayer
}

// NewUrbanRuralDistributionTool creates a new UrbanRuralDistributionTool.
func NewUrbanRuralDistributionTool(dbl *DuckDBLayer) *UrbanRuralDistributionTool {
	return &UrbanRuralDistributionTool{dbl: dbl}
}

func (t *UrbanRuralDistributionTool) Name() string { return "urbanRuralDistribution" }
func (t *UrbanRuralDistributionTool) Description() string {
	return "Computes urban vs rural population split for a country. Optional threshold defaults to 50%."
}

func (t *UrbanRuralDistributionTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	code, _ := args["countryCode"].(string)
	if code == "" {
		return nil, fmt.Errorf("countryCode is required (ISO 3166-1 alpha-3)")
	}
	d, ok := lookupCountry(code)
	if !ok {
		return nil, fmt.Errorf("unknown country code: %s", code)
	}

	threshold := 50.0
	if t, ok := args["threshold"]; ok {
		if v, ok := t.(float64); ok {
			threshold = v
		}
	}

	urbanPct := d.Profile.UrbanizationPct
	ruralPct := 100.0 - urbanPct
	urbanPop := int64(float64(d.Profile.Population) * urbanPct / 100.0)
	ruralPop := d.Profile.Population - urbanPop

	return map[string]any{
		"country_code":     code,
		"country_name":     d.Profile.CountryName,
		"urban_pct":        urbanPct,
		"rural_pct":        ruralPct,
		"urban_population": urbanPop,
		"rural_population": ruralPop,
		"above_threshold":  urbanPct >= threshold,
		"threshold_pct":    threshold,
		"classification":   classifyUrbanRural(urbanPct),
		"is_synthetic":     !t.dbl.IsAvailable(),
	}, nil
}

func classifyUrbanRural(urbanPct float64) string {
	switch {
	case urbanPct >= 80:
		return "highly_urbanized"
	case urbanPct >= 50:
		return "mostly_urban"
	case urbanPct >= 30:
		return "mixed"
	default:
		return "mostly_rural"
	}
}

// MigrationPatternsTool provides migration stock data between two countries.
type MigrationPatternsTool struct {
	dbl *DuckDBLayer
}

// NewMigrationPatternsTool creates a new MigrationPatternsTool.
func NewMigrationPatternsTool(dbl *DuckDBLayer) *MigrationPatternsTool {
	return &MigrationPatternsTool{dbl: dbl}
}

func (t *MigrationPatternsTool) Name() string { return "migrationPatterns" }
func (t *MigrationPatternsTool) Description() string {
	return "Returns estimated migration stock data between origin and destination countries (ISO alpha-3 codes)."
}

func (t *MigrationPatternsTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	origin, _ := args["originCountry"].(string)
	dest, _ := args["destCountry"].(string)
	if origin == "" || dest == "" {
		return nil, fmt.Errorf("originCountry and destCountry are required (ISO 3166-1 alpha-3)")
	}

	f, ok := lookupMigration(origin, dest)
	if !ok {
		return map[string]any{
			"origin": origin,
			"dest":   dest,
			"stock":  int64(0),
			"note":   "no data available for this migration corridor",
		}, nil
	}
	return map[string]any{
		"origin":       f.OriginCountryCode,
		"origin_name":  f.OriginCountryName,
		"dest":         f.DestCountryCode,
		"dest_name":    f.DestCountryName,
		"stock":        f.Stock,
		"is_synthetic": !t.dbl.IsAvailable(),
	}, nil
}
