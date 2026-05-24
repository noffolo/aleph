package sources

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type opdmMockWatermark struct {
	lastSourceName string
	lastCursor     string
}

func (m *opdmMockWatermark) Set(sourceName string, lastRun time.Time, cursor string, metadata string) error {
	m.lastSourceName = sourceName
	m.lastCursor = cursor
	return nil
}

func TestOPDMPagination(t *testing.T) {
	page := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		if page <= 2 {
			nextURL := server.URL + "/page_" + fmt.Sprint(page+1)
			w.Write([]byte(fmt.Sprintf(
				`{"data":[{"person_id":%d,"org_id":1,"role":"Membro","person_name":"Persona %d"}],"next":%q}`,
				page, page, nextURL,
			)))
		} else {
			w.Write([]byte(`{"data":[],"next":null}`))
		}
	}))
	defer server.Close()

	db := setupTestDuckDB(t)
	defer db.Close()
	wm := &opdmMockWatermark{}

	err := RunOPDM(context.Background(), server.URL, db, wm, "test-key", t.TempDir())
	require.NoError(t, err)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM opdm_memberships").Scan(&count)
	assert.Equal(t, 2, count)
	assert.Equal(t, "opdm", wm.lastSourceName)
	assert.Contains(t, wm.lastCursor, "/page_2")
}

func TestOPDMSinglePage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[{"person_id":1,"org_id":1,"role":"Membro","person_name":"Mario Rossi"}],"next":null}`))
	}))
	defer server.Close()

	db := setupTestDuckDB(t)
	defer db.Close()
	wm := &opdmMockWatermark{}

	err := RunOPDM(context.Background(), server.URL, db, wm, "test-key", t.TempDir())
	require.NoError(t, err)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM opdm_memberships").Scan(&count)
	assert.Equal(t, 1, count)
}

func TestOPDMEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[],"next":null}`))
	}))
	defer server.Close()

	db := setupTestDuckDB(t)
	defer db.Close()
	wm := &opdmMockWatermark{}

	err := RunOPDM(context.Background(), server.URL, db, wm, "test-key", t.TempDir())
	require.NoError(t, err)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM opdm_memberships").Scan(&count)
	assert.Equal(t, 0, count)
}

func TestOPDMHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid api key"}`))
	}))
	defer server.Close()

	db := setupTestDuckDB(t)
	wm := &opdmMockWatermark{}

	err := RunOPDM(context.Background(), server.URL, db, wm, "wrong-key", t.TempDir())
	assert.ErrorContains(t, err, "401")
}

func TestOPDMRateLimiterEnforcement(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			w.Write([]byte(fmt.Sprintf(
				`{"data":[{"person_id":%d,"org_id":1,"role":"Membro","person_name":"P%d"}],"next":"page_%d"}`,
				callCount, callCount, callCount+1,
			)))
		} else {
			w.Write([]byte(`{"data":[],"next":null}`))
		}
	}))
	defer server.Close()

	rl := newOPDMRateLimiter(5)
	start := time.Now()
	for i := 0; i < 3; i++ {
		rl.Wait()
		resp, err := http.Get(server.URL)
		require.NoError(t, err)
		resp.Body.Close()
	}
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 2*time.Second)
}

func TestOPDMInvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	db := setupTestDuckDB(t)
	wm := &opdmMockWatermark{}

	err := RunOPDM(context.Background(), server.URL, db, wm, "test-key", t.TempDir())
	assert.ErrorContains(t, err, "decode")
}

func TestOPDMMultipleFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[
			{"person_id":1,"org_id":10,"role":"Presidente","person_name":"Anna Bianchi","start_date":"2023-01-01","end_date":""},
			{"person_id":2,"org_id":10,"role":"Vicepresidente","person_name":"Luigi Verdi","start_date":"2022-06-15","end_date":"2024-12-31"}
		],"next":null}`))
	}))
	defer server.Close()

	db := setupTestDuckDB(t)
	defer db.Close()
	wm := &opdmMockWatermark{}

	err := RunOPDM(context.Background(), server.URL, db, wm, "test-key", t.TempDir())
	require.NoError(t, err)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM opdm_memberships").Scan(&count)
	assert.Equal(t, 2, count)

	var role, name string
	db.QueryRow("SELECT role, person_name FROM opdm_memberships WHERE person_id = 1").Scan(&role, &name)
	assert.Equal(t, "Presidente", role)
	assert.Equal(t, "Anna Bianchi", name)
}
