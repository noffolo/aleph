package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"
	alephv1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	nlpv1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1"
	"github.com/ff3300/aleph-v2/internal/decision"
	"github.com/ff3300/aleph-v2/internal/registry"
)

// toolExecutor implements decision.ToolExecutor by wrapping the handler's
// existing tool dispatch logic exactly as it was in Chat().
type toolExecutor struct {
	executeQuery func(ctx context.Context, req *connect.Request[alephv1.ExecuteQueryRequest]) (*connect.Response[alephv1.ExecuteQueryResponse], error)
	nlpHandler   *NLPHandler
	reg          *registry.DuckDBRegistry
}

// Compile-time interface check
var _ decision.ToolExecutor = (*toolExecutor)(nil)

// ExecuteTool dispatches a tool call by name, mirroring the original Chat() switch.
// The bool return indicates whether the tool requires user confirmation.
func (e *toolExecutor) ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}, projectID string, agentID string) (string, bool, error) {
	switch toolName {
	case "search_data":
		return e.executeSearchData(ctx, args, projectID)
	case "analyze_sentiment":
		return e.executeAnalyzeSentiment(ctx, args)
	case "get_trust_score":
		return e.executeGetTrustScore(ctx, args)
	default:
		// Unknown tool — requires confirmation
		return fmt.Sprintf("Proposta azione '%s' in attesa di conferma.", toolName), true, nil
	}
}

func (e *toolExecutor) executeSearchData(ctx context.Context, args map[string]interface{}, projectID string) (string, bool, error) {
	objName, _ := args["object_name"].(string)
	if objName == "" {
		return "", false, fmt.Errorf("Errore: parametro object_name mancante")
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	res, err := e.executeQuery(ctx, connect.NewRequest(&alephv1.ExecuteQueryRequest{
		ObjectType: objName,
		ProjectId:  projectID,
		Limit:      int32(limit),
	}))
	if err != nil {
		return "", false, fmt.Errorf("Errore: %v", err)
	}

	jb, _ := json.Marshal(res.Msg.Rows)
	resultStr := string(jb)
	if len(resultStr) > 2000 {
		resultStr = resultStr[:2000] + "\n... [Risultati troncati per limiti di contesto.]"
	}
	return resultStr, false, nil
}

func (e *toolExecutor) executeAnalyzeSentiment(ctx context.Context, args map[string]interface{}) (string, bool, error) {
	text, _ := args["text"].(string)
	if text == "" {
		return "", false, fmt.Errorf("Errore: parametro text mancante per analyze_sentiment")
	}

	if e.nlpHandler != nil {
		resp, err := e.nlpHandler.AnalyzeSentiment(ctx, connect.NewRequest(&nlpv1.AnalyzeSentimentRequest{Text: text}))
		if err != nil {
			return "", false, fmt.Errorf("Errore analisi sentiment: %v", err)
		}
		result := map[string]interface{}{
			"score": resp.Msg.Score,
			"label": resp.Msg.Label,
		}
		jb, _ := json.Marshal(result)
		return string(jb), false, nil
	}
	return `{"error": "servizio sentiment non disponibile"}`, false, nil
}

func (e *toolExecutor) executeGetTrustScore(ctx context.Context, args map[string]interface{}) (string, bool, error) {
	entityID, _ := args["entity_id"].(string)
	if entityID == "" {
		return "", false, fmt.Errorf("Errore: parametro entity_id mancante per get_trust_score")
	}

	if e.reg != nil {
		comp, err := e.reg.GetComponentByID(ctx, entityID)
		if err != nil || comp == nil {
			return "", false, fmt.Errorf(`{"error": "entità %s non trovata"}`, entityID)
		}
		result := map[string]interface{}{
			"entity_id":       entityID,
			"avg_brier_score": comp.AvgBrierScore,
			"trust_score":     comp.TrustScore,
		}
		jb, _ := json.Marshal(result)
		return string(jb), false, nil
	}
	return `{"error": "registry non disponibile"}`, false, nil
}

// NewHandlerToolExecutor creates a new toolExecutor that wraps the handler's dispatch logic.
func NewHandlerToolExecutor(
	executeQuery func(ctx context.Context, req *connect.Request[alephv1.ExecuteQueryRequest]) (*connect.Response[alephv1.ExecuteQueryResponse], error),
	nlpHandler *NLPHandler,
	reg *registry.DuckDBRegistry,
) decision.ToolExecutor {
	return &toolExecutor{
		executeQuery: executeQuery,
		nlpHandler:   nlpHandler,
		reg:          reg,
	}
}


