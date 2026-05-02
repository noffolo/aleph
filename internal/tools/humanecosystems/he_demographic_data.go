// Package humanecosystems provides Human Ecosystems tool implementations.
package humanecosystems

// DemographicProfile holds basic demographic indicators for a country.
type DemographicProfile struct {
	CountryCode     string  // ISO 3166-1 alpha-3
	CountryName     string
	Population      int64   // 2024 estimate
	GDPPerCapita    float64 // USD, 2024 estimate
	UrbanizationPct float64 // 0-100
	MedianAge       float64 // years
	LifeExpectancy  float64 // years, at birth
}

// SocioeconomicIndicators holds socioeconomic data for a country.
type SocioeconomicIndicators struct {
	CountryCode       string
	CountryName       string
	GiniCoefficient   float64 // 0-100
	PovertyRate       float64 // % below national poverty line
	LiteracyRate      float64 // % adult literacy
	UnemploymentRate  float64 // % of labor force
}

// CulturalMetrics holds cultural/infrastructure metrics for a country.
type CulturalMetrics struct {
	CountryCode       string
	CountryName       string
	LanguageDiversity float64 // 0-1 linguistic fractionalization index
	InternetPct       float64 // % of population with internet access
	MobileAdoptionPct float64 // mobile cellular subscriptions per 100 people
}

// UrbanRuralSplit holds urban vs rural population data.
type UrbanRuralSplit struct {
	CountryCode      string
	CountryName      string
	UrbanPct         float64 // % urban
	RuralPct         float64 // % rural
	UrbanPopulation  int64
	RuralPopulation  int64
}

// MigrationFlow holds migration stock data between two countries.
type MigrationFlow struct {
	OriginCountryCode string
	DestCountryCode   string
	OriginCountryName string
	DestCountryName   string
	Stock             int64 // estimated migrant stock, 2024
}

// countryData assembles all data for a single country.
type countryData struct {
	Profile     DemographicProfile
	Socio       SocioeconomicIndicators
	Cultural    CulturalMetrics
}

var (
	// demographyData is the master embedded dataset of 30 countries.
	// Sources: World Bank Open Data / UN Population Division (2024 estimates).
	demographyData = map[string]countryData{
		"USA": {
			Profile: DemographicProfile{
				CountryCode: "USA", CountryName: "United States",
				Population: 339996563, GDPPerCapita: 80412, UrbanizationPct: 83.1, MedianAge: 38.3, LifeExpectancy: 77.4,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "USA", CountryName: "United States",
				GiniCoefficient: 41.5, PovertyRate: 11.6, LiteracyRate: 99.0, UnemploymentRate: 3.6,
			},
			Cultural: CulturalMetrics{
				CountryCode: "USA", CountryName: "United States",
				LanguageDiversity: 0.32, InternetPct: 91.8, MobileAdoptionPct: 110.6,
			},
		},
		"CHN": {
			Profile: DemographicProfile{
				CountryCode: "CHN", CountryName: "China",
				Population: 1425671352, GDPPerCapita: 12541, UrbanizationPct: 64.6, MedianAge: 39.0, LifeExpectancy: 77.1,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "CHN", CountryName: "China",
				GiniCoefficient: 38.2, PovertyRate: 0.6, LiteracyRate: 97.2, UnemploymentRate: 5.2,
			},
			Cultural: CulturalMetrics{
				CountryCode: "CHN", CountryName: "China",
				LanguageDiversity: 0.50, InternetPct: 75.6, MobileAdoptionPct: 121.8,
			},
		},
		"IND": {
			Profile: DemographicProfile{
				CountryCode: "IND", CountryName: "India",
				Population: 1428627663, GDPPerCapita: 2496, UrbanizationPct: 36.3, MedianAge: 28.4, LifeExpectancy: 70.8,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "IND", CountryName: "India",
				GiniCoefficient: 35.7, PovertyRate: 21.9, LiteracyRate: 76.3, UnemploymentRate: 7.3,
			},
			Cultural: CulturalMetrics{
				CountryCode: "IND", CountryName: "India",
				LanguageDiversity: 0.91, InternetPct: 52.4, MobileAdoptionPct: 82.6,
			},
		},
		"IDN": {
			Profile: DemographicProfile{
				CountryCode: "IDN", CountryName: "Indonesia",
				Population: 277534122, GDPPerCapita: 4788, UrbanizationPct: 58.6, MedianAge: 30.2, LifeExpectancy: 71.7,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "IDN", CountryName: "Indonesia",
				GiniCoefficient: 38.2, PovertyRate: 9.8, LiteracyRate: 96.0, UnemploymentRate: 5.3,
			},
			Cultural: CulturalMetrics{
				CountryCode: "IDN", CountryName: "Indonesia",
				LanguageDiversity: 0.81, InternetPct: 62.1, MobileAdoptionPct: 131.7,
			},
		},
		"BRA": {
			Profile: DemographicProfile{
				CountryCode: "BRA", CountryName: "Brazil",
				Population: 216422446, GDPPerCapita: 8934, UrbanizationPct: 87.6, MedianAge: 34.3, LifeExpectancy: 75.7,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "BRA", CountryName: "Brazil",
				GiniCoefficient: 53.9, PovertyRate: 19.0, LiteracyRate: 94.7, UnemploymentRate: 8.5,
			},
			Cultural: CulturalMetrics{
				CountryCode: "BRA", CountryName: "Brazil",
				LanguageDiversity: 0.07, InternetPct: 80.6, MobileAdoptionPct: 102.4,
			},
		},
		"PAK": {
			Profile: DemographicProfile{
				CountryCode: "PAK", CountryName: "Pakistan",
				Population: 241499431, GDPPerCapita: 1568, UrbanizationPct: 37.4, MedianAge: 23.0, LifeExpectancy: 67.8,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "PAK", CountryName: "Pakistan",
				GiniCoefficient: 32.0, PovertyRate: 24.3, LiteracyRate: 62.3, UnemploymentRate: 6.5,
			},
			Cultural: CulturalMetrics{
				CountryCode: "PAK", CountryName: "Pakistan",
				LanguageDiversity: 0.73, InternetPct: 36.9, MobileAdoptionPct: 78.6,
			},
		},
		"NGA": {
			Profile: DemographicProfile{
				CountryCode: "NGA", CountryName: "Nigeria",
				Population: 223804632, GDPPerCapita: 2209, UrbanizationPct: 53.4, MedianAge: 18.6, LifeExpectancy: 62.0,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "NGA", CountryName: "Nigeria",
				GiniCoefficient: 42.2, PovertyRate: 40.1, LiteracyRate: 62.0, UnemploymentRate: 14.8,
			},
			Cultural: CulturalMetrics{
				CountryCode: "NGA", CountryName: "Nigeria",
				LanguageDiversity: 0.87, InternetPct: 38.1, MobileAdoptionPct: 92.3,
			},
		},
		"BGD": {
			Profile: DemographicProfile{
				CountryCode: "BGD", CountryName: "Bangladesh",
				Population: 172954319, GDPPerCapita: 2401, UrbanizationPct: 41.8, MedianAge: 27.8, LifeExpectancy: 73.0,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "BGD", CountryName: "Bangladesh",
				GiniCoefficient: 34.8, PovertyRate: 24.3, LiteracyRate: 76.9, UnemploymentRate: 5.0,
			},
			Cultural: CulturalMetrics{
				CountryCode: "BGD", CountryName: "Bangladesh",
				LanguageDiversity: 0.14, InternetPct: 35.7, MobileAdoptionPct: 108.7,
			},
		},
		"RUS": {
			Profile: DemographicProfile{
				CountryCode: "RUS", CountryName: "Russia",
				Population: 144444359, GDPPerCapita: 11400, UrbanizationPct: 75.4, MedianAge: 40.5, LifeExpectancy: 72.6,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "RUS", CountryName: "Russia",
				GiniCoefficient: 41.0, PovertyRate: 13.5, LiteracyRate: 99.5, UnemploymentRate: 3.9,
			},
			Cultural: CulturalMetrics{
				CountryCode: "RUS", CountryName: "Russia",
				LanguageDiversity: 0.43, InternetPct: 88.2, MobileAdoptionPct: 163.0,
			},
		},
		"MEX": {
			Profile: DemographicProfile{
				CountryCode: "MEX", CountryName: "Mexico",
				Population: 128455567, GDPPerCapita: 10120, UrbanizationPct: 81.0, MedianAge: 29.3, LifeExpectancy: 75.2,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "MEX", CountryName: "Mexico",
				GiniCoefficient: 45.4, PovertyRate: 26.3, LiteracyRate: 95.3, UnemploymentRate: 3.4,
			},
			Cultural: CulturalMetrics{
				CountryCode: "MEX", CountryName: "Mexico",
				LanguageDiversity: 0.27, InternetPct: 70.5, MobileAdoptionPct: 98.4,
			},
		},
		"JPN": {
			Profile: DemographicProfile{
				CountryCode: "JPN", CountryName: "Japan",
				Population: 123294513, GDPPerCapita: 39300, UrbanizationPct: 91.9, MedianAge: 48.6, LifeExpectancy: 84.5,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "JPN", CountryName: "Japan",
				GiniCoefficient: 34.8, PovertyRate: 15.4, LiteracyRate: 99.0, UnemploymentRate: 2.6,
			},
			Cultural: CulturalMetrics{
				CountryCode: "JPN", CountryName: "Japan",
				LanguageDiversity: 0.03, InternetPct: 92.6, MobileAdoptionPct: 138.7,
			},
		},
		"ETH": {
			Profile: DemographicProfile{
				CountryCode: "ETH", CountryName: "Ethiopia",
				Population: 126527060, GDPPerCapita: 1201, UrbanizationPct: 22.6, MedianAge: 19.2, LifeExpectancy: 68.3,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "ETH", CountryName: "Ethiopia",
				GiniCoefficient: 39.4, PovertyRate: 23.5, LiteracyRate: 51.8, UnemploymentRate: 7.8,
			},
			Cultural: CulturalMetrics{
				CountryCode: "ETH", CountryName: "Ethiopia",
				LanguageDiversity: 0.80, InternetPct: 20.6, MobileAdoptionPct: 48.7,
			},
		},
		"PHL": {
			Profile: DemographicProfile{
				CountryCode: "PHL", CountryName: "Philippines",
				Population: 117337368, GDPPerCapita: 3760, UrbanizationPct: 48.1, MedianAge: 25.5, LifeExpectancy: 71.4,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "PHL", CountryName: "Philippines",
				GiniCoefficient: 44.5, PovertyRate: 16.7, LiteracyRate: 97.9, UnemploymentRate: 5.7,
			},
			Cultural: CulturalMetrics{
				CountryCode: "PHL", CountryName: "Philippines",
				LanguageDiversity: 0.84, InternetPct: 53.1, MobileAdoptionPct: 132.5,
			},
		},
		"EGY": {
			Profile: DemographicProfile{
				CountryCode: "EGY", CountryName: "Egypt",
				Population: 112716598, GDPPerCapita: 3959, UrbanizationPct: 43.1, MedianAge: 24.5, LifeExpectancy: 71.8,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "EGY", CountryName: "Egypt",
				GiniCoefficient: 31.8, PovertyRate: 29.7, LiteracyRate: 73.1, UnemploymentRate: 7.3,
			},
			Cultural: CulturalMetrics{
				CountryCode: "EGY", CountryName: "Egypt",
				LanguageDiversity: 0.10, InternetPct: 60.2, MobileAdoptionPct: 107.8,
			},
		},
		"VNM": {
			Profile: DemographicProfile{
				CountryCode: "VNM", CountryName: "Vietnam",
				Population: 98186856, GDPPerCapita: 4100, UrbanizationPct: 39.2, MedianAge: 31.9, LifeExpectancy: 75.4,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "VNM", CountryName: "Vietnam",
				GiniCoefficient: 37.6, PovertyRate: 9.8, LiteracyRate: 95.8, UnemploymentRate: 2.4,
			},
			Cultural: CulturalMetrics{
				CountryCode: "VNM", CountryName: "Vietnam",
				LanguageDiversity: 0.18, InternetPct: 73.5, MobileAdoptionPct: 129.6,
			},
		},
		"COD": {
			Profile: DemographicProfile{
				CountryCode: "COD", CountryName: "DR Congo",
				Population: 102262808, GDPPerCapita: 596, UrbanizationPct: 47.2, MedianAge: 17.0, LifeExpectancy: 60.7,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "COD", CountryName: "DR Congo",
				GiniCoefficient: 44.4, PovertyRate: 63.9, LiteracyRate: 77.0, UnemploymentRate: 7.5,
			},
			Cultural: CulturalMetrics{
				CountryCode: "COD", CountryName: "DR Congo",
				LanguageDiversity: 0.75, InternetPct: 14.2, MobileAdoptionPct: 43.1,
			},
		},
		"TUR": {
			Profile: DemographicProfile{
				CountryCode: "TUR", CountryName: "Turkey",
				Population: 85279553, GDPPerCapita: 12960, UrbanizationPct: 77.5, MedianAge: 32.2, LifeExpectancy: 78.3,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "TUR", CountryName: "Turkey",
				GiniCoefficient: 44.0, PovertyRate: 14.7, LiteracyRate: 96.7, UnemploymentRate: 10.3,
			},
			Cultural: CulturalMetrics{
				CountryCode: "TUR", CountryName: "Turkey",
				LanguageDiversity: 0.23, InternetPct: 82.8, MobileAdoptionPct: 108.6,
			},
		},
		"IRN": {
			Profile: DemographicProfile{
				CountryCode: "IRN", CountryName: "Iran",
				Population: 89172767, GDPPerCapita: 4620, UrbanizationPct: 76.3, MedianAge: 32.4, LifeExpectancy: 76.2,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "IRN", CountryName: "Iran",
				GiniCoefficient: 38.2, PovertyRate: 18.5, LiteracyRate: 88.8, UnemploymentRate: 11.6,
			},
			Cultural: CulturalMetrics{
				CountryCode: "IRN", CountryName: "Iran",
				LanguageDiversity: 0.55, InternetPct: 86.9, MobileAdoptionPct: 151.2,
			},
		},
		"DEU": {
			Profile: DemographicProfile{
				CountryCode: "DEU", CountryName: "Germany",
				Population: 83294633, GDPPerCapita: 51520, UrbanizationPct: 77.6, MedianAge: 47.1, LifeExpectancy: 80.8,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "DEU", CountryName: "Germany",
				GiniCoefficient: 32.5, PovertyRate: 16.0, LiteracyRate: 99.0, UnemploymentRate: 3.0,
			},
			Cultural: CulturalMetrics{
				CountryCode: "DEU", CountryName: "Germany",
				LanguageDiversity: 0.16, InternetPct: 93.9, MobileAdoptionPct: 129.7,
			},
		},
		"THA": {
			Profile: DemographicProfile{
				CountryCode: "THA", CountryName: "Thailand",
				Population: 71801279, GDPPerCapita: 7314, UrbanizationPct: 53.6, MedianAge: 40.1, LifeExpectancy: 79.4,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "THA", CountryName: "Thailand",
				GiniCoefficient: 43.1, PovertyRate: 6.8, LiteracyRate: 93.8, UnemploymentRate: 1.2,
			},
			Cultural: CulturalMetrics{
				CountryCode: "THA", CountryName: "Thailand",
				LanguageDiversity: 0.50, InternetPct: 66.3, MobileAdoptionPct: 180.8,
			},
		},
		"GBR": {
			Profile: DemographicProfile{
				CountryCode: "GBR", CountryName: "United Kingdom",
				Population: 67508936, GDPPerCapita: 47100, UrbanizationPct: 84.3, MedianAge: 40.6, LifeExpectancy: 80.7,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "GBR", CountryName: "United Kingdom",
				GiniCoefficient: 35.4, PovertyRate: 18.6, LiteracyRate: 99.0, UnemploymentRate: 3.9,
			},
			Cultural: CulturalMetrics{
				CountryCode: "GBR", CountryName: "United Kingdom",
				LanguageDiversity: 0.15, InternetPct: 95.9, MobileAdoptionPct: 113.8,
			},
		},
		"FRA": {
			Profile: DemographicProfile{
				CountryCode: "FRA", CountryName: "France",
				Population: 64756584, GDPPerCapita: 43550, UrbanizationPct: 81.5, MedianAge: 42.3, LifeExpectancy: 82.5,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "FRA", CountryName: "France",
				GiniCoefficient: 31.8, PovertyRate: 14.3, LiteracyRate: 99.0, UnemploymentRate: 7.1,
			},
			Cultural: CulturalMetrics{
				CountryCode: "FRA", CountryName: "France",
				LanguageDiversity: 0.16, InternetPct: 87.3, MobileAdoptionPct: 109.6,
			},
		},
		"ITA": {
			Profile: DemographicProfile{
				CountryCode: "ITA", CountryName: "Italy",
				Population: 58761971, GDPPerCapita: 35300, UrbanizationPct: 71.6, MedianAge: 47.7, LifeExpectancy: 83.2,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "ITA", CountryName: "Italy",
				GiniCoefficient: 35.4, PovertyRate: 24.9, LiteracyRate: 99.0, UnemploymentRate: 7.8,
			},
			Cultural: CulturalMetrics{
				CountryCode: "ITA", CountryName: "Italy",
				LanguageDiversity: 0.11, InternetPct: 74.9, MobileAdoptionPct: 131.1,
			},
		},
		"TZA": {
			Profile: DemographicProfile{
				CountryCode: "TZA", CountryName: "Tanzania",
				Population: 65497748, GDPPerCapita: 1206, UrbanizationPct: 36.0, MedianAge: 18.0, LifeExpectancy: 67.8,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "TZA", CountryName: "Tanzania",
				GiniCoefficient: 42.0, PovertyRate: 26.4, LiteracyRate: 82.0, UnemploymentRate: 8.7,
			},
			Cultural: CulturalMetrics{
				CountryCode: "TZA", CountryName: "Tanzania",
				LanguageDiversity: 0.89, InternetPct: 30.0, MobileAdoptionPct: 81.5,
			},
		},
		"ZAF": {
			Profile: DemographicProfile{
				CountryCode: "ZAF", CountryName: "South Africa",
				Population: 60414495, GDPPerCapita: 6770, UrbanizationPct: 68.8, MedianAge: 27.6, LifeExpectancy: 66.8,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "ZAF", CountryName: "South Africa",
				GiniCoefficient: 63.0, PovertyRate: 55.5, LiteracyRate: 87.0, UnemploymentRate: 33.6,
			},
			Cultural: CulturalMetrics{
				CountryCode: "ZAF", CountryName: "South Africa",
				LanguageDiversity: 0.83, InternetPct: 70.4, MobileAdoptionPct: 168.4,
			},
		},
		"KEN": {
			Profile: DemographicProfile{
				CountryCode: "KEN", CountryName: "Kenya",
				Population: 55100586, GDPPerCapita: 2010, UrbanizationPct: 29.5, MedianAge: 19.6, LifeExpectancy: 67.7,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "KEN", CountryName: "Kenya",
				GiniCoefficient: 40.8, PovertyRate: 36.1, LiteracyRate: 82.0, UnemploymentRate: 10.6,
			},
			Cultural: CulturalMetrics{
				CountryCode: "KEN", CountryName: "Kenya",
				LanguageDiversity: 0.86, InternetPct: 34.0, MobileAdoptionPct: 113.5,
			},
		},
		"KOR": {
			Profile: DemographicProfile{
				CountryCode: "KOR", CountryName: "South Korea",
				Population: 51784123, GDPPerCapita: 33100, UrbanizationPct: 81.4, MedianAge: 44.8, LifeExpectancy: 83.3,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "KOR", CountryName: "South Korea",
				GiniCoefficient: 31.4, PovertyRate: 14.4, LiteracyRate: 99.0, UnemploymentRate: 2.9,
			},
			Cultural: CulturalMetrics{
				CountryCode: "KOR", CountryName: "South Korea",
				LanguageDiversity: 0.01, InternetPct: 97.2, MobileAdoptionPct: 136.0,
			},
		},
		"COL": {
			Profile: DemographicProfile{
				CountryCode: "COL", CountryName: "Colombia",
				Population: 52085168, GDPPerCapita: 6330, UrbanizationPct: 81.3, MedianAge: 31.7, LifeExpectancy: 77.6,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "COL", CountryName: "Colombia",
				GiniCoefficient: 54.2, PovertyRate: 39.3, LiteracyRate: 95.6, UnemploymentRate: 9.7,
			},
			Cultural: CulturalMetrics{
				CountryCode: "COL", CountryName: "Colombia",
				LanguageDiversity: 0.03, InternetPct: 69.1, MobileAdoptionPct: 133.4,
			},
		},
		"ESP": {
			Profile: DemographicProfile{
				CountryCode: "ESP", CountryName: "Spain",
				Population: 47415132, GDPPerCapita: 32250, UrbanizationPct: 81.1, MedianAge: 45.9, LifeExpectancy: 83.3,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "ESP", CountryName: "Spain",
				GiniCoefficient: 34.4, PovertyRate: 20.7, LiteracyRate: 98.6, UnemploymentRate: 12.2,
			},
			Cultural: CulturalMetrics{
				CountryCode: "ESP", CountryName: "Spain",
				LanguageDiversity: 0.22, InternetPct: 91.5, MobileAdoptionPct: 115.3,
			},
		},
		"UKR": {
			Profile: DemographicProfile{
				CountryCode: "UKR", CountryName: "Ukraine",
				Population: 37000000, GDPPerCapita: 4500, UrbanizationPct: 69.9, MedianAge: 41.3, LifeExpectancy: 72.6,
			},
			Socio: SocioeconomicIndicators{
				CountryCode: "UKR", CountryName: "Ukraine",
				GiniCoefficient: 32.5, PovertyRate: 21.8, LiteracyRate: 99.8, UnemploymentRate: 9.4,
			},
			Cultural: CulturalMetrics{
				CountryCode: "UKR", CountryName: "Ukraine",
				LanguageDiversity: 0.22, InternetPct: 70.1, MobileAdoptionPct: 135.0,
			},
		},
	}

	// migrationFlows holds estimated migrant stock for major corridors.
	// Source: UN Population Division, International Migrant Stock 2024.
	migrationFlows = map[[2]string]MigrationFlow{
		{"MEX", "USA"}: {
			OriginCountryCode: "MEX", DestCountryCode: "USA",
			OriginCountryName: "Mexico", DestCountryName: "United States",
			Stock: 10700000,
		},
		{"IND", "USA"}: {
			OriginCountryCode: "IND", DestCountryCode: "USA",
			OriginCountryName: "India", DestCountryName: "United States",
			Stock: 2900000,
		},
		{"CHN", "USA"}: {
			OriginCountryCode: "CHN", DestCountryCode: "USA",
			OriginCountryName: "China", DestCountryName: "United States",
			Stock: 2400000,
		},
		{"PHL", "USA"}: {
			OriginCountryCode: "PHL", DestCountryCode: "USA",
			OriginCountryName: "Philippines", DestCountryName: "United States",
			Stock: 2100000,
		},
		{"IND", "ARE"}: {
			OriginCountryCode: "IND", DestCountryCode: "ARE",
			OriginCountryName: "India", DestCountryName: "United Arab Emirates",
			Stock: 3500000,
		},
		{"BGD", "IND"}: {
			OriginCountryCode: "BGD", DestCountryCode: "IND",
			OriginCountryName: "Bangladesh", DestCountryName: "India",
			Stock: 3100000,
		},
		{"TUR", "DEU"}: {
			OriginCountryCode: "TUR", DestCountryCode: "DEU",
			OriginCountryName: "Turkey", DestCountryName: "Germany",
			Stock: 2800000,
		},
		{"IND", "GBR"}: {
			OriginCountryCode: "IND", DestCountryCode: "GBR",
			OriginCountryName: "India", DestCountryName: "United Kingdom",
			Stock: 1400000,
		},
		{"PAK", "GBR"}: {
			OriginCountryCode: "PAK", DestCountryCode: "GBR",
			OriginCountryName: "Pakistan", DestCountryName: "United Kingdom",
			Stock: 1200000,
		},
		{"CHN", "JPN"}: {
			OriginCountryCode: "CHN", DestCountryCode: "JPN",
			OriginCountryName: "China", DestCountryName: "Japan",
			Stock: 750000,
		},
		{"IDN", "NLD"}: {
			OriginCountryCode: "IDN", DestCountryCode: "NLD",
			OriginCountryName: "Indonesia", DestCountryName: "Netherlands",
			Stock: 350000,
		},
		{"BRA", "USA"}: {
			OriginCountryCode: "BRA", DestCountryCode: "USA",
			OriginCountryName: "Brazil", DestCountryName: "United States",
			Stock: 540000,
		},
		{"NGA", "GBR"}: {
			OriginCountryCode: "NGA", DestCountryCode: "GBR",
			OriginCountryName: "Nigeria", DestCountryName: "United Kingdom",
			Stock: 350000,
		},
		{"DEU", "USA"}: {
			OriginCountryCode: "DEU", DestCountryCode: "USA",
			OriginCountryName: "Germany", DestCountryName: "United States",
			Stock: 1100000,
		},
		{"KOR", "USA"}: {
			OriginCountryCode: "KOR", DestCountryCode: "USA",
			OriginCountryName: "South Korea", DestCountryName: "United States",
			Stock: 1100000,
		},
		{"RUS", "DEU"}: {
			OriginCountryCode: "RUS", DestCountryCode: "DEU",
			OriginCountryName: "Russia", DestCountryName: "Germany",
			Stock: 1200000,
		},
		{"UKR", "POL"}: {
			OriginCountryCode: "UKR", DestCountryCode: "POL",
			OriginCountryName: "Ukraine", DestCountryName: "Poland",
			Stock: 1500000,
		},
	}
)

// lookupCountry returns countryData for the given ISO alpha-3 code.
func lookupCountry(code string) (countryData, bool) {
	d, ok := demographyData[code]
	return d, ok
}

// lookupMigration looks up the migration stock from origin to destination.
func lookupMigration(origin, dest string) (MigrationFlow, bool) {
	f, ok := migrationFlows[[2]string{origin, dest}]
	return f, ok
}
