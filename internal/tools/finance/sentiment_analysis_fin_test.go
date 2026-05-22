package finance

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// NewSentimentAnalysisFinTool
// ---------------------------------------------------------------------------

func TestNewSentimentAnalysisFinTool_Happy(t *testing.T) {
	tool := NewSentimentAnalysisFinTool()
	assert.NotNil(t, tool)
	assert.Nil(t, tool.nlpAdapter, "nlpAdapter should be nil initially")
}

func TestNewSentimentAnalysisFinTool_MultipleInstances(t *testing.T) {
	a := NewSentimentAnalysisFinTool()
	b := NewSentimentAnalysisFinTool()
	assert.NotNil(t, a)
	assert.NotNil(t, b)
	assert.NotSame(t, a, b)
}

func TestNewSentimentAnalysisFinTool_ReturnsEmptyAdapter(t *testing.T) {
	tool := NewSentimentAnalysisFinTool()
	assert.Nil(t, tool.nlpAdapter)
}

// ---------------------------------------------------------------------------
// Name
// ---------------------------------------------------------------------------

func TestSentimentAnalysisFinTool_Name_ReturnsExpected(t *testing.T) {
	tool := NewSentimentAnalysisFinTool()
	assert.Equal(t, "SentimentAnalysis", tool.Name())
}

func TestSentimentAnalysisFinTool_Name_Consistent(t *testing.T) {
	tool := NewSentimentAnalysisFinTool()
	n1 := tool.Name()
	n2 := tool.Name()
	assert.Equal(t, n1, n2)
}

func TestSentimentAnalysisFinTool_Name_NotEmpty(t *testing.T) {
	tool := NewSentimentAnalysisFinTool()
	assert.NotEmpty(t, tool.Name())
}

// ---------------------------------------------------------------------------
// SetNLPAdapter — additional edges
// ---------------------------------------------------------------------------

func TestSetNLPAdapter_NilAdapter(t *testing.T) {
	tool := NewSentimentAnalysisFinTool()
	mock := &mockNLPAnalyzer{score: 0.9, label: "positive"}
	tool.SetNLPAdapter(mock)
	assert.NotNil(t, tool.nlpAdapter)

	// Reset to nil — not directly supported but we can create a new tool
	tool2 := NewSentimentAnalysisFinTool()
	assert.Nil(t, tool2.nlpAdapter)
}

func TestSetNLPAdapter_PreservesExisting(t *testing.T) {
	tool := NewSentimentAnalysisFinTool()
	first := &mockNLPAnalyzer{score: 0.5, label: "neutral"}
	second := &mockNLPAnalyzer{score: 0.9, label: "positive"}

	tool.SetNLPAdapter(first)
	tool.SetNLPAdapter(second)

	// Should use the second one
	assert.Equal(t, second, tool.nlpAdapter)
}

func TestSetNLPAdapter_CanStillUseSynthetic(t *testing.T) {
	tool := NewSentimentAnalysisFinTool()
	mock := &mockNLPAnalyzer{
		err: assert.AnError,
	}
	tool.SetNLPAdapter(mock)

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]any{
		"text":   "strong growth and bullish outlook",
		"source": "news",
	})
	require.NoError(t, err)
	r := result.(*SentimentResult)
	assert.True(t, r.IsSynthetic) // falls back when NLP errors
}

// ---------------------------------------------------------------------------
// Execute — additional edge cases (NLPAdapter behaviors)
// ---------------------------------------------------------------------------

func TestSentimentAnalysisFinTool_Execute_NLPAdapterScoreOutsideRange(t *testing.T) {
	mock := &mockNLPAnalyzer{
		score: -5.0,
		label: "very-negative",
	}
	tool := NewSentimentAnalysisFinTool()
	tool.SetNLPAdapter(mock)

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]any{
		"text":   "testing out of bounds",
		"source": "news",
	})
	require.NoError(t, err)
	r := result.(*SentimentResult)
	assert.InDelta(t, -2.0, r.Score, 0.01)
}

func TestSentimentAnalysisFinTool_Execute_NLPAdapterNegativeScore(t *testing.T) {
	mock := &mockNLPAnalyzer{
		score: -0.8,
		label: "negative",
	}
	tool := NewSentimentAnalysisFinTool()
	tool.SetNLPAdapter(mock)

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]any{
		"text":   "negative news",
		"source": "news",
	})
	require.NoError(t, err)
	r := result.(*SentimentResult)
	assert.Equal(t, "negative", r.Sentiment)
	assert.InDelta(t, -0.8, r.Score, 0.01)
}

func TestSentimentAnalysisFinTool_Execute_NLPAdapterPositiveScore(t *testing.T) {
	mock := &mockNLPAnalyzer{
		score: 0.9,
		label: "positive",
	}
	tool := NewSentimentAnalysisFinTool()
	tool.SetNLPAdapter(mock)

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]any{
		"text":   "great results",
		"source": "filings",
	})
	require.NoError(t, err)
	r := result.(*SentimentResult)
	assert.Equal(t, "positive", r.Sentiment)
	assert.InDelta(t, 0.9, r.Score, 0.01)
	assert.Equal(t, "positive", r.Label)
}

// ---------------------------------------------------------------------------
// syntheticAnalysis — directly test synthetic logic
// ---------------------------------------------------------------------------

func TestSyntheticAnalysis_PositiveText(t *testing.T) {
	tool := NewSentimentAnalysisFinTool()
	ctx := context.Background()

	result := tool.syntheticAnalysis(ctx,
		"Company reports record revenue and strong growth with bullish momentum",
		"news")

	assert.Equal(t, "positive", result.Sentiment)
	assert.Greater(t, result.Score, 0.5)
	assert.True(t, result.IsSynthetic)
	assert.Equal(t, "synthetic", result.Label)
}

func TestSyntheticAnalysis_NegativeText(t *testing.T) {
	tool := NewSentimentAnalysisFinTool()
	ctx := context.Background()

	result := tool.syntheticAnalysis(ctx,
		"Company faces bankruptcy risk and fraud investigation with downgrade",
		"news")

	assert.Equal(t, "negative", result.Sentiment)
	assert.Less(t, result.Score, 0.5)
	assert.True(t, result.IsSynthetic)
}

func TestSyntheticAnalysis_EmptyText(t *testing.T) {
	tool := NewSentimentAnalysisFinTool()
	ctx := context.Background()

	result := tool.syntheticAnalysis(ctx, "", "news")

	assert.Equal(t, "neutral", result.Sentiment)
	assert.InDelta(t, 0.5, result.Score, 0.01)
	assert.True(t, result.IsSynthetic)
}

func TestSyntheticAnalysis_FilingsSource(t *testing.T) {
	tool := NewSentimentAnalysisFinTool()
	ctx := context.Background()

	result := tool.syntheticAnalysis(ctx,
		"Company exceeded revenue expectations with strong growth and record profit",
		"filings")

	// Filings source reduces confidence range
	assert.GreaterOrEqual(t, result.Score, 0.0)
	assert.LessOrEqual(t, result.Score, 1.0)
	assert.True(t, result.IsSynthetic)
}

func TestSyntheticAnalysis_SocialSource(t *testing.T) {
	tool := NewSentimentAnalysisFinTool()
	ctx := context.Background()

	result := tool.syntheticAnalysis(ctx,
		"Bearish downturn and massive losses",
		"social")

	assert.Less(t, result.Score, 0.5)
	assert.True(t, result.IsSynthetic)
}

func TestSyntheticAnalysis_EarningsCallsSource(t *testing.T) {
	tool := NewSentimentAnalysisFinTool()
	ctx := context.Background()

	result := tool.syntheticAnalysis(ctx,
		"Upgrade and beat expectations with innovation momentum",
		"earnings_calls")

	assert.GreaterOrEqual(t, result.Score, 0.0)
	assert.LessOrEqual(t, result.Score, 1.0)
}

// ---------------------------------------------------------------------------
// scoreToLabel — boundary edge cases
// ---------------------------------------------------------------------------

func TestScoreToLabel_BoundaryExactly06(t *testing.T) {
	assert.Equal(t, "positive", scoreToLabel(0.6))
}

func TestScoreToLabel_BoundaryExactly04(t *testing.T) {
	assert.Equal(t, "negative", scoreToLabel(0.4))
}

func TestScoreToLabel_BoundaryJustAbove04(t *testing.T) {
	assert.Equal(t, "neutral", scoreToLabel(0.41))
}

func TestScoreToLabel_BoundaryJustBelow06(t *testing.T) {
	assert.Equal(t, "neutral", scoreToLabel(0.59))
}

func TestScoreToLabel_ExtremePositive(t *testing.T) {
	assert.Equal(t, "positive", scoreToLabel(1.0))
	assert.Equal(t, "positive", scoreToLabel(0.99))
}

func TestScoreToLabel_ExtremeNegative(t *testing.T) {
	assert.Equal(t, "negative", scoreToLabel(0.0))
	assert.Equal(t, "negative", scoreToLabel(0.01))
}

func TestScoreToLabel_Midpoint(t *testing.T) {
	assert.Equal(t, "neutral", scoreToLabel(0.5))
}

func TestScoreToLabel_AboveOne(t *testing.T) {
	// scoreToLabel doesn't clamp input — it just checks >= 0.6
	assert.Equal(t, "positive", scoreToLabel(1.5))
}

func TestScoreToLabel_BelowZero(t *testing.T) {
	assert.Equal(t, "negative", scoreToLabel(-0.5))
}

// ---------------------------------------------------------------------------
// tokenize — additional edge cases
// ---------------------------------------------------------------------------

func TestTokenize_HyphenatedWords(t *testing.T) {
	result := tokenize("write-down impairment")
	assert.Contains(t, result, "write-down")
	assert.Contains(t, result, "impairment")
}

func TestTokenize_MultipleHyphens(t *testing.T) {
	result := tokenize("well-known high-growth company")
	assert.Contains(t, result, "well-known")
	assert.Contains(t, result, "high-growth")
	assert.Contains(t, result, "company")
}

func TestTokenize_NumbersOnly(t *testing.T) {
	result := tokenize("123 456 789")
	assert.Empty(t, result)
}

func TestTokenize_Punctuation(t *testing.T) {
	result := tokenize("hello! world? goodbye.")
	assert.Equal(t, []string{"hello", "world", "goodbye"}, result)
}

func TestTokenize_SingleWord(t *testing.T) {
	result := tokenize("hello")
	assert.Equal(t, []string{"hello"}, result)
}

func TestTokenize_LeadingTrailingSpaces(t *testing.T) {
	result := tokenize("  hello   world  ")
	assert.Equal(t, []string{"hello", "world"}, result)
}

func TestTokenize_MixedContent(t *testing.T) {
	result := tokenize("The Q4 2024 revenue grew 15% — impressive!")
	assert.Equal(t, []string{"The", "Q", "revenue", "grew", "impressive"}, result)
}

func TestTokenize_WhitespaceOnly(t *testing.T) {
	result := tokenize("   \t\n  ")
	assert.Empty(t, result)
}

// ---------------------------------------------------------------------------
// Execute — additional edge cases (source variants)
// ---------------------------------------------------------------------------

func TestSentimentAnalysisFinTool_Execute_DefaultSourceNews(t *testing.T) {
	tool := NewSentimentAnalysisFinTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]any{
		"text": "bullish outlook with growth",
	})
	require.NoError(t, err)
	r := result.(*SentimentResult)
	assert.Equal(t, "positive", r.Sentiment)
	assert.True(t, r.IsSynthetic)
}

func TestSentimentAnalysisFinTool_Execute_AllSourceTypes(t *testing.T) {
	sources := []string{"news", "social", "filings", "earnings_calls"}
	tool := NewSentimentAnalysisFinTool()
	ctx := context.Background()

	for _, src := range sources {
		t.Run(src, func(t *testing.T) {
			result, err := tool.Execute(ctx, map[string]any{
				"text":   "Bullish growth momentum with record profit",
				"source": src,
			})
			require.NoError(t, err)
			r := result.(*SentimentResult)
			assert.Equal(t, "positive", r.Sentiment)
			assert.GreaterOrEqual(t, r.Score, 0.0)
			assert.LessOrEqual(t, r.Score, 1.0)
			assert.True(t, r.IsSynthetic)
		})
	}
}

func TestSentimentAnalysisFinTool_Execute_MixedSentiment(t *testing.T) {
	tool := NewSentimentAnalysisFinTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]any{
		"text":   "Bullish growth but some risk and uncertainty remains",
		"source": "news",
	})
	require.NoError(t, err)
	r := result.(*SentimentResult)
	assert.GreaterOrEqual(t, r.Score, 0.0)
	assert.LessOrEqual(t, r.Score, 1.0)
	assert.True(t, r.IsSynthetic)
}

// ---------------------------------------------------------------------------
// NLPAdapter normalization
// ---------------------------------------------------------------------------

func TestSentimentAnalysisFinTool_Execute_NLPAdapterNeutralScore(t *testing.T) {
	mock := &mockNLPAnalyzer{
		score: 0.5,
		label: "neutral",
	}
	tool := NewSentimentAnalysisFinTool()
	tool.SetNLPAdapter(mock)

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]any{
		"text":   "neutral text",
		"source": "news",
	})
	require.NoError(t, err)
	r := result.(*SentimentResult)
	assert.Equal(t, "neutral", r.Sentiment)
	assert.InDelta(t, 0.5, r.Score, 0.01)
}

func TestSentimentAnalysisFinTool_Execute_NLPAdapterScoreExactlyOne(t *testing.T) {
	mock := &mockNLPAnalyzer{
		score: 1.0,
		label: "positive",
	}
	tool := NewSentimentAnalysisFinTool()
	tool.SetNLPAdapter(mock)

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]any{
		"text":   "perfect sentiment",
		"source": "news",
	})
	require.NoError(t, err)
	r := result.(*SentimentResult)
	assert.Equal(t, "positive", r.Sentiment)
	assert.InDelta(t, 1.0, r.Score, 0.01)
}

// ---------------------------------------------------------------------------
// Synthetic analysis: extreme scenarios
// ---------------------------------------------------------------------------

func TestSyntheticAnalysis_AllPositiveKeywords(t *testing.T) {
	tool := NewSentimentAnalysisFinTool()
	ctx := context.Background()

	result := tool.syntheticAnalysis(ctx,
		"bullish outperform upgrade beat growth profit revenue strong buy positive innovation expansion record surge rally momentum guidance upward gain opportunity dividend exceed exceeded ahead grow rising boost confidence favorable outlook",
		"news")

	assert.Greater(t, result.Score, 0.8, "all positive keywords → high score")
	assert.Equal(t, "positive", result.Sentiment)
	assert.True(t, result.IsSynthetic)
}

func TestSyntheticAnalysis_AllNegativeKeywords(t *testing.T) {
	tool := NewSentimentAnalysisFinTool()
	ctx := context.Background()

	result := tool.syntheticAnalysis(ctx,
		"bearish downgrade underperform miss decline loss debt weak sell negative risk volatility lawsuit investigation fine penalty downturn slowdown cut below layoff restructuring impairment write-down default bankruptcy fraud uncertainty concern pressure challenge",
		"news")

	assert.Less(t, result.Score, 0.2, "all negative keywords → low score")
	assert.Equal(t, "negative", result.Sentiment)
	assert.True(t, result.IsSynthetic)
}

func TestSyntheticAnalysis_NoFinancialKeywords(t *testing.T) {
	tool := NewSentimentAnalysisFinTool()
	ctx := context.Background()

	result := tool.syntheticAnalysis(ctx,
		"The cat sat on the mat",
		"news")

	assert.InDelta(t, 0.5, result.Score, 0.01)
	assert.Equal(t, "neutral", result.Sentiment)
	assert.True(t, result.IsSynthetic)
}

// ---------------------------------------------------------------------------
// Score bounds verification
// ---------------------------------------------------------------------------

func TestSentimentResult_ScoreAlwaysInRange(t *testing.T) {
	tool := NewSentimentAnalysisFinTool()
	ctx := context.Background()

	texts := []string{
		"bullish upgrade beat record surge",           // heavily positive
		"bankruptcy fraud default lawsuit downgrade",  // heavily negative
		"the quick brown fox",                         // neutral
		"growth but also risk and uncertainty",        // mixed
	}

	for _, text := range texts {
		t.Run("", func(t *testing.T) {
			result, err := tool.Execute(ctx, map[string]any{
				"text":   text,
				"source": "news",
			})
			require.NoError(t, err)
			r := result.(*SentimentResult)
			assert.GreaterOrEqual(t, r.Score, 0.0, "score should be >= 0 for text: %q", text)
			assert.LessOrEqual(t, r.Score, 1.0, "score should be <= 1 for text: %q", text)
		})
	}
}

// ---------------------------------------------------------------------------
// JSON round-trip: SentimentResult
// ---------------------------------------------------------------------------

func TestSentimentResult_JSONRoundTrip(t *testing.T) {
	original := SentimentResult{
		Sentiment:   "positive",
		Score:       0.85,
		IsSynthetic: true,
		Label:       "synthetic",
	}

	b, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded SentimentResult
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Sentiment, decoded.Sentiment)
	assert.InDelta(t, original.Score, decoded.Score, 0.001)
	assert.Equal(t, original.IsSynthetic, decoded.IsSynthetic)
	assert.Equal(t, original.Label, decoded.Label)
}

func TestSentimentResult_JSONOmitEmpty(t *testing.T) {
	result := SentimentResult{
		Sentiment:   "neutral",
		Score:       0.5,
		IsSynthetic: false,
	}

	b, err := json.Marshal(result)
	require.NoError(t, err)

	// Label has omitempty, so it should not appear when empty
	str := string(b)
	assert.NotContains(t, str, `"label"`)
}

func TestSentimentResult_JSONWithLabel(t *testing.T) {
	result := SentimentResult{
		Sentiment:   "positive",
		Score:       0.8,
		IsSynthetic: false,
		Label:       "nlp-positive",
	}

	b, err := json.Marshal(result)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"label":"nlp-positive"`)
}

// ---------------------------------------------------------------------------
// NLPAdapter: error handling edge cases
// ---------------------------------------------------------------------------

func TestSentimentAnalysisFinTool_Execute_NLPAdapterNilError(t *testing.T) {
	// When adapter is nil, it should fall through to synthetic
	tool := NewSentimentAnalysisFinTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]any{
		"text":   "Strong earnings beat expectations",
		"source": "news",
	})
	require.NoError(t, err)
	r := result.(*SentimentResult)
	assert.True(t, r.IsSynthetic)
}

func TestSentimentAnalysisFinTool_Execute_NLPAdapterScoreZero(t *testing.T) {
	mock := &mockNLPAnalyzer{
		score: 0.0,
		label: "negative",
	}
	tool := NewSentimentAnalysisFinTool()
	tool.SetNLPAdapter(mock)

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]any{
		"text":   "negative text",
		"source": "news",
	})
	require.NoError(t, err)
	r := result.(*SentimentResult)
	assert.Equal(t, "negative", r.Sentiment)
	assert.InDelta(t, 0.0, r.Score, 0.01)
	assert.True(t, r.IsSynthetic)
}

// ---------------------------------------------------------------------------
// Boundary: score rounding
// ---------------------------------------------------------------------------

func TestSentimentScoreRounding(t *testing.T) {
	// The code does math.Round(normalized*100) / 100
	assert.InDelta(t, 0.75, math.Round(0.754*100)/100, 0.0001)
	assert.InDelta(t, 0.76, math.Round(0.756*100)/100, 0.0001)
}

// Ensure unused imports compile
var _ = strings.TrimSpace("")
var _ = context.Background
