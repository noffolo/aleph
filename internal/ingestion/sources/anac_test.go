package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type anacMockWatermark struct {
	lastSourceName string
}

func (m *anacMockWatermark) Set(sourceName string, lastRun time.Time, cursor string, metadata string) error {
	m.lastSourceName = sourceName
	return nil
}

func TestANACCSVEncoding(t *testing.T) {
	csvData := []byte("CIG;Anno;Importo;StazioneAppaltante;Aggiudicatario;Partecipanti\r\n" +
		"1234567ABC;2024;150000.00;Comune di Forl\xec;Societ\xe0 Esempio S.r.l.;\r\n")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv; charset=ISO-8859-1")
		w.Write(csvData)
	}))
	defer server.Close()

	db := setupTestDuckDB(t)
	wm := &anacMockWatermark{}
	err := RunANAC(context.Background(), server.URL, db, wm, 2024, t.TempDir())
	require.NoError(t, err)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM public_contracts").Scan(&count)
	assert.Equal(t, 1, count)

	var name string
	db.QueryRow("SELECT stazione_appaltante FROM public_contracts LIMIT 1").Scan(&name)
	assert.Contains(t, name, "Forlì")
}

func TestANACCSVMultipleRows(t *testing.T) {
	csvData := []byte("CIG;Anno;Importo;StazioneAppaltante;Aggiudicatario;Partecipanti\r\n" +
		"AAA111;2024;100000.00;Comune Roma;Società A;\r\n" +
		"BBB222;2024;200000.00;Comune Milano;Società B;\r\n")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(csvData)
	}))
	defer server.Close()

	db := setupTestDuckDB(t)
	wm := &anacMockWatermark{}
	err := RunANAC(context.Background(), server.URL, db, wm, 2024, t.TempDir())
	require.NoError(t, err)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM public_contracts").Scan(&count)
	assert.Equal(t, 2, count)
}

func TestANACHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	db := setupTestDuckDB(t)
	wm := &anacMockWatermark{}
	err := RunANAC(context.Background(), server.URL, db, wm, 2024, t.TempDir())
	require.Error(t, err)
}

func TestANACDryRun_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := RunANACDryRun(server.URL, 2024)
	require.NoError(t, err)
}

func TestANACDryRun_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	err := RunANACDryRun(server.URL, 2024)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// UTF-8 CSV without encoding header should still load correctly
func TestANACMissingHeader(t *testing.T) {
	csvData := []byte("CIG;Anno;Importo;StazioneAppaltante;Aggiudicatario;Partecipanti\r\n" +
		"XYZ999;2024;50000.00;Comune Napoli;Ditta Napoli S.p.A.;\r\n")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(csvData)
	}))
	defer server.Close()

	db := setupTestDuckDB(t)
	wm := &anacMockWatermark{}
	err := RunANAC(context.Background(), server.URL, db, wm, 2024, t.TempDir())
	require.NoError(t, err)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM public_contracts").Scan(&count)
	assert.Equal(t, 1, count)
}
