package dsl

import (
	"fmt"
	"regexp"
	"strings"
)

const maxDSLInputSize = 100_000

var sqlInjectionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bDROP\s+TABLE\b`),
	regexp.MustCompile(`(?i)\bDROP\s+SCHEMA\b`),
	regexp.MustCompile(`(?i)\bTRUNCATE\b`),
	regexp.MustCompile(`(?i)\bCREATE\s+TABLE\b`),
	regexp.MustCompile(`(?i)\bALTER\s+TABLE\b`),
	regexp.MustCompile(`(?i)\bINSERT\s+INTO\b`),
	regexp.MustCompile(`(?i)\bUPDATE\s+\w+\s+SET\b`),
	regexp.MustCompile(`(?i)\bDELETE\s+FROM\b`),
	regexp.MustCompile(`(?i)\bGRANT\b`),
	regexp.MustCompile(`(?i)\bREVOKE\b`),
}

// ValidateDSLInput checks tool definitions and generated code for
// injection attacks. It enforces size limits and blocks dangerous
// SQL patterns that should never appear in DSL input.
func ValidateDSLInput(input string) error {
	if len(input) > maxDSLInputSize {
		return fmt.Errorf("DSL input exceeds maximum size of %d bytes (got %d)", maxDSLInputSize, len(input))
	}
	if strings.ContainsRune(input, 0) {
		return fmt.Errorf("DSL input contains null bytes")
	}
	for _, pat := range sqlInjectionPatterns {
		if pat.MatchString(input) {
			return fmt.Errorf("DSL input contains forbidden SQL pattern: %s", pat.String())
		}
	}
	return nil
}