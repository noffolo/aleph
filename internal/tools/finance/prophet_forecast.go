package finance

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/repository"
)

// maxSMAWindow is the maximum trailing window size for SMA fallback.
const maxSMAWindow = 5

// ProphetForecastArgs represents the input arguments for forecasting.
type ProphetForecastArgs struct {
	Data    []float64 `json:"data"`
	Periods int       `json:"periods"`
}

// ProphetForecastResult represents the forecast output.
type ProphetForecastResult struct {
	Predictions []float64 `json:"predictions"`
	Confidence  float64   `json:"confidence"`
	Method      string    `json:"method"`
}

// ProphetForecastTool provides time-series forecasting using SMA or linear regression.
type ProphetForecastTool struct{}

// NewProphetForecastTool returns a new ProphetForecastTool instance.
func NewProphetForecastTool() *ProphetForecastTool {
	return &ProphetForecastTool{}
}

// Execute runs a forecast using pure Go algorithms. Args:
//   - data: []float64 — historical data points
//   - periods: int — number of periods to forecast
//
// Uses Simple Moving Average when data is short (< 4 points) or stationary,
// otherwise uses linear regression. Returns predictions with confidence and method info.
func (t *ProphetForecastTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	var pArgs ProphetForecastArgs
	if err := parseArgs(args, &pArgs); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("invalid prophet_forecast args: %w", err))
	}

	if len(pArgs.Data) < 2 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("prophet_forecast requires at least 2 data points, got %d", len(pArgs.Data)))
	}
	if pArgs.Periods <= 0 {
		pArgs.Periods = 1
	}

	// Determine method: SMA for ≤3 points, linear regression for more
	var predictions []float64
	var confidence float64
	var method string

	if len(pArgs.Data) <= 3 {
		predictions, confidence = smaForecast(pArgs.Data, pArgs.Periods)
		method = "sma"
	} else {
		predictions, confidence = linearRegressionForecast(pArgs.Data, pArgs.Periods)
		method = "linear"
	}

	return &ProphetForecastResult{
		Predictions: predictions,
		Confidence:  confidence,
		Method:      method,
	}, nil
}

// Register registers the tool in the metadata repository.
func (t *ProphetForecastTool) Register(metaRepo *repository.MetadataRepository) error {
	return metaRepo.CreateTool(&repository.ToolRecord{
		ID:           "finance_prophet_forecast",
		Name:         "finance_prophet_forecast",
		Description:  "Time-series forecasting using SMA/linear regression (no Python Prophet dependency)",
		Code:         "",
		Category:     "finance",
		Version:      "1.0.0",
		HealthStatus: "unknown",
		SourceType:   "package",
	})
}

// smaForecast computes a Simple Moving Average from the last window data points
// and projects it forward for `periods` steps.
func smaForecast(data []float64, periods int) ([]float64, float64) {
	window := maxSMAWindow
	if len(data) < window {
		window = len(data)
	}

	// Compute trailing window average
	var sum float64
	for i := len(data) - window; i < len(data); i++ {
		sum += data[i]
	}
	avg := sum / float64(window)

	// Calculate variance for confidence
	var variance float64
	for i := len(data) - window; i < len(data); i++ {
		diff := data[i] - avg
		variance += diff * diff
	}
	variance /= float64(window)

	// Confidence inversely related to coefficient of variation
	confidence := 1.0
	if avg != 0 {
		cv := math.Sqrt(variance) / math.Abs(avg)
		confidence = 1.0 - math.Min(cv, 0.95)
	}

	predictions := make([]float64, periods)
	for i := range predictions {
		predictions[i] = avg
	}

	return predictions, math.Round(confidence*100) / 100
}

// linearRegressionForecast fits a line y = a + b*x to the data and projects forward.
func linearRegressionForecast(data []float64, periods int) ([]float64, float64) {
	n := float64(len(data))

	var sumX, sumY, sumXY, sumX2 float64
	for i, y := range data {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// Slope b = (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	denom := n*sumX2 - sumX*sumX
	var b float64
	if denom != 0 {
		b = (n*sumXY - sumX*sumY) / denom
	}
	// Intercept a = (sumY - b*sumX) / n
	a := (sumY - b*sumX) / n

	// Calculate R² for confidence
	var ssRes, ssTot float64
	meanY := sumY / n
	for i, y := range data {
		fit := a + b*float64(i)
		residual := y - fit
		ssRes += residual * residual
		diff := y - meanY
		ssTot += diff * diff
	}

	confidence := 0.5 // default medium confidence
	if ssTot > 0 {
		r2 := 1.0 - ssRes/ssTot
		confidence = math.Max(0.1, math.Min(r2, 1.0))
	}

	predictions := make([]float64, periods)
	lastIdx := float64(len(data) - 1)
	for i := range predictions {
		predictions[i] = a + b*(lastIdx+float64(i)+1)
	}

	return predictions, math.Round(confidence*100) / 100
}

// parseArgs unmarshals args into the target struct via JSON round-trip.
// This handles the map[string]any → typed struct conversion cleanly.
func parseArgs(args map[string]any, target any) error {
	if args == nil {
		return nil
	}
	b, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("marshal args: %w", err)
	}
	if err := json.Unmarshal(b, target); err != nil {
		return fmt.Errorf("unmarshal args: %w", err)
	}
	return nil
}
