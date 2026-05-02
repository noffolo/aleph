package safeident

import (
	"math/rand"
	"strings"
	"testing"
)

func TestValidateIdentifierAcceptsValid(t *testing.T) {
	valid := []string{
		"my_table",
		"users",
		"_private",
		"table123",
		"a",
		"_",
		"CamelCase",
		strings.Repeat("a", 64),
	}
	for _, id := range valid {
		if err := ValidateIdentifier(id); err != nil {
			t.Errorf("ValidateIdentifier(%q) = %v, want nil", id, err)
		}
	}
}

func TestValidateIdentifierRejectsSQLInjection(t *testing.T) {
	injections := []string{
		`users"; DROP TABLE accounts--`,
		`'; DELETE FROM users WHERE '1'='1`,
		`users"; INSERT INTO admin VALUES('hacked')--`,
		`users" OR "1"="1`,
		`users"; --`,
		`users/*comment*/`,
		`users*/ DROP TABLE x;--`,
		"users`; INSERT INTO x VALUES(1); --",
		`1; DROP TABLE users`,
		"users\\`; DROP TABLE users",
		`; DROP TABLE users`,
		`users' OR '1'='1`,
		`users" OR "1"="1`,
	}
	for _, id := range injections {
		if err := ValidateIdentifier(id); err == nil {
			t.Errorf("ValidateIdentifier(%q) = nil, want error (SQL injection not rejected)", id)
		}
	}
}

func TestValidateIdentifierRejectsKeywords(t *testing.T) {
	keywords := []string{
		"SELECT", "select", "Select",
		"INSERT", "insert",
		"UPDATE", "update",
		"DELETE", "delete",
		"DROP", "drop",
		"CREATE", "create",
		"ALTER", "alter",
		"TRUNCATE", "truncate",
		"REPLACE", "replace",
		"GRANT", "grant",
		"EXEC", "exec",
		"EXECUTE", "execute",
		"TABLE", "table",
		"VIEW", "view",
		"INDEX", "index",
		"DATABASE", "database",
		"SCHEMA", "schema",
		"ATTACH", "attach",
		"LOAD", "load",
		"INSTALL", "install",
		"COPY", "copy",
		"UNION", "union",
		"TRUE", "FALSE", "NULL",
		"COMMIT", "ROLLBACK", "BEGIN",
	}
	for _, kw := range keywords {
		if err := ValidateIdentifier(kw); err == nil {
			t.Errorf("ValidateIdentifier(%q) = nil, want error (keyword not rejected)", kw)
		}
	}
}

func TestValidateIdentifierRejectsInvalidFormat(t *testing.T) {
	invalid := []string{
		"",                          // empty
		"123abc",                    // starts with digit
		"my-table",                  // hyphen
		"my table",                  // space
		"my.table",                  // dot
		"my/table",                  // slash
		strings.Repeat("a", 65),     // too long
		"users\x00name",            // null byte
		"users\nname",               // newline
		"users\rname",               // carriage return
		"users\tname",               // tab
	}
	for _, id := range invalid {
		if err := ValidateIdentifier(id); err == nil {
			t.Errorf("ValidateIdentifier(%q) = nil, want error", id)
		}
	}
}

func TestValidateColumnNameAcceptsKeywords(t *testing.T) {
	// Column names in quoted position are allowed even if they're keywords
	keywords := []string{"table", "select", "index", "create", "drop"}
	for _, kw := range keywords {
		if err := ValidateColumnName(kw); err != nil {
			t.Errorf("ValidateColumnName(%q) = %v, want nil (column names accept keywords)", kw, err)
		}
	}
}

func TestValidateColumnNameRejectsInvalid(t *testing.T) {
	invalid := []string{
		"", "123abc", "my-column", strings.Repeat("a", 65),
		"col;drop", "col'inject",
	}
	for _, id := range invalid {
		if err := ValidateColumnName(id); err == nil {
			t.Errorf("ValidateColumnName(%q) = nil, want error", id)
		}
	}
}

func TestSanitizeFilePathAcceptsValid(t *testing.T) {
	valid := []string{
		"/tmp/data.csv",
		"/home/user/projects/raw/file.json",
		"raw/data.parquet",
		"file.csv",
	}
	for _, p := range valid {
		if err := SanitizeFilePath(p); err != nil {
			t.Errorf("SanitizeFilePath(%q) = %v, want nil", p, err)
		}
	}
}

func TestSanitizeFilePathRejectsTraversal(t *testing.T) {
	traversal := []string{
		"../etc/passwd",
		"/tmp/../etc/passwd",
		"./../secret",
		"..",
	}
	for _, p := range traversal {
		if err := SanitizeFilePath(p); err == nil {
			t.Errorf("SanitizeFilePath(%q) = nil, want error (traversal not rejected)", p)
		}
	}
}

func TestSanitizeFilePathRejectsMetacharacters(t *testing.T) {
	paths := []string{
		"/tmp/file;rm -rf /",
		"/tmp/file`whoami`",
		"/tmp/file$(id)",
		"/tmp/file|cat /etc/passwd",
		"/tmp/file'OR'1'='1",
		"/tmp/file&echo owned",
		"/tmp/file>output",
		"/tmp/file<input",
	}
	for _, p := range paths {
		if err := SanitizeFilePath(p); err == nil {
			t.Errorf("SanitizeFilePath(%q) = nil, want error (metacharacter not rejected)", p)
		}
	}
}

func TestQuoteIdentifier(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"users", `"users"`},
		{"my_table", `"my_table"`},
		{`say"hi"`, `"say""hi"""`},
	}
	for _, tt := range tests {
		got := QuoteIdentifier(tt.input)
		if got != tt.want {
			t.Errorf("QuoteIdentifier(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestValidateStrictIdentifier_AcceptsValid(t *testing.T) {
	valid := []string{
		"my_table",
		"users",
		"_private",
		"table123",
		"a",
		"_",
		"CamelCase",
		strings.Repeat("a", 64),
	}
	for _, id := range valid {
		if err := ValidateStrictIdentifier(id); err != nil {
			t.Errorf("ValidateStrictIdentifier(%q) = %v, want nil", id, err)
		}
	}
}

func TestValidateStrictIdentifier_RejectsSQLKeywords(t *testing.T) {
	keywords := []string{
		"SELECT", "select", "Select",
		"INSERT", "insert",
		"UPDATE", "update",
		"DELETE", "delete",
		"DROP", "drop",
		"CREATE", "create",
		"ALTER", "alter",
		"TRUNCATE", "truncate",
		"REPLACE", "replace",
		"GRANT", "grant",
		"EXEC", "exec",
		"EXECUTE", "execute",
		"TABLE", "table",
		"VIEW", "view",
		"INDEX", "index",
		"DATABASE", "database",
		"SCHEMA", "schema",
		"ATTACH", "attach",
		"LOAD", "load",
		"INSTALL", "install",
		"COPY", "copy",
		"UNION", "union",
		"TRUE", "FALSE", "NULL",
		"COMMIT", "ROLLBACK", "BEGIN",
	}
	for _, kw := range keywords {
		if err := ValidateStrictIdentifier(kw); err == nil {
			t.Errorf("ValidateStrictIdentifier(%q) = nil, want error (keyword not rejected)", kw)
		}
	}
}

func TestValidateStrictIdentifier_RejectsTooLong(t *testing.T) {
	long := strings.Repeat("a", 65)
	if err := ValidateStrictIdentifier(long); err == nil {
		t.Errorf("ValidateStrictIdentifier(65-char string) = nil, want error (too long)")
	}
}

func TestValidateStrictIdentifier_RejectsSpecialChars(t *testing.T) {
	invalid := []string{
		"",                          // empty
		"123abc",                    // starts with digit
		"my-table",                  // hyphen
		"my table",                  // space
		"my.table",                  // dot
		"my/table",                  // slash
		"users; DROP TABLE",         // semicolon injection
		"users' OR '1'='1",          // single quote
		`users" OR "1"="1`,          // double quote
		"users-- DROP",              // SQL comment
		"users/* DROP */",           // block comment
		"users\x00name",            // null byte
		"users\nname",               // newline
	}
	for _, id := range invalid {
		if err := ValidateStrictIdentifier(id); err == nil {
			t.Errorf("ValidateStrictIdentifier(%q) = nil, want error", id)
		}
	}
}

func TestQuoteStringLiteral(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "'simple'"},
		{"it's", "'it''s'"},
		{"O'Brien", "'O''Brien'"},
	}
	for _, tt := range tests {
		got := QuoteStringLiteral(tt.input)
		if got != tt.want {
			t.Errorf("QuoteStringLiteral(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// FuzzValidateIdentifier verifies that ValidateIdentifier never panics on
// arbitrary input, and that any identifier it accepts can be safely
// round-tripped through QuoteIdentifier.
func FuzzValidateIdentifier(f *testing.F) {
	seeds := []string{
		"",
		"users",
		"my_table",
		"_private",
		"123abc",
		"DROP",
		"; DROP TABLE",
		"' OR '1'='1",
		strings.Repeat("a", 65),
		"\x00",
		"\n",
		"a",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, id string) {
		err := ValidateIdentifier(id)
		if err == nil {
			// Property: accepted identifiers must have valid first char
			if len(id) > 0 && id[0] != '_' && !((id[0] >= 'a' && id[0] <= 'z') || (id[0] >= 'A' && id[0] <= 'Z')) {
				t.Errorf("ValidateIdentifier accepted %q, but first char is not letter or underscore", id)
			}
			// Property: accepted identifiers round-trip through QuoteIdentifier
			quoted := QuoteIdentifier(id)
			if !strings.HasPrefix(quoted, `"`) || !strings.HasSuffix(quoted, `"`) {
				t.Errorf("QuoteIdentifier(%q) = %q, not properly quoted", id, quoted)
			}
		}
	})
}

// TestValidateIdentifier_PropertyBased generates random valid identifiers
// and verifies they all pass validation and round-trip through quoting.
func TestValidateIdentifier_PropertyBased(t *testing.T) {
	chars := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_")
	firstChars := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_")
	keywords := []string{"SELECT", "DROP", "TABLE", "INSERT", "DELETE", "CREATE", "ALTER"}

	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 500; i++ {
		length := rng.Intn(63) + 1
		var sb strings.Builder
		sb.WriteByte(firstChars[rng.Intn(len(firstChars))])
		for j := 1; j < length; j++ {
			sb.WriteByte(chars[rng.Intn(len(chars))])
		}
		id := sb.String()

		// Skip keywords — they are explicitly rejected
		isKeyword := false
		upper := strings.ToUpper(id)
		for _, kw := range keywords {
			if upper == kw {
				isKeyword = true
				break
			}
		}
		if isKeyword {
			continue
		}

		if err := ValidateIdentifier(id); err != nil {
			t.Errorf("ValidateIdentifier(%q) = %v, want nil (generated from pattern)", id, err)
		}
		quoted := QuoteIdentifier(id)
		if !strings.HasPrefix(quoted, `"`) || !strings.HasSuffix(quoted, `"`) {
			t.Errorf("QuoteIdentifier(%q) = %q, not properly quoted", id, quoted)
		}
	}
}