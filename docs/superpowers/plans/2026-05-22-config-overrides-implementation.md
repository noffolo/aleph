# Config Overrides Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `config_overrides` JSON merge field to RunTaskRequest so users can override any configJson parameter (start_date, end_date, etc.) at task execution time.

**Architecture:** Add `optional string config_overrides = 3` to the RunTaskRequest proto. In the backend RunTask handler, shallow-merge config_overrides on top of stored configJson before passing to Engine.RunTask. The frontend RunTask dialog gets date input fields that populate config_overrides.

**Tech Stack:** Protobuf (buf), Go 1.26 (connectrpc), TypeScript/React (protobuf-es)

---

### Task 1: Proto + Regenerate Code

**Files:**
- Modify: `api/proto/aleph/v1/query.proto:140`
- Auto-generated: `internal/api/proto/aleph/v1/query.pb.go`
- Auto-generated: `frontend/src/api/proto/aleph/v1/query_pb.ts`
- Auto-generated: `frontend/src/api/proto/aleph/v1/query_connect.ts`

- [ ] **Step 1: Add config_overrides field to RunTaskRequest proto**

Replace line 140 in `api/proto/aleph/v1/query.proto`:
```protobuf
message RunTaskRequest { string project_id = 1; string task_id = 2; }
```
With:
```protobuf
message RunTaskRequest {
  string project_id = 1;
  string task_id = 2;
  // Optional JSON object with key-value overrides for the task's configJson.
  // Shallow merge: top-level keys replace corresponding keys in stored configJson.
  // Nested objects are replaced wholesale, not deep-merged.
  // Example: {"start_date":"2024-01-01","end_date":"2024-12-31"}
  optional string config_overrides = 3;
}
```

- [ ] **Step 2: Regenerate Go protobuf code**

```bash
cd /tmp/opencode/aleph && buf generate api/proto/aleph/v1/query.proto
```
Expected: regenerates `internal/api/proto/aleph/v1/query.pb.go` with new `ConfigOverrides` field.

If `buf generate` fails, manually add the field to the Go struct:

```go
type RunTaskRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ProjectId       string `protobuf:"bytes,1,opt,name=project_id,json=projectId,proto3" json:"project_id,omitempty"`
	TaskId          string `protobuf:"bytes,2,opt,name=task_id,json=taskId,proto3" json:"task_id,omitempty"`
	ConfigOverrides *string `protobuf:"bytes,3,opt,name=config_overrides,json=configOverrides,proto3,oneof" json:"config_overrides,omitempty"`
}
```

And add getter/setter:
```go
func (x *RunTaskRequest) GetConfigOverrides() string {
	if x != nil && x.ConfigOverrides != nil {
		return *x.ConfigOverrides
	}
	return ""
}
```

Update `Fields()` list, `Size()`, `Marshal()`, `Unmarshal()` accordingly, OR just use `buf generate`.

- [ ] **Step 3: Regenerate TypeScript protobuf code**

```bash
cd /tmp/opencode/aleph/frontend && npx buf generate ../api/proto/aleph/v1/query.proto
```

Expected: regenerates `query_pb.ts` and `query_connect.ts` with new `configOverrides` field.

If buf generate fails, manually add to `RunTaskRequest` class in `query_pb.ts`:

```typescript
export class RunTaskRequest extends Message<RunTaskRequest> {
  projectId = "";
  taskId = "";
  configOverrides?: string;

  constructor(data?: PartialMessage<RunTaskRequest>) {
    super();
    proto3.util.initPartial(data, this);
  }

  static readonly fields: FieldList = proto3.util.newFieldList(() => [
    { no: 1, name: "project_id", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 2, name: "task_id", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 3, name: "config_overrides", kind: "scalar", T: 9 /* ScalarType.STRING */, opt: true },
  ]);
}
```

- [ ] **Step 4: Verify compilation**

```bash
cd /tmp/opencode/aleph && go build ./...
cd /tmp/opencode/aleph/frontend && npx tsc --noEmit
```

Expected: both compile clean.

- [ ] **Step 5: Commit**

```bash
cd /tmp/opencode/aleph && git add api/proto/aleph/v1/query.proto internal/api/proto/ frontend/src/api/proto/
git commit -m "feat(proto): add config_overrides field to RunTaskRequest"
```

---

### Task 2: Backend — Shallow Merge in RunTask Handler

**Files:**
- Modify: `internal/api/handler/ingestion.go:88-118`

- [ ] **Step 6: Add merge logic in RunTask handler**

In `internal/api/handler/ingestion.go`, update the `RunTask` function at line 109 to merge config_overrides before passing to engine:

```go
import (
	"encoding/json"
	// ...existing imports
)

func (h *IngestionHandler) RunTask(
	ctx context.Context,
	req *connect.Request[v1.RunTaskRequest],
) (*connect.Response[v1.RunTaskResponse], error) {
	projectID := req.Msg.ProjectId
	taskID := req.Msg.TaskId

	t, err := h.metaRepo.GetTaskByID(taskID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.NewAPIErrorWithMeta(
			errors.ErrNotFound, "ingestion task not found", err,
			"ingestion", "query", false, 0,
		))
	}

	// Apply config_overrides (shallow merge) if provided
	configJSON := t.ConfigJSON
	if overrides := req.Msg.ConfigOverrides; overrides != "" {
		merged, err := mergeConfigOverrides(configJSON, overrides)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("invalid config_overrides: %w", err))
		}
		configJSON = merged
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("ingestion RunTask goroutine panic", "projectID", projectID, "taskID", t.ID, "recover", r)
			}
		}()
		v1Task := &v1.IngestionTask{Id: t.ID, Name: t.Name, SourceType: t.SourceType, ConfigJson: configJSON}
		taskCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
		defer cancel()
		if err := h.engine.RunTask(taskCtx, projectID, v1Task); err != nil {
			slog.Error("ingestion task failed", "projectID", projectID, "taskID", v1Task.Id, "error", err)
		}
	}()

	return connect.NewResponse(&v1.RunTaskResponse{Status: "started"}), nil
}

// mergeConfigOverrides shallow-merges override JSON on top of base JSON.
// Top-level keys in overrides replace corresponding keys in base.
// Nested objects are replaced wholesale, not deep-merged.
func mergeConfigOverrides(baseJSON, overridesJSON string) (string, error) {
	var base map[string]any
	if err := json.Unmarshal([]byte(baseJSON), &base); err != nil {
		return "", fmt.Errorf("failed to parse stored config: %w", err)
	}
	// If base is nil (empty/empty string), init empty map
	if base == nil {
		base = make(map[string]any)
	}

	var overrides map[string]any
	if err := json.Unmarshal([]byte(overridesJSON), &overrides); err != nil {
		return "", fmt.Errorf("failed to parse config_overrides: %w", err)
	}

	for k, v := range overrides {
		base[k] = v
	}

	result, err := json.Marshal(base)
	if err != nil {
		return "", fmt.Errorf("failed to marshal merged config: %w", err)
	}

	return string(result), nil
}
```

- [ ] **Step 7: Write unit tests for mergeConfigOverrides**

Create or modify `internal/api/handler/ingestion_handler_test.go`:

```go
func TestMergeConfigOverrides(t *testing.T) {
	tests := []struct {
		name      string
		base      string
		overrides string
		want      string
		wantErr   bool
	}{
		{
			name:      "merge start_date override",
			base:      `{"url":"https://example.com/sitemap.xml"}`,
			overrides: `{"start_date":"2024-01-01"}`,
			want:      `{"url":"https://example.com/sitemap.xml","start_date":"2024-01-01"}`,
		},
		{
			name:      "override replaces existing key",
			base:      `{"url":"https://example.com","start_date":"2023-01-01"}`,
			overrides: `{"start_date":"2024-06-01"}`,
			want:      `{"url":"https://example.com","start_date":"2024-06-01"}`,
		},
		{
			name:      "empty base with overrides",
			base:      `{}`,
			overrides: `{"start_date":"2024-01-01","end_date":"2024-12-31"}`,
			want:      `{"start_date":"2024-01-01","end_date":"2024-12-31"}`,
		},
		{
			name:      "no overrides returns base unchanged",
			base:      `{"url":"https://example.com"}`,
			overrides: `{}`,
			want:      `{"url":"https://example.com"}`,
		},
		{
			name:      "multiple keys overridden",
			base:      `{"url":"https://example.com","max_articles":"50"}`,
			overrides: `{"start_date":"2024-01-01","end_date":"2024-12-31","max_articles":"100"}`,
			want:      `{"url":"https://example.com","max_articles":"100","start_date":"2024-01-01","end_date":"2024-12-31"}`,
		},
		{
			name:      "invalid overrides JSON errors",
			base:      `{}`,
			overrides: `{invalid}`,
			wantErr:   true,
		},
		{
			name:      "empty string base",
			base:      ``,
			overrides: `{"start_date":"2024-01-01"}`,
			want:      `{"start_date":"2024-01-01"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeConfigOverrides(tt.base, tt.overrides)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.JSONEq(t, tt.want, got)
		})
	}
}
```

Add import:
```go
import (
	"github.com/stretchr/testify/require"
)
```

- [ ] **Step 8: Run tests**

```bash
cd /tmp/opencode/aleph && go test ./internal/api/handler/ -run TestMergeConfigOverrides -v
```
Expected: 7/7 PASS

Run all handler tests:
```bash
cd /tmp/opencode/aleph && go test ./internal/api/handler/ -count=1 -v 2>&1 | tail -20
```
Expected: all pass (note: some pre-existing failures in unrelated tests are OK)

- [ ] **Step 9: Run vet + build**

```bash
cd /tmp/opencode/aleph && go vet ./internal/api/handler/ && go build ./...
```
Expected: clean

- [ ] **Step 10: Commit**

```bash
cd /tmp/opencode/aleph && git add internal/api/handler/ingestion.go internal/api/handler/ingestion_handler_test.go
git commit -m "feat(ingestion): add config_overrides shallow merge to RunTask handler"
```

---

### Task 3: Frontend — Date Inputs in RunTask Dialog

**Files:**
- Modify: `frontend/src/hooks/domain/useDataSourceActions.ts`
- Modify or locate: the component that triggers runTask (find where `runTask` is called from the UI)

- [ ] **Step 11: Find and read the RunTask UI component**

```bash
grep -rn "runTask\|RunTask\|run.*task\|execute.*ingestion" /tmp/opencode/aleph/frontend/src/components/ --include="*.tsx" --include="*.ts" | grep -v node_modules | head -20
```

Expected: identifies which component renders the "Run" action button and handles the execution dialog.

- [ ] **Step 12: Add date fields to the RunTask dialog**

Add `start_date`/`end_date` date inputs to the UI that calls `runTask`. When the user fills them in, create a `config_overrides` JSON string with `{"start_date":"YYYY-MM-DD","end_date":"YYYY-MM-DD"}` and pass it to the `runTask` call.

The exact UI component location depends on what's found in Step 11. The pattern should be:

```tsx
// Inside the RunTask dialog component
const [startDate, setStartDate] = useState('');
const [endDate, setEndDate] = useState('');

const handleRun = async () => {
  const overrides: Record<string, string> = {};
  if (startDate) overrides.start_date = startDate;
  if (endDate) overrides.end_date = endDate;

  const configOverrides = Object.keys(overrides).length > 0
    ? JSON.stringify(overrides)
    : undefined;

  await ingestionClient.runTask({
    projectId: projectID,
    taskId: id,
    configOverrides,
  });
};
```

- [ ] **Step 13: Verify TypeScript compilation**

```bash
cd /tmp/opencode/aleph/frontend && npx tsc --noEmit
```
Expected: clean compile.

- [ ] **Step 14: Commit**

```bash
cd /tmp/opencode/aleph && git add frontend/src/
git commit -m "feat(frontend): add date range inputs to RunTask with config_overrides"
```
