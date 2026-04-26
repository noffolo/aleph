package finance

import (
	"context"
	"errors"
	"math"
	"testing"

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

	// Disable SSRF validation for tests that don't make real HTTP calls
	originalValidate := validateSSRF
	validateSSRF = func(url string) error { return nil }
	t.Cleanup(func() { validateSSRF = originalValidate })

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
			name   string
			args   map[string]any
			check  func(t *testing.T, result any)
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
		name     string
		data     []float64
		periods  int
		wantLen  int
		check    func(t *testing.T, predictions []float64, confidence float64)
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

func TestValidateSSRFDefault(t *testing.T) {
	// The package-level validateSSRF should be set to mcp.ValidateSSRF
	assert.NotNil(t, validateSSRF, "validateSSRF should be initialized")
	// Check it rejects private IPs
	err := validateSSRF("http://localhost:8080/data")
	assert.Error(t, err)
	err = validateSSRF("http://192.168.1.1/data")
	assert.Error(t, err)
	// Check it allows public URLs
	err = validateSSRF("https://query1.finance.yahoo.com/v8/finance/chart/AAPL")
	assert.NoError(t, err)
}

func TestMathRounding(t *testing.T) {
	// Verify math.Round works as expected in our context
	assert.Equal(t, 123.46, math.Round(123.456*100)/100)
	assert.Equal(t, 123.45, math.Round(123.454*100)/100)
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
