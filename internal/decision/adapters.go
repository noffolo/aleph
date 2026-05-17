package decision

import (
	"context"

	"github.com/ff3300/aleph-v2/internal/registry"
	"github.com/ff3300/aleph-v2/internal/repository"
)

// metaRepoClient is the minimal interface needed by MetaRepoAdapter.
// Using an interface instead of a concrete type enables test mocking.
type metaRepoClient interface {
	SaveChatMessage(ctx context.Context, projectID, agentID, role, content, toolCall string) error
	GetChatMessages(ctx context.Context, projectID, agentID string) ([]repository.ChatMessage, error)
	ListTools() ([]repository.ToolRecord, error)
}

// MetaRepoAdapter adapts a metadata repository to decision.ToolRepository.
type MetaRepoAdapter struct {
	Repo metaRepoClient
}

func (a *MetaRepoAdapter) SaveChatMessage(ctx context.Context, projectID, agentID, role, content, toolCall string) error {
	return a.Repo.SaveChatMessage(ctx, projectID, agentID, role, content, toolCall)
}

func (a *MetaRepoAdapter) GetChatMessages(ctx context.Context, projectID, agentID string) ([]ChatMessage, error) {
	msgs, err := a.Repo.GetChatMessages(ctx, projectID, agentID)
	if err != nil {
		return nil, err
	}
	result := make([]ChatMessage, len(msgs))
	for i, m := range msgs {
		result[i] = ChatMessage{
			Role:     m.Role,
			Content:  m.Content,
			ToolCall: m.ToolCall,
		}
	}
	return result, nil
}

func (a *MetaRepoAdapter) ListTools(ctx context.Context) ([]ToolDef, error) {
	tools, err := a.Repo.ListTools()
	if err != nil {
		return nil, err
	}
	result := make([]ToolDef, len(tools))
	for i, t := range tools {
		result[i] = ToolDef{
			Name:        t.Name,
			Description: t.Description,
			Code:        t.Code,
		}
	}
	return result, nil
}

// registryClient is the minimal interface needed by RegistryAdapter.
type registryClient interface {
	GetComponentByID(ctx context.Context, id string) (*registry.ComponentMetadata, error)
}

// RegistryAdapter adapts a registry client to decision.PluginRegistry.
type RegistryAdapter struct {
	Reg registryClient
}

func (a *RegistryAdapter) GetComponentByID(ctx context.Context, id string) (*ComponentMetadata, error) {
	comp, err := a.Reg.GetComponentByID(ctx, id)
	if err != nil || comp == nil {
		return nil, err
	}
	return &ComponentMetadata{
		ID:       comp.ID,
		Name:     comp.Name,
		Category: comp.Category,
		Status:   comp.Status,
	}, nil
}