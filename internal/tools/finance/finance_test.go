package finance

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockNLPAnalyzer implements NLPAnalyzer for testing.
type mockNLPAnalyzer struct {
	score float32
	label string
	err   error
}

func (m *mockNLPAnalyzer) AnalyzeSentiment(ctx context.Context, text string) (float32, string, error) {
	return m.score, m.label, m.err
}

func TestProphetForecastTool(t *testing.T) {
	tool := NewProphetForecastTool()
	ctx := context.Background()

	tests := []struct {
		name    string
		args    map[string]any
		wantErr bool
		check   func(t *testing.T, result any)
	}{
		{
			name: "sma with 2 data points",
			args: map[string]any{
				"data":    []float64{100, 110},
				"periods": 3,
			},
			wantErr: false,
			check: func(t *testing.T, result any) {
				r, ok := result.(*ProphetForecastResult)
				require.True(t, ok, "result should be *ProphetForecastResult")
				assert.Equal(t, "sma", r.Method)
				assert.Len(t, r.Predictions, 3)
				for _, p := range r.Predictions {
					assert.Equal(t, 105.0, p, "SMA should be average of [100, 110]")
				}
				assert.GreaterOrEqual(t, r.Confidence, 0.0)
				assert.LessOrEqual(t, r.Confidence, 1.0)
			},
		},
		{
			name: "linear regression with 5 data points",
			args: map[string]any{
				"data":    []float64{10, 20, 30, 40, 50},
				"periods": 2,
			},
			wantErr: false,
			check: func(t *testing.T, result any) {
				r, ok := result.(*ProphetForecastResult)
				require.True(t, ok, "result should be *ProphetForecastResult")
				assert.Equal(t, "linear", r.Method)
				assert.Len(t, r.Predictions, 2)
				// Perfect linear: slope=10, intercept=10
				// y = 10 + 10*x, next points at x=5,6 -> 60, 70
				assert.InDelta(t, 60, r.Predictions[0], 0.01)
				assert.InDelta(t, 70, r.Predictions[1], 0.01)
				assert.InDelta(t, 1.0, r.Confidence, 0.01)
			},
		},
		{
			name: "default periods",
			args: map[string]any{
				"data": []float64{100, 110, 120},
			},
			wantErr: false,
			check: func(t *testing.T, result any) {
				r, ok := result.(*ProphetForecastResult)
				require.True(t, ok)
				assert.Len(t, r.Predictions, 1)
				assert.Equal(t, "sma", r.Method)
			},
		},
		{
			name: "error too few data points",
			args: map[string]any{
				"data":    []float64{100},
				"periods": 1,
			},
			wantErr: true,
		},
		{
			name: "error invalid args",
			args: map[string]any{
				"data": "not-an-array",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tool.Execute(ctx, tc.args)
			if tc.wantErr {
				assert.Error(t, err)
				var connectErr *connect.Error
				assert.True(t, errors.As(err, &connectErr))
				return
			}
			require.NoError(t, err)
			tc.check(t, result)
		})
	}
}

func TestOpenBBMarketDataTool(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	ctx := context.Background()

	tests := []struct {
		name    string
		args    map[string]any
		wantErr bool
		check   func(t *testing.T, result any)
	}{
		{
			name: "mock fallback for unreachable API",
			args: map[string]any{
				"symbol": "AAPL",
			},
			wantErr: false,
			check: func(t *testing.T, result any) {
				r, ok := result.(*OpenBBMarketDataResult)
				require.True(t, ok, "result should be *OpenBBMarketDataResult")
				assert.Equal(t, "AAPL", r.Symbol)
				assert.Greater(t, r.Price, 0.0)
				assert.Greater(t, r.Volume, int64(0))
				assert.NotEmpty(t, r.Timestamp)
			},
		},
		{
			name: "mock fallback for unknown symbol",
			args: map[string]any{
				"symbol": "ZZZZZ",
			},
			wantErr: false,
			check: func(t *testing.T, result any) {
				r, ok := result.(*OpenBBMarketDataResult)
				require.True(t, ok)
				assert.Equal(t, "ZZZZZ", r.Symbol)
				assert.Greater(t, r.Price, 0.0)
			},
		},
		{
			name: "error missing symbol",
			args: map[string]any{
				"symbol": "",
			},
			wantErr: true,
		},
		{
			name: "error invalid args type",
			args: map[string]any{
				"symbol": 123, // wrong type
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tool.Execute(ctx, tc.args)
			if tc.wantErr {
				assert.Error(t, err)
				var connectErr *connect.Error
				assert.True(t, errors.As(err, &connectErr))
				return
			}
			require.NoError(t, err)
			tc.check(t, result)
		})
	}
}

func TestSentimentAnalysisFinTool(t *testing.T) {
	ctx := context.Background()

	t.Run("synthetic fallback without NLPAdapter", func(t *testing.T) {
		tool := NewSentimentAnalysisFinTool()
		require.Nil(t, tool.nlpAdapter) // no adapter set

		tests := []struct {
			name  string
			args  map[string]any
			check func(t *testing.T, result any)
		}{
			{
				name: "positive sentiment",
				args: map[string]any{
					"text":   "The company reported strong revenue growth and bullish outlook",
					"source": "news",
				},
				check: func(t *testing.T, result any) {
					r, ok := result.(*SentimentResult)
					require.True(t, ok)
					assert.Equal(t, "positive", r.Sentiment)
					assert.Greater(t, r.Score, 0.5)
					assert.True(t, r.IsSynthetic)
					assert.Equal(t, "synthetic", r.Label)
				},
			},
			{
				name: "negative sentiment",
				args: map[string]any{
					"text":   "The company reported massive losses and bankruptcy risk",
					"source": "news",
				},
				check: func(t *testing.T, result any) {
					r, ok := result.(*SentimentResult)
					require.True(t, ok)
					assert.Equal(t, "negative", r.Sentiment)
					assert.Less(t, r.Score, 0.5)
					assert.True(t, r.IsSynthetic)
				},
			},
			{
				name: "neutral sentiment",
				args: map[string]any{
					"text":   "The company released its quarterly report today",
					"source": "filings",
				},
				check: func(t *testing.T, result any) {
					r, ok := result.(*SentimentResult)
					require.True(t, ok)
					assert.Equal(t, "neutral", r.Sentiment)
					assert.True(t, r.IsSynthetic)
				},
			},
			{
				name: "default source",
				args: map[string]any{
					"text": "Positive earnings beat expectations",
				},
				check: func(t *testing.T, result any) {
					r, ok := result.(*SentimentResult)
					require.True(t, ok)
					assert.Equal(t, "positive", r.Sentiment)
					assert.True(t, r.IsSynthetic)
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				result, err := tool.Execute(ctx, tc.args)
				require.NoError(t, err)
				tc.check(t, result)
			})
		}
	})

	t.Run("error empty text", func(t *testing.T) {
		tool := NewSentimentAnalysisFinTool()
		_, err := tool.Execute(ctx, map[string]any{
			"text": "  ",
		})
		assert.Error(t, err)
		var connectErr *connect.Error
		assert.True(t, errors.As(err, &connectErr))
	})

	t.Run("error invalid args", func(t *testing.T) {
		tool := NewSentimentAnalysisFinTool()
		_, err := tool.Execute(ctx, map[string]any{
			"text": 123,
		})
		assert.Error(t, err)
	})

	t.Run("with NLPAdapter", func(t *testing.T) {
		mockAdapter := &mockNLPAnalyzer{
			score: 0.8,
			label: "positive",
			err:   nil,
		}
		tool := NewSentimentAnalysisFinTool()
		tool.SetNLPAdapter(mockAdapter)

		result, err := tool.Execute(ctx, map[string]any{
			"text":   "Great company performance",
			"source": "news",
		})
		require.NoError(t, err)

		r, ok := result.(*SentimentResult)
		require.True(t, ok)
		assert.Equal(t, "positive", r.Sentiment)
		assert.InDelta(t, 0.8, r.Score, 0.01)
		assert.Equal(t, "positive", r.Label)
	})

	t.Run("with NLPAdapter error falls back to synthetic", func(t *testing.T) {
		mockAdapter := &mockNLPAnalyzer{
			err: errors.New("nlp service unavailable"),
		}
		tool := NewSentimentAnalysisFinTool()
		tool.SetNLPAdapter(mockAdapter)

		result, err := tool.Execute(ctx, map[string]any{
			"text":   "Profits are up significantly",
			"source": "earnings_calls",
		})
		require.NoError(t, err)

		r, ok := result.(*SentimentResult)
		require.True(t, ok)
		assert.True(t, r.IsSynthetic, "should fall back to synthetic when NLP fails")
		assert.Equal(t, "synthetic", r.Label)
	})
}

func TestParseArgs(t *testing.T) {
	t.Run("nil args", func(t *testing.T) {
		var target struct {
			Name string `json:"name"`
		}
		err := parseArgs(nil, &target)
		require.NoError(t, err)
		assert.Empty(t, target.Name)
	})

	t.Run("valid args", func(t *testing.T) {
		type testStruct struct {
			Name  string  `json:"name"`
			Value float64 `json:"value"`
		}
		var target testStruct
		err := parseArgs(map[string]any{
			"name":  "test",
			"value": 42.5,
		}, &target)
		require.NoError(t, err)
		assert.Equal(t, "test", target.Name)
		assert.Equal(t, 42.5, target.Value)
	})
}

func TestScoreToLabel(t *testing.T) {
	tests := []struct {
		score float64
		want  string
	}{
		{1.0, "positive"},
		{0.8, "positive"},
		{0.6, "positive"},
		{0.59, "neutral"},
		{0.5, "neutral"},
		{0.41, "neutral"},
		{0.4, "negative"},
		{0.2, "negative"},
		{0.0, "negative"},
	}

	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			got := scoreToLabel(tc.score)
			assert.Equal(t, tc.want, got, "scoreToLabel(%f)", tc.score)
		})
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{
			input: "hello world",
			want:  []string{"hello", "world"},
		},
		{
			input: "write-down, impairment!",
			want:  []string{"write-down", "impairment"},
		},
		{
			input: "",
			want:  []string{},
		},
	}

	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			got := tokenize(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestMockMarketData(t *testing.T) {
	tests := []struct {
		symbol string
		check  func(t *testing.T, result *OpenBBMarketDataResult)
	}{
		{
			symbol: "AAPL",
			check: func(t *testing.T, r *OpenBBMarketDataResult) {
				assert.Equal(t, "AAPL", r.Symbol)
				assert.Greater(t, r.Price, 0.0)
				assert.NotEmpty(t, r.Timestamp)
			},
		},
		{
			symbol: "UNKNOWN",
			check: func(t *testing.T, r *OpenBBMarketDataResult) {
				assert.Equal(t, "UNKNOWN", r.Symbol)
				assert.InDelta(t, 100.0, r.Price, 10.0)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.symbol, func(t *testing.T) {
			result := mockMarketData(tc.symbol)
			tc.check(t, result)
		})
	}
}

func TestLinearRegressionForecast(t *testing.T) {
	tests := []struct {
		name    string
		data    []float64
		periods int
		wantLen int
		check   func(t *testing.T, predictions []float64, confidence float64)
	}{
		{
			name:    "perfect linear",
			data:    []float64{1, 2, 3, 4},
			periods: 2,
			wantLen: 2,
			check: func(t *testing.T, predictions []float64, confidence float64) {
				assert.InDelta(t, 5, predictions[0], 0.01)
				assert.InDelta(t, 6, predictions[1], 0.01)
				assert.InDelta(t, 1.0, confidence, 0.01)
			},
		},
		{
			name:    "noisy data has lower confidence",
			data:    []float64{10, 200, 5, 300},
			periods: 1,
			wantLen: 1,
			check: func(t *testing.T, predictions []float64, confidence float64) {
				assert.Less(t, confidence, 1.0)
				assert.GreaterOrEqual(t, confidence, 0.0)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			predictions, confidence := linearRegressionForecast(tc.data, tc.periods)
			assert.Len(t, predictions, tc.wantLen)
			tc.check(t, predictions, confidence)
		})
	}
}

func TestSMAForecast(t *testing.T) {
	tests := []struct {
		name     string
		data     []float64
		periods  int
		expected float64
	}{
		{
			name:     "constant data",
			data:     []float64{50, 50, 50},
			periods:  2,
			expected: 50,
		},
		{
			name:     "trending data",
			data:     []float64{100, 110, 120, 130},
			periods:  1,
			expected: (100 + 110 + 120 + 130) / 4,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			predictions, confidence := smaForecast(tc.data, tc.periods)
			assert.Len(t, predictions, tc.periods)
			for _, p := range predictions {
				assert.InDelta(t, tc.expected, p, 0.01)
			}
			assert.GreaterOrEqual(t, confidence, 0.0)
			assert.LessOrEqual(t, confidence, 1.0)
		})
	}
}

func TestSetNLPAdapter(t *testing.T) {
	tool := NewSentimentAnalysisFinTool()
	assert.Nil(t, tool.nlpAdapter, "nlpAdapter should be nil initially")

	mock := &mockNLPAnalyzer{
		score: 0.9,
		label: "positive",
	}
	tool.SetNLPAdapter(mock)
	assert.NotNil(t, tool.nlpAdapter)
}

func TestMathRounding(t *testing.T) {
	// Verify math.Round works as expected in our context
	assert.Equal(t, 123.46, math.Round(123.456*100)/100)
	assert.Equal(t, 123.45, math.Round(123.454*100)/100)
}

func TestParseYahooCSV(t *testing.T) {
	t.Run("parses valid CSV correctly", func(t *testing.T) {
		csvData := `Date,Open,High,Low,Close,Volume,Adj Close
2024-01-05,181.25,182.50,180.75,182.00,50000000,182.00
2024-01-04,180.00,181.75,179.80,181.20,45000000,181.20
2024-01-03,179.50,180.50,178.90,179.80,48000000,179.80
2024-01-02,178.00,179.80,177.50,179.00,52000000,179.00`
		r := strings.NewReader(csvData)
		points, err := parseYahooCSV(r)
		require.NoError(t, err)
		require.Len(t, points, 4)

		// Should be sorted ascending by date
		assert.Equal(t, "2024-01-02", points[0].Date.Format("2006-01-02"))
		assert.Equal(t, "2024-01-05", points[3].Date.Format("2006-01-02"))

		// Verify last data point
		last := points[3]
		assert.InDelta(t, 182.00, last.Close, 0.01)
		assert.InDelta(t, 181.25, last.Open, 0.01)
		assert.InDelta(t, 182.50, last.High, 0.01)
		assert.InDelta(t, 180.75, last.Low, 0.01)
		assert.Equal(t, int64(50000000), last.Volume)
	})

	t.Run("skips null close rows (dividends)", func(t *testing.T) {
		csvData := `Date,Open,High,Low,Close,Volume,Adj Close
2024-01-05,181.25,182.50,180.75,182.00,50000000,182.00
2024-01-04,180.00,181.75,179.80,null,,null`
		r := strings.NewReader(csvData)
		points, err := parseYahooCSV(r)
		require.NoError(t, err)
		require.Len(t, points, 1)
	})

	t.Run("handles empty body", func(t *testing.T) {
		r := strings.NewReader("Date,Open,High,Low,Close,Volume,Adj Close\n")
		_, err := parseYahooCSV(r)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no valid data points")
	})

	t.Run("rejects missing columns", func(t *testing.T) {
		r := strings.NewReader("Date,Price\n2024-01-05,100")
		_, err := parseYahooCSV(r)
		require.Error(t, err)
	})
}

func TestComputeRSI(t *testing.T) {
	t.Run("rising prices give high RSI", func(t *testing.T) {
		closes := make([]float64, 20)
		for i := range closes {
			closes[i] = 100.0 + float64(i)*2 // steady uptrend
		}
		rsi := computeRSI(closes, 14)
		assert.Greater(t, rsi, 50.0, "RSI should be above 50 for uptrend")
		assert.LessOrEqual(t, rsi, 100.0)
	})

	t.Run("falling prices give low RSI", func(t *testing.T) {
		closes := make([]float64, 20)
		for i := range closes {
			closes[i] = 200.0 - float64(i)*3 // steady downtrend
		}
		rsi := computeRSI(closes, 14)
		assert.Less(t, rsi, 50.0, "RSI should be below 50 for downtrend")
		assert.GreaterOrEqual(t, rsi, 0.0)
	})

	t.Run("flat prices give neutral RSI", func(t *testing.T) {
		closes := make([]float64, 20)
		for i := range closes {
			closes[i] = 100.0
		}
		rsi := computeRSI(closes, 14)
		assert.InDelta(t, 50.0, rsi, 0.01)
	})

	t.Run("insufficient data returns neutral", func(t *testing.T) {
		rsi := computeRSI([]float64{100, 101}, 14)
		assert.InDelta(t, 50.0, rsi, 0.01)
	})

	t.Run("no losses gives RSI 100", func(t *testing.T) {
		closes := make([]float64, 20)
		for i := range closes {
			closes[i] = 100.0 + float64(i)*0.5
		}
		rsi := computeRSI(closes, 14)
		assert.InDelta(t, 100.0, rsi, 0.01)
	})
}

func TestComputeIndicators(t *testing.T) {
	t.Run("computes SMA20 and RSI14 from data", func(t *testing.T) {
		now := time.Now().UTC()
		data := make([]HistoricalDataPoint, 25)
		for i := range data {
			price := 100.0 + float64(i)*0.5
			data[i] = HistoricalDataPoint{
				Date:   now.AddDate(0, 0, -(25 - i)),
				Open:   price - 0.5,
				High:   price + 1.0,
				Low:    price - 1.0,
				Close:  price,
				Volume: 1000000,
			}
		}

		result := computeIndicators("AAPL", data)
		require.NotNil(t, result)
		assert.Equal(t, "AAPL", result.Symbol)
		assert.Equal(t, 25, result.DataPoints)
		assert.NotNil(t, result.SMA20)
		assert.NotNil(t, result.RSI14)
		assert.Greater(t, result.LastClose, 0.0)
	})

	t.Run("insufficient data for SMA", func(t *testing.T) {
		now := time.Now().UTC()
		data := make([]HistoricalDataPoint, 10)
		for i := range data {
			data[i] = HistoricalDataPoint{
				Date:   now.AddDate(0, 0, -(10 - i)),
				Close:  100.0,
				Volume: 1000000,
			}
		}

		result := computeIndicators("MSFT", data)
		require.NotNil(t, result)
		assert.Nil(t, result.SMA20)
		assert.Nil(t, result.RSI14)
		assert.Equal(t, 10, result.DataPoints)
	})

	t.Run("empty data returns zero-point result", func(t *testing.T) {
		result := computeIndicators("EMPTY", nil)
		assert.Equal(t, "EMPTY", result.Symbol)
		assert.Equal(t, 0, result.DataPoints)
	})
}

func TestOpenBBMarketDataTool_HistoricalDataType(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]any{
		"symbol":    "AAPL",
		"data_type": "historical",
		"days":      10,
	})
	require.NoError(t, err)

	// Should fall back to mock data since API is unreachable
	points, ok := result.([]HistoricalDataPoint)
	require.True(t, ok, "result should be []HistoricalDataPoint")
	require.Greater(t, len(points), 0, "should have data points from mock")
	for _, p := range points {
		assert.Greater(t, p.Close, 0.0)
		assert.Greater(t, p.Volume, int64(0))
		assert.False(t, p.Date.IsZero())
	}
}

func TestOpenBBMarketDataTool_IndicatorsDataType(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]any{
		"symbol":    "AAPL",
		"data_type": "indicators",
		"days":      30,
	})
	require.NoError(t, err)

	ind, ok := result.(*IndicatorResult)
	require.True(t, ok, "result should be *IndicatorResult")
	assert.Equal(t, "AAPL", ind.Symbol)
	assert.Greater(t, ind.DataPoints, 0)
	assert.Greater(t, ind.LastClose, 0.0)
	// Should have enough mock data for SMA20 and RSI14
	if ind.SMA20 != nil {
		assert.Greater(t, *ind.SMA20, 0.0)
	}
	if ind.RSI14 != nil {
		assert.GreaterOrEqual(t, *ind.RSI14, 0.0)
		assert.LessOrEqual(t, *ind.RSI14, 100.0)
	}
}

func TestOpenBBMarketDataTool_UnsupportedDataType(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]any{
		"symbol":    "AAPL",
		"data_type": "options",
	})
	require.Error(t, err)
	var connectErr *connect.Error
	assert.True(t, errors.As(err, &connectErr))
	assert.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
}

func TestMockHistoricalData(t *testing.T) {
	data := mockHistoricalData("AAPL", 30)
	require.Greater(t, len(data), 0)
	require.LessOrEqual(t, len(data), 30)

	for i, p := range data {
		assert.Greater(t, p.Close, 0.0, "point %d should have positive close", i)
		assert.Greater(t, p.Volume, int64(0), "point %d should have positive volume", i)
		assert.False(t, p.Date.IsZero(), "point %d should have a date", i)
		// No weekends
		assert.NotEqual(t, time.Saturday, p.Date.Weekday(), "point %d is Saturday", i)
		assert.NotEqual(t, time.Sunday, p.Date.Weekday(), "point %d is Sunday", i)
	}

	// Verify chronological order
	for i := 1; i < len(data); i++ {
		assert.True(t, data[i].Date.After(data[i-1].Date) || data[i].Date.Equal(data[i-1].Date),
			"data should be in chronological order")
	}
}

func TestOpenBBMarketDataTool_PriceWithDays(t *testing.T) {
	tool := NewOpenBBMarketDataTool()
	ctx := context.Background()

	// data_type="price" with days param should still work (default path via getStockPrice mock)
	result, err := tool.Execute(ctx, map[string]any{
		"symbol":    "MSFT",
		"data_type": "price",
		"days":      100,
	})
	require.NoError(t, err)
	r, ok := result.(*OpenBBMarketDataResult)
	require.True(t, ok)
	assert.Equal(t, "MSFT", r.Symbol)
	assert.Greater(t, r.Price, 0.0)
}

func TestMeanFunction(t *testing.T) {
	assert.InDelta(t, 3.0, mean([]float64{1, 2, 3, 4, 5}), 0.01)
	assert.InDelta(t, 0.0, mean([]float64{}), 0.01)
	assert.InDelta(t, 5.0, mean([]float64{5}), 0.01)
}

func TestSentimentScoreBounds(t *testing.T) {
	tool := NewSentimentAnalysisFinTool()
	ctx := context.Background()

	tests := []struct {
		name string
		text string
	}{
		{"very positive", "bullish upgrade outperform beat record surge rally"},
		{"very negative", "bankruptcy fraud default lawsuit downgrade bearish"},
		{"mixed", "growth opportunity but also risk and uncertainty"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tool.Execute(ctx, map[string]any{
				"text":   tc.text,
				"source": "news",
			})
			require.NoError(t, err)
			r, ok := result.(*SentimentResult)
			require.True(t, ok)
			assert.GreaterOrEqual(t, r.Score, 0.0, "score should be >= 0")
			assert.LessOrEqual(t, r.Score, 1.0, "score should be <= 1")
		})
	}
}
