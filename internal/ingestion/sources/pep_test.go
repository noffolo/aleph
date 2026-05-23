package sources

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var sampleFtM = `{
  "id": "ita-deputy-123",
  "schema": "Person",
  "properties": {
    "name": ["Mario Rossi"],
    "country": ["it"],
    "birthDate": ["1970-01-15"],
    "position": ["Deputato"],
    "topics": ["role.pep"],
    "nationality": ["IT"]
  },
  "datasets": ["it_deputies"],
  "first_seen": "2018-03-23",
  "last_seen": "2024-01-01"
}`

func TestPEPParseFtM(t *testing.T) {
	entity, err := ParseFtMEntity([]byte(sampleFtM))
	require.NoError(t, err)
	assert.Equal(t, "ita-deputy-123", entity.ID)
	assert.Equal(t, "Person", entity.Schema)
	assert.Equal(t, "Mario Rossi", entity.Name)
	assert.Equal(t, "it", entity.Country)
	assert.Equal(t, "1970-01-15", entity.BirthDate)
	assert.Equal(t, "Deputato", entity.Position)
	assert.Equal(t, "it_deputies", entity.Dataset)
	assert.Equal(t, "2018-03-23", entity.FirstSeen)
	assert.Equal(t, "2024-01-01", entity.LastSeen)
}

func TestPEPParseFtM_EmptyProperties(t *testing.T) {
	raw := []byte(`{
		"id": "empty-entity-1",
		"schema": "Person",
		"properties": {},
		"datasets": []
	}`)
	entity, err := ParseFtMEntity(raw)
	require.NoError(t, err)
	assert.Equal(t, "empty-entity-1", entity.ID)
	assert.Empty(t, entity.Name)
	assert.Empty(t, entity.BirthDate)
}

func TestPEPParseFtM_Malformed(t *testing.T) {
	_, err := ParseFtMEntity([]byte(`{invalid json}`))
	assert.Error(t, err)
}

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := sql.Open("duckdb", filepath.Join(dir, "test.duckdb"))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

type testWatermarker struct {
	sources   map[string]testWatermark
	createdAt time.Time
}

type testWatermark struct {
	lastRun  time.Time
	cursor   string
	metadata string
}

func newTestWatermarker() *testWatermarker {
	return &testWatermarker{
		sources:   make(map[string]testWatermark),
		createdAt: time.Now(),
	}
}

func (w *testWatermarker) Set(sourceName string, lastRun time.Time, cursor string, metadata string) error {
	w.sources[sourceName] = testWatermark{lastRun: lastRun, cursor: cursor, metadata: metadata}
	return nil
}

func TestPEPIngestion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"entities": [` + sampleFtM + `]}`))
	}))
	defer server.Close()

	db := newTestDB(t)
	wm := newTestWatermarker()
	rawDir := t.TempDir()

	err := RunPEP(context.Background(), server.URL, db, wm, rawDir)
	require.NoError(t, err)

	// Verify data in pep_entities table
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM pep_entities").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Verify raw file saved
	rawFiles, err := os.ReadDir(filepath.Join(rawDir, "pep"))
	require.NoError(t, err)
	assert.Greater(t, len(rawFiles), 0)

	// Verify watermark set
	wmEntry, ok := wm.sources["pep"]
	require.True(t, ok, "watermark should be set for 'pep'")
	assert.Equal(t, "ita-deputy-123", wmEntry.cursor)
}

func TestPEPIngestion_MultipleEntities(t *testing.T) {
	extraEntity := `{
		"id": "ita-senator-456",
		"schema": "Person",
		"properties": {
			"name": ["Anna Bianchi"],
			"country": ["it"],
			"birthDate": ["1980-06-20"],
			"position": ["Senatore"]
		},
		"datasets": ["it_senate"],
		"first_seen": "2022-01-01",
		"last_seen": "2024-01-01"
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"entities": [` + sampleFtM + `,` + extraEntity + `]}`))
	}))
	defer server.Close()

	db := newTestDB(t)
	wm := newTestWatermarker()
	rawDir := t.TempDir()

	err := RunPEP(context.Background(), server.URL, db, wm, rawDir)
	require.NoError(t, err)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM pep_entities").Scan(&count)
	assert.Equal(t, 2, count)

	// Verify both entities
	var name string
	db.QueryRow("SELECT name FROM pep_entities WHERE id = 'ita-senator-456'").Scan(&name)
	assert.Equal(t, "Anna Bianchi", name)
}

func TestPEPIngestion_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	db := newTestDB(t)
	wm := newTestWatermarker()

	err := RunPEP(context.Background(), server.URL, db, wm, t.TempDir())
	assert.Error(t, err)
}
