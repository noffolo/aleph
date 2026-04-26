package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuditInterceptor_IsMutatingOperation(t *testing.T) {
	tests := []struct {
		procedure string
		expected  bool
	}{
		{"aleph.v1.AgentService.CreateAgent", true},
		{"aleph.v1.ToolService.CreateTool", true},
		{"aleph.v1.SkillService.CreateSkill", true},
		{"aleph.v1.IngestionService.StartIngestion", true},
		{"aleph.v1.TaskService.DeleteTask", true},
		{"aleph.v1.NotificationService.SendNotification", true},
		{"aleph.v1.AgentService.ListAgents", false},
		{"aleph.v1.QueryService.Query", false},
		{"aleph.v1.NLPService.GetSentiment", false},
		{"aleph.v1.ProjectService.GetProject", false},
	}

	for _, tt := range tests {
		t.Run(tt.procedure, func(t *testing.T) {
			assert.Equal(t, tt.expected, isMutatingOperation(tt.procedure))
		})
	}
}

func TestAuditInterceptor_ExtractAuditInfo(t *testing.T) {
	tests := []struct {
		procedure          string
		expectedAction     string
		expectedResource   string
	}{
		{"aleph.v1.AgentService.CreateAgent", "create", "agent"},
		{"aleph.v1.ToolService.UpdateTool", "update", "tool"},
		{"aleph.v1.SkillService.DeleteSkill", "delete", "skill"},
		{"aleph.v1.IngestionService.StartIngestion", "modify", "ingestion"},
		{"aleph.v1.NotificationService.SendNotification", "modify", "notification"},
	}

	for _, tt := range tests {
		t.Run(tt.procedure, func(t *testing.T) {
			action, resourceType, resourceID := extractAuditInfo(tt.procedure, nil, nil)
			assert.Equal(t, tt.expectedAction, action)
			assert.Equal(t, tt.expectedResource, resourceType)
			assert.NotEmpty(t, resourceID)
		})
	}
}