package ingestion

import (
	"bufio"
	"context"
	"strings"
	"testing"

	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// stripAndValidateName edge cases
// =============================================================================

// =============================================================================
// resolveTableName more edge cases
// =============================================================================

func TestResolveTableName_ConfigHasBadTableName(t *testing.T) {
	t.Parallel()
	task := &v1.IngestionTask{
		Id:         "task-99",
		Name:       "fallback_name",
		ConfigJson: `{"tableName": "select"}`,
	}
	_, err := resolveTableName(task)
	assert.Error(t, err, "SQL keyword 'select' must be rejected as table name")
	assert.Contains(t, err.Error(), "invalid tableName")
	assert.Contains(t, err.Error(), "reserved SQL keyword")
}

func TestResolveTableName_ConfigEmptyTableName(t *testing.T) {
	t.Parallel()
	task := &v1.IngestionTask{
		Id:         "xyz-123",
		Name:       "",
		ConfigJson: `{"tableName": ""}`,
	}
	name, err := resolveTableName(task)
	require.NoError(t, err)
	assert.Equal(t, "xyz_123", name)
}

func TestResolveTableName_OnlyIDWithSpecials(t *testing.T) {
	t.Parallel()
	task := &v1.IngestionTask{
		Id:         "my task!@#",
		Name:       "",
		ConfigJson: `{}`,
	}
	name, err := resolveTableName(task)
	require.NoError(t, err)
	assert.Equal(t, "my_task___", name)
}

// =============================================================================
// validateCode more edge cases
// =============================================================================

func TestValidateCode_SpecificCryptoSubpackage(t *testing.T) {
	t.Parallel()
	err := validateCode(`package main; import "crypto/x509"; func main() {}`)
	assert.Error(t, err)
}

func TestValidateCode_NetSubpackagePrefix(t *testing.T) {
	t.Parallel()
	err := validateCode(`package main; import "net/http/httptrace"; func main() {}`)
	assert.Error(t, err)
}

func TestValidateCode_AllowedImport(t *testing.T) {
	t.Parallel()
	err := validateCode(`package main; import "strings"; func main() {}`)
	assert.NoError(t, err)
}

func TestValidateCode_EmptyImportPath(t *testing.T) {
	t.Parallel()
	// Empty code is an error already tested
	err := validateCode(`package main; func main() {}`)
	assert.NoError(t, err)
}

// =============================================================================
// readIMAPResponse tests via bufio
// =============================================================================

func TestReadIMAPResponse_OK(t *testing.T) {
	t.Parallel()
	input := "A001 OK FETCH completed\r\n"
	r := bufio.NewReader(strings.NewReader(input))
	result, err := readIMAPResponse(r, "A001")
	assert.NoError(t, err)
	assert.Contains(t, result, "A001 OK")
}

func TestReadIMAPResponse_BAD(t *testing.T) {
	t.Parallel()
	input := "A001 BAD invalid command\r\n"
	r := bufio.NewReader(strings.NewReader(input))
	_, err := readIMAPResponse(r, "A001")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "A001 BAD")
}

func TestReadIMAPResponse_NO(t *testing.T) {
	t.Parallel()
	input := "A001 NO permission denied\r\n"
	r := bufio.NewReader(strings.NewReader(input))
	_, err := readIMAPResponse(r, "A001")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "A001 NO")
}

func TestReadIMAPResponse_MultiLine(t *testing.T) {
	t.Parallel()
	input := "* 1 FETCH (BODY[] {10}\r\nHello\r\n)\r\nA001 OK done\r\n"
	r := bufio.NewReader(strings.NewReader(input))
	result, err := readIMAPResponse(r, "A001")
	assert.NoError(t, err)
	assert.Contains(t, result, "A001 OK")
}

func TestReadIMAPResponse_Incomplete(t *testing.T) {
	t.Parallel()
	input := "partial response without terminator"
	r := bufio.NewReader(strings.NewReader(input))
	_, err := readIMAPResponse(r, "A001")
	assert.Error(t, err)
}

// =============================================================================
// parseIMAPFetchMessages more edge cases
// =============================================================================

func TestParseIMAPFetchMessages_Multiple(t *testing.T) {
	t.Parallel()
	msg1 := "Subject: First\nFrom: first@test.com\n\nB1\n"
	msg2 := "Subject: Second\nFrom: second@test.com\n\nB2\n"
	resp := "* 1 FETCH \n" + msg1 + "* 2 FETCH \n" + msg2 + "A003 OK done\n"
	rows, err := parseIMAPFetchMessages(resp)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(rows), 1, "must extract at least one valid email row")
	for i := range rows {
		assert.NotEmpty(t, rows[i].Subject, "row %d subject must not be empty", i)
		assert.NotEmpty(t, rows[i].Body, "row %d body must not be empty", i)
		assert.NotEmpty(t, rows[i].From, "row %d from must not be empty", i)
	}
}

func TestParseIMAPFetchMessages_WithGarbage(t *testing.T) {
	t.Parallel()
	msg := "Subject: Solo\nFrom: solo@test.com\n\nBody\n"
	resp := "garbage line\n* 1 FETCH \n" + msg + "more garbage\nA003 OK\n"
	rows, err := parseIMAPFetchMessages(resp)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(rows), 1, "must extract at least one valid email row")
	assert.NotEmpty(t, rows[0].Subject)
	assert.NotEmpty(t, rows[0].Body)
}

// =============================================================================
// parseRFC822 multipart
// =============================================================================

func TestParseRFC822_MultipartMIME(t *testing.T) {
	t.Parallel()
	raw := "Subject: Multipart Test\r\n" +
		"From: sender@test.com\r\n" +
		"Date: Mon, 12 May 2025 10:00:00 +0000\r\n" +
		"Message-ID: <multipart@test.com>\r\n" +
		"Content-Type: multipart/mixed; boundary=\"boundary123\"\r\n" +
		"\r\n" +
		"--boundary123\r\n" +
		"Content-Type: text/plain\r\n" +
		"\r\n" +
		"Hello from text part\r\n" +
		"--boundary123--\r\n"
	row, err := parseRFC822(raw)
	assert.NoError(t, err)
	assert.Equal(t, "Multipart Test", row.Subject)
	assert.Equal(t, "Hello from text part", row.Body)
}

func TestParseRFC822_NoContentType(t *testing.T) {
	t.Parallel()
	raw := "Subject: No Type\r\nFrom: test@example.com\r\nDate: Mon, 12 May 2025 10:00:00 +0000\r\n" +
		"Message-ID: <notype@example.com>\r\n\r\nPlain body content"
	row, err := parseRFC822(raw)
	assert.NoError(t, err)
	assert.Equal(t, "Plain body content", row.Body)
}

func TestParseRFC822_BodyTruncation(t *testing.T) {
	t.Parallel()
	longBody := strings.Repeat("x", 2500)
	raw := "Subject: Long\r\nFrom: test@example.com\r\nDate: Mon, 12 May 2025 10:00:00 +0000\r\n" +
		"Message-ID: <long@example.com>\r\nContent-Type: text/plain\r\n\r\n" + longBody
	row, err := parseRFC822(raw)
	assert.NoError(t, err)
	assert.Len(t, row.Body, 2000)
}

// =============================================================================
// classifySourceType in probe.go additional tests
// =============================================================================

func TestClassifySourceType_AtomFeed(t *testing.T) {
	t.Parallel()
	body := []byte(`<feed xmlns="http://www.w3.org/2005/Atom"><entry></entry></feed>`)
	got := classifySourceType("https://example.com/atom.xml", "application/xml", body)
	assert.Equal(t, "rss", got)
}

func TestClassifySourceType_SitemapIndexXML(t *testing.T) {
	t.Parallel()
	body := []byte(`<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"></sitemapindex>`)
	got := classifySourceType("https://example.com/sitemap_index.xml", "application/xml", body)
	assert.Equal(t, "sitemap", got)
}

func TestClassifySourceType_NonXMLContentTypeButXMLBody(t *testing.T) {
	t.Parallel()
	body := []byte(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"></urlset>`)
	got := classifySourceType("https://example.com/data", "text/plain", body)
	assert.Equal(t, "sitemap", got)
}

func TestClassifySourceType_HTMLTagBody(t *testing.T) {
	t.Parallel()
	body := []byte(`<html lang="en"><head></head><body>Test</body></html>`)
	got := classifySourceType("https://example.com", "text/html", body)
	assert.Equal(t, "web", got)
}

// =============================================================================
// setupTestEngine - existing engine_test.go helper, reuse
// =============================================================================

// =============================================================================
// RunTask error paths
// =============================================================================

func TestRunTask_BogusSourceType(t *testing.T) {
	t.Parallel()
	eng, projectsRoot := setupTestEngine(t)
	createTestProject(t, projectsRoot, "proj")
	eng.projectsRoot = projectsRoot
	task := &v1.IngestionTask{
		Id:         "unknown-src",
		SourceType: "bogus_type",
		ConfigJson: `{}`,
	}
	err := eng.RunTask(context.Background(), "proj", task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown source type")
}

func TestRunTask_CustomCodeInvalid(t *testing.T) {
	t.Parallel()
	eng, projectsRoot := setupTestEngine(t)
	createTestProject(t, projectsRoot, "proj")
	eng.projectsRoot = projectsRoot
	task := &v1.IngestionTask{
		Id:         "custom-invalid",
		SourceType: "custom_code",
		ConfigJson: `{}`,
	}
	err := eng.RunTask(context.Background(), "proj", task)
	assert.Error(t, err)
}
