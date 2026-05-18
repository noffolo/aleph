package finance

import (
	"testing"

	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/stretchr/testify/assert"
)

func TestListTools_ReturnsAllTools(t *testing.T) {
	t.Parallel()
	tools := ListTools(nil)
	assert.Len(t, tools, 3, "ListTools should return 3 finance tools")
}

func TestNewTools_NotNil(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewSentimentAnalysisFinTool())
	assert.NotNil(t, NewOpenBBMarketDataTool())
	assert.NotNil(t, NewProphetForecastTool())
}

func TestListTools_ToolNames(t *testing.T) {
	t.Parallel()
	tools := ListTools(nil)
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, any(tool).(interface{ Name() string }).Name())
	}
	assert.Contains(t, names, "ProphetForecast")
	assert.Contains(t, names, "OpenBBMarketData")
	assert.Contains(t, names, "SentimentAnalysis")
}

type namedTool interface {
	Register(metaRepo *repository.MetadataRepository) error
	Name() string
}

func TestListTools_AllImplementRegister(t *testing.T) {
	t.Parallel()
	tools := ListTools(nil)
	for _, tool := range tools {
		_, ok := any(tool).(namedTool)
		assert.True(t, ok, "each tool should implement namedTool interface")
	}
}
