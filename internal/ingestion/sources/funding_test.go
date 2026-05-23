package sources

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockFundingWM struct {
	called bool
}

func (m *mockFundingWM) Set(sourceName string, lastRun time.Time, cursor string, metadata string) error {
	m.called = true
	return nil
}

func TestPartyFundingCSV(t *testing.T) {
	csvData := "declaration_id,donation_amount,donation_year,recipient_party,donor_type,donor_name_01,donor_name_02,source_name\r\n" +
		"1,50000,2023,Partito Democratico,Persona Fisica,Mario Rossi,,Bilancio Camera 2023\r\n"

	tmpFile := filepath.Join(t.TempDir(), "political_finance.csv")
	os.WriteFile(tmpFile, []byte(csvData), 0644)

	db, err := sql.Open("duckdb", "")
	require.NoError(t, err)
	defer db.Close()

	wm := &mockFundingWM{}
	err = ImportFundingCSV(context.Background(), db, wm, tmpFile, t.TempDir())
	require.NoError(t, err)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM party_funding").Scan(&count)
	assert.Equal(t, 1, count)

	var amount float64
	db.QueryRow("SELECT donation_amount FROM party_funding LIMIT 1").Scan(&amount)
	assert.Equal(t, 50000.0, amount)

	assert.True(t, wm.called, "watermark should have been set")
}

func TestPartyFundingCSV_MultipleRows(t *testing.T) {
	csvData := "declaration_id,donation_amount,donation_year,recipient_party,donor_type,donor_name_01,donor_name_02,source_name\r\n" +
		"1,50000,2023,Partito Democratico,Persona Fisica,Mario Rossi,,Bilancio 2023\r\n" +
		"2,25000,2022,Lega,Persona Fisica,Luigi Verdi,,Bilancio 2022\r\n"

	tmpFile := filepath.Join(t.TempDir(), "political_finance.csv")
	os.WriteFile(tmpFile, []byte(csvData), 0644)

	db, err := sql.Open("duckdb", "")
	require.NoError(t, err)
	defer db.Close()

	wm := &mockFundingWM{}
	err = ImportFundingCSV(context.Background(), db, wm, tmpFile, t.TempDir())
	require.NoError(t, err)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM party_funding").Scan(&count)
	assert.Equal(t, 2, count)
}

func TestPartyFundingCSV_EmptyRows(t *testing.T) {
	csvData := "declaration_id,donation_amount,donation_year,recipient_party,donor_type,donor_name_01,donor_name_02,source_name\r\n"
	tmpFile := filepath.Join(t.TempDir(), "political_finance.csv")
	os.WriteFile(tmpFile, []byte(csvData), 0644)

	db, err := sql.Open("duckdb", "")
	require.NoError(t, err)
	defer db.Close()

	wm := &mockFundingWM{}
	err = ImportFundingCSV(context.Background(), db, wm, tmpFile, t.TempDir())
	require.NoError(t, err)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM party_funding").Scan(&count)
	assert.Equal(t, 0, count)
}

func TestPartyFundingCSV_NonexistentFile(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	require.NoError(t, err)
	defer db.Close()

	wm := &mockFundingWM{}
	err = ImportFundingCSV(context.Background(), db, wm, "/nonexistent/file.csv", t.TempDir())
	assert.Error(t, err)
}
