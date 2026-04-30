package handler

import (
	"context"
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
	"github.com/ff3300/aleph-v2/internal/decision"
	"github.com/ff3300/aleph-v2/internal/dsl"
	"github.com/ff3300/aleph-v2/internal/middleware"
	"github.com/ff3300/aleph-v2/internal/registry"
	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/safeident"
	"github.com/ff3300/aleph-v2/internal/ssrf"
	"github.com/ff3300/aleph-v2/internal/storage"
	"github.com/ff3300/aleph-v2/internal/telemetry"
)

type QueryHandler struct {
	db           *storage.DuckDB
	projectsRoot string
	metaRepo     *repository.MetadataRepository
	httpClient   *http.Client
	nlpHandler   *NLPHandler
	registry     *registry.DuckDBRegistry
	programs     *programCache
	executor     decision.ToolExecutor   // set after construction — bridges engine to handler dispatch
	engine       decision.DecisionEngine // optional, nil = degraded mode (uses hardcoded if-else fallback)
}

// SetDecisionEngine attaches a decision engine and tool executor to the handler.
// These are set after construction to avoid changing the constructor signature.
// engine may be nil for degraded mode (uses hardcoded if-else fallback).
func (h *QueryHandler) SetDecisionEngine(eng decision.DecisionEngine, exec decision.ToolExecutor) {
	h.engine = eng
	h.executor = exec
}

var validProjectID = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*$`)

func NewQueryHandler(db *storage.DuckDB, projectsRoot string, metaRepo *repository.MetadataRepository, nlpHandler *NLPHandler, reg *registry.DuckDBRegistry) *QueryHandler {
	return &QueryHandler{
		db:           db, 
		projectsRoot: projectsRoot,
		programs:     newProgramCache(),
		metaRepo:     metaRepo,
		httpClient:   ssrf.NewClient(),
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
	msgs, err := h.metaRepo.GetChatHistory(ctx, projectID, agentID)
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
	start := time.Now()
	defer func() {
		telemetry.RecordDBQuery("execute_query", time.Since(start).Seconds())
	}()
	objName := req.Msg.ObjectType
	projectID := middleware.ProjectIDFromContext(ctx)
	if projectID == "" {
		projectID = req.Msg.ProjectId
	}

	if err := safeident.ValidateStrictIdentifier(objName); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("nome oggetto non valido: %w", err))
	}

	lowerObjName := strings.ToLower(objName)

	// Defense-in-depth: validate lowerObjName (derived from objName which passed ValidateStrictIdentifier check)
	if err := safeident.ValidateStrictIdentifier(lowerObjName); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("nome oggetto non valido: %w", err))
	}
	if projectID != "" && !validProjectID.MatchString(projectID) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid project_id"))
	}

	limit := req.Msg.Limit
	if limit <= 0 {
		limit = 1000
	}

	sql := ""
	checkRows, err2 := h.db.QueryContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = $1 AND table_name = $2", "main", lowerObjName)
	if err2 == nil {
		var count int
		if checkRows.Next() { checkRows.Scan(&count) }
		checkRows.Close()
		if count > 0 {
			sql = "SELECT * FROM " + safeident.QuoteIdentifier(lowerObjName) + fmt.Sprintf(" LIMIT %d", limit) // safe: lowerObjName validated via safeident.ValidateStrictIdentifier
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
			checkRows, err2 := h.db.QueryContext(queryCtx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = $1 AND table_name = $2", "main", lowerObjName)
			if err2 == nil {
				var count int
				if checkRows.Next() { checkRows.Scan(&count) }
				checkRows.Close()
				if count > 0 {
					sql = "SELECT * FROM " + safeident.QuoteIdentifier(lowerObjName) + fmt.Sprintf(" LIMIT %d", limit) // safe: lowerObjName validated via safeident.ValidateStrictIdentifier
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
	if err := safeident.ValidateStrictIdentifier(objName); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("nome oggetto non valido: %w", err))
	}
	projectPath, prog, err := h.resolveProject(projectID)
	if err != nil { return nil, connect.NewError(connect.CodeNotFound, err) }
	dataRoot := filepath.Join(projectPath, "raw"); compiler := dsl.NewCompiler(prog, dataRoot)
	baseSql, err := compiler.CompileObject(objName)
	if err != nil { return nil, connect.NewError(connect.CodeInvalidArgument, err) }
	
	// Query 1: Discover columns and types
	rows, err := h.db.Query(fmt.Sprintf("SELECT * FROM (%s) LIMIT 0", baseSql))
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
	allCols, _ := rows.Columns()
	colTypes, _ := rows.ColumnTypes()
	rows.Close()

	// Filter to valid column names, determine aggregatability
	var cols []string
	var aggFlags []bool
	for i, c := range allCols {
		if err := safeident.ValidateColumnName(c); err != nil {
			continue
		}
		cols = append(cols, c)
		dbType := strings.ToUpper(colTypes[i].DatabaseTypeName())
		isAgg := !strings.Contains(dbType, "STRUCT") && !strings.Contains(dbType, "LIST") && !strings.Contains(dbType, "MAP") && dbType != "BOOLEAN"
		aggFlags = append(aggFlags, isAgg)
	}

	stats := make([]*v1.ColumnStats, len(cols))
	for i, c := range cols {
		stats[i] = &v1.ColumnStats{ColumnName: c, TopValues: make(map[string]int64)}
	}

	if len(cols) == 0 {
		return connect.NewResponse(&v1.GetDataStatsResponse{Stats: stats}), nil
	}

	// Query 2: Single batch — MIN/MAX/COUNT/COUNT(DISTINCT) for ALL columns (1 query total)
	// Build aggregate SELECT parts — column names validated by safeident.ValidateColumnName; SQL identifiers cannot be parameterized
	var selectParts []string
	var scanTargets []interface{}
	for i, c := range cols {
		quoted := safeident.QuoteIdentifier(c)
		if aggFlags[i] {
			selectParts = append(selectParts,
				"MIN("+quoted+"), MAX("+quoted+"), COUNT("+quoted+"), COUNT(DISTINCT "+quoted+")")
		} else {
			selectParts = append(selectParts,
				"NULL, NULL, COUNT("+quoted+"), COUNT(DISTINCT "+quoted+")")
		}
		scanTargets = append(scanTargets, new(interface{}), new(interface{}), new(int64), new(int64))
	}

	aggSQL := fmt.Sprintf("SELECT %s FROM (%s)", strings.Join(selectParts, ", "), baseSql)
	if err := h.db.QueryRow(aggSQL).Scan(scanTargets...); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	for i := range cols {
		minPtr := scanTargets[i*4+0].(*interface{})
		maxPtr := scanTargets[i*4+1].(*interface{})
		countPtr := scanTargets[i*4+2].(*int64)
		distinctPtr := scanTargets[i*4+3].(*int64)
		if *minPtr != nil {
			stats[i].Min = fmt.Sprintf("%v", *minPtr)
		}
		if *maxPtr != nil {
			stats[i].Max = fmt.Sprintf("%v", *maxPtr)
		}
		stats[i].Count = *countPtr
		stats[i].UniqueCount = *distinctPtr
	}

	// Query 3: UNION ALL — top 10 values for aggregatable columns only (1 query total)
	var aggIndices []int
	for i, agg := range aggFlags {
		if agg {
			aggIndices = append(aggIndices, i)
		}
	}

	if len(aggIndices) > 0 {
		var unionParts []string
		for _, si := range aggIndices {
			c := cols[si]
			quoted := safeident.QuoteIdentifier(c)
			unionParts = append(unionParts,
				"SELECT * FROM (SELECT "+safeident.QuoteStringLiteral(c)+" AS cn, CAST("+quoted+" AS VARCHAR) AS val, COUNT(*) AS cnt FROM ("+baseSql+") GROUP BY "+quoted+" ORDER BY cnt DESC LIMIT 10)")
		}
		topSQL := strings.Join(unionParts, " UNION ALL ")
		topRows, topErr := h.db.Query(topSQL)
		if topErr == nil {
			for topRows.Next() {
				var cn, val string
				var cnt int64
				if err := topRows.Scan(&cn, &val, &cnt); err == nil {
					for _, si := range aggIndices {
						if cols[si] == cn {
							stats[si].TopValues[val] = cnt
							break
						}
					}
				}
			}
			topRows.Close()
		}
	}
	return connect.NewResponse(&v1.GetDataStatsResponse{Stats: stats}), nil
}

func (h *QueryHandler) GetDataLineage(ctx context.Context, req *connect.Request[v1.GetDataLineageRequest]) (*connect.Response[v1.GetDataLineageResponse], error) {
	projectID := req.Msg.ProjectId
	tableName := req.Msg.TableName
	if projectID == "" || tableName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("project_id and table_name are required"))
	}
	if !validProjectID.MatchString(projectID) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid project_id"))
	}
	if err := safeident.ValidateStrictIdentifier(tableName); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid table_name: %w", err))
	}
	var columnsCount int64
	var rowsCount int64
	var jsonCols string
	row := h.db.QueryRowContext(ctx,
		"SELECT (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = $1 AND table_name = $2), (SELECT COUNT(*) FROM "+safeident.QuoteIdentifier(projectID)+"."+safeident.QuoteIdentifier(tableName)+"), json_group_array(column_name || ':' || data_type) FROM information_schema.columns WHERE table_schema = $1 AND table_name = $2", // safe: projectID validated via validProjectID regex; tableName validated via safeident.ValidateStrictIdentifier
		projectID, tableName,
	)
	if row == nil {
		return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("duckdb resource exhausted"))
	}
	err := row.Scan(&columnsCount, &rowsCount, &jsonCols)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("table not found: %w", err))
	}
	return connect.NewResponse(&v1.GetDataLineageResponse{
		Provenance: &v1.DataProvenance{
			TableName:    tableName,
			Source:       "duckdb:" + projectID,
			ColumnsCount: columnsCount,
			RowsCount:    rowsCount,
		},
	}), nil
}

func (h *QueryHandler) GetChecksum(ctx context.Context, req *connect.Request[v1.GetChecksumRequest]) (*connect.Response[v1.GetChecksumResponse], error) {
	projectID := req.Msg.ProjectId
	tableName := req.Msg.TableName

	if projectID == "" || tableName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("project_id and table_name are required"))
	}

	if !validProjectID.MatchString(projectID) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid project_id"))
	}
	if err := safeident.ValidateStrictIdentifier(tableName); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid table_name: %w", err))
	}

	// safe: identifiers validated (validProjectID regex + safeident.ValidateStrictIdentifier) and quoted via safeident.QuoteIdentifier
	var checksum string
	row1 := h.db.QueryRowContext(ctx,
		"SELECT md5(CAST(SUM(hash(TO_JSON(*))) AS VARCHAR)) FROM "+safeident.QuoteIdentifier(projectID)+"."+safeident.QuoteIdentifier(tableName),
	)
	if row1 == nil {
		return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("duckdb resource exhausted"))
	}
	err := row1.Scan(&checksum)
	if err != nil {
		// Table may be empty — use structure-based checksum
		row2 := h.db.QueryRowContext(ctx,
			"SELECT md5(column_list) FROM (SELECT string_agg(column_name || ':' || data_type, ',' ORDER BY column_name) AS column_list FROM information_schema.columns WHERE table_schema = $1 AND table_name = $2)",
			projectID, tableName,
		)
		if row2 == nil {
			return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("duckdb resource exhausted"))
		}
		err = row2.Scan(&checksum)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("table not found: %w", err))
		}
	}

	return connect.NewResponse(&v1.GetChecksumResponse{
		Checksum:  checksum,
		TableName: tableName,
	}), nil
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
	if err != nil {
		return connect.NewError(connect.CodeNotFound, err)
	}

	ontPath := filepath.Join(projectPath, "ontologies", "core.aleph")
	ontContent, ontErr := os.ReadFile(ontPath)
	if ontErr != nil {
		slog.Warn("ontology file not found", "path", ontPath, "error", ontErr)
	}

	h.metaRepo.SaveChatMessage(ctx, projectID, agentID, "user", msg, "")

	agent, err := h.resolveAgent(ctx, agentID)
	if err != nil {
		return err
	}

	fullSystemPrompt := agent.SystemPrompt
	if len(ontContent) > 0 {
		fullSystemPrompt += "\n\nCONTEXTUAL DATA ONTOLOGY (Aleph Format):\n" + string(ontContent) +
			"\n\nUse the 'search_data' tool to query the objects defined above. Always refer to columns exactly as named in the ontology."
	}

	session := NewChatSession(ctx, stream, h, projectID, agentID, msg, agent, ontContent, fullSystemPrompt)
	return session.Run()
}

func (h *QueryHandler) resolveAgent(ctx context.Context, agentID string) (AgentInfo, error) {
	var agent AgentInfo
	if agentID == "" {
		return agent, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("agent ID is required"))
	}

	agentRec, err := h.metaRepo.GetAgentForChat(agentID)
	if err != nil || agentRec == nil {
		return agent, connect.NewError(connect.CodeNotFound, fmt.Errorf("agent %s not found", agentID))
	}

	agent = AgentInfo{
		Provider:     agentRec.Provider,
		Model:        agentRec.Model,
		ApiKey:       agentRec.ApiKey,
		SystemPrompt: agentRec.SystemPrompt,
	}

	if agentRec.BaseURL != "" {
		agent.BaseURL = agentRec.BaseURL
	} else {
		switch agent.Provider {
		case "openai":
			agent.BaseURL = "https://api.openai.com"
		case "anthropic":
			agent.BaseURL = "https://api.anthropic.com"
		default:
			agent.BaseURL = "http://localhost:11434"
		}
	}

	if agent.Model == "" {
		return agent, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("agent %s has no model configured", agentID))
	}
	if agent.Provider == "" {
		return agent, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("agent %s has no provider configured", agentID))
	}

	return agent, nil
}

func truncateJSON(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	var depth int
	var maxDepth int
	var inString bool
	var escaped bool
	truncateAt := -1
	for i := 0; i < len(s) && i < limit; i++ {
		ch := s[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch ch {
		case '{', '[':
			depth++
		case '}', ']':
			depth--
			if depth < 0 {
				depth = 0
			}
		}
		if depth > maxDepth {
			maxDepth = depth
		}
		if i >= limit-4 && depth <= 1 {
			truncateAt = i
			break
		}
	}
	// If no JSON structure was entered, just truncate flat content at limit.
	if maxDepth == 0 {
		return s[:limit]
	}
	if truncateAt < 0 {
		truncateAt = limit
	}
	result := s[:truncateAt]
	if truncateAt < len(s) {
		result += "..."
	}
	return result
}

