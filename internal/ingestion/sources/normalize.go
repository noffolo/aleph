package sources

import (
	"strings"
	"unicode"
)

var italianPrefixes = map[string]bool{
	"de": true, "del": true, "della": true, "delle": true,
	"dei": true, "degli": true, "di": true,
	"dal": true, "dallo": true, "dalla": true,
	"dai": true, "dagli": true,
	"la": true, "lo": true, "le": true, "li": true,
	"van": true, "von": true, "ten": true,
}

func NormalizeName(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	parts := strings.Fields(raw)
	result := make([]string, len(parts))

	for i, part := range parts {
		result[i] = normalizeWord(part, i == 0)
	}

	return strings.Join(result, " ")
}

func normalizeWord(word string, isFirst bool) string {
	if word == "" {
		return word
	}

	lower := strings.ToLower(word)
	runes := []rune(lower)

	runes[0] = unicode.ToUpper(runes[0])

	for i, r := range runes {
		if r == '\'' && i+1 < len(runes) {
			runes[i+1] = unicode.ToUpper(runes[i+1])
		}
	}

	result := string(runes)

	if !isFirst && italianPrefixes[lower] && !strings.ContainsRune(word, '\'') {
		return lower
	}

	return result
}

func NormalizeFullName(cognome, nome string) string {
	return NormalizeName(cognome) + " " + NormalizeName(nome)
}
