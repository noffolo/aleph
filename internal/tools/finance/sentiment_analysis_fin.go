package finance

import (
	"context"
	"fmt"
	"math"
	"strings"
	"unicode"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/repository"
)

// NLPAnalyzer defines the interface for sentiment analysis.
// Mirrors ingestion.NLPAnalyzer for use within the finance package.
type NLPAnalyzer interface {
	AnalyzeSentiment(ctx context.Context, text string) (score float32, label string, err error)
}

// SentimentArgs represents the input arguments for sentiment analysis.
type SentimentArgs struct {
	Text   string `json:"text"`
	Source string `json:"source"`
}

// SentimentResult represents the sentiment analysis output.
type SentimentResult struct {
	Sentiment   string  `json:"sentiment"`
	Score       float64 `json:"score"`
	IsSynthetic bool    `json:"is_synthetic"`
	Label       string  `json:"label,omitempty"`
}

// SentimentAnalysisFinTool provides financial sentiment analysis.
// Delegates to NLPAdapter when available; falls back to keyword-based
// synthetic analysis otherwise.
type SentimentAnalysisFinTool struct {
	nlpAdapter NLPAnalyzer
}

// NewSentimentAnalysisFinTool returns a new SentimentAnalysisFinTool instance.
func NewSentimentAnalysisFinTool() *SentimentAnalysisFinTool {
	return &SentimentAnalysisFinTool{}
}

// SetNLPAdapter sets the NLP adapter for real sentiment analysis.
// When set, the tool delegates to the adapter instead of using synthetic analysis.
func (t *SentimentAnalysisFinTool) SetNLPAdapter(adapter NLPAnalyzer) {
	t.nlpAdapter = adapter
}

// Execute runs financial sentiment analysis. Args:
//   - text: string — input text to analyze
//   - source: string — "news", "social", "filings", "earnings_calls"
//
// Returns sentiment result with score [0,1], label, and is_synthetic flag.
func (t *SentimentAnalysisFinTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	var sArgs SentimentArgs
	if err := parseArgs(args, &sArgs); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("invalid sentiment args: %w", err))
	}
	if strings.TrimSpace(sArgs.Text) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("sentiment analysis requires non-empty text"))
	}
	if sArgs.Source == "" {
		sArgs.Source = "news"
	}

	// Try NLP adapter first
	if t.nlpAdapter != nil {
		score, label, err := t.nlpAdapter.AnalyzeSentiment(ctx, sArgs.Text)
		if err == nil {
			// Normalize from [-1,1] scope to [0,1] if needed
			normalized := float64(score)
			if normalized < -1 || normalized > 1 {
				normalized = (normalized + 1) / 2
			}
			sentiment := scoreToLabel(normalized)
			return &SentimentResult{
				Sentiment:   sentiment,
				Score:       math.Round(normalized*100) / 100,
				IsSynthetic: label == "synthetic" || score < 0.5,
				Label:       label,
			}, nil
		}
	}

	// Fallback to synthetic keyword-based analysis
	return t.syntheticAnalysis(ctx, sArgs.Text, sArgs.Source), nil
}

// syntheticAnalysis performs keyword-based financial sentiment analysis.
// It uses domain-specific positive/negative financial terms to compute
// a sentiment score. Always returns is_synthetic: true.
func (t *SentimentAnalysisFinTool) syntheticAnalysis(ctx context.Context, text, source string) *SentimentResult {
	positiveWords := map[string]int{
		"bullish": 3, "outperform": 3, "upgrade": 3, "beat": 2,
		"growth": 2, "profit": 2, "revenue": 1, "strong": 2,
		"buy": 2, "positive": 2, "innovation": 1, "expansion": 2,
		"record": 2, "surge": 2, "rally": 2, "momentum": 2,
		"guidance": 1, "upward": 2, "gain": 1, "opportunity": 1,
		"dividend": 1, 		"exceed": 2, "exceeded": 2,
		"ahead": 1, "grow": 2, "rising": 1, "boost": 2,
		"confidence": 1, "favorable": 2, "outlook": 1,
	}

	negativeWords := map[string]int{
		"bearish": 3, "downgrade": 3, "underperform": 3, "miss": 2,
		"decline": 2, "loss": 2, "debt": 1, "weak": 2,
		"sell": 2, "negative": 2, "risk": 1, "volatility": 1,
		"lawsuit": 3, "investigation": 2, "fine": 2, "penalty": 2,
		"downturn": 2, "slowdown": 2, "cut": 1, "below": 1,
		"layoff": 2, "restructuring": 1, "impairment": 2,
		"write-down": 2, "default": 3, "bankruptcy": 3,
		"fraud": 3, "uncertainty": 1,
		"concern": 1, "pressure": 1, "challenge": 1,
	}

	lower := strings.ToLower(text)
	words := tokenize(lower)

	var positiveScore, negativeScore int
	for _, w := range words {
		if val, ok := positiveWords[w]; ok {
			positiveScore += val
		}
		if val, ok := negativeWords[w]; ok {
			negativeScore += val
		}
	}

	// Compute normalized score [0, 1]: 0 = most negative, 1 = most positive
	var normalized float64
	total := positiveScore + negativeScore
	if total > 0 {
		// weighted sentiment in [-1, 1]
		rawSentiment := float64(positiveScore-negativeScore) / float64(total)
		// map to [0, 1]
		normalized = (rawSentiment + 1.0) / 2.0
	} else {
		normalized = 0.5 // neutral when no keywords found
	}

	// Adjusted for source credibility
	switch source {
	case "filings", "earnings_calls":
		// Reduce confidence for official sources since language is typically neutral
		normalized = 0.3 + normalized*0.4
	case "social":
		// Social media tends to be more extreme, dampen slightly
		normalized = 0.2 + normalized*0.6
	}

	normalized = math.Round(math.Max(0, math.Min(1, normalized))*100) / 100

	return &SentimentResult{
		Sentiment:   scoreToLabel(normalized),
		Score:       normalized,
		IsSynthetic: true,
		Label:       "synthetic",
	}
}

// scoreToLabel converts a [0,1] score to a sentiment label.
func scoreToLabel(score float64) string {
	if score >= 0.6 {
		return "positive"
	}
	if score <= 0.4 {
		return "negative"
	}
	return "neutral"
}

// tokenize splits text into lowercase words, stripping punctuation.
func tokenize(text string) []string {
	words := make([]string, 0)
	var current strings.Builder
	for _, r := range text {
		if unicode.IsLetter(r) || r == '-' {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		}
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}
	return words
}

// Register registers the tool in the metadata repository.
func (t *SentimentAnalysisFinTool) Register(metaRepo *repository.MetadataRepository) error {
	return metaRepo.CreateTool(&repository.ToolRecord{
		ID:           "finance_sentiment_analysis",
		Name:         "finance_sentiment_analysis",
		Description:  "Financial sentiment analysis via NLPAdapter with synthetic fallback (beta) | is_synthetic=true",
		Code:         "",
		Category:     "finance",
		Version:      "1.0.0",
		HealthStatus: "unknown",
		SourceType:   "package",
	})
}
