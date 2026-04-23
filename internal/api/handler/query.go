package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	nlpv1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1"
	"github.com/ff3300/aleph-v2/internal/dsl"
	"github.com/ff3300/aleph-v2/internal/llm"
	"github.com/ff3300/aleph-v2/internal/middleware"
	"github.com/ff3300/aleph-v2/internal/registry"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/storage"
)

type QueryHandler struct {
	db           *storage.DuckDB
	projectsRoot string
	metaRepo     *repository.MetadataRepository
	httpClient   *http.Client
	nlpHandler   *NLPHandler
	registry     *registry.DuckDBRegistry
	programs     *programCache
}

var validName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func NewQueryHandler(db *storage.DuckDB, projectsRoot string, metaRepo *repository.MetadataRepository, nlpHandler *NLPHandler, reg *registry.DuckDBRegistry) *QueryHandler {
	return &QueryHandler{
		db:           db, 
		projectsRoot: projectsRoot,
		programs:     newProgramCache(),
		metaRepo:     metaRepo,
		httpClient:   &http.Client{Timeout: 2 * time.Minute},
		nlpHandler:   nlpHandler,
		registry:     reg,
	}
}

func (h *QueryHandler) resolveProject(projectID string) (string, *dsl.Program, error) {
	if projectID == "" { projectID = "default" }
	prog := h.programs.Get(projectID)
	projectPath, err := sanitizePath(h.projectsRoot, projectID)
	if err != nil { return "", nil, connect.NewError(connect.CodeInvalidArgument, err) }
	if _, serr := os.Stat(projectPath); os.IsNotExist(serr) { return "", nil, fmt.Errorf("project %s not found", projectID) }
	if prog == nil {
		ontPath := filepath.Join(projectPath, "ontologies", "core.aleph")
		content, err := os.ReadFile(ontPath)
		if err != nil { return "", nil, fmt.Errorf("failed to read ontology: %v", err) }
		prog, err = dsl.Parse(string(content))
		if err != nil { return "", nil, fmt.Errorf("failed to parse ontology: %v", err) }
		h.programs.Set(projectID, prog)
	}
	return projectPath, prog, nil
}

func (h *QueryHandler) ConfirmAction(ctx context.Context, req *connect.Request[v1.ConfirmActionRequest]) (*connect.Response[v1.ConfirmActionResponse], error) {
	projectID := middleware.ProjectIDFromContext(ctx)
	if projectID == "" {
		projectID = req.Msg.ProjectId
	}
	agentID := req.Msg.AgentId

	if projectID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("project_id is required"))
	}

	_, _, err := h.resolveProject(projectID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	if agentID != "" {
		exists, err := h.metaRepo.ConfirmAgentInProject(agentID, projectID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if !exists {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("agent %s not found in project %s", agentID, projectID))
		}
	}

	return connect.NewResponse(&v1.ConfirmActionResponse{Success: req.Msg.Approved}), nil
}

func (h *QueryHandler) GetChatHistory(ctx context.Context, req *connect.Request[v1.GetChatHistoryRequest]) (*connect.Response[v1.GetChatHistoryResponse], error) {
	projectID := middleware.ProjectIDFromContext(ctx)
	if projectID == "" {
		projectID = req.Msg.ProjectId
	}
	agentID := req.Msg.AgentId
	msgs, err := h.metaRepo.GetChatHistory(projectID, agentID)
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
	var messages []*v1.ChatMessage
	for _, m := range msgs {
		messages = append(messages, &v1.ChatMessage{Role: m.Role, Content: m.Content, ToolCall: m.ToolCall, CreatedAt: m.CreatedAt.Unix()})
	}
	return connect.NewResponse(&v1.GetChatHistoryResponse{Messages: messages}), nil
}

func (h *QueryHandler) ExecuteQuery(
	ctx context.Context,
	req *connect.Request[v1.ExecuteQueryRequest],
) (*connect.Response[v1.ExecuteQueryResponse], error) {
	objName := req.Msg.ObjectType
	projectID := middleware.ProjectIDFromContext(ctx)
	if projectID == "" {
		projectID = req.Msg.ProjectId
	}

	if !validName.MatchString(objName) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("nome oggetto non valido: %s", objName))
	}

	lowerObjName := strings.ToLower(objName)

	// Defense-in-depth: validate lowerObjName (derived from objName which passed validName check)
	// to ensure no SQL injection in Sprintf-constructed queries.
	if !validName.MatchString(lowerObjName) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("nome oggetto non valido: %s", objName))
	}

	limit := req.Msg.Limit
	if limit <= 0 {
		limit = 1000
	}

	sql := ""
	checkRows, err2 := h.db.Query(fmt.Sprintf("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'main' AND table_name = '%s'", lowerObjName))
	if err2 == nil {
		var count int
		if checkRows.Next() { checkRows.Scan(&count) }
		checkRows.Close()
		if count > 0 {
			sql = fmt.Sprintf("SELECT * FROM \"%s\" LIMIT %d", lowerObjName, limit)
		}
	}

	if sql == "" {
		projectPath, prog, err := h.resolveProject(projectID)
		if err != nil { return nil, connect.NewError(connect.CodeNotFound, err) }
		dataRoot := filepath.Join(projectPath, "raw")
		compiler := dsl.NewCompiler(prog, dataRoot)
		compiledSQL, err := compiler.CompileObject(objName)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("oggetto '%s' non trovato nell'ontologia né tra le tabelle disponibili", objName))
		}
		sql = compiledSQL
	}
	
	sql = fmt.Sprintf("SELECT * FROM (%s) LIMIT %d", sql, limit)

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
		if strings.Contains(err.Error(), "No files found") || strings.Contains(err.Error(), "IO Error") {
			checkRows, err2 := h.db.Query(fmt.Sprintf("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'main' AND table_name = '%s'", strings.ToLower(objName)))
			if err2 == nil {
				var count int
				if checkRows.Next() { checkRows.Scan(&count) }
				checkRows.Close()
				if count > 0 {
					sql = fmt.Sprintf("SELECT * FROM \"%s\" LIMIT %d", strings.ToLower(objName), limit)
					rows, err = h.db.QueryContext(queryCtx, sql)
					if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
				} else {
					return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("oggetto '%s' non trovato", objName))
				}
			} else {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
		} else {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
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
	projectID := middleware.ProjectIDFromContext(ctx)
	if projectID == "" {
		projectID = req.Msg.ProjectId
	}
	objName := req.Msg.ObjectType
	if !validName.MatchString(objName) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("nome oggetto non valido: %s", objName))
	}
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
		if !validName.MatchString(col) {
			continue
		}
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
		h.db.QueryRow(q).Scan(&min, &max, &s.Count, &s.UniqueCount)
		if min != nil { s.Min = fmt.Sprintf("%v", min) }
		if max != nil { s.Max = fmt.Sprintf("%v", max) }

		if isAggregatable {
			topQ := fmt.Sprintf(`SELECT "%s", COUNT(*) as cnt FROM (%s) GROUP BY "%s" ORDER BY cnt DESC LIMIT 10`, col, baseSql, col)
			topRows, topErr := h.db.Query(topQ)
			if topErr == nil {
				for topRows.Next() {
					var val interface{}
					var cnt int64
					if err := topRows.Scan(&val, &cnt); err == nil && val != nil {
						s.TopValues[fmt.Sprintf("%v", val)] = cnt
					}
				}
				topRows.Close()
			}
		}

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
	projectID := middleware.ProjectIDFromContext(ctx)
	if projectID == "" {
		projectID = req.Msg.ProjectId
	}
	agentID := req.Msg.AgentId
	projectPath, _, err := h.resolveProject(projectID)
	if err != nil { return connect.NewError(connect.CodeNotFound, err) }

	ontPath := filepath.Join(projectPath, "ontologies", "core.aleph")
	ontContent, ontErr := os.ReadFile(ontPath)
	if ontErr != nil {
		slog.Warn("ontology file not found, chat will proceed without ontology context", "path", ontPath, "error", ontErr)
	}

	h.metaRepo.SaveChatMessage(projectID, agentID, "user", msg, "")

	var agent v1.Agent
	if agentID != "" {
		agentRec, err := h.metaRepo.GetAgentForChat(agentID)
		if err == nil && agentRec != nil {
			agent.Provider = agentRec.Provider
			agent.Model = agentRec.Model
			agent.ApiKey = agentRec.ApiKey
			agent.SystemPrompt = agentRec.SystemPrompt
			agent.BaseUrl = agentRec.BaseURL
			if agentRec.SkillIDsJSON != "" {
				json.Unmarshal([]byte(agentRec.SkillIDsJSON), &agent.SkillIds)
			}
		}
	}
	if agent.Model == "" {
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("agent %s has no model configured; configure a model before chatting", agentID))
	}
	if agent.Provider == "" {
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("agent %s has no provider configured; configure a provider before chatting", agentID))
	}
	if agent.BaseUrl == "" {
		switch agent.Provider {
		case "openai":
			agent.BaseUrl = "https://api.openai.com"
		case "anthropic":
			agent.BaseUrl = "https://api.anthropic.com"
		default:
			agent.BaseUrl = "http://localhost:11434"
		}
	}

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
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "analyze_sentiment",
				"description": "Analyze the sentiment of text data. Returns a score from -1.0 (negative) to 1.0 (positive) and a label (positive/negative/neutral).",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"text": map[string]interface{}{"type": "string", "description": "The text to analyze"},
					},
					"required": []string{"text"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "get_trust_score",
				"description": "Get the trust score for a prediction entity. Returns the Brier score (0.0 = perfect, 1.0 = worst) and trust level.",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"entity_id": map[string]interface{}{"type": "string", "description": "The entity ID to check trust for"},
					},
					"required": []string{"entity_id"},
				},
			},
		},
	}

	if h.metaRepo != nil {
		registeredTools, err := h.metaRepo.ListTools()
		if err == nil {
			for _, t := range registeredTools {
				toolDef := map[string]interface{}{
					"type": "function",
					"function": map[string]interface{}{
						"name":        t.Name,
						"description": t.Description,
					},
				}
				if t.Code != "" {
					var params map[string]interface{}
					if json.Unmarshal([]byte(t.Code), &params) == nil {
						toolDef["function"].(map[string]interface{})["parameters"] = params
					}
				}
				tools = append(tools, toolDef)
			}
		}
	}

	chatMessages := []map[string]interface{}{
		{"role": "system", "content": fullSystemPrompt},
	}

	// Load chat history for this agent
	history, histErr := h.metaRepo.GetChatMessages(projectID, agentID)
	if histErr == nil {
		for _, m := range history {
			if m.Role == "user" {
				chatMessages = append(chatMessages, map[string]interface{}{"role": "user", "content": m.Content})
			} else if m.Role == "assistant" && m.ToolCall == "" {
				chatMessages = append(chatMessages, map[string]interface{}{"role": "assistant", "content": m.Content})
			}
			// Skip tool messages to keep context clean for the LLM
		}
	}
	chatMessages = append(chatMessages, map[string]interface{}{"role": "user", "content": msg})

	baseUrl := strings.TrimRight(agent.BaseUrl, "/")
	provider := llm.NewProvider(agent.Provider, baseUrl, h.httpClient)
	if provider == nil {
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("unsupported provider: %s", agent.Provider))
	}

	for i := 0; i < 5; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var systemPrompt string
		if agent.Provider == "anthropic" {
			systemPrompt = agent.SystemPrompt + "\n\nCONTEXTUAL DATA ONTOLOGY (Aleph Format):\n" + string(ontContent)
		}

		req := llm.CompletionRequest{
			Model:        agent.Model,
			Messages:     chatMessages,
			Tools:        tools,
			SystemPrompt: systemPrompt,
			ApiKey:       agent.ApiKey,
			BaseURL:      baseUrl,
		}

		completion, err := provider.Complete(ctx, req)
		if err != nil {
			return connect.NewError(connect.CodeUnavailable, err)
		}

		responseContent := completion.Content
		toolCalls := completion.ToolCalls

		type toolCall struct {
			Function struct {
				Name      string
				Arguments map[string]interface{}
			}
		}
		
		var localToolCalls []toolCall
		for _, tc := range toolCalls {
			localToolCalls = append(localToolCalls, toolCall{Function: struct {
				Name      string
				Arguments map[string]interface{}
			}{Name: tc.Name, Arguments: tc.Arguments}})
		}

		if responseContent != "" {
			stream.Send(&v1.ChatResponse{Token: responseContent})
			h.metaRepo.SaveChatMessage(projectID, agentID, "assistant", responseContent, "")
		}

		if len(localToolCalls) == 0 { break }

		assistantMsg := map[string]interface{}{"role": "assistant", "content": responseContent}
		if agent.Provider == "ollama" {
			assistantMsg["tool_calls"] = localToolCalls
		} else {
			tcList := make([]map[string]interface{}, len(localToolCalls))
			for j, tc := range localToolCalls {
				argsJSON, _ := json.Marshal(tc.Function.Arguments)
				tcList[j] = map[string]interface{}{
					"id":   fmt.Sprintf("call_%d_%d", i, j),
					"type": "function",
					"function": map[string]interface{}{
						"name":      tc.Function.Name,
						"arguments": string(argsJSON),
					},
				}
			}
			assistantMsg["tool_calls"] = tcList
		}
		chatMessages = append(chatMessages, assistantMsg)

		for _, tc := range toolCalls {
			reasoning := fmt.Sprintf("Executing tool: %s", tc.Name)
			stream.Send(&v1.ChatResponse{ToolCall: reasoning})
			h.metaRepo.SaveChatMessage(projectID, agentID, "assistant", "", reasoning)

			var resultStr string
			if tc.Name == "search_data" {
				objName, _ := tc.Arguments["object_name"].(string)
				if objName == "" {
					resultStr = "Errore: parametro object_name mancante"
				} else {
					limit := 10
					if l, ok := tc.Arguments["limit"].(float64); ok { limit = int(l) }

					res, err := h.ExecuteQuery(ctx, connect.NewRequest(&v1.ExecuteQueryRequest{ObjectType: objName, ProjectId: projectID, Limit: int32(limit)}))
					if err != nil {
						resultStr = "Errore: " + err.Error()
					} else {
						jb, _ := json.Marshal(res.Msg.Rows)
						resultStr = string(jb)
						if len(resultStr) > 2000 {
							resultStr = resultStr[:2000] + "\n... [Risultati troncati per limiti di contesto.]"
						}
					}
				}
			} else if tc.Name == "analyze_sentiment" {
				text, _ := tc.Arguments["text"].(string)
				if text == "" {
					resultStr = "Errore: parametro text mancante per analyze_sentiment"
				} else if h.nlpHandler != nil {
					resp, err := h.nlpHandler.AnalyzeSentiment(ctx, connect.NewRequest(&nlpv1.AnalyzeSentimentRequest{Text: text}))
					if err != nil {
						resultStr = fmt.Sprintf("Errore analisi sentiment: %v", err)
					} else {
						result := map[string]interface{}{
							"score": resp.Msg.Score,
							"label": resp.Msg.Label,
						}
						jb, _ := json.Marshal(result)
						resultStr = string(jb)
					}
				} else {
					resultStr = `{"error": "servizio sentiment non disponibile"}`
				}
			} else if tc.Name == "get_trust_score" {
				entityID, _ := tc.Arguments["entity_id"].(string)
				if entityID == "" {
					resultStr = "Errore: parametro entity_id mancante per get_trust_score"
				} else if h.registry != nil {
					comp, err := h.registry.GetComponentByID(entityID)
					if err != nil || comp == nil {
						resultStr = fmt.Sprintf(`{"error": "entità %s non trovata"}`, entityID)
					} else {
						result := map[string]interface{}{
							"entity_id":       entityID,
							"avg_brier_score": comp.AvgBrierScore,
							"trust_score":      comp.TrustScore,
						}
						jb, _ := json.Marshal(result)
						resultStr = string(jb)
					}
				} else {
					resultStr = `{"error": "registry non disponibile"}`
				}
			} else {
				stream.Send(&v1.ChatResponse{RequiresConfirmation: true})
				resultStr = fmt.Sprintf("Proposta azione '%s' in attesa di conferma.", tc.Name)
			}
			if agent.Provider == "ollama" {
				chatMessages = append(chatMessages, map[string]interface{}{"role": "tool", "content": resultStr})
			} else {
				chatMessages = append(chatMessages, map[string]interface{}{
					"role":          "tool",
					"content":       resultStr,
					"tool_call_id":  fmt.Sprintf("call_%d_tools_0", i),
				})
			}
		}
		h.db.Cleanup()
	}
	return nil
}
