package ingestion

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestEngine(t *testing.T) (*Engine, string) {
	t.Helper()
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	tmpDir := t.TempDir()
	projectsRoot := filepath.Join(tmpDir, "projects")
	require.NoError(t, os.MkdirAll(projectsRoot, 0755))

	eng := NewEngine(projectsRoot, nil, db, nil)
	return eng, projectsRoot
}

func createTestProject(t *testing.T, projectsRoot, projectID string) {
	t.Helper()
	for _, d := range []string{"raw", "ontologies", "agents", "skills", "logs"} {
		require.NoError(t, os.MkdirAll(filepath.Join(projectsRoot, projectID, d), 0755))
	}
}

func TestSanitizeIdentifier(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
	}{
		{"my_table", true},
		{"table_1", true},
		{"abc123", true},
		{"", false},
		{"table-1", false},
		{"table;DROP", false},
		{"table name", false},
		{"../../etc", false},
		{"SELECT", false},
		{"DROP", false},
	}
	for _, tt := range tests {
		err := sanitizeIdentifier(tt.id)
		if tt.valid {
			assert.NoError(t, err, "expected %q to be valid", tt.id)
		} else {
			assert.Error(t, err, "expected %q to be invalid", tt.id)
		}
	}
}

func TestSanitizeFilePath(t *testing.T) {
	assert.NoError(t, sanitizeFilePath("data.csv"))
	assert.NoError(t, sanitizeFilePath("raw/data.csv"))
	assert.NoError(t, sanitizeFilePath("/tmp/data.csv"))
	assert.Error(t, sanitizeFilePath("../etc/passwd"))
	assert.Error(t, sanitizeFilePath("foo/../../../bar"))
	assert.Error(t, sanitizeFilePath("file;rm -rf /"))
	assert.Error(t, sanitizeFilePath("file`whoami`"))
}

func TestValidateCode(t *testing.T) {
	assert.NoError(t, validateCode(`package main; import "fmt"; func main() { fmt.Println("hi") }`))
	assert.Error(t, validateCode(`package main; import "os/exec"; func main() {}`))
	assert.Error(t, validateCode(`package main; import "net"; func main() {}`))
	assert.Error(t, validateCode(`package main; import "syscall"; func main() {}`))
	assert.Error(t, validateCode(`package main; import "os/signal"; func main() {}`))
}

func TestRunTask_Dedup(t *testing.T) {
	eng, projectsRoot := setupTestEngine(t)
	createTestProject(t, projectsRoot, "test-proj")

	task := &v1.IngestionTask{
		Id:         "task-dup",
		SourceType: "csv",
		ConfigJson: `{"path": "/nonexistent/file.csv"}`,
	}

	err := eng.RunTask(context.Background(), "test-proj", task)
	assert.Error(t, err) // file doesn't exist

	// Second call should also work (previous task removed from map)
	err = eng.RunTask(context.Background(), "test-proj", task)
	assert.Error(t, err) // still file doesn't exist, but no panic
}

func TestRunTask_ConcurrentDedup(t *testing.T) {
	eng, projectsRoot := setupTestEngine(t)
	createTestProject(t, projectsRoot, "conc-proj")

	task := &v1.IngestionTask{
		Id:         "concurrent-task",
		SourceType: "csv",
		ConfigJson: `{"path": "/nonexistent.csv"}`,
	}

	results := make(chan error, 2)
	go func() { results <- eng.RunTask(context.Background(), "conc-proj", task) }()
	go func() { results <- eng.RunTask(context.Background(), "conc-proj", task) }()

	errCount := 0
	for i := 0; i < 2; i++ {
		if err := <-results; err != nil {
			errCount++
		}
	}
	assert.Equal(t, 2, errCount) // both fail (one may be dedup, other file not found)
}

func TestRunCSVLoad(t *testing.T) {
	eng, projectsRoot := setupTestEngine(t)
	createTestProject(t, projectsRoot, "csv-proj")

	csvContent := "name,age\nAlice,30\nBob,25\n"
	csvPath := filepath.Join(t.TempDir(), "people.csv")
	require.NoError(t, os.WriteFile(csvPath, []byte(csvContent), 0644))

	task := &v1.IngestionTask{
		Id:         "people",
		SourceType: "csv",
		ConfigJson: `{"path": "` + csvPath + `"}`,
	}

	// Create a temp log file
	logDir := filepath.Join(projectsRoot, "csv-proj", "logs")
	require.NoError(t, os.MkdirAll(logDir, 0755))
	logPath := filepath.Join(logDir, "people.log")
	f, err := os.Create(logPath)
	require.NoError(t, err)
	defer f.Close()

	err = eng.runCSVLoad(context.Background(), f, "csv-proj", task)
	require.NoError(t, err)

	rows, err := eng.db.Query(`SELECT name, age FROM "people" ORDER BY age`)
	require.NoError(t, err)
	defer rows.Close()

	var results []string
	for rows.Next() {
		var name string
		var age int
		require.NoError(t, rows.Scan(&name, &age))
		results = append(results, name)
	}
	assert.Equal(t, []string{"Bob", "Alice"}, results)
}

func TestRunTask_InvalidID(t *testing.T) {
	eng, projectsRoot := setupTestEngine(t)
	createTestProject(t, projectsRoot, "inv-proj")

	task := &v1.IngestionTask{
		Id:         "bad;id",
		SourceType: "csv",
		ConfigJson: `{"path": "/tmp/test.csv"}`,
	}
	err := eng.RunTask(context.Background(), "inv-proj", task)
	assert.Error(t, err)
}

func TestEngine_Close(t *testing.T) {
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	defer db.Close()
	eng := NewEngine("", nil, db, nil)
	require.NotNil(t, eng)
	assert.NoError(t, eng.Close())
	assert.NoError(t, eng.Close())
}

func TestFetchIMAP_SSRFBlocked(t *testing.T) {
	_, err := fetchIMAP("127.0.0.1:1993", "dummy", "dummy", "INBOX", 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "localhost")

	_, err = fetchIMAP("127.0.0.1", "dummy", "dummy", "INBOX", 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "localhost")
}
