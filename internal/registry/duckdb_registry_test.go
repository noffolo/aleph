package registry

import (
	"context"
	"log/slog"
	"testing"
)

var testCtx = context.Background()

func setupRegistry(t *testing.T) *DuckDBRegistry {
	t.Helper()
	r, err := NewDuckDBRegistry(":memory:", slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	// :memory: DuckDB needs explicit table creation (no migrations run)
	const ddl = `CREATE TABLE IF NOT EXISTS components (
		id TEXT PRIMARY KEY, name TEXT, description TEXT, version TEXT, type TEXT,
		category TEXT, source TEXT, status TEXT, approval_status TEXT,
		config_schema_json TEXT, execution_command TEXT, dependencies_json TEXT,
		input_schema_json TEXT, output_schema_json TEXT, prompt_template TEXT,
		tool_ids_json TEXT, avg_cpu_usage DOUBLE DEFAULT 0, avg_memory_mb DOUBLE DEFAULT 0,
		avg_exec_time_ms DOUBLE DEFAULT 0, avg_brier_score DOUBLE DEFAULT 0,
		avg_latency_ms DOUBLE DEFAULT 0, trust_score DOUBLE DEFAULT 0,
		created_by_agent_id TEXT, creation_timestamp TIMESTAMP, last_updated_timestamp TIMESTAMP)`
	if _, err := r.db.Exec(ddl); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestDuckDBRegistry_RegisterAndList(t *testing.T) {
	r := setupRegistry(t)

	id, err := r.RegisterComponent(ComponentMetadata{
		Name:        "test-tool",
		Type:        "tool",
		Description: "A test tool",
		Version:     "1.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Error("expected non-empty id")
	}

	comps, err := r.ListComponents(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
	if comps[0].Name != "test-tool" {
		t.Errorf("expected name test-tool, got %s", comps[0].Name)
	}
}

func TestDuckDBRegistry_GetComponentByID(t *testing.T) {
	r := setupRegistry(t)

	id, _ := r.RegisterComponent(ComponentMetadata{
		Name:   "my-skill",
		Type:   "skill",
		Status: "active",
	})

	meta, err := r.GetComponentByID(testCtx, id)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Name != "my-skill" {
		t.Errorf("expected my-skill, got %s", meta.Name)
	}
	if meta.Status != "active" {
		t.Errorf("expected active, got %s", meta.Status)
	}
}

func TestDuckDBRegistry_GetComponentByID_NotFound(t *testing.T) {
	r := setupRegistry(t)

	_, err := r.GetComponentByID(testCtx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent id")
	}
}

func TestParseToolIdsJSON(t *testing.T) {
	tests := []struct {
		name string
		json string
		want []string
	}{
		{"empty_string", "", nil},
		{"empty_array", "[]", []string{}},
		{"single_id", `["tool-1"]`, []string{"tool-1"}},
		{"multiple_ids", `["tool-1","tool-2","tool-3"]`, []string{"tool-1", "tool-2", "tool-3"}},
		{"invalid_json", "{malformed", nil},
		{"not_array", `{"key":"value"}`, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseToolIdsJSON(tt.json)
			if len(got) != len(tt.want) {
				t.Errorf("ParseToolIdsJSON(%q) len=%d, want len=%d", tt.json, len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("ParseToolIdsJSON(%q)[%d] = %q, want %q", tt.json, i, v, tt.want[i])
				}
			}
		})
	}
}

func TestDuckDBRegistry_UpdateComponentStatus(t *testing.T) {
	r := setupRegistry(t)

	id, _ := r.RegisterComponent(ComponentMetadata{Name: "test", Type: "tool"})

	err := r.UpdateComponentStatus(id, "approved")
	if err != nil {
		t.Fatal(err)
	}

	meta, _ := r.GetComponentByID(testCtx, id)
	if meta.Status != "approved" {
		t.Errorf("expected approved, got %s", meta.Status)
	}
}

func TestDuckDBRegistry_DeleteComponent(t *testing.T) {
	r := setupRegistry(t)

	id, err := r.RegisterComponent(ComponentMetadata{
		Name: "delete-me",
		Type: "tool",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify it exists
	_, err = r.GetComponentByID(testCtx, id)
	if err != nil {
		t.Fatal("expected component to exist before delete:", err)
	}

	// Delete it
	err = r.DeleteComponent(testCtx, id)
	if err != nil {
		t.Fatal(err)
	}

	// Verify it's gone
	_, err = r.GetComponentByID(testCtx, id)
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

func TestDuckDBRegistry_DeleteComponent_NotFound(t *testing.T) {
	r := setupRegistry(t)

	err := r.DeleteComponent(testCtx, "nonexistent-id")
	if err == nil {
		t.Error("expected error for nonexistent id, got nil")
	}
}

func TestDuckDBRegistry_UpdateComponent(t *testing.T) {
	r := setupRegistry(t)

	id, err := r.RegisterComponent(ComponentMetadata{
		Name:        "original",
		Description: "original desc",
		Version:     "1.0",
		Type:        "tool",
		Category:    "finance",
		Source:      "builtin",
		Status:      "active",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Capture original timestamp
	orig, err := r.GetComponentByID(testCtx, id)
	if err != nil {
		t.Fatal(err)
	}
	origTS := orig.LastUpdatedTimestamp

	// Update ALL fields
	updated := ComponentMetadata{
		ID:                 id,
		Name:               "updated-name",
		Description:        "updated desc",
		Version:            "2.0",
		Type:               "skill",
		Category:           "osint",
		Source:             "external",
		Status:             "deprecated",
		ApprovalStatus:     "approved",
		ConfigSchemaJSON:   `{"key":"val"}`,
		ExecutionCommand:   "./run.sh",
		DependenciesJSON:   `["dep1"]`,
		InputSchemaJSON:    `{"input":true}`,
		OutputSchemaJSON:   `{"output":true}`,
		PromptTemplate:     "hello {{.name}}",
		ToolIdsJSON:        `["t1","t2"]`,
		AvgCpuUsage:        12.5,
		AvgMemoryMb:        256.0,
		AvgExecTimeMs:      150.0,
		AvgBrierScore:      0.05,
		AvgLatencyMs:       45.0,
		TrustScore:         0.99,
		CreatedByAgentId:   "agent-42",
	}

	err = r.UpdateComponent(testCtx, updated)
	if err != nil {
		t.Fatal(err)
	}

	// Fetch and verify all fields
	got, err := r.GetComponentByID(testCtx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != updated.Name {
		t.Errorf("Name: got %q, want %q", got.Name, updated.Name)
	}
	if got.Description != updated.Description {
		t.Errorf("Description: got %q, want %q", got.Description, updated.Description)
	}
	if got.Version != updated.Version {
		t.Errorf("Version: got %q, want %q", got.Version, updated.Version)
	}
	if got.Type != updated.Type {
		t.Errorf("Type: got %q, want %q", got.Type, updated.Type)
	}
	if got.Category != updated.Category {
		t.Errorf("Category: got %q, want %q", got.Category, updated.Category)
	}
	if got.Source != updated.Source {
		t.Errorf("Source: got %q, want %q", got.Source, updated.Source)
	}
	if got.Status != updated.Status {
		t.Errorf("Status: got %q, want %q", got.Status, updated.Status)
	}
	if got.ApprovalStatus != updated.ApprovalStatus {
		t.Errorf("ApprovalStatus: got %q, want %q", got.ApprovalStatus, updated.ApprovalStatus)
	}
	if got.ConfigSchemaJSON != updated.ConfigSchemaJSON {
		t.Errorf("ConfigSchemaJSON: got %q, want %q", got.ConfigSchemaJSON, updated.ConfigSchemaJSON)
	}
	if got.ExecutionCommand != updated.ExecutionCommand {
		t.Errorf("ExecutionCommand: got %q, want %q", got.ExecutionCommand, updated.ExecutionCommand)
	}
	if got.DependenciesJSON != updated.DependenciesJSON {
		t.Errorf("DependenciesJSON: got %q, want %q", got.DependenciesJSON, updated.DependenciesJSON)
	}
	if got.InputSchemaJSON != updated.InputSchemaJSON {
		t.Errorf("InputSchemaJSON: got %q, want %q", got.InputSchemaJSON, updated.InputSchemaJSON)
	}
	if got.OutputSchemaJSON != updated.OutputSchemaJSON {
		t.Errorf("OutputSchemaJSON: got %q, want %q", got.OutputSchemaJSON, updated.OutputSchemaJSON)
	}
	if got.PromptTemplate != updated.PromptTemplate {
		t.Errorf("PromptTemplate: got %q, want %q", got.PromptTemplate, updated.PromptTemplate)
	}
	if got.ToolIdsJSON != updated.ToolIdsJSON {
		t.Errorf("ToolIdsJSON: got %q, want %q", got.ToolIdsJSON, updated.ToolIdsJSON)
	}
	if got.AvgCpuUsage != updated.AvgCpuUsage {
		t.Errorf("AvgCpuUsage: got %f, want %f", got.AvgCpuUsage, updated.AvgCpuUsage)
	}
	if got.AvgMemoryMb != updated.AvgMemoryMb {
		t.Errorf("AvgMemoryMb: got %f, want %f", got.AvgMemoryMb, updated.AvgMemoryMb)
	}
	if got.AvgExecTimeMs != updated.AvgExecTimeMs {
		t.Errorf("AvgExecTimeMs: got %f, want %f", got.AvgExecTimeMs, updated.AvgExecTimeMs)
	}
	if got.AvgBrierScore != updated.AvgBrierScore {
		t.Errorf("AvgBrierScore: got %f, want %f", got.AvgBrierScore, updated.AvgBrierScore)
	}
	if got.AvgLatencyMs != updated.AvgLatencyMs {
		t.Errorf("AvgLatencyMs: got %f, want %f", got.AvgLatencyMs, updated.AvgLatencyMs)
	}
	if got.TrustScore != updated.TrustScore {
		t.Errorf("TrustScore: got %f, want %f", got.TrustScore, updated.TrustScore)
	}
	if got.CreatedByAgentId != updated.CreatedByAgentId {
		t.Errorf("CreatedByAgentId: got %q, want %q", got.CreatedByAgentId, updated.CreatedByAgentId)
	}

	// Verify timestamp was updated
	if !got.LastUpdatedTimestamp.After(origTS) {
		t.Error("expected LastUpdatedTimestamp to be updated")
	}
}

func TestDuckDBRegistry_UpdateComponent_NotFound(t *testing.T) {
	r := setupRegistry(t)

	err := r.UpdateComponent(testCtx, ComponentMetadata{
		ID:   "nonexistent-id",
		Name: "ghost",
	})
	if err == nil {
		t.Error("expected error for nonexistent id, got nil")
	}
}

func TestDuckDBRegistry_UpdateComponent_EmptyID(t *testing.T) {
	r := setupRegistry(t)

	err := r.UpdateComponent(testCtx, ComponentMetadata{
		Name: "no-id",
	})
	if err == nil {
		t.Error("expected error for empty id, got nil")
	}
}

// TestRegistryInterface verifies that *DuckDBRegistry satisfies the Registry interface
func TestRegistryInterface(t *testing.T) {
	var _ Registry = (*DuckDBRegistry)(nil)
}
