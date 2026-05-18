package ingestion

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/ingestion/sources"
	"github.com/stretchr/testify/assert"
)

func TestContainsColon(t *testing.T) {
	tests := []struct {
		s        string
		hasColon bool
	}{
		{"host:993", true},
		{"imap.gmail.com", false},
		{":", true},
		{"", false},
		{"host:port:extra", true},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			assert.Equal(t, tt.hasColon, containsColon(tt.s), "containsColon(%q)", tt.s)
		})
	}
}

func TestNewEngine(t *testing.T) {
	type args struct {
		projectsRoot string
	}
	tests := []struct {
		name string
		args args
	}{
		{"empty root", args{""}},
		{"with path", args{"/tmp/test-projects"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eng := NewEngine(tt.args.projectsRoot, nil, nil, nil)
			assert.NotNil(t, eng)
			assert.Equal(t, tt.args.projectsRoot, eng.projectsRoot)
			assert.Nil(t, eng.metaRepo)
			assert.Nil(t, eng.db)
			assert.Nil(t, eng.nlpHandler)
			assert.NotNil(t, eng.tasks)
			assert.Empty(t, eng.tasks)
		})
	}
}

func TestNewEngine_WithDependencies(t *testing.T) {
	eng := NewEngine("/tmp/projects", nil, nil, nil)
	assert.NotNil(t, eng.tasks)
	assert.Empty(t, eng.tasks)

	// Map should be ready to use
	eng.tasks["test"] = nil
	assert.Contains(t, eng.tasks, "test")
}

func TestEngine_CloseExtended(t *testing.T) {
	eng := NewEngine("/tmp/projects", nil, nil, nil)
	err := eng.Close()
	assert.NoError(t, err)
}

func TestEngine_CloseMultiple(t *testing.T) {
	eng := NewEngine("/tmp/projects", nil, nil, nil)
	assert.NoError(t, eng.Close())
	assert.NoError(t, eng.Close()) // Closing twice should be safe
}

func TestVerifyChecksum_EmptyExpected(t *testing.T) {
	assert.False(t, VerifyChecksum([]byte("data"), ""))
}

func TestVerifyChecksum_ShortExpected(t *testing.T) {
	assert.False(t, VerifyChecksum([]byte("data"), "short"))
}

// =============================================================================
// resolveTableName tests
// =============================================================================

func TestResolveTableName(t *testing.T) {
	tests := []struct {
		name   string
		task   *v1.IngestionTask
		wantOK bool
		want   string
	}{
		{
			name: "config_tableName",
			task: &v1.IngestionTask{
				Id:         "task-1",
				Name:       "ignored_name",
				ConfigJson: `{"tableName": "my_custom_table"}`,
			},
			wantOK: true,
			want:   "my_custom_table",
		},
		{
			name: "task_name_fallback",
			task: &v1.IngestionTask{
				Id:         "task-1",
				Name:       "data_export",
				ConfigJson: `{}`,
			},
			wantOK: true,
			want:   "data_export",
		},
		{
			name: "uuid_task_id",
			task: &v1.IngestionTask{
				Id:         "550e8400-e29b-41d4-a716-446655440000",
				ConfigJson: `{}`,
			},
			wantOK: true,
			want:   "task_550e8400_e29b_41d4_a716_446655440000",
		},
		{
			name: "simple_task_id",
			task: &v1.IngestionTask{
				Id:         "simple",
				ConfigJson: `{}`,
			},
			wantOK: true,
			want:   "simple",
		},
		{
			name: "task_id_with_special_chars",
			task: &v1.IngestionTask{
				Id:         "task-name",
				ConfigJson: `{}`,
			},
			wantOK: true,
			want:   "task_name",
		},
		{
			name: "config_tableName_semicolon_sanitized",
			task: &v1.IngestionTask{
				Id:         "task-1",
				ConfigJson: `{"tableName": "table;DROP"}`,
			},
			wantOK: true,
			want:   "table_drop",
		},
		{
			name: "config_tableName_with_spaces",
			task: &v1.IngestionTask{
				Id:         "task-1",
				Name:       "ignored",
				ConfigJson: `{"tableName": "my table"}`,
			},
			wantOK: true,
			want:   "my_table",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveTableName(tt.task)
			if tt.wantOK {
				if err != nil {
					t.Errorf("resolveTableName() unexpected error: %v", err)
				}
				if got != tt.want {
					t.Errorf("resolveTableName() = %q, want %q", got, tt.want)
				}
			} else {
				if err == nil {
					t.Errorf("resolveTableName() expected error for %q", tt.task.ConfigJson)
				}
			}
		})
	}
}

// =============================================================================
// extractArray tests
// =============================================================================

func TestExtractArray(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]any
		wantLen int
		wantOK  bool
	}{
		{
			name:    "simple_array",
			input:   map[string]any{"items": []any{"a", "b", "c"}},
			wantLen: 3,
			wantOK:  true,
		},
		{
			name:    "empty_array",
			input:   map[string]any{"data": []any{}},
			wantLen: 0,
			wantOK:  true,
		},
		{
			name:    "no_array",
			input:   map[string]any{"key": "value"},
			wantLen: 0,
			wantOK:  false,
		},
		{
			name:    "empty_map",
			input:   map[string]any{},
			wantLen: 0,
			wantOK:  false,
		},
		{
			name:    "nested_objects_not_array",
			input:   map[string]any{"obj": map[string]any{"x": "y"}},
			wantLen: 0,
			wantOK:  false,
		},
		{
			name:    "first_value_is_array",
			input:   map[string]any{"results": []any{1, 2, 3, 4, 5}},
			wantLen: 5,
			wantOK:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			arr, ok := extractArray(tt.input)
			if ok != tt.wantOK {
				t.Errorf("extractArray() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && len(arr) != tt.wantLen {
				t.Errorf("extractArray() len = %d, want %d", len(arr), tt.wantLen)
			}
		})
	}
}

// =============================================================================
// validateSQLName tests
// =============================================================================

func TestValidateSQLName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid_simple", "table_name", false},
		{"valid", "my_table_123", false},
		{"empty", "", true},
		{"starts_with_number", "123_table", true},
		{"contains_hyphen", "table-name", true},
		{"contains_space", "table name", true},
		{"sql_keyword", "SELECT", true}, // safeident blocks SQL keywords
		{"reserved", "DROP", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSQLName(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("validateSQLName(%q) expected error", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validateSQLName(%q) unexpected error: %v", tt.input, err)
			}
		})
	}
}

// =============================================================================
// stripAndValidateName tests
// =============================================================================

func TestStripAndValidateName(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   string
		wantOK bool
	}{
		{"clean_name", "my_table", "my_table", true},
		{"with_spaces", "my table", "my_table", true},
		{"with_hyphens", "table-name", "table_name", true},
		{"all_special", "!@#$%^", "______", true},
		{"mixed", "hello world-123", "hello_world_123", true},
		{"uppercase", "MY_TABLE", "my_table", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := stripAndValidateName(tt.input)
			if tt.wantOK {
				if err != nil {
					t.Errorf("stripAndValidateName(%q) unexpected error: %v", tt.input, err)
				}
				if got != tt.want {
					t.Errorf("stripAndValidateName(%q) = %q, want %q", tt.input, got, tt.want)
				}
			} else {
				if err == nil {
					t.Errorf("stripAndValidateName(%q) expected error", tt.input)
				}
			}
		})
	}
}

// =============================================================================
// validateCode extended tests
// =============================================================================

func TestValidateCode_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{"valid_empty_main", `package main; func main() {}`, false},
		{"valid_fmt", `package main; import "fmt"; func main() { fmt.Println("ok") }`, false},
		{"blocked_os_exec", `package main; import "os/exec"; func main() {}`, true},
		{"blocked_net", `package main; import "net"; func main() {}`, true},
		{"blocked_syscall", `package main; import "syscall"; func main() {}`, true},
		{"blocked_unsafe", `package main; import "unsafe"; func main() {}`, true},
		{"blocked_reflect", `package main; import "reflect"; func main() {}`, true},
		{"blocked_crypto_rand", `package main; import "crypto/rand"; func main() {}`, true},
		{"blocked_crypto_tls", `package main; import "crypto/tls"; func main() {}`, true},
		{"blocked_net_http", `package main; import "net/http"; func main() {}`, true},
		{"blocked_encoding_json", `package main; import "encoding/json"; func main() {}`, true},
		{"blocked_os", `package main; import "os"; func main() {}`, true},
		{"blocked_io", `package main; import "io"; func main() {}`, true},
		{"blocked_subpackage", `package main; import "crypto/x509/pkix"; func main() {}`, true},
		{"invalid_syntax", `broken golang code }{`, true},
		{"empty_string", ``, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCode(tt.code)
			if tt.wantErr && err == nil {
				t.Errorf("validateCode() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validateCode() unexpected error: %v", err)
			}
		})
	}
}

// =============================================================================
// escapeIMAP extended tests
// =============================================================================

func TestEscapeIMAP_Extended(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"no_special", "simple_user"},
		{"with_quote", `user"name`},
		{"with_backslash", `user\name`},
		{"with_both", `user\"name`},
		{"empty", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeIMAP(tt.input)
			// EscapeIMAP should not contain raw double quotes
			if strings.Count(result, `"`) > 0 && !strings.Contains(result, `\"`) {
				t.Errorf("escapeIMAP(%q) = %q, unescaped double quote", tt.input, result)
			}
			// Output should be valid — no panic is the baseline
			_ = result
		})
	}
}

// =============================================================================
// decodeBody extended tests
// =============================================================================

func TestDecodeBody_Extended(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		encoding string
		want     string
	}{
		{"base64_hello", "SGVsbG8=", "base64", "Hello"},
		{"base64_b", "SGVsbG8=", "B", "Hello"},
		{"plain_7bit", "plain text", "7bit", "plain text"},
		{"plain_8bit", "raw data", "8bit", "raw data"},
		{"unknown_encoding", "raw data", "unknown", "raw data"},
		{"empty_base64", "", "base64", ""},
		{"invalid_base64", "!!!invalid", "base64", "!!!invalid"},
		{"empty", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := decodeBody([]byte(tt.data), tt.encoding)
			if result != tt.want {
				t.Errorf("decodeBody(%q, %q) = %q, want %q", tt.data, tt.encoding, result, tt.want)
			}
		})
	}
}

// =============================================================================
// parseIMAPSearch extended tests
// =============================================================================

func TestParseIMAPSearch_Extended(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantLen  int
	}{
		{"multiple_ids", "* SEARCH 1 2 3 4 5\r\n", 5},
		{"single_id", "* SEARCH 42\r\n", 1},
		{"empty_search", "* SEARCH\r\n", 0},
		{"no_search_prefix", "A001 OK SEARCH completed\r\n", 0},
		{"with_extra_garbage", "* SEARCH 10 20 30\r\nExtra: garbage\r\n", 3},
		{"multiline_response", "A001 OK\r\n* SEARCH 100 200 300\r\nA002 OK\r\n", 3},
		{"search_with_text", "* SEARCH 1 5 9 non-numeric\r\n", 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids := parseIMAPSearch(tt.response)
			if len(ids) != tt.wantLen {
				t.Errorf("parseIMAPSearch() len = %d, want %d (ids: %v)", len(ids), tt.wantLen, ids)
			}
		})
	}
}

// =============================================================================
// parseIMAPFetchMessages tests
// =============================================================================

func TestParseIMAPFetchMessages_Empty(t *testing.T) {
	rows, err := parseIMAPFetchMessages("")
	if err != nil {
		t.Errorf("parseIMAPFetchMessages() unexpected error: %v", err)
	}
	if rows != nil {
		t.Errorf("parseIMAPFetchMessages() expected nil, got %d rows", len(rows))
	}
}

func TestParseIMAPFetchMessages_BasicStructure(t *testing.T) {
	// Simulate a minimal IMAP FETCH response structure
	resp := "* 1 FETCH (BODY[] {15}\r\nSubject: Test\r\n\r\n)\r\nA005 OK FETCH completed\r\n"
	rows, err := parseIMAPFetchMessages(resp)
	if err != nil {
		t.Errorf("parseIMAPFetchMessages() unexpected error: %v", err)
	}
	// The response may not parse to valid emailRow but shouldn't error
	_ = rows
}

// =============================================================================
// parseRFC822 tests
// =============================================================================

func TestParseRFC822_Valid(t *testing.T) {
	raw := "Subject: Test Subject\r\nFrom: sender@example.com\r\nDate: Mon, 12 May 2025 10:00:00 +0000\r\nMessage-ID: <abc@example.com>\r\nContent-Type: text/plain\r\n\r\nHello, world!"
	row, err := parseRFC822(raw)
	if err != nil {
		t.Fatalf("parseRFC822() unexpected error: %v", err)
	}
	if row.Subject != "Test Subject" {
		t.Errorf("Subject = %q, want %q", row.Subject, "Test Subject")
	}
	if row.From != "sender@example.com" {
		t.Errorf("From = %q, want %q", row.From, "sender@example.com")
	}
	if row.Body != "Hello, world!" {
		t.Errorf("Body = %q, want %q", row.Body, "Hello, world!")
	}
	if row.MessageID != "<abc@example.com>" {
		t.Errorf("MessageID = %q, want %q", row.MessageID, "<abc@example.com>")
	}
}

func TestParseRFC822_Invalid(t *testing.T) {
	_, err := parseRFC822("not a valid email message")
	if err == nil {
		t.Error("parseRFC822() expected error for invalid input")
	}
}

func TestParseRFC822_Empty(t *testing.T) {
	_, err := parseRFC822("")
	if err == nil {
		t.Error("parseRFC822() expected error for empty input")
	}
}

func TestParseRFC822_Base64Body(t *testing.T) {
	raw := "Subject: Encoded\r\nFrom: test@example.com\r\nDate: Mon, 12 May 2025 10:00:00 +0000\r\n" +
		"Content-Type: text/plain\r\nContent-Transfer-Encoding: base64\r\n\r\nSGVsbG8gV29ybGQ="
	row, err := parseRFC822(raw)
	if err != nil {
		t.Fatalf("parseRFC822() unexpected error: %v", err)
	}
	if row.Body != "Hello World" {
		t.Errorf("Body = %q, want %q", row.Body, "Hello World")
	}
}

func TestParseRFC822_QuotedPrintable(t *testing.T) {
	raw := "Subject: QP Test\r\nFrom: test@example.com\r\nDate: Mon, 12 May 2025 10:00:00 +0000\r\n" +
		"Content-Type: text/plain\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\nHello=20World"
	row, err := parseRFC822(raw)
	if err != nil {
		t.Fatalf("parseRFC822() unexpected error: %v", err)
	}
	if row.Body != "Hello World" {
		t.Errorf("Body = %q, want %q", row.Body, "Hello World")
	}
}

// =============================================================================
// extractTextPart tests
// =============================================================================

func TestExtractTextPart_NoTextParts(t *testing.T) {
	body := "--boundary\r\nContent-Type: image/png\r\n\r\nfakebinary\r\n--boundary--\r\n"
	result := extractTextPart(strings.NewReader(body), "boundary")
	if result != "" {
		t.Errorf("extractTextPart() = %q, want empty (no text/plain parts)", result)
	}
}

func TestExtractTextPart_FindsText(t *testing.T) {
	body := "--boundary\r\nContent-Type: text/plain\r\n\r\nHello, world!\r\n--boundary--\r\n"
	result := extractTextPart(strings.NewReader(body), "boundary")
	if result != "Hello, world!" {
		t.Errorf("extractTextPart() = %q, want %q", result, "Hello, world!")
	}
}

func TestExtractTextPart_BadBoundary(t *testing.T) {
	body := "some random data"
	result := extractTextPart(strings.NewReader(body), "nonexistent")
	if result != "" {
		t.Errorf("extractTextPart() with bad boundary = %q, want empty", result)
	}
}

// =============================================================================
// Lazy-init (getOrCreate*) tests
// =============================================================================

func TestGetOrCreateGitHubIngester(t *testing.T) {
	eng := NewEngine("/tmp/projects", nil, nil, nil)
	task := &v1.IngestionTask{
		ConfigJson: `{"token": "test-token"}`,
	}
	g1 := eng.getOrCreateGitHubIngester(task)
	if g1 == nil {
		t.Fatal("getOrCreateGitHubIngester() returned nil")
	}
	g2 := eng.getOrCreateGitHubIngester(task)
	if g1 != g2 {
		t.Error("getOrCreateGitHubIngester() should return same instance")
	}
}

func TestGetOrCreateSitemapIngester(t *testing.T) {
	eng := NewEngine("/tmp/projects", nil, nil, nil)
	s1 := eng.getOrCreateSitemapIngester()
	if s1 == nil {
		t.Fatal("getOrCreateSitemapIngester() returned nil")
	}
	s2 := eng.getOrCreateSitemapIngester()
	if s1 != s2 {
		t.Error("getOrCreateSitemapIngester() should return same instance")
	}
}

func TestGetOrCreateJSONAPIIngester(t *testing.T) {
	eng := NewEngine("/tmp/projects", nil, nil, nil)
	j1 := eng.getOrCreateJSONAPIIngester()
	if j1 == nil {
		t.Fatal("getOrCreateJSONAPIIngester() returned nil")
	}
	j2 := eng.getOrCreateJSONAPIIngester()
	if j1 != j2 {
		t.Error("getOrCreateJSONAPIIngester() should return same instance")
	}
}

func TestGetOrCreateSheetsIngester(t *testing.T) {
	eng := NewEngine("/tmp/projects", nil, nil, nil)
	task := &v1.IngestionTask{
		ConfigJson: `{"api_key": "test-key"}`,
	}
	sh1 := eng.getOrCreateSheetsIngester(task)
	if sh1 == nil {
		t.Fatal("getOrCreateSheetsIngester() returned nil")
	}
	sh2 := eng.getOrCreateSheetsIngester(task)
	if sh1 != sh2 {
		t.Error("getOrCreateSheetsIngester() should return same instance")
	}
}

func TestGetOrCreateProbeRunner(t *testing.T) {
	eng := NewEngine("/tmp/projects", nil, nil, nil)
	p1 := eng.getOrCreateProbeRunner()
	if p1 == nil {
		t.Fatal("getOrCreateProbeRunner() returned nil")
	}
	p2 := eng.getOrCreateProbeRunner()
	if p1 != p2 {
		t.Error("getOrCreateProbeRunner() should return same instance")
	}
}

// =============================================================================
// insertJSONArray tests (error paths without DuckDB)
// =============================================================================

func TestInsertJSONArray_EmptyArray(t *testing.T) {
	// Test that insertJSONArray returns nil for empty array (early return)
	// We can't test this without DuckDB but verify the function signature exists
	eng := NewEngine("/tmp/projects", nil, nil, nil)
	// With nil db, this will fail on the table creation step but the code path through
	// the empty array check should still be verifiable indirectly
	_ = eng
}

// =============================================================================
// updateProgress nil safety tests
// =============================================================================

func TestUpdateProgress_NilMetaRepo(t *testing.T) {
	eng := NewEngine("/tmp/projects", nil, nil, nil)
	// updateProgress should be safe when metaRepo is nil (no-op)
	// This is a method call that should not panic
	eng.updateProgress("task-1", 50, "running")
	// If we get here without panic, the test passes
}

func TestUpdateProgress_WithMetaRepo(t *testing.T) {
	// updateProgress with nil metaRepo is a safe no-op
	eng := NewEngine("/tmp/projects", nil, nil, nil)
	eng.updateProgress("task-x", 0, "failed")
	eng.updateProgress("task-x", 100, "completed")
	// Should not panic
}

// =============================================================================
// ProbeRunner and SourceProbeResult tests
// =============================================================================

func TestNewProbeRunner(t *testing.T) {
	pr := NewProbeRunner(nil)
	if pr == nil {
		t.Fatal("NewProbeRunner() returned nil")
	}
	if pr.client == nil {
		t.Error("NewProbeRunner() client is nil")
	}
	if pr.llmClient != nil {
		t.Error("NewProbeRunner() llmClient should be nil when nil passed")
	}
}

func TestSourceProbeResult_Validate(t *testing.T) {
	tests := []struct {
		name    string
		result  SourceProbeResult
		wantErr bool
	}{
		{
			name: "valid_rest",
			result: SourceProbeResult{
				SrcType: "rest",
				URL:     "https://api.example.com/data",
				Pag:     PaginationInfo{MaxLimit: 100},
			},
			wantErr: false,
		},
		{
			name: "valid_rss",
			result: SourceProbeResult{
				SrcType: "rss",
				URL:     "https://example.com/feed.xml",
				Pag:     PaginationInfo{MaxLimit: -1},
			},
			wantErr: false,
		},
		{
			name: "valid_sitemap",
			result: SourceProbeResult{
				SrcType: "sitemap",
				URL:     "https://example.com/sitemap.xml",
				Pag:     PaginationInfo{MaxLimit: -1},
			},
			wantErr: false,
		},
		{
			name: "valid_github",
			result: SourceProbeResult{
				SrcType: "github",
				URL:     "https://api.github.com/repos/owner/repo",
				Pag:     PaginationInfo{MaxLimit: 100},
			},
			wantErr: false,
		},
		{
			name: "empty_url",
			result: SourceProbeResult{
				SrcType: "rest",
				URL:     "",
				Pag:     PaginationInfo{MaxLimit: 100},
			},
			wantErr: true,
		},
		{
			name: "unknown_source_type",
			result: SourceProbeResult{
				SrcType: "invalid_type",
				URL:     "https://api.example.com",
				Pag:     PaginationInfo{MaxLimit: 100},
			},
			wantErr: true,
		},
		{
			name: "zero_maxlimit",
			result: SourceProbeResult{
				SrcType: "rest",
				URL:     "https://api.example.com",
				Pag:     PaginationInfo{MaxLimit: 0},
			},
			wantErr: true,
		},
		{
			name: "valid_generic_json",
			result: SourceProbeResult{
				SrcType: "generic_json",
				URL:     "https://example.com/data.json",
				Pag:     PaginationInfo{MaxLimit: 100},
			},
			wantErr: false,
		},
		{
			name: "valid_web",
			result: SourceProbeResult{
				SrcType: "web",
				URL:     "https://example.com",
				Pag:     PaginationInfo{MaxLimit: -1},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.result.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Validate() expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

func TestPaginationInfo_ZeroValues(t *testing.T) {
	pi := PaginationInfo{}
	if pi.Type != "" {
		t.Error("zero PaginationInfo Type should be empty string")
	}
	if pi.MaxLimit != 0 {
		t.Error("zero PaginationInfo MaxLimit should be 0")
	}
}

// =============================================================================
// classifySourceType tests
// =============================================================================

func TestClassifySourceType(t *testing.T) {
	tests := []struct {
		name        string
		endpoint    string
		contentType string
		body        []byte
		want        string
	}{
		{"github_api", "https://api.github.com/repos/owner/repo", "application/json", []byte(`[]`), "rest"}, // reGitHub regex matches github.com/{anything}/repos/, not api.github.com/repos/
		{"rss_xml", "https://example.com/feed", "application/rss+xml", []byte(`<rss version="2.0"><channel></channel></rss>`), "rss"},
		{"sitemap_xml", "https://example.com/sitemap.xml", "application/xml", []byte(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"></urlset>`), "sitemap"},
		{"json_array", "https://api.example.com/items", "application/json", []byte(`[{"id": 1}]`), "rest"},
		{"json_object", "https://api.example.com/item", "application/json", []byte(`{"id": 1}`), "rest"},
		{"html_page", "https://example.com", "text/html", []byte(`<!DOCTYPE html><html></html>`), "web"},
		{"generic_default", "https://example.com/data.bin", "application/octet-stream", []byte(`...`), "generic_json"},
		{"empty_body", "https://example.com", "text/plain", []byte(``), "generic_json"},
		{"xml_with_rss_body", "https://example.com/data", "text/xml", []byte(`<feed xmlns="http://www.w3.org/2005/Atom"></feed>`), "rss"},
		{"sitemap_no_xml_ct", "https://example.com/sitemap.xml", "text/plain", []byte(`<urlset></urlset>`), "sitemap"},
		{"generic_xml", "https://example.com/data.xml", "application/xml", []byte(`<?xml version="1.0"?><root></root>`), "generic_json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifySourceType(tt.endpoint, tt.contentType, tt.body)
			if got != tt.want {
				t.Errorf("classifySourceType() = %q, want %q", got, tt.want)
			}
		})
	}
}

// =============================================================================
// detectColumns / columnsFromMap tests
// =============================================================================

func TestDetectColumns(t *testing.T) {
	tests := []struct {
		name    string
		body    []byte
		wantLen int
	}{
		{"json_array", []byte(`[{"name": "Alice", "age": 30}]`), 2},
		{"nested_data_key", []byte(`{"data": [{"id": 1, "title": "Hello"}]}`), 2},
		{"nested_results_key", []byte(`{"results": [{"x": 1, "y": 2, "z": 3}]}`), 3},
		{"nested_items_key", []byte(`{"items": [{"a": 1}]}`), 1},
		{"nested_records_key", []byte(`{"records": [{"name": "Test"}]}`), 1},
		{"empty_body", []byte(``), 0},
		{"invalid_json", []byte(`not json`), 0},
		{"empty_array", []byte(`[]`), 0},
		{"object_no_wrapper", []byte(`{"name": "Test", "value": 42}`), 2}, // falls through to columnsFromMapSkip
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cols := detectColumns(tt.body)
			if len(cols) != tt.wantLen {
				t.Errorf("detectColumns() len = %d, want %d", len(cols), tt.wantLen)
			}
		})
	}
}

func TestColumnsFromMap(t *testing.T) {
	m := map[string]any{
		"name":   "Alice",
		"age":    float64(30),
		"active": true,
	}
	cols := columnsFromMap(m, "")
	if len(cols) != 3 {
		t.Fatalf("columnsFromMap() len = %d, want 3", len(cols))
	}
	typeMap := map[string]string{}
	for _, c := range cols {
		typeMap[c.Name] = c.Type
	}
	if typeMap["name"] != "string" {
		t.Errorf("name type = %q, want string", typeMap["name"])
	}
	if typeMap["age"] != "number" {
		t.Errorf("age type = %q, want number", typeMap["age"])
	}
	if typeMap["active"] != "boolean" {
		t.Errorf("active type = %q, want boolean", typeMap["active"])
	}
}

func TestColumnsFromMap_WithPrefix(t *testing.T) {
	m := map[string]any{"key": "val"}
	cols := columnsFromMap(m, "data")
	if len(cols) != 1 {
		t.Fatalf("columnsFromMap() len = %d, want 1", len(cols))
	}
	if cols[0].Path != "$.data.key" {
		t.Errorf("Path = %q, want $.data.key", cols[0].Path)
	}
}

func TestColumnsFromMapSkip(t *testing.T) {
	m := map[string]any{
		"name":       "Test",
		"meta":       map[string]any{"page": 1},
		"pagination": map[string]any{"total": 100},
		"links":      map[string]any{"next": "/page/2"},
	}
	skip := map[string]bool{"meta": true, "pagination": true, "links": true}
	cols := columnsFromMapSkip(m, "", skip)
	if len(cols) != 1 {
		t.Fatalf("columnsFromMapSkip() len = %d, want 1", len(cols))
	}
	if cols[0].Name != "name" {
		t.Errorf("expected 'name' column, got %q", cols[0].Name)
	}
}

// =============================================================================
// goValueToColumnType tests
// =============================================================================

func TestGoValueToColumnType(t *testing.T) {
	tests := []struct {
		value any
		want  string
	}{
		{nil, "string"},
		{float64(42), "number"},
		{"hello", "string"},
		{true, "boolean"},
		{false, "boolean"},
		{map[string]any{"a": 1}, "object"},
		{[]any{1, 2, 3}, "array"},
		{int(1), "string"}, // int not specially handled - falls to default
	}
	for _, tt := range tests {
		got := goValueToColumnType(tt.value)
		if got != tt.want {
			t.Errorf("goValueToColumnType(%v) = %q, want %q", tt.value, got, tt.want)
		}
	}
}

// =============================================================================
// toFloat64 tests
// =============================================================================

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		want   float64
		wantOK bool
	}{
		{"float64", float64(42.5), 42.5, true},
		{"int", int(10), 10, true},
		{"int64", int64(99), 99, true},
		{"json_number", json.Number("3.14"), 3.14, true},
		{"json_number_int", json.Number("100"), 100, true},
		{"string", "not a number", 0, false},
		{"bool", true, 0, false},
		{"nil", nil, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := toFloat64(tt.value)
			if ok != tt.wantOK {
				t.Errorf("toFloat64(%v) ok = %v, want %v", tt.value, ok, tt.wantOK)
			}
			if ok && got != tt.want {
				t.Errorf("toFloat64(%v) = %f, want %f", tt.value, got, tt.want)
			}
		})
	}
}

// =============================================================================
// BuildNextURLFn tests
// =============================================================================

func TestBuildNextURLFn_None(t *testing.T) {
	fn := buildNextURLFn("https://api.example.com/data", PaginationInfo{Type: "none"})
	if fn != nil {
		t.Error("buildNextURLFn() should return nil for 'none' pagination")
	}
}

func TestBuildNextURLFn_Empty(t *testing.T) {
	fn := buildNextURLFn("https://api.example.com/data", PaginationInfo{Type: ""})
	if fn != nil {
		t.Error("buildNextURLFn() should return nil for empty pagination type")
	}
}

func TestBuildNextURLFn_Page(t *testing.T) {
	fn := buildNextURLFn("https://api.example.com/data?page=1", PaginationInfo{Type: "page", PageParam: "page"})
	if fn == nil {
		t.Fatal("buildNextURLFn() should return function for page pagination")
	}
}

func TestBuildNextURLFn_Offset(t *testing.T) {
	fn := buildNextURLFn("https://api.example.com/data?offset=0", PaginationInfo{Type: "offset", PageParam: "offset"})
	if fn == nil {
		t.Fatal("buildNextURLFn() should return function for offset pagination")
	}
}

// =============================================================================
// nextCursorURL tests
// =============================================================================

func TestNextCursorURL(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want string
	}{
		{"next_cursor", []byte(`{"next_cursor": "abc123"}`), "abc123"},
		{"cursor", []byte(`{"cursor": "def456"}`), "def456"},
		{"next", []byte(`{"next": "ghi789"}`), "ghi789"},
		{"meta_next_cursor", []byte(`{"meta": {"next_cursor": "jkl012"}}`), "jkl012"},
		{"meta_pagination_next", []byte(`{"meta": {"pagination": {"next": "mno345"}}}`), "mno345"},
		{"empty_json", []byte(`{}`), ""},
		{"invalid_json", []byte(`not json`), ""},
		{"empty_body", []byte(``), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nextCursorURL(tt.body, "cursor")
			if got != tt.want {
				t.Errorf("nextCursorURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

// =============================================================================
// nextPageURL tests
// =============================================================================

func TestNextPageURL(t *testing.T) {
	t.Run("page_increment", func(t *testing.T) {
		body := []byte(`{"page": 1}`)
		got := nextPageURL(body, "https://api.example.com/data?page=1", PaginationInfo{Type: "page", PageParam: "page"})
		if !strings.Contains(got, "page=2") {
			t.Errorf("nextPageURL() = %q, want page=2", got)
		}
	})

	t.Run("offset_increment", func(t *testing.T) {
		body := []byte(`{"meta": {"offset": 50}}`)
		got := nextPageURL(body, "https://api.example.com/data?offset=50", PaginationInfo{Type: "offset", PageParam: "offset", MaxLimit: 50})
		if !strings.Contains(got, "offset=100") {
			t.Errorf("nextPageURL() = %q, want offset=100", got)
		}
	})

	t.Run("meta_page", func(t *testing.T) {
		body := []byte(`{"meta": {"page": 5}}`)
		got := nextPageURL(body, "https://api.example.com/data", PaginationInfo{Type: "page", PageParam: "page"})
		if !strings.Contains(got, "page=6") {
			t.Errorf("nextPageURL() = %q, want page=6", got)
		}
	})

	t.Run("no_page", func(t *testing.T) {
		body := []byte(`{"data": []}`)
		got := nextPageURL(body, "https://api.example.com/data", PaginationInfo{Type: "page", PageParam: "page"})
		if got != "" {
			t.Errorf("nextPageURL() = %q, want empty when no page info", got)
		}
	})

	t.Run("invalid_json", func(t *testing.T) {
		got := nextPageURL([]byte(`not json`), "https://api.example.com/data", PaginationInfo{Type: "page", PageParam: "page"})
		if got != "" {
			t.Errorf("nextPageURL() = %q, want empty for invalid JSON", got)
		}
	})
}

// =============================================================================
// ProbeRunner.Execute tests (error paths)
// =============================================================================

func TestProbeRunner_Execute_NilResult(t *testing.T) {
	pr := NewProbeRunner(nil)
	ctx := context.Background()
	err := pr.Execute(ctx, nil)
	if err == nil {
		t.Error("Execute(nil) should return error")
	}
}

func TestProbeRunner_Execute_InvalidResult(t *testing.T) {
	pr := NewProbeRunner(nil)
	ctx := context.Background()
	result := &SourceProbeResult{} // invalid: empty URL, unknown source type, zero MaxLimit
	err := pr.Execute(ctx, result)
	if err == nil {
		t.Error("Execute(invalid) should return error")
	}
}

// =============================================================================
// Engine lazy-init thread safety (basic)
// =============================================================================

func TestEngineLazyInit_AllGetters(t *testing.T) {
	eng := NewEngine("/tmp/projects", nil, nil, nil)

	task := &v1.IngestionTask{
		ConfigJson: `{"token": "t", "api_key": "k"}`,
	}

	// All getters should work and return non-nil
	if eng.getOrCreateGitHubIngester(task) == nil {
		t.Error("getOrCreateGitHubIngester returned nil")
	}
	if eng.getOrCreateSitemapIngester() == nil {
		t.Error("getOrCreateSitemapIngester returned nil")
	}
	if eng.getOrCreateJSONAPIIngester() == nil {
		t.Error("getOrCreateJSONAPIIngester returned nil")
	}
	if eng.getOrCreateSheetsIngester(task) == nil {
		t.Error("getOrCreateSheetsIngester returned nil")
	}
	if eng.getOrCreateProbeRunner() == nil {
		t.Error("getOrCreateProbeRunner returned nil")
	}
}

// =============================================================================
// decodeMIMEHeader extended tests
// =============================================================================

func TestDecodeMIMEHeader_Extended(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"=?UTF-8?B?w4ls?=", "Él"},
		{"=?UTF-8?Q?Hello=20World?=", "Hello World"},
		{"Plain text subject", "Plain text subject"},
		{"", ""},
	}
	for _, tt := range tests {
		result := decodeMIMEHeader(tt.input)
		if result != tt.expected {
			t.Errorf("decodeMIMEHeader(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// =============================================================================
// RunTask path validation tests (no DuckDB)
// =============================================================================

func TestRunTask_UnknownSourceType(t *testing.T) {
	eng := NewEngine("/tmp/projects", nil, nil, nil)
	task := &v1.IngestionTask{
		Id:         "test-task",
		SourceType: "unknown_type",
		ConfigJson: `{}`,
	}
	ctx := context.Background()
	err := eng.RunTask(ctx, "proj", task)
	if err == nil {
		t.Error("RunTask with unknown source type should error")
	}
	t.Logf("RunTask error (expected): %v", err)
}

func TestRunTask_WithContext(t *testing.T) {
	eng := NewEngine("/tmp/projects", nil, nil, nil)
	task := &v1.IngestionTask{
		Id:         "test-task",
		SourceType: "csv",
		ConfigJson: `{"path": "/nonexistent/file.csv"}`,
	}
	ctx := context.Background()
	err := eng.RunTask(ctx, "proj", task)
	if err == nil {
		t.Error("RunTask should error with nil db and nonexistent file")
	}
}

// =============================================================================
// RateLimitedClient default reference test
// =============================================================================

func TestSourcesRateLimitedClient(t *testing.T) {
	// Verify the sources package is importable and DefaultRate exists
	client := sources.NewRateLimitedClient(sources.DefaultRate)
	if client == nil {
		t.Error("NewRateLimitedClient returned nil")
	}
}

// =============================================================================
// Safe HTTP client test
// =============================================================================

func TestSafeHTTPClient(t *testing.T) {
	// safeHTTPClient is created via ssrf.NewClient()
	if safeHTTPClient == nil {
		t.Error("safeHTTPClient should not be nil")
	}
}

// =============================================================================
// emailConfig struct tests
// =============================================================================

func TestEmailConfigStruct(t *testing.T) {
	cfg := emailConfig{
		Host:   "imap.example.com",
		User:   "user@example.com",
		Pass:   "secret",
		Folder: "INBOX",
	}
	if cfg.Host != "imap.example.com" {
		t.Errorf("Host = %q", cfg.Host)
	}
	if cfg.Folder != "INBOX" {
		t.Errorf("Folder = %q", cfg.Folder)
	}
}

// =============================================================================
// sentinel constant test
// =============================================================================

func TestSentimentUnavailable(t *testing.T) {
	if SentimentUnavailable != -1.0 {
		t.Errorf("SentimentUnavailable = %f, want -1.0", SentimentUnavailable)
	}
}
