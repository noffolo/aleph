package genesis

import (
	"context"
	"log/slog"
)

type SuggesterInput struct {
	ProjectID     string
	AgentID       string
	ChatHistory   []ChatMessage
	ToolUsage     []ToolUsageStat
	ExistingTools []string
}

type ChatMessage struct {
	Role    string
	Content string
}

type ToolUsageStat struct {
	ToolName string
	Count    int
}

type Suggester struct{}

func NewSuggester() *Suggester {
	return &Suggester{}
}

func (s *Suggester) Analyze(ctx context.Context, input SuggesterInput) ([]Suggestion, error) {
	slog.Info("genesis: analyzing patterns", "project", input.ProjectID, "agent", input.AgentID)
	return []Suggestion{}, nil
}
