package sources

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractItemDate_PubDateRFC1123Z(t *testing.T) {
	attrs := map[string]string{
		"pubDate": "Mon, 15 Jan 2025 10:00:00 GMT",
	}
	ts, err := ExtractItemDate(attrs)
	require.NoError(t, err)
	require.NotNil(t, ts)
	assert.Equal(t, 2025, ts.Year())
	assert.Equal(t, time.January, ts.Month())
	assert.Equal(t, 15, ts.Day())
	assert.Equal(t, 10, ts.Hour())
	assert.Equal(t, 0, ts.Minute())
}

func TestExtractItemDate_PubDateRFC1123(t *testing.T) {
	attrs := map[string]string{
		"pubDate": "Mon, 15 Jan 2025 10:00:00 UTC",
	}
	ts, err := ExtractItemDate(attrs)
	require.NoError(t, err)
	require.NotNil(t, ts)
	assert.Equal(t, 2025, ts.Year())
	assert.Equal(t, time.January, ts.Month())
	assert.Equal(t, 15, ts.Day())
}

func TestExtractItemDate_AtomUpdatedRFC3339(t *testing.T) {
	attrs := map[string]string{
		"updated": "2025-06-15T14:30:00Z",
	}
	ts, err := ExtractItemDate(attrs)
	require.NoError(t, err)
	require.NotNil(t, ts)
	assert.Equal(t, 2025, ts.Year())
	assert.Equal(t, time.June, ts.Month())
	assert.Equal(t, 15, ts.Day())
	assert.Equal(t, 14, ts.Hour())
	assert.Equal(t, 30, ts.Minute())
}

func TestExtractItemDate_AtomPublishedRFC3339Nano(t *testing.T) {
	attrs := map[string]string{
		"published": "2025-03-20T08:15:42.123456789Z",
	}
	ts, err := ExtractItemDate(attrs)
	require.NoError(t, err)
	require.NotNil(t, ts)
	assert.Equal(t, 2025, ts.Year())
	assert.Equal(t, time.March, ts.Month())
	assert.Equal(t, 20, ts.Day())
	assert.Equal(t, 8, ts.Hour())
	assert.Equal(t, 15, ts.Minute())
	assert.Equal(t, 42, ts.Second())
}

func TestExtractItemDate_DublinCoreDateOnly(t *testing.T) {
	attrs := map[string]string{
		"dc:date": "2025-05-22",
	}
	ts, err := ExtractItemDate(attrs)
	require.NoError(t, err)
	require.NotNil(t, ts)
	assert.Equal(t, 2025, ts.Year())
	assert.Equal(t, time.May, ts.Month())
	assert.Equal(t, 22, ts.Day())
}

func TestExtractItemDate_DublinCoreDateTime(t *testing.T) {
	attrs := map[string]string{
		"dc:date": "2025-05-22T14:30:00Z",
	}
	ts, err := ExtractItemDate(attrs)
	require.NoError(t, err)
	require.NotNil(t, ts)
	assert.Equal(t, 2025, ts.Year())
	assert.Equal(t, time.May, ts.Month())
	assert.Equal(t, 22, ts.Day())
	assert.Equal(t, 14, ts.Hour())
}

func TestExtractItemDate_NoDateField(t *testing.T) {
	attrs := map[string]string{
		"title":       "Some Post",
		"description": "Some content",
	}
	ts, err := ExtractItemDate(attrs)
	assert.NoError(t, err)
	assert.Nil(t, ts)
}

func TestExtractItemDate_InvalidDateString(t *testing.T) {
	attrs := map[string]string{
		"pubDate": "this-is-not-a-valid-date",
	}
	ts, err := ExtractItemDate(attrs)
	assert.Error(t, err)
	assert.Nil(t, ts)
}

func TestExtractItemDate_EmptyMap(t *testing.T) {
	attrs := map[string]string{}
	ts, err := ExtractItemDate(attrs)
	assert.NoError(t, err)
	assert.Nil(t, ts)
}

func TestExtractItemDate_PriorityUpdatedOverPublished(t *testing.T) {
	// When both "updated" and "published" are present, "updated" takes priority
	// because it's checked first in the ordered key list.
	attrs := map[string]string{
		"updated":   "2025-08-01T00:00:00Z",
		"published": "2025-01-01T00:00:00Z",
	}
	ts, err := ExtractItemDate(attrs)
	require.NoError(t, err)
	require.NotNil(t, ts)
	assert.Equal(t, 2025, ts.Year())
	assert.Equal(t, time.August, ts.Month())
}

func TestExtractItemDate_PubDateOverDcDate(t *testing.T) {
	// pubDate is checked before dc:date, so it takes priority
	attrs := map[string]string{
		"pubDate": "Mon, 01 Dec 2025 00:00:00 GMT",
		"dc:date": "2025-01-01",
	}
	ts, err := ExtractItemDate(attrs)
	require.NoError(t, err)
	require.NotNil(t, ts)
	assert.Equal(t, time.December, ts.Month())
}

func TestExtractItemDate_PubDateWithDayNameAbbreviation(t *testing.T) {
	attrs := map[string]string{
		"pubDate": "Wed, 03 Sep 2025 09:45:30 +0000",
	}
	ts, err := ExtractItemDate(attrs)
	require.NoError(t, err)
	require.NotNil(t, ts)
	assert.Equal(t, time.September, ts.Month())
	assert.Equal(t, 3, ts.Day())
	assert.Equal(t, 9, ts.Hour())
	assert.Equal(t, 45, ts.Minute())
}

func TestExtractItemDate_AtomUpdatedWithTimezone(t *testing.T) {
	attrs := map[string]string{
		"updated": "2025-11-10T12:00:00+01:00",
	}
	ts, err := ExtractItemDate(attrs)
	require.NoError(t, err)
	require.NotNil(t, ts)
	assert.Equal(t, 2025, ts.Year())
	assert.Equal(t, time.November, ts.Month())
	assert.Equal(t, 10, ts.Day())
}

func TestExtractItemDate_DublinCoreDateOnlyW3C(t *testing.T) {
	attrs := map[string]string{
		"dc:date": "2025-07-04",
	}
	ts, err := ExtractItemDate(attrs)
	require.NoError(t, err)
	require.NotNil(t, ts)
	assert.Equal(t, 2025, ts.Year())
	assert.Equal(t, time.July, ts.Month())
	assert.Equal(t, 4, ts.Day())
}

func TestExtractItemDate_InvalidAtomDate(t *testing.T) {
	attrs := map[string]string{
		"updated": "garbage-date-value",
	}
	ts, err := ExtractItemDate(attrs)
	assert.Error(t, err)
	assert.Nil(t, ts)
}

func TestExtractItemDate_EmptyStringFieldValues(t *testing.T) {
	attrs := map[string]string{
		"pubDate": "",
		"updated": "",
	}
	ts, err := ExtractItemDate(attrs)
	assert.NoError(t, err)
	assert.Nil(t, ts)
}
