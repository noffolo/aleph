package sources

// ElectionDateMap maps (election_type, year) -> election_date (YYYYMMDD).
// Source: https://elezioni.interno.gov.it/ historical data
var ElectionDateMap = map[string]map[int]string{
	"politiche": {
		2006: "20060409",
		2008: "20080413",
		2013: "20130224",
		2018: "20180304",
		2022: "20220925",
	},
	"europee": {
		2009: "20090606",
		2014: "20140525",
		2019: "20190526",
		2024: "20240609",
	},
	"regionali": {
		2010: "20100328",
		2015: "20150531",
		2020: "20200920",
	},
	"comunali":   {}, // varies by comune — too many to list
	"provinciali": {},
	"referendum": {
		2006: "20060625",
		2009: "20090621",
		2011: "20110612",
		2016: "20161204",
		2020: "20200920",
		2022: "20220612",
	},
}

// GetElectionDate returns the election date for a given type and year.
// Returns empty string if not found.
func GetElectionDate(electionType string, year int) string {
	if m, ok := ElectionDateMap[electionType]; ok {
		return m[year]
	}
	return ""
}
