package sources

import (
	"fmt"
	"strings"
	"time"
)

var dateFieldKeys = []string{"pubDate", "updated", "published", "dc:date"}

var pubDateFormats = []string{
	time.RFC1123Z,
	time.RFC1123,
}

var atomDateFormats = []string{
	time.RFC3339Nano,
	time.RFC3339,
}

var dcDateFormats = []string{
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02",
}

// ExtractItemDate extracts a parsed time from RSS/Atom item attributes.
// It checks pubDate (RSS 2.0), updated (Atom), published (Atom), and dc:date (Dublin Core)
// in priority order. Returns (nil, nil) if no date field is found, and (nil, error)
// only when a date field exists but cannot be parsed.
func ExtractItemDate(attrs map[string]string) (*time.Time, error) {
	for _, key := range dateFieldKeys {
		val, ok := attrs[key]
		if !ok || strings.TrimSpace(val) == "" {
			continue
		}
		val = strings.TrimSpace(val)

		var formats []string
		switch key {
		case "pubDate":
			formats = pubDateFormats
		case "updated", "published":
			formats = atomDateFormats
		case "dc:date":
			formats = dcDateFormats
		}

		for _, layout := range formats {
			t, err := time.Parse(layout, val)
			if err == nil {
				return &t, nil
			}
		}

		return nil, fmt.Errorf("cannot parse %s=%q with known formats", key, val)
	}
	return nil, nil
}
