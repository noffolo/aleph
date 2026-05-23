package sources

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// DateRangeConfig holds optional temporal filter bounds for ingestion tasks.
type DateRangeConfig struct {
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
}

// ParseDateRangeFromConfig extracts DateRangeConfig from a task's config_json.
// Recognised formats: YYYY-MM-DD, RFC3339, Unix timestamp (int seconds).
func ParseDateRangeFromConfig(raw json.RawMessage) (DateRangeConfig, error) {
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		return DateRangeConfig{}, nil // ignore — not a date-range issue
	}

	var dr DateRangeConfig

	if rawStart, ok := rawMap["start_date"]; ok && len(rawStart) > 0 {
		t, err := parseDateValue(rawStart)
		if err != nil {
			return DateRangeConfig{}, fmt.Errorf("start_date: %w", err)
		}
		dr.StartDate = &t
	}

	if rawEnd, ok := rawMap["end_date"]; ok && len(rawEnd) > 0 {
		t, err := parseDateValue(rawEnd)
		if err != nil {
			return DateRangeConfig{}, fmt.Errorf("end_date: %w", err)
		}
		dr.EndDate = &t
	}

	return dr, nil
}

func parseDateValue(raw json.RawMessage) (time.Time, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		s = strings.TrimSpace(s)
		// Try ISO date (YYYY-MM-DD)
		if t, err := time.Parse("2006-01-02", s); err == nil {
			return t, nil
		}
		// Try RFC3339
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t, nil
		}
		// Try RFC3339 without timezone (assume UTC)
		if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
			return t, nil
		}
		return time.Time{}, fmt.Errorf("unrecognised date format: %q", s)
	}

	// Try integer (Unix timestamp)
	var n int64
	if err := json.Unmarshal(raw, &n); err == nil {
		if n > 0 {
			return time.Unix(n, 0).UTC(), nil
		}
	}

	// Try float (Unix timestamp with fractional seconds)
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil && f > 0 {
		sec := int64(f)
		return time.Unix(sec, 0).UTC(), nil
	}

	return time.Time{}, fmt.Errorf("cannot parse date value from JSON: %s", string(raw))
}

// IsInRange checks an extracted time against the filter bounds.
// If dr has no bounds set, returns true (no filter).
// If itemDate is nil (date not extractable), returns true (include anyway).
func (dr DateRangeConfig) IsInRange(itemDate *time.Time) bool {
	if dr.StartDate == nil && dr.EndDate == nil {
		return true // no filter
	}
	if itemDate == nil {
		return true // date not extractable — include anyway
	}
	if dr.StartDate != nil && itemDate.Before(*dr.StartDate) {
		return false
	}
	if dr.EndDate != nil && itemDate.After(*dr.EndDate) {
		return false
	}
	return true
}
