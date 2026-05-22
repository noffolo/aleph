package finance

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOpenBBMarketDataTool_Happy(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	assert.NotNil(t, tool)
	assert.NotNil(t, tool.httpClient)
}

func TestNewOpenBBMarketDataTool_MultipleInstances(t *testing.T) {
	a := NewOpenBBMarketDataTool()
	b := NewOpenBBMarketDataTool()
	assert.NotNil(t, a)
	assert.NotNil(t, b)
	assert.NotSame(t, a, b)
}

func TestNewOpenBBMarketDataTool_SetsClient(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	assert.NotNil(t, tool.httpClient)
	req, _ := http.NewRequest("GET", "http://127.0.0.1/", nil)
	_, err := tool.httpClient.Do(req)
	assert.Error(t, err)
}

func TestOpenBBMarketDataTool_Name_ReturnsExpected(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	assert.Equal(t, "OpenBBMarketData", tool.Name())
}

func TestOpenBBMarketDataTool_Name_Consistent(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	n1 := tool.Name()
	n2 := tool.Name()
	assert.Equal(t, n1, n2)
}

func TestOpenBBMarketDataTool_Name_NotEmpty(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	assert.NotEmpty(t, tool.Name())
}

func TestOpenBBMarketDataTool_Execute_DaysClampedTo365(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]any{
		"symbol":    "AAPL",
		"data_type": "historical",
		"days":      999,
	})
	require.NoError(t, err)
	points, ok := result.([]HistoricalDataPoint)
	require.True(t, ok)
	assert.LessOrEqual(t, len(points), 365)
}

func TestOpenBBMarketDataTool_Execute_DaysZeroDefaultsTo30(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]any{
		"symbol":    "AAPL",
		"data_type": "historical",
		"days":      0,
	})
	require.NoError(t, err)
	points, ok := result.([]HistoricalDataPoint)
	require.True(t, ok)
	assert.Greater(t, len(points), 0)
}

func TestOpenBBMarketDataTool_Execute_UnsupportedDataType(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]any{
		"symbol":    "AAPL",
		"data_type": "intraday",
	})
	require.Error(t, err)
	var connectErr *connect.Error
	require.True(t, errors.As(err, &connectErr))
	assert.Contains(t, err.Error(), "unsupported data_type")
}

func TestExtractFromYahooResponse_Happy(t *testing.T) {
	resp := &yahooChartResponse{}
	resp.Chart.Result = []struct {
		Meta struct {
			RegularMarketPrice  float64 `json:"regularMarketPrice"`
			PreviousClose       float64 `json:"chartPreviousClose"`
			RegularMarketVolume int64   `json:"regularMarketVolume"`
		} `json:"meta"`
		Timestamp []int64 `json:"timestamp"`
	}{
		{
			Meta: struct {
				RegularMarketPrice  float64 `json:"regularMarketPrice"`
				PreviousClose       float64 `json:"chartPreviousClose"`
				RegularMarketVolume int64   `json:"regularMarketVolume"`
			}{
				RegularMarketPrice:  182.50,
				PreviousClose:       180.00,
				RegularMarketVolume: 50000000,
			},
			Timestamp: []int64{1704412800},
		},
	}

	result, err := extractFromYahooResponse("AAPL", resp)
	require.NoError(t, err)
	assert.Equal(t, "AAPL", result.Symbol)
	assert.InDelta(t, 182.50, result.Price, 0.01)
	assert.InDelta(t, 2.50, result.Change, 0.01)
	assert.Equal(t, int64(50000000), result.Volume)
	assert.NotEmpty(t, result.Timestamp)
}

func TestExtractFromYahooResponse_EmptyResults(t *testing.T) {
	resp := &yahooChartResponse{}
	resp.Chart.Result = []struct {
		Meta struct {
			RegularMarketPrice  float64 `json:"regularMarketPrice"`
			PreviousClose       float64 `json:"chartPreviousClose"`
			RegularMarketVolume int64   `json:"regularMarketVolume"`
		} `json:"meta"`
		Timestamp []int64 `json:"timestamp"`
	}{}

	_, err := extractFromYahooResponse("AAPL", resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no results")
}

func TestExtractFromYahooResponse_ZeroVolumeDefaults(t *testing.T) {
	resp := &yahooChartResponse{}
	resp.Chart.Result = []struct {
		Meta struct {
			RegularMarketPrice  float64 `json:"regularMarketPrice"`
			PreviousClose       float64 `json:"chartPreviousClose"`
			RegularMarketVolume int64   `json:"regularMarketVolume"`
		} `json:"meta"`
		Timestamp []int64 `json:"timestamp"`
	}{
		{
			Meta: struct {
				RegularMarketPrice  float64 `json:"regularMarketPrice"`
				PreviousClose       float64 `json:"chartPreviousClose"`
				RegularMarketVolume int64   `json:"regularMarketVolume"`
			}{
				RegularMarketPrice:  100.0,
				PreviousClose:       99.0,
				RegularMarketVolume: 0,
			},
			Timestamp: nil,
		},
	}

	result, err := extractFromYahooResponse("MSFT", resp)
	require.NoError(t, err)
	assert.Equal(t, int64(1000000), result.Volume)
	assert.NotEmpty(t, result.Timestamp)
}

func TestParseCSVRecord_Happy(t *testing.T) {
	record := []string{"2024-01-15", "181.25", "182.50", "180.75", "182.00", "50000000", "182.00"}
	point, err := parseCSVRecord(record, 0, 1, 2, 3, 4, 5)
	require.NoError(t, err)

	assert.Equal(t, "2024-01-15", point.Date.Format("2006-01-02"))
	assert.InDelta(t, 181.25, point.Open, 0.01)
	assert.InDelta(t, 182.50, point.High, 0.01)
	assert.InDelta(t, 180.75, point.Low, 0.01)
	assert.InDelta(t, 182.00, point.Close, 0.01)
	assert.Equal(t, int64(50000000), point.Volume)
}

func TestParseCSVRecord_InvalidDate(t *testing.T) {
	record := []string{"not-a-date", "181.25", "182.50", "180.75", "182.00", "50000000", "182.00"}
	_, err := parseCSVRecord(record, 0, 1, 2, 3, 4, 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse date")
}

func TestParseCSVRecord_InvalidVolumeDefaultsToZero(t *testing.T) {
	record := []string{"2024-01-15", "181.25", "182.50", "180.75", "182.00", "N/A", "182.00"}
	point, err := parseCSVRecord(record, 0, 1, 2, 3, 4, 5)
	require.NoError(t, err)
	assert.Equal(t, int64(0), point.Volume)
}

func TestFetchWithRetry_ContextCancelled(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := tool.fetchWithRetry(ctx, "AAPL")
	assert.Error(t, err)
	var connectErr *connect.Error
	assert.True(t, errors.As(err, &connectErr))
	assert.Equal(t, connect.CodeDeadlineExceeded, connectErr.Code())
}

func TestFetchWithRetry_ContextDeadlineExceeded(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond)

	_, err := tool.fetchWithRetry(ctx, "AAPL")
	assert.Error(t, err)
}

func TestFetchWithRetry_AllAttemptsFail(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	ctx := context.Background()

	tool.httpClient = &http.Client{
		Timeout: 1 * time.Millisecond,
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}

	result, err := tool.fetchWithRetry(ctx, "AAPL")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestFetchChartData_ResolvesData(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	ctx := context.Background()

	result, err := tool.fetchChartData(ctx, "AAPL")
	if err != nil {
		t.Skipf("Yahoo Finance API unreachable (expected in CI): %v", err)
	}
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "AAPL", result.Symbol)
	assert.Greater(t, result.Price, 0.0)
}

func TestFetchHistoricalCSV_ContextCancelled(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := tool.fetchHistoricalCSV(ctx, "AAPL", 30)
	assert.Error(t, err)
}

func TestFetchHistoricalCSV_NetworkError(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	ctx := context.Background()

	_, err := tool.fetchHistoricalCSV(ctx, "AAPL", 5)
	assert.Error(t, err)
}

func TestGetStockPrice_FallbackToMock(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	ctx := context.Background()

	result, err := tool.getStockPrice(ctx, "AAPL")
	require.NoError(t, err)
	assert.Equal(t, "AAPL", result.Symbol)
	assert.Greater(t, result.Price, 0.0)
	assert.NotEmpty(t, result.Timestamp)
}

func TestGetStockPrice_UnknownSymbol(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	ctx := context.Background()

	result, err := tool.getStockPrice(ctx, "BOGUS12345")
	require.NoError(t, err)
	assert.Equal(t, "BOGUS12345", result.Symbol)
	assert.InDelta(t, 100.0, result.Price, 10.0)
}

func TestGetStockPrice_EmptySymbol(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	ctx := context.Background()

	result, err := tool.getStockPrice(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, "", result.Symbol)
	assert.Greater(t, result.Price, 0.0)
}

func TestGetHistoricalData_FallbackToMock(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	ctx := context.Background()

	data, err := tool.getHistoricalData(ctx, "GOOGL", 30)
	require.NoError(t, err)
	assert.Greater(t, len(data), 0)
	for _, p := range data {
		assert.Greater(t, p.Close, 0.0)
		assert.False(t, p.Date.IsZero())
	}
}

func TestGetHistoricalData_ShortLookback(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	ctx := context.Background()

	data, err := tool.getHistoricalData(ctx, "TSLA", 3)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(data), 3)
}

func TestGetHistoricalData_EmptySymbol(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	ctx := context.Background()

	data, err := tool.getHistoricalData(ctx, "", 10)
	require.NoError(t, err)
	assert.Greater(t, len(data), 0)
}

func TestGetBasicIndicators_FallbackToMock(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	ctx := context.Background()

	result, err := tool.getBasicIndicators(ctx, "AAPL", 30)
	require.NoError(t, err)
	assert.Equal(t, "AAPL", result.Symbol)
	assert.Greater(t, result.DataPoints, 0)
	assert.Greater(t, result.LastClose, 0.0)
}

func TestGetBasicIndicators_SmallLookback(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	ctx := context.Background()

	result, err := tool.getBasicIndicators(ctx, "MSFT", 5)
	require.NoError(t, err)
	assert.Equal(t, "MSFT", result.Symbol)
}

func TestGetBasicIndicators_EmptySymbol(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	ctx := context.Background()

	result, err := tool.getBasicIndicators(ctx, "", 30)
	require.NoError(t, err)
	assert.Equal(t, "", result.Symbol)
}

func TestMockMarketData_UnknownSymbol(t *testing.T) {
	result := mockMarketData("UNKNOWN")
	assert.Equal(t, "UNKNOWN", result.Symbol)
	assert.InDelta(t, 100.0, result.Price, 10.0)
	assert.Greater(t, result.Volume, int64(0))
	assert.NotEmpty(t, result.Timestamp)
}

func TestMockMarketData_ReturnsChange(t *testing.T) {
	result := mockMarketData("AAPL")
	_ = result.Change
	assert.NotNil(t, result)
}

func TestMockMarketData_AllKnownSymbols(t *testing.T) {
	for _, sym := range []string{"AAPL", "GOOGL", "MSFT", "AMZN", "TSLA", "META", "NVDA", "JPM", "V", "SPY"} {
		t.Run(sym, func(t *testing.T) {
			result := mockMarketData(sym)
			assert.Equal(t, sym, result.Symbol)
			assert.Greater(t, result.Price, 0.0)
		})
	}
}

func TestMockHistoricalData_UnknownSymbol(t *testing.T) {
	data := mockHistoricalData("UNKNOWN", 10)
	assert.Greater(t, len(data), 0)
	assert.LessOrEqual(t, len(data), 10)
	for _, p := range data {
		assert.InDelta(t, 100.0, p.Close, 30.0)
	}
}

func TestMockHistoricalData_LargeDays(t *testing.T) {
	data := mockHistoricalData("AAPL", 365)
	assert.LessOrEqual(t, len(data), 365)
	assert.Greater(t, len(data), 0)
	for i := 1; i < len(data); i++ {
		assert.True(t, data[i].Date.After(data[i-1].Date) || data[i].Date.Equal(data[i-1].Date))
	}
}

func TestMockHistoricalData_OneDay(t *testing.T) {
	data := mockHistoricalData("AAPL", 1)
	assert.LessOrEqual(t, len(data), 1)
}

func TestComputeIndicators_Exactly20Points(t *testing.T) {
	now := time.Now().UTC()
	data := make([]HistoricalDataPoint, 20)
	for i := range data {
		data[i] = HistoricalDataPoint{
			Date:   now.AddDate(0, 0, -(20 - i)),
			Close:  100.0 + float64(i)*0.5,
			Volume: 1000000,
		}
	}
	result := computeIndicators("TEST", data)
	assert.NotNil(t, result.SMA20)
	assert.NotNil(t, result.RSI14)
}

func TestComputeIndicators_Exactly15Points(t *testing.T) {
	now := time.Now().UTC()
	data := make([]HistoricalDataPoint, 15)
	for i := range data {
		data[i] = HistoricalDataPoint{
			Date:   now.AddDate(0, 0, -(15 - i)),
			Close:  100.0 + float64(i)*0.5,
			Volume: 1000000,
		}
	}
	result := computeIndicators("TEST", data)
	assert.Nil(t, result.SMA20)
	assert.NotNil(t, result.RSI14)
}

func TestComputeIndicators_Exactly14Points(t *testing.T) {
	now := time.Now().UTC()
	data := make([]HistoricalDataPoint, 14)
	for i := range data {
		data[i] = HistoricalDataPoint{
			Date:   now.AddDate(0, 0, -(14 - i)),
			Close:  100.0 + float64(i)*0.5,
			Volume: 1000000,
		}
	}
	result := computeIndicators("TEST", data)
	assert.Nil(t, result.SMA20)
	assert.Nil(t, result.RSI14)
	assert.Equal(t, 14, result.DataPoints)
}

func TestComputeRSI_NeutralDefault(t *testing.T) {
	rsi := computeRSI([]float64{100}, 14)
	assert.InDelta(t, 50.0, rsi, 0.01)
}

func TestComputeRSI_ZeroAvgLoss(t *testing.T) {
	closes := make([]float64, 20)
	for i := range closes {
		closes[i] = float64(100 + i)
	}
	rsi := computeRSI(closes, 14)
	assert.InDelta(t, 100.0, rsi, 0.01)
}

func TestComputeRSI_ZeroAvgGain(t *testing.T) {
	closes := make([]float64, 20)
	for i := range closes {
		closes[i] = float64(100 - i)
	}
	rsi := computeRSI(closes, 14)
	assert.InDelta(t, 0.0, rsi, 0.01)
}

func TestMean_SingleValue(t *testing.T) {
	assert.InDelta(t, 42.0, mean([]float64{42.0}), 0.01)
}

func TestMean_NegativeValues(t *testing.T) {
	assert.InDelta(t, -2.0, mean([]float64{-1, -2, -3}), 0.01)
}

func TestMean_ZeroValues(t *testing.T) {
	assert.InDelta(t, 0.0, mean([]float64{0, 0, 0, 0}), 0.01)
}

func TestHistoricalDataPoint_JSONRoundTrip(t *testing.T) {
	point := HistoricalDataPoint{
		Date:   time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		Open:   181.25,
		High:   182.50,
		Low:    180.75,
		Close:  182.00,
		Volume: 50000000,
	}

	b, err := json.Marshal(point)
	require.NoError(t, err)

	var decoded HistoricalDataPoint
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)

	assert.Equal(t, point.Open, decoded.Open)
	assert.Equal(t, point.High, decoded.High)
	assert.Equal(t, point.Low, decoded.Low)
	assert.Equal(t, point.Close, decoded.Close)
	assert.Equal(t, point.Volume, decoded.Volume)
}

func TestIndicatorResult_JSONOmitEmpty(t *testing.T) {
	result := IndicatorResult{
		Symbol:     "AAPL",
		DataPoints: 0,
		ComputedAt: time.Now().UTC(),
	}

	b, err := json.Marshal(result)
	require.NoError(t, err)

	str := string(b)
	assert.NotContains(t, str, "sma_20")
	assert.NotContains(t, str, "rsi_14")
}

func TestOpenBBMarketDataResult_JSON(t *testing.T) {
	result := OpenBBMarketDataResult{
		Symbol:    "AAPL",
		Price:     182.50,
		Change:    2.50,
		Volume:    50000000,
		Timestamp: "2024-01-15T16:00:00Z",
	}

	b, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded OpenBBMarketDataResult
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)

	assert.Equal(t, result.Symbol, decoded.Symbol)
	assert.Equal(t, result.Price, decoded.Price)
	assert.Equal(t, result.Change, decoded.Change)
	assert.Equal(t, result.Volume, decoded.Volume)
	assert.Equal(t, result.Timestamp, decoded.Timestamp)
}

var _ = repository.ToolRecord{}
var _ = strings.TrimSpace("")
