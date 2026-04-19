package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1/nlpconnect"
	"github.com/ff3300/aleph-v2/internal/dsl"
	"github.com/ff3300/aleph-v2/internal/storage"
	"github.com/ff3300/aleph-v2/internal/repository"
	"net/http"
)

type QueryHandler struct {
	db           *storage.DuckDB
	projectsRoot string
	nlpClient    nlpconnect.NLPServiceClient
	metaRepo     *repository.MetadataRepository
	ollamaClient *http.Client
	
	mu       sync.RWMutex
	programs map[string]*dsl.Program
}

func NewQueryHandler(db *storage.DuckDB, projectsRoot string, metaRepo *repository.MetadataRepository, nlpAddr string) *QueryHandler {
	if nlpAddr == "" { nlpAddr = "http://localhost:8001" }
	if !strings.HasPrefix(nlpAddr, "http") { nlpAddr = "http://" + nlpAddr }
	
	nlpClient := nlpconnect.NewNLPServiceClient(http.DefaultClient, nlpAddr)
	return &QueryHandler{
		db:           db, 
		projectsRoot: projectsRoot,
		programs:     make(map[string]*dsl.Program),
		nlpClient:    nlpClient,
		metaRepo:     metaRepo,
		ollamaClient: &http.Client{Timeout: 2 * time.Minute},
	}
}

func (h *QueryHandler) resolveProject(projectID string) (string, *dsl.Program, error) {
	if projectID == "" { projectID = "default" }
	h.mu.RLock()
	prog, ok := h.programs[projectID]
	h.mu.RUnlock()
	projectPath := filepath.Join(h.projectsRoot, projectID)
	if _, err := os.Stat(projectPath); os.IsNotExist(err) { return "", nil, fmt.Errorf("project %s not found", projectID) }
	if !ok {
		h.mu.Lock()
		defer h.mu.Unlock()
		if prog, ok = h.programs[projectID]; ok { return projectPath, prog, nil }
		ontPath := filepath.Join(projectPath, "ontologies", "core.aleph")
		content, err := os.ReadFile(ontPath)
		if err != nil { return "", nil, fmt.Errorf("failed to read ontology: %v", err) }
		prog, err = dsl.Parse(string(content))
		if err != nil { return "", nil, fmt.Errorf("failed to parse ontology: %v", err) }
		h.programs[projectID] = prog
	}
	return projectPath, prog, nil
}

func (h *QueryHandler) ConfirmAction(ctx context.Context, req *connect.Request[v1.ConfirmActionRequest]) (*connect.Response[v1.ConfirmActionResponse], error) {
    return connect.NewResponse(&v1.ConfirmActionResponse{Success: true}), nil
}

func (h *QueryHandler) GetChatHistory(
	ctx context.Context,
	req *connect.Request[v1.GetChatHistoryRequest],
) (*connect.Response[v1.GetChatHistoryResponse], error) {
	projectID := req.Msg.ProjectId
	agentID := req.Msg.AgentId
	rows, err := h.metaRepo.DB().Query("SELECT role, content, tool_call, created_at FROM system_chat_history WHERE project_id = ? AND agent_id = ? ORDER BY created_at ASC", projectID, agentID)
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
	defer rows.Close()
	var messages []*v1.ChatMessage
	for rows.Next() {
		var m v1.ChatMessage
		var createdAt time.Time
		if err := rows.Scan(&m.Role, &m.Content, &m.ToolCall, &createdAt); err != nil { continue }
		m.CreatedAt = createdAt.Unix()
		messages = append(messages, &m)
	}
	return connect.NewResponse(&v1.GetChatHistoryResponse{Messages: messages}), nil
}

func (h *QueryHandler) ExecuteQuery(
	ctx context.Context,
	req *connect.Request[v1.ExecuteQueryRequest],
) (*connect.Response[v1.ExecuteQueryResponse], error) {
	objName := req.Msg.ObjectType
	projectID := req.Msg.ProjectId
	projectPath, prog, err := h.resolveProject(projectID)
	if err != nil { return nil, connect.NewError(connect.CodeNotFound, err) }
	dataRoot := filepath.Join(projectPath, "raw")
	compiler := dsl.NewCompiler(prog, dataRoot)
	sql, err := compiler.CompileObject(objName)
	if err != nil { return nil, connect.NewError(connect.CodeInvalidArgument, err) }
	
	if req.Msg.Limit > 0 {
		sql = fmt.Sprintf("SELECT * FROM (%s) LIMIT %d", sql, req.Msg.Limit)
	}

	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rows, err := h.db.QueryContext(queryCtx, sql)
	if err != nil {
		if strings.Contains(err.Error(), "resource exhausted") {
			return nil, connect.NewError(connect.CodeResourceExhausted, err)
		}
		if queryCtx.Err() == context.DeadlineExceeded {
			return nil, connect.NewError(connect.CodeDeadlineExceeded, fmt.Errorf("query analysis timed out (limit: 30s)"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	var protoRows []*v1.Row
	for rows.Next() {
		row := make([]interface{}, len(cols)); rp := make([]interface{}, len(cols))
		for i := range row { rp[i] = &row[i] }
		if err := rows.Scan(rp...); err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
		values := make(map[string]string)
		for i, colName := range cols {
			if row[i] != nil { values[colName] = fmt.Sprintf("%v", row[i]) } else { values[colName] = "" }
		}
		protoRows = append(protoRows, &v1.Row{Values: values})
	}
	res := connect.NewResponse(&v1.ExecuteQueryResponse{ Sql: sql, Columns: cols, Rows: protoRows })
	return res, nil
}

func (h *QueryHandler) suggestView(cols []string) string {
	hasLat, hasLon, hasTime := false, false, false
	for _, c := range cols {
		cl := strings.ToLower(c)
		if strings.Contains(cl, "lat") { hasLat = true }
		if strings.Contains(cl, "lon") || strings.Contains(cl, "lng") { hasLon = true }
		if strings.Contains(cl, "date") || strings.Contains(cl, "time") { hasTime = true }
	}
	if hasLat && hasLon { return "map" }
	if hasTime { return "timeline" }
	return "table"
}

func (h *QueryHandler) GetDataStats(ctx context.Context, req *connect.Request[v1.GetDataStatsRequest]) (*connect.Response[v1.GetDataStatsResponse], error) {
	projectID := req.Msg.ProjectId; objName := req.Msg.ObjectType
	projectPath, prog, err := h.resolveProject(projectID)
	if err != nil { return nil, connect.NewError(connect.CodeNotFound, err) }
	dataRoot := filepath.Join(projectPath, "raw"); compiler := dsl.NewCompiler(prog, dataRoot)
	baseSql, err := compiler.CompileObject(objName)
	if err != nil { return nil, connect.NewError(connect.CodeInvalidArgument, err) }
	
	rows, err := h.db.Query(fmt.Sprintf("SELECT * FROM (%s) LIMIT 0", baseSql))
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
	cols, _ := rows.Columns()
	colTypes, _ := rows.ColumnTypes()
	rows.Close()

	var stats []*v1.ColumnStats
	for i, col := range cols {
		s := &v1.ColumnStats{ColumnName: col, TopValues: make(map[string]int64)}
		
		dbType := strings.ToUpper(colTypes[i].DatabaseTypeName())
		isAggregatable := !strings.Contains(dbType, "STRUCT") && !strings.Contains(dbType, "LIST") && !strings.Contains(dbType, "MAP") && dbType != "BOOLEAN"

		var q string
		if isAggregatable {
			q = fmt.Sprintf(`SELECT MIN("%s"), MAX("%s"), COUNT("%s"), COUNT(DISTINCT "%s") FROM (%s)`, col, col, col, col, baseSql)
		} else {
			q = fmt.Sprintf(`SELECT NULL, NULL, COUNT("%s"), COUNT(DISTINCT "%s") FROM (%s)`, col, col, baseSql)
		}
		
		var min, max interface{}
		h.db.DB().QueryRow(q).Scan(&min, &max, &s.Count, &s.UniqueCount)
		if min != nil { s.Min = fmt.Sprintf("%v", min) }
		if max != nil { s.Max = fmt.Sprintf("%v", max) }
		stats = append(stats, s)
	}
	return connect.NewResponse(&v1.GetDataStatsResponse{Stats: stats}), nil
}

func (h *QueryHandler) GlobalQuery(ctx context.Context, req *connect.Request[v1.GlobalQueryRequest]) (*connect.Response[v1.GlobalQueryResponse], error) {
	// Re-map the request and call ExecuteQuery. Since signatures now match conceptually but not type-wise, 
	// we perform a light manual mapping or cast. Given the structures are identical, we can safely adapt.
	execReq := &connect.Request[v1.ExecuteQueryRequest]{
		Msg: &v1.ExecuteQueryRequest{
			ObjectType: req.Msg.ObjectType,
			ProjectId:  req.Msg.ProjectId,
			Limit:      req.Msg.Limit,
		},
	}
	resp, err := h.ExecuteQuery(ctx, execReq)
	if err != nil { return nil, err }
	
	return connect.NewResponse(&v1.GlobalQueryResponse{
		Sql:      resp.Msg.Sql,
		Columns:  resp.Msg.Columns,
		Rows:     resp.Msg.Rows,
	}), nil
}

func (h *QueryHandler) Chat(
	ctx context.Context,
	req *connect.Request[v1.ChatRequest],
	stream *connect.ServerStream[v1.ChatResponse],
) error {
	msg := req.Msg.Message
	projectID := req.Msg.ProjectId
	agentID := req.Msg.AgentId
	projectPath, _, err := h.resolveProject(projectID)
	if err != nil { return connect.NewError(connect.CodeNotFound, err) }

	ontPath := filepath.Join(projectPath, "ontologies", "core.aleph")
	ontContent, _ := os.ReadFile(ontPath)

	h.metaRepo.SaveChatMessage(projectID, agentID, "user", msg, "")

	var agent v1.Agent
	if agentID != "" {
		data, _ := h.metaRepo.DB().Query("SELECT model, system_prompt FROM system_agents WHERE id = ?", agentID)
		if data.Next() { data.Scan(&agent.Model, &agent.SystemPrompt) }
		data.Close()
	}
	if agent.Model == "" { agent.Model = "llama3" }

	fullSystemPrompt := agent.SystemPrompt + "\n\nCONTEXTUAL DATA ONTOLOGY (Aleph Format):\n" + string(ontContent) + "\n\nUse the 'search_data' tool to query the objects defined above. Always refer to columns exactly as named in the ontology."

	tools := []map[string]interface{}{
		{
			"type": "function",
			"function": map[string]interface{}{
				"name": "search_data",
				"description": "Search records from a specific business object defined in the ontology.",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"object_name": map[string]interface{}{"type": "string"},
						"limit":       map[string]interface{}{"type": "integer", "default": 10},
					},
					"required": []string{"object_name"},
				},
			},
		},
	}

	ollamaReq := map[string]interface{}{
		"model": agent.Model,
		"messages": []map[string]interface{}{
			{"role": "system", "content": fullSystemPrompt},
			{"role": "user", "content": msg},
		},
		"stream": false,
		"tools": tools,
	}

	for i := 0; i < 5; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		body, _ := json.Marshal(ollamaReq)
		reqBody := strings.NewReader(string(body))
		httpReq, _ := http.NewRequestWithContext(ctx, "POST", "http://localhost:11434/api/chat", reqBody)
		httpReq.Header.Set("Content-Type", "application/json")
		
		resp, err := h.ollamaClient.Do(httpReq)
		if err != nil { return connect.NewError(connect.CodeUnavailable, err) }
		defer resp.Body.Close()

		var ollamaResp struct {
			Message struct {
				Role      string `json:"role"`
				Content   string `json:"content"`
				ToolCalls []struct {
					Function struct { Name string `json:"name"`; Arguments map[string]interface{} `json:"arguments"` } `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		}
		json.NewDecoder(resp.Body).Decode(&ollamaResp)

		if ollamaResp.Message.Content != "" {
			stream.Send(&v1.ChatResponse{Token: ollamaResp.Message.Content})
			h.metaRepo.SaveChatMessage(projectID, agentID, "assistant", ollamaResp.Message.Content, "")
		}

		if len(ollamaResp.Message.ToolCalls) == 0 { break }

		messages := ollamaReq["messages"].([]map[string]interface{})
		messages = append(messages, map[string]interface{}{ "role": "assistant", "content": ollamaResp.Message.Content, "tool_calls": ollamaResp.Message.ToolCalls })

		for _, tc := range ollamaResp.Message.ToolCalls {
			reasoning := fmt.Sprintf("Ragionamento: Accesso sicuro ai dati per '%s'.", tc.Function.Name)
			stream.Send(&v1.ChatResponse{ToolCall: reasoning})
			h.metaRepo.SaveChatMessage(projectID, agentID, "assistant", "", reasoning)
			
			var resultStr string
			if tc.Function.Name == "search_data" {
				objName := tc.Function.Arguments["object_name"].(string)
				limit := 10
				if l, ok := tc.Function.Arguments["limit"].(float64); ok { limit = int(l) }
				
				res, err := h.ExecuteQuery(ctx, connect.NewRequest(&v1.ExecuteQueryRequest{ObjectType: objName, ProjectId: projectID, Limit: int32(limit)}))
				if err != nil { resultStr = "Errore: " + err.Error() } else {
					jb, _ := json.Marshal(res.Msg.Rows)
					resultStr = string(jb)
					if len(resultStr) > 2000 {
						resultStr = resultStr[:2000] + "\n... [Risultati troncati per limiti di contesto. Usa filtri più specifici se necessario.]"
					}
				}
				h.db.Cleanup()
			}
			messages = append(messages, map[string]interface{}{ "role": "tool", "content": resultStr })
		}
		ollamaReq["messages"] = messages
	}
	return nil
}
