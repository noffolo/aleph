package genesis

import (
	"context"
	"time"
)

type Suggestion struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Priority    int       `json:"priority"`
	Confidence  float64   `json:"confidence"`
	Code        string    `json:"code,omitempty"`
	Parameters  string    `json:"parameters,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type GenesisEngine struct {
	suggester *Suggester
	sandbox   *Sandbox
	veto      *VetoRegistry
}

func NewGenesisEngine(suggester *Suggester, sandbox *Sandbox, veto *VetoRegistry) *GenesisEngine {
	return &GenesisEngine{
		suggester: suggester,
		sandbox:   sandbox,
		veto:      veto,
	}
}

func (g *GenesisEngine) Suggest(ctx context.Context, projectID string, agentID string) ([]Suggestion, error) {
	suggestions, err := g.suggester.Analyze(ctx, SuggesterInput{
		ProjectID: projectID,
		AgentID:   agentID,
	})
	if err != nil {
		return nil, err
	}

	for i, s := range suggestions {
		valid, err := g.sandbox.Validate(ctx, s)
		if err != nil || !valid {
			suggestions[i].Status = "invalid"
			continue
		}
		suggestions[i].Status = "pending"
	}

	for _, s := range suggestions {
		if s.Status == "pending" {
			g.veto.Register(s)
		}
	}

	return suggestions, nil
}

func (g *GenesisEngine) Approve(ctx context.Context, suggestionID string) error {
	return g.veto.Approve(ctx, suggestionID)
}

func (g *GenesisEngine) Reject(ctx context.Context, suggestionID string) error {
	return g.veto.Reject(ctx, suggestionID)
}

func (g *GenesisEngine) ListPending(ctx context.Context) ([]Suggestion, error) {
	return g.veto.ListPending(ctx)
}
