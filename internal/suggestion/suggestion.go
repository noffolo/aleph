package suggestion

import (
	"context"

	"github.com/ff3300/aleph-v2/internal/tools"
)

// Embedder is the interface for generating text embeddings.
// memory.Embedder implements this interface.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// Suggestion ranks a tool for a given user message.
type Suggestion struct {
	ToolName    string  `json:"tool_name"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Score       float64 `json:"score"`
	Similarity  float64 `json:"similarity"`
	UsageScore  float64 `json:"usage_score"`
	Reason      string  `json:"reason"`
}

// Engine suggests tools based on user input, using embedding similarity
// and usage frequency to rank available tools.
type Engine struct {
	registry *tools.ToolRegistry
	embedder Embedder
	usage    *usageTracker
	cache    *embedCache
}
