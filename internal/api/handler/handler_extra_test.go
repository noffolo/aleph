package handler

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/registry"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/sandbox"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryHelpers(t *testing.T) {
	assert.Equal(t, "hello", derefStr(strPtr("hello")))
	assert.Equal(t, "", derefStr(nil))
	assert.Equal(t, 3.14, derefFloat64(float64Ptr(3.14)))
	assert.Equal(t, 0.0, derefFloat64(nil))
	assert.InDelta(t, 2.71, derefFloat32(float32Ptr(2.71)), 0.001)
	assert.Equal(t, 0.0, derefFloat32(nil))
}

func TestProtoFromMeta(t *testing.T) {
	now := time.Now()
	meta := registry.ComponentMetadata{
		ID: "c1", Name: "comp", Description: "desc",
		Version: "v1", Type: "agent", Category: "ai",
		Source: "test", Status: "active", ApprovalStatus: "approved",
		ConfigSchemaJSON:     `{"type":"object"}`,
		ExecutionCommand:     "echo hi",
		DependenciesJSON:     `[]`,
		InputSchemaJSON:      `{"type":"object"}`,
		OutputSchemaJSON:     `{"type":"object"}`,
		PromptTemplate:       "{{.input}}",
		ToolIdsJSON:          `["t1"]`,
		AvgCpuUsage:          10.5,
		AvgMemoryMb:          256.0,
		AvgExecTimeMs:        100.0,
		AvgBrierScore:        0.25,
		AvgLatencyMs:         50.0,
		TrustScore:           0.9,
		CreatedByAgentId:     "a1",
		CreationTimestamp:    now,
		LastUpdatedTimestamp: now,
	}

	pb := protoFromMeta(meta)
	assert.Equal(t, "c1", pb.Id)
	assert.Equal(t, "comp", pb.Name)
	assert.Equal(t, "desc", pb.Description)
	assert.Equal(t, "v1", pb.Version)
	assert.Equal(t, "agent", pb.Type)
	assert.Equal(t, "ai", pb.Category)
	assert.Equal(t, "test", pb.Source)
	assert.Equal(t, "active", pb.Status)
	assert.Equal(t, "approved", pb.ApprovalStatus)
	assert.Equal(t, `{"type":"object"}`, *pb.ConfigSchemaJson)
	assert.Equal(t, "echo hi", *pb.ExecutionCommand)
	assert.Equal(t, `[]`, *pb.DependenciesJson)
	assert.Equal(t, `{"type":"object"}`, *pb.InputSchemaJson)
	assert.Equal(t, `{"type":"object"}`, *pb.OutputSchemaJson)
	assert.Equal(t, "{{.input}}", *pb.PromptTemplate)
	assert.Equal(t, `["t1"]`, *pb.ToolIdsJson)
	assert.Equal(t, 10.5, *pb.AvgCpuUsage)
	assert.Equal(t, 256.0, *pb.AvgMemoryMb)
	assert.Equal(t, 100.0, *pb.AvgExecTimeMs)
	assert.Equal(t, 0.25, *pb.AvgBrierScore)
	assert.Equal(t, 50.0, *pb.AvgLatencyMs)
	assert.Equal(t, float32(0.9), *pb.TrustScore)
	assert.Equal(t, "a1", *pb.CreatedByAgentId)
	assert.NotNil(t, pb.CreationTimestamp)
	assert.NotNil(t, pb.LastUpdatedTimestamp)
}

func TestRegistryHandler_Constructors(t *testing.T) {
	h := NewRegistryServiceHandler(nil, nil)
	assert.NotNil(t, h)
}

func setupMetaRepo(t *testing.T) *repository.MetadataRepository {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`CREATE TABLE system_tools (id TEXT PRIMARY KEY, name TEXT, description TEXT, code TEXT, category TEXT DEFAULT '', version TEXT DEFAULT '', health_status TEXT DEFAULT 'unknown', last_checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, source_type TEXT DEFAULT 'builtin')`)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE system_skills (id TEXT PRIMARY KEY, project_id TEXT, name TEXT, description TEXT, tool_ids TEXT)`)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE system_agents (id TEXT PRIMARY KEY, project_id TEXT, name TEXT, provider TEXT, model TEXT, api_key TEXT, system_prompt TEXT, skill_ids TEXT, base_url TEXT)`)
	require.NoError(t, err)

	repo, err := repository.NewMetadataRepository(db)
	require.NoError(t, err)
	return repo
}

func TestToolHandler(t *testing.T) {
	repo := setupMetaRepo(t)
	th := NewToolHandler("/tmp", repo)

	ctx := context.Background()
	_, err := th.CreateTool(ctx, connect.NewRequest(&v1.CreateToolRequest{Tool: &v1.Tool{Id: "t1", Name: "hammer", Description: "hits things", Code: "return 1"}}))
	assert.NoError(t, err)

	list, err := th.ListTools(ctx, connect.NewRequest(&v1.ListToolsRequest{}))
	assert.NoError(t, err)
	assert.Len(t, list.Msg.Tools, 1)
	assert.Equal(t, "hammer", list.Msg.Tools[0].Name)

	_, err = th.DeleteTool(ctx, connect.NewRequest(&v1.DeleteToolRequest{Id: "t1"}))
	assert.NoError(t, err)
}

func TestSkillHandler(t *testing.T) {
	repo := setupMetaRepo(t)
	sh := NewSkillHandler("/tmp", repo)

	ctx := context.Background()
	_, err := sh.CreateSkill(ctx, connect.NewRequest(&v1.CreateSkillRequest{ProjectId: "p1", Skill: &v1.Skill{Id: "s1", Name: "read", Description: "reads data", ToolIds: []string{"t1"}}}))
	assert.NoError(t, err)

	list, err := sh.ListSkills(ctx, connect.NewRequest(&v1.ListSkillsRequest{ProjectId: "p1"}))
	assert.NoError(t, err)
	assert.Len(t, list.Msg.Skills, 1)
	assert.Equal(t, "read", list.Msg.Skills[0].Name)

	_, err = sh.DeleteSkill(ctx, connect.NewRequest(&v1.DeleteSkillRequest{Id: "s1", ProjectId: "p1"}))
	assert.NoError(t, err)
}

type mockSandboxMgr struct{}

func (m *mockSandboxMgr) ExecuteTool(ctx context.Context, toolID string, input map[string]any) (sandbox.ExecutionResult, error) {
	return sandbox.ExecutionResult{Stdout: "out", ExitCode: 0}, nil
}
func (m *mockSandboxMgr) RunSkill(ctx context.Context, skillID string, input map[string]any) (sandbox.ExecutionResult, error) {
	return sandbox.ExecutionResult{Stdout: "skill-out", ExitCode: 0}, nil
}

func TestSandboxHandler(t *testing.T) {
	mgr := &mockSandboxMgr{}
	h := NewSandboxServiceHandler(mgr, nil)

	ctx := context.Background()
	res, err := h.ExecuteTool(ctx, connect.NewRequest(&v1.ExecuteToolRequest{ToolId: "t1"}))
	assert.NoError(t, err)
	assert.Equal(t, "out", res.Msg.Result.Stdout)
	assert.Equal(t, int32(0), res.Msg.Result.ExitCode)

	res2, err := h.RunSkill(ctx, connect.NewRequest(&v1.RunSkillRequest{SkillId: "s1"}))
	assert.NoError(t, err)
	assert.Equal(t, "skill-out", res2.Msg.Result.Stdout)
}
