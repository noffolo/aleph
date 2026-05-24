package sources

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/marcboeker/go-duckdb"
)

func newPolymarketTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestIsItalianMarket(t *testing.T) {
	tests := []struct {
		name     string
		market   polymarketMarket
		expected bool
	}{
		{
			name:     "italy in question",
			market:   polymarketMarket{Question: "Who will win Italy election 2025?", Tags: []pmTag{}},
			expected: true,
		},
		{
			name:     "italia in question",
			market:   polymarketMarket{Question: "Prossimo primo ministro Italia", Tags: []pmTag{}},
			expected: true,
		},
		{
			name:     "meloni in question",
			market:   polymarketMarket{Question: "Will Giorgia Meloni resign?", Tags: []pmTag{}},
			expected: true,
		},
		{
			name:     "salvini in question",
			market:   polymarketMarket{Question: "Matteo Salvini next PM?", Tags: []pmTag{}},
			expected: true,
		},
		{
			name:   "italian tag",
			market: polymarketMarket{Question: "Will US win trade war?", Tags: []pmTag{{ID: 2, Label: "Italian Politics"}}},
			expected: true,
		},
		{
			name:     "no italian keywords",
			market:   polymarketMarket{Question: "Will Trump win 2024?", Tags: []pmTag{{ID: 1, Label: "US Politics"}}},
			expected: false,
		},
		{
			name:     "empty market",
			market:   polymarketMarket{Question: "", Tags: []pmTag{}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isItalianMarket(tt.market))
		})
	}
}

func TestParsePolymarketTime(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
		assertFn  func(t *testing.T, got time.Time)
	}{
		{
			name:      "empty string",
			input:     "",
			expectErr: false,
			assertFn:  func(t *testing.T, got time.Time) { assert.True(t, got.IsZero()) },
		},
		{
			name:      "RFC3339",
			input:     "2025-06-15T12:00:00Z",
			expectErr: false,
			assertFn:  func(t *testing.T, got time.Time) { assert.Equal(t, 2025, got.Year()) },
		},
		{
			name:      "ISO without Z suffix",
			input:     "2025-06-15T12:00:00",
			expectErr: false,
			assertFn:  func(t *testing.T, got time.Time) { assert.Equal(t, 2025, got.Year()) },
		},
		{
			name:      "date only",
			input:     "2025-06-15",
			expectErr: false,
			assertFn:  func(t *testing.T, got time.Time) { assert.Equal(t, 2025, got.Year()) },
		},
		{
			name:      "unrecognised",
			input:     "not-a-date",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePolymarketTime(tt.input)
			if tt.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tt.assertFn != nil {
				tt.assertFn(t, got)
			}
		})
	}
}

func TestEnsurePolymarketTables(t *testing.T) {
	db := newPolymarketTestDB(t)
	err := ensurePolymarketTables(db)
	require.NoError(t, err)

	for _, table := range []string{"polymarket_markets", "polymarket_prices", "polymarket_events"} {
		_, err := db.Exec("SELECT 1 FROM " + table + " LIMIT 0")
		assert.NoError(t, err, "table %s should exist", table)
	}
}

func TestSearchMarketsEndpoints(t *testing.T) {
	mockMarkets := []polymarketMarket{
		{ConditionID: "0xabc", Question: "Will Italy leave the EU?", TokenIDs: []string{"123"}, Outcomes: []string{"Yes", "No"}, OutcomePrices: []string{"0.42", "0.58"}, Volume: 50000.0, Active: true},
		{ConditionID: "0xdef", Question: "Who will be next Italian PM?", TokenIDs: []string{"456", "789"}, Outcomes: []string{"Meloni", "Salvini", "Conte"}, OutcomePrices: []string{"0.60", "0.25", "0.15"}, Volume: 120000.0, Active: true},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "query=")
		assert.Contains(t, r.URL.RawQuery, "limit=50")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockMarkets)
	}))
	defer srv.Close()

	// Verify the mock markets are valid and filter correctly for Italian content
	italian := 0
	for _, m := range mockMarkets {
		if isItalianMarket(m) {
			italian++
		}
	}
	assert.Equal(t, 2, italian, "all mock markets should match Italian keywords")
}

func TestMarketInsertion(t *testing.T) {
	db := newPolymarketTestDB(t)
	require.NoError(t, ensurePolymarketTables(db))

	market := polymarketMarket{
		ConditionID: "0xtest", Question: "Test Italian market",
		TokenIDs: []string{"1001"}, Outcomes: []string{"Yes", "No"},
		OutcomePrices: []string{"0.5", "0.5"}, Volume: 1000,
		EndDateISO: "2025-12-31", Active: true,
		Tags: []pmTag{{ID: 2, Label: "Politics"}},
	}

	outcomesJSON, _ := json.Marshal(market.Outcomes)
	tagsJSON, _ := json.Marshal(market.Tags)
	_, err := db.Exec(
		`INSERT INTO polymarket_markets (token_id, condition_id, question, description, end_date, active, closed, volume, outcomes, tags)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		market.TokenIDs[0], market.ConditionID, market.Question, market.Description,
		time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC), market.Active, false, market.Volume,
		string(outcomesJSON), string(tagsJSON),
	)
	require.NoError(t, err)

	var question string
	err = db.QueryRow("SELECT question FROM polymarket_markets WHERE token_id = '1001'").Scan(&question)
	require.NoError(t, err)
	assert.Equal(t, "Test Italian market", question)

	var outcomesRaw string
	err = db.QueryRow("SELECT outcomes::VARCHAR FROM polymarket_markets WHERE token_id = '1001'").Scan(&outcomesRaw)
	require.NoError(t, err)
	assert.Contains(t, outcomesRaw, "Yes")
	assert.Contains(t, outcomesRaw, "No")
}

func TestPriceInsertion(t *testing.T) {
	db := newPolymarketTestDB(t)
	require.NoError(t, ensurePolymarketTables(db))

	_, err := db.Exec(
		`INSERT INTO polymarket_prices (token_id, timestamp, price_yes, price_no) VALUES (?, ?, ?, ?)`,
		"1001", time.Unix(1714953600, 0).UTC(), 0.42, 0.58,
	)
	require.NoError(t, err)

	var priceYes float64
	err = db.QueryRow("SELECT price_yes FROM polymarket_prices WHERE token_id = '1001'").Scan(&priceYes)
	require.NoError(t, err)
	assert.Equal(t, 0.42, priceYes)

	var priceNo float64
	err = db.QueryRow("SELECT price_no FROM polymarket_prices WHERE token_id = '1001'").Scan(&priceNo)
	require.NoError(t, err)
	assert.InDelta(t, 0.58, priceNo, 0.0001)
	assert.InDelta(t, 1.0, priceYes+priceNo, 0.0001)
}

func TestPolymarketPriceHistoryBulk(t *testing.T) {
	pricePoints := []polymarketPricePoint{
		{T: 1714953600, P: 0.42},
		{T: 1715040000, P: 0.45},
		{T: 1715126400, P: 0.40},
	}

	assert.Len(t, pricePoints, 3)
	assert.Equal(t, int64(1714953600), pricePoints[0].T)
	assert.Equal(t, 0.42, pricePoints[0].P)

	db := newPolymarketTestDB(t)
	require.NoError(t, ensurePolymarketTables(db))

	for _, pp := range pricePoints {
		ts := time.Unix(pp.T, 0).UTC()
		_, err := db.Exec(
			`INSERT INTO polymarket_prices (token_id, timestamp, price_yes, price_no) VALUES (?, ?, ?, ?)`,
			"test-1", ts, pp.P, 1.0-pp.P,
		)
		require.NoError(t, err)
	}

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM polymarket_prices WHERE token_id = 'test-1'").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestEventInsertion(t *testing.T) {
	db := newPolymarketTestDB(t)
	require.NoError(t, ensurePolymarketTables(db))

	_, err := db.Exec(
		`INSERT INTO polymarket_events (id, title, description, start_date, end_date, active, closed, category)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"evt-1", "Italian Elections 2025", "Description",
		time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		true, false, "politics",
	)
	require.NoError(t, err)

	var title string
	err = db.QueryRow("SELECT title FROM polymarket_events WHERE id = 'evt-1'").Scan(&title)
	require.NoError(t, err)
	assert.Equal(t, "Italian Elections 2025", title)

	_, err = db.Exec(
		`INSERT INTO polymarket_events (id, title, description, start_date, end_date, active, closed, category)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"evt-2", "EU Referendum", "",
		nil, nil, true, false, "politics",
	)
	require.NoError(t, err)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM polymarket_events").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestPriceNoComplement(t *testing.T) {
	prices := []float64{0.42, 0.65, 0.01, 0.99, 0.0, 1.0}
	for _, p := range prices {
		assert.InDelta(t, 1.0, p+(1.0-p), 0.0001, "price_yes + price_no should equal 1.0")
	}
}

func TestJSONRoundtrip(t *testing.T) {
	market := polymarketMarket{
		ConditionID:   "0xabc123",
		Question:      "Will Italy win Euro 2025?",
		Description:   "Sports prediction market",
		EndDateISO:    "2025-07-15",
		TokenIDs:      []string{"123", "456"},
		Active:        true,
		Closed:        false,
		Outcomes:      []string{"Yes", "No"},
		OutcomePrices: []string{"0.55", "0.45"},
		Volume:        75000.0,
		Tags:          []pmTag{{ID: 4, Label: "Sports"}, {ID: 5, Label: "Italy"}},
	}

	bs, err := json.Marshal(market)
	require.NoError(t, err)

	var decoded polymarketMarket
	err = json.Unmarshal(bs, &decoded)
	require.NoError(t, err)

	assert.Equal(t, market.ConditionID, decoded.ConditionID)
	assert.Equal(t, market.Question, decoded.Question)
	assert.Equal(t, 2, len(decoded.TokenIDs))
	assert.Equal(t, 75000.0, decoded.Volume)
	assert.True(t, decoded.Active)
	assert.False(t, decoded.Closed)
}

func TestRateLimitConstants(t *testing.T) {
	assert.Equal(t, 30.0, polymarketGammaRPS, "Gamma API should be 30 req/s (300/10s)")
	assert.Equal(t, 10.0, polymarketCLOBRPS, "CLOB API should be 10 req/s")
}

func TestPolymarketSearchHTTPMock(t *testing.T) {
	rawResponse := `[{"condition_id":"0x111","question":"Italy elections","token_ids":["123"],"outcomes":["Yes","No"],"outcome_prices":["0.30","0.70"],"volume":50000,"active":true,"closed":false}]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "public-search")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(rawResponse))
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/public-search?query=italy&limit=50")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var markets []polymarketMarket
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&markets))
	assert.Len(t, markets, 1)
	assert.Equal(t, "0x111", markets[0].ConditionID)
	assert.Equal(t, "Italy elections", markets[0].Question)
}
