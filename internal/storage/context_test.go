package storage

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/ff3300/aleph-v2/internal/safeident"
)

// FuzzSanitizeProjectID verifies that SanitizeProjectID never panics on
// arbitrary input, and that any accepted project ID round-trips safely
// through QuoteIdentifier (produces a valid double-quoted SQL identifier).
func FuzzSanitizeProjectID(f *testing.F) {
	seeds := []string{
		"my_project",
		"test123",
		"_private",
		"a",
		"A",
		strings.Repeat("a", 128),
		"",
		"123abc",
		"; DROP TABLE",
		"valid-name",
		"valid name",
		"../etc/passwd",
		"' OR '1'='1",
		`" OR "1"="1`,
		strings.Repeat("a", 256),
		"\x00",
		"\n",
		"\r",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, projectID string) {
		// Must never panic regardless of input
		err := SanitizeProjectID(projectID)
		if err == nil {
			// Property: any accepted project ID must produce a valid
			// double-quoted SQL identifier.
			quoted := safeident.QuoteIdentifier(projectID)
			if !strings.HasPrefix(quoted, `"`) || !strings.HasSuffix(quoted, `"`) {
				t.Errorf("QuoteIdentifier(%q) = %q, not properly quoted", projectID, quoted)
			}
			// Property: identifier must start with letter or underscore
			if len(projectID) > 0 && projectID[0] != '_' && !isLetter(projectID[0]) {
				t.Errorf("SanitizeProjectID accepted %q, but first char is not letter or underscore", projectID)
			}
		}
	})
}

func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// TestSanitizeProjectID_PropertyBased generates valid project IDs matching
// the allowed regex and verifies they all pass SanitizeProjectID and
// round-trip through QuoteIdentifier safely.
func TestSanitizeProjectID_PropertyBased(t *testing.T) {
	chars := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-")
	firstChars := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_")

	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 1000; i++ {
		length := rng.Intn(64) + 1
		var sb strings.Builder
		sb.WriteByte(firstChars[rng.Intn(len(firstChars))])
		for j := 1; j < length; j++ {
			sb.WriteByte(chars[rng.Intn(len(chars))])
		}
		id := sb.String()

		// Property: all generated IDs matching the pattern pass validation
		if err := SanitizeProjectID(id); err != nil {
			t.Errorf("SanitizeProjectID(%q) = %v, want nil (generated from pattern)", id, err)
		}

		// Property: round-trip through QuoteIdentifier works
		quoted := safeident.QuoteIdentifier(id)
		if !strings.HasPrefix(quoted, `"`) || !strings.HasSuffix(quoted, `"`) {
			t.Errorf("QuoteIdentifier(%q) = %q, not properly quoted", id, quoted)
		}
	}

	// Property: all invalid patterns must be rejected
	invalid := []string{
		"",
		"123abc",
		strings.Repeat("a", 129),
		"abc;drop",
		"abc'",
		`abc"`,
	}
	for _, id := range invalid {
		if err := SanitizeProjectID(id); err == nil {
			t.Errorf("SanitizeProjectID(%q) = nil, want error", id)
		}
	}
}
