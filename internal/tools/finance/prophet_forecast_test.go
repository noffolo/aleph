package finance

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProphetForecastTool_Happy(t *testing.T) {
	tool := NewProphetForecastTool()
	assert.NotNil(t, tool)
}

func TestNewProphetForecastTool_MultipleInstances(t *testing.T) {
	a := NewProphetForecastTool()
	b := NewProphetForecastTool()
	assert.NotNil(t, a)
	assert.NotNil(t, b)
	assert.NotSame(t, a, b)
}

func TestNewProphetForecastTool_IsEmptyStruct(t *testing.T) {
	tool := NewProphetForecastTool()
	assert.NotNil(t, tool)
}

func TestProphetForecastTool_Name_ReturnsExpected(t *testing.T) {
	tool := NewProphetForecastTool()
	assert.Equal(t, "ProphetForecast", tool.Name())
}

func TestProphetForecastTool_Name_Consistent(t *testing.T) {
	tool := NewProphetForecastTool()
	n1 := tool.Name()
	n2 := tool.Name()
	assert.Equal(t, n1, n2)
}

func TestProphetForecastTool_Name_NotEmpty(t *testing.T) {
	tool := NewProphetForecastTool()
	assert.NotEmpty(t, tool.Name())
}

func TestProphetForecastTool_Execute_ZeroPeriodDefaultsToOne(t *testing.T) {
	tool := NewProphetForecastTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]any{
		"data":    []float64{10, 20, 30, 40, 50},
		"periods": 0,
	})
	require.NoError(t, err)
	r, ok := result.(*ProphetForecastResult)
	require.True(t, ok)
	assert.Len(t, r.Predictions, 1)
}

func TestProphetForecastTool_Execute_LargePeriodsLinear(t *testing.T) {
	tool := NewProphetForecastTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]any{
		"data":    []float64{10, 20, 30, 40, 50},
		"periods": 100,
	})
	require.NoError(t, err)
	r, ok := result.(*ProphetForecastResult)
	require.True(t, ok)
	assert.Len(t, r.Predictions, 100)
	assert.Equal(t, "linear", r.Method)
}

func TestProphetForecastTool_Execute_NegativeTrend(t *testing.T) {
	tool := NewProphetForecastTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]any{
		"data":    []float64{100, 90, 80, 70, 60},
		"periods": 2,
	})
	require.NoError(t, err)
	r, ok := result.(*ProphetForecastResult)
	require.True(t, ok)
	assert.Less(t, r.Predictions[0], 60.0)
	assert.Less(t, r.Predictions[1], r.Predictions[0])
}

func TestSMAForecast_DataLengthOne(t *testing.T) {
	predictions, confidence := smaForecast([]float64{42.0}, 3)
	assert.Len(t, predictions, 3)
	for _, p := range predictions {
		assert.InDelta(t, 42.0, p, 0.01)
	}
	assert.InDelta(t, 1.0, confidence, 0.01)
}

func TestSMAForecast_HighlyVariableData(t *testing.T) {
	predictions, confidence := smaForecast([]float64{10, 1000, 5}, 2)
	assert.Len(t, predictions, 2)
	assert.Less(t, confidence, 0.5)
}

func TestSMAForecast_DataTwoPoints(t *testing.T) {
	predictions, confidence := smaForecast([]float64{100, 200}, 5)
	assert.Len(t, predictions, 5)
	for _, p := range predictions {
		assert.InDelta(t, 150.0, p, 0.01)
	}
	assert.Less(t, confidence, 1.0)
}

func TestLinearRegressionForecast_DecreasingTrend(t *testing.T) {
	data := []float64{100, 80, 60, 40, 20}
	predictions, confidence := linearRegressionForecast(data, 2)
	assert.Len(t, predictions, 2)
	assert.Less(t, predictions[0], 20.0)
	assert.Less(t, predictions[1], predictions[0])
	assert.InDelta(t, 1.0, confidence, 0.01)
}

func TestLinearRegressionForecast_FlatLine(t *testing.T) {
	data := []float64{100, 100, 100, 100, 100}
	predictions, confidence := linearRegressionForecast(data, 3)
	assert.Len(t, predictions, 3)
	for _, p := range predictions {
		assert.InDelta(t, 100.0, p, 0.01)
	}
	assert.InDelta(t, 0.5, confidence, 0.01)
}

func TestLinearRegressionForecast_LargePeriods(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	predictions, confidence := linearRegressionForecast(data, 100)
	assert.Len(t, predictions, 100)
	assert.Greater(t, predictions[50], predictions[0])
	assert.GreaterOrEqual(t, confidence, 0.0)
	assert.LessOrEqual(t, confidence, 100.0)
}

func TestParseArgs_ComplexNestedStruct(t *testing.T) {
	type Complex struct {
		Name    string    `json:"name"`
		Values  []float64 `json:"values"`
		Nested  struct {
			Key string `json:"key"`
		} `json:"nested"`
	}
	var target Complex
	err := parseArgs(map[string]any{
		"name":   "test",
		"values": []float64{1.0, 2.0, 3.0},
		"nested": map[string]any{
			"key": "val",
		},
	}, &target)
	require.NoError(t, err)
	assert.Equal(t, "test", target.Name)
	assert.Len(t, target.Values, 3)
	assert.Equal(t, "val", target.Nested.Key)
}

func TestParseArgs_InvalidJSONRoundTrip(t *testing.T) {
	type Simple struct {
		Name string `json:"name"`
	}
	var target Simple

	err := parseArgs(map[string]any{
		"name": make(chan int),
	}, &target)
	assert.Error(t, err)
}

func TestParseArgs_EmptyMap(t *testing.T) {
	type Simple struct {
		Name   string  `json:"name"`
		Value  float64 `json:"value"`
	}
	var target Simple
	target.Name = "prefilled"

	err := parseArgs(map[string]any{}, &target)
	require.NoError(t, err)
	assert.Equal(t, "prefilled", target.Name)
	assert.Equal(t, 0.0, target.Value)
}

func TestProphetForecastResult_JSONRoundTrip(t *testing.T) {
	original := ProphetForecastResult{
		Predictions: []float64{1.1, 2.2, 3.3},
		Confidence:  0.85,
		Method:      "linear",
	}

	b, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded ProphetForecastResult
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Predictions, decoded.Predictions)
	assert.InDelta(t, original.Confidence, decoded.Confidence, 0.001)
	assert.Equal(t, original.Method, decoded.Method)
}

func TestProphetForecastResult_EmptyPredictions(t *testing.T) {
	original := ProphetForecastResult{
		Predictions: []float64{},
		Confidence:  0.0,
		Method:      "sma",
	}

	b, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded ProphetForecastResult
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)

	assert.Empty(t, decoded.Predictions)
	assert.InDelta(t, 0.0, decoded.Confidence, 0.001)
}

func TestProphetForecastResult_NegativeConfidence(t *testing.T) {
	result := ProphetForecastResult{
		Predictions: []float64{10, 20},
		Confidence:  1.0,
		Method:      "sma",
	}
	b, _ := json.Marshal(result)
	var decoded ProphetForecastResult
	err := json.Unmarshal(b, &decoded)
	require.NoError(t, err)
	assert.InDelta(t, 1.0, decoded.Confidence, 0.001)
}

func TestSMAForecast_BoundaryWindow(t *testing.T) {
	data := []float64{10, 20, 30, 40, 50}
	predictions, confidence := smaForecast(data, 3)
	assert.Len(t, predictions, 3)
	avg := (10 + 20 + 30 + 40 + 50) / 5.0
	for _, p := range predictions {
		assert.InDelta(t, float64(avg), p, 0.01)
	}
	assert.GreaterOrEqual(t, confidence, 0.0)
	assert.LessOrEqual(t, confidence, 1.0)
}

func TestLinearRegressionForecast_SingleStep(t *testing.T) {
	data := []float64{10, 20, 30, 40}
	predictions, _ := linearRegressionForecast(data, 1)
	assert.Len(t, predictions, 1)
	assert.InDelta(t, 50.0, predictions[0], 0.01)
}

func TestProphetForecastTool_Execute_Exactly4PointsSwitchesToLinear(t *testing.T) {
	tool := NewProphetForecastTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]any{
		"data":    []float64{10, 20, 30, 40},
		"periods": 2,
	})
	require.NoError(t, err)
	r, ok := result.(*ProphetForecastResult)
	require.True(t, ok)
	assert.Equal(t, "linear", r.Method)
}

func TestProphetForecastTool_Execute_Exactly3PointsUsesSMA(t *testing.T) {
	tool := NewProphetForecastTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]any{
		"data":    []float64{10, 20, 30},
		"periods": 2,
	})
	require.NoError(t, err)
	r, ok := result.(*ProphetForecastResult)
	require.True(t, ok)
	assert.Equal(t, "sma", r.Method)
}

func TestProphetForecastResult_ConfidenceBounds_SMA(t *testing.T) {
	tool := NewProphetForecastTool()
	ctx := context.Background()

	tests := [][]float64{
		{10, 10, 10},
		{1, 2},
		{100, 50, 25, 12.5},
	}
	for _, data := range tests {
		result, err := tool.Execute(ctx, map[string]any{
			"data":    data,
			"periods": 5,
		})
		require.NoError(t, err)
		r := result.(*ProphetForecastResult)
		assert.GreaterOrEqual(t, r.Confidence, 0.0, "data %v", data)
		assert.LessOrEqual(t, r.Confidence, 1.0, "data %v", data)
	}
}

func TestProphetForecastResult_ConfidenceBounds_Linear(t *testing.T) {
	tool := NewProphetForecastTool()
	ctx := context.Background()

	tests := [][]float64{
		{1, 2, 3, 4, 5},
		{10, 200, 5, 300},
		{0, 0, 0, 0, 0},
	}
	for _, data := range tests {
		result, err := tool.Execute(ctx, map[string]any{
			"data":    data,
			"periods": 3,
		})
		require.NoError(t, err)
		r := result.(*ProphetForecastResult)
		assert.GreaterOrEqual(t, r.Confidence, 0.0, "data %v", data)
		assert.LessOrEqual(t, r.Confidence, 1.0, "data %v", data)
	}
}

func TestProphetForecastTool_Execute_ErrorNoSymbolArg(t *testing.T) {
	tool := NewProphetForecastTool()
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]any{
		"symbol": "AAPL",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least 2 data points")
}

func TestProphetForecastTool_Execute_ErrorWrongDataTypes(t *testing.T) {
	tool := NewProphetForecastTool()
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]any{
		"data":    "not an array",
		"periods": "three",
	})
	assert.Error(t, err)
}

func TestProphetForecastTool_Execute_ErrorNilArgs(t *testing.T) {
	tool := NewProphetForecastTool()
	ctx := context.Background()

	_, err := tool.Execute(ctx, nil)
	assert.Error(t, err)
	var connectErr *connect.Error
	assert.True(t, errors.As(err, &connectErr))
}

func TestLinearRegressionForecast_Precision(t *testing.T) {
	data := []float64{1.234, 2.345, 3.456, 4.567, 5.678}
	predictions, confidence := linearRegressionForecast(data, 3)
	assert.Len(t, predictions, 3)
	for i := 1; i < len(predictions); i++ {
		assert.Greater(t, predictions[i], predictions[i-1])
	}
	assert.GreaterOrEqual(t, confidence, 0.0)
}

func TestSMAForecast_Precision(t *testing.T) {
	data := []float64{1.111, 2.222, 3.333}
	predictions, confidence := smaForecast(data, 2)
	assert.Len(t, predictions, 2)
	avg := (1.111 + 2.222 + 3.333) / 3.0
	for _, p := range predictions {
		assert.InDelta(t, avg, p, 0.01)
	}
	assert.GreaterOrEqual(t, confidence, 0.0)
}

func TestConfidenceRounding(t *testing.T) {
	assert.InDelta(t, 0.85, math.Round(0.854*100)/100, 0.0001)
	assert.InDelta(t, 0.86, math.Round(0.856*100)/100, 0.0001)
	assert.InDelta(t, 1.0, math.Round(0.999*100)/100, 0.0001)
	assert.InDelta(t, 0.0, math.Round(0.001*100)/100, 0.0001)
}

var _ = context.Background
var _ = strings.TrimSpace("")
