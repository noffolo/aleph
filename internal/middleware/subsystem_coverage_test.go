package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeriveSubsystem(t *testing.T) {
	t.Parallel()

	cases := []struct {
		procedure         string
		expectedSubsystem string
		expectedOperation string
	}{
		{"/aleph.v1.QueryService/ExecuteQuery", "handler", "query"},
		{"/aleph.v1.QueryService/Chat", "handler", "chat"},
		{"/aleph.v1.ProjectService/CreateProject", "handler", "insert"},
		{"/aleph.v1.AgentService/ListAgents", "handler", "query"},
		{"/aleph.v1.SkillService/DeleteSkill", "handler", "delete"},
		{"/aleph.v1.ToolService/InstallTool", "handler", "insert"},
		{"/aleph.v1.LibraryService/GetAssetContent", "handler", "getassetcontent"},
		{"/aleph.v1.NotificationService/SendWebhook", "handler", "sendwebhook"},
		{"/aleph.v1.AuthService/Login", "handler", "login"},
		{"/aleph.v1.IngestionService/IngestFromRSS", "ingestion", "insert"},
		{"/aleph.v1.IngestionService/IngestFromGitHub", "ingestion", "insert"},
		{"/aleph.v1.SandboxService/ExecuteTool", "sandbox", "execute"},
		{"/aleph.registry.v1.RegistryService/RegisterComponent", "handler", "insert"},
		{"/aleph.nlp.v1.NLPService/AnalyzeSentiment", "nlp", "execute"},
		{"/aleph.nlp.v1.NLPService/StreamPredictions", "nlp", "execute"},
		{"/unknown.Service/UnknownMethod", "", "unknownmethod"},
		{"/aleph.v1.IngestionService/GetIngestionStatus", "ingestion", "query"},
		{"/aleph.v1.IngestionService/ListIngestions", "ingestion", "query"},
		{"/aleph.v1.SandboxService/RunSkill", "sandbox", "runskill"},
	}
	for _, tc := range cases {
		t.Run(tc.procedure, func(t *testing.T) {
			subsystem, operation := deriveSubsystem(tc.procedure)
			assert.Equal(t, tc.expectedSubsystem, subsystem)
			assert.Equal(t, tc.expectedOperation, operation)
		})
	}
}

func TestDeriveSubsystem_NoSlash(t *testing.T) {
	t.Parallel()
	subsystem, operation := deriveSubsystem("noSlash")
	assert.Empty(t, subsystem)
	assert.Empty(t, operation)
}

func TestNewSubsystemInterceptor(t *testing.T) {
	t.Parallel()
	si := NewSubsystemInterceptor()
	assert.NotNil(t, si)
}
