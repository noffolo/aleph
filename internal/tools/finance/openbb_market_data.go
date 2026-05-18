package finance

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand" // #nosec G404 — safe: time-seeded PRNG for mock market data fallback, not security-sensitive
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/ssrf"
)

const (
	maxMarketDataRetries = 3
)

// HistoricalDataPoint represents a single day of historical market data.
type HistoricalDataPoint struct {
	Date   time.Time `json:"date"`
	Open   float64   `json:"open"`
	High   float64   `json:"high"`
	Low    float64   `json:"low"`
	Close  float64   `json:"close"`
	Volume int64     `json:"volume"`
}

// IndicatorResult holds computed technical indicators for a symbol.
type IndicatorResult struct {
	Symbol     string    `json:"symbol"`
	SMA20      *float64  `json:"sma_20,omitempty"`
	RSI14      *float64  `json:"rsi_14,omitempty"`
	LastClose  float64   `json:"last_close"`
	DataPoints int       `json:"data_points"`
	ComputedAt time.Time `json:"computed_at"`
}

// OpenBBMarketDataArgs represents the input arguments for market data.
type OpenBBMarketDataArgs struct {
	Symbol   string `json:"symbol"`
	DataType string `json:"data_type"`
	Days     int    `json:"days,omitempty"`
}

// OpenBBMarketDataResult represents the market data output for a single price snapshot.
type OpenBBMarketDataResult struct {
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Change    float64 `json:"change"`
	Volume    int64   `json:"volume"`
	Timestamp string  `json:"timestamp"`
}

// OpenBBMarketDataTool provides market data via Yahoo Finance free API.
type OpenBBMarketDataTool struct {
	httpClient *http.Client
}

func (t *OpenBBMarketDataTool) Name() string { return "OpenBBMarketData" }

func NewOpenBBMarketDataTool() *OpenBBMarketDataTool {
	return &OpenBBMarketDataTool{
		httpClient: ssrf.NewClient(),
	}
}

// Execute fetches market data. Args:
//   - symbol: string — ticker symbol (e.g., "AAPL")
//   - data_type: string — "price" (default), "historical", "indicators"
//   - days: int — lookback days for historical/indicators (default 30)
//
// Uses Yahoo Finance free APIs: chart endpoint for price, CSV download for historical data.
func (t *OpenBBMarketDataTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	var mArgs OpenBBMarketDataArgs
	if err := parseArgs(args, &mArgs); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("invalid market_data args: %w", err))
	}
	if mArgs.Symbol == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("market_data requires a non-empty symbol"))
	}
	if mArgs.DataType == "" {
		mArgs.DataType = "price"
	}
	if mArgs.Days <= 0 {
		mArgs.Days = 30
	}
	if mArgs.Days > 365 {
		mArgs.Days = 365
	}

	switch mArgs.DataType {
	case "price":
		return t.getStockPrice(ctx, mArgs.Symbol)
	case "historical":
		return t.getHistoricalData(ctx, mArgs.Symbol, mArgs.Days)
	case "indicators":
		return t.getBasicIndicators(ctx, mArgs.Symbol, mArgs.Days)
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("unsupported data_type %q: use price, historical, or indicators", mArgs.DataType))
	}
}

// getStockPrice fetches the current price for a symbol via the Yahoo Finance chart endpoint.
func (t *OpenBBMarketDataTool) getStockPrice(ctx context.Context, symbol string) (*OpenBBMarketDataResult, error) {
	result, err := t.fetchWithRetry(ctx, symbol)
	if err == nil {
		return result, nil
	}
	return mockMarketData(symbol), nil
}

// getHistoricalData fetches historical daily OHLCV data via the Yahoo Finance CSV download endpoint.
func (t *OpenBBMarketDataTool) getHistoricalData(ctx context.Context, symbol string, days int) ([]HistoricalDataPoint, error) {
	data, err := t.fetchHistoricalCSV(ctx, symbol, days)
	if err == nil && len(data) > 0 {
		return data, nil
	}
	return mockHistoricalData(symbol, days), nil
}

// getBasicIndicators computes SMA(20) and RSI(14) from historical data.
func (t *OpenBBMarketDataTool) getBasicIndicators(ctx context.Context, symbol string, days int) (*IndicatorResult, error) {
	data, err := t.fetchHistoricalCSV(ctx, symbol, days)
	if err != nil || len(data) == 0 {
		data = mockHistoricalData(symbol, days)
	}

	return computeIndicators(symbol, data), nil
}

// fetchWithRetry attempts the chart endpoint with exponential backoff.
func (t *OpenBBMarketDataTool) fetchWithRetry(ctx context.Context, symbol string) (*OpenBBMarketDataResult, error) {
	var lastErr error
	for attempt := 0; attempt < maxMarketDataRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, connect.NewError(connect.CodeDeadlineExceeded,
					fmt.Errorf("market data request cancelled: %w", ctx.Err()))
			case <-time.After(time.Duration(math.Pow(2, float64(attempt))) * 100 * time.Millisecond):
			}
		}

		result, err := t.fetchChartData(ctx, symbol)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// fetchChartData calls the Yahoo Finance v8 chart endpoint for current price data.
func (t *OpenBBMarketDataTool) fetchChartData(ctx context.Context, symbol string) (*OpenBBMarketDataResult, error) {
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s", symbol)

	if err := ssrf.ValidateURL(url); err != nil {
		return nil, fmt.Errorf("SSRF validation failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chart API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var yahooResp yahooChartResponse
	if err := json.Unmarshal(body, &yahooResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return extractFromYahooResponse(symbol, &yahooResp)
}

// fetchHistoricalCSV downloads historical data via the Yahoo Finance CSV download endpoint.
func (t *OpenBBMarketDataTool) fetchHistoricalCSV(ctx context.Context, symbol string, days int) ([]HistoricalDataPoint, error) {
	now := time.Now()
	end := now.Unix()
	start := now.AddDate(0, 0, -days).Unix()

	url := fmt.Sprintf(
		"https://query1.finance.yahoo.com/v7/finance/download/%s?period1=%d&period2=%d&interval=1d&events=history",
		symbol, start, end,
	)

	if err := ssrf.ValidateURL(url); err != nil {
		return nil, fmt.Errorf("SSRF validation failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download API returned status %d", resp.StatusCode)
	}

	return parseYahooCSV(resp.Body)
}

// yahooChartResponse maps the Yahoo Finance v8 chart JSON response.
type yahooChartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				RegularMarketPrice  float64 `json:"regularMarketPrice"`
				PreviousClose       float64 `json:"chartPreviousClose"`
				RegularMarketVolume int64   `json:"regularMarketVolume"`
			} `json:"meta"`
			Timestamp []int64 `json:"timestamp"`
		} `json:"result"`
	} `json:"chart"`
}

func extractFromYahooResponse(symbol string, resp *yahooChartResponse) (*OpenBBMarketDataResult, error) {
	if len(resp.Chart.Result) == 0 {
		return nil, fmt.Errorf("no results in Yahoo Finance response")
	}

	r := resp.Chart.Result[0]
	price := r.Meta.RegularMarketPrice
	volume := r.Meta.RegularMarketVolume
	if volume == 0 {
		volume = 1000000
	}

	change := math.Round((price-r.Meta.PreviousClose)*100) / 100

	ts := time.Now().UTC().Format(time.RFC3339)
	if len(r.Timestamp) > 0 {
		ts = time.Unix(r.Timestamp[len(r.Timestamp)-1], 0).UTC().Format(time.RFC3339)
	}

	return &OpenBBMarketDataResult{
		Symbol:    symbol,
		Price:     price,
		Change:    change,
		Volume:    volume,
		Timestamp: ts,
	}, nil
}

// parseYahooCSV parses the CSV output from the Yahoo Finance download endpoint.
// Expected header: Date,Open,High,Low,Close,Volume,Adj Close
func parseYahooCSV(r io.Reader) ([]HistoricalDataPoint, error) {
	reader := csv.NewReader(r)

	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read CSV header: %w", err)
	}

	// Find column indices by header name
	colMap := make(map[string]int, len(header))
	for i, name := range header {
		colMap[strings.TrimSpace(name)] = i
	}

	getIdx := func(name string) (int, error) {
		i, ok := colMap[name]
		if !ok {
			return 0, fmt.Errorf("missing column %q in CSV", name)
		}
		return i, nil
	}

	dateIdx, err := getIdx("Date")
	if err != nil {
		return nil, err
	}
	openIdx, err := getIdx("Open")
	if err != nil {
		return nil, err
	}
	highIdx, err := getIdx("High")
	if err != nil {
		return nil, err
	}
	lowIdx, err := getIdx("Low")
	if err != nil {
		return nil, err
	}
	closeIdx, err := getIdx("Close")
	if err != nil {
		return nil, err
	}
	volumeIdx, err := getIdx("Volume")
	if err != nil {
		return nil, err
	}

	var points []HistoricalDataPoint
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read CSV record: %w", err)
		}

		// Skip rows where Close is "null" (dividends/splits)
		if record[closeIdx] == "null" {
			continue
		}

		point, err := parseCSVRecord(record, dateIdx, openIdx, highIdx, lowIdx, closeIdx, volumeIdx)
		if err != nil {
			continue // skip malformed rows
		}
		points = append(points, point)
	}

	if len(points) == 0 {
		return nil, fmt.Errorf("no valid data points in CSV")
	}

	// Sort by date ascending
	sort.Slice(points, func(i, j int) bool {
		return points[i].Date.Before(points[j].Date)
	})

	return points, nil
}

func parseCSVRecord(record []string, dateIdx, openIdx, highIdx, lowIdx, closeIdx, volumeIdx int) (HistoricalDataPoint, error) {
	date, err := time.Parse("2006-01-02", strings.TrimSpace(record[dateIdx]))
	if err != nil {
		return HistoricalDataPoint{}, fmt.Errorf("parse date %q: %w", record[dateIdx], err)
	}

	open, err := strconv.ParseFloat(strings.TrimSpace(record[openIdx]), 64)
	if err != nil {
		return HistoricalDataPoint{}, fmt.Errorf("parse open %q: %w", record[openIdx], err)
	}

	high, err := strconv.ParseFloat(strings.TrimSpace(record[highIdx]), 64)
	if err != nil {
		return HistoricalDataPoint{}, fmt.Errorf("parse high %q: %w", record[highIdx], err)
	}

	low, err := strconv.ParseFloat(strings.TrimSpace(record[lowIdx]), 64)
	if err != nil {
		return HistoricalDataPoint{}, fmt.Errorf("parse low %q: %w", record[lowIdx], err)
	}

	closeVal, err := strconv.ParseFloat(strings.TrimSpace(record[closeIdx]), 64)
	if err != nil {
		return HistoricalDataPoint{}, fmt.Errorf("parse close %q: %w", record[closeIdx], err)
	}

	volume, err := strconv.ParseInt(strings.TrimSpace(record[volumeIdx]), 10, 64)
	if err != nil {
		volume = 0
	}

	return HistoricalDataPoint{
		Date:   date,
		Open:   open,
		High:   high,
		Low:    low,
		Close:  closeVal,
		Volume: volume,
	}, nil
}

// computeIndicators computes SMA(20) and RSI(14) from historical price data.
func computeIndicators(symbol string, data []HistoricalDataPoint) *IndicatorResult {
	if len(data) == 0 {
		return &IndicatorResult{
			Symbol:     symbol,
			DataPoints: 0,
			ComputedAt: time.Now().UTC(),
		}
	}

	// Extract closing prices in chronological order
	closes := make([]float64, len(data))
	for i, p := range data {
		closes[i] = p.Close
	}

	result := &IndicatorResult{
		Symbol:     symbol,
		LastClose:  closes[len(closes)-1],
		DataPoints: len(closes),
		ComputedAt: time.Now().UTC(),
	}

	// SMA(20)
	if len(closes) >= 20 {
		var sum float64
		for i := len(closes) - 20; i < len(closes); i++ {
			sum += closes[i]
		}
		sma := math.Round(sum/20.0*100) / 100
		result.SMA20 = &sma
	}

	// RSI(14)
	if len(closes) >= 15 {
		rsi := computeRSI(closes, 14)
		result.RSI14 = &rsi
	}

	return result
}

// computeRSI calculates the Relative Strength Index for a given period.
// RSI = 100 - (100 / (1 + RS)) where RS = average gain / average loss.
func computeRSI(closes []float64, period int) float64 {
	if len(closes) < period+1 {
		return 50.0 // neutral default
	}

	// Compute price changes
	gains := make([]float64, 0, len(closes)-1)
	losses := make([]float64, 0, len(closes)-1)
	for i := 1; i < len(closes); i++ {
		change := closes[i] - closes[i-1]
		if change >= 0 {
			gains = append(gains, change)
			losses = append(losses, 0)
		} else {
			gains = append(gains, 0)
			losses = append(losses, -change)
		}
	}

	// Use only the last `period` changes for calculation
	if len(gains) > period {
		gains = gains[len(gains)-period:]
		losses = losses[len(losses)-period:]
	}

	avgGain := mean(gains)
	avgLoss := mean(losses)

	if avgLoss == 0 {
		if avgGain == 0 {
			return 50.0 // completely flat — neutral
		}
		return 100.0
	}
	rs := avgGain / avgLoss
	rsi := 100.0 - (100.0 / (1.0 + rs))
	return math.Round(rsi*100) / 100
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// mockMarketData returns structured mock data for when the API is unreachable.
func mockMarketData(symbol string) *OpenBBMarketDataResult {
	basePrices := map[string]float64{
		"AAPL": 175.50, "GOOGL": 141.80, "MSFT": 378.90,
		"AMZN": 178.25, "TSLA": 245.60, "META": 474.30,
		"NVDA": 820.10, "JPM": 183.45, "V": 275.20,
		"SPY": 478.30,
	}

	price, ok := basePrices[symbol]
	if !ok {
		price = 100.00
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	jitter := (rng.Float64() - 0.5) * price * 0.02
	price = math.Round((price+jitter)*100) / 100

	return &OpenBBMarketDataResult{
		Symbol:    symbol,
		Price:     price,
		Change:    math.Round((rng.Float64()-0.5)*price*0.03*100) / 100,
		Volume:    int64(rng.Int63n(50000000) + 1000000),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// mockHistoricalData generates synthetic historical data for testing/mock fallback.
func mockHistoricalData(symbol string, days int) []HistoricalDataPoint {
	basePrices := map[string]float64{
		"AAPL": 175.50, "GOOGL": 141.80, "MSFT": 378.90,
		"AMZN": 178.25, "TSLA": 245.60, "META": 474.30,
		"NVDA": 820.10, "JPM": 183.45, "V": 275.20,
		"SPY": 478.30,
	}

	basePrice, ok := basePrices[symbol]
	if !ok {
		basePrice = 100.00
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	now := time.Now().UTC()
	points := make([]HistoricalDataPoint, 0, days)

	currentPrice := basePrice
	for i := days - 1; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		// Skip weekends for realistic data
		if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
			continue
		}

		// Random walk
		change := (rng.Float64() - 0.48) * currentPrice * 0.025
		currentPrice += change
		if currentPrice < 1 {
			currentPrice = 1
		}

		open := currentPrice - change*0.5
		high := math.Max(open, currentPrice) + rng.Float64()*currentPrice*0.01
		low := math.Min(open, currentPrice) - rng.Float64()*currentPrice*0.01
		volume := int64(rng.Int63n(30000000) + 5000000)

		points = append(points, HistoricalDataPoint{
			Date:   date,
			Open:   math.Round(open*100) / 100,
			High:   math.Round(high*100) / 100,
			Low:    math.Round(low*100) / 100,
			Close:  math.Round(currentPrice*100) / 100,
			Volume: volume,
		})
	}

	return points
}

// Register registers the tool in the metadata repository.
func (t *OpenBBMarketDataTool) Register(metaRepo *repository.MetadataRepository) error {
	return metaRepo.CreateTool(&repository.ToolRecord{
		ID:           "finance_openbb_market_data",
		Name:         "finance_openbb_market_data",
		Description:  "Market data via Yahoo Finance free API (price, historical, indicators SMA20/RSI14) with mock fallback",
		Code:         "",
		Category:     "finance",
		Version:      "2.0.0",
		HealthStatus: "unknown",
		SourceType:   "package",
	})
}
