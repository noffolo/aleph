package sources

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDateRangeFromConfig_Empty(t *testing.T) {
	raw := json.RawMessage(`{}`)
	dr, err := ParseDateRangeFromConfig(raw)
	require.NoError(t, err)
	assert.Nil(t, dr.StartDate)
	assert.Nil(t, dr.EndDate)
}

func TestParseDateRangeFromConfig_StartOnly(t *testing.T) {
	raw := json.RawMessage(`{"start_date":"2025-05-22"}`)
	dr, err := ParseDateRangeFromConfig(raw)
	require.NoError(t, err)
	require.NotNil(t, dr.StartDate)
	assert.Equal(t, 2025, dr.StartDate.Year())
	assert.Equal(t, time.May, dr.StartDate.Month())
	assert.Equal(t, 22, dr.StartDate.Day())
	assert.Nil(t, dr.EndDate)
}

func TestParseDateRangeFromConfig_Both(t *testing.T) {
	raw := json.RawMessage(`{"start_date":"2025-01-01","end_date":"2026-05-22"}`)
	dr, err := ParseDateRangeFromConfig(raw)
	require.NoError(t, err)
	require.NotNil(t, dr.StartDate)
	require.NotNil(t, dr.EndDate)
	assert.True(t, dr.EndDate.After(*dr.StartDate))
}

func TestParseDateRangeFromConfig_RFC3339(t *testing.T) {
	raw := json.RawMessage(`{"start_date":"2025-05-22T15:04:05Z"}`)
	dr, err := ParseDateRangeFromConfig(raw)
	require.NoError(t, err)
	require.NotNil(t, dr.StartDate)
	assert.Equal(t, 15, dr.StartDate.Hour())
	assert.Equal(t, 4, dr.StartDate.Minute())
}

func TestParseDateRangeFromConfig_UnixTimestamp(t *testing.T) {
	raw := json.RawMessage(`{"start_date":1700000000}`)
	dr, err := ParseDateRangeFromConfig(raw)
	require.NoError(t, err)
	require.NotNil(t, dr.StartDate)
	assert.Equal(t, int64(1700000000), dr.StartDate.Unix())
}

func TestParseDateRangeFromConfig_InvalidFormat(t *testing.T) {
	raw := json.RawMessage(`{"start_date":"not-a-date"}`)
	_, err := ParseDateRangeFromConfig(raw)
	assert.Error(t, err)
}

func TestIsInRange_NoFilter(t *testing.T) {
	dr := DateRangeConfig{}
	now := time.Now()
	assert.True(t, dr.IsInRange(&now))
}

func TestIsInRange_NilDate(t *testing.T) {
	dr := DateRangeConfig{StartDate: ptr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))}
	assert.True(t, dr.IsInRange(nil))
}

func TestIsInRange_BeforeStart(t *testing.T) {
	start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	dr := DateRangeConfig{StartDate: &start}
	early := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	assert.False(t, dr.IsInRange(&early))
}

func TestIsInRange_AfterEnd(t *testing.T) {
	end := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	dr := DateRangeConfig{EndDate: &end}
	late := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	assert.False(t, dr.IsInRange(&late))
}

func TestIsInRange_WithinRange(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	dr := DateRangeConfig{StartDate: &start, EndDate: &end}
	mid := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	assert.True(t, dr.IsInRange(&mid))
}

func TestIsInRange_OnStartBoundary(t *testing.T) {
	start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	dr := DateRangeConfig{StartDate: &start}
	same := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	assert.True(t, dr.IsInRange(&same))
}

func TestIsInRange_OnEndBoundary(t *testing.T) {
	end := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	dr := DateRangeConfig{EndDate: &end}
	same := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	assert.True(t, dr.IsInRange(&same))
}

func ptr(t time.Time) *time.Time { return &t }
