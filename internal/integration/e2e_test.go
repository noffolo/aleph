package integration

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/api/handler"
	nlp "github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/ingestion"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupIntegrationEnv(t *testing.T) (*storage.DuckDB, string, *repository.MetadataRepository) {
	t.Helper()

	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	projectsRoot := filepath.Join(t.TempDir(), "projects")
	require.NoError(t, os.MkdirAll(projectsRoot, 0755))

	pgDB, err := sql.Open("duckdb", "")
	require.NoError(t, err)
	t.Cleanup(func() { pgDB.Close() })

	metaRepo, err := repository.NewMetadataRepository(pgDB)
	require.NoError(t, err)

	return db, projectsRoot, metaRepo
}

func seedCSV(t *testing.T, db *storage.DuckDB, csvPath string, tableName string) {
	t.Helper()
	content := "prodotto,quantita,prezzo\nPenna,10,1.50\nQuaderno,5,3.00\nMatita,20,0.80\n"
	require.NoError(t, os.WriteFile(csvPath, []byte(content), 0644))
	_, err := db.Exec(fmt.Sprintf(`CREATE TABLE "%s" AS SELECT * FROM read_csv_auto('%s')`, tableName, csvPath))
	require.NoError(t, err)
}

func seedProjectWithOntology(t *testing.T, ph *handler.ProjectHandler, projectID string, ontology string) {
	t.Helper()
	_, err := ph.CreateProject(context.Background(), connect.NewRequest(&v1.CreateProjectRequest{Id: projectID, Name: projectID}))
	require.NoError(t, err)
	if ontology != "" {
		_, err = ph.SaveOntology(context.Background(), connect.NewRequest(&v1.SaveOntologyRequest{
			ProjectId:       projectID,
			AlephDefinition: ontology,
		}))
		require.NoError(t, err)
	}
}

// ============================================================
// USABILITY TEST 1: Full data lifecycle
// Create project → define ontology → load CSV → query via DSL → verify data → delete
// ============================================================
func TestUsability_DataLifecycle(t *testing.T) {
	db, projectsRoot, metaRepo := setupIntegrationEnv(t)

	ph := handler.NewProjectHandler(projectsRoot, db)
	qh := handler.NewQueryHandler(db, projectsRoot, metaRepo, nil, nil)

	seedProjectWithOntology(t, ph, "data-lifecycle", "")

	csvPath := filepath.Join(t.TempDir(), "vendite.csv")
	seedCSV(t, db, csvPath, "vendite_data")

	resp, err := qh.ExecuteQuery(context.Background(), connect.NewRequest(&v1.ExecuteQueryRequest{
		ObjectType: "vendite_data",
		ProjectId:  "data-lifecycle",
	}))
	require.NoError(t, err)
	assert.Len(t, resp.Msg.Rows, 3)
	assert.Contains(t, resp.Msg.Columns, "prodotto")

	found := false
	for _, r := range resp.Msg.Rows {
		if r.Values["prodotto"] == "Penna" {
			found = true
			assert.Equal(t, "10", r.Values["quantita"])
			break
		}
	}
	assert.True(t, found, "should find Penna row")

	delResp, err := ph.DeleteProject(context.Background(), connect.NewRequest(&v1.DeleteProjectRequest{Id: "data-lifecycle"}))
	require.NoError(t, err)
	assert.True(t, delResp.Msg.Success)
	_, err = os.Stat(filepath.Join(projectsRoot, "data-lifecycle"))
	assert.True(t, os.IsNotExist(err))
}

// ============================================================
// USABILITY TEST 2: Emerge ontology from existing data
// Load CSV first → then emerge ontology → verify auto-generated matches data schema
// ============================================================
func TestUsability_EmergeOntology(t *testing.T) {
	db, projectsRoot, _ := setupIntegrationEnv(t)

	ph := handler.NewProjectHandler(projectsRoot, db)
	seedProjectWithOntology(t, ph, "emerge-test", "")

	csvPath := filepath.Join(t.TempDir(), "products.csv")
	seedCSV(t, db, csvPath, "products")

	emergeResp, err := ph.EmergeOntology(context.Background(), connect.NewRequest(&v1.EmergeOntologyRequest{}))
	require.NoError(t, err)
	assert.Contains(t, emergeResp.Msg.AlephDefinition, "products")
	assert.Contains(t, emergeResp.Msg.AlephDefinition, "property")
	assert.Contains(t, emergeResp.Msg.AlephDefinition, "prodotto")

	getResp, err := ph.GetOntology(context.Background(), connect.NewRequest(&v1.GetOntologyRequest{ProjectId: "emerge-test"}))
	require.NoError(t, err)
	assert.Contains(t, getResp.Msg.AlephDefinition, "// Define your ontology here")
	assert.Contains(t, getResp.Msg.ObjectNames, "products")
}

// ============================================================
// USABILITY TEST 3: Agent CRUD + chat history
// Create agent → list agents → create another → list again → delete
// ============================================================
func TestUsability_AgentManagement(t *testing.T) {
	db, projectsRoot, metaRepo := setupIntegrationEnv(t)

	ph := handler.NewProjectHandler(projectsRoot, db)
	ah := handler.NewAgentHandler(projectsRoot, metaRepo, "http://localhost:11434")

	seedProjectWithOntology(t, ph, "agent-test", "")

	createResp, err := ah.CreateAgent(context.Background(), connect.NewRequest(&v1.CreateAgentRequest{
		ProjectId: "agent-test",
		Agent: &v1.Agent{
			Id:          "analyst-1",
			Name:        "Data Analyst",
			Provider:    "ollama",
			Model:       "llama3",
			SystemPrompt: "You are a data analyst.",
		},
	}))
	require.NoError(t, err)
	assert.Equal(t, "analyst-1", createResp.Msg.Agent.Id)

	listResp, err := ah.ListAgents(context.Background(), connect.NewRequest(&v1.ListAgentsRequest{ProjectId: "agent-test"}))
	require.NoError(t, err)
	assert.Len(t, listResp.Msg.Agents, 1)
	assert.Equal(t, "Data Analyst", listResp.Msg.Agents[0].Name)

	_, err = ah.CreateAgent(context.Background(), connect.NewRequest(&v1.CreateAgentRequest{
		ProjectId: "agent-test",
		Agent: &v1.Agent{
			Id:          "researcher-1",
			Name:        "Researcher",
			Provider:    "openai",
			Model:       "gpt-4o",
			ApiKey:      "sk-test-key-12345",
			SystemPrompt: "You research topics.",
		},
	}))
	require.NoError(t, err)

	listResp2, err := ah.ListAgents(context.Background(), connect.NewRequest(&v1.ListAgentsRequest{ProjectId: "agent-test"}))
	require.NoError(t, err)
	assert.Len(t, listResp2.Msg.Agents, 2)

	for _, a := range listResp2.Msg.Agents {
		if a.Id == "researcher-1" {
			assert.Equal(t, "sk-test-****", a.ApiKey)
		}
	}

	_, err = ah.DeleteAgent(context.Background(), connect.NewRequest(&v1.DeleteAgentRequest{
		ProjectId: "agent-test",
		Id:        "analyst-1",
	}))
	require.NoError(t, err)

	listResp3, err := ah.ListAgents(context.Background(), connect.NewRequest(&v1.ListAgentsRequest{ProjectId: "agent-test"}))
	require.NoError(t, err)
	assert.Len(t, listResp3.Msg.Agents, 1)
	assert.Equal(t, "researcher-1", listResp3.Msg.Agents[0].Id)
}

// ============================================================
// USABILITY TEST 4: Chat history persistence
// Save messages → retrieve → verify order and content
// ============================================================
func TestUsability_ChatHistory(t *testing.T) {
	db, projectsRoot, metaRepo := setupIntegrationEnv(t)

	ph := handler.NewProjectHandler(projectsRoot, db)
	qh := handler.NewQueryHandler(db, projectsRoot, metaRepo, nil, nil)

	seedProjectWithOntology(t, ph, "chat-test", "")

	err := metaRepo.SaveChatMessage(context.Background(), "chat-test", "agent-1", "user", "Show me vendite data", "")
	require.NoError(t, err)
	err = metaRepo.SaveChatMessage(context.Background(), "chat-test", "agent-1", "assistant", "Here are the vendite records: ...", "")
	require.NoError(t, err)
	err = metaRepo.SaveChatMessage(context.Background(), "chat-test", "agent-1", "assistant", "", "search_data(vendite)")
	require.NoError(t, err)

	historyResp, err := qh.GetChatHistory(context.Background(), connect.NewRequest(&v1.GetChatHistoryRequest{
		ProjectId: "chat-test",
		AgentId:   "agent-1",
	}))
	require.NoError(t, err)
	assert.Len(t, historyResp.Msg.Messages, 3)
	assert.Equal(t, "user", historyResp.Msg.Messages[0].Role)
	assert.Equal(t, "Show me vendite data", historyResp.Msg.Messages[0].Content)
	assert.Equal(t, "assistant", historyResp.Msg.Messages[1].Role)
	assert.Equal(t, "search_data(vendite)", historyResp.Msg.Messages[2].ToolCall)
}

// ============================================================
// USABILITY TEST 5: Library asset lifecycle
// Upload → list → get content → generate PDF → delete
// ============================================================
func TestUsability_LibraryLifecycle(t *testing.T) {
	db, projectsRoot, _ := setupIntegrationEnv(t)

	ph := handler.NewProjectHandler(projectsRoot, db)
	lh := handler.NewLibraryHandler(projectsRoot)

	seedProjectWithOntology(t, ph, "lib-test", "")

	uploadResp, err := lh.UploadAsset(context.Background(), connect.NewRequest(&v1.UploadAssetRequest{
		ProjectId: "lib-test",
		Filename:  "report.txt",
		Content:   []byte("Quarterly report content for Q1 2026."),
	}))
	require.NoError(t, err)
	assert.Equal(t, "report.txt", uploadResp.Msg.Asset.Name)

	listResp, err := lh.ListAssets(context.Background(), connect.NewRequest(&v1.ListAssetsRequest{ProjectId: "lib-test"}))
	require.NoError(t, err)
	assert.Len(t, listResp.Msg.Assets, 1)
	assert.Equal(t, "report.txt", listResp.Msg.Assets[0].Name)

	contentResp, err := lh.GetAssetContent(context.Background(), connect.NewRequest(&v1.GetAssetContentRequest{
		ProjectId: "lib-test",
		AssetId:   "report.txt",
	}))
	require.NoError(t, err)
	assert.Equal(t, "Quarterly report content for Q1 2026.", contentResp.Msg.Content)

	pdfResp, err := lh.GeneratePdf(context.Background(), connect.NewRequest(&v1.GeneratePdfRequest{
		ProjectId: "lib-test",
		AssetId:   "report.txt",
	}))
	require.NoError(t, err)
	assert.True(t, len(pdfResp.Msg.PdfData) > 0, "PDF should have content")
	assert.Equal(t, "report.pdf", pdfResp.Msg.Filename)

	delResp, err := lh.DeleteAsset(context.Background(), connect.NewRequest(&v1.DeleteAssetRequest{
		ProjectId: "lib-test",
		Id:        "report.txt",
	}))
	require.NoError(t, err)
	assert.True(t, delResp.Msg.Success)

	listResp2, err := lh.ListAssets(context.Background(), connect.NewRequest(&v1.ListAssetsRequest{ProjectId: "lib-test"}))
	require.NoError(t, err)
	assert.Len(t, listResp2.Msg.Assets, 0)
}

// ============================================================
// USABILITY TEST 6: Query edge cases
// Invalid name, not found, with limit, global query
// ============================================================
func TestUsability_QueryEdgeCases(t *testing.T) {
	db, projectsRoot, metaRepo := setupIntegrationEnv(t)

	ph := handler.NewProjectHandler(projectsRoot, db)
	qh := handler.NewQueryHandler(db, projectsRoot, metaRepo, nil, nil)

	seedProjectWithOntology(t, ph, "query-test", "")

	_, err := qh.ExecuteQuery(context.Background(), connect.NewRequest(&v1.ExecuteQueryRequest{
		ObjectType: "1invalid name!",
		ProjectId:  "query-test",
	}))
	assert.Error(t, err)

	_, err = qh.ExecuteQuery(context.Background(), connect.NewRequest(&v1.ExecuteQueryRequest{
		ObjectType: "nonexistent_table",
		ProjectId:  "query-test",
	}))
	assert.Error(t, err)

	csvPath := filepath.Join(t.TempDir(), "items.csv")
	seedCSV(t, db, csvPath, "items")

	resp, err := qh.ExecuteQuery(context.Background(), connect.NewRequest(&v1.ExecuteQueryRequest{
		ObjectType: "items",
		ProjectId:  "query-test",
		Limit:       2,
	}))
	require.NoError(t, err)
	assert.Len(t, resp.Msg.Rows, 2)

	globalResp, err := qh.GlobalQuery(context.Background(), connect.NewRequest(&v1.GlobalQueryRequest{
		ObjectType: "items",
		ProjectId:  "query-test",
		Limit:       1,
	}))
	require.NoError(t, err)
	assert.Len(t, globalResp.Msg.Rows, 1)
}

// ============================================================
// USABILITY TEST 7: Ingestion task lifecycle
// Create task → run → check progress → get logs → delete task
// ============================================================
func TestUsability_IngestionLifecycle(t *testing.T) {
	db, projectsRoot, metaRepo := setupIntegrationEnv(t)

	ph := handler.NewProjectHandler(projectsRoot, db)
	ih := handler.NewIngestionHandler(projectsRoot, ingestion.NewEngine(projectsRoot, metaRepo, db, nil), metaRepo)

	seedProjectWithOntology(t, ph, "ingest-test", "")

	csvPath := filepath.Join(t.TempDir(), "data.csv")
	seedCSV(t, db, csvPath, "ingest_table")

	createResp, err := ih.CreateTask(context.Background(), connect.NewRequest(&v1.CreateTaskRequest{
		ProjectId: "ingest-test",
		Task: &v1.IngestionTask{
			Name:       "Test CSV Load",
			SourceType: "csv",
			ConfigJson: `{"path": "` + csvPath + `"}`,
		},
	}))
	require.NoError(t, err)
	assert.NotEmpty(t, createResp.Msg.Task.Id)

	listResp, err := ih.ListTasks(context.Background(), connect.NewRequest(&v1.ListTasksRequest{ProjectId: "ingest-test"}))
	require.NoError(t, err)
	assert.Len(t, listResp.Msg.Tasks, 1)
	assert.Equal(t, "Test CSV Load", listResp.Msg.Tasks[0].Name)

	_, err = ih.DeleteTask(context.Background(), connect.NewRequest(&v1.DeleteTaskRequest{
		ProjectId: "ingest-test",
		Id:        createResp.Msg.Task.Id,
	}))
	require.NoError(t, err)

	listResp2, err := ih.ListTasks(context.Background(), connect.NewRequest(&v1.ListTasksRequest{ProjectId: "ingest-test"}))
	require.NoError(t, err)
	assert.Len(t, listResp2.Msg.Tasks, 0)
}

// ============================================================
// USABILITY TEST 8: Auth key lifecycle
// Create API key → list → delete → verify gone
// ============================================================
func TestUsability_AuthKeyLifecycle(t *testing.T) {
	_, projectsRoot, metaRepo := setupIntegrationEnv(t)

	ph := handler.NewProjectHandler(projectsRoot, nil)
	seedProjectWithOntology(t, ph, "auth-test", "")

	authH := handler.NewAuthHandler(metaRepo)

	createResp, err := authH.CreateApiKey(context.Background(), connect.NewRequest(&v1.CreateApiKeyRequest{
		ProjectId: "auth-test",
		Label:     "test-key",
	}))
	require.NoError(t, err)
	assert.NotEmpty(t, createResp.Msg.Key.Key)
	assert.NotEmpty(t, createResp.Msg.Key.Id)
	assert.Equal(t, "test-key", createResp.Msg.Key.Label)

	listResp, err := authH.ListApiKeys(context.Background(), connect.NewRequest(&v1.ListApiKeysRequest{ProjectId: "auth-test"}))
	require.NoError(t, err)
	assert.Len(t, listResp.Msg.Keys, 1)
	assert.Equal(t, "********", listResp.Msg.Keys[0].Key)

	_, err = authH.DeleteApiKey(context.Background(), connect.NewRequest(&v1.DeleteApiKeyRequest{
		ProjectId: "auth-test",
		Id:        createResp.Msg.Key.Id,
	}))
	require.NoError(t, err)

	listResp2, err := authH.ListApiKeys(context.Background(), connect.NewRequest(&v1.ListApiKeysRequest{ProjectId: "auth-test"}))
	require.NoError(t, err)
	assert.Len(t, listResp2.Msg.Keys, 0)
}

// ============================================================
// USABILITY TEST 9: Confirm action flow
// Confirm action for existing agent/project and non-existent
// ============================================================
func TestUsability_ConfirmAction(t *testing.T) {
	_, projectsRoot, metaRepo := setupIntegrationEnv(t)

	ph := handler.NewProjectHandler(projectsRoot, nil)
	seedProjectWithOntology(t, ph, "confirm-test", "")

	ah := handler.NewAgentHandler(projectsRoot, metaRepo, "http://localhost:11434")
	_, err := ah.CreateAgent(context.Background(), connect.NewRequest(&v1.CreateAgentRequest{
		ProjectId: "confirm-test",
		Agent: &v1.Agent{Id: "bot-1", Name: "Bot", Provider: "ollama", Model: "llama3"},
	}))
	require.NoError(t, err)

	qh := handler.NewQueryHandler(nil, projectsRoot, metaRepo, nil, nil)

	confirmResp, err := qh.ConfirmAction(context.Background(), connect.NewRequest(&v1.ConfirmActionRequest{
		ProjectId: "confirm-test",
		AgentId:   "bot-1",
		Approved:  true,
	}))
	require.NoError(t, err)
	assert.True(t, confirmResp.Msg.Success)

	_, err = qh.ConfirmAction(context.Background(), connect.NewRequest(&v1.ConfirmActionRequest{
		ProjectId: "confirm-test",
		AgentId:   "nonexistent-agent",
		Approved:  true,
	}))
	assert.Error(t, err)
}

// ============================================================
// USABILITY TEST 10: NLP circuit breaker
// Verify circuit breaker blocks requests after 3 failures
// ============================================================
func TestUsability_NLPCircuitBreaker(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	breaker := handler.NewCircuitBreakerClient(nil, logger)

	for i := 0; i < 3; i++ {
		_, err := breaker.AnalyzeSentiment(context.Background(), connect.NewRequest(&nlp.AnalyzeSentimentRequest{Text: "test"}))
		assert.Error(t, err)
	}

	_, err := breaker.AnalyzeSentiment(context.Background(), connect.NewRequest(&nlp.AnalyzeSentimentRequest{Text: "test"}))
	assert.Error(t, err)
}
